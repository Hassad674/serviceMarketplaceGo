import '../entities/availability_status.dart';
import '../entities/location.dart';
import '../entities/pricing.dart';
import '../entities/pricing_kind.dart';

/// Abstract data seam for the Tier 1 completion feature.
///
/// Groups the four independently-updatable profile blocks —
/// location, languages, availability, pricing — behind one
/// interface so the presentation layer does not know (or care)
/// which endpoints power them.
///
/// The pricing surface is split in three methods because the
/// backend exposes a separate endpoint for each verb: list, upsert
/// per kind, and delete per kind. The UI uses all three.
abstract class ProfileTier1Repository {
  Future<void> updateLocation(Location location);

  Future<void> updateLanguages(
    List<String> professional,
    List<String> conversational,
  );

  Future<void> updateAvailability(
    AvailabilityStatus direct,
    AvailabilityStatus? referrer,
  );

  /// Fetches the 0..2 pricing rows declared by the current
  /// organization. Returns an empty list when nothing is declared.
  Future<List<Pricing>> getPricing();

  /// Upsert a pricing row. Returns the persisted value echoed by
  /// the backend so the caller can update local state with the
  /// canonical representation.
  Future<Pricing> upsertPricing(Pricing pricing);

  /// Remove the pricing row for the given [kind]. Succeeds even
  /// when no row exists — the backend treats the delete as
  /// idempotent.
  Future<void> deletePricing(PricingKind kind);
}
