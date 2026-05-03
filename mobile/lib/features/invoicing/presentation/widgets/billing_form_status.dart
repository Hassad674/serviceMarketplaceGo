import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/missing_field.dart';
import '_missing_fields_copy.dart';
import '../../../../core/theme/app_palette.dart';

/// Amber warning banner that lists the missing required fields above
/// the form whenever the snapshot has `isComplete = false`.
class BillingMissingBanner extends StatelessWidget {
  const BillingMissingBanner({super.key, required this.fields});

  final List<MissingField> fields;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppPalette.orange50,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: AppPalette.amber300),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(
            Icons.warning_amber_rounded,
            size: 18,
            color: AppPalette.amber800,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Quelques informations restent à compléter',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                    color: AppPalette.amber800,
                  ),
                ),
                const SizedBox(height: 4),
                ...fields.map(
                  (f) => Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: Text(
                      '• ${describeMissing(f)}',
                      style: TextStyle(
                        fontSize: 12,
                        color:
                            AppPalette.amber800.withValues(alpha: 0.9),
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

/// Top-of-form status row: "Profil non synchronisé" + a "Sync depuis
/// Stripe" button when [syncedAt] is null, or a green confirmation line
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
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: syncedAt == null
                  ? Text(
                      'Profil non synchronisé depuis Stripe',
                      style: theme.textTheme.bodySmall,
                    )
                  : Row(
                      children: [
                        const Icon(
                          Icons.check_circle,
                          size: 14,
                          color: AppPalette.green500,
                        ),
                        const SizedBox(width: 6),
                        Expanded(
                          child: Text(
                            'Synchronisé le '
                            '${DateFormat('dd/MM/yyyy').format(syncedAt!)}',
                            style: theme.textTheme.bodySmall,
                          ),
                        ),
                      ],
                    ),
            ),
            if (syncedAt == null)
              OutlinedButton.icon(
                onPressed: syncing ? null : onSync,
                icon: syncing
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.sync, size: 16),
                label: const Text('Sync depuis Stripe'),
              ),
          ],
        ),
        if (error != null) ...[
          const SizedBox(height: 6),
          Text(
            error!,
            style: TextStyle(color: theme.colorScheme.error, fontSize: 12),
          ),
        ],
      ],
    );
  }
}

/// Centered indeterminate spinner shown while the form snapshot is
/// loading.
class BillingFormLoader extends StatelessWidget {
  const BillingFormLoader({super.key});

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 48),
      child: Center(child: CircularProgressIndicator()),
    );
  }
}

/// Generic load-error card shown when the snapshot fetch fails.
class BillingFormLoadError extends StatelessWidget {
  const BillingFormLoadError({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Text(
        'Impossible de charger le profil de facturation. Réessaie dans '
        'un instant.',
        style: theme.textTheme.bodyMedium,
      ),
    );
  }
}
