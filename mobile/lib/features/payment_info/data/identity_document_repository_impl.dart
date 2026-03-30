import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/identity_document_entity.dart';

/// Repository for identity document operations.
abstract class IdentityDocumentRepository {
  Future<List<IdentityDocument>> listDocuments();
  Future<IdentityDocument> uploadDocument({
    required String documentType,
    required String side,
    required String filePath,
    required String fileName,
  });
  Future<void> deleteDocument(String id);
}

/// Concrete implementation using the Go backend API.
class IdentityDocumentRepositoryImpl implements IdentityDocumentRepository {
  final ApiClient _api;

  IdentityDocumentRepositoryImpl(this._api);

  @override
  Future<List<IdentityDocument>> listDocuments() async {
    final response = await _api.get('/api/v1/identity-documents');
    final data = response.data;
    if (data == null || data is! List) return [];
    return data
        .map(
          (e) =>
              IdentityDocument.fromJson(e as Map<String, dynamic>),
        )
        .toList();
  }

  @override
  Future<IdentityDocument> uploadDocument({
    required String documentType,
    required String side,
    required String filePath,
    required String fileName,
  }) async {
    final formData = FormData.fromMap({
      'category': 'identity',
      'document_type': documentType,
      'side': side,
      'file': await MultipartFile.fromFile(filePath, filename: fileName),
    });

    final response = await _api.upload(
      '/api/v1/identity-documents/upload',
      data: formData,
    );

    return IdentityDocument.fromJson(
      response.data as Map<String, dynamic>,
    );
  }

  @override
  Future<void> deleteDocument(String id) async {
    await _api.delete('/api/v1/identity-documents/$id');
  }
}
