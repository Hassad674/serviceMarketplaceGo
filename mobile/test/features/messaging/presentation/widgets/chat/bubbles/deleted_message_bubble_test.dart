import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/bubbles/deleted_message_bubble.dart';
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
  testWidgets('renders the block icon', (tester) async {
    await tester.pumpWidget(_wrap(const DeletedMessageBubble(isOwn: true)));
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.block), findsOneWidget);
  });

  testWidgets('renders the localized "deleted" label', (tester) async {
    await tester.pumpWidget(_wrap(const DeletedMessageBubble(isOwn: true)));
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.text(l10n.messagingDeleted), findsOneWidget);
  });

  testWidgets('aligns to the right when isOwn=true', (tester) async {
    await tester.pumpWidget(_wrap(const DeletedMessageBubble(isOwn: true)));
    await tester.pumpAndSettle();
    final align = tester.widget<Align>(find.byType(Align));
    expect(align.alignment, Alignment.centerRight);
  });

  testWidgets('aligns to the left when isOwn=false', (tester) async {
    await tester.pumpWidget(_wrap(const DeletedMessageBubble(isOwn: false)));
    await tester.pumpAndSettle();
    final align = tester.widget<Align>(find.byType(Align));
    expect(align.alignment, Alignment.centerLeft);
  });
}
