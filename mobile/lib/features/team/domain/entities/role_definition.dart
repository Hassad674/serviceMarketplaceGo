/// Permission metadata returned by GET
/// /api/v1/organizations/role-definitions. Used by the mobile team
/// screen's "About roles" section to show what each role can do.
class RoleDefinitionPermission {
  final String key;
  final String group;
  final String label;
  final String description;

  const RoleDefinitionPermission({
    required this.key,
    required this.group,
    required this.label,
    required this.description,
  });

  factory RoleDefinitionPermission.fromJson(Map<String, dynamic> json) {
    return RoleDefinitionPermission(
      key: (json['key'] as String?) ?? '',
      group: (json['group'] as String?) ?? 'other',
      label: (json['label'] as String?) ?? '',
      description: (json['description'] as String?) ?? '',
    );
  }
}

/// A single role row from the role-definitions endpoint. Carries the
/// English defaults; the mobile UI is English-only so the defaults
/// are rendered directly.
class RoleDefinition {
  final String key;
  final String label;
  final String description;
  final List<String> permissions;

  const RoleDefinition({
    required this.key,
    required this.label,
    required this.description,
    required this.permissions,
  });

  factory RoleDefinition.fromJson(Map<String, dynamic> json) {
    final rawPerms = (json['permissions'] as List<dynamic>?) ?? const [];
    return RoleDefinition(
      key: (json['key'] as String?) ?? '',
      label: (json['label'] as String?) ?? '',
      description: (json['description'] as String?) ?? '',
      permissions: rawPerms.cast<String>().toList(),
    );
  }
}

/// Full payload returned by the role-definitions endpoint. Roles
/// reference permissions by key; the [permissions] catalogue carries
/// the human-readable labels so the UI can render them once without
/// duplicating strings inside every role row.
class RoleDefinitionsPayload {
  final List<RoleDefinition> roles;
  final List<RoleDefinitionPermission> permissions;

  const RoleDefinitionsPayload({
    required this.roles,
    required this.permissions,
  });

  /// Returns the permission metadata for a key, or `null` when the
  /// key is unknown to the catalogue (e.g. backend ahead of mobile).
  RoleDefinitionPermission? permissionByKey(String key) {
    for (final p in permissions) {
      if (p.key == key) return p;
    }
    return null;
  }

  factory RoleDefinitionsPayload.fromJson(Map<String, dynamic> json) {
    final rawRoles = (json['roles'] as List<dynamic>?) ?? const [];
    final rawPerms = (json['permissions'] as List<dynamic>?) ?? const [];
    return RoleDefinitionsPayload(
      roles: rawRoles
          .cast<Map<String, dynamic>>()
          .map(RoleDefinition.fromJson)
          .toList(),
      permissions: rawPerms
          .cast<Map<String, dynamic>>()
          .map(RoleDefinitionPermission.fromJson)
          .toList(),
    );
  }
}
