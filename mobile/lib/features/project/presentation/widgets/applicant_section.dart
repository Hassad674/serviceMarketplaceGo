// ignore_for_file: deprecated_member_use
import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../../types/project.dart';

/// Section 5: Who can apply + negotiable checkbox.
class ApplicantSection extends StatelessWidget {
  const ApplicantSection({
    super.key,
    required this.applicantType,
    required this.onApplicantTypeChanged,
    required this.negotiable,
    required this.onNegotiableChanged,
  });

  final ApplicantType applicantType;
  final ValueChanged<ApplicantType> onApplicantTypeChanged;
  final bool negotiable;
  final ValueChanged<bool> onNegotiableChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.whoCanApply, style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        _ApplicantRadio(
          value: ApplicantType.freelancersAndAgencies,
          groupValue: applicantType,
          label: l10n.freelancersAndAgencies,
          onChanged: onApplicantTypeChanged,
        ),
        _ApplicantRadio(
          value: ApplicantType.freelancersOnly,
          groupValue: applicantType,
          label: l10n.freelancersOnly,
          onChanged: onApplicantTypeChanged,
        ),
        _ApplicantRadio(
          value: ApplicantType.agenciesOnly,
          groupValue: applicantType,
          label: l10n.agenciesOnly,
          onChanged: onApplicantTypeChanged,
        ),
        const SizedBox(height: 8),
        CheckboxListTile(
          value: negotiable,
          onChanged: (value) => onNegotiableChanged(value ?? false),
          title: Text(
            l10n.negotiable,
            style: theme.textTheme.bodyMedium,
          ),
          controlAffinity: ListTileControlAffinity.leading,
          contentPadding: EdgeInsets.zero,
          activeColor: theme.colorScheme.primary,
        ),
      ],
    );
  }
}

class _ApplicantRadio extends StatelessWidget {
  const _ApplicantRadio({
    required this.value,
    required this.groupValue,
    required this.label,
    required this.onChanged,
  });

  final ApplicantType value;
  final ApplicantType groupValue;
  final String label;
  final ValueChanged<ApplicantType> onChanged;

  @override
  Widget build(BuildContext context) {
    return RadioListTile<ApplicantType>(
      value: value,
      groupValue: groupValue,
      onChanged: (v) {
        if (v != null) onChanged(v);
      },
      title: Text(
        label,
        style: Theme.of(context).textTheme.bodyMedium,
      ),
      contentPadding: EdgeInsets.zero,
      activeColor: Theme.of(context).colorScheme.primary,
    );
  }
}
