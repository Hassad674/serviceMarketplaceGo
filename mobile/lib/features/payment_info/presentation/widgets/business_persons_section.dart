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
// Business persons section (KYC flags + per-role expandable person forms)
// ---------------------------------------------------------------------------

/// Business KYC section with 4 boolean checkboxes. When a checkbox is
/// unchecked, a role-specific "Add person" section appears below it,
/// matching the web UX exactly.
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
        // 1. Representative
        _KycCheckbox(
          label: l10n.paymentInfoSelfRepresentative,
          value: data.isSelfRepresentative,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(isSelfRepresentative: v)),
        ),
        if (!data.isSelfRepresentative)
          _RolePersonList(
            role: 'representative',
            label: l10n.paymentInfoRepresentative,
            addLabel: l10n.paymentInfoAddRepresentative,
            data: data,
            onUpdate: onUpdate,
          ),

        // 2. Director
        _KycCheckbox(
          label: l10n.paymentInfoSelfDirector,
          value: data.isSelfDirector,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(isSelfDirector: v)),
        ),
        if (!data.isSelfDirector)
          _RolePersonList(
            role: 'director',
            label: l10n.paymentInfoDirectorLabel,
            addLabel: l10n.paymentInfoAddDirector,
            data: data,
            onUpdate: onUpdate,
          ),

        // 3. Owners (shareholders >25%)
        _KycCheckbox(
          label: l10n.paymentInfoNoMajorOwners,
          value: data.noMajorOwners,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(noMajorOwners: v)),
        ),
        if (!data.noMajorOwners)
          _RolePersonList(
            role: 'owner',
            label: l10n.paymentInfoOwnerLabel,
            addLabel: l10n.paymentInfoAddOwner,
            data: data,
            onUpdate: onUpdate,
          ),

        // 4. Executive
        _KycCheckbox(
          label: l10n.paymentInfoSelfExecutive,
          value: data.isSelfExecutive,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(isSelfExecutive: v)),
        ),
        if (!data.isSelfExecutive)
          _RolePersonList(
            role: 'executive',
            label: l10n.paymentInfoExecutiveLabel,
            addLabel: l10n.paymentInfoAddExecutive,
            data: data,
            onUpdate: onUpdate,
          ),
      ],
    );
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
// Role-specific person list
// ---------------------------------------------------------------------------

class _RolePersonList extends StatelessWidget {
  const _RolePersonList({
    required this.role,
    required this.label,
    required this.addLabel,
    required this.data,
    required this.onUpdate,
  });

  final String role;
  final String label;
  final String addLabel;
  final PaymentInfoFormData data;
  final FormUpdater onUpdate;

  @override
  Widget build(BuildContext context) {
    final persons = data.businessPersons;
    final rolePersons = <int>[];
    for (int i = 0; i < persons.length; i++) {
      if (persons[i].role == role) rolePersons.add(i);
    }

    return Padding(
      padding: const EdgeInsets.only(left: 12, bottom: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  fontWeight: FontWeight.w500,
                  color: Theme.of(context)
                      .colorScheme
                      .onSurface
                      .withValues(alpha: 0.6),
                ),
          ),
          const SizedBox(height: 8),
          for (int i = 0; i < rolePersons.length; i++)
            _BusinessPersonCard(
              roleLabel: '$label #${i + 1}',
              person: persons[rolePersons[i]],
              role: role,
              onUpdate: (updated) {
                final list = List<BusinessPerson>.from(persons);
                list[rolePersons[i]] = updated;
                onUpdate((d) => d.copyWith(businessPersons: list));
              },
              onRemove: () {
                final list = List<BusinessPerson>.from(persons);
                list.removeAt(rolePersons[i]);
                onUpdate((d) => d.copyWith(businessPersons: list));
              },
            ),
          const SizedBox(height: 4),
          OutlinedButton.icon(
            onPressed: () {
              final list = List<BusinessPerson>.from(persons)
                ..add(BusinessPerson(role: role));
              onUpdate((d) => d.copyWith(businessPersons: list));
            },
            icon: const Icon(Icons.add, size: 18),
            label: Text(addLabel),
            style: OutlinedButton.styleFrom(
              foregroundColor: const Color(0xFFF43F5E),
              side: const BorderSide(color: Color(0xFFF43F5E)),
              textStyle: const TextStyle(fontSize: 13),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Individual business person card with role-specific fields
// ---------------------------------------------------------------------------

class _BusinessPersonCard extends StatelessWidget {
  const _BusinessPersonCard({
    required this.roleLabel,
    required this.person,
    required this.role,
    required this.onUpdate,
    required this.onRemove,
  });

  final String roleLabel;
  final BusinessPerson person;
  final String role;
  final ValueChanged<BusinessPerson> onUpdate;
  final VoidCallback onRemove;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    final hasPhone = role == 'representative' ||
        role == 'owner' ||
        role == 'executive';
    final hasAddress = role == 'representative' ||
        role == 'owner' ||
        role == 'executive';
    final hasTitle = role == 'representative' || role == 'director';

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
                roleLabel,
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
          // Common fields: firstName, lastName, dateOfBirth, email
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
          PaymentDateField(
            label: l10n.paymentInfoDateOfBirth,
            value: person.dateOfBirth,
            onChanged: (v) =>
                onUpdate(person.copyWith(dateOfBirth: v)),
          ),
          PaymentFormField(
            label: l10n.email,
            value: person.email,
            onChanged: (v) =>
                onUpdate(person.copyWith(email: v)),
            keyboardType: TextInputType.emailAddress,
          ),
          // Role-specific fields
          if (hasPhone)
            PaymentFormField(
              label: l10n.paymentInfoPhone,
              value: person.phone,
              onChanged: (v) =>
                  onUpdate(person.copyWith(phone: v)),
              keyboardType: TextInputType.phone,
            ),
          if (hasAddress) ...[
            PaymentFormField(
              label: l10n.paymentInfoAddress,
              value: person.address,
              onChanged: (v) =>
                  onUpdate(person.copyWith(address: v)),
            ),
            PaymentFormField(
              label: l10n.paymentInfoCity,
              value: person.city,
              onChanged: (v) =>
                  onUpdate(person.copyWith(city: v)),
            ),
            PaymentFormField(
              label: l10n.paymentInfoPostalCode,
              value: person.postalCode,
              onChanged: (v) =>
                  onUpdate(person.copyWith(postalCode: v)),
            ),
          ],
          if (hasTitle)
            PaymentFormField(
              label: l10n.paymentInfoPersonTitle,
              value: person.title,
              onChanged: (v) =>
                  onUpdate(person.copyWith(title: v)),
              placeholder: 'CEO, Director...',
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
