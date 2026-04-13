/// A pending invitation row from
/// `GET /api/v1/organizations/{orgID}/invitations`.
///
/// Mirrors the slim `InvitationResponse` returned by the backend
/// (Phase 2). The secret token is intentionally absent — it only ever
/// leaves the backend through the email link, never through the API
/// list endpoint.
class PendingInvitation {
  final String id;
  final String organizationId;
  final String email;
  final String firstName;
  final String lastName;
  final String title;
  final String role;
  final String status;
  final String invitedByUserId;
  final DateTime sentAt;
  final DateTime expiresAt;

  const PendingInvitation({
    required this.id,
    required this.organizationId,
    required this.email,
    required this.firstName,
    required this.lastName,
    required this.title,
    required this.role,
    required this.status,
    required this.invitedByUserId,
    required this.sentAt,
    required this.expiresAt,
  });

  /// Best-effort full name. Falls back to the email when neither
  /// first nor last name is present.
  String displayName() {
    final fl = '$firstName $lastName'.trim();
    if (fl.isNotEmpty) return fl;
    return email;
  }

  factory PendingInvitation.fromJson(Map<String, dynamic> json) {
    return PendingInvitation(
      id: (json['id'] as String?) ?? '',
      organizationId: (json['organization_id'] as String?) ?? '',
      email: (json['email'] as String?) ?? '',
      firstName: (json['first_name'] as String?) ?? '',
      lastName: (json['last_name'] as String?) ?? '',
      title: (json['title'] as String?) ?? '',
      role: (json['role'] as String?) ?? 'member',
      status: (json['status'] as String?) ?? 'pending',
      invitedByUserId: (json['invited_by_user_id'] as String?) ?? '',
      sentAt: DateTime.tryParse((json['created_at'] as String?) ?? '') ??
          DateTime.now(),
      expiresAt: DateTime.tryParse((json['expires_at'] as String?) ?? '') ??
          DateTime.now(),
    );
  }
}
