import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../../core/network/api_client.dart';
import '../../../core/storage/secure_storage.dart';

/// Provides the singleton [MessagingWsService].
final messagingWsServiceProvider = Provider<MessagingWsService>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  final storage = ref.watch(secureStorageProvider);
  return MessagingWsService(apiClient: apiClient, storage: storage);
});

/// WebSocket service for real-time messaging events.
///
/// Authenticates via a single-use `ws_token` ticket fetched from
/// `POST /api/v1/auth/ws-token` (SEC-15) and connects to
/// `ws://{API_URL}/api/v1/ws?ws_token={ticket}`. The ticket has a
/// ~30-second TTL — even if it leaks into proxy logs, the credential
/// is useless almost immediately.
///
/// Exposes a stream of incoming JSON events (new_message, typing,
/// status_update, etc.). Handles heartbeat (every 30s) and automatic
/// reconnection with exponential backoff on disconnect.
///
/// On successful (re)connection, emits a synthetic `{"type":"reconnected"}`
/// event so that listeners can refresh stale state (e.g. presence).
class MessagingWsService {
  final ApiClient _api;
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

  MessagingWsService({
    required ApiClient apiClient,
    required SecureStorageService storage,
  })  : _api = apiClient,
        _storage = storage {
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
  /// SEC-15: fetches a single-use `ws_token` from
  /// `POST /api/v1/auth/ws-token` (Bearer-authenticated through the
  /// ApiClient interceptor) and connects with that ticket as a query
  /// parameter. The long-lived JWT never appears in any URL or
  /// access log.
  ///
  /// On successful reconnection, emits a synthetic `reconnected`
  /// event for stale-state refresh.
  Future<void> connect() async {
    if (_disposed || _isConnecting || _isConnected) return;
    _isConnecting = true;

    try {
      // No bearer token = no chance of fetching a ws ticket.
      final accessToken = await _storage.getAccessToken();
      if (accessToken == null) {
        _isConnecting = false;
        return;
      }

      final wsTicket = await _fetchWsTicket();
      if (wsTicket == null) {
        _isConnecting = false;
        _scheduleReconnect();
        return;
      }

      final wsUrl = _buildWsUrl(wsTicket);

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

  /// Exchanges the stored Bearer JWT for a short-lived single-use
  /// WebSocket ticket. Returns null when the request fails — the
  /// caller schedules a reconnect attempt rather than aborting
  /// permanently. The `ApiClient` interceptor automatically adds the
  /// `Authorization: Bearer …` header from secure storage.
  Future<String?> _fetchWsTicket() async {
    try {
      final response = await _api.get<Map<String, dynamic>>('/api/v1/auth/ws-token');
      final data = response.data;
      if (data == null) return null;
      final token = data['token'];
      if (token is String && token.isNotEmpty) return token;
      return null;
    } on DioException {
      return null;
    } catch (_) {
      return null;
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

  /// Builds the WebSocket URL with the single-use ws_token as a
  /// query parameter.
  ///
  /// SEC-15: the long-lived JWT no longer appears in the URL — only
  /// the short-lived (~30s) ws_token does. Even if the URL ends up
  /// in proxy/access logs the credential is useless almost
  /// immediately.
  ///
  /// The WebSocket protocol (RFC 6455) does not support custom
  /// headers during the upgrade handshake, so a query-string ticket
  /// is the standard pattern. WSS (TLS) protects the URL in transit
  /// in production.
  String _buildWsUrl(String wsTicket) {
    const httpUrl = ApiClient.baseUrl;
    final wsScheme = httpUrl.startsWith('https') ? 'wss' : 'ws';
    final host = httpUrl
        .replaceFirst('http://', '')
        .replaceFirst('https://', '');
    return '$wsScheme://$host/api/v1/ws?ws_token=$wsTicket';
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
