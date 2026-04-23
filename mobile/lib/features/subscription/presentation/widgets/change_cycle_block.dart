import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/subscription.dart';
import '../providers/subscription_providers.dart';
import 'cycle_preview_card.dart';

/// Interactive cycle-change UI. Mirrors the web `ChangeCycleBlock`:
///
///   * If the subscription has a pending cycle change → button disabled
///     with label "Changement déjà programmé".
///   * Otherwise the button label adapts to the current cycle:
///     "Passer à l'annuel (-21%)" (monthly → annual) or
///     "Repasser en mensuel" (annual → monthly).
///   * Tapping the button unfolds the confirm section: a [CyclePreviewCard]
///     + "Annuler" / "Confirmer" buttons.
///   * Confirm calls [ChangeCycleUseCase], invalidates the subscription +
///     preview providers, and collapses the confirm state on success.
///   * On error, inline red text shows under the confirm buttons — the
///     user can retry without re-expanding the block.
class ChangeCycleBlock extends ConsumerStatefulWidget {
  const ChangeCycleBlock({super.key, required this.subscription});

  final Subscription subscription;

  @override
  ConsumerState<ChangeCycleBlock> createState() => _ChangeCycleBlockState();
}

class _ChangeCycleBlockState extends ConsumerState<ChangeCycleBlock> {
  BillingCycle? _target;
  bool _pending = false;
  String? _error;

  BillingCycle get _nextTarget =>
      widget.subscription.billingCycle == BillingCycle.monthly
          ? BillingCycle.annual
          : BillingCycle.monthly;

  bool get _hasPending => widget.subscription.pendingBillingCycle != null;

  void _onStart() {
    setState(() {
      _target = _nextTarget;
      _error = null;
    });
  }

  void _onCancel() {
    setState(() {
      _target = null;
      _error = null;
    });
  }

  Future<void> _onConfirm() async {
    final target = _target;
    if (target == null) return;
    setState(() {
      _pending = true;
      _error = null;
    });
    try {
      final useCase = ref.read(changeCycleUseCaseProvider);
      await useCase(billingCycle: target);
      // Wipe the caches so the updated subscription flows through.
      ref.invalidate(subscriptionProvider);
      ref.invalidate(cyclePreviewProvider(target));
      if (!mounted) return;
      setState(() {
        _target = null;
        _pending = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _pending = false;
        _error = "Impossible d'appliquer le changement. Réessaie.";
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final currentLabel =
        widget.subscription.billingCycle == BillingCycle.annual
            ? 'annuel'
            : 'mensuel';

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: appColors?.border ?? theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          RichText(
            text: TextSpan(
              style: theme.textTheme.bodySmall,
              children: [
                const TextSpan(text: 'Cycle actuel : '),
                TextSpan(
                  text: currentLabel,
                  style: TextStyle(
                    fontWeight: FontWeight.w600,
                    color: theme.colorScheme.onSurface,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          if (_target == null) _buildTrigger() else _buildConfirm(),
        ],
      ),
    );
  }

  Widget _buildTrigger() {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final label = _hasPending
        ? 'Changement déjà programmé'
        : (_nextTarget == BillingCycle.annual
            ? "Passer à l'annuel (-21%)"
            : 'Repasser en mensuel');

    return SizedBox(
      width: double.infinity,
      child: OutlinedButton(
        onPressed: _hasPending ? null : _onStart,
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(double.infinity, 40),
          foregroundColor: primary,
          backgroundColor: primary.withValues(alpha: 0.08),
          side: BorderSide(color: primary.withValues(alpha: 0.6)),
          textStyle: const TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w700,
          ),
        ),
        child: Text(label),
      ),
    );
  }

  Widget _buildConfirm() {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        CyclePreviewCard(
          target: _target!,
          currentPeriodEnd: widget.subscription.currentPeriodEnd,
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: OutlinedButton(
                onPressed: _pending ? null : _onCancel,
                style: OutlinedButton.styleFrom(
                  minimumSize: const Size(double.infinity, 40),
                  textStyle: const TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                child: const Text('Annuler'),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: ElevatedButton(
                onPressed: _pending ? null : _onConfirm,
                style: ElevatedButton.styleFrom(
                  minimumSize: const Size(double.infinity, 40),
                  textStyle: const TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                child: _pending
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Text('Confirmer'),
              ),
            ),
          ],
        ),
        if (_error != null) ...[
          const SizedBox(height: 8),
          Text(
            _error!,
            style: TextStyle(
              fontSize: 12,
              color: theme.colorScheme.error,
            ),
          ),
        ],
      ],
    );
  }
}
