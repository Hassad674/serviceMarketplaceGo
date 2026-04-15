import 'dart:io';

import '../entities/freelance_pricing.dart';
import '../entities/freelance_profile.dart';

/// Abstract data seam for the freelance profile feature. Groups
/// every endpoint exposed under `/api/v1/freelance-profile` behind a
/// single interface so the presentation layer never knows which HTTP
/// calls power it.
abstract class FreelanceProfileRepository {
  /// Fetches the authenticated operator's own freelance profile.
  /// Returns a populated entity or throws when the row is missing
  /// (e.g. agency org type trying to read this endpoint).
  Future<FreelanceProfile> getMy();

  /// Fetches the public read-only freelance profile for the given
  /// organization id.
  Future<FreelanceProfile> getPublic(String organizationId);

  /// Updates the core text fields (title, about, video URL).
  Future<void> updateCore({
    required String title,
    required String about,
    required String videoUrl,
  });

  /// Updates the availability slot on the freelance persona.
  /// [wireValue] must be one of `available_now`, `available_soon`,
  /// `not_available`.
  Future<void> updateAvailability(String wireValue);

  /// Updates the expertise domain selection (list of domain keys).
  Future<void> updateExpertise(List<String> domains);

  /// Fetches the current pricing row or null when none declared.
  Future<FreelancePricing?> getPricing();

  /// Upsert the pricing row. Returns the persisted value echoed by
  /// the backend so the caller can refresh local state without a
  /// second round-trip.
  Future<FreelancePricing> upsertPricing(FreelancePricing pricing);

  /// Remove the current pricing row. Succeeds even when no row
  /// exists — the backend treats the delete as idempotent.
  Future<void> deletePricing();

  /// Uploads a presentation video for the freelance persona.
  /// Posts the file as multipart form data to
  /// `POST /api/v1/freelance-profile/video` and returns the
  /// persisted public URL.
  Future<String> uploadVideo(File file);

  /// Removes the freelance presentation video. Calls
  /// `DELETE /api/v1/freelance-profile/video`. Idempotent — succeeds
  /// when no video is set.
  Future<void> deleteVideo();
}
