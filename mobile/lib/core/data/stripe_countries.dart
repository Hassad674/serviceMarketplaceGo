/// Countries Stripe Connect Custom accounts can be created in, FROM a
/// FR platform. Mirrors the canonical list at
/// `web/src/shared/lib/stripe-countries.ts` — the two MUST stay in sync.
///
/// Each entry: ISO 3166-1 alpha-2 code, emoji flag, French label,
/// English label, region (used to group entries in selectors).
class StripeCountry {
  const StripeCountry({
    required this.code,
    required this.flag,
    required this.labelFr,
    required this.labelEn,
    required this.region,
    this.companyOnly = false,
  });

  final String code;
  final String flag;
  final String labelFr;
  final String labelEn;
  final StripeCountryRegion region;

  /// True when the country cannot onboard as an individual from a FR
  /// platform — only as a company. Used by the payment-info onboarding
  /// to filter business-type options.
  final bool companyOnly;
}

enum StripeCountryRegion { eu, europeOther, americas, apac, mena }

/// Human-readable region label, French (default app locale).
const Map<StripeCountryRegion, String> kStripeRegionLabelsFr =
    <StripeCountryRegion, String>{
  StripeCountryRegion.eu: 'Union européenne',
  StripeCountryRegion.europeOther: 'Europe (hors UE)',
  StripeCountryRegion.americas: 'Amériques',
  StripeCountryRegion.apac: 'Asie-Pacifique',
  StripeCountryRegion.mena: 'Moyen-Orient',
};

/// 43 supported countries. Sorted by region then label for intuitive
/// grouping in country selectors.
const List<StripeCountry> kStripeConnectCountries = <StripeCountry>[
  // European Union
  StripeCountry(code: 'FR', flag: '🇫🇷', labelFr: 'France', labelEn: 'France', region: StripeCountryRegion.eu),
  StripeCountry(code: 'BE', flag: '🇧🇪', labelFr: 'Belgique', labelEn: 'Belgium', region: StripeCountryRegion.eu),
  StripeCountry(code: 'LU', flag: '🇱🇺', labelFr: 'Luxembourg', labelEn: 'Luxembourg', region: StripeCountryRegion.eu),
  StripeCountry(code: 'DE', flag: '🇩🇪', labelFr: 'Allemagne', labelEn: 'Germany', region: StripeCountryRegion.eu),
  StripeCountry(code: 'AT', flag: '🇦🇹', labelFr: 'Autriche', labelEn: 'Austria', region: StripeCountryRegion.eu),
  StripeCountry(code: 'NL', flag: '🇳🇱', labelFr: 'Pays-Bas', labelEn: 'Netherlands', region: StripeCountryRegion.eu),
  StripeCountry(code: 'IT', flag: '🇮🇹', labelFr: 'Italie', labelEn: 'Italy', region: StripeCountryRegion.eu),
  StripeCountry(code: 'ES', flag: '🇪🇸', labelFr: 'Espagne', labelEn: 'Spain', region: StripeCountryRegion.eu),
  StripeCountry(code: 'PT', flag: '🇵🇹', labelFr: 'Portugal', labelEn: 'Portugal', region: StripeCountryRegion.eu),
  StripeCountry(code: 'IE', flag: '🇮🇪', labelFr: 'Irlande', labelEn: 'Ireland', region: StripeCountryRegion.eu),
  StripeCountry(code: 'DK', flag: '🇩🇰', labelFr: 'Danemark', labelEn: 'Denmark', region: StripeCountryRegion.eu),
  StripeCountry(code: 'SE', flag: '🇸🇪', labelFr: 'Suède', labelEn: 'Sweden', region: StripeCountryRegion.eu),
  StripeCountry(code: 'FI', flag: '🇫🇮', labelFr: 'Finlande', labelEn: 'Finland', region: StripeCountryRegion.eu),
  StripeCountry(code: 'PL', flag: '🇵🇱', labelFr: 'Pologne', labelEn: 'Poland', region: StripeCountryRegion.eu),
  StripeCountry(code: 'CZ', flag: '🇨🇿', labelFr: 'République Tchèque', labelEn: 'Czech Republic', region: StripeCountryRegion.eu),
  StripeCountry(code: 'GR', flag: '🇬🇷', labelFr: 'Grèce', labelEn: 'Greece', region: StripeCountryRegion.eu),
  StripeCountry(code: 'EE', flag: '🇪🇪', labelFr: 'Estonie', labelEn: 'Estonia', region: StripeCountryRegion.eu),
  StripeCountry(code: 'LV', flag: '🇱🇻', labelFr: 'Lettonie', labelEn: 'Latvia', region: StripeCountryRegion.eu),
  StripeCountry(code: 'LT', flag: '🇱🇹', labelFr: 'Lituanie', labelEn: 'Lithuania', region: StripeCountryRegion.eu),
  StripeCountry(code: 'SK', flag: '🇸🇰', labelFr: 'Slovaquie', labelEn: 'Slovakia', region: StripeCountryRegion.eu),
  StripeCountry(code: 'SI', flag: '🇸🇮', labelFr: 'Slovénie', labelEn: 'Slovenia', region: StripeCountryRegion.eu),
  StripeCountry(code: 'HU', flag: '🇭🇺', labelFr: 'Hongrie', labelEn: 'Hungary', region: StripeCountryRegion.eu),
  StripeCountry(code: 'BG', flag: '🇧🇬', labelFr: 'Bulgarie', labelEn: 'Bulgaria', region: StripeCountryRegion.eu),
  StripeCountry(code: 'HR', flag: '🇭🇷', labelFr: 'Croatie', labelEn: 'Croatia', region: StripeCountryRegion.eu),
  StripeCountry(code: 'RO', flag: '🇷🇴', labelFr: 'Roumanie', labelEn: 'Romania', region: StripeCountryRegion.eu),
  StripeCountry(code: 'CY', flag: '🇨🇾', labelFr: 'Chypre', labelEn: 'Cyprus', region: StripeCountryRegion.eu),
  StripeCountry(code: 'MT', flag: '🇲🇹', labelFr: 'Malte', labelEn: 'Malta', region: StripeCountryRegion.eu),

  // Europe (non-EU)
  StripeCountry(code: 'GB', flag: '🇬🇧', labelFr: 'Royaume-Uni', labelEn: 'United Kingdom', region: StripeCountryRegion.europeOther),
  StripeCountry(code: 'CH', flag: '🇨🇭', labelFr: 'Suisse', labelEn: 'Switzerland', region: StripeCountryRegion.europeOther),
  StripeCountry(code: 'NO', flag: '🇳🇴', labelFr: 'Norvège', labelEn: 'Norway', region: StripeCountryRegion.europeOther),
  StripeCountry(code: 'LI', flag: '🇱🇮', labelFr: 'Liechtenstein', labelEn: 'Liechtenstein', region: StripeCountryRegion.europeOther),
  StripeCountry(code: 'GI', flag: '🇬🇮', labelFr: 'Gibraltar', labelEn: 'Gibraltar', region: StripeCountryRegion.europeOther),

  // Americas
  StripeCountry(code: 'US', flag: '🇺🇸', labelFr: 'États-Unis', labelEn: 'United States', region: StripeCountryRegion.americas),
  StripeCountry(code: 'CA', flag: '🇨🇦', labelFr: 'Canada', labelEn: 'Canada', region: StripeCountryRegion.americas),
  StripeCountry(code: 'MX', flag: '🇲🇽', labelFr: 'Mexique', labelEn: 'Mexico', region: StripeCountryRegion.americas),

  // APAC
  StripeCountry(code: 'AU', flag: '🇦🇺', labelFr: 'Australie', labelEn: 'Australia', region: StripeCountryRegion.apac),
  StripeCountry(code: 'NZ', flag: '🇳🇿', labelFr: 'Nouvelle-Zélande', labelEn: 'New Zealand', region: StripeCountryRegion.apac),
  StripeCountry(code: 'SG', flag: '🇸🇬', labelFr: 'Singapour', labelEn: 'Singapore', region: StripeCountryRegion.apac),
  StripeCountry(code: 'HK', flag: '🇭🇰', labelFr: 'Hong Kong', labelEn: 'Hong Kong', region: StripeCountryRegion.apac),
  StripeCountry(code: 'JP', flag: '🇯🇵', labelFr: 'Japon', labelEn: 'Japan', region: StripeCountryRegion.apac),
  StripeCountry(code: 'TH', flag: '🇹🇭', labelFr: 'Thaïlande', labelEn: 'Thailand', region: StripeCountryRegion.apac),
  StripeCountry(code: 'MY', flag: '🇲🇾', labelFr: 'Malaisie', labelEn: 'Malaysia', region: StripeCountryRegion.apac),

  // MENA
  StripeCountry(code: 'AE', flag: '🇦🇪', labelFr: 'Émirats Arabes Unis', labelEn: 'United Arab Emirates', region: StripeCountryRegion.mena, companyOnly: true),
];

/// Iterates [kStripeConnectCountries] in the canonical region order so
/// callers can render section headers without re-grouping themselves.
Iterable<StripeCountryRegion> get kStripeRegionOrder => const <StripeCountryRegion>[
      StripeCountryRegion.eu,
      StripeCountryRegion.europeOther,
      StripeCountryRegion.americas,
      StripeCountryRegion.apac,
      StripeCountryRegion.mena,
    ];

/// Returns the entries for [region]. Order matches the source list.
List<StripeCountry> stripeCountriesByRegion(StripeCountryRegion region) =>
    kStripeConnectCountries.where((c) => c.region == region).toList();
