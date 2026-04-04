import 'package:flutter/material.dart';

import '../../domain/entities/country_field_spec.dart';
import '../../domain/entities/identity_document_entity.dart';
import 'package:marketplace_mobile/features/payment_info/lib/country_states.dart';
import 'payment_info_widgets.dart';

/// Section title labels for known title keys from the backend.
const _sectionTitles = <String, String>{
  'personalInfo': 'Personal Information',
  'companyInfo': 'Company Information',
  'bankAccount': 'Bank Account',
  'representative': 'Legal Representative',
  'additionalDocuments': 'Additional Documents',
};

/// Field labels for known label keys from the backend.
const _fieldLabels = <String, String>{
  'firstName': 'First name',
  'lastName': 'Last name',
  'dob': 'Date of birth',
  'email': 'Email',
  'phone': 'Phone number',
  'addressLine1': 'Address',
  'addressCity': 'City',
  'addressPostalCode': 'Postal code',
  'addressState': 'State / Province',
  'nationality': 'Nationality',
  'companyName': 'Company name',
  'companyTaxId': 'Tax ID / EIN',
  'companyPhone': 'Company phone',
  'companyAddressLine1': 'Company address',
  'companyAddressCity': 'Company city',
  'companyAddressPostalCode': 'Company postal code',
  'companyAddressState': 'Company state',
  'companyAddressCountry': 'Company country',
  'iban': 'IBAN',
  'bic': 'BIC / SWIFT (optional)',
  'accountNumber': 'Account number',
  'routingNumber': 'Routing number',
  'accountHolderName': 'Account holder name',
  'bankCountry': 'Bank country',
  'idNumber': 'National ID Number',
  'ssnLast4': 'SSN (last 4 digits)',
  'politicalExposure': 'Political Exposure',
  'state': 'State / Province',
  'firstNameKana': 'First name (Kana)',
  'lastNameKana': 'Last name (Kana)',
  'firstNameKanji': 'First name (Kanji)',
  'lastNameKanji': 'Last name (Kanji)',
  'gender': 'Gender',
};

/// Renders a [FieldSection] dynamically, producing the correct widget
/// for each field based on its type and label key.
class DynamicSection extends StatelessWidget {
  const DynamicSection({
    super.key,
    required this.section,
    required this.values,
    required this.onChanged,
    this.fieldErrors = const {},
    this.fieldWarnings = const {},
    this.documents = const [],
    this.countryCode = '',
  });

  final FieldSection section;
  final Map<String, String> values;
  final void Function(String key, String value) onChanged;
  final Map<String, String> fieldErrors;
  final Map<String, String> fieldWarnings;
  final List<IdentityDocument> documents;
  final String countryCode;

  @override
  Widget build(BuildContext context) {
    if (section.fields.isEmpty) return const SizedBox.shrink();

    final title = _sectionTitles[section.titleKey] ??
        _humanize(section.titleKey);

    return PaymentSectionCard(
      title: title,
      children: section.fields
          .map((f) => f.type == 'document_upload'
              ? _buildDocumentField(context, f)
              : _buildField(f))
          .toList(),
    );
  }

  Widget _buildField(FieldSpec field) {
    final label = _fieldLabels[field.labelKey] ?? _humanize(field.labelKey);
    final value = values[field.key] ?? '';
    final error = fieldErrors[field.key];
    final isRequired = field.required;

    // State dropdown for countries with known states
    if (_isStateField(field) && countryCode.isNotEmpty && hasStates(countryCode)) {
      return _StateDropdown(
        label: label,
        value: value,
        countryCode: countryCode,
        onChanged: (v) => onChanged(field.key, v),
        errorText: error,
        required: isRequired,
      );
    }

    // Country/nationality select
    if (_isCountryField(field)) {
      return PaymentCountryDropdown(
        label: label,
        value: value,
        onChanged: (v) => onChanged(field.key, v),
        errorText: error,
      );
    }

    // Political exposure select
    if (field.labelKey == 'politicalExposure' || field.key == 'political_exposure') {
      return _PoliticalExposureDropdown(
        label: label,
        value: value,
        onChanged: (v) => onChanged(field.key, v),
        errorText: error,
        required: isRequired,
      );
    }

    // Gender select
    if (field.labelKey == 'gender') {
      return _GenderDropdown(
        label: label,
        value: value,
        onChanged: (v) => onChanged(field.key, v),
        errorText: error,
        required: isRequired,
      );
    }

    // Date field
    if (field.type == 'date' || field.labelKey == 'dob') {
      return PaymentDateField(
        label: '$label${isRequired ? ' *' : ''}',
        value: value,
        onChanged: (v) => onChanged(field.key, v),
      );
    }

    // Default text field
    return PaymentFormField(
      label: label,
      value: value,
      onChanged: (v) => onChanged(field.key, v),
      required: isRequired,
      errorText: error,
      placeholder: field.placeholder ?? '',
      keyboardType: _keyboardType(field),
    );
  }

  Widget _buildDocumentField(BuildContext context, FieldSpec field) {
    final label = _fieldLabels[field.labelKey] ?? _humanize(field.labelKey);
    final value = values[field.key] ?? '';
    final error = fieldErrors[field.key];
    final warning = fieldWarnings[field.key];
    final isUploaded = value == 'uploaded';

    // Find matching document for status display
    final category = field.path.startsWith('company') ||
            field.path.startsWith('documents')
        ? 'company'
        : 'identity';
    final docType = _deriveDocType(field.path);
    final matchingDoc = documents.where(
      (d) => d.category == category && d.documentType == docType,
    ).firstOrNull;

    // Suppress stale error when already uploaded
    final effectiveError = isUploaded ? null : error;

    return _DocumentStatusTile(
      label: label,
      isRequired: field.required,
      isUploaded: isUploaded,
      docStatus: matchingDoc?.status,
      rejectionReason: matchingDoc?.rejectionReason ?? '',
      error: effectiveError,
      warning: isUploaded ? null : warning,
    );
  }

  String _deriveDocType(String path) {
    if (path.contains('proof_of_liveness')) return 'proof_of_liveness';
    if (path.contains('additional_document')) return 'additional_document';
    if (path.contains('company_authorization')) return 'company_authorization';
    if (path.contains('passport')) return 'passport';
    if (path.contains('bank_account_ownership')) return 'bank_account_ownership';
    return 'document';
  }

  bool _isStateField(FieldSpec field) {
    return isStateField(field.labelKey, field.path);
  }

  bool _isCountryField(FieldSpec field) {
    final lk = field.labelKey.toLowerCase();
    return lk == 'nationality' ||
        lk == 'bankcountry' ||
        lk == 'companyaddresscountry' ||
        field.type == 'select' && field.path.endsWith('.country');
  }

  TextInputType _keyboardType(FieldSpec field) {
    if (field.type == 'phone' || field.labelKey == 'phone' || field.labelKey == 'companyPhone') {
      return TextInputType.phone;
    }
    if (field.type == 'email' || field.labelKey == 'email') {
      return TextInputType.emailAddress;
    }
    return TextInputType.text;
  }

  static String _humanize(String key) {
    // "addressPostalCode" -> "Address Postal Code"
    final result = key.replaceAllMapped(
      RegExp(r'([a-z])([A-Z])'),
      (m) => '${m.group(1)} ${m.group(2)}',
    );
    if (result.isEmpty) return key;
    return result[0].toUpperCase() + result.substring(1);
  }
}

// ---------------------------------------------------------------------------
// Specific dropdown widgets
// ---------------------------------------------------------------------------

class _StateDropdown extends StatelessWidget {
  const _StateDropdown({
    required this.label,
    required this.value,
    required this.countryCode,
    required this.onChanged,
    this.errorText,
    this.required = false,
  });

  final String label;
  final String value;
  final String countryCode;
  final ValueChanged<String> onChanged;
  final String? errorText;
  final bool required;

  @override
  Widget build(BuildContext context) {
    final states = getStatesForCountry(countryCode);
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: DropdownButtonFormField<String>(
        initialValue: value.isEmpty ? null : value,
        decoration: InputDecoration(
          labelText: '$label${required ? ' *' : ''}',
          border: const OutlineInputBorder(),
          errorText: errorText,
        ),
        isExpanded: true,
        items: states
            .map((s) => DropdownMenuItem(
                  value: s.code,
                  child: Text(s.name, softWrap: true),
                ))
            .toList(),
        onChanged: (v) {
          if (v != null) onChanged(v);
        },
      ),
    );
  }
}

class _PoliticalExposureDropdown extends StatelessWidget {
  const _PoliticalExposureDropdown({
    required this.label,
    required this.value,
    required this.onChanged,
    this.errorText,
    this.required = false,
  });

  final String label;
  final String value;
  final ValueChanged<String> onChanged;
  final String? errorText;
  final bool required;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: DropdownButtonFormField<String>(
        initialValue: value.isEmpty ? null : value,
        decoration: InputDecoration(
          labelText: '$label${required ? ' *' : ''}',
          border: const OutlineInputBorder(),
          errorText: errorText,
        ),
        items: const [
          DropdownMenuItem(value: 'none', child: Text('None')),
          DropdownMenuItem(value: 'existing', child: Text('Existing')),
        ],
        onChanged: (v) {
          if (v != null) onChanged(v);
        },
      ),
    );
  }
}

class _GenderDropdown extends StatelessWidget {
  const _GenderDropdown({
    required this.label,
    required this.value,
    required this.onChanged,
    this.errorText,
    this.required = false,
  });

  final String label;
  final String value;
  final ValueChanged<String> onChanged;
  final String? errorText;
  final bool required;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: DropdownButtonFormField<String>(
        initialValue: value.isEmpty ? null : value,
        decoration: InputDecoration(
          labelText: '$label${required ? ' *' : ''}',
          border: const OutlineInputBorder(),
          errorText: errorText,
        ),
        items: const [
          DropdownMenuItem(value: 'male', child: Text('Male')),
          DropdownMenuItem(value: 'female', child: Text('Female')),
        ],
        onChanged: (v) {
          if (v != null) onChanged(v);
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Document status tile (inline display for document_upload fields)
// ---------------------------------------------------------------------------

class _DocumentStatusTile extends StatelessWidget {
  const _DocumentStatusTile({
    required this.label,
    required this.isRequired,
    required this.isUploaded,
    this.docStatus,
    this.rejectionReason = '',
    this.error,
    this.warning,
  });

  final String label;
  final bool isRequired;
  final bool isUploaded;
  final String? docStatus;
  final String rejectionReason;
  final String? error;
  final String? warning;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    if (isUploaded) {
      return _buildStatusContainer(context, theme);
    }

    return _buildEmptyContainer(context, theme);
  }

  Widget _buildStatusContainer(BuildContext context, ThemeData theme) {
    if (docStatus == 'rejected') {
      return _StatusBanner(
        label: label,
        statusText: rejectionReason.isNotEmpty
            ? 'Rejected: $rejectionReason'
            : 'Document rejected — please re-upload',
        icon: Icons.cancel_outlined,
        bgColor: theme.colorScheme.error.withValues(alpha: 0.08),
        borderColor: theme.colorScheme.error.withValues(alpha: 0.3),
        textColor: theme.colorScheme.error,
      );
    }

    if (docStatus == 'verified') {
      return _StatusBanner(
        label: label,
        statusText: 'Document verified',
        icon: Icons.check_circle_outline,
        bgColor: const Color(0xFF22C55E).withValues(alpha: 0.08),
        borderColor: const Color(0xFF22C55E).withValues(alpha: 0.3),
        textColor: const Color(0xFF16A34A),
      );
    }

    // Pending or unknown
    return _StatusBanner(
      label: label,
      statusText: 'Uploaded — pending verification',
      icon: Icons.check_circle_outline,
      bgColor: const Color(0xFF22C55E).withValues(alpha: 0.08),
      borderColor: const Color(0xFF22C55E).withValues(alpha: 0.3),
      textColor: const Color(0xFF16A34A),
    );
  }

  Widget _buildEmptyContainer(BuildContext context, ThemeData theme) {
    final hasError = error != null;
    final hasWarning = warning != null;

    final borderColor = hasError
        ? theme.colorScheme.error.withValues(alpha: 0.5)
        : hasWarning
            ? const Color(0xFFF59E0B).withValues(alpha: 0.5)
            : theme.dividerColor;
    final bgColor = hasError
        ? theme.colorScheme.error.withValues(alpha: 0.04)
        : hasWarning
            ? const Color(0xFFF59E0B).withValues(alpha: 0.04)
            : Colors.transparent;

    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: bgColor,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: borderColor),
            ),
            child: Column(
              children: [
                Icon(Icons.upload_file, size: 32, color: theme.hintColor),
                const SizedBox(height: 8),
                Text(
                  '$label${isRequired ? ' *' : ''}',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w500,
                  ),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 4),
                Text(
                  'Upload via Identity Verification section below',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.hintColor,
                  ),
                  textAlign: TextAlign.center,
                ),
              ],
            ),
          ),
          if (hasError)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                error!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
            ),
          if (!hasError && hasWarning)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                warning!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: const Color(0xFFD97706),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _StatusBanner extends StatelessWidget {
  const _StatusBanner({
    required this.label,
    required this.statusText,
    required this.icon,
    required this.bgColor,
    required this.borderColor,
    required this.textColor,
  });

  final String label;
  final String statusText;
  final IconData icon;
  final Color bgColor;
  final Color borderColor;
  final Color textColor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: bgColor,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: borderColor),
        ),
        child: Row(
          children: [
            Icon(icon, size: 20, color: textColor),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    label,
                    style: theme.textTheme.bodySmall?.copyWith(
                      fontWeight: FontWeight.w600,
                      color: textColor,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    statusText,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: textColor.withValues(alpha: 0.8),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
