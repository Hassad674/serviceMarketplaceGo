import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/availability_pill.dart';
import '../../../../shared/widgets/profile_identity_header.dart';
import '../../../../core/theme/app_palette.dart';

/// Thin composition: [ProfileIdentityHeader] + freelance-tinted
/// [AvailabilityPill]. Keeps the screen files short and puts the
/// name/title/availability trio in one place.
class FreelanceProfileHeader extends StatelessWidget {
  const FreelanceProfileHeader({
    super.key,
    required this.displayName,
    required this.title,
    required this.photoUrl,
    required this.initials,
    required this.availabilityWireValue,
    this.portraitSeed,
    this.trailing,
  });

  /// Freelance persona accent — rose-500 to match the primary tone.
  static const Color kAccent = AppPalette.rose500;

  final String displayName;
  final String title;
  final String photoUrl;
  final String initials;
  final String availabilityWireValue;

  /// Stable seed used to pick a Soleil [Portrait] palette when no
  /// [photoUrl] is set. Falls back to the legacy initials avatar
  /// when null so existing callers keep their behaviour.
  final int? portraitSeed;

  /// Optional override for the trailing slot. When null the header
  /// renders the default availability pill so old callers keep
  /// working unchanged.
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return ProfileIdentityHeader(
      displayName: displayName,
      initials: initials,
      accentColor: kAccent,
      title: title,
      photoUrl: photoUrl,
      portraitSeed: portraitSeed,
      trailing: trailing ??
          AvailabilityPill(
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
