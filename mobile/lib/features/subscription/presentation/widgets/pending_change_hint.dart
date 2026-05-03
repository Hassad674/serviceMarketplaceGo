import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/subscription.dart';
import '../../../../core/theme/app_palette.dart';

/// Amber banner shown inside the plan summary when the subscription has
/// a pending billing-cycle change scheduled. Mirrors the web variant.
///
/// Returns [SizedBox.shrink] when nothing is pending so the parent can
/// render it unconditionally.
class PendingChangeHint extends StatelessWidget {
  const PendingChangeHint({super.key, required this.subscription});

  final Subscription subscription;

  @override
  Widget build(BuildContext context) {
    final pending = subscription.pendingBillingCycle;
    final effectiveAt = subscription.pendingCycleEffectiveAt;
    if (pending == null || effectiveAt == null) {
      return const SizedBox.shrink();
    }
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    final nextLabel = pending == BillingCycle.annual ? 'annuel' : 'mensuel';
    final currentLabel =
        subscription.billingCycle == BillingCycle.annual ? 'annuel' : 'mensuel';
    final formattedDate = DateFormat('dd/MM/yyyy').format(effectiveAt);
    final amberBorder = (appColors?.warning ?? AppPalette.amber500)
        .withValues(alpha: 0.4);
    final amberBg = (appColors?.warning ?? AppPalette.amber500)
        .withValues(alpha: 0.1);
    const amberFg = AppPalette.amber800; // amber-800 for legible copy

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: amberBg,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: Border.all(color: amberBorder),
      ),
      child: RichText(
        text: TextSpan(
          style: const TextStyle(fontSize: 12, color: amberFg),
          children: [
            TextSpan(text: 'Passage en $nextLabel prévu le '),
            TextSpan(
              text: formattedDate,
              style: const TextStyle(fontWeight: FontWeight.w700),
            ),
            TextSpan(text: ". Tu gardes ton accès $currentLabel jusqu'à cette date."),
          ],
        ),
      ),
    );
  }
}
