/// DeletionStatus carries the GDPR right-to-erasure schedule for the
/// signed-in user (P5).
///
/// All fields are nullable: a healthy account has both [scheduledAt]
/// and [hardDeleteAt] set to null. Once the user confirms deletion via
/// the email link, the backend stamps the row's deleted_at and exposes
/// both dates so the dashboard banner can render the 30-day countdown
/// without a separate fetch.
class DeletionStatus {
  final DateTime? scheduledAt;
  final DateTime? hardDeleteAt;

  const DeletionStatus({this.scheduledAt, this.hardDeleteAt});

  /// True when the account is currently in its 30-day GDPR cooldown.
  bool get isPending => scheduledAt != null;

  factory DeletionStatus.fromJson(Map<String, dynamic> json) {
    return DeletionStatus(
      scheduledAt: json['deleted_at'] != null
          ? DateTime.parse(json['deleted_at'] as String)
          : null,
      hardDeleteAt: json['hard_delete_at'] != null
          ? DateTime.parse(json['hard_delete_at'] as String)
          : null,
    );
  }

  static const DeletionStatus none = DeletionStatus();
}

/// Result of POST /me/account/request-deletion. The handler echoes
/// the email back so the UI can show "we sent the link to xx@yy.com".
class RequestDeletionResult {
  final String emailSentTo;
  final DateTime expiresAt;

  const RequestDeletionResult({
    required this.emailSentTo,
    required this.expiresAt,
  });

  factory RequestDeletionResult.fromJson(Map<String, dynamic> json) {
    return RequestDeletionResult(
      emailSentTo: json['email_sent_to'] as String,
      expiresAt: DateTime.parse(json['expires_at'] as String),
    );
  }
}

/// Lightweight org descriptor returned by the 409 owner-blocked
/// payload (Decision 6 of the P5 brief).
class BlockedOrg {
  final String orgId;
  final String orgName;
  final int memberCount;
  final List<({String userId, String email})> availableAdmins;
  final List<String> actions;

  const BlockedOrg({
    required this.orgId,
    required this.orgName,
    required this.memberCount,
    required this.availableAdmins,
    required this.actions,
  });

  factory BlockedOrg.fromJson(Map<String, dynamic> json) {
    final admins = (json['available_admins'] as List? ?? const [])
        .map(
          (a) => (
            userId: (a as Map<String, dynamic>)['user_id'] as String,
            email: a['email'] as String,
          ),
        )
        .toList(growable: false);
    return BlockedOrg(
      orgId: json['org_id'] as String,
      orgName: json['org_name'] as String,
      memberCount: (json['member_count'] as num).toInt(),
      availableAdmins: admins,
      actions: (json['actions'] as List? ?? const [])
          .map((a) => a as String)
          .toList(growable: false),
    );
  }
}

/// Typed representation of the 409 conflict body.
class OwnerBlockedException implements Exception {
  final List<BlockedOrg> blockedOrgs;
  const OwnerBlockedException(this.blockedOrgs);

  @override
  String toString() =>
      'OwnerBlockedException(blocked_orgs=${blockedOrgs.length})';
}
