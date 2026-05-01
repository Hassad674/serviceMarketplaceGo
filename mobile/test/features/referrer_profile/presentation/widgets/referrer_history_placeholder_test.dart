import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_profile/presentation/widgets/referrer_history_placeholder.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders the title', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const ReferrerHistoryPlaceholder(
          title: 'Project history',
          emptyLabel: 'no deals yet',
        ),
      ),
    );
    expect(find.text('Project history'), findsOneWidget);
  });

  testWidgets('renders the empty label', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const ReferrerHistoryPlaceholder(
          title: 'Project history',
          emptyLabel: 'Nothing here for now',
        ),
      ),
    );
    expect(find.text('Nothing here for now'), findsOneWidget);
  });

  testWidgets('renders the icons', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const ReferrerHistoryPlaceholder(
          title: 't',
          emptyLabel: 'l',
        ),
      ),
    );
    expect(find.byIcon(Icons.history_edu_outlined), findsOneWidget);
    expect(find.byIcon(Icons.info_outline), findsOneWidget);
  });

  testWidgets('always renders both rows even with very short labels',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const ReferrerHistoryPlaceholder(title: 'a', emptyLabel: 'b'),
      ),
    );
    expect(find.byType(Container), findsAtLeastNWidgets(2));
  });
}
