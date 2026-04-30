import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/portfolio/presentation/widgets/portfolio_form_media.dart';

/// Suppress NetworkImageLoadException — the test framework treats it as an
/// unhandled exception even though we never assert on rendered pixels.
void _suppressNetworkImageErrors() {
  final originalOnError = FlutterError.onError;
  FlutterError.onError = (FlutterErrorDetails details) {
    final exception = details.exception.toString();
    if (exception.contains('NetworkImageLoadException') ||
        exception.contains('HTTP request failed')) {
      return;
    }
    originalOnError?.call(details);
  };
}

Widget _wrap(Widget child) => MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(child: child),
      ),
    );

void main() {
  setUpAll(_suppressNetworkImageErrors);

  group('PortfolioMediaDraft', () {
    test('isVideo is true for video type', () {
      final draft = PortfolioMediaDraft(
        mediaUrl: 'https://example.com/video.mp4',
        mediaType: 'video',
        position: 0,
      );
      expect(draft.isVideo, isTrue);
    });

    test('isVideo is false for image type', () {
      final draft = PortfolioMediaDraft(
        mediaUrl: 'https://example.com/img.png',
        mediaType: 'image',
        position: 0,
      );
      expect(draft.isVideo, isFalse);
    });

    test('thumbnailUrl defaults to empty', () {
      final draft = PortfolioMediaDraft(
        mediaUrl: 'x',
        mediaType: 'image',
        position: 1,
      );
      expect(draft.thumbnailUrl, isEmpty);
      expect(draft.position, 1);
    });
  });

  group('PortfolioFormMediaSection - empty state', () {
    testWidgets('renders empty uploader CTA, header label and counter 0/8',
        (tester) async {
      var addSheetCalls = 0;
      await tester.pumpWidget(
        _wrap(
          PortfolioFormMediaSection(
            media: const [],
            uploadingMedia: false,
            onShowAddSheet: () => addSheetCalls++,
            onRemoveMedia: (_) {},
            onPickCustomThumbnail: (_) {},
            onRevertCustomThumbnail: (_) {},
          ),
        ),
      );

      expect(find.text('Media'), findsOneWidget);
      expect(find.text('0/$kPortfolioMaxMedia'), findsOneWidget);
      expect(find.text('Tap to add images or videos'), findsOneWidget);
      expect(find.text('Up to $kPortfolioMaxMedia files'), findsOneWidget);

      // Tap the empty uploader → onShowAddSheet
      await tester.tap(find.text('Tap to add images or videos'));
      await tester.pump();
      expect(addSheetCalls, 1);
    });

    testWidgets('shows progress indicator when uploadingMedia is true',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          PortfolioFormMediaSection(
            media: const [],
            uploadingMedia: true,
            onShowAddSheet: () {},
            onRemoveMedia: (_) {},
            onPickCustomThumbnail: (_) {},
            onRevertCustomThumbnail: (_) {},
          ),
        ),
      );

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text('Tap to add images or videos'), findsNothing);
    });
  });

  group('PortfolioFormMediaSection - populated grid', () {
    testWidgets('shows cover badge on first item, hint text, no add button',
        (tester) async {
      // Fill the grid up to the max — no add button should be rendered.
      final media = List.generate(
        kPortfolioMaxMedia,
        (i) => PortfolioMediaDraft(
          mediaUrl: 'https://example.com/img$i.png',
          mediaType: 'image',
          position: i,
        ),
      );
      await tester.pumpWidget(
        _wrap(
          PortfolioFormMediaSection(
            media: media,
            uploadingMedia: false,
            onShowAddSheet: () {},
            onRemoveMedia: (_) {},
            onPickCustomThumbnail: (_) {},
            onRevertCustomThumbnail: (_) {},
          ),
        ),
      );
      // Drain image-loading errors that originate from Image.network.
      await tester.pump(const Duration(milliseconds: 100));
      // Consume any image network exceptions.
      tester.takeException();

      expect(
        find.text('$kPortfolioMaxMedia/$kPortfolioMaxMedia'),
        findsOneWidget,
      );
      expect(find.text('Cover'), findsOneWidget);
      expect(
        find.text('The first media will be used as the cover.'),
        findsOneWidget,
      );
      expect(find.byIcon(Icons.add), findsNothing);
    });

    testWidgets('renders add button when below max, video shows play icon',
        (tester) async {
      final media = [
        PortfolioMediaDraft(
          mediaUrl: 'https://example.com/v.mp4',
          mediaType: 'video',
          position: 0,
        ),
      ];
      await tester.pumpWidget(
        _wrap(
          PortfolioFormMediaSection(
            media: media,
            uploadingMedia: false,
            onShowAddSheet: () {},
            onRemoveMedia: (_) {},
            onPickCustomThumbnail: (_) {},
            onRevertCustomThumbnail: (_) {},
          ),
        ),
      );
      await tester.pump(const Duration(milliseconds: 100));
      tester.takeException();

      expect(find.byIcon(Icons.add), findsOneWidget);
      expect(find.byIcon(Icons.play_circle_fill), findsOneWidget);
      expect(find.text('Cover perso'), findsOneWidget);
    });

    testWidgets('remove button calls onRemoveMedia with correct index',
        (tester) async {
      var removed = -1;
      final media = [
        PortfolioMediaDraft(
          mediaUrl: 'https://example.com/a.png',
          mediaType: 'image',
          position: 0,
        ),
        PortfolioMediaDraft(
          mediaUrl: 'https://example.com/b.png',
          mediaType: 'image',
          position: 1,
        ),
      ];
      await tester.pumpWidget(
        _wrap(
          SizedBox(
            height: 600,
            child: PortfolioFormMediaSection(
              media: media,
              uploadingMedia: false,
              onShowAddSheet: () {},
              onRemoveMedia: (idx) => removed = idx,
              onPickCustomThumbnail: (_) {},
              onRevertCustomThumbnail: (_) {},
            ),
          ),
        ),
      );
      await tester.pump(const Duration(milliseconds: 100));
      tester.takeException();

      final closeIcons = find.byIcon(Icons.close);
      expect(closeIcons, findsAtLeastNWidgets(2));
      await tester.tap(closeIcons.first);
      await tester.pump();
      expect(removed, 0);
    });

    testWidgets(
        'video with thumbnailUrl set shows "Custom" label and refresh icon',
        (tester) async {
      final media = [
        PortfolioMediaDraft(
          mediaUrl: 'https://example.com/v.mp4',
          mediaType: 'video',
          thumbnailUrl: 'https://example.com/cover.png',
          position: 0,
        ),
      ];
      await tester.pumpWidget(
        _wrap(
          PortfolioFormMediaSection(
            media: media,
            uploadingMedia: false,
            onShowAddSheet: () {},
            onRemoveMedia: (_) {},
            onPickCustomThumbnail: (_) {},
            onRevertCustomThumbnail: (_) {},
          ),
        ),
      );
      await tester.pump(const Duration(milliseconds: 100));
      tester.takeException();

      expect(find.text('Custom'), findsOneWidget);
      expect(find.byIcon(Icons.refresh), findsOneWidget);
    });
  });
}
