import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';

/// Soleil v2 visual section card with a Fraunces title, optional
/// caption subtitle, and an arbitrary [child] body. Shared by every
/// section of the billing form.
class BillingSection extends StatelessWidget {
  const BillingSection({
    super.key,
    required this.title,
    required this.child,
    this.subtitle,
  });

  final String title;
  final String? subtitle;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            title,
            style: SoleilTextStyles.titleMedium.copyWith(
              color: colorScheme.onSurface,
            ),
          ),
          if (subtitle != null) ...[
            const SizedBox(height: 6),
            Text(
              subtitle!,
              style: SoleilTextStyles.body.copyWith(
                color: colorScheme.onSurfaceVariant,
              ),
            ),
          ],
          const SizedBox(height: 14),
          child,
        ],
      ),
    );
  }
}

/// Single labelled `TextFormField` row — the universal input used by
/// every section of the billing form. Border, fill, focus ring, and
/// hint colors come from the global Soleil `inputDecorationTheme`, so
/// the field gets the corail focus ring out of the box.
class BillingLabeledField extends StatelessWidget {
  const BillingLabeledField({
    super.key,
    required this.label,
    required this.controller,
    this.hint,
    this.validator,
    this.keyboardType,
    this.maxLength,
  });

  final String label;
  final TextEditingController controller;
  final String? hint;
  final String? Function(String?)? validator;
  final TextInputType? keyboardType;
  final int? maxLength;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: SoleilTextStyles.bodyEmphasis.copyWith(
            color: colorScheme.onSurface,
            fontSize: 13,
          ),
        ),
        const SizedBox(height: 6),
        TextFormField(
          controller: controller,
          keyboardType: keyboardType,
          maxLength: maxLength,
          validator: validator,
          style: SoleilTextStyles.body.copyWith(
            color: colorScheme.onSurface,
          ),
          decoration: InputDecoration(
            isDense: true,
            counterText: '',
            hintText: hint,
          ),
        ),
      ],
    );
  }
}

/// Single radio tile (label + check icon + corail-soft selected state)
/// used by [BillingProfileTypeRadio]. Exposed publicly for tests.
class BillingRadioTile extends StatelessWidget {
  const BillingRadioTile({
    super.key,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    final selectedBg =
        appColors?.accentSoft ?? colorScheme.primaryContainer;
    final selectedFg = colorScheme.primary;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        decoration: BoxDecoration(
          color: selected
              ? selectedBg
              : colorScheme.surfaceContainerLowest,
          border: Border.all(
            color: selected
                ? selectedFg
                : (appColors?.border ?? theme.dividerColor),
          ),
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        ),
        child: Row(
          children: [
            Icon(
              selected
                  ? Icons.radio_button_checked
                  : Icons.radio_button_unchecked,
              size: 18,
              color: selected
                  ? selectedFg
                  : colorScheme.onSurfaceVariant,
            ),
            const SizedBox(width: 10),
            Text(
              label,
              style: SoleilTextStyles.bodyEmphasis.copyWith(
                color: selected
                    ? selectedFg
                    : colorScheme.onSurface,
                fontWeight:
                    selected ? FontWeight.w700 : FontWeight.w500,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Helper that returns the standard "required" error string when the
/// passed-in value is null or trims to an empty string.
String? billingRequiredValidator(String? v) {
  if (v == null || v.trim().isEmpty) return 'Champ obligatoire';
  return null;
}

/// SIRET validator: 14 digits with no spaces.
String? billingSiretValidator(String? v) {
  if (v == null || v.trim().isEmpty) return 'Champ obligatoire';
  final digits = v.trim();
  if (digits.length != 14 || !RegExp(r'^[0-9]{14}$').hasMatch(digits)) {
    return 'Le SIRET doit comporter 14 chiffres';
  }
  return null;
}
