import '../../../core/network/api_client.dart';

/// Result of initiating a call.
class InitiateCallResult {
  const InitiateCallResult({
    required this.callId,
    required this.roomName,
    required this.token,
  });

  final String callId;
  final String roomName;
  final String token;

  factory InitiateCallResult.fromJson(Map<String, dynamic> json) {
    return InitiateCallResult(
      callId: json['call_id'] as String,
      roomName: json['room_name'] as String,
      token: json['token'] as String,
    );
  }
}

/// Result of accepting a call.
class AcceptCallResult {
  const AcceptCallResult({required this.token, required this.roomName});

  final String token;
  final String roomName;

  factory AcceptCallResult.fromJson(Map<String, dynamic> json) {
    return AcceptCallResult(
      token: json['token'] as String,
      roomName: json['room_name'] as String,
    );
  }
}

/// API calls for the call feature.
class CallRepository {
  const CallRepository(this._api);
  final ApiClient _api;

  Future<InitiateCallResult> initiateCall({
    required String conversationId,
    required String recipientId,
    String type = 'audio',
  }) async {
    final response = await _api.post(
      '/api/v1/calls/initiate',
      data: {
        'conversation_id': conversationId,
        'recipient_id': recipientId,
        'type': type,
      },
    );
    return InitiateCallResult.fromJson(
      response.data as Map<String, dynamic>,
    );
  }

  Future<AcceptCallResult> acceptCall(String callId) async {
    final response = await _api.post('/api/v1/calls/$callId/accept');
    return AcceptCallResult.fromJson(
      response.data as Map<String, dynamic>,
    );
  }

  Future<void> declineCall(String callId) async {
    await _api.post('/api/v1/calls/$callId/decline');
  }

  Future<void> endCall(String callId, int duration) async {
    await _api.post(
      '/api/v1/calls/$callId/end',
      data: {'duration': duration},
    );
  }
}
