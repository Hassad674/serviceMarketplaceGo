import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/theme/theme_provider.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../expertise/presentation/widgets/expertise_section_widget.dart';
import '../../../organization_shared/presentation/providers/organization_shared_providers.dart';
import '../../../organization_shared/presentation/widgets/shared_languages_section_widget.dart';
import '../../../organization_shared/presentation/widgets/shared_location_section_widget.dart';
import '../../../organization_shared/presentation/widgets/shared_photo_upload_widget.dart';
import '../../../portfolio/presentation/widgets/portfolio_grid_widget.dart';
import '../../../project_history/presentation/widgets/project_history_widget.dart';
import '../../../skill/presentation/widgets/skills_section_widget.dart';
import '../../domain/entities/freelance_profile.dart';
import '../providers/freelance_profile_providers.dart';
import '../widgets/freelance_pricing_section_widget.dart';
import '../widgets/freelance_profile_header.dart';

/// Editable freelance profile screen mounted on `/profile` for
/// provider_personal users. Renders persona-specific fields from
/// the freelance aggregate plus the shared organization block so
/// the user edits everything in one place.
class FreelanceProfileScreen extends ConsumerWidget {
  const FreelanceProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(freelanceProfileProvider);
    final sharedAsync = ref.watch(organizationSharedProvider);
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.myProfile),
      ),
      body: SafeArea(
        child: profileAsync.when(
          loading: () => const _LoadingState(),
          error: (_, __) => _ErrorState(
            onRetry: () => ref.invalidate(freelanceProfileProvider),
          ),
          data: (profile) => _FreelanceProfileBody(
            profile: profile,
            sharedAsync: sharedAsync,
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Body
// ---------------------------------------------------------------------------

class _FreelanceProfileBody extends ConsumerWidget {
  const _FreelanceProfileBody({
    required this.profile,
    required this.sharedAsync,
  });

  final FreelanceProfile profile;
  final AsyncValue<dynamic> sharedAsync;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final authState = ref.watch(authProvider);
    final canEdit = ref.watch(
      hasPermissionProvider(OrgPermission.orgProfileEdit),
    );
    final user = authState.user;
    final displayName = user?['display_name'] as String? ??
        user?['first_name'] as String? ??
        '';

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          // Header with shared photo + availability pill
          SharedPhotoUploadWidget(
            canEdit: canEdit,
            child: FreelanceProfileHeader(
              displayName: displayName,
              title: profile.title,
              photoUrl: profile.photoUrl,
              initials: _buildInitials(displayName),
              availabilityWireValue: profile.availabilityStatus,
            ),
          ),
          const SizedBox(height: 16),

          // Expertise + Skills — reuse existing feature widgets
          ExpertiseSectionWidget(
            orgType: 'provider_personal',
            initialDomains: profile.expertiseDomains,
            canEdit: canEdit,
            onSaved: () => ref.invalidate(freelanceProfileProvider),
          ),
          const SizedBox(height: 16),
          SkillsSectionWidget(
            orgType: 'provider_personal',
            expertiseKeys: profile.expertiseDomains,
            canEdit: canEdit,
            onSaved: () => ref.invalidate(freelanceProfileProvider),
          ),
          const SizedBox(height: 16),

          // Pricing (freelance variants only)
          FreelancePricingSectionWidget(canEdit: canEdit),
          const SizedBox(height: 16),

          // Shared org fields rendered only on the freelance edit screen
          sharedAsync.when(
            loading: () => const SizedBox.shrink(),
            error: (_, __) => const SizedBox.shrink(),
            data: (shared) => Column(
              children: [
                SharedLocationSectionWidget(
                  initial: shared,
                  canEdit: canEdit,
                  onSaved: () {
                    ref.invalidate(organizationSharedProvider);
                    ref.invalidate(freelanceProfileProvider);
                  },
                ),
                const SizedBox(height: 16),
                SharedLanguagesSectionWidget(
                  initial: shared,
                  canEdit: canEdit,
                  onSaved: () {
                    ref.invalidate(organizationSharedProvider);
                    ref.invalidate(freelanceProfileProvider);
                  },
                ),
                const SizedBox(height: 16),
              ],
            ),
          ),

          // Title section
          _SectionCard(
            title: l10n.professionalTitle,
            icon: Icons.badge_outlined,
            trailing: canEdit
                ? IconButton(
                    icon: const Icon(Icons.edit_outlined, size: 18),
                    onPressed: () =>
                        _openEditCoreSheet(context, ref, profile),
                  )
                : null,
            child: Text(
              profile.title.isNotEmpty ? profile.title : l10n.titlePlaceholder,
              style: theme.textTheme.bodyMedium?.copyWith(
                fontStyle:
                    profile.title.isEmpty ? FontStyle.italic : FontStyle.normal,
              ),
            ),
          ),
          const SizedBox(height: 16),

          // About section
          GestureDetector(
            onTap: canEdit
                ? () => _openEditAboutSheet(context, ref, profile.about)
                : null,
            child: _SectionCard(
              title: l10n.about,
              icon: Icons.info_outline,
              child: Text(
                profile.about.isNotEmpty
                    ? profile.about
                    : l10n.aboutPlaceholder,
                style: theme.textTheme.bodyMedium?.copyWith(
                  height: 1.5,
                  fontStyle: profile.about.isEmpty
                      ? FontStyle.italic
                      : FontStyle.normal,
                ),
              ),
            ),
          ),
          const SizedBox(height: 16),

          // Video section
          _VideoCard(
            videoUrl: profile.videoUrl,
            canEdit: canEdit,
            onManage: () => _openVideoManage(context),
          ),
          const SizedBox(height: 16),

          // Portfolio
          if (profile.organizationId.isNotEmpty) ...[
            PortfolioGridWidget(
              orgId: profile.organizationId,
              readOnly: false,
            ),
            const SizedBox(height: 16),
            ProjectHistoryWidget(orgId: profile.organizationId),
            const SizedBox(height: 16),
          ],

          // Dark mode toggle + logout
          const _DarkModeToggle(),
          const SizedBox(height: 24),
          _LogoutButton(
            onPressed: () async {
              await ref.read(authProvider.notifier).logout();
              if (context.mounted) context.go(RoutePaths.login);
            },
          ),
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  String _buildInitials(String name) {
    if (name.isEmpty) return '?';
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }

  void _openVideoManage(BuildContext context) {
    // Delegates to the existing upload_bottom_sheet flow — left as
    // a no-op here because the freelance screen inherits the legacy
    // upload widget via the shared header. A dedicated video
    // management surface can replace this once we migrate the
    // upload service to the split profile.
  }

  Future<void> _openEditAboutSheet(
    BuildContext context,
    WidgetRef ref,
    String currentAbout,
  ) async {
    final l10n = AppLocalizations.of(context)!;
    final controller = TextEditingController(text: currentAbout);
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => Padding(
        padding:
            EdgeInsets.only(bottom: MediaQuery.of(ctx).viewInsets.bottom),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(l10n.about, style: Theme.of(ctx).textTheme.titleLarge),
              const SizedBox(height: 16),
              TextField(
                controller: controller,
                maxLines: 5,
                maxLength: 1000,
                decoration: InputDecoration(
                  hintText: l10n.aboutEditHint,
                  border: const OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: ElevatedButton(
                  onPressed: () async {
                    final api = ref.read(apiClientProvider);
                    await api.put(
                      '/api/v1/freelance-profile',
                      data: {'about': controller.text},
                    );
                    ref.invalidate(freelanceProfileProvider);
                    if (ctx.mounted) Navigator.pop(ctx);
                  },
                  child: Text(l10n.save),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _openEditCoreSheet(
    BuildContext context,
    WidgetRef ref,
    FreelanceProfile profile,
  ) async {
    final l10n = AppLocalizations.of(context)!;
    final controller = TextEditingController(text: profile.title);
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => Padding(
        padding:
            EdgeInsets.only(bottom: MediaQuery.of(ctx).viewInsets.bottom),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                l10n.professionalTitle,
                style: Theme.of(ctx).textTheme.titleLarge,
              ),
              const SizedBox(height: 16),
              TextField(
                controller: controller,
                decoration: InputDecoration(
                  hintText: l10n.titlePlaceholder,
                  border: const OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: ElevatedButton(
                  onPressed: () async {
                    final ok = await ref
                        .read(freelanceCoreEditorProvider.notifier)
                        .save(
                          title: controller.text,
                          about: profile.about,
                          videoUrl: profile.videoUrl,
                        );
                    if (ok) ref.invalidate(freelanceProfileProvider);
                    if (ctx.mounted) Navigator.pop(ctx);
                  },
                  child: Text(l10n.save),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Helper widgets
// ---------------------------------------------------------------------------

class _SectionCard extends StatelessWidget {
  const _SectionCard({
    required this.title,
    required this.icon,
    required this.child,
    this.trailing,
  });

  final String title;
  final IconData icon;
  final Widget child;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 20, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Expanded(
                child: Text(title, style: theme.textTheme.titleMedium),
              ),
              if (trailing != null) trailing!,
            ],
          ),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

class _VideoCard extends StatelessWidget {
  const _VideoCard({
    required this.videoUrl,
    required this.canEdit,
    required this.onManage,
  });

  final String videoUrl;
  final bool canEdit;
  final VoidCallback onManage;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.videocam_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Text(l10n.presentationVideo, style: theme.textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 16),
          if (videoUrl.isNotEmpty)
            VideoPlayerWidget(videoUrl: videoUrl)
          else
            Container(
              width: double.infinity,
              padding: const EdgeInsets.symmetric(
                vertical: 24,
                horizontal: 16,
              ),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(AppTheme.radiusMd),
              ),
              child: Column(
                children: [
                  Icon(
                    Icons.videocam_outlined,
                    size: 40,
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                  const SizedBox(height: 12),
                  Text(
                    l10n.noVideo,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

class _LoadingState extends StatelessWidget {
  const _LoadingState();

  @override
  Widget build(BuildContext context) {
    return const Center(child: CircularProgressIndicator());
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.error_outline,
            size: 48,
            color: theme.colorScheme.error,
          ),
          const SizedBox(height: 12),
          Text(l10n.couldNotLoadProfile),
          const SizedBox(height: 12),
          ElevatedButton(onPressed: onRetry, child: Text(l10n.retry)),
        ],
      ),
    );
  }
}

class _DarkModeToggle extends ConsumerWidget {
  const _DarkModeToggle();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final themeMode = ref.watch(themeModeProvider);
    final isDark = themeMode == ThemeMode.dark;
    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: ListTile(
        leading: Icon(
          isDark ? Icons.dark_mode : Icons.light_mode,
          color: theme.colorScheme.primary,
        ),
        title: Text(l10n.darkMode),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        ),
        trailing: Switch(
          value: isDark,
          activeTrackColor: theme.colorScheme.primary,
          onChanged: (_) => ref.read(themeModeProvider.notifier).toggle(),
        ),
      ),
    );
  }
}

class _LogoutButton extends StatelessWidget {
  const _LogoutButton({required this.onPressed});

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return SizedBox(
      width: double.infinity,
      child: OutlinedButton.icon(
        onPressed: onPressed,
        icon: Icon(Icons.logout, color: theme.colorScheme.error),
        label: Text(
          l10n.signOut,
          style: TextStyle(color: theme.colorScheme.error),
        ),
        style: OutlinedButton.styleFrom(
          side: BorderSide(
            color: theme.colorScheme.error.withValues(alpha: 0.3),
          ),
          minimumSize: const Size(double.infinity, 48),
        ),
      ),
    );
  }
}
