/// Backend wire values for the profile availability field shared by
/// the freelance, referrer, and legacy profile personas.
///
/// Lives under `shared/profile/` so every feature can import the
/// canonical list of values without pulling a feature's domain.
abstract final class AvailabilityStatusWire {
  static const String availableNow = 'available_now';
  static const String availableSoon = 'available_soon';
  static const String notAvailable = 'not_available';

  static const List<String> values = <String>[
    availableNow,
    availableSoon,
    notAvailable,
  ];

  /// Normalizes an unknown or empty value to `available_now` so the
  /// UI never renders a blank badge.
  static String normalize(String? raw) {
    if (raw == null || raw.isEmpty) return availableNow;
    return values.contains(raw) ? raw : availableNow;
  }
}
