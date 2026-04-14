import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/upload_service.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
import '../providers/organization_shared_providers.dart';

/// Thin wrapper around the existing [UploadService] photo upload
/// flow. On success, invalidates the shared profile provider so the
/// new URL flows through every reader in the tree. Rendered only on
/// the freelance profile edit screen — the referrer edit screen
/// relies on the same org row being touched from `/profile`.
class SharedPhotoUploadWidget extends ConsumerWidget {
  const SharedPhotoUploadWidget({
    super.key,
    required this.child,
    required this.canEdit,
  });

  /// The visual target (typically a [ProfileIdentityHeader]). Tapping
  /// it opens the photo upload bottom sheet when [canEdit] is true.
  final Widget child;

  final bool canEdit;

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
        ref.invalidate(organizationSharedProvider);
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.photoUpdated)),
          );
        }
      },
    );
  }
}
