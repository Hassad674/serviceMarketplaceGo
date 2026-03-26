import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:flutter/foundation.dart';
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
class MessagingWsService {
  final SecureStorageService _storage;

  WebSocketChannel? _channel;
  Timer? _heartbeatTimer;
  Timer? _reconnectTimer;

  final _eventController =
      StreamController<Map<String, dynamic>>.broadcast();

  bool _isConnected = false;
  bool _disposed = false;
  int _reconnectAttempts = 0;
  static const int _maxReconnectDelay = 30; // seconds

  MessagingWsService({required SecureStorageService storage})
      : _storage = storage;

  /// Stream of incoming WebSocket events (JSON maps).
  ///
  /// Each event has a `type` field:
  /// `new_message`, `typing`, `status_update`, `unread_count`,
  /// `message_edited`, `message_deleted`.
  Stream<Map<String, dynamic>> get events => _eventController.stream;

  /// Whether the WebSocket is currently connected.
  bool get isConnected => _isConnected;

  /// Establishes the WebSocket connection.
  ///
  /// Retrieves the JWT from secure storage and connects with it
  /// as a query parameter.
  Future<void> connect() async {
    if (_disposed) return;

    final token = await _storage.getAccessToken();
    if (token == null) return;

    final wsUrl = _buildWsUrl(token);

    try {
      _channel = WebSocketChannel.connect(Uri.parse(wsUrl));
      await _channel!.ready;
      _isConnected = true;
      _reconnectAttempts = 0;

      _startHeartbeat();

      _channel!.stream.listen(
        _onMessage,
        onError: _onError,
        onDone: _onDone,
      );
    } catch (e) {
      debugPrint('WebSocket connect failed: $e');
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
  }

  /// Permanently disposes the service. Cannot reconnect after this.
  void dispose() {
    _disposed = true;
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
    _send({'type': 'sync', 'last_seqs': lastSeqs});
  }

  // ---------------------------------------------------------------------------
  // Internal
  // ---------------------------------------------------------------------------

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
    } catch (e) {
      debugPrint('WebSocket parse error: $e');
    }
  }

  void _onError(Object error) {
    debugPrint('WebSocket error: $error');
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

    debugPrint('WebSocket reconnecting in ${delay}s '
        '(attempt $_reconnectAttempts)');

    _reconnectTimer = Timer(
      Duration(seconds: delay),
      () => connect(),
    );
  }
}
