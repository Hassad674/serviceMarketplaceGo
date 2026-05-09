import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/upload_service.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
import '../../../profile_completion/presentation/providers/profile_completion_providers.dart';
import '../providers/organization_shared_providers.dart';

/// Thin wrapper around the existing [UploadService] photo upload
/// flow. On success, invalidates the shared profile provider so the
/// new URL flows through every reader in the tree. Rendered only on
/// the freelance profile edit screen — the referrer edit screen
/// relies on the same org row being touched from `/profile`.
///
/// The optional [onUploaded] callback fires after the shared org
/// provider has been invalidated. Host screens use it to refresh
/// their own persona-specific caches (freelance profile, referrer
/// profile) without forcing this widget to import from those
/// features and break the feature-isolation rule.
class SharedPhotoUploadWidget extends ConsumerWidget {
  const SharedPhotoUploadWidget({
    super.key,
    required this.child,
    required this.canEdit,
    this.onUploaded,
  });

  /// The visual target (typically a [ProfileIdentityHeader]). Tapping
  /// it opens the photo upload bottom sheet when [canEdit] is true.
  final Widget child;

  final bool canEdit;

  /// Called after a successful upload + invalidate of the shared
  /// provider. The host screen typically re-invalidates its own
  /// persona-specific provider here so the JOIN view picks up the
  /// new URL too. Optional — surfaces that only read from the shared
  /// row do not need to wire this.
  final VoidCallback? onUploaded;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (!canEdit) return child;
    return GestureDetector(
      onTap: () => _handleTap(context, ref),
      behavior: HitTestBehavior.opaque,
      child: child,
    );
  }

  void _handleTap(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    showUploadBottomSheet(
      context: context,
      type: UploadMediaType.photo,
      onUpload: (File file) async {
        final uploadService = ref.read(uploadServiceProvider);
        final url = await uploadService.uploadPhoto(file);
        // Persist on the org row via the shared endpoint so both
        // personas pick it up from the organization JOIN.
        await ref.read(sharedPhotoEditorProvider.notifier).save(url);
        // Refresh the shared-row reader.
        ref.invalidate(organizationSharedProvider);
        // Photo is the first section of every persona checklist —
        // refresh the bar so the % climbs without a screen reload.
        ref.invalidate(profileCompletionProvider);
        // Host screens fan out further (freelance / referrer JOIN
        // views) via [onUploaded] so this widget stays free of
        // cross-feature imports for those persona modules.
        onUploaded?.call();
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.photoUpdated)),
          );
        }
      },
    );
  }
}
