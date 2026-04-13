/// Cell state values returned by the backend's role-permissions
/// matrix. They mirror the `PermissionState` constants defined in
/// `backend/internal/domain/organization/permissions.go`:
///
///   - default_granted  — the static default grants this permission
///   - default_revoked  — the static default does NOT grant this
///   - granted_override — an org-level override flipped it on
///   - revoked_override — an org-level override flipped it off
///   - locked           — non-overridable (Owner-only, cannot be edited)
abstract final class RolePermissionCellState {
  static const defaultGranted = 'default_granted';
  static const defaultRevoked = 'default_revoked';
  static const grantedOverride = 'granted_override';
  static const revokedOverride = 'revoked_override';
  static const locked = 'locked';
}

/// A single permission cell inside a role row. Carries the human-
/// readable label and description so the UI does not need to duplicate
/// the catalogue locally — the backend is the source of truth for the
/// defaults (the mobile i18n layer only overrides role/group labels).
class RolePermissionCell {
  final String key;
  final String group;
  final String label;
  final String description;
  final bool granted;
  final String state;
  final bool locked;

  const RolePermissionCell({
    required this.key,
    required this.group,
    required this.label,
    required this.description,
    required this.granted,
    required this.state,
    required this.locked,
  });

  /// Convenience: this cell has been customized away from the default.
  bool get isOverridden =>
      state == RolePermissionCellState.grantedOverride ||
      state == RolePermissionCellState.revokedOverride;

  factory RolePermissionCell.fromJson(Map<String, dynamic> json) {
    return RolePermissionCell(
      key: (json['key'] as String?) ?? '',
      group: (json['group'] as String?) ?? 'other',
      label: (json['label'] as String?) ?? '',
      description: (json['description'] as String?) ?? '',
      granted: (json['granted'] as bool?) ?? false,
      state: (json['state'] as String?) ??
          RolePermissionCellState.defaultRevoked,
      locked: (json['locked'] as bool?) ?? false,
    );
  }
}

/// One role row in the matrix: the role key + its label + every
/// permission cell the backend knows about (including locked ones).
class RolePermissionsRow {
  final String role;
  final String label;
  final String description;
  final List<RolePermissionCell> permissions;

  const RolePermissionsRow({
    required this.role,
    required this.label,
    required this.description,
    required this.permissions,
  });

  /// Returns `true` when at least one non-locked cell in this row
  /// carries an override. Used by the UI to render the "modified"
  /// badge next to the role tab.
  bool get hasOverrides {
    for (final p in permissions) {
      if (p.isOverridden) return true;
    }
    return false;
  }

  factory RolePermissionsRow.fromJson(Map<String, dynamic> json) {
    final rawPerms = (json['permissions'] as List<dynamic>?) ?? const [];
    return RolePermissionsRow(
      role: (json['role'] as String?) ?? '',
      label: (json['label'] as String?) ?? '',
      description: (json['description'] as String?) ?? '',
      permissions: rawPerms
          .cast<Map<String, dynamic>>()
          .map(RolePermissionCell.fromJson)
          .toList(),
    );
  }
}

/// Top-level payload of GET /api/v1/organizations/{id}/role-permissions.
/// The roles list is in the canonical order (Owner, Admin, Member,
/// Viewer) so the UI can index into it without resorting.
class RolePermissionsMatrix {
  final List<RolePermissionsRow> roles;

  const RolePermissionsMatrix({required this.roles});

  RolePermissionsRow? rowFor(String role) {
    for (final r in roles) {
      if (r.role == role) return r;
    }
    return null;
  }

  factory RolePermissionsMatrix.fromJson(Map<String, dynamic> json) {
    final rawRoles = (json['roles'] as List<dynamic>?) ?? const [];
    return RolePermissionsMatrix(
      roles: rawRoles
          .cast<Map<String, dynamic>>()
          .map(RolePermissionsRow.fromJson)
          .toList(),
    );
  }
}

/// Response payload of the PATCH save endpoint. Bundles the counts
/// and the refreshed matrix so a single round-trip keeps everything
/// in sync.
class RolePermissionsUpdateResult {
  final String role;
  final List<String> grantedKeys;
  final List<String> revokedKeys;
  final int affectedMembers;
  final RolePermissionsMatrix? matrix;

  const RolePermissionsUpdateResult({
    required this.role,
    required this.grantedKeys,
    required this.revokedKeys,
    required this.affectedMembers,
    this.matrix,
  });

  factory RolePermissionsUpdateResult.fromJson(Map<String, dynamic> json) {
    final rawGranted = (json['granted_keys'] as List<dynamic>?) ?? const [];
    final rawRevoked = (json['revoked_keys'] as List<dynamic>?) ?? const [];
    final rawMatrix = json['matrix'];
    return RolePermissionsUpdateResult(
      role: (json['role'] as String?) ?? '',
      grantedKeys: rawGranted.cast<String>().toList(),
      revokedKeys: rawRevoked.cast<String>().toList(),
      affectedMembers: (json['affected_members'] as int?) ?? 0,
      matrix: rawMatrix is Map<String, dynamic>
          ? RolePermissionsMatrix.fromJson(rawMatrix)
          : null,
    );
  }
}
