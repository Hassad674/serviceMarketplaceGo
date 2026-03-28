/// Represents a notification from the backend.
///
/// Maps to the backend notification response:
/// `GET /api/v1/notifications`
class AppNotification {
  final String id;
  final String userId;
  final String type;
  final String title;
  final String body;
  final Map<String, dynamic> data;
  final DateTime? readAt;
  final DateTime createdAt;

  const AppNotification({
    required this.id,
    required this.userId,
    required this.type,
    required this.title,
    required this.body,
    this.data = const {},
    this.readAt,
    required this.createdAt,
  });

  bool get isRead => readAt != null;

  factory AppNotification.fromJson(Map<String, dynamic> json) {
    return AppNotification(
      id: json['id'] as String,
      userId: json['user_id'] as String,
      type: json['type'] as String,
      title: json['title'] as String,
      body: (json['body'] as String?) ?? '',
      data: (json['data'] as Map<String, dynamic>?) ?? {},
      readAt: json['read_at'] != null
          ? DateTime.parse(json['read_at'] as String)
          : null,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

/// User preference for a specific notification type.
///
/// Maps to backend `GET /api/v1/notifications/preferences`.
class NotificationPreference {
  final String type;
  final bool inApp;
  final bool push;
  final bool email;

  const NotificationPreference({
    required this.type,
    required this.inApp,
    required this.push,
    required this.email,
  });

  factory NotificationPreference.fromJson(Map<String, dynamic> json) {
    return NotificationPreference(
      type: json['type'] as String,
      inApp: json['in_app'] as bool,
      push: json['push'] as bool,
      email: json['email'] as bool,
    );
  }

  Map<String, dynamic> toJson() => {
        'type': type,
        'in_app': inApp,
        'push': push,
        'email': email,
      };
}
