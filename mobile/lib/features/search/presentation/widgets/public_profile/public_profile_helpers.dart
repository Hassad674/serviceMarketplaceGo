import 'package:flutter/material.dart';

import '../../../../../l10n/app_localizations.dart';
import '../../../../profile_tier1/domain/entities/pricing.dart';
import '../../../../profile_tier1/domain/entities/pricing_kind.dart';
import '../../../../../core/theme/app_palette.dart';

/// Maps the legacy `pricing` array to a single [Pricing] row keyed by
/// `direct`. Agencies only advertise a direct rate on the public
/// page — referral commissions live on the referrer profile. Returns
/// null when no row exists so the card hides itself.
Pricing? pickDirectPricing(Map<String, dynamic> profile) {
  final raw = profile['pricing'];
  if (raw is! List) return null;
  for (final row in raw) {
    if (row is! Map<String, dynamic>) continue;
    try {
      final pricing = Pricing.fromJson(row);
      if (pricing.kind == PricingKind.direct) return pricing;
    } on FormatException {
      // Ignore malformed rows — never crash the public page.
    }
  }
  return null;
}

int? readIntField(Object? value) {
  if (value == null) return null;
  if (value is int) return value;
  if (value is double) return value.toInt();
  if (value is String) return int.tryParse(value);
  return null;
}

String workModeLabel(String key, AppLocalizations l10n) {
  switch (key) {
    case 'remote':
      return l10n.tier1LocationWorkModeRemote;
    case 'on_site':
      return l10n.tier1LocationWorkModeOnSite;
    case 'hybrid':
      return l10n.tier1LocationWorkModeHybrid;
    default:
      return key;
  }
}

Color publicProfileRoleColor(String? orgType) {
  switch (orgType) {
    case 'agency':
      return AppPalette.blue600;
    case 'enterprise':
      return AppPalette.violet500;
    case 'provider_personal':
      return AppPalette.rose500;
    default:
      return AppPalette.slate500;
  }
}

String buildInitialsFromName(String name) {
  if (name.isEmpty || name.startsWith('Org')) return '?';
  final parts = name.trim().split(RegExp(r'\s+'));
  if (parts.length == 1) return parts[0][0].toUpperCase();
  return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
}

String resolvePublicDisplayName(
  Map<String, dynamic> profile,
  String? navName,
) {
  if (navName != null && navName.isNotEmpty) return navName;
  final name = profile['name'] as String?;
  if (name != null && name.isNotEmpty) return name;
  final orgId = profile['organization_id'] as String?;
  if (orgId != null && orgId.length >= 8) {
    return 'Org ${orgId.substring(0, 8)}';
  }
  return 'Organization';
}
