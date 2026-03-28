import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/payment_info.dart';

// ---------------------------------------------------------------------------
// Country data — shared between widgets
// ---------------------------------------------------------------------------

class Country {
  const Country(this.code, this.name);
  final String code;
  final String name;
}

const countries = [
  Country('FR', 'France'),
  Country('DE', 'Germany'),
  Country('ES', 'Spain'),
  Country('IT', 'Italy'),
  Country('PT', 'Portugal'),
  Country('NL', 'Netherlands'),
  Country('BE', 'Belgium'),
  Country('LU', 'Luxembourg'),
  Country('CH', 'Switzerland'),
  Country('AT', 'Austria'),
  Country('IE', 'Ireland'),
  Country('GB', 'United Kingdom'),
  Country('SE', 'Sweden'),
  Country('DK', 'Denmark'),
  Country('NO', 'Norway'),
  Country('FI', 'Finland'),
  Country('PL', 'Poland'),
  Country('CZ', 'Czech Republic'),
  Country('RO', 'Romania'),
  Country('GR', 'Greece'),
  Country('US', 'United States'),
  Country('CA', 'Canada'),
  Country('AU', 'Australia'),
  Country('MA', 'Morocco'),
  Country('TN', 'Tunisia'),
  Country('SN', 'Senegal'),
];

const ibanCountryCodes = {
  'FR', 'DE', 'ES', 'IT', 'PT', 'NL', 'BE', 'LU', 'CH', 'AT', 'IE', 'GB',
  'SE', 'DK', 'NO', 'FI', 'PL', 'CZ', 'RO', 'GR',
};

/// Callback signature used to mutate [PaymentInfoFormData].
typedef FormUpdater = void Function(
  PaymentInfoFormData Function(PaymentInfoFormData) updater,
);

// ---------------------------------------------------------------------------
// Status banner — saved or incomplete
// ---------------------------------------------------------------------------

class PaymentStatusBanner extends StatelessWidget {
  const PaymentStatusBanner({super.key, required this.saved});
  final bool saved;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final isDark = Theme.of(context).brightness == Brightness.dark;

    if (saved) {
      return _banner(
        isDark: isDark,
        icon: Icons.check_circle_outline,
        text: l10n.paymentInfoSaved,
        bgLight: const Color(0xFFECFDF5),
        bgDark: const Color(0xFF22C55E),
        borderLight: const Color(0xFFA7F3D0),
        borderDark: const Color(0xFF22C55E),
        fgLight: const Color(0xFF15803D),
        fgDark: const Color(0xFF4ADE80),
        iconLight: const Color(0xFF16A34A),
        iconDark: const Color(0xFF4ADE80),
      );
    }

    return _banner(
      isDark: isDark,
      icon: Icons.warning_amber_outlined,
      text: l10n.paymentInfoIncomplete,
      bgLight: const Color(0xFFFFFBEB),
      bgDark: const Color(0xFFF59E0B),
      borderLight: const Color(0xFFFDE68A),
      borderDark: const Color(0xFFF59E0B),
      fgLight: const Color(0xFF92400E),
      fgDark: const Color(0xFFFBBF24),
      iconLight: const Color(0xFFD97706),
      iconDark: const Color(0xFFFBBF24),
    );
  }

  Widget _banner({
    required bool isDark,
    required IconData icon,
    required String text,
    required Color bgLight,
    required Color bgDark,
    required Color borderLight,
    required Color borderDark,
    required Color fgLight,
    required Color fgDark,
    required Color iconLight,
    required Color iconDark,
  }) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: isDark ? bgDark.withValues(alpha: 0.1) : bgLight,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: isDark ? borderDark.withValues(alpha: 0.3) : borderLight,
        ),
      ),
      child: Row(
        children: [
          Icon(icon, size: 20, color: isDark ? iconDark : iconLight),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              text,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w500,
                color: isDark ? fgDark : fgLight,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Business toggle
// ---------------------------------------------------------------------------

class PaymentBusinessToggle extends StatelessWidget {
  const PaymentBusinessToggle({
    super.key,
    required this.value,
    required this.onChanged,
  });

  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Row(
      children: [
        Switch.adaptive(
          value: value,
          onChanged: onChanged,
          activeTrackColor: const Color(0xFFF43F5E),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            l10n.paymentInfoIsBusiness,
            style: theme.textTheme.bodyMedium?.copyWith(
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Section card wrapper
// ---------------------------------------------------------------------------

class PaymentSectionCard extends StatelessWidget {
  const PaymentSectionCard({
    super.key,
    required this.title,
    required this.children,
  });

  final String title;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 12),
          ...children,
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Styled text field
// ---------------------------------------------------------------------------

class PaymentFormField extends StatelessWidget {
  const PaymentFormField({
    super.key,
    required this.label,
    required this.value,
    required this.onChanged,
    this.keyboardType,
    this.placeholder,
    this.required = false,
  });

  final String label;
  final String value;
  final ValueChanged<String> onChanged;
  final TextInputType? keyboardType;
  final String? placeholder;
  final bool required;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextFormField(
        initialValue: value,
        onChanged: onChanged,
        keyboardType: keyboardType,
        decoration: InputDecoration(
          labelText: required ? '$label *' : label,
          hintText: placeholder,
          floatingLabelBehavior: FloatingLabelBehavior.auto,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Date field with picker
// ---------------------------------------------------------------------------

class PaymentDateField extends StatelessWidget {
  const PaymentDateField({
    super.key,
    required this.label,
    required this.value,
    required this.onChanged,
  });

  final String label;
  final String value;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: GestureDetector(
        onTap: () async {
          final now = DateTime.now();
          final picked = await showDatePicker(
            context: context,
            initialDate: DateTime(1990),
            firstDate: DateTime(1900),
            lastDate: now,
          );
          if (picked != null) {
            onChanged(
              '${picked.year}-${picked.month.toString().padLeft(2, '0')}'
              '-${picked.day.toString().padLeft(2, '0')}',
            );
          }
        },
        child: AbsorbPointer(
          child: TextFormField(
            key: ValueKey(value),
            initialValue: value,
            decoration: InputDecoration(
              labelText: '$label *',
              suffixIcon: const Icon(Icons.calendar_today, size: 18),
            ),
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Country dropdown
// ---------------------------------------------------------------------------

class PaymentCountryDropdown extends StatelessWidget {
  const PaymentCountryDropdown({
    super.key,
    this.label,
    required this.value,
    required this.onChanged,
  });

  final String? label;
  final String value;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final displayLabel = label ?? l10n.paymentInfoNationality;

    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: DropdownButtonFormField<String>(
        initialValue: value.isEmpty ? null : value,
        decoration: InputDecoration(
          labelText: '$displayLabel *',
        ),
        items: countries
            .map(
              (c) => DropdownMenuItem(value: c.code, child: Text(c.name)),
            )
            .toList(),
        onChanged: (v) {
          if (v != null) onChanged(v);
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Role dropdown
// ---------------------------------------------------------------------------

class PaymentRoleDropdown extends StatelessWidget {
  const PaymentRoleDropdown({
    super.key,
    required this.value,
    required this.onChanged,
  });

  final BusinessRole? value;
  final ValueChanged<BusinessRole> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final items = {
      BusinessRole.owner: l10n.paymentInfoRoleOwner,
      BusinessRole.ceo: l10n.paymentInfoRoleCeo,
      BusinessRole.director: l10n.paymentInfoRoleDirector,
      BusinessRole.partner: l10n.paymentInfoRolePartner,
      BusinessRole.other: l10n.paymentInfoRoleOther,
    };

    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: DropdownButtonFormField<BusinessRole>(
        initialValue: value,
        decoration: InputDecoration(
          labelText: '${l10n.paymentInfoYourRole} *',
        ),
        items: items.entries
            .map(
              (e) => DropdownMenuItem(value: e.key, child: Text(e.value)),
            )
            .toList(),
        onChanged: (v) {
          if (v != null) onChanged(v);
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// No IBAN checkbox
// ---------------------------------------------------------------------------

class PaymentNoIbanCheckbox extends StatelessWidget {
  const PaymentNoIbanCheckbox({
    super.key,
    required this.value,
    required this.label,
    required this.onChanged,
  });

  final bool value;
  final String label;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        onTap: () => onChanged(!value),
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        child: Row(
          children: [
            Checkbox(
              value: value,
              onChanged: (v) => onChanged(v ?? false),
              activeColor: const Color(0xFFF43F5E),
              materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
              visualDensity: VisualDensity.compact,
            ),
            const SizedBox(width: 4),
            Expanded(
              child: Text(
                label,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                  color: Theme.of(context).colorScheme.onSurface,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// IBAN help text with link
// ---------------------------------------------------------------------------

class PaymentIbanHelpText extends StatelessWidget {
  const PaymentIbanHelpText({super.key, required this.helpText});

  final String helpText;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Wrap(
              children: [
                Text(
                  '$helpText ',
                  style: TextStyle(
                    fontSize: 12,
                    color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
                  ),
                ),
                GestureDetector(
                  onTap: () => launchUrl(
                    Uri.parse('https://www.iban.com/calculate-iban'),
                    mode: LaunchMode.externalApplication,
                  ),
                  child: const Text(
                    'iban.com',
                    style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                      color: Color(0xFFF43F5E),
                      decoration: TextDecoration.underline,
                      decorationColor: Color(0xFFF43F5E),
                    ),
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
