import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/profile/presentation/widgets/profile_video_section.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) => MaterialApp(
      theme: AppTheme.light,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    );

void main() {
  group('ProfileVideoSection - empty state', () {
    testWidgets('shows "Add a video" CTA when onUploadTap is set',
        (tester) async {
      var uploads = 0;
      await tester.pumpWidget(
        _wrap(
          ProfileVideoSection(onUploadTap: () => uploads++),
        ),
      );
      await tester.pump();
      // Empty state container holds at least one videocam icon header + one
      // empty-state placeholder icon.
      expect(find.byIcon(Icons.videocam_outlined), findsAtLeastNWidgets(1));
      expect(find.byIcon(Icons.add), findsOneWidget);

      // Tap the CTA button (ElevatedButton in empty state).
      await tester.tap(find.byType(ElevatedButton));
      await tester.pump();
      expect(uploads, 1);
    });

    testWidgets('hides CTA button when onUploadTap is null', (tester) async {
      await tester.pumpWidget(_wrap(const ProfileVideoSection()));
      await tester.pump();
      expect(find.byType(ElevatedButton), findsNothing);
    });
  });

  group('ProfileVideoSection - filled state (videoUrl present)', () {
    testWidgets(
        'shows replace and remove buttons when both callbacks provided',
        (tester) async {
      var replaces = 0;
      var deletes = 0;
      await tester.pumpWidget(
        _wrap(
          ProfileVideoSection(
            videoUrl: 'https://example.com/v.mp4',
            onUploadTap: () => replaces++,
            onDeleteTap: () => deletes++,
          ),
        ),
      );
      // Allow the embedded video player a frame to settle.
      await tester.pump(const Duration(milliseconds: 100));

      // 2 outlined buttons (replace + remove).
      expect(find.byType(OutlinedButton), findsNWidgets(2));
      expect(find.byIcon(Icons.upload_outlined), findsOneWidget);
      expect(find.byIcon(Icons.delete_outline), findsOneWidget);

      await tester.tap(find.byIcon(Icons.upload_outlined));
      await tester.pump();
      expect(replaces, 1);

      await tester.tap(find.byIcon(Icons.delete_outline));
      await tester.pump();
      expect(deletes, 1);
    });

    testWidgets('hides replace+remove buttons when callbacks are null',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProfileVideoSection(
            videoUrl: 'https://example.com/v.mp4',
          ),
        ),
      );
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.byType(OutlinedButton), findsNothing);
    });
  });
}
