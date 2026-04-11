/// Identity block embedded inside a [TeamMember]. Mirrors the slim
/// `MemberUserResponse` returned by the backend (R13) so the team
/// list can render an avatar fallback + display name + email without
/// a second round-trip per row.
class TeamMemberUser {
  final String id;
  final String email;
  final String displayName;
  final String firstName;
  final String lastName;

  const TeamMemberUser({
    required this.id,
    required this.email,
    required this.displayName,
    required this.firstName,
    required this.lastName,
  });

  factory TeamMemberUser.fromJson(Map<String, dynamic> json) {
    return TeamMemberUser(
      id: (json['id'] as String?) ?? '',
      email: (json['email'] as String?) ?? '',
      displayName: (json['display_name'] as String?) ?? '',
      firstName: (json['first_name'] as String?) ?? '',
      lastName: (json['last_name'] as String?) ?? '',
    );
  }
}

/// Membership row from `GET /api/v1/organizations/{orgID}/members`.
///
/// The user block is optional because older backend versions did not
/// join the users table — the mobile UI handles a missing block by
/// falling back to a generic "Member" label.
class TeamMember {
  final String id;
  final String organizationId;
  final String userId;
  final String role;
  final String title;
  final DateTime joinedAt;
  final TeamMemberUser? user;

  const TeamMember({
    required this.id,
    required this.organizationId,
    required this.userId,
    required this.role,
    required this.title,
    required this.joinedAt,
    this.user,
  });

  /// Best-effort display name. Prefers the user's display_name, then
  /// first+last, then email, and finally a generic fallback.
  String displayLabel(String fallback) {
    final u = user;
    if (u == null) return fallback;
    if (u.displayName.isNotEmpty) return u.displayName;
    final fl = '${u.firstName} ${u.lastName}'.trim();
    if (fl.isNotEmpty) return fl;
    if (u.email.isNotEmpty) return u.email;
    return fallback;
  }

  /// Initials for the avatar fallback. Returns "?" when nothing
  /// resolvable is available.
  String initials() {
    final u = user;
    if (u == null) return '?';
    final f = u.firstName.isNotEmpty ? u.firstName.substring(0, 1) : '';
    final l = u.lastName.isNotEmpty ? u.lastName.substring(0, 1) : '';
    final raw = (f + l).toUpperCase();
    if (raw.isNotEmpty) return raw;
    if (u.displayName.isNotEmpty) {
      return u.displayName.substring(0, 1).toUpperCase();
    }
    if (u.email.isNotEmpty) {
      return u.email.substring(0, 1).toUpperCase();
    }
    return '?';
  }

  factory TeamMember.fromJson(Map<String, dynamic> json) {
    return TeamMember(
      id: (json['id'] as String?) ?? '',
      organizationId: (json['organization_id'] as String?) ?? '',
      userId: (json['user_id'] as String?) ?? '',
      role: (json['role'] as String?) ?? 'member',
      title: (json['title'] as String?) ?? '',
      joinedAt: DateTime.tryParse((json['joined_at'] as String?) ?? '') ??
          DateTime.now(),
      user: json['user'] is Map<String, dynamic>
          ? TeamMemberUser.fromJson(json['user'] as Map<String, dynamic>)
          : null,
    );
  }
}
