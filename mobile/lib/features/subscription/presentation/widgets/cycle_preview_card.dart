import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/cycle_preview.dart';
import '../../domain/entities/subscription.dart';
import '../providers/subscription_providers.dart';

/// Renders the amount + message for a pending cycle change.
///
/// Mirrors the web `PreviewMessage` inside `change-cycle-block`:
///   * `prorateImmediately == true` → upgrade copy: "Tu seras facturé
///     X,XX € aujourd'hui. Ton cycle passe en {target} immédiatement."
///   * `prorateImmediately == false` → downgrade copy: "Aucun débit
///     aujourd'hui. Tu gardes ton accès jusqu'au JJ/MM/AAAA, puis tu
///     passeras en {target}."
///
/// The downgrade copy intentionally uses [currentPeriodEnd] (from the
/// [Subscription]) instead of `preview.periodEnd` — Stripe's preview
/// for an interval change returns the NEXT monthly invoice window,
/// which would shorten the displayed access date misleadingly.
class CyclePreviewCard extends ConsumerWidget {
  const CyclePreviewCard({
    super.key,
    required this.target,
    required this.currentPeriodEnd,
  });

  /// The billing cycle the user is considering switching to.
  final BillingCycle target;

  /// Access-until date used for the downgrade (no-charge) branch.
  final DateTime currentPeriodEnd;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(cyclePreviewProvider(target));
    return async.when(
      loading: () => _PreviewText(
        child: Text(
          'Calcul du montant…',
          style: Theme.of(context).textTheme.bodySmall,
        ),
      ),
      error: (_, __) => _PreviewText(
        child: Text(
          "Impossible d'afficher le montant. Réessaie plus tard.",
          style: TextStyle(
            fontSize: 13,
            color: Theme.of(context).colorScheme.error,
          ),
        ),
      ),
      data: (preview) => _PreviewText(
        child: _buildMessage(context, preview),
      ),
    );
  }

  Widget _buildMessage(BuildContext context, CyclePreview preview) {
    final theme = Theme.of(context);
    final targetLabel = target == BillingCycle.annual ? 'annuel' : 'mensuel';
    final bold = theme.textTheme.bodySmall?.copyWith(
      fontWeight: FontWeight.w700,
      color: theme.colorScheme.onSurface,
    );

    if (preview.prorateImmediately) {
      // Upgrade — Stripe charges the delta today.
      return RichText(
        text: TextSpan(
          style: theme.textTheme.bodySmall,
          children: [
            const TextSpan(text: 'Tu seras facturé '),
            TextSpan(
              text: _formatEuros(preview.amountDueCents),
              style: bold,
            ),
            TextSpan(
              text: " aujourd'hui. Ton cycle passe en $targetLabel immédiatement.",
            ),
          ],
        ),
      );
    }
    // Downgrade — scheduled, no charge today. Keep the user's access
    // until the end of the CURRENT period (see doc comment above for
    // why we ignore preview.periodEnd here).
    return RichText(
      text: TextSpan(
        style: theme.textTheme.bodySmall,
        children: [
          const TextSpan(text: "Aucun débit aujourd'hui. Tu gardes ton accès jusqu'au "),
          TextSpan(
            text: _formatDate(currentPeriodEnd),
            style: bold,
          ),
          TextSpan(text: ', puis tu passeras en $targetLabel.'),
        ],
      ),
    );
  }
}

String _formatDate(DateTime date) {
  return DateFormat('dd/MM/yyyy').format(date);
}

String _formatEuros(int cents) {
  final euros = cents / 100.0;
  final formatter = NumberFormat.currency(
    locale: 'fr_FR',
    symbol: '€',
    decimalDigits: 2,
  );
  try {
    return formatter.format(euros);
  } catch (_) {
    // Fallback when fr_FR locale data isn't loaded.
    return '${euros.toStringAsFixed(2).replaceAll('.', ',')} €';
  }
}

class _PreviewText extends StatelessWidget {
  const _PreviewText({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: appColors?.muted ?? theme.dividerColor,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
      ),
      child: child,
    );
  }
}
