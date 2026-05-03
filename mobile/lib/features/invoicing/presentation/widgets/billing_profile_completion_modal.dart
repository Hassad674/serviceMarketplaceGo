import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/missing_field.dart';
import '_missing_fields_copy.dart';
import '../../../../core/theme/app_palette.dart';

/// Opens the gate modal explaining that the billing profile is
/// incomplete and routing to `/settings/billing-profile`.
///
/// Mirrors the web `BillingProfileCompletionModal`: same title, same
/// FR copies, same primary CTA. Implemented as a bottom-sheet here
/// because the rest of the mobile app uses bottom-sheets for
/// gate/manage flows (see `manage_bottom_sheet.dart`).
///
/// [missingFields] — list returned either by the snapshot provider
/// (proactive gate) or by a 403 envelope (defensive gate). Either is
/// valid; the modal only renders the FR labels.
///
/// [message] — optional override paragraph shown above the bullet
/// list. Lets the caller tailor copy ("Complète ton profil pour
/// retirer", etc.) without forking the widget.
Future<void> showBillingProfileCompletionModal(
  BuildContext context, {
  required List<MissingField> missingFields,
  String? message,
  String? returnTo,
}) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(
        top: Radius.circular(AppTheme.radiusLg),
      ),
    ),
    builder: (_) => _BillingProfileCompletionSheet(
      missingFields: missingFields,
      message: message,
      returnTo: returnTo,
    ),
  );
}

class _BillingProfileCompletionSheet extends StatelessWidget {
  const _BillingProfileCompletionSheet({
    required this.missingFields,
    this.message,
    this.returnTo,
  });

  final List<MissingField> missingFields;
  final String? message;
  final String? returnTo;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Center(
            child: Container(
              width: 40,
              height: 4,
              decoration: BoxDecoration(
                color: appColors?.mutedForeground.withValues(alpha: 0.3) ??
                    theme.dividerColor,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ),
          const SizedBox(height: 16),
          Text(
            'Complète ton profil de facturation pour continuer',
            style: theme.textTheme.titleLarge,
          ),
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: AppPalette.orange50, // amber-50
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
              border: Border.all(
                color: AppPalette.amber300, // amber-300
                width: 1,
              ),
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Icon(
                  Icons.warning_amber_rounded,
                  size: 18,
                  color: AppPalette.amber800, // amber-800
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    message ??
                        'Avant de pouvoir effectuer cette opération, '
                            'complète les informations suivantes. Elles '
                            'apparaîtront sur tes factures.',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: AppPalette.amber800,
                      fontSize: 13,
                    ),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          if (missingFields.isEmpty)
            Text(
              'Quelques informations restent à compléter sur ton profil '
              'de facturation.',
              style: theme.textTheme.bodyMedium,
            )
          else
            _MissingList(fields: missingFields),
          const SizedBox(height: 20),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text('Plus tard'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton.icon(
                  onPressed: () {
                    Navigator.of(context).pop();
                    // Defer push so the modal pop animation doesn't
                    // conflict with the route transition.
                    WidgetsBinding.instance.addPostFrameCallback((_) {
                      if (!context.mounted) return;
                      final target = returnTo == null
                          ? RoutePaths.billingProfile
                          : '${RoutePaths.billingProfile}?return_to=${Uri.encodeComponent(returnTo!)}';
                      context.push(target);
                    });
                  },
                  icon: const Icon(Icons.arrow_forward, size: 16),
                  label: const Text('Compléter mon profil'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: AppPalette.rose500,
                    foregroundColor: Colors.white,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _MissingList extends StatelessWidget {
  const _MissingList({required this.fields});

  final List<MissingField> fields;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: fields.map((field) {
        final label = describeMissing(field);
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                margin: const EdgeInsets.only(top: 7, right: 10),
                width: 6,
                height: 6,
                decoration: const BoxDecoration(
                  color: AppPalette.rose500,
                  shape: BoxShape.circle,
                ),
              ),
              Expanded(
                child: Text(
                  label,
                  style: theme.textTheme.bodyMedium,
                ),
              ),
            ],
          ),
        );
      }).toList(),
    );
  }
}
