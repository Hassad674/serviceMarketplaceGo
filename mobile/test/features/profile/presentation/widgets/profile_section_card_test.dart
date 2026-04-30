import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/profile/presentation/widgets/profile_section_card.dart';

void main() {
  testWidgets('ProfileSectionCard renders title, icon and child',
      (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.light,
        home: const Scaffold(
          body: ProfileSectionCard(
            title: 'About',
            icon: Icons.info_outline,
            child: Text('Body content'),
          ),
        ),
      ),
    );

    expect(find.text('About'), findsOneWidget);
    expect(find.byIcon(Icons.info_outline), findsOneWidget);
    expect(find.text('Body content'), findsOneWidget);
  });
}
