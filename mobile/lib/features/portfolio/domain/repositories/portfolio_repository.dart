import '../entities/portfolio_item.dart';

/// Abstract repository for portfolio operations.
abstract class PortfolioRepository {
  /// Fetches all portfolio items for a user.
  Future<List<PortfolioItem>> getPortfolioByUser(String userId);

  /// Fetches a single portfolio item by ID.
  Future<PortfolioItem> getPortfolioItem(String id);

  /// Creates a new portfolio item.
  Future<PortfolioItem> createPortfolioItem({
    required String title,
    String? description,
    String? linkUrl,
    required int position,
    List<Map<String, dynamic>>? media,
  });

  /// Updates an existing portfolio item.
  Future<PortfolioItem> updatePortfolioItem(
    String id, {
    String? title,
    String? description,
    String? linkUrl,
    List<Map<String, dynamic>>? media,
  });

  /// Deletes a portfolio item.
  Future<void> deletePortfolioItem(String id);

  /// Reorders portfolio items by providing ordered item IDs.
  Future<void> reorderPortfolio(List<String> itemIds);

  /// Uploads a portfolio image and returns the URL.
  Future<String> uploadPortfolioImage(String filePath);

  /// Uploads a portfolio video and returns the URL.
  Future<String> uploadPortfolioVideo(String filePath);
}
