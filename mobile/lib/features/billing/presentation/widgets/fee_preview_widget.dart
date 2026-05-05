import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/fee_preview.dart';
import '../providers/fee_preview_provider.dart';

/// Formats an integer in cents as a euro string: `12345` -> `123,45 €`.
///
/// Top-level helper (not a widget method) so it can be reused by both the
/// single-amount and milestone breakdown layouts without duplication.
String _formatCents(int cents) {
  final sign = cents < 0 ? '-' : '';
  final abs = cents.abs();
  final euros = abs ~/ 100;
  final remainder = (abs % 100).toString().padLeft(2, '0');
  return '$sign$euros,$remainder €';
}

/// A single milestone line item for the breakdown mode.
///
/// Kept as a top-level class so widget tests can drive it directly and the
/// enclosing widget's build stays small (≤100 lines).
class FeeMilestoneLine {
  const FeeMilestoneLine({required this.label, required this.amountCents});

  final String label;
  final int amountCents;
}

/// Shows the platform fee grid and the concrete earnings for the
/// prestataire drafting (or viewing) a proposal.
///
/// Two modes:
///   1. **Single amount** — pass a non-null [amountCents]. The widget shows
///      "You earn X (Platform fees: Y)" plus the 3-tier grid with the
///      active tier highlighted.
///   2. **Milestones** — pass [milestones] (non-empty). The widget shows
///      one row per milestone (gross / fee / net) and the cumulative total,
///      plus the grid. Each milestone drives its own fee lookup.
///
/// The widget NEVER triggers network calls eagerly. The caller is
/// responsible for debouncing the amount input and passing the debounced
/// value(s) — typically via a 300ms timer on the text field.
///
/// Role-gating is enforced INSIDE this widget — if the backend resolves
/// the viewer to a client-side role (`viewerIsProvider == false`), the
/// widget collapses to `SizedBox.shrink()`. Callers do not need to check
/// the role themselves; passing a [recipientId] lets the backend
/// disambiguate agency pairings.
///
/// Soleil v2 styling: ivoire surface card with Soleil radius 2xl,
/// Fraunces title, corail-soft icon disc, Geist (mono) numerals,
/// corail-tinted active tier highlight. Colors come from the theme
/// extension `AppColors` — no `Color(0xFF...)` literals.
class FeePreviewWidget extends ConsumerWidget {
  const FeePreviewWidget({
    super.key,
    this.amountCents,
    this.milestones = const [],
    this.recipientId,
  }) : assert(
          amountCents != null || milestones.length > 0,
          'Provide either amountCents or a non-empty milestones list',
        );

  /// Single-amount mode. Ignored when [milestones] is non-empty.
  final int? amountCents;

  /// Milestone breakdown mode. Empty list falls back to single-amount.
  final List<FeeMilestoneLine> milestones;

  /// Recipient of the proposal. When non-null the backend resolves the
  /// viewer's role relative to that recipient (agency ↔ enterprise vs.
  /// agency ↔ provider) and can flip `viewerIsProvider` to false — in
  /// which case the widget hides itself.
  final String? recipientId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Zero or negative amounts are meaningless — render nothing so the
    // proposal form stays clean while the user is still typing.
    if (milestones.isEmpty) {
      final amt = amountCents ?? 0;
      if (amt <= 0) return const SizedBox.shrink();
      return _SingleAmountView(
        amountCents: amt,
        recipientId: recipientId,
      );
    }
    return _MilestonesView(
      milestones: milestones,
      recipientId: recipientId,
    );
  }
}

/// Shared chrome around every variant: ivoire card with corail-soft
/// icon disc + Fraunces title.
class _FeePreviewCard extends StatelessWidget {
  const _FeePreviewCard({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final iconBg = appColors?.accentSoft ?? theme.colorScheme.primary.withValues(alpha: 0.12);

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                  color: iconBg,
                  borderRadius: BorderRadius.circular(AppTheme.radiusLg),
                ),
                alignment: Alignment.center,
                child: Icon(
                  Icons.receipt_long_outlined,
                  size: 18,
                  color: appColors?.primaryDeep ?? theme.colorScheme.primary,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Platform fees',
                      style: SoleilTextStyles.titleMedium.copyWith(
                        color: theme.colorScheme.onSurface,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Flat fee per milestone',
                      style: SoleilTextStyles.caption.copyWith(
                        color: appColors?.mutedForeground ?? theme.hintColor,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }
}

/// Single-amount mode. Watches the provider for `amountCents` and renders
/// loading / error / data states.
class _SingleAmountView extends ConsumerWidget {
  const _SingleAmountView({required this.amountCents, this.recipientId});

  final int amountCents;
  final String? recipientId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final key = FeePreviewKey(
      amountCents: amountCents,
      recipientId: recipientId,
    );
    final async = ref.watch(feePreviewProvider(key));
    return async.when(
      // Role gate: when the viewer is client-side the backend sets
      // viewerIsProvider=false and we render nothing at all — no card,
      // no grid, no label. This is the ONE place that enforces the
      // billing-visibility contract for every caller.
      data: (preview) => preview.viewerIsProvider
          ? _FeePreviewCard(child: _SingleAmountData(preview: preview))
          : const SizedBox.shrink(),
      loading: () => const _FeePreviewCard(child: _FeePreviewSkeleton()),
      error: (err, _) => _FeePreviewCard(
        child: _FeePreviewError(
          onRetry: () => ref.invalidate(feePreviewProvider(key)),
        ),
      ),
    );
  }
}

/// Data-state rendering for the single-amount mode.
class _SingleAmountData extends StatelessWidget {
  const _SingleAmountData({required this.preview});

  final FeePreview preview;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        RichText(
          text: TextSpan(
            style: SoleilTextStyles.body.copyWith(
              color: theme.colorScheme.onSurface,
            ),
            children: [
              const TextSpan(text: 'You earn '),
              TextSpan(
                text: _formatCents(preview.netCents),
                style: SoleilTextStyles.monoLarge.copyWith(
                  color: appColors?.primaryDeep ?? theme.colorScheme.primary,
                ),
              ),
              TextSpan(
                text: '  (Platform fees: ${_formatCents(preview.feeCents)})',
                style: SoleilTextStyles.caption.copyWith(
                  color: appColors?.mutedForeground ?? theme.hintColor,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 14),
        _FeeGrid(
          tiers: preview.tiers,
          activeIndex: preview.activeTierIndex,
        ),
      ],
    );
  }
}

/// Milestone-breakdown mode. Watches one provider per unique amount so
/// that edits to a single milestone do not retrigger the others.
class _MilestonesView extends ConsumerWidget {
  const _MilestonesView({required this.milestones, this.recipientId});

  final List<FeeMilestoneLine> milestones;
  final String? recipientId;

  FeePreviewKey _keyFor(int amountCents) => FeePreviewKey(
        amountCents: amountCents,
        recipientId: recipientId,
      );

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    // Aggregate state: if any preview is loading/error, reflect that at
    // the top-level widget while still showing the partial breakdown.
    final previews = <int, AsyncValue<FeePreview>>{
      for (final m in milestones)
        if (m.amountCents > 0)
          m.amountCents: ref.watch(feePreviewProvider(_keyFor(m.amountCents))),
    };

    int totalGross = 0;
    int totalFees = 0;
    int totalNet = 0;
    List<FeeTier>? latestTiers;
    int? latestActive;
    bool anyLoading = false;
    bool anyError = false;
    // Role gate: the backend returns viewerIsProvider=false uniformly for
    // a given viewer → if ANY resolved preview tells us the viewer is
    // client-side, hide the whole milestones block.
    bool viewerIsClient = false;

    for (final m in milestones) {
      if (m.amountCents <= 0) continue;
      final async = previews[m.amountCents]!;
      async.when(
        data: (p) {
          if (!p.viewerIsProvider) {
            viewerIsClient = true;
            return;
          }
          totalGross += p.amountCents;
          totalFees += p.feeCents;
          totalNet += p.netCents;
          latestTiers = p.tiers;
          latestActive = p.activeTierIndex;
        },
        loading: () => anyLoading = true,
        error: (_, __) => anyError = true,
      );
    }

    if (viewerIsClient) return const SizedBox.shrink();

    return _FeePreviewCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final m in milestones)
            _MilestoneRow(line: m, state: previews[m.amountCents]),
          const SizedBox(height: 12),
          Divider(
            height: 1,
            color: appColors?.border ?? theme.dividerColor,
          ),
          const SizedBox(height: 12),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                'Total',
                style: SoleilTextStyles.titleMedium.copyWith(
                  color: theme.colorScheme.onSurface,
                ),
              ),
              Text(
                '${_formatCents(totalNet)}'
                '  (Fees: ${_formatCents(totalFees)} / ${_formatCents(totalGross)})',
                style: SoleilTextStyles.monoLarge.copyWith(
                  color: appColors?.primaryDeep ?? theme.colorScheme.primary,
                ),
              ),
            ],
          ),
          if (anyLoading) ...[
            const SizedBox(height: 12),
            const _FeePreviewSkeleton(),
          ] else if (anyError) ...[
            const SizedBox(height: 12),
            _FeePreviewError(
              onRetry: () {
                for (final m in milestones) {
                  if (m.amountCents > 0) {
                    ref.invalidate(feePreviewProvider(_keyFor(m.amountCents)));
                  }
                }
              },
            ),
          ] else if (latestTiers != null) ...[
            const SizedBox(height: 14),
            _FeeGrid(tiers: latestTiers!, activeIndex: latestActive ?? -1),
          ],
        ],
      ),
    );
  }
}

/// Single milestone row: label + gross / fee / net.
class _MilestoneRow extends StatelessWidget {
  const _MilestoneRow({required this.line, required this.state});

  final FeeMilestoneLine line;
  final AsyncValue<FeePreview>? state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final muted = appColors?.mutedForeground ?? theme.hintColor;

    String trailing;
    if (line.amountCents <= 0) {
      trailing = '—';
    } else {
      trailing = state?.when(
            data: (p) =>
                '${_formatCents(p.netCents)}  (-${_formatCents(p.feeCents)})',
            loading: () => '…',
            error: (_, __) => '!',
          ) ??
          '—';
    }

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 5),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Expanded(
            child: Text(
              line.label.isNotEmpty ? line.label : 'Milestone',
              style: SoleilTextStyles.body.copyWith(color: muted),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          const SizedBox(width: 8),
          Text(
            trailing,
            style: SoleilTextStyles.mono.copyWith(
              fontSize: 13,
              color: theme.colorScheme.onSurface,
            ),
          ),
        ],
      ),
    );
  }
}

/// Renders the 3-tier grid. The active tier is highlighted with a
/// corail-soft background and a corail-tinted left bar.
class _FeeGrid extends StatelessWidget {
  const _FeeGrid({required this.tiers, required this.activeIndex});

  final List<FeeTier> tiers;
  final int activeIndex;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          for (var i = 0; i < tiers.length; i++) ...[
            if (i > 0)
              Divider(
                height: 1,
                color: appColors?.border ?? theme.dividerColor,
              ),
            _FeeGridRow(
              tier: tiers[i],
              active: i == activeIndex,
              activeBg: appColors?.accentSoft ??
                  theme.colorScheme.primary.withValues(alpha: 0.08),
              activeAccent: appColors?.primaryDeep ?? theme.colorScheme.primary,
            ),
          ],
        ],
      ),
    );
  }
}

class _FeeGridRow extends StatelessWidget {
  const _FeeGridRow({
    required this.tier,
    required this.active,
    required this.activeBg,
    required this.activeAccent,
  });

  final FeeTier tier;
  final bool active;
  final Color activeBg;
  final Color activeAccent;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: active ? activeBg : null,
        border: Border(
          left: BorderSide(
            color: active ? activeAccent : Colors.transparent,
            width: 4,
          ),
        ),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Expanded(
            child: Text(
              tier.label,
              style: SoleilTextStyles.body.copyWith(
                fontWeight: active ? FontWeight.w600 : FontWeight.w500,
                color: active
                    ? activeAccent
                    : theme.colorScheme.onSurface,
              ),
            ),
          ),
          const SizedBox(width: 8),
          Text(
            _formatCents(tier.feeCents),
            style: SoleilTextStyles.monoLarge.copyWith(
              fontSize: 14,
              fontWeight: active ? FontWeight.w700 : FontWeight.w500,
              color: active
                  ? activeAccent
                  : (appColors?.mutedForeground ?? theme.hintColor),
            ),
          ),
        ],
      ),
    );
  }
}

/// Skeleton placeholder while the preview is loading.
class _FeePreviewSkeleton extends StatelessWidget {
  const _FeePreviewSkeleton();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final bar = BoxDecoration(
      color: appColors?.muted ?? theme.dividerColor,
      borderRadius: BorderRadius.circular(AppTheme.radiusLg),
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(height: 16, width: 200, decoration: bar),
        const SizedBox(height: 12),
        Container(height: 40, decoration: bar),
        const SizedBox(height: 8),
        Container(height: 40, decoration: bar),
        const SizedBox(height: 8),
        Container(height: 40, decoration: bar),
      ],
    );
  }
}

/// Retryable error state for network/server failures.
class _FeePreviewError extends StatelessWidget {
  const _FeePreviewError({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: theme.colorScheme.error.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.colorScheme.error.withValues(alpha: 0.4),
        ),
      ),
      child: Row(
        children: [
          Icon(
            Icons.error_outline,
            size: 18,
            color: theme.colorScheme.error,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              'Could not load the platform fee. Tap to retry.',
              style: SoleilTextStyles.caption.copyWith(
                color: theme.colorScheme.error,
              ),
            ),
          ),
          TextButton(
            onPressed: onRetry,
            child: const Text('Retry'),
          ),
        ],
      ),
    );
  }
}
