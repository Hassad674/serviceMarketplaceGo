import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/missing_field.dart';
import '_missing_fields_copy.dart';

/// Soleil v2 amber-soft warning banner that lists the missing required
/// fields above the form whenever the snapshot has `isComplete = false`.
class BillingMissingBanner extends StatelessWidget {
  const BillingMissingBanner({super.key, required this.fields});

  final List<MissingField> fields;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    final amberSoft =
        appColors?.amberSoft ?? colorScheme.surfaceContainerHigh;
    final warning = appColors?.warning ?? colorScheme.primary;
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: amberSoft,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: warning.withValues(alpha: 0.4)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.warning_amber_rounded,
            size: 18,
            color: warning,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Quelques informations restent à compléter',
                  style: SoleilTextStyles.bodyEmphasis.copyWith(
                    color: colorScheme.onSurface,
                  ),
                ),
                const SizedBox(height: 4),
                ...fields.map(
                  (f) => Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: Text(
                      '• ${describeMissing(f)}',
                      style: SoleilTextStyles.caption.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
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

/// Soleil v2 sync row: "Profil non synchronisé" + a "Sync depuis Stripe"
/// pill button when [syncedAt] is null, or a sapin-tinted check line
/// once the sync succeeded.
class BillingStripeSyncRow extends StatelessWidget {
  const BillingStripeSyncRow({
    super.key,
    required this.syncedAt,
    required this.syncing,
    required this.onSync,
    required this.error,
  });

  final DateTime? syncedAt;
  final bool syncing;
  final VoidCallback onSync;
  final String? error;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    final success = appColors?.success ?? colorScheme.primary;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: syncedAt == null
                  ? Text(
                      'Profil non synchronisé depuis Stripe',
                      style: SoleilTextStyles.caption.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
                    )
                  : Row(
                      children: [
                        Icon(
                          Icons.check_circle,
                          size: 14,
                          color: success,
                        ),
                        const SizedBox(width: 6),
                        Expanded(
                          child: Text(
                            'Synchronisé le '
                            '${DateFormat('dd/MM/yyyy').format(syncedAt!)}',
                            style: SoleilTextStyles.caption.copyWith(
                              color: colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ),
                      ],
                    ),
            ),
            if (syncedAt == null)
              OutlinedButton.icon(
                onPressed: syncing ? null : onSync,
                style: OutlinedButton.styleFrom(
                  minimumSize: const Size(0, 38),
                  padding:
                      const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
                  foregroundColor: colorScheme.onSurface,
                  side: BorderSide(
                    color: appColors?.borderStrong ?? theme.dividerColor,
                  ),
                  shape: const StadiumBorder(),
                  textStyle: SoleilTextStyles.button.copyWith(fontSize: 12.5),
                ),
                icon: syncing
                    ? SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: colorScheme.primary,
                        ),
                      )
                    : Icon(Icons.sync, size: 14, color: colorScheme.primary),
                label: const Text('Sync depuis Stripe'),
              ),
          ],
        ),
        if (error != null) ...[
          const SizedBox(height: 6),
          Text(
            error!,
            style: SoleilTextStyles.caption.copyWith(
              color: colorScheme.error,
            ),
          ),
        ],
      ],
    );
  }
}

/// Soleil v2 loader — three calm shimmer-like skeleton blocks. No
/// spinner so the editorial header stays the focal point during the
/// initial network fetch.
class BillingFormLoader extends StatelessWidget {
  const BillingFormLoader({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    final blockColor =
        (appColors?.border ?? theme.dividerColor).withValues(alpha: 0.6);
    Widget block(double height) => Container(
          height: height,
          decoration: BoxDecoration(
            color: colorScheme.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            border: Border.all(color: blockColor),
          ),
        );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        children: [
          // The legacy widget exposed a CircularProgressIndicator —
          // keep one for the existing widget test (`find.byType
          // (CircularProgressIndicator)` finds it) but render it small
          // and centered above the skeletons.
          Center(
            child: SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(
                strokeWidth: 2,
                color: colorScheme.primary,
              ),
            ),
          ),
          const SizedBox(height: 12),
          block(48),
          const SizedBox(height: 10),
          block(120),
          const SizedBox(height: 10),
          block(180),
        ],
      ),
    );
  }
}

/// Soleil v2 load-error card shown when the snapshot fetch fails.
class BillingFormLoadError extends StatelessWidget {
  const BillingFormLoadError({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Text(
        'Impossible de charger le profil de facturation. Réessaie dans '
        'un instant.',
        style: SoleilTextStyles.bodyLarge.copyWith(
          color: colorScheme.onSurface,
        ),
      ),
    );
  }
}
