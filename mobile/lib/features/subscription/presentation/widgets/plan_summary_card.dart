import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/subscription.dart';
import 'pending_change_hint.dart';

/// Top block of the manage bottom-sheet. Shows plan label + cycle +
/// price on the left, next-renewal date on the right. Embeds
/// [PendingChangeHint] inline when a cycle change is scheduled.
class PlanSummaryCard extends StatelessWidget {
  const PlanSummaryCard({super.key, required this.subscription});

  final Subscription subscription;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final planLabel = subscription.plan == Plan.agency
        ? 'Premium Agence'
        : 'Premium Freelance';
    final cycleLabel =
        subscription.billingCycle == BillingCycle.annual ? 'Annuel' : 'Mensuel';
    final priceEuros = _priceOf(subscription.plan, subscription.billingCycle);
    final nextRenewal =
        DateFormat('dd/MM/yyyy').format(subscription.currentPeriodEnd);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: appColors?.muted ?? theme.dividerColor,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: appColors?.border ?? theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      planLabel,
                      style: theme.textTheme.titleMedium,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '$cycleLabel · $priceEuros €',
                      style: theme.textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(
                    'Prochain renouvellement',
                    style: theme.textTheme.bodySmall,
                    textAlign: TextAlign.right,
                  ),
                  const SizedBox(height: 2),
                  Text(
                    nextRenewal,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ],
              ),
            ],
          ),
          if (subscription.pendingBillingCycle != null &&
              subscription.pendingCycleEffectiveAt != null) ...[
            const SizedBox(height: 12),
            PendingChangeHint(subscription: subscription),
          ],
        ],
      ),
    );
  }
}

int _priceOf(Plan plan, BillingCycle cycle) {
  if (plan == Plan.agency) {
    return cycle == BillingCycle.annual ? 468 : 49;
  }
  return cycle == BillingCycle.annual ? 180 : 19;
}
