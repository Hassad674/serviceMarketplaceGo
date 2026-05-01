import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/presentation/widgets/freelance_section_card.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  testWidgets('renders title and child', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const FreelanceSectionCard(
          title: 'Section title',
          icon: Icons.info_outline,
          child: Text('Section body'),
        ),
      ),
    );
    expect(find.text('Section title'), findsOneWidget);
    expect(find.text('Section body'), findsOneWidget);
  });

  testWidgets('renders the icon', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const FreelanceSectionCard(
          title: 'About',
          icon: Icons.badge_outlined,
          child: SizedBox(),
        ),
      ),
    );
    expect(find.byIcon(Icons.badge_outlined), findsOneWidget);
  });

  testWidgets('renders trailing widget when provided', (tester) async {
    await tester.pumpWidget(
      _wrap(
        FreelanceSectionCard(
          title: 'About',
          icon: Icons.info_outline,
          trailing: IconButton(
            icon: const Icon(Icons.edit_outlined),
            onPressed: () {},
          ),
          child: const SizedBox(),
        ),
      ),
    );
    expect(find.byIcon(Icons.edit_outlined), findsOneWidget);
  });

  testWidgets('hides trailing area when not provided', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const FreelanceSectionCard(
          title: 'About',
          icon: Icons.info_outline,
          child: Text('body'),
        ),
      ),
    );
    expect(find.byType(IconButton), findsNothing);
  });
}
