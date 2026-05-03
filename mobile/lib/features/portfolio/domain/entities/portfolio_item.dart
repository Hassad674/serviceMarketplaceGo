/// Represents a single media (image or video) in a portfolio gallery.
class PortfolioMedia {
  final String id;
  final String mediaUrl;
  final String mediaType; // 'image' or 'video'
  final String thumbnailUrl; // Optional custom thumbnail (videos only)
  final int position;
  final DateTime createdAt;

  const PortfolioMedia({
    required this.id,
    required this.mediaUrl,
    required this.mediaType,
    this.thumbnailUrl = '',
    required this.position,
    required this.createdAt,
  });

  factory PortfolioMedia.fromJson(Map<String, dynamic> json) {
    return PortfolioMedia(
      id: json['id'] as String,
      mediaUrl: json['media_url'] as String,
      mediaType: json['media_type'] as String,
      thumbnailUrl: json['thumbnail_url'] as String? ?? '',
      position: json['position'] as int,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }

  bool get isImage => mediaType == 'image';
  bool get isVideo => mediaType == 'video';
  bool get hasCustomThumbnail => thumbnailUrl.isNotEmpty;
}

/// Represents a portfolio project entry.
///
/// Since phase R2 a portfolio item belongs to an organization — the
/// team's shared showcase. The anchor is [organizationId], not a
/// specific user id.
class PortfolioItem {
  final String id;
  final String organizationId;
  final String title;
  final String description;
  final String linkUrl;
  final String coverUrl;
  final int position;
  final List<PortfolioMedia> media;
  final DateTime createdAt;
  final DateTime updatedAt;

  const PortfolioItem({
    required this.id,
    required this.organizationId,
    required this.title,
    this.description = '',
    this.linkUrl = '',
    this.coverUrl = '',
    required this.position,
    this.media = const [],
    required this.createdAt,
    required this.updatedAt,
  });

  factory PortfolioItem.fromJson(Map<String, dynamic> json) {
    final mediaList = (json['media'] as List?)
            ?.map((m) => PortfolioMedia.fromJson(m as Map<String, dynamic>))
            .toList() ??
        [];

    return PortfolioItem(
      id: json['id'] as String,
      organizationId: json['organization_id'] as String,
      title: json['title'] as String,
      description: json['description'] as String? ?? '',
      linkUrl: json['link_url'] as String? ?? '',
      coverUrl: json['cover_url'] as String? ?? '',
      position: json['position'] as int,
      media: mediaList,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  int get imageCount => media.where((m) => m.isImage).length;
  int get videoCount => media.where((m) => m.isVideo).length;
}
