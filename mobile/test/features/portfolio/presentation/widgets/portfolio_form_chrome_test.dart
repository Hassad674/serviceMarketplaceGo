import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/portfolio/presentation/widgets/portfolio_form_chrome.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('PortfolioFormChrome', () {
    testWidgets('shows "Add project" + helper line when not editing',
        (tester) async {
      await tester.pumpWidget(
        _wrap(PortfolioFormChrome(isEdit: false, onClose: () {})),
      );
      expect(find.text('Add project'), findsOneWidget);
      expect(
        find.text('Showcase a project with images, videos and a link'),
        findsOneWidget,
      );
      expect(find.text('Edit project'), findsNothing);
    });

    testWidgets('shows "Edit project" + helper line when editing',
        (tester) async {
      await tester.pumpWidget(
        _wrap(PortfolioFormChrome(isEdit: true, onClose: () {})),
      );
      expect(find.text('Edit project'), findsOneWidget);
      expect(find.text('Update your project details'), findsOneWidget);
    });

    testWidgets('close button calls onClose', (tester) async {
      var closes = 0;
      await tester.pumpWidget(
        _wrap(
          PortfolioFormChrome(isEdit: false, onClose: () => closes++),
        ),
      );
      await tester.tap(find.byIcon(Icons.close));
      await tester.pump();
      expect(closes, 1);
    });
  });

  group('PortfolioAddMediaSheet', () {
    testWidgets('renders both list tiles with correct icons + labels',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          PortfolioAddMediaSheet(
            onPickImage: () {},
            onPickVideo: () {},
          ),
        ),
      );
      expect(find.text('Add an image'), findsOneWidget);
      expect(find.text('Add a video'), findsOneWidget);
      expect(find.byIcon(Icons.image_outlined), findsOneWidget);
      expect(find.byIcon(Icons.videocam_outlined), findsOneWidget);
    });

    testWidgets('image tile pops then invokes onPickImage', (tester) async {
      var picks = 0;
      // The widget calls Navigator.of(context).pop(), which requires being
      // pushed via showModalBottomSheet — wrap in a Navigator with a route.
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) => ElevatedButton(
                onPressed: () => showModalBottomSheet<void>(
                  context: context,
                  builder: (_) => PortfolioAddMediaSheet(
                    onPickImage: () => picks++,
                    onPickVideo: () {},
                  ),
                ),
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Add an image'));
      await tester.pumpAndSettle();
      expect(picks, 1);
      // The sheet should have closed
      expect(find.text('Add an image'), findsNothing);
    });

    testWidgets('video tile pops then invokes onPickVideo', (tester) async {
      var picks = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) => ElevatedButton(
                onPressed: () => showModalBottomSheet<void>(
                  context: context,
                  builder: (_) => PortfolioAddMediaSheet(
                    onPickImage: () {},
                    onPickVideo: () => picks++,
                  ),
                ),
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Add a video'));
      await tester.pumpAndSettle();
      expect(picks, 1);
    });
  });
}
