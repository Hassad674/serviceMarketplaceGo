import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/search/presentation/widgets/public_profile/public_profile_states.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';
import 'package:shimmer/shimmer.dart';

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
  group('PublicProfileShimmer', () {
    testWidgets('renders a Shimmer wrapper', (tester) async {
      await tester.pumpWidget(_wrap(const PublicProfileShimmer()));
      expect(find.byType(Shimmer), findsOneWidget);
    });

    testWidgets('renders a CircleAvatar placeholder', (tester) async {
      await tester.pumpWidget(_wrap(const PublicProfileShimmer()));
      expect(find.byType(CircleAvatar), findsOneWidget);
    });
  });

  group('PublicProfileErrorState', () {
    testWidgets('shows the error icon and retry CTA', (tester) async {
      await tester.pumpWidget(
        _wrap(PublicProfileErrorState(onRetry: () {})),
      );
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.byIcon(Icons.error_outline), findsOneWidget);
      expect(find.text(l10n.couldNotLoadProfile), findsOneWidget);
      expect(find.text(l10n.retry), findsOneWidget);
    });

    testWidgets('retry button invokes the callback', (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        _wrap(PublicProfileErrorState(onRetry: () => calls++)),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(ElevatedButton));
      await tester.pumpAndSettle();
      expect(calls, 1);
    });

    testWidgets('renders the connection retry hint', (tester) async {
      await tester.pumpWidget(
        _wrap(PublicProfileErrorState(onRetry: () {})),
      );
      await tester.pumpAndSettle();
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.checkConnectionRetry), findsOneWidget);
    });
  });
}
