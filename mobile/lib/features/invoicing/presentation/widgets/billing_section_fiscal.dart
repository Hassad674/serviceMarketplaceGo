import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/data/stripe_countries.dart';
import '../../../../core/theme/app_theme.dart';
import 'billing_form_atoms.dart';

/// Soleil v2 "Pays" dropdown — every Stripe-supported country grouped
/// by region. Region headers render as disabled visual separators
/// (Flutter has no native optgroup concept on
/// `DropdownButtonFormField`).
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
    final colorScheme = theme.colorScheme;
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
            style: SoleilTextStyles.mono.copyWith(
              color: colorScheme.primary,
              fontSize: 10.5,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.4,
            ),
          ),
        ),
      );
      for (final c in entries) {
        items.add(
          DropdownMenuItem<String>(
            value: c.code,
            child: Text(
              '${c.flag}  ${c.labelFr}',
              style: SoleilTextStyles.body.copyWith(
                color: colorScheme.onSurface,
              ),
            ),
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
      decoration: const InputDecoration(
        isDense: true,
        hintText: '— Sélectionne ton pays —',
      ),
      items: _buildItems(context),
      onChanged: onChanged,
      validator: (v) =>
          v == null || v.isEmpty ? 'Champ obligatoire' : null,
    );
  }
}

/// Soleil v2 "Numéro de TVA intracommunautaire" row — labelled input,
/// validate pill button, optional sapin success line ("Validé le ...")
/// and corail-deep error chip.
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
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    final success = appColors?.success ?? colorScheme.primary;
    final canTap = !validating && controller.text.trim().isNotEmpty;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        BillingLabeledField(
          label: 'Numéro de TVA intracommunautaire',
          controller: controller,
          hint: 'FR12345678901',
        ),
        const SizedBox(height: 10),
        Align(
          alignment: Alignment.centerLeft,
          child: OutlinedButton.icon(
            onPressed: canTap ? onValidate : null,
            style: OutlinedButton.styleFrom(
              minimumSize: const Size(0, 38),
              padding:
                  const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
              foregroundColor: colorScheme.onSurface,
              side: BorderSide(
                color: appColors?.borderStrong ?? theme.dividerColor,
              ),
              shape: const StadiumBorder(),
              textStyle: SoleilTextStyles.button.copyWith(fontSize: 12.5),
            ),
            icon: validating
                ? SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: colorScheme.primary,
                    ),
                  )
                : Icon(
                    Icons.check_circle_outline,
                    size: 14,
                    color: colorScheme.primary,
                  ),
            label: const Text('Valider mon n° TVA'),
          ),
        ),
        if (validatedAt != null && error == null) ...[
          const SizedBox(height: 8),
          Row(
            children: [
              Icon(Icons.check_circle, size: 14, color: success),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  registeredName != null && registeredName!.isNotEmpty
                      ? '$registeredName · validé le '
                          '${DateFormat('dd/MM/yyyy').format(validatedAt!)}'
                      : 'Validé le ${DateFormat('dd/MM/yyyy').format(validatedAt!)}',
                  style: SoleilTextStyles.caption.copyWith(color: success),
                ),
              ),
            ],
          ),
        ],
        if (error != null) ...[
          const SizedBox(height: 8),
          Row(
            children: [
              Icon(Icons.cancel, size: 14, color: colorScheme.error),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  error!,
                  style: SoleilTextStyles.caption.copyWith(
                    color: colorScheme.error,
                  ),
                ),
              ),
            ],
          ),
        ],
      ],
    );
  }
}

/// Soleil v2 "Identifiants fiscaux" section — SIRET (FR) or generic
/// tax id (other), optional VAT row when the country is in the EU.
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
            const SizedBox(height: 16),
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
