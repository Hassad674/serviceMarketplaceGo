import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/role_definition.dart';
import '../providers/team_provider.dart';

/// Collapsible "About roles" section listing every role with the
/// permissions it grants. Loaded lazily from the role-definitions
/// endpoint and cached in [roleDefinitionsProvider].
class AboutRolesSection extends ConsumerStatefulWidget {
  const AboutRolesSection({super.key});

  @override
  ConsumerState<AboutRolesSection> createState() => _AboutRolesSectionState();
}

class _AboutRolesSectionState extends ConsumerState<AboutRolesSection> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final defs = ref.watch(roleDefinitionsProvider);

    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Column(
        children: [
          InkWell(
            onTap: () => setState(() => _expanded = !_expanded),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  Container(
                    height: 36,
                    width: 36,
                    decoration: const BoxDecoration(
                      color: Color(0xFFFFE4E6),
                      shape: BoxShape.circle,
                    ),
                    alignment: Alignment.center,
                    child: const Icon(
                      Icons.info_outline,
                      color: Color(0xFFE11D48),
                      size: 18,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'About roles',
                          style: theme.textTheme.titleSmall?.copyWith(
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          'What each role can do in the organization.',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: appColors?.mutedForeground,
                          ),
                        ),
                      ],
                    ),
                  ),
                  Icon(
                    _expanded ? Icons.expand_less : Icons.expand_more,
                    color: appColors?.mutedForeground,
                  ),
                ],
              ),
            ),
          ),
          if (_expanded)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
              child: defs.when(
                data: (payload) => _RoleList(payload: payload),
                loading: () => const Padding(
                  padding: EdgeInsets.symmetric(vertical: 24),
                  child: Center(child: CircularProgressIndicator()),
                ),
                error: (err, _) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 16),
                  child: Text(
                    'Could not load role catalogue',
                    style: TextStyle(
                      color: appColors?.mutedForeground,
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _RoleList extends StatelessWidget {
  final RoleDefinitionsPayload payload;

  const _RoleList({required this.payload});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        for (final role in payload.roles) ...[
          _RoleCard(role: role, payload: payload),
          const SizedBox(height: 8),
        ],
      ],
    );
  }
}

class _RoleCard extends StatelessWidget {
  final RoleDefinition role;
  final RoleDefinitionsPayload payload;

  const _RoleCard({required this.role, required this.payload});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final grouped = _groupPermissions();

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              _RoleIcon(role: role.key),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  role.label,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              Text(
                '${role.permissions.length} perms',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: appColors?.mutedForeground,
                  fontSize: 11,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            role.description,
            style: theme.textTheme.bodySmall?.copyWith(
              color: appColors?.mutedForeground,
              height: 1.45,
            ),
          ),
          if (grouped.isNotEmpty) ...[
            const SizedBox(height: 12),
            for (final entry in grouped.entries) ...[
              Text(
                _groupLabel(entry.key),
                style: const TextStyle(
                  color: Color(0xFFE11D48),
                  fontSize: 10,
                  fontWeight: FontWeight.w800,
                  letterSpacing: 0.6,
                ),
              ),
              const SizedBox(height: 4),
              for (final p in entry.value)
                Padding(
                  padding: const EdgeInsets.only(left: 4, bottom: 2),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('•  ', style: TextStyle(fontSize: 11)),
                      Expanded(
                        child: Text(
                          p.label,
                          style: theme.textTheme.bodySmall?.copyWith(
                            fontSize: 11,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              const SizedBox(height: 6),
            ],
          ],
        ],
      ),
    );
  }

  Map<String, List<RoleDefinitionPermission>> _groupPermissions() {
    final map = <String, List<RoleDefinitionPermission>>{};
    for (final key in role.permissions) {
      final meta = payload.permissionByKey(key);
      if (meta == null) continue;
      map.putIfAbsent(meta.group, () => []).add(meta);
    }
    return map;
  }

  String _groupLabel(String key) {
    switch (key) {
      case 'team':
        return 'TEAM';
      case 'org_profile':
        return 'PUBLIC PROFILE';
      case 'jobs':
        return 'JOBS';
      case 'proposals':
        return 'PROPOSALS';
      case 'messaging':
        return 'MESSAGING';
      case 'reviews':
        return 'REVIEWS';
      case 'wallet':
        return 'WALLET';
      case 'billing':
        return 'BILLING';
      case 'kyc':
        return 'KYC';
      case 'danger':
        return 'DANGER ZONE';
      default:
        return key.toUpperCase();
    }
  }
}

class _RoleIcon extends StatelessWidget {
  final String role;

  const _RoleIcon({required this.role});

  @override
  Widget build(BuildContext context) {
    final IconData icon;
    final Color background;
    final Color foreground;
    switch (role) {
      case 'owner':
        icon = Icons.workspace_premium_outlined;
        background = const Color(0xFFFEF3C7);
        foreground = const Color(0xFFB45309);
        break;
      case 'admin':
        icon = Icons.shield_outlined;
        background = const Color(0xFFEDE9FE);
        foreground = const Color(0xFF6D28D9);
        break;
      case 'member':
        icon = Icons.person_outline;
        background = const Color(0xFFDBEAFE);
        foreground = const Color(0xFF1D4ED8);
        break;
      case 'viewer':
      default:
        icon = Icons.visibility_outlined;
        background = const Color(0xFFE2E8F0);
        foreground = const Color(0xFF334155);
        break;
    }
    return Container(
      height: 32,
      width: 32,
      decoration: BoxDecoration(
        color: background,
        shape: BoxShape.circle,
      ),
      alignment: Alignment.center,
      child: Icon(icon, color: foreground, size: 16),
    );
  }
}
