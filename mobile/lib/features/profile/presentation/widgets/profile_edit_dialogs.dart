import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/network/upload_service.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
import '../providers/profile_provider.dart';

/// Helpers that show the legacy profile screen's edit dialogs.
/// They live here so the screen orchestrator stays focused on layout.

void openProfilePhotoUpload(BuildContext context, WidgetRef ref) {
  final l10n = AppLocalizations.of(context)!;
  showUploadBottomSheet(
    context: context,
    type: UploadMediaType.photo,
    onUpload: (File file) async {
      final uploadService = ref.read(uploadServiceProvider);
      await uploadService.uploadPhoto(file);
      ref.invalidate(profileProvider);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.photoUpdated)),
        );
      }
    },
  );
}

void openProfileVideoUpload(BuildContext context, WidgetRef ref) {
  final l10n = AppLocalizations.of(context)!;
  showUploadBottomSheet(
    context: context,
    type: UploadMediaType.video,
    maxSizeBytes: 50 * 1024 * 1024, // 50 MB for videos
    onUpload: (File file) async {
      final uploadService = ref.read(uploadServiceProvider);
      await uploadService.uploadVideo(file);
      ref.invalidate(profileProvider);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.videoUpdated)),
        );
      }
    },
  );
}

void openProfileAboutEditor(
  BuildContext context,
  WidgetRef ref,
  String? currentAbout,
) {
  final l10n = AppLocalizations.of(context)!;
  final controller = TextEditingController(text: currentAbout ?? '');
  showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
    ),
    builder: (ctx) => Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.of(ctx).viewInsets.bottom),
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
                    '/api/v1/profile',
                    data: {'about': controller.text},
                  );
                  ref.invalidate(profileProvider);
                  if (ctx.mounted) Navigator.pop(ctx);
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text(l10n.aboutUpdated)),
                    );
                  }
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

void confirmDeleteProfileVideo(BuildContext context, WidgetRef ref) {
  final l10n = AppLocalizations.of(context)!;
  showDialog<void>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: Text(l10n.removeVideoConfirmTitle),
      content: Text(l10n.removeVideoConfirmMessage),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(ctx),
          child: Text(l10n.cancel),
        ),
        TextButton(
          onPressed: () async {
            Navigator.pop(ctx);
            final api = ref.read(apiClientProvider);
            await api.delete('/api/v1/upload/video');
            ref.invalidate(profileProvider);
            if (context.mounted) {
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text(l10n.videoRemoved)),
              );
            }
          },
          child: Text(
            l10n.remove,
            style: TextStyle(color: Theme.of(ctx).colorScheme.error),
          ),
        ),
      ],
    ),
  );
}
