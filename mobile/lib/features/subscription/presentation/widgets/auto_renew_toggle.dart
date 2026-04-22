import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/subscription.dart';
import '../providers/subscription_providers.dart';

/// Row with a label + description on the left and a Material [Switch] on
/// the right. Mirrors the web `AutoRenewSwitch`.
///
/// Mutation flow:
///   * While the [ToggleAutoRenewUseCase] call is in flight the switch
///     is disabled and shows a small progress indicator.
///   * On success, `subscriptionProvider` is invalidated so the parent
///     picks up the refreshed state.
///   * On error, a red [SnackBar] is shown; the switch snaps back to
///     its server-side value on rebuild.
class AutoRenewToggle extends ConsumerStatefulWidget {
  const AutoRenewToggle({super.key, required this.subscription});

  final Subscription subscription;

  @override
  ConsumerState<AutoRenewToggle> createState() => _AutoRenewToggleState();
}

class _AutoRenewToggleState extends ConsumerState<AutoRenewToggle> {
  bool _pending = false;

  Future<void> _onChanged(bool next) async {
    if (_pending) return;
    setState(() => _pending = true);
    try {
      final useCase = ref.read(toggleAutoRenewUseCaseProvider);
      // `cancel_at_period_end == true` means auto-renew is OFF. We want
      // the opposite contract on the use case: `autoRenew == true` means
      // keep auto-renew on.
      await useCase(autoRenew: next);
      // Force the provider to re-fetch so the badge + every consumer
      // reflects the fresh server state.
      ref.invalidate(subscriptionProvider);
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: const Text(
            'Impossible de mettre à jour le renouvellement. Réessaie.',
          ),
          backgroundColor: Theme.of(context).colorScheme.error,
          behavior: SnackBarBehavior.floating,
        ),
      );
    } finally {
      if (mounted) setState(() => _pending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final checked = !widget.subscription.cancelAtPeriodEnd;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: appColors?.border ?? theme.dividerColor),
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Renouvellement automatique',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontSize: 14,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  checked
                      ? 'Tu seras facturé automatiquement à chaque échéance'
                      : 'Premium expirera à la fin de la période actuelle',
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          if (_pending)
            const SizedBox(
              width: 20,
              height: 20,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          else
            Switch(
              value: checked,
              onChanged: _pending ? null : _onChanged,
            ),
        ],
      ),
    );
  }
}
