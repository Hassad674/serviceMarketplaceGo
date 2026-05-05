import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

// =============================================================================
// Soleil v2 — Dashboard atoms.
//
// Editorial greeting (corail mono uppercase eyebrow + Fraunces title with
// italic-corail accent + tabac subtitle), Soleil stat cards (ivoire surface,
// soft-tinted icon disc, Geist Mono numbers), search action chips on the
// warm palette. Mirrors the web W-11 dashboard anatomy in Flutter idiom.
// =============================================================================

/// Editorial greeting block shown at the top of the dashboard.
///
/// Mirrors the web W-11 anatomy:
///   - Corail mono uppercase eyebrow (e.g. "ATELIER · TABLEAU DE BORD")
///   - Fraunces title with italic-corail accent on the trailing words
///     (e.g. "Bonjour Jean, *belle journée en perspective.*")
///   - Tabac subtitle giving role-specific context
class DashboardWelcomeBanner extends StatelessWidget {
  const DashboardWelcomeBanner({
    super.key,
    required this.displayName,
    required this.subtitle,
    this.eyebrow,
  });

  /// Localised user-facing first name (or display name fallback).
  final String displayName;

  /// Role-specific tagline shown under the title.
  final String subtitle;

  /// Optional eyebrow override (defaults to the localised
  /// `mobileDashboard_eyebrow` line — uppercase corail mono).
  final String? eyebrow;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final primary = theme.colorScheme.primary;
    final eyebrowText = eyebrow ?? l10n.mobileDashboard_eyebrow;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          eyebrowText,
          style: SoleilTextStyles.mono.copyWith(
            color: primary,
            fontSize: 11,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.4,
          ),
        ),
        const SizedBox(height: 10),
        RichText(
          text: TextSpan(
            style: SoleilTextStyles.headlineLarge.copyWith(
              color: theme.colorScheme.onSurface,
            ),
            children: [
              TextSpan(
                text: '${l10n.mobileDashboard_welcomePrefix(displayName)} ',
              ),
              TextSpan(
                text: l10n.mobileDashboard_welcomeAccent,
                style: SoleilTextStyles.headlineLarge.copyWith(
                  color: primary,
                  fontStyle: FontStyle.italic,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        Text(
          subtitle,
          style: SoleilTextStyles.bodyLarge.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

/// Describes a single search action button on the dashboard.
class DashboardSearchAction {
  DashboardSearchAction({
    required this.label,
    required this.icon,
    required this.type,
    required this.tone,
  });

  final String label;
  final IconData icon;
  final String type;

  /// Soleil tone palette pair (background pill + foreground icon/text).
  final DashboardTone tone;
}

/// Soleil-compatible tone scale used by stat cards and search chips.
///
/// Each tone is a `(background, foreground)` pair pulled from the warm
/// palette. The stat-card grid cycles through them so cards stay visually
/// distinct without leaving the Soleil identity.
class DashboardTone {
  const DashboardTone({
    required this.background,
    required this.foreground,
  });

  final Color background;
  final Color foreground;

  static DashboardTone corail(BuildContext context) {
    final colors = Theme.of(context).extension<AppColors>()!;
    return DashboardTone(
      background: colors.accentSoft,
      foreground: Theme.of(context).colorScheme.primary,
    );
  }

  static DashboardTone sapin(BuildContext context) {
    final colors = Theme.of(context).extension<AppColors>()!;
    return DashboardTone(
      background: colors.successSoft,
      foreground: colors.success,
    );
  }

  static DashboardTone pink(BuildContext context) {
    final colors = Theme.of(context).extension<AppColors>()!;
    return DashboardTone(
      background: colors.pinkSoft,
      foreground: colors.primaryDeep,
    );
  }

  static DashboardTone amber(BuildContext context) {
    final colors = Theme.of(context).extension<AppColors>()!;
    return DashboardTone(
      background: colors.amberSoft,
      foreground: Theme.of(context).colorScheme.onSurface,
    );
  }
}

/// Wraps a list of [DashboardSearchActionChip] into a flow layout —
/// each chip pushes `/search/<type>` when tapped.
class DashboardSearchActions extends StatelessWidget {
  const DashboardSearchActions({super.key, required this.actions});

  final List<DashboardSearchAction> actions;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 10,
      runSpacing: 10,
      children: actions
          .map((action) => DashboardSearchActionChip(action: action))
          .toList(),
    );
  }
}

class DashboardSearchActionChip extends StatelessWidget {
  const DashboardSearchActionChip({super.key, required this.action});

  final DashboardSearchAction action;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: action.tone.background,
      shape: const StadiumBorder(),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => GoRouter.of(context).push('/search/${action.type}'),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                action.icon,
                size: 16,
                color: action.tone.foreground,
              ),
              const SizedBox(width: 8),
              Text(
                action.label,
                style: SoleilTextStyles.caption.copyWith(
                  color: theme.colorScheme.onSurface,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Soleil v2 stat card.
///
/// Anatomy: ivoire surface, rounded 20, soft border, soft-tinted icon disc,
/// tabac caption label, Geist Mono large value (numbers/amounts).
///
/// Used in a 1-col stack on narrow viewports and a 2-col grid on wider ones
/// — the [DashboardStatGrid] handles the layout choice.
class DashboardStatCard extends StatelessWidget {
  const DashboardStatCard({
    super.key,
    required this.icon,
    required this.title,
    required this.value,
    required this.tone,
  });

  final IconData icon;
  final String title;
  final String value;
  final DashboardTone tone;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(color: colors.border),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: tone.background,
              borderRadius: BorderRadius.circular(AppTheme.radiusXl),
            ),
            child: Icon(icon, color: tone.foreground, size: 22),
          ),
          const SizedBox(height: 16),
          Text(
            title,
            style: SoleilTextStyles.caption.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            value,
            style: SoleilTextStyles.monoLarge.copyWith(
              color: theme.colorScheme.onSurface,
              fontWeight: FontWeight.w600,
              fontSize: 22,
            ),
          ),
        ],
      ),
    );
  }
}

/// Responsive grid that lays out [DashboardStatCard] in:
///   - 1 column on very narrow screens (< 380dp)
///   - 2 columns on standard mobile widths
///
/// Each card stretches to fill its grid cell so heights stay aligned. The
/// grid is wrapped in a [LayoutBuilder] so it adapts to the constraints
/// of the surrounding scroll view (no `MediaQuery` lookup at build time).
class DashboardStatGrid extends StatelessWidget {
  const DashboardStatGrid({super.key, required this.cards});

  final List<DashboardStatCard> cards;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final twoColumns = constraints.maxWidth >= 380;
        const gap = 12.0;
        if (!twoColumns) {
          return Column(
            children: [
              for (var i = 0; i < cards.length; i++) ...[
                if (i > 0) const SizedBox(height: gap),
                cards[i],
              ],
            ],
          );
        }
        final cellWidth = (constraints.maxWidth - gap) / 2;
        return Wrap(
          spacing: gap,
          runSpacing: gap,
          children: cards
              .map(
                (card) => SizedBox(
                  width: cellWidth,
                  child: card,
                ),
              )
              .toList(),
        );
      },
    );
  }
}

/// Pill-shaped Soleil button used as the referrer/freelance switch.
///
/// Renders a stadium-shaped button with a leading icon and the label.
/// The accent palette pair (`background`, `foreground`) is supplied
/// by the caller (corail-soft for "switch to referrer", sapin-soft for
/// "back to freelance").
class DashboardSwitchPill extends StatelessWidget {
  const DashboardSwitchPill({
    super.key,
    required this.label,
    required this.icon,
    required this.onPressed,
    required this.tone,
  });

  final String label;
  final IconData icon;
  final VoidCallback onPressed;
  final DashboardTone tone;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: tone.background,
      shape: const StadiumBorder(),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onPressed,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 16, color: tone.foreground),
              const SizedBox(width: 8),
              Text(
                label,
                style: SoleilTextStyles.button.copyWith(
                  color: tone.foreground,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
