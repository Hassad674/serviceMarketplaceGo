import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../expertise/presentation/widgets/expertise_section_widget.dart';
import '../../domain/entities/referrer_profile.dart';
import '../providers/referrer_profile_providers.dart';
import '../widgets/referrer_pricing_section_widget.dart';
import '../widgets/referrer_profile_header.dart';

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
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (_, __) => _ErrorState(
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
          _VideoCard(
            videoUrl: profile.videoUrl,
            canEdit: canEdit,
            onUpload: () => _openVideoUpload(context, ref),
            onDelete: () => _confirmDeleteVideo(context, ref),
          ),
          const SizedBox(height: 16),

          // Empty project history placeholder — the referral_deals
          // feature does not ship yet. Keep this block visible so
          // the screen layout stays stable once the backend arrives.
          _ReferrerHistoryPlaceholder(
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
                      '/api/v1/referrer-profile',
                      data: {'about': controller.text},
                    );
                    ref.invalidate(referrerProfileProvider);
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

  Future<void> _openVideoUpload(BuildContext context, WidgetRef ref) async {
    final l10n = AppLocalizations.of(context)!;
    await showUploadBottomSheet(
      context: context,
      type: UploadMediaType.video,
      maxSizeBytes: 50 * 1024 * 1024,
      onUpload: (file) async {
        final ok =
            await ref.read(referrerVideoEditorProvider.notifier).upload(file);
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

  Future<void> _openEditCoreSheet(
    BuildContext context,
    WidgetRef ref,
    ReferrerProfile profile,
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
                        .read(referrerCoreEditorProvider.notifier)
                        .save(
                          title: controller.text,
                          about: profile.about,
                          videoUrl: profile.videoUrl,
                        );
                    if (ok) ref.invalidate(referrerProfileProvider);
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
// Helpers
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
    required this.onUpload,
    required this.onDelete,
  });

  final String videoUrl;
  final bool canEdit;
  final VoidCallback onUpload;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasVideo = videoUrl.isNotEmpty;
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
          _ReferrerVideoHeader(
            hasVideo: hasVideo,
            canEdit: canEdit,
            onReplace: onUpload,
            onDelete: onDelete,
          ),
          const SizedBox(height: 16),
          if (hasVideo)
            VideoPlayerWidget(videoUrl: videoUrl)
          else
            _ReferrerVideoEmptyState(canEdit: canEdit, onAdd: onUpload),
        ],
      ),
    );
  }
}

class _ReferrerVideoHeader extends StatelessWidget {
  const _ReferrerVideoHeader({
    required this.hasVideo,
    required this.canEdit,
    required this.onReplace,
    required this.onDelete,
  });

  final bool hasVideo;
  final bool canEdit;
  final VoidCallback onReplace;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Row(
      children: [
        Icon(
          Icons.videocam_outlined,
          size: 20,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            l10n.presentationVideo,
            style: theme.textTheme.titleMedium,
          ),
        ),
        if (hasVideo && canEdit) ...[
          TextButton.icon(
            onPressed: onReplace,
            icon: const Icon(Icons.cached, size: 18),
            label: Text(l10n.replaceVideo),
          ),
          IconButton(
            onPressed: onDelete,
            icon: Icon(Icons.delete_outline, color: theme.colorScheme.error),
            tooltip: l10n.removeVideo,
          ),
        ],
      ],
    );
  }
}

class _ReferrerVideoEmptyState extends StatelessWidget {
  const _ReferrerVideoEmptyState({
    required this.canEdit,
    required this.onAdd,
  });

  final bool canEdit;
  final VoidCallback onAdd;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 16),
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
          if (canEdit) ...[
            const SizedBox(height: 16),
            ElevatedButton.icon(
              onPressed: onAdd,
              icon: const Icon(Icons.add, size: 18),
              label: Text(l10n.addVideo),
            ),
          ],
        ],
      ),
    );
  }
}

class _ReferrerHistoryPlaceholder extends StatelessWidget {
  const _ReferrerHistoryPlaceholder({
    required this.title,
    required this.emptyLabel,
  });

  final String title;
  final String emptyLabel;

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
              Icon(
                Icons.history_edu_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Text(title, style: theme.textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 16),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: Row(
              children: [
                Icon(
                  Icons.info_outline,
                  size: 18,
                  color: theme.colorScheme.onSurfaceVariant,
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    emptyLabel,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                      height: 1.4,
                    ),
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
