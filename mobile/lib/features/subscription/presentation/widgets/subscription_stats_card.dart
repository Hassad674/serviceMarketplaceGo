import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../providers/subscription_providers.dart';

/// Rose gradient card showing "Tu as économisé X € depuis le JJ/MM/AAAA".
///
/// Consumes [subscriptionStatsProvider]. Collapses to nothing on the
/// free tier (provider returns null) or while the first fetch is still
/// in flight — the manage sheet already shows the plan summary, so an
/// extra skeleton here would be noise.
class SubscriptionStatsCard extends ConsumerWidget {
  const SubscriptionStatsCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(subscriptionStatsProvider);
    return async.when(
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
      data: (stats) {
        if (stats == null) return const SizedBox.shrink();

        final theme = Theme.of(context);
        final primary = theme.colorScheme.primary;
        final savedEuros = _formatEuros(stats.savedFeeCents);
        final sinceLabel = DateFormat('dd/MM/yyyy').format(stats.since);

        return Container(
          width: double.infinity,
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [
                primary.withValues(alpha: 0.08),
                theme.colorScheme.surface,
              ],
            ),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            border: Border.all(color: primary.withValues(alpha: 0.3)),
          ),
          child: RichText(
            text: TextSpan(
              style: theme.textTheme.bodyMedium,
              children: [
                const TextSpan(text: 'Tu as économisé '),
                TextSpan(
                  text: savedEuros,
                  style: TextStyle(
                    fontWeight: FontWeight.w700,
                    color: primary,
                    fontFamily: 'monospace',
                  ),
                ),
                const TextSpan(text: ' depuis le '),
                TextSpan(
                  text: sinceLabel,
                  style: const TextStyle(fontWeight: FontWeight.w600),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

String _formatEuros(int cents) {
  final euros = cents / 100.0;
  try {
    return NumberFormat.currency(
      locale: 'fr_FR',
      symbol: '€',
      decimalDigits: 2,
    ).format(euros);
  } catch (_) {
    return '${euros.toStringAsFixed(2).replaceAll('.', ',')} €';
  }
}
