import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/team_member.dart';
import '../providers/team_provider.dart';
import '../widgets/about_roles_section.dart';
import '../widgets/team_member_tile.dart';

/// Read-only team screen for the mobile app (R13 minimal scope).
///
/// Surfaces three things:
///   1. Member list with identity (avatar fallback + name + role)
///   2. Role badge column so users can see who is what
///   3. "About roles" expandable section listing every role and its
///      permissions
///
/// Edit / invite / transfer flows are deferred to a later phase —
/// users still perform those actions on the web app.
class TeamScreen extends ConsumerWidget {
  const TeamScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final orgId = ref.watch(currentOrganizationIdProvider);
    final membersAsync = ref.watch(teamMembersProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Team'),
        elevation: 0,
      ),
      body: orgId == null
          ? _NoOrganizationState(appColors: appColors)
          : RefreshIndicator(
              onRefresh: () async {
                ref.invalidate(teamMembersProvider);
                ref.invalidate(roleDefinitionsProvider);
                await ref.read(teamMembersProvider.future);
              },
              child: membersAsync.when(
                data: (members) => _TeamBody(members: members),
                loading: () => const Center(child: CircularProgressIndicator()),
                error: (err, _) => _ErrorState(
                  message: 'Could not load team',
                  onRetry: () => ref.invalidate(teamMembersProvider),
                ),
              ),
            ),
    );
  }
}

class _TeamBody extends StatelessWidget {
  final List<TeamMember> members;

  const _TeamBody({required this.members});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text(
          'Members',
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 8),
        if (members.isEmpty)
          _EmptyMembers(appColors: appColors)
        else
          ...members.map(
            (m) => Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: TeamMemberTile(member: m),
            ),
          ),
        const SizedBox(height: 24),
        const AboutRolesSection(),
        const SizedBox(height: 24),
      ],
    );
  }
}

class _EmptyMembers extends StatelessWidget {
  final AppColors? appColors;

  const _EmptyMembers({required this.appColors});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      alignment: Alignment.center,
      child: Text(
        'No members',
        style: theme.textTheme.bodyMedium?.copyWith(
          color: appColors?.mutedForeground,
        ),
      ),
    );
  }
}

class _NoOrganizationState extends StatelessWidget {
  final AppColors? appColors;

  const _NoOrganizationState({required this.appColors});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.group_outlined,
              size: 48,
              color: appColors?.mutedForeground ?? Colors.grey,
            ),
            const SizedBox(height: 16),
            Text(
              'No organization',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              'You are not attached to any organization yet.',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodySmall?.copyWith(
                color: appColors?.mutedForeground,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  final String message;
  final VoidCallback onRetry;

  const _ErrorState({required this.message, required this.onRetry});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.error_outline, size: 48, color: Colors.redAccent),
          const SizedBox(height: 12),
          Text(message),
          const SizedBox(height: 12),
          ElevatedButton(
            onPressed: onRetry,
            child: const Text('Retry'),
          ),
        ],
      ),
    );
  }
}
