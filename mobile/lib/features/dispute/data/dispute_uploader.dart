import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import '../../../core/network/api_client.dart';
import '../../../core/utils/mime_type_helper.dart';

/// Uploads dispute attachment files using the messaging presigned URL endpoint.
///
/// Flow:
/// 1. POST /api/v1/messaging/upload-url for each file → returns {upload_url, public_url}
/// 2. PUT file bytes to the upload_url with a separate Dio instance (no JWT interceptor)
/// 3. Return the list of {filename, url, size, mime_type} ready for the dispute API
Future<List<Map<String, dynamic>>> uploadDisputeFiles(
  ApiClient apiClient,
  List<File> files,
) async {
  if (files.isEmpty) return const [];

  final results = <Map<String, dynamic>>[];
  final uploadDio = Dio();

  for (final file in files) {
    try {
      final filename = file.path.split('/').last;
      final contentType = guessContentType(filename);
      final bytes = await file.readAsBytes();

      // Step 1: get presigned upload URL
      final response = await apiClient.post(
        '/api/v1/messaging/upload-url',
        data: {'filename': filename, 'content_type': contentType},
      );

      final body = response.data;
      final data = body is Map<String, dynamic> && body.containsKey('data')
          ? body['data'] as Map<String, dynamic>
          : body as Map<String, dynamic>;

      final uploadUrl = data['upload_url'] as String;
      final publicUrl = data['public_url'] as String? ?? '';

      // Step 2: PUT bytes directly to the storage URL
      await uploadDio.put<void>(
        uploadUrl,
        data: Stream.fromIterable([bytes]),
        options: Options(
          contentType: contentType,
          headers: {
            Headers.contentLengthHeader: bytes.length,
          },
        ),
      );

      // Step 3: collect metadata for the backend
      results.add({
        'filename': filename,
        'url': publicUrl,
        'size': bytes.length,
        'mime_type': contentType,
      });
    } catch (e) {
      debugPrint('[DisputeUploader] failed to upload ${file.path}: $e');
      rethrow;
    }
  }

  return results;
}
