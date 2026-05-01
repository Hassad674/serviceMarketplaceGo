import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/search/presentation/widgets/public_profile/public_profile_misc.dart';
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
  group('PublicProfileSpacerIfVisible', () {
    testWidgets('renders a 16px spacer when visible=true', (tester) async {
      await tester.pumpWidget(
        _wrap(const PublicProfileSpacerIfVisible(visible: true)),
      );
      // SizedBox.shrink() has no specific height; the visible variant
      // is a SizedBox with height 16.
      final sizedBoxes = tester.widgetList<SizedBox>(find.byType(SizedBox));
      expect(sizedBoxes.any((sb) => sb.height == 16), isTrue);
    });

    testWidgets('renders SizedBox.shrink when visible=false',
        (tester) async {
      await tester.pumpWidget(
        _wrap(const PublicProfileSpacerIfVisible(visible: false)),
      );
      final sizedBoxes = tester.widgetList<SizedBox>(find.byType(SizedBox));
      expect(sizedBoxes.any((sb) => sb.height == 16), isFalse);
    });
  });

  group('PublicProfileSendMessageButton', () {
    testWidgets('renders the chat icon when not sending', (tester) async {
      await tester.pumpWidget(
        _wrap(
          PublicProfileSendMessageButton(
            sending: false,
            onPressed: () {},
          ),
        ),
      );
      expect(find.byIcon(Icons.chat_outlined), findsOneWidget);
    });

    testWidgets('renders a spinner when sending=true', (tester) async {
      await tester.pumpWidget(
        _wrap(
          PublicProfileSendMessageButton(
            sending: true,
            onPressed: () {},
          ),
        ),
      );
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('disables the button while sending', (tester) async {
      await tester.pumpWidget(
        _wrap(
          PublicProfileSendMessageButton(
            sending: true,
            onPressed: () {},
          ),
        ),
      );
      final btn = tester.widget<ElevatedButton>(find.byType(ElevatedButton));
      expect(btn.onPressed, isNull);
    });

    testWidgets('invokes onPressed when tapped', (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        _wrap(
          PublicProfileSendMessageButton(
            sending: false,
            onPressed: () => calls++,
          ),
        ),
      );
      await tester.tap(find.byType(ElevatedButton));
      await tester.pumpAndSettle();
      expect(calls, 1);
    });
  });

  group('PublicProfileOrgTypeBadge', () {
    testWidgets('renders the agency label', (tester) async {
      await tester.pumpWidget(
        _wrap(const PublicProfileOrgTypeBadge(orgType: 'agency')),
      );
      expect(find.text('Agency'), findsOneWidget);
    });

    testWidgets('renders the enterprise label', (tester) async {
      await tester.pumpWidget(
        _wrap(const PublicProfileOrgTypeBadge(orgType: 'enterprise')),
      );
      expect(find.text('Enterprise'), findsOneWidget);
    });

    testWidgets('renders the freelance label for provider_personal',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const PublicProfileOrgTypeBadge(orgType: 'provider_personal'),
        ),
      );
      expect(find.text('Freelance'), findsOneWidget);
    });

    testWidgets('renders the raw type for unknown variants', (tester) async {
      await tester.pumpWidget(
        _wrap(const PublicProfileOrgTypeBadge(orgType: 'mystery')),
      );
      expect(find.text('mystery'), findsOneWidget);
    });
  });
}
