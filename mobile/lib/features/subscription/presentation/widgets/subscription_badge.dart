import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/subscription.dart';
import '../providers/subscription_providers.dart';

/// Compact, clickable Premium pill surfaced in the drawer / header /
/// profile strip. Mirrors the web `SubscriptionBadge` — the badge owns
/// no state beyond the cached subscription. The parent decides which
/// sheet (upgrade / manage) opens on tap, keeping this composable.
class SubscriptionBadge extends ConsumerWidget {
  const SubscriptionBadge({super.key, required this.onTap});

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(subscriptionProvider);
    return async.when(
      loading: () => const _BadgeSkeleton(),
      error: (_, __) => const SizedBox.shrink(),
      data: (sub) {
        final variant = _pickVariant(context, sub);
        return _BadgePill(variant: variant, onTap: onTap);
      },
    );
  }
}

/// Visual descriptor of a single badge state.
class _BadgeVariant {
  const _BadgeVariant({
    required this.label,
    required this.semantics,
    required this.background,
    required this.foreground,
    required this.border,
    required this.showIcon,
  });

  final String label;
  final String semantics;
  final Color background;
  final Color foreground;
  final Color? border;
  final bool showIcon;
}

_BadgeVariant _pickVariant(BuildContext context, Subscription? sub) {
  final theme = Theme.of(context);
  final primary = theme.colorScheme.primary;

  if (sub == null) {
    // Free tier — rose filled CTA with sparkle.
    return _BadgeVariant(
      label: 'Passer Premium',
      semantics: 'Passer Premium',
      background: primary,
      foreground: Colors.white,
      border: null,
      showIcon: true,
    );
  }
  if (sub.status == SubscriptionStatus.pastDue) {
    // Paiement échoué — informative orange.
    return const _BadgeVariant(
      label: 'Paiement échoué · gérer',
      semantics: 'Paiement Premium échoué, gérer l\'abonnement',
      background: Color(0xFFFFEDD5), // orange-100
      foreground: Color(0xFFC2410C), // orange-700
      border: Color(0xFFFDBA74), // orange-300
      showIcon: false,
    );
  }
  if (sub.cancelAtPeriodEnd) {
    // Auto-renew off — outline rose.
    return _BadgeVariant(
      label: 'Gérer l\'abonnement',
      semantics: 'Abonnement Premium actif, gérer',
      background: const Color(0xFFFFF1F2), // rose-50
      foreground: primary,
      border: primary,
      showIcon: false,
    );
  }
  // Active + renewing — filled rose.
  return _BadgeVariant(
    label: 'Gérer l\'abonnement',
    semantics: 'Abonnement Premium actif, gérer',
    background: primary,
    foreground: Colors.white,
    border: null,
    showIcon: false,
  );
}

class _BadgePill extends StatelessWidget {
  const _BadgePill({required this.variant, required this.onTap});

  final _BadgeVariant variant;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: variant.semantics,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(999),
        child: Container(
          height: 26,
          padding: const EdgeInsets.symmetric(horizontal: 12),
          decoration: BoxDecoration(
            color: variant.background,
            borderRadius: BorderRadius.circular(999),
            border: variant.border != null
                ? Border.all(color: variant.border!)
                : null,
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (variant.showIcon) ...[
                Icon(
                  Icons.auto_awesome,
                  size: 14,
                  color: variant.foreground,
                ),
                const SizedBox(width: 6),
              ],
              Text(
                variant.label,
                style: TextStyle(
                  color: variant.foreground,
                  fontSize: 11,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 0.1,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _BadgeSkeleton extends StatelessWidget {
  const _BadgeSkeleton();

  @override
  Widget build(BuildContext context) {
    final appColors = Theme.of(context).extension<AppColors>();
    return Container(
      height: 26,
      width: 110,
      decoration: BoxDecoration(
        color: appColors?.muted ?? Theme.of(context).dividerColor,
        borderRadius: BorderRadius.circular(999),
      ),
    );
  }
}
