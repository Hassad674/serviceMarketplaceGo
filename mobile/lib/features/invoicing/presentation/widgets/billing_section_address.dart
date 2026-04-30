import 'package:flutter/material.dart';

import 'billing_form_atoms.dart';

/// "Adresse" section — line 1 (required), optional line 2, postal code
/// + city on a single row.
class BillingAddressSection extends StatelessWidget {
  const BillingAddressSection({
    super.key,
    required this.addressLine1,
    required this.addressLine2,
    required this.postalCode,
    required this.city,
  });

  final TextEditingController addressLine1;
  final TextEditingController addressLine2;
  final TextEditingController postalCode;
  final TextEditingController city;

  @override
  Widget build(BuildContext context) {
    return BillingSection(
      title: 'Adresse',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          BillingLabeledField(
            label: 'Adresse',
            controller: addressLine1,
            validator: billingRequiredValidator,
          ),
          const SizedBox(height: 12),
          BillingLabeledField(
            label: "Complément d'adresse (optionnel)",
            controller: addressLine2,
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: BillingLabeledField(
                  label: 'Code postal',
                  controller: postalCode,
                  validator: billingRequiredValidator,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: BillingLabeledField(
                  label: 'Ville',
                  controller: city,
                  validator: billingRequiredValidator,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
