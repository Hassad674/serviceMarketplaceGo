import 'dart:io';

import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/freelance_pricing.dart';
import '../domain/entities/freelance_profile.dart';
import '../domain/repositories/freelance_profile_repository.dart';

/// Dio-backed implementation of [FreelanceProfileRepository].
///
/// Endpoints:
///
/// - `GET    /api/v1/freelance-profile`               -> getMy
/// - `PUT    /api/v1/freelance-profile`               -> updateCore
/// - `PUT    /api/v1/freelance-profile/availability`  -> updateAvailability
/// - `PUT    /api/v1/freelance-profile/expertise`     -> updateExpertise
/// - `GET    /api/v1/freelance-profile/pricing`       -> getPricing
/// - `PUT    /api/v1/freelance-profile/pricing`       -> upsertPricing
/// - `DELETE /api/v1/freelance-profile/pricing`       -> deletePricing
/// - `POST   /api/v1/freelance-profile/video`         -> uploadVideo
/// - `DELETE /api/v1/freelance-profile/video`         -> deleteVideo
/// - `GET    /api/v1/freelance-profiles/{orgID}`      -> getPublic
///
/// Tolerates both `{ "data": ... }` envelopes and raw payloads.
class FreelanceProfileRepositoryImpl implements FreelanceProfileRepository {
  FreelanceProfileRepositoryImpl(this._api);

  final ApiClient _api;

  @override
  Future<FreelanceProfile> getMy() async {
    final response = await _api.get('/api/v1/freelance-profile');
    final body = _unwrapMap(response.data);
    if (body == null) return FreelanceProfile.empty;
    return FreelanceProfile.fromJson(body);
  }

  @override
  Future<FreelanceProfile> getPublic(String organizationId) async {
    final response = await _api.get(
      '/api/v1/freelance-profiles/$organizationId',
    );
    final body = _unwrapMap(response.data);
    if (body == null) return FreelanceProfile.empty;
    return FreelanceProfile.fromJson(body);
  }

  @override
  Future<void> updateCore({
    required String title,
    required String about,
    required String videoUrl,
  }) async {
    await _api.put(
      '/api/v1/freelance-profile',
      data: <String, dynamic>{
        'title': title,
        'about': about,
        'video_url': videoUrl,
      },
    );
  }

  @override
  Future<void> updateAvailability(String wireValue) async {
    await _api.put(
      '/api/v1/freelance-profile/availability',
      data: <String, dynamic>{'availability_status': wireValue},
    );
  }

  @override
  Future<void> updateExpertise(List<String> domains) async {
    await _api.put(
      '/api/v1/freelance-profile/expertise',
      data: <String, dynamic>{'domains': domains},
    );
  }

  @override
  Future<FreelancePricing?> getPricing() async {
    final response =
        await _api.get('/api/v1/freelance-profile/pricing');
    final body = _unwrapMap(response.data);
    if (body == null || body.isEmpty) return null;
    try {
      return FreelancePricing.fromJson(body);
    } on FormatException {
      return null;
    }
  }

  @override
  Future<FreelancePricing> upsertPricing(FreelancePricing pricing) async {
    final response = await _api.put(
      '/api/v1/freelance-profile/pricing',
      data: pricing.toUpdatePayload(),
    );
    final body = _unwrapMap(response.data);
    if (body == null) return pricing;
    try {
      return FreelancePricing.fromJson(body);
    } on FormatException {
      return pricing;
    }
  }

  @override
  Future<void> deletePricing() async {
    await _api.delete('/api/v1/freelance-profile/pricing');
  }

  @override
  Future<String> uploadVideo(File file) async {
    final formData = FormData.fromMap(<String, dynamic>{
      'file': await MultipartFile.fromFile(
        file.path,
        filename: file.path.split('/').last,
      ),
    });
    final response = await _api.upload(
      '/api/v1/freelance-profile/video',
      data: formData,
    );
    final body = _unwrapMap(response.data);
    final url = body?['video_url'];
    if (url is! String || url.isEmpty) {
      throw const FormatException('freelance video upload: missing video_url');
    }
    return url;
  }

  @override
  Future<void> deleteVideo() async {
    await _api.delete('/api/v1/freelance-profile/video');
  }

  // ---------------------------------------------------------------------------
  // Parsing helpers
  // ---------------------------------------------------------------------------

  Map<String, dynamic>? _unwrapMap(Object? raw) {
    if (raw is Map<String, dynamic>) {
      final data = raw['data'];
      if (data is Map<String, dynamic>) return data;
      return raw;
    }
    return null;
  }
}
