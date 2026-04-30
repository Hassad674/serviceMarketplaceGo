import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/data/stripe_countries.dart';
import '../../../../core/theme/app_theme.dart';
import 'billing_form_atoms.dart';

/// "Pays" dropdown — every Stripe-supported country grouped by region.
/// Region headers render as disabled visual separators (Flutter has no
/// native optgroup concept on `DropdownButtonFormField`).
class BillingCountryDropdown extends StatelessWidget {
  const BillingCountryDropdown({
    super.key,
    required this.value,
    required this.onChanged,
  });

  final String value;
  final ValueChanged<String?> onChanged;

  List<DropdownMenuItem<String>> _buildItems(BuildContext context) {
    final theme = Theme.of(context);
    final items = <DropdownMenuItem<String>>[];
    for (final region in kStripeRegionOrder) {
      final entries = stripeCountriesByRegion(region);
      if (entries.isEmpty) continue;
      items.add(
        DropdownMenuItem<String>(
          enabled: false,
          value: '__header_${region.name}',
          child: Text(
            kStripeRegionLabelsFr[region]!,
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
              fontWeight: FontWeight.w600,
              letterSpacing: 0.5,
            ),
          ),
        ),
      );
      for (final c in entries) {
        items.add(
          DropdownMenuItem<String>(
            value: c.code,
            child: Text('${c.flag}  ${c.labelFr}'),
          ),
        );
      }
    }
    return items;
  }

  @override
  Widget build(BuildContext context) {
    return DropdownButtonFormField<String>(
      initialValue: value.isEmpty ? null : value,
      isExpanded: true,
      decoration: InputDecoration(
        isDense: true,
        hintText: '— Sélectionne ton pays —',
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        ),
      ),
      items: _buildItems(context),
      onChanged: onChanged,
      validator: (v) =>
          v == null || v.isEmpty ? 'Champ obligatoire' : null,
    );
  }
}

/// "Numéro de TVA intracommunautaire" row — input + validate button +
/// optional success line ("Validé le ... ") and error chip.
class BillingVatRow extends StatelessWidget {
  const BillingVatRow({
    super.key,
    required this.controller,
    required this.validatedAt,
    required this.registeredName,
    required this.validating,
    required this.error,
    required this.onValidate,
  });

  final TextEditingController controller;
  final DateTime? validatedAt;
  final String? registeredName;
  final bool validating;
  final String? error;
  final VoidCallback onValidate;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        BillingLabeledField(
          label: 'Numéro de TVA intracommunautaire',
          controller: controller,
          hint: 'FR12345678901',
        ),
        const SizedBox(height: 8),
        Align(
          alignment: Alignment.centerLeft,
          child: OutlinedButton.icon(
            onPressed: validating || controller.text.trim().isEmpty
                ? null
                : onValidate,
            icon: validating
                ? const SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.check_circle_outline, size: 16),
            label: const Text('Valider mon n° TVA'),
          ),
        ),
        if (validatedAt != null && error == null) ...[
          const SizedBox(height: 6),
          Row(
            children: [
              const Icon(
                Icons.check_circle,
                size: 14,
                color: Color(0xFF22C55E),
              ),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  registeredName != null && registeredName!.isNotEmpty
                      ? '$registeredName · validé le '
                          '${DateFormat('dd/MM/yyyy').format(validatedAt!)}'
                      : 'Validé le ${DateFormat('dd/MM/yyyy').format(validatedAt!)}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: const Color(0xFF15803D),
                  ),
                ),
              ),
            ],
          ),
        ],
        if (error != null) ...[
          const SizedBox(height: 6),
          Row(
            children: [
              Icon(
                Icons.cancel,
                size: 14,
                color: theme.colorScheme.error,
              ),
              const SizedBox(width: 6),
              Text(
                error!,
                style: TextStyle(
                  color: theme.colorScheme.error,
                  fontSize: 12,
                ),
              ),
            ],
          ),
        ],
      ],
    );
  }
}

/// "Identifiants fiscaux" section — SIRET (FR) or generic tax id (other),
/// optional VAT row when the country is in the EU.
class BillingFiscalSection extends StatelessWidget {
  const BillingFiscalSection({
    super.key,
    required this.isFr,
    required this.isEu,
    required this.taxId,
    required this.vatNumber,
    required this.vatValidatedAt,
    required this.vatRegisteredName,
    required this.validatingVat,
    required this.vatError,
    required this.onValidateVat,
  });

  final bool isFr;
  final bool isEu;
  final TextEditingController taxId;
  final TextEditingController vatNumber;
  final DateTime? vatValidatedAt;
  final String? vatRegisteredName;
  final bool validatingVat;
  final String? vatError;
  final VoidCallback onValidateVat;

  @override
  Widget build(BuildContext context) {
    return BillingSection(
      title: 'Identifiants fiscaux',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (isFr)
            BillingLabeledField(
              label: 'Numéro SIRET',
              controller: taxId,
              hint: '14 chiffres, sans espace',
              keyboardType: TextInputType.number,
              maxLength: 14,
              validator: billingSiretValidator,
            )
          else
            BillingLabeledField(
              label: 'Identifiant fiscal',
              controller: taxId,
            ),
          if (isEu) ...[
            const SizedBox(height: 12),
            BillingVatRow(
              controller: vatNumber,
              validatedAt: vatValidatedAt,
              registeredName: vatRegisteredName,
              validating: validatingVat,
              error: vatError,
              onValidate: onValidateVat,
            ),
          ],
        ],
      ),
    );
  }
}
