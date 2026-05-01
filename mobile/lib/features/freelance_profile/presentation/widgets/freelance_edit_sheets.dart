import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/freelance_profile.dart';
import '../providers/freelance_profile_providers.dart';

/// Opens a bottom sheet allowing the user to edit their About text.
Future<void> showFreelanceAboutSheet({
  required BuildContext context,
  required WidgetRef ref,
  required String currentAbout,
}) async {
  final l10n = AppLocalizations.of(context)!;
  final controller = TextEditingController(text: currentAbout);
  await showModalBottomSheet<void>(
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

/// Opens a bottom sheet allowing the user to edit their professional
/// title (the headline shown above the about text).
Future<void> showFreelanceTitleSheet({
  required BuildContext context,
  required WidgetRef ref,
  required FreelanceProfile profile,
}) async {
  final l10n = AppLocalizations.of(context)!;
  final controller = TextEditingController(text: profile.title);
  await showModalBottomSheet<void>(
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
