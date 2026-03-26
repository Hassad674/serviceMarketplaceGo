import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../../core/network/api_client.dart';
import '../../../core/storage/secure_storage.dart';

/// Provides the singleton [MessagingWsService].
final messagingWsServiceProvider = Provider<MessagingWsService>((ref) {
  final storage = ref.watch(secureStorageProvider);
  return MessagingWsService(storage: storage);
});

/// WebSocket service for real-time messaging events.
///
/// Connects to `ws://{API_URL}/api/v1/ws?token={jwt}` and exposes a stream
/// of incoming JSON events (new_message, typing, status_update, etc.).
///
/// Handles heartbeat (every 30s) and automatic reconnection with
/// exponential backoff on disconnect.
///
/// On successful (re)connection, emits a synthetic `{"type":"reconnected"}`
/// event so that listeners can refresh stale state (e.g. presence).
class MessagingWsService {
  final SecureStorageService _storage;

  WebSocketChannel? _channel;
  Timer? _heartbeatTimer;
  Timer? _reconnectTimer;

  final _eventController =
      StreamController<Map<String, dynamic>>.broadcast();

  bool _isConnected = false;
  bool _isConnecting = false;
  bool _disposed = false;
  int _reconnectAttempts = 0;
  bool _hasConnectedBefore = false;
  static const int _maxReconnectDelay = 30; // seconds

  AppLifecycleListener? _lifecycleListener;

  MessagingWsService({required SecureStorageService storage})
      : _storage = storage {
    _lifecycleListener = AppLifecycleListener(
      onResume: _onAppResumed,
    );
  }

  /// Called when the app returns to the foreground. If the WS is
  /// disconnected (common after background), reconnect immediately
  /// instead of waiting for exponential backoff.
  void _onAppResumed() {
    if (_disposed) return;
    if (!_isConnected) {
      _reconnectTimer?.cancel();
      _reconnectAttempts = 0;
      connect();
    }
  }

  /// Stream of incoming WebSocket events (JSON maps).
  ///
  /// Each event has a `type` field:
  /// `new_message`, `typing`, `status_update`, `unread_count`,
  /// `message_edited`, `message_deleted`, `presence`, `reconnected`.
  ///
  /// The `reconnected` type is a synthetic client-side event emitted
  /// after a successful reconnection so consumers can refresh state.
  Stream<Map<String, dynamic>> get events => _eventController.stream;

  /// Whether the WebSocket is currently connected.
  bool get isConnected => _isConnected;

  /// Establishes the WebSocket connection.
  ///
  /// Retrieves the JWT from secure storage and connects with it
  /// as a query parameter. On successful reconnection, emits a
  /// synthetic `reconnected` event for stale-state refresh.
  Future<void> connect() async {
    if (_disposed || _isConnecting || _isConnected) return;
    _isConnecting = true;

    try {
      final token = await _storage.getAccessToken();
      if (token == null) {
        _isConnecting = false;
        return;
      }

      final wsUrl = _buildWsUrl(token);

      _channel = WebSocketChannel.connect(Uri.parse(wsUrl));
      await _channel!.ready;

      final isReconnect = _hasConnectedBefore;
      _isConnected = true;
      _isConnecting = false;
      _hasConnectedBefore = true;
      _reconnectAttempts = 0;

      _startHeartbeat();

      _channel!.stream.listen(
        _onMessage,
        onError: _onError,
        onDone: _onDone,
      );

      // Notify listeners so they can refresh stale presence/state
      if (isReconnect) {
        _eventController.add(const {'type': 'reconnected'});
      }
    } catch (_) {
      _isConnecting = false;
      _scheduleReconnect();
    }
  }

  /// Disconnects and cleans up resources.
  void disconnect() {
    _heartbeatTimer?.cancel();
    _reconnectTimer?.cancel();
    _channel?.sink.close();
    _channel = null;
    _isConnected = false;
    _isConnecting = false;
  }

  /// Permanently disposes the service. Cannot reconnect after this.
  void dispose() {
    _disposed = true;
    _lifecycleListener?.dispose();
    _lifecycleListener = null;
    disconnect();
    _eventController.close();
  }

  /// Sends a typing indicator for the given conversation.
  void sendTyping(String conversationId) {
    _send({'type': 'typing', 'conversation_id': conversationId});
  }

  /// Acknowledges receipt of a message.
  void sendAck(String messageId) {
    _send({'type': 'ack', 'message_id': messageId});
  }

  /// Sends a sync request with the last known sequence numbers
  /// per conversation.
  void sendSync(Map<String, int> lastSeqs) {
    _send({'type': 'sync', 'conversations': lastSeqs});
  }

  // ---------------------------------------------------------------------------
  // Internal
  // ---------------------------------------------------------------------------

  /// Builds the WebSocket URL with the JWT as a query parameter.
  ///
  /// **Security tradeoff**: the token appears in the URL because the
  /// WebSocket API (RFC 6455) does not support custom headers during
  /// the initial HTTP upgrade handshake. This means the token may be
  /// logged by intermediary proxies or show up in server access logs.
  ///
  /// Mitigations in place:
  /// - Access tokens are short-lived (15 min TTL).
  /// - The connection uses WSS (TLS) in production, so the URL is
  ///   encrypted in transit.
  ///
  /// **TODO (planned)**: replace with single-use, short-lived WS
  /// tickets (POST /api/v1/auth/ws-ticket -> one-time token with
  /// ~30s TTL) so that the long-lived JWT never appears in the URL.
  /// Tracked for the next auth iteration.
  String _buildWsUrl(String token) {
    const httpUrl = ApiClient.baseUrl;
    final wsScheme = httpUrl.startsWith('https') ? 'wss' : 'ws';
    final host = httpUrl
        .replaceFirst('http://', '')
        .replaceFirst('https://', '');
    return '$wsScheme://$host/api/v1/ws?token=$token';
  }

  void _send(Map<String, dynamic> payload) {
    if (_channel == null || !_isConnected) return;
    _channel!.sink.add(jsonEncode(payload));
  }

  void _startHeartbeat() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = Timer.periodic(
      const Duration(seconds: 30),
      (_) => _send({'type': 'heartbeat'}),
    );
  }

  void _onMessage(dynamic raw) {
    try {
      final data = jsonDecode(raw as String) as Map<String, dynamic>;
      _eventController.add(data);
    } catch (_) {
      // Silently ignore malformed frames
    }
  }

  void _onError(Object _) {
    _isConnected = false;
    _scheduleReconnect();
  }

  void _onDone() {
    _isConnected = false;
    _heartbeatTimer?.cancel();
    if (!_disposed) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    if (_disposed) return;
    _reconnectTimer?.cancel();

    final delay = min(
      pow(2, _reconnectAttempts).toInt(),
      _maxReconnectDelay,
    );
    _reconnectAttempts++;

    _reconnectTimer = Timer(
      Duration(seconds: delay),
      () => connect(),
    );
  }
}
