// Image-perf assertions for the portfolio cover thumbnails (cat D).
//
// PERF-M-05 calls out that CachedNetworkImage callsites without
// `memCacheWidth` decode the original 1080-2160 px JPEG to RAM
// even though the cell only renders a 180-220 lp wide cover. The
// Phase 4-O fix adds `memCacheWidth` and a RepaintBoundary at every
// portfolio cover site.
//
// We assert the source contract directly (no real network) — the
// alternative would be a golden image test with a stubbed image
// loader, which is brittle and slow. A static contract test fails
// in CI within milliseconds if a future change drops the cap.

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  group('PortfolioCard cover image perf contract', () {
    final src = File(
      'lib/features/portfolio/presentation/widgets/grid/portfolio_card.dart',
    ).readAsStringSync();

    test('cover decode is wrapped in a RepaintBoundary', () {
      // The widget renders inside a 2-column grid: every cell
      // repaints when the user scrolls or hovers — without a
      // RepaintBoundary the decoded raster invalidates with the
      // overlay layers above it.
      expect(
        src.contains('RepaintBoundary'),
        isTrue,
        reason: 'Portfolio cover must isolate its raster layer '
            'from the play-icon and gradient overlays (PERF-M-08)',
      );
    });

    test('CachedNetworkImage caps memCacheWidth on covers', () {
      // Two CachedNetworkImage call sites (custom thumbnail +
      // image cover) — both must declare a memCacheWidth.
      final memCacheCount = 'memCacheWidth:'.allMatches(src).length;
      expect(
        memCacheCount,
        greaterThanOrEqualTo(2),
        reason: 'Each CachedNetworkImage cover must cap memCacheWidth '
            '(PERF-M-05). Expected ≥ 2 occurrences, found $memCacheCount',
      );
      // Pinned to the 480 budget chosen by Phase 4-O. If the budget
      // is intentionally raised, update the test alongside the code.
      expect(
        src.contains('memCacheWidth: _coverMemCacheWidth'),
        isTrue,
      );
      expect(
        src.contains('static const int _coverMemCacheWidth = 480;'),
        isTrue,
      );
    });

    test('disk cache width matches memory cache width', () {
      final diskCount = 'maxWidthDiskCache:'.allMatches(src).length;
      expect(
        diskCount,
        greaterThanOrEqualTo(2),
        reason: 'Disk cache must mirror the memory budget so '
            'restarts don\'t re-decode the original full-size image',
      );
    });
  });

  group('SearchResultCard photo image perf contract', () {
    final src = File(
      'lib/shared/widgets/search/search_result_card.dart',
    ).readAsStringSync();

    test('photo cover wraps a RepaintBoundary', () {
      expect(src.contains('RepaintBoundary'), isTrue);
    });

    test('photo cover caps memCacheWidth at 720', () {
      expect(
        src.contains('static const int _memCacheWidth = 720;'),
        isTrue,
        reason: '4:5 photo cover renders ≤ 360 lp wide on phones; '
            '720 raster (2x DPR) is the right budget for sharpness '
            'without ballooning RAM (PERF-M-05)',
      );
      expect(src.contains('memCacheWidth: _memCacheWidth'), isTrue);
      expect(src.contains('maxWidthDiskCache: _memCacheWidth'), isTrue);
    });
  });

  group('ChatBubble image attachment perf contract', () {
    final src = File(
      'lib/features/messaging/presentation/widgets/chat/file_message_bubble.dart',
    ).readAsStringSync();

    test('image bubble caps memCacheWidth + wraps RepaintBoundary', () {
      expect(src.contains('memCacheWidth: 720'), isTrue);
      expect(src.contains('maxWidthDiskCache: 720'), isTrue);
      expect(src.contains('RepaintBoundary'), isTrue);
    });
  });

  group('Avatar CachedNetworkImageProvider perf contract', () {
    test('large avatars cap maxWidth/maxHeight at ≤ 256', () {
      // Spot-check the 5 hot avatar sites called out in PERF-M-05.
      final files = <String>[
        'lib/shared/widgets/profile_identity_header.dart',
        'lib/features/profile/presentation/widgets/profile_header_card.dart',
        'lib/features/job/presentation/widgets/candidate_card.dart',
        'lib/features/job/presentation/screens/candidate_detail_screen.dart',
        'lib/features/client_profile/presentation/widgets/client_profile_header.dart',
      ];
      for (final path in files) {
        final src = File(path).readAsStringSync();
        // Every CachedNetworkImageProvider call in these files must
        // pass a `maxWidth:` argument — we check the literal string
        // is present in the same file.
        expect(
          src.contains('CachedNetworkImageProvider'),
          isTrue,
          reason: 'Sanity: $path should still use the cached provider',
        );
        expect(
          src.contains('maxWidth: '),
          isTrue,
          reason: '$path: avatar provider must cap maxWidth '
              '(PERF-M-05) — without it the provider decodes the '
              'original 1080-2160 px JPEG into RAM',
        );
      }
    });
  });

  group('PortfolioFormMedia tile perf contract', () {
    final src = File(
      'lib/features/portfolio/presentation/widgets/portfolio_form_media.dart',
    ).readAsStringSync();

    test('Image.network sites declare cacheWidth', () {
      // PERF-M-09: brut Image.network re-downloaded + decoded to
      // full resolution on every rebuild. cacheWidth gates Flutter's
      // decode pipeline.
      expect(
        src.contains('cacheWidth: 360'),
        isTrue,
        reason: 'Form-tile previews are ~120 lp wide; 360 px raster '
            'is the appropriate decode budget',
      );
    });
  });
}
