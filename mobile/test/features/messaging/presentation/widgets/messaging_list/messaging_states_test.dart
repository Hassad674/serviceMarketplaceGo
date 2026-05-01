import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/messaging_list/messaging_states.dart';
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
  group('ConversationListShimmer', () {
    testWidgets('renders inside a Shimmer', (tester) async {
      await tester.pumpWidget(_wrap(const ConversationListShimmer()));
      expect(find.byType(Shimmer), findsOneWidget);
    });

    testWidgets('renders 6 placeholder rows', (tester) async {
      await tester.pumpWidget(_wrap(const ConversationListShimmer()));
      // 6 CircleAvatars confirms the row count.
      expect(find.byType(CircleAvatar), findsNWidgets(6));
    });
  });

  group('MessagingEmptyState', () {
    testWidgets('renders the message and chat icon', (tester) async {
      await tester.pumpWidget(
        _wrap(const MessagingEmptyState(message: 'No conversations')),
      );
      await tester.pumpAndSettle();
      expect(find.text('No conversations'), findsOneWidget);
      expect(find.byIcon(Icons.chat_outlined), findsOneWidget);
    });
  });

  group('MessagingErrorState', () {
    testWidgets('renders message + retry button', (tester) async {
      await tester.pumpWidget(
        _wrap(
          MessagingErrorState(message: 'failed', onRetry: () {}),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('failed'), findsOneWidget);
      final l10n = await AppLocalizations.delegate.load(const Locale('en'));
      expect(find.text(l10n.retry), findsOneWidget);
    });

    testWidgets('retry button invokes the callback', (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        _wrap(
          MessagingErrorState(message: 'fail', onRetry: () => calls++),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(ElevatedButton));
      await tester.pumpAndSettle();
      expect(calls, 1);
    });

    testWidgets('renders the error icon', (tester) async {
      await tester.pumpWidget(
        _wrap(
          MessagingErrorState(message: 'x', onRetry: () {}),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byIcon(Icons.error_outline), findsOneWidget);
    });
  });
}
