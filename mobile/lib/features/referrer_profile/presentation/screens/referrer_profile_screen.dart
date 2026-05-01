import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../expertise/presentation/widgets/expertise_section_widget.dart';
import '../../../freelance_profile/presentation/widgets/freelance_section_card.dart';
import '../../../freelance_profile/presentation/widgets/freelance_states.dart';
import '../../../freelance_profile/presentation/widgets/freelance_video_card.dart';
import '../../domain/entities/referrer_profile.dart';
import '../providers/referrer_profile_providers.dart';
import '../widgets/referrer_edit_sheets.dart';
import '../widgets/referrer_history_placeholder.dart';
import '../widgets/referrer_pricing_section_widget.dart';
import '../widgets/referrer_profile_header.dart';
import '../widgets/referrer_social_links_section_widget.dart';

/// Editable referrer profile screen mounted on `/referral`. Shows
/// ONLY persona-specific fields — the shared org block (photo,
/// location, languages) is managed from `/profile` so the user
/// edits it once and both personas reflect the change.
///
/// Sections intentionally absent from this surface:
/// - Skills — skill vocabularies describe what a person does, not
///   what deals they bring in.
/// - Portfolio — no body of work is attached to a referrer persona.
///
/// Project history renders an empty placeholder until the referrer
/// deal backend ships.
class ReferrerProfileScreen extends ConsumerWidget {
  const ReferrerProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(referrerProfileProvider);
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.referrerMode),
      ),
      body: SafeArea(
        child: profileAsync.when(
          loading: () => const FreelanceLoadingState(),
          error: (_, __) => FreelanceErrorState(
            onRetry: () => ref.invalidate(referrerProfileProvider),
          ),
          data: (profile) => _ReferrerBody(profile: profile),
        ),
      ),
    );
  }
}

class _ReferrerBody extends ConsumerWidget {
  const _ReferrerBody({required this.profile});

  final ReferrerProfile profile;

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
          ReferrerProfileHeader(
            displayName: displayName,
            title: profile.title,
            photoUrl: profile.photoUrl,
            initials: _buildInitials(displayName),
            availabilityWireValue: profile.availabilityStatus,
          ),
          const SizedBox(height: 16),

          // Expertise is shared conceptually between personas but
          // each profile owns its own selection on the backend.
          ExpertiseSectionWidget(
            orgType: 'provider_personal',
            initialDomains: profile.expertiseDomains,
            canEdit: canEdit,
            onSaved: () => ref.invalidate(referrerProfileProvider),
          ),
          const SizedBox(height: 16),

          // Pricing (commission variants only)
          ReferrerPricingSectionWidget(canEdit: canEdit),
          const SizedBox(height: 16),

          // Social links (referrer persona — independent from the
          // freelance set on /profile).
          ReferrerSocialLinksSectionWidget(canEdit: canEdit),
          const SizedBox(height: 16),

          // Title section
          FreelanceSectionCard(
            title: l10n.professionalTitle,
            icon: Icons.badge_outlined,
            trailing: canEdit
                ? IconButton(
                    icon: const Icon(Icons.edit_outlined, size: 18),
                    onPressed: () => showReferrerTitleSheet(
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
                ? () => showReferrerAboutSheet(
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

          // Video
          FreelanceVideoCard(
            videoUrl: profile.videoUrl,
            canEdit: canEdit,
            onUpload: () => _openVideoUpload(context, ref),
            onDelete: () => _confirmDeleteVideo(context, ref),
          ),
          const SizedBox(height: 16),

          // Empty project history placeholder — the referral_deals
          // feature does not ship yet. Keep this block visible so
          // the screen layout stays stable once the backend arrives.
          ReferrerHistoryPlaceholder(
            title: l10n.projectHistory,
            emptyLabel: l10n.referrerProjectHistoryEmpty,
          ),
          // TODO: wire referral_deals feature when backend ships
          const SizedBox(height: 32),
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
            .read(referrerVideoEditorProvider.notifier)
            .upload(file);
        if (!ok) throw Exception('upload_failed');
        ref.invalidate(referrerProfileProvider);
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
    final ok = await ref.read(referrerVideoEditorProvider.notifier).remove();
    if (ok) ref.invalidate(referrerProfileProvider);
  }
}
