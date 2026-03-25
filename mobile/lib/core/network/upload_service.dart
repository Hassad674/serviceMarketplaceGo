import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'api_client.dart';

/// Provides the singleton [UploadService] for photo and video uploads.
final uploadServiceProvider = Provider<UploadService>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return UploadService(apiClient: apiClient);
});

/// Handles multipart file uploads for photos and videos.
///
/// Delegates to [ApiClient.upload] which injects the JWT token automatically
/// and handles token refresh on 401.
class UploadService {
  final ApiClient _apiClient;

  UploadService({required ApiClient apiClient}) : _apiClient = apiClient;

  /// Uploads a photo file and returns the resulting URL.
  ///
  /// The backend expects a multipart form with field name `file`.
  Future<String> uploadPhoto(File file) async {
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(
        file.path,
        filename: file.path.split('/').last,
      ),
    });
    final response = await _apiClient.upload(
      '/api/v1/upload/photo',
      data: formData,
    );
    return response.data['url'] as String;
  }

  /// Uploads a video file and returns the resulting URL.
  ///
  /// The backend expects a multipart form with field name `file`.
  Future<String> uploadVideo(File file) async {
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(
        file.path,
        filename: file.path.split('/').last,
      ),
    });
    final response = await _apiClient.upload(
      '/api/v1/upload/video',
      data: formData,
    );
    return response.data['url'] as String;
  }
}
