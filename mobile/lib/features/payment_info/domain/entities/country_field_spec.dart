/// Represents a single form field specification from the backend.
class FieldSpec {
  const FieldSpec({
    required this.path,
    required this.key,
    required this.type,
    required this.labelKey,
    required this.required,
    required this.isExtra,
    this.placeholder,
  });

  final String path;
  final String key;
  final String type;
  final String labelKey;
  final bool required;
  final bool isExtra;
  final String? placeholder;

  factory FieldSpec.fromJson(Map<String, dynamic> json) {
    return FieldSpec(
      path: json['path'] as String? ?? '',
      key: json['key'] as String? ?? '',
      type: json['type'] as String? ?? 'text',
      labelKey: json['label_key'] as String? ?? '',
      required: json['required'] as bool? ?? false,
      isExtra: json['is_extra'] as bool? ?? false,
      placeholder: json['placeholder'] as String?,
    );
  }
}

/// A section of related fields.
class FieldSection {
  const FieldSection({required this.id, required this.fields});

  final String id;
  final List<FieldSpec> fields;

  factory FieldSection.fromJson(Map<String, dynamic> json) {
    return FieldSection(
      id: json['id'] as String? ?? '',
      fields: (json['fields'] as List<dynamic>?)
              ?.map((e) => FieldSpec.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }
}

/// Country field requirements from the backend.
class CountryFieldsResponse {
  const CountryFieldsResponse({
    required this.country,
    required this.businessType,
    required this.sections,
    required this.individualDocRequired,
    required this.companyDocRequired,
    required this.personRoles,
  });

  final String country;
  final String businessType;
  final List<FieldSection> sections;
  final bool individualDocRequired;
  final bool companyDocRequired;
  final List<String> personRoles;

  factory CountryFieldsResponse.fromJson(Map<String, dynamic> json) {
    final docsReq = json['documents_required'] as Map<String, dynamic>? ?? {};
    return CountryFieldsResponse(
      country: json['country'] as String? ?? '',
      businessType: json['business_type'] as String? ?? 'individual',
      sections: (json['sections'] as List<dynamic>?)
              ?.map((e) => FieldSection.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      individualDocRequired: docsReq['individual'] as bool? ?? true,
      companyDocRequired: docsReq['company'] as bool? ?? false,
      personRoles: (json['person_roles'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          [],
    );
  }

  /// Returns only the extra fields from all sections.
  List<FieldSpec> get extraFields =>
      sections.expand((s) => s.fields).where((f) => f.isExtra).toList();
}
