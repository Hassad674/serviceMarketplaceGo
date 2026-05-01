import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/presentation/widgets/freelance_video_card.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) => MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    );

void main() {
  testWidgets('renders empty state when no video and canEdit=true',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        FreelanceVideoCard(
          videoUrl: '',
          canEdit: true,
          onUpload: () {},
          onDelete: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.text(l10n.noVideo), findsOneWidget);
    expect(find.text(l10n.addVideo), findsOneWidget);
  });

  testWidgets('hides Add CTA when canEdit=false', (tester) async {
    await tester.pumpWidget(
      _wrap(
        FreelanceVideoCard(
          videoUrl: '',
          canEdit: false,
          onUpload: () {},
          onDelete: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.text(l10n.noVideo), findsOneWidget);
    expect(find.text(l10n.addVideo), findsNothing);
  });

  testWidgets('shows replace + delete when video set and canEdit=true',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        FreelanceVideoCard(
          videoUrl: 'https://example.com/video.mp4',
          canEdit: true,
          onUpload: () {},
          onDelete: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.text(l10n.replaceVideo), findsOneWidget);
    expect(find.byIcon(Icons.delete_outline), findsOneWidget);
  });

  testWidgets('hides replace + delete when canEdit=false', (tester) async {
    await tester.pumpWidget(
      _wrap(
        FreelanceVideoCard(
          videoUrl: 'https://example.com/video.mp4',
          canEdit: false,
          onUpload: () {},
          onDelete: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.text(l10n.replaceVideo), findsNothing);
    expect(find.byIcon(Icons.delete_outline), findsNothing);
  });

  testWidgets('Add CTA invokes onUpload', (tester) async {
    var uploadCalls = 0;
    await tester.pumpWidget(
      _wrap(
        FreelanceVideoCard(
          videoUrl: '',
          canEdit: true,
          onUpload: () => uploadCalls++,
          onDelete: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(Icons.add));
    await tester.pumpAndSettle();
    expect(uploadCalls, 1);
  });
}
