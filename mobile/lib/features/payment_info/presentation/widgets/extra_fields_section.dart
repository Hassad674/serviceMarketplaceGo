import 'package:flutter/material.dart';

import '../../domain/entities/country_field_spec.dart';

/// Label map for known extra field keys.
const _fieldLabels = <String, String>{
  'id_number': 'National ID Number',
  'ssn_last_4': 'SSN (last 4 digits)',
  'state': 'State / Province',
  'political_exposure': 'Political Exposure',
  'first_name_kana': 'First Name (Kana)',
  'last_name_kana': 'Last Name (Kana)',
  'first_name_kanji': 'First Name (Kanji)',
  'last_name_kanji': 'Last Name (Kanji)',
};

/// Renders country-specific extra fields dynamically.
class ExtraFieldsSection extends StatelessWidget {
  const ExtraFieldsSection({
    super.key,
    required this.fields,
    required this.values,
    required this.onChanged,
  });

  final List<FieldSpec> fields;
  final Map<String, String> values;
  final void Function(String key, String value) onChanged;

  @override
  Widget build(BuildContext context) {
    if (fields.isEmpty) return const SizedBox.shrink();

    final theme = Theme.of(context);

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.dividerColor),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.settings, color: Colors.amber.shade700),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Country-Specific Requirements',
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        'Additional fields required for your country.',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            ...fields.map((f) => _buildField(f)),
          ],
        ),
      ),
    );
  }

  Widget _buildField(FieldSpec field) {
    final label = _fieldLabels[field.key] ?? field.key;
    final value = values[field.key] ?? '';

    if (field.key == 'political_exposure') {
      return Padding(
        padding: const EdgeInsets.only(bottom: 12),
        child: DropdownButtonFormField<String>(
          initialValue: value.isEmpty ? null : value,
          decoration: InputDecoration(
            labelText: label,
            border: const OutlineInputBorder(),
          ),
          items: const [
            DropdownMenuItem(value: 'none', child: Text('None')),
            DropdownMenuItem(value: 'existing', child: Text('Existing')),
          ],
          onChanged: (v) {
            if (v != null) onChanged(field.key, v);
          },
        ),
      );
    }

    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextFormField(
        initialValue: value,
        decoration: InputDecoration(
          labelText: label,
          hintText: field.placeholder ?? '',
          border: const OutlineInputBorder(),
        ),
        onChanged: (v) => onChanged(field.key, v),
      ),
    );
  }
}
