import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/profile/presentation/widgets/profile_header_card.dart';

Widget _wrap(Widget child) => MaterialApp(
      theme: AppTheme.light,
      home: Scaffold(body: child),
    );

void main() {
  group('ProfileHeaderCard', () {
    testWidgets('renders displayName, email, initials and role badge',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProfileHeaderCard(
            initials: 'JD',
            displayName: 'Jane Doe',
            email: 'jane@example.com',
            role: 'agency',
          ),
        ),
      );
      expect(find.text('Jane Doe'), findsOneWidget);
      expect(find.text('jane@example.com'), findsOneWidget);
      expect(find.text('JD'), findsOneWidget);
      expect(find.text('Agency'), findsOneWidget);
    });

    testWidgets('falls back to "User" when displayName is empty',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProfileHeaderCard(
            initials: '?',
            displayName: '',
            email: '',
            role: null,
          ),
        ),
      );
      expect(find.text('User'), findsOneWidget);
    });

    testWidgets('camera badge appears when onPhotoTap provided', (tester) async {
      var taps = 0;
      await tester.pumpWidget(
        _wrap(
          ProfileHeaderCard(
            initials: 'JD',
            displayName: 'Jane Doe',
            email: 'jane@example.com',
            role: 'agency',
            onPhotoTap: () => taps++,
          ),
        ),
      );
      expect(find.byIcon(Icons.camera_alt), findsOneWidget);

      await tester.tap(find.byType(GestureDetector).first);
      await tester.pump();
      expect(taps, 1);
    });

    testWidgets('camera badge hidden when onPhotoTap is null', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const ProfileHeaderCard(
            initials: 'JD',
            displayName: 'Jane Doe',
            email: 'jane@example.com',
            role: 'agency',
          ),
        ),
      );
      expect(find.byIcon(Icons.camera_alt), findsNothing);
    });
  });

  group('ProfileAvatar', () {
    testWidgets('shows initials when no photo URL', (tester) async {
      await tester.pumpWidget(
        _wrap(const ProfileAvatar(initials: 'AB')),
      );
      expect(find.text('AB'), findsOneWidget);
      expect(find.byIcon(Icons.camera_alt), findsNothing);
    });

    testWidgets('camera badge appears when onTap provided', (tester) async {
      await tester.pumpWidget(
        _wrap(ProfileAvatar(initials: 'AB', onTap: () {})),
      );
      expect(find.byIcon(Icons.camera_alt), findsOneWidget);
    });
  });

  group('ProfileRoleBadge', () {
    testWidgets('agency role renders "Agency"', (tester) async {
      await tester.pumpWidget(_wrap(const ProfileRoleBadge(role: 'agency')));
      expect(find.text('Agency'), findsOneWidget);
    });

    testWidgets('enterprise role renders "Enterprise"', (tester) async {
      await tester.pumpWidget(
        _wrap(const ProfileRoleBadge(role: 'enterprise')),
      );
      expect(find.text('Enterprise'), findsOneWidget);
    });

    testWidgets('provider role renders "Freelance"', (tester) async {
      await tester.pumpWidget(
        _wrap(const ProfileRoleBadge(role: 'provider')),
      );
      expect(find.text('Freelance'), findsOneWidget);
    });

    testWidgets('null role renders "Unknown"', (tester) async {
      await tester.pumpWidget(_wrap(const ProfileRoleBadge(role: null)));
      expect(find.text('Unknown'), findsOneWidget);
    });

    testWidgets('unknown role string renders the raw value', (tester) async {
      await tester.pumpWidget(
        _wrap(const ProfileRoleBadge(role: 'admin')),
      );
      expect(find.text('admin'), findsOneWidget);
    });
  });
}
