import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';

/// Shared visual shell for all `/stats` cards (visibility, applications,
/// keywords). Soleil v2 idiom: white surface, soft border, 16-radius
/// corners, padded body.
///
/// Centralises the chrome so the rule-of-three doesn't drift across the
/// three card files (each file just supplies its title + body).
class StatsCardShell extends StatelessWidget {
  const StatsCardShell({
    super.key,
    required this.title,
    required this.child,
    this.subtitle,
    this.trailing,
  });

  final String title;
  final Widget child;
  final String? subtitle;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      padding: const EdgeInsets.fromLTRB(18, 16, 18, 18),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: SoleilTextStyles.titleMedium.copyWith(
                        color: theme.colorScheme.onSurface,
                      ),
                    ),
                    if (subtitle != null) ...[
                      const SizedBox(height: 2),
                      Text(
                        subtitle!,
                        style: SoleilTextStyles.caption.copyWith(
                          color: appColors?.mutedForeground ??
                              theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              if (trailing != null) trailing!,
            ],
          ),
          const SizedBox(height: 14),
          child,
        ],
      ),
    );
  }
}

/// Tiny stand-in for the in-card "data insufficient" empty state. Used
/// by every card when the series is empty / all-zero.
class StatsCardEmpty extends StatelessWidget {
  const StatsCardEmpty({super.key, required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Text(
        message,
        style: SoleilTextStyles.body.copyWith(
          color: appColors?.mutedForeground ??
              theme.colorScheme.onSurfaceVariant,
        ),
      ),
    );
  }
}

/// Inline error fallback rendered inside a card when its FutureProvider
/// failed. Offers a retry callback the parent can wire to
/// `ref.invalidate`.
class StatsCardError extends StatelessWidget {
  const StatsCardError({super.key, required this.message, this.onRetry});

  final String message;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      children: [
        Icon(
          Icons.error_outline_rounded,
          size: 18,
          color: theme.colorScheme.error,
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            message,
            style: SoleilTextStyles.body.copyWith(
              color: theme.colorScheme.error,
            ),
          ),
        ),
        if (onRetry != null)
          TextButton(
            onPressed: onRetry,
            child: Text(
              MaterialLocalizations.of(context).okButtonLabel,
              style: SoleilTextStyles.button,
            ),
          ),
      ],
    );
  }
}

/// Skeleton placeholder while a card is loading. Soleil pattern: soft
/// muted block(s) at the same height as the eventual content so the
/// layout doesn't shift when data arrives.
class StatsCardSkeleton extends StatelessWidget {
  const StatsCardSkeleton({super.key, this.height = 90});

  final double height;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      height: height,
      decoration: BoxDecoration(
        color: appColors?.muted ?? theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(12),
      ),
    );
  }
}
