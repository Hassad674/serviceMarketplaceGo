import 'dart:io';

import '../entities/referrer_pricing.dart';
import '../entities/referrer_profile.dart';

/// Abstract data seam for the referrer profile feature. Mirrors
/// [FreelanceProfileRepository] method-by-method so both personas
/// stay symmetrical from the presentation layer's perspective.
abstract class ReferrerProfileRepository {
  /// Fetches the authenticated operator's own referrer profile. The
  /// backend auto-creates the row on first read so callers never see
  /// a 404 — an empty row is returned instead.
  Future<ReferrerProfile> getMy();

  /// Fetches the public read-only referrer profile for the given
  /// organization id.
  Future<ReferrerProfile> getPublic(String organizationId);

  /// Updates the core text fields (title, about, video URL).
  Future<void> updateCore({
    required String title,
    required String about,
    required String videoUrl,
  });

  /// Updates the availability slot on the referrer persona.
  Future<void> updateAvailability(String wireValue);

  /// Updates the expertise domain selection.
  Future<void> updateExpertise(List<String> domains);

  /// Fetches the current pricing row or null when none declared.
  Future<ReferrerPricing?> getPricing();

  /// Upsert the pricing row.
  Future<ReferrerPricing> upsertPricing(ReferrerPricing pricing);

  /// Remove the current pricing row.
  Future<void> deletePricing();

  /// Uploads a presentation video for the referrer persona.
  /// Posts the file as multipart form data to
  /// `POST /api/v1/referrer-profile/video` and returns the persisted
  /// public URL.
  Future<String> uploadVideo(File file);

  /// Removes the referrer presentation video. Calls
  /// `DELETE /api/v1/referrer-profile/video`.
  Future<void> deleteVideo();
}
