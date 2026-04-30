import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/portfolio/domain/entities/portfolio_item.dart';

void main() {
  group('PortfolioMedia.fromJson', () {
    test('parses required fields', () {
      final m = PortfolioMedia.fromJson({
        'id': 'm-1',
        'media_url': 'https://x/img.jpg',
        'media_type': 'image',
        'position': 0,
        'created_at': '2026-04-01T00:00:00Z',
      });
      expect(m.id, 'm-1');
      expect(m.mediaUrl, 'https://x/img.jpg');
      expect(m.mediaType, 'image');
      expect(m.position, 0);
      expect(m.thumbnailUrl, '');
    });

    test('parses thumbnail when present', () {
      final m = PortfolioMedia.fromJson({
        'id': 'm-1',
        'media_url': 'https://x/v.mp4',
        'media_type': 'video',
        'thumbnail_url': 'https://x/t.jpg',
        'position': 0,
        'created_at': '2026-04-01T00:00:00Z',
      });
      expect(m.thumbnailUrl, 'https://x/t.jpg');
      expect(m.hasCustomThumbnail, isTrue);
    });
  });

  group('PortfolioMedia helpers', () {
    test('isImage returns true for image type', () {
      final m = PortfolioMedia(
        id: 'm',
        mediaUrl: 'u',
        mediaType: 'image',
        position: 0,
        createdAt: DateTime.now(),
      );
      expect(m.isImage, isTrue);
      expect(m.isVideo, isFalse);
    });

    test('isVideo returns true for video type', () {
      final m = PortfolioMedia(
        id: 'm',
        mediaUrl: 'u',
        mediaType: 'video',
        position: 0,
        createdAt: DateTime.now(),
      );
      expect(m.isVideo, isTrue);
      expect(m.isImage, isFalse);
    });

    test('hasCustomThumbnail is false when empty', () {
      final m = PortfolioMedia(
        id: 'm',
        mediaUrl: 'u',
        mediaType: 'video',
        position: 0,
        createdAt: DateTime.now(),
      );
      expect(m.hasCustomThumbnail, isFalse);
    });
  });

  group('PortfolioItem.fromJson', () {
    final base = {
      'id': 'p-1',
      'organization_id': 'org-1',
      'title': 'Project A',
      'position': 0,
      'created_at': '2026-04-01T00:00:00Z',
      'updated_at': '2026-04-02T00:00:00Z',
    };

    test('parses required fields', () {
      final p = PortfolioItem.fromJson(Map<String, dynamic>.from(base));
      expect(p.id, 'p-1');
      expect(p.organizationId, 'org-1');
      expect(p.title, 'Project A');
      expect(p.description, '');
      expect(p.linkUrl, '');
      expect(p.coverUrl, '');
      expect(p.media, isEmpty);
    });

    test('parses optional description/linkUrl/coverUrl', () {
      final json = Map<String, dynamic>.from(base)
        ..['description'] = 'desc'
        ..['link_url'] = 'https://l'
        ..['cover_url'] = 'https://c';
      final p = PortfolioItem.fromJson(json);
      expect(p.description, 'desc');
      expect(p.linkUrl, 'https://l');
      expect(p.coverUrl, 'https://c');
    });

    test('parses media list', () {
      final json = Map<String, dynamic>.from(base)
        ..['media'] = [
          {
            'id': 'm-1',
            'media_url': 'https://x/i.jpg',
            'media_type': 'image',
            'position': 0,
            'created_at': '2026-04-01T00:00:00Z',
          },
          {
            'id': 'm-2',
            'media_url': 'https://x/v.mp4',
            'media_type': 'video',
            'position': 1,
            'created_at': '2026-04-01T00:00:00Z',
          },
        ];
      final p = PortfolioItem.fromJson(json);
      expect(p.media.length, 2);
      expect(p.imageCount, 1);
      expect(p.videoCount, 1);
    });
  });

  group('PortfolioItem aggregate counts', () {
    test('imageCount/videoCount work with mixed media', () {
      final p = PortfolioItem(
        id: 'p',
        organizationId: 'org',
        title: 't',
        position: 0,
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
        media: [
          PortfolioMedia(
            id: 'a',
            mediaUrl: 'u',
            mediaType: 'image',
            position: 0,
            createdAt: DateTime.now(),
          ),
          PortfolioMedia(
            id: 'b',
            mediaUrl: 'u',
            mediaType: 'image',
            position: 1,
            createdAt: DateTime.now(),
          ),
          PortfolioMedia(
            id: 'c',
            mediaUrl: 'u',
            mediaType: 'video',
            position: 2,
            createdAt: DateTime.now(),
          ),
        ],
      );
      expect(p.imageCount, 2);
      expect(p.videoCount, 1);
    });

    test('imageCount/videoCount are 0 when media is empty', () {
      final p = PortfolioItem(
        id: 'p',
        organizationId: 'org',
        title: 't',
        position: 0,
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
      );
      expect(p.imageCount, 0);
      expect(p.videoCount, 0);
    });
  });
}
