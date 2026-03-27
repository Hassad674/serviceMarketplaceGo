/// Represents a call's current state.
enum CallStatus { idle, ringingOutgoing, ringingIncoming, active, ended }

/// Represents call type.
enum CallType { audio, video }

/// Data class holding the state of an active or incoming call.
class CallEntity {
  const CallEntity({
    required this.callId,
    required this.conversationId,
    required this.initiatorId,
    required this.recipientId,
    required this.callType,
    this.roomName = '',
    this.token = '',
    this.startedAt,
  });

  final String callId;
  final String conversationId;
  final String initiatorId;
  final String recipientId;
  final CallType callType;
  final String roomName;
  final String token;
  final DateTime? startedAt;

  CallEntity copyWith({
    String? callId,
    String? conversationId,
    String? initiatorId,
    String? recipientId,
    CallType? callType,
    String? roomName,
    String? token,
    DateTime? startedAt,
  }) {
    return CallEntity(
      callId: callId ?? this.callId,
      conversationId: conversationId ?? this.conversationId,
      initiatorId: initiatorId ?? this.initiatorId,
      recipientId: recipientId ?? this.recipientId,
      callType: callType ?? this.callType,
      roomName: roomName ?? this.roomName,
      token: token ?? this.token,
      startedAt: startedAt ?? this.startedAt,
    );
  }
}
