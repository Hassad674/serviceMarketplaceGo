import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/availability_pill.dart';
import '../../../../shared/widgets/profile_identity_header.dart';

/// Thin composition: [ProfileIdentityHeader] + teal-tinted
/// [AvailabilityPill]. Puts the name/title/availability trio in
/// one place for the referrer persona.
class ReferrerProfileHeader extends StatelessWidget {
  const ReferrerProfileHeader({
    super.key,
    required this.displayName,
    required this.title,
    required this.photoUrl,
    required this.initials,
    required this.availabilityWireValue,
  });

  /// Referrer persona accent — teal-500 to distinguish from the
  /// freelance rose.
  static const Color kAccent = Color(0xFF14B8A6);

  final String displayName;
  final String title;
  final String photoUrl;
  final String initials;
  final String availabilityWireValue;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return ProfileIdentityHeader(
      displayName: displayName,
      initials: initials,
      accentColor: kAccent,
      title: title,
      photoUrl: photoUrl,
      trailing: AvailabilityPill(
        wireValue: availabilityWireValue,
        label: _availabilityLabel(l10n, availabilityWireValue),
      ),
    );
  }

  String _availabilityLabel(AppLocalizations l10n, String wire) {
    switch (wire) {
      case 'available_soon':
        return l10n.tier1AvailabilityStatusAvailableSoon;
      case 'not_available':
        return l10n.tier1AvailabilityStatusNotAvailable;
      case 'available_now':
      default:
        return l10n.tier1AvailabilityStatusAvailableNow;
    }
  }
}
