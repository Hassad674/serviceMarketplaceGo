/// The three availability states an organization can declare for
/// the marketplace. Matches the backend `availability_status` enum
/// from migration 083_profile_tier1_completion.
///
/// The canonical wire values are `available_now`, `available_soon`
/// and `not_available`. Any unknown or missing string defaults to
/// [AvailabilityStatus.availableNow] so a profile row never renders
/// an empty badge — the marketplace surface always needs a value.
enum AvailabilityStatus {
  availableNow('available_now'),
  availableSoon('available_soon'),
  notAvailable('not_available');

  const AvailabilityStatus(this.wire);

  /// The backend wire-format string. Used for JSON (de)serialization.
  final String wire;

  /// Parse a wire value. Returns `null` if the input is null or
  /// empty, [AvailabilityStatus.availableNow] for unknown values so
  /// the UI never breaks on stale backend data.
  static AvailabilityStatus? fromWireOrNull(String? raw) {
    if (raw == null || raw.isEmpty) return null;
    for (final value in AvailabilityStatus.values) {
      if (value.wire == raw) return value;
    }
    return AvailabilityStatus.availableNow;
  }

  /// Parse a wire value, falling back to a non-null default.
  static AvailabilityStatus fromWire(String? raw) {
    return fromWireOrNull(raw) ?? AvailabilityStatus.availableNow;
  }
}
