// SearchDocument — mobile mirror of the web
// `shared/lib/search/search-document.ts` contract. Keeping the two
// shapes aligned makes the future Typesense swap a mobile-side one-
// file change (only this adapter needs to learn the new payload).
//
// The shape is intentionally frozen. Every monetary amount is in the
// smallest currency unit (centimes / basis points) — matching the
// backend `profile_pricing.min_amount` and the aggregate that the
// search endpoint computes from `proposal_milestones.amount`.

enum SearchDocumentPersona { freelance, agency, referrer }

enum SearchDocumentAvailability {
  availableNow,
  availableSoon,
  notAvailable,
}

enum SearchDocumentPricingType {
  daily,
  hourly,
  projectFrom,
  projectRange,
  commissionPct,
  commissionFlat,
}

class SearchDocumentPricing {
  const SearchDocumentPricing({
    required this.type,
    required this.minAmount,
    required this.maxAmount,
    required this.currency,
    required this.negotiable,
  });

  final SearchDocumentPricingType type;
  final int minAmount;
  final int? maxAmount;
  final String currency;
  final bool negotiable;
}

class SearchDocumentRating {
  const SearchDocumentRating({required this.average, required this.count});

  final double average;
  final int count;
}

class SearchDocument {
  const SearchDocument({
    required this.id,
    required this.persona,
    required this.displayName,
    required this.title,
    required this.photoUrl,
    required this.city,
    required this.countryCode,
    required this.languagesProfessional,
    required this.availabilityStatus,
    required this.expertiseDomains,
    required this.skills,
    required this.pricing,
    required this.rating,
    required this.totalEarned,
    required this.completedProjects,
    required this.createdAt,
  });

  final String id;
  final SearchDocumentPersona persona;
  final String displayName;
  final String title;
  final String photoUrl;
  final String city;
  final String countryCode;
  final List<String> languagesProfessional;
  final SearchDocumentAvailability availabilityStatus;
  final List<String> expertiseDomains;
  final List<String> skills;
  final SearchDocumentPricing? pricing;
  final SearchDocumentRating rating;
  final int totalEarned;
  final int completedProjects;
  final String createdAt;

  // fromLegacyJson projects the legacy PublicProfileSummary envelope
  // into a fully-typed SearchDocument. Tolerates missing fields so
  // older backend versions can still feed the card. The `persona`
  // argument overrides the `org_type` guess because the directory
  // context (freelance vs referrer) is what the caller actually wants.
  factory SearchDocument.fromLegacyJson(
    Map<String, dynamic> json,
    SearchDocumentPersona persona,
  ) {
    final skills = <String>[];
    final rawSkills = json['skills'];
    if (rawSkills is List) {
      for (final entry in rawSkills) {
        if (skills.length >= 6) break;
        if (entry is Map) {
          final display = entry['display_text'] ?? entry['skill_text'];
          if (display is String && display.isNotEmpty) {
            skills.add(display);
          }
        } else if (entry is String && entry.isNotEmpty) {
          skills.add(entry);
        }
      }
    }

    final languages = <String>[];
    final rawLanguages = json['languages_professional'];
    if (rawLanguages is List) {
      for (final entry in rawLanguages) {
        if (entry is String) languages.add(entry);
      }
    }

    return SearchDocument(
      id: (json['organization_id'] ?? json['id'] ?? '') as String,
      persona: persona,
      displayName: (json['name'] ?? json['display_name'] ?? '') as String,
      title: (json['title'] ?? '') as String,
      photoUrl: (json['photo_url'] ?? '') as String,
      city: (json['city'] ?? '') as String,
      countryCode: (json['country_code'] ?? '') as String,
      languagesProfessional: languages,
      availabilityStatus: _availabilityFromWire(
        json['availability_status'] as String?,
      ),
      expertiseDomains: _stringList(json['expertise_domains']),
      skills: skills,
      pricing: _pickPricing(json['pricing'], persona),
      rating: SearchDocumentRating(
        average: _readDouble(json['average_rating']),
        count: _readInt(json['review_count']),
      ),
      totalEarned: _readInt(json['total_earned']),
      completedProjects: _readInt(json['completed_projects']),
      createdAt: (json['created_at'] ?? '') as String,
    );
  }
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

SearchDocumentAvailability _availabilityFromWire(String? raw) {
  switch (raw) {
    case 'available_soon':
      return SearchDocumentAvailability.availableSoon;
    case 'not_available':
      return SearchDocumentAvailability.notAvailable;
    case 'available_now':
    default:
      return SearchDocumentAvailability.availableNow;
  }
}

List<String> _stringList(dynamic raw) {
  if (raw is! List) return const [];
  final out = <String>[];
  for (final entry in raw) {
    if (entry is String) out.add(entry);
  }
  return out;
}

double _readDouble(dynamic raw) {
  if (raw is num) return raw.toDouble();
  return 0;
}

int _readInt(dynamic raw) {
  if (raw is num) return raw.toInt();
  return 0;
}

SearchDocumentPricing? _pickPricing(
  dynamic rows,
  SearchDocumentPersona persona,
) {
  if (rows is! List || rows.isEmpty) return null;
  final preferredKind =
      persona == SearchDocumentPersona.referrer ? 'referral' : 'direct';
  Map<String, dynamic>? chosen;
  for (final entry in rows) {
    if (entry is Map) {
      final map = Map<String, dynamic>.from(entry);
      if (map['kind'] == preferredKind) {
        chosen = map;
        break;
      }
      chosen ??= map;
    }
  }
  if (chosen == null) return null;

  final type = _pricingTypeFromWire(chosen['type'] as String?);
  if (type == null) return null;

  return SearchDocumentPricing(
    type: type,
    minAmount: _readInt(chosen['min_amount']),
    maxAmount: chosen['max_amount'] is num
        ? (chosen['max_amount'] as num).toInt()
        : null,
    currency: (chosen['currency'] ?? 'EUR') as String,
    negotiable: chosen['negotiable'] == true,
  );
}

SearchDocumentPricingType? _pricingTypeFromWire(String? raw) {
  switch (raw) {
    case 'daily':
      return SearchDocumentPricingType.daily;
    case 'hourly':
      return SearchDocumentPricingType.hourly;
    case 'project_from':
      return SearchDocumentPricingType.projectFrom;
    case 'project_range':
      return SearchDocumentPricingType.projectRange;
    case 'commission_pct':
      return SearchDocumentPricingType.commissionPct;
    case 'commission_flat':
      return SearchDocumentPricingType.commissionFlat;
    default:
      return null;
  }
}
