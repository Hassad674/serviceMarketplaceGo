import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
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
import '../widgets/freelance_dark_mode_toggle.dart';
import '../widgets/freelance_edit_sheets.dart';
import '../widgets/freelance_logout_button.dart';
import '../widgets/freelance_pricing_section_widget.dart';
import '../widgets/freelance_profile_header.dart';
import '../widgets/freelance_section_card.dart';
import '../widgets/freelance_social_links_section_widget.dart';
import '../widgets/freelance_states.dart';
import '../widgets/freelance_video_card.dart';

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
          loading: () => const FreelanceLoadingState(),
          error: (_, __) => FreelanceErrorState(
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

          // Social links (freelance persona — independent from the
          // referrer set on /referral).
          FreelanceSocialLinksSectionWidget(canEdit: canEdit),
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
          FreelanceSectionCard(
            title: l10n.professionalTitle,
            icon: Icons.badge_outlined,
            trailing: canEdit
                ? IconButton(
                    icon: const Icon(Icons.edit_outlined, size: 18),
                    onPressed: () => showFreelanceTitleSheet(
                      context: context,
                      ref: ref,
                      profile: profile,
                    ),
                  )
                : null,
            child: Text(
              profile.title.isNotEmpty
                  ? profile.title
                  : l10n.titlePlaceholder,
              style: theme.textTheme.bodyMedium?.copyWith(
                fontStyle: profile.title.isEmpty
                    ? FontStyle.italic
                    : FontStyle.normal,
              ),
            ),
          ),
          const SizedBox(height: 16),

          // About section
          GestureDetector(
            onTap: canEdit
                ? () => showFreelanceAboutSheet(
                      context: context,
                      ref: ref,
                      currentAbout: profile.about,
                    )
                : null,
            child: FreelanceSectionCard(
              title: l10n.about,
              icon: Icons.info_outline,
              child: SizedBox(
                width: double.infinity,
                child: Text(
                  profile.about.isNotEmpty
                      ? profile.about
                      : l10n.aboutPlaceholder,
                  softWrap: true,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    height: 1.5,
                    fontStyle: profile.about.isEmpty
                        ? FontStyle.italic
                        : FontStyle.normal,
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(height: 16),

          // Video section
          FreelanceVideoCard(
            videoUrl: profile.videoUrl,
            canEdit: canEdit,
            onUpload: () => _openVideoUpload(context, ref),
            onDelete: () => _confirmDeleteVideo(context, ref),
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
          const FreelanceDarkModeToggle(),
          const SizedBox(height: 24),
          FreelanceLogoutButton(
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

  Future<void> _openVideoUpload(BuildContext context, WidgetRef ref) async {
    final l10n = AppLocalizations.of(context)!;
    await showUploadBottomSheet(
      context: context,
      type: UploadMediaType.video,
      maxSizeBytes: 50 * 1024 * 1024,
      onUpload: (file) async {
        final ok = await ref
            .read(freelanceVideoEditorProvider.notifier)
            .upload(file);
        if (!ok) throw Exception('upload_failed');
        ref.invalidate(freelanceProfileProvider);
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.videoUpdated)),
          );
        }
      },
    );
  }

  Future<void> _confirmDeleteVideo(
    BuildContext context,
    WidgetRef ref,
  ) async {
    final l10n = AppLocalizations.of(context)!;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.removeVideoConfirmTitle),
        content: Text(l10n.removeVideoConfirmMessage),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(l10n.removeVideo),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    final ok =
        await ref.read(freelanceVideoEditorProvider.notifier).remove();
    if (ok) ref.invalidate(freelanceProfileProvider);
  }
}
