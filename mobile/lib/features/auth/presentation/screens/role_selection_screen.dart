import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

/// M-02 — Inscription · choix de rôle. Soleil v2 visual port.
///
/// Single column layout: a minimal top bar with a back arrow and the
/// account-creation eyebrow / "Étape 1 sur 3" indicator, an editorial
/// Fraunces headline with italic corail accent, then three vertically
/// stacked role cards (Agency / Provider / Enterprise) and a corail
/// "Continuer" pill button anchored to the bottom of the safe area.
///
/// Source: `design/assets/sources/phase1/soleil-app-lot5.jsx`
/// `AppSignupRole` (lines 65-132) + `design/assets/pdf/app-native-ios.pdf`
/// page 3 (right frame).
///
/// Repo taxonomy precedence: the JSX shows three options
/// (provider / enterprise / "the two"), but the existing register flow
/// pins three sub-routes (agency / provider / enterprise) and the
/// freelance card already exposes the apporteur d'affaires toggle via
/// `referrer_enabled`. We mirror that taxonomy — agency on top, the
/// freelance/business-referrer card preselected (it's the most common
/// signup path), enterprise last. Splitting "Apporteur" into its own
/// card would require a backend / register schema change.
class RoleSelectionScreen extends StatefulWidget {
  const RoleSelectionScreen({super.key});

  @override
  State<RoleSelectionScreen> createState() => _RoleSelectionScreenState();
}

class _RoleSelectionScreenState extends State<RoleSelectionScreen> {
  /// Locally-selected card. Defaults to provider — most common signup
  /// path and matches the source maquette which shows the freelance
  /// card highlighted by default.
  String _selected = 'provider';

  void _select(String role) {
    if (_selected == role) return;
    setState(() => _selected = role);
  }

  void _onContinue() {
    final route = switch (_selected) {
      'agency' => RoutePaths.registerAgency,
      'enterprise' => RoutePaths.registerEnterprise,
      _ => RoutePaths.registerProvider,
    };
    context.go(route);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      backgroundColor: colorScheme.surfaceContainerLowest,
      body: SafeArea(
        child: Column(
          children: [
            _SignupTopBar(
              eyebrow: l10n.m02_eyebrowLabel,
              step: l10n.m02_stepIndicator,
              onBack: () => context.go(RoutePaths.login),
            ),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(28, 20, 28, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    _Header(
                      titlePrefix: l10n.m02_titlePrefix,
                      titleAccent: l10n.m02_titleAccent,
                      subtitle: l10n.m02_subtitle,
                    ),
                    const SizedBox(height: 28),
                    _RoleCard(
                      icon: Icons.work_rounded,
                      title: l10n.roleAgency,
                      description: l10n.m02_agencyDesc,
                      selected: _selected == 'agency',
                      onTap: () => _select('agency'),
                    ),
                    const SizedBox(height: 12),
                    _RoleCard(
                      icon: Icons.person_rounded,
                      title: l10n.roleFreelance,
                      description: l10n.m02_providerDesc,
                      selected: _selected == 'provider',
                      onTap: () => _select('provider'),
                    ),
                    const SizedBox(height: 12),
                    _RoleCard(
                      icon: Icons.apartment_rounded,
                      title: l10n.roleEnterprise,
                      description: l10n.m02_enterpriseDesc,
                      selected: _selected == 'enterprise',
                      onTap: () => _select('enterprise'),
                    ),
                  ],
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(28, 0, 28, 24),
              child: SizedBox(
                height: 52,
                child: ElevatedButton(
                  onPressed: _onContinue,
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Text(l10n.m02_continue),
                      const SizedBox(width: 6),
                      const Icon(Icons.arrow_forward_rounded, size: 18),
                    ],
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Top bar — minimal, ivoire-bg back button on the left, then a small
/// stack with the eyebrow caption and the step indicator. Mirrors the
/// JSX top bar (lot 5 lines 68-76).
class _SignupTopBar extends StatelessWidget {
  const _SignupTopBar({
    required this.eyebrow,
    required this.step,
    required this.onBack,
  });

  final String eyebrow;
  final String step;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 6, 28, 8),
      child: Row(
        children: [
          // Round back button — ivoire bg, no border, encre glyph
          Material(
            color: colorScheme.surface,
            shape: const CircleBorder(),
            child: InkWell(
              customBorder: const CircleBorder(),
              onTap: onBack,
              child: SizedBox(
                width: 36,
                height: 36,
                child: Icon(
                  Icons.arrow_back_rounded,
                  size: 18,
                  color: colorScheme.onSurface,
                ),
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  eyebrow,
                  style: SoleilTextStyles.caption.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  step,
                  style: SoleilTextStyles.caption.copyWith(
                    color: colorScheme.onSurface,
                    fontWeight: FontWeight.w600,
                    fontSize: 13.5,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Editorial header — Fraunces 28 display title with the accent half
/// rendered in italic corail (Soleil signature), plus an italic Fraunces
/// subtitle in tabac.
class _Header extends StatelessWidget {
  const _Header({
    required this.titlePrefix,
    required this.titleAccent,
    required this.subtitle,
  });

  final String titlePrefix;
  final String titleAccent;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    final base = SoleilTextStyles.headlineLarge.copyWith(
      color: colorScheme.onSurface,
      fontWeight: FontWeight.w600,
      height: 1.15,
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        RichText(
          text: TextSpan(
            style: base,
            children: [
              TextSpan(text: '$titlePrefix '),
              TextSpan(
                text: titleAccent,
                style: base.copyWith(
                  fontStyle: FontStyle.italic,
                  color: colorScheme.primary,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        Text(
          subtitle,
          style: SoleilTextStyles.body.copyWith(
            fontStyle: FontStyle.italic,
            color: colorScheme.onSurfaceVariant,
            fontFamily: SoleilTextStyles.headlineLarge.fontFamily,
            fontSize: 14,
          ),
        ),
      ],
    );
  }
}

/// Single role card — radius 18, padding 18, leading 46x46 icon square,
/// title (Fraunces 17/600) + description (Inter Tight 13/tabac), trailing
/// 22x22 selection pip. The selected state lifts the card with corailSoft
/// background, 2px corail border, corail-filled icon square + white check
/// pip. Unselected stays surface-on-border.
class _RoleCard extends StatelessWidget {
  const _RoleCard({
    required this.icon,
    required this.title,
    required this.description,
    required this.selected,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String description;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final palette = theme.extension<AppColors>()!;

    final backgroundColor =
        selected ? palette.accentSoft : colorScheme.surface;
    final borderColor =
        selected ? colorScheme.primary : colorScheme.outline;
    final borderWidth = selected ? 2.0 : 1.0;
    final iconBackground =
        selected ? colorScheme.primary : colorScheme.surfaceContainerHigh;
    final iconColor =
        selected ? colorScheme.onPrimary : colorScheme.onSurface;

    return Semantics(
      button: true,
      selected: selected,
      label: title,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          curve: Curves.easeOut,
          padding: const EdgeInsets.all(18),
          decoration: BoxDecoration(
            color: backgroundColor,
            borderRadius: BorderRadius.circular(AppTheme.radiusXl),
            border: Border.all(color: borderColor, width: borderWidth),
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _RoleIconSquare(
                icon: icon,
                background: iconBackground,
                foreground: iconColor,
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: SoleilTextStyles.titleMedium.copyWith(
                        color: colorScheme.onSurface,
                        fontSize: 17,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      description,
                      style: SoleilTextStyles.caption.copyWith(
                        color: colorScheme.onSurfaceVariant,
                        fontSize: 12.5,
                        height: 1.45,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 12),
              _SelectionPip(selected: selected),
            ],
          ),
        ),
      ),
    );
  }
}

/// Leading 46x46 rounded square holding the role glyph. Filled corail
/// when selected, otherwise filled with the warm ivoire surface.
class _RoleIconSquare extends StatelessWidget {
  const _RoleIconSquare({
    required this.icon,
    required this.background,
    required this.foreground,
  });

  final IconData icon;
  final Color background;
  final Color foreground;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 46,
      height: 46,
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      ),
      alignment: Alignment.center,
      child: Icon(icon, size: 20, color: foreground),
    );
  }
}

/// Trailing 22x22 selection indicator. When selected: filled corail
/// circle + white check glyph. When not selected: outlined circle with a
/// 2px sand border (Soleil radio-style).
class _SelectionPip extends StatelessWidget {
  const _SelectionPip({required this.selected});

  final bool selected;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    if (selected) {
      return Container(
        width: 22,
        height: 22,
        decoration: BoxDecoration(
          color: colorScheme.primary,
          shape: BoxShape.circle,
        ),
        alignment: Alignment.center,
        child: Icon(
          Icons.check_rounded,
          size: 13,
          color: colorScheme.onPrimary,
        ),
      );
    }
    return Container(
      width: 22,
      height: 22,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: Border.all(color: colorScheme.outline, width: 2),
      ),
    );
  }
}
