import 'package:flutter/material.dart';

import '../../domain/entities/billing_profile.dart';
import 'billing_form_atoms.dart';

/// "Type de profil" radio (individual vs business).
class BillingProfileTypeRadio extends StatelessWidget {
  const BillingProfileTypeRadio({
    super.key,
    required this.value,
    required this.onChanged,
  });

  final ProfileType? value;
  final ValueChanged<ProfileType> onChanged;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        BillingRadioTile(
          label: 'Particulier',
          selected: value == ProfileType.individual,
          onTap: () => onChanged(ProfileType.individual),
        ),
        const SizedBox(height: 8),
        BillingRadioTile(
          label: 'Entreprise',
          selected: value == ProfileType.business,
          onTap: () => onChanged(ProfileType.business),
        ),
      ],
    );
  }
}

/// "Identité légale" section — legal name (always required) plus the
/// trading name and legal form (only shown when [isBusiness] is true).
class BillingLegalIdentitySection extends StatelessWidget {
  const BillingLegalIdentitySection({
    super.key,
    required this.isBusiness,
    required this.legalName,
    required this.tradingName,
    required this.legalForm,
  });

  final bool isBusiness;
  final TextEditingController legalName;
  final TextEditingController tradingName;
  final TextEditingController legalForm;

  @override
  Widget build(BuildContext context) {
    return BillingSection(
      title: 'Identité légale',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          BillingLabeledField(
            label: 'Raison sociale ou nom légal',
            controller: legalName,
            validator: billingRequiredValidator,
          ),
          if (isBusiness) ...[
            const SizedBox(height: 12),
            BillingLabeledField(
              label: 'Nom commercial (optionnel)',
              controller: tradingName,
            ),
            const SizedBox(height: 12),
            BillingLabeledField(
              label: 'Forme juridique',
              controller: legalForm,
              hint: 'SAS, SARL, EURL, etc.',
            ),
          ],
        ],
      ),
    );
  }
}
