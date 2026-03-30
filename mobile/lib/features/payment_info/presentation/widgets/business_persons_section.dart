import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/payment_info.dart';
import 'payment_info_widgets.dart';

// ---------------------------------------------------------------------------
// Activity sector section
// ---------------------------------------------------------------------------

/// Section card wrapping the activity sector dropdown.
class ActivitySectorSection extends StatelessWidget {
  const ActivitySectorSection({
    super.key,
    required this.data,
    required this.onUpdate,
  });

  final PaymentInfoFormData data;
  final FormUpdater onUpdate;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return PaymentSectionCard(
      title: l10n.paymentInfoActivitySector,
      children: [
        PaymentActivitySectorDropdown(
          value: data.activitySector,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(activitySector: v)),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Business persons section (KYC flags + expandable person forms)
// ---------------------------------------------------------------------------

/// Business KYC section with 4 boolean checkboxes and expandable
/// person forms when any checkbox is unchecked.
class BusinessPersonsSection extends StatelessWidget {
  const BusinessPersonsSection({
    super.key,
    required this.data,
    required this.onUpdate,
  });

  final PaymentInfoFormData data;
  final FormUpdater onUpdate;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return PaymentSectionCard(
      title: l10n.paymentInfoBusinessPersons,
      children: [
        _KycCheckbox(
          label: l10n.paymentInfoSelfRepresentative,
          value: data.isSelfRepresentative,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(isSelfRepresentative: v)),
        ),
        _KycCheckbox(
          label: l10n.paymentInfoSelfDirector,
          value: data.isSelfDirector,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(isSelfDirector: v)),
        ),
        _KycCheckbox(
          label: l10n.paymentInfoNoMajorOwners,
          value: data.noMajorOwners,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(noMajorOwners: v)),
        ),
        _KycCheckbox(
          label: l10n.paymentInfoSelfExecutive,
          value: data.isSelfExecutive,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(isSelfExecutive: v)),
        ),
        if (_needsPersonForms) ...[
          const SizedBox(height: 12),
          _BusinessPersonsList(data: data, onUpdate: onUpdate),
        ],
      ],
    );
  }

  bool get _needsPersonForms {
    return !data.isSelfDirector ||
        !data.noMajorOwners ||
        !data.isSelfExecutive;
  }
}

// ---------------------------------------------------------------------------
// KYC checkbox
// ---------------------------------------------------------------------------

class _KycCheckbox extends StatelessWidget {
  const _KycCheckbox({
    required this.label,
    required this.value,
    required this.onChanged,
  });

  final String label;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
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
// Business persons list with add/remove
// ---------------------------------------------------------------------------

class _BusinessPersonsList extends StatelessWidget {
  const _BusinessPersonsList({
    required this.data,
    required this.onUpdate,
  });

  final PaymentInfoFormData data;
  final FormUpdater onUpdate;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final persons = data.businessPersons;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.paymentInfoAddPerson,
          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                fontWeight: FontWeight.w500,
              ),
        ),
        const SizedBox(height: 8),
        for (int i = 0; i < persons.length; i++)
          _BusinessPersonCard(
            index: i,
            person: persons[i],
            onUpdate: (updated) {
              final list = List<BusinessPerson>.from(persons);
              list[i] = updated;
              onUpdate((d) => d.copyWith(businessPersons: list));
            },
            onRemove: () {
              final list = List<BusinessPerson>.from(persons);
              list.removeAt(i);
              onUpdate((d) => d.copyWith(businessPersons: list));
            },
          ),
        const SizedBox(height: 8),
        OutlinedButton.icon(
          onPressed: () {
            final list = List<BusinessPerson>.from(persons)
              ..add(const BusinessPerson());
            onUpdate((d) => d.copyWith(businessPersons: list));
          },
          icon: const Icon(Icons.add, size: 18),
          label: Text(l10n.paymentInfoAddPerson),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Individual business person card
// ---------------------------------------------------------------------------

class _BusinessPersonCard extends StatelessWidget {
  const _BusinessPersonCard({
    required this.index,
    required this.person,
    required this.onUpdate,
    required this.onRemove,
  });

  final int index;
  final BusinessPerson person;
  final ValueChanged<BusinessPerson> onUpdate;
  final VoidCallback onRemove;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Text(
                '${l10n.paymentInfoPerson} ${index + 1}',
                style: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.close, size: 18),
                onPressed: onRemove,
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(),
              ),
            ],
          ),
          const SizedBox(height: 8),
          PaymentFormField(
            label: l10n.paymentInfoFirstName,
            value: person.firstName,
            onChanged: (v) =>
                onUpdate(person.copyWith(firstName: v)),
          ),
          PaymentFormField(
            label: l10n.paymentInfoLastName,
            value: person.lastName,
            onChanged: (v) =>
                onUpdate(person.copyWith(lastName: v)),
          ),
          PaymentFormField(
            label: l10n.email,
            value: person.email,
            onChanged: (v) =>
                onUpdate(person.copyWith(email: v)),
            keyboardType: TextInputType.emailAddress,
          ),
          PaymentFormField(
            label: l10n.paymentInfoPhone,
            value: person.phone,
            onChanged: (v) =>
                onUpdate(person.copyWith(phone: v)),
            keyboardType: TextInputType.phone,
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Save button
// ---------------------------------------------------------------------------

/// Extracted save button for the payment info form.
class PaymentInfoSaveButton extends StatelessWidget {
  const PaymentInfoSaveButton({
    super.key,
    required this.isValid,
    required this.isSaving,
    required this.onSave,
  });

  final bool isValid;
  final bool isSaving;
  final VoidCallback onSave;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return SizedBox(
      width: double.infinity,
      height: 48,
      child: ElevatedButton(
        onPressed: isValid && !isSaving ? onSave : null,
        style: ElevatedButton.styleFrom(
          backgroundColor: const Color(0xFFF43F5E),
          foregroundColor: Colors.white,
          disabledBackgroundColor:
              theme.colorScheme.onSurface.withValues(alpha: 0.12),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          ),
        ),
        child: isSaving
            ? const SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  valueColor:
                      AlwaysStoppedAnimation<Color>(Colors.white),
                ),
              )
            : Text(
                l10n.paymentInfoSave,
                style: const TextStyle(
                  fontWeight: FontWeight.w600,
                  fontSize: 15,
                ),
              ),
      ),
    );
  }
}
