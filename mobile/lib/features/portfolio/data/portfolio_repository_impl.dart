import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../domain/entities/portfolio_item.dart';
import '../domain/repositories/portfolio_repository.dart';

/// Concrete implementation of [PortfolioRepository] using the Go backend API.
class PortfolioRepositoryImpl implements PortfolioRepository {
  final ApiClient _api;

  PortfolioRepositoryImpl(this._api);

  @override
  Future<List<PortfolioItem>> getPortfolioByOrganization(String orgId) async {
    final response =
        await _api.get('/api/v1/portfolio/org/$orgId?limit=30');
    final list = response.data['data'] as List? ?? [];
    return list
        .map((json) => PortfolioItem.fromJson(json as Map<String, dynamic>))
        .toList();
  }

  @override
  Future<PortfolioItem> getPortfolioItem(String id) async {
    final response = await _api.get('/api/v1/portfolio/$id');
    return PortfolioItem.fromJson(
      response.data['data'] as Map<String, dynamic>,
    );
  }

  @override
  Future<PortfolioItem> createPortfolioItem({
    required String title,
    String? description,
    String? linkUrl,
    required int position,
    List<Map<String, dynamic>>? media,
  }) async {
    final body = <String, dynamic>{
      'title': title,
      'position': position,
    };
    if (description != null && description.isNotEmpty) {
      body['description'] = description;
    }
    if (linkUrl != null && linkUrl.isNotEmpty) {
      body['link_url'] = linkUrl;
    }
    if (media != null && media.isNotEmpty) {
      body['media'] = media;
    }

    final response = await _api.post('/api/v1/portfolio', data: body);
    return PortfolioItem.fromJson(
      response.data['data'] as Map<String, dynamic>,
    );
  }

  @override
  Future<PortfolioItem> updatePortfolioItem(
    String id, {
    String? title,
    String? description,
    String? linkUrl,
    List<Map<String, dynamic>>? media,
  }) async {
    final body = <String, dynamic>{};
    if (title != null) body['title'] = title;
    if (description != null) body['description'] = description;
    if (linkUrl != null) body['link_url'] = linkUrl;
    if (media != null) body['media'] = media;

    final response = await _api.put('/api/v1/portfolio/$id', data: body);
    return PortfolioItem.fromJson(
      response.data['data'] as Map<String, dynamic>,
    );
  }

  @override
  Future<void> deletePortfolioItem(String id) async {
    await _api.delete('/api/v1/portfolio/$id');
  }

  @override
  Future<void> reorderPortfolio(List<String> itemIds) async {
    await _api.put(
      '/api/v1/portfolio/reorder',
      data: {'item_ids': itemIds},
    );
  }

  @override
  Future<String> uploadPortfolioImage(String filePath) async {
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(filePath),
    });
    final response = await _api.post(
      '/api/v1/upload/portfolio-image',
      data: formData,
    );
    return response.data['url'] as String;
  }

  @override
  Future<String> uploadPortfolioVideo(String filePath) async {
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(filePath),
    });
    final response = await _api.post(
      '/api/v1/upload/portfolio-video',
      data: formData,
    );
    return response.data['url'] as String;
  }
}
