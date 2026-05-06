import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';

/// Editorial hero shown above the search field on the freelancer
/// persona screen. Eyebrow + Fraunces title with italic-corail accent
/// + tabac italic subtitle.
///
/// Pure presentation. Extracted from `search_screen.dart` as part of
/// the NF-9 file split (V7 audit). Behaviour unchanged.
class SearchM12Hero extends StatelessWidget {
  const SearchM12Hero({super.key, required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.freelancesSearchM12Eyebrow,
            style: SoleilTextStyles.mono.copyWith(
              fontSize: 11,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.8,
              color: colorScheme.primary,
            ),
          ),
          const SizedBox(height: 8),
          Text.rich(
            TextSpan(
              children: [
                TextSpan(
                  text: '${l10n.freelancesSearchM12TitleLead} ',
                  style: SoleilTextStyles.headlineLarge.copyWith(
                    fontSize: 26,
                    fontWeight: FontWeight.w500,
                    letterSpacing: -0.5,
                    color: colorScheme.onSurface,
                  ),
                ),
                TextSpan(
                  text: l10n.freelancesSearchM12TitleAccent,
                  style: SoleilTextStyles.headlineLarge.copyWith(
                    fontSize: 26,
                    fontWeight: FontWeight.w500,
                    letterSpacing: -0.5,
                    fontStyle: FontStyle.italic,
                    color: colorScheme.primary,
                  ),
                ),
                TextSpan(
                  text: '.',
                  style: SoleilTextStyles.headlineLarge.copyWith(
                    fontSize: 26,
                    fontWeight: FontWeight.w500,
                    color: colorScheme.onSurface,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 6),
          Text(
            l10n.freelancesSearchM12Subtitle,
            style: SoleilTextStyles.body.copyWith(
              fontSize: 13.5,
              fontStyle: FontStyle.italic,
              color: colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}

/// Soleil search field — full-pill, ivoire bg, corail focus aura.
///
/// Extracted from `search_screen.dart` as part of the NF-9 file split.
class SearchField extends StatelessWidget {
  const SearchField({
    super.key,
    required this.controller,
    required this.onChanged,
    required this.hintText,
  });

  final TextEditingController controller;
  final ValueChanged<String> onChanged;
  final String hintText;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 14, 20, 6),
      child: Semantics(
        textField: true,
        label: hintText,
        child: TextField(
          controller: controller,
          onChanged: onChanged,
          textInputAction: TextInputAction.search,
          style: SoleilTextStyles.body.copyWith(
            color: colorScheme.onSurface,
          ),
          decoration: InputDecoration(
            hintText: hintText,
            hintStyle: SoleilTextStyles.body.copyWith(
              fontStyle: FontStyle.italic,
              color: colors.subtleForeground,
            ),
            prefixIcon: Icon(
              Icons.search_rounded,
              size: 18,
              color: colorScheme.onSurfaceVariant,
            ),
            filled: true,
            fillColor: colorScheme.surfaceContainerLowest,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
              borderSide: BorderSide(color: colors.border),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
              borderSide: BorderSide(color: colors.border),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
              borderSide: BorderSide(color: colorScheme.primary, width: 1.5),
            ),
            isDense: true,
          ),
        ),
      ),
    );
  }
}

/// Round filter button used in the AppBar actions.
///
/// Extracted from `search_screen.dart` as part of the NF-9 file split.
class SearchFilterButton extends StatelessWidget {
  const SearchFilterButton({
    super.key,
    required this.onTap,
    required this.tooltip,
  });

  final VoidCallback onTap;
  final String tooltip;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Tooltip(
      message: tooltip,
      child: Material(
        color: colorScheme.surfaceContainerLowest,
        shape: CircleBorder(side: BorderSide(color: colors.border)),
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: onTap,
          child: SizedBox(
            width: 36,
            height: 36,
            child: Icon(
              Icons.tune_rounded,
              size: 18,
              color: colorScheme.onSurface,
            ),
          ),
        ),
      ),
    );
  }
}
