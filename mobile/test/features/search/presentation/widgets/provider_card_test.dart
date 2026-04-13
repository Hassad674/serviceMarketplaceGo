import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/search/presentation/widgets/provider_card.dart';
import 'package:marketplace_mobile/features/search/presentation/widgets/skills_display_widget.dart';

// =============================================================================
// Helper to pump a testable ProviderCard
// =============================================================================

Widget _buildTestableCard(Map<String, dynamic> profile) {
  return MaterialApp(
    theme: AppTheme.light,
    home: Scaffold(
      body: SingleChildScrollView(
        child: ProviderCard(profile: profile),
      ),
    ),
  );
}

// =============================================================================
// Tests
// =============================================================================

void main() {
  // ---------------------------------------------------------------------------
  // Display name resolution
  // ---------------------------------------------------------------------------

  group('ProviderCard display name', () {
    testWidgets('renders display_name when present', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Acme Agency',
        'role': 'agency',
      }));
      await tester.pumpAndSettle();

      expect(find.text('Acme Agency'), findsOneWidget);
    });

    testWidgets('renders first_name + last_name when no display_name',
        (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'first_name': 'John',
        'last_name': 'Doe',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.text('John Doe'), findsOneWidget);
    });

    testWidgets('renders "Unknown" when no name fields present',
        (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.text('Unknown'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // Title display
  // ---------------------------------------------------------------------------

  group('ProviderCard title', () {
    testWidgets('renders title when present', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'John',
        'title': 'Senior Developer',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.text('Senior Developer'), findsOneWidget);
    });

    testWidgets('renders "No title" when title is null', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'John',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.text('No title'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // Role badge
  // ---------------------------------------------------------------------------

  group('ProviderCard role badge', () {
    testWidgets('shows "Agency" for agency role', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Test Agency',
        'role': 'agency',
      }));
      await tester.pumpAndSettle();

      expect(find.text('Agency'), findsOneWidget);
    });

    testWidgets('shows "Enterprise" for enterprise role', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Test Corp',
        'role': 'enterprise',
      }));
      await tester.pumpAndSettle();

      expect(find.text('Enterprise'), findsOneWidget);
    });

    testWidgets('shows "Freelance" for provider role', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'John Doe',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.text('Freelance'), findsOneWidget);
    });

    testWidgets('shows raw role string for unknown role', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Test',
        'role': 'admin',
      }));
      await tester.pumpAndSettle();

      expect(find.text('admin'), findsOneWidget);
    });

    testWidgets('shows "Unknown" when role is null', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Test',
      }));
      await tester.pumpAndSettle();

      expect(find.text('Unknown'), findsAtLeastNWidgets(1));
    });
  });

  // ---------------------------------------------------------------------------
  // Role badge colors
  // ---------------------------------------------------------------------------

  group('ProviderCard role badge colors', () {
    testWidgets('agency badge uses blue color', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Test Agency',
        'role': 'agency',
      }));
      await tester.pumpAndSettle();

      // Find the badge text widget and verify its style color
      final badgeTextFinder = find.text('Agency');
      final textWidget = tester.widget<Text>(badgeTextFinder);
      expect(textWidget.style?.color, equals(const Color(0xFF2563EB)));
    });

    testWidgets('enterprise badge uses violet color', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Test Corp',
        'role': 'enterprise',
      }));
      await tester.pumpAndSettle();

      final badgeTextFinder = find.text('Enterprise');
      final textWidget = tester.widget<Text>(badgeTextFinder);
      expect(textWidget.style?.color, equals(const Color(0xFF8B5CF6)));
    });

    testWidgets('provider badge uses rose color', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'John Doe',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      final badgeTextFinder = find.text('Freelance');
      final textWidget = tester.widget<Text>(badgeTextFinder);
      expect(textWidget.style?.color, equals(const Color(0xFFF43F5E)));
    });

    testWidgets('unknown role badge uses slate color', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Test',
        'role': 'unknown_role',
      }));
      await tester.pumpAndSettle();

      final badgeTextFinder = find.text('unknown_role');
      final textWidget = tester.widget<Text>(badgeTextFinder);
      expect(textWidget.style?.color, equals(const Color(0xFF64748B)));
    });
  });

  // ---------------------------------------------------------------------------
  // Initials avatar (when no photo)
  // ---------------------------------------------------------------------------

  group('ProviderCard initials avatar', () {
    testWidgets('shows initials when no photo_url', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'John Doe',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      // Initials "JD" should be rendered inside a CircleAvatar
      expect(find.text('JD'), findsOneWidget);
    });

    testWidgets('shows single initial for single-word name', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Madonna',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.text('M'), findsOneWidget);
    });

    testWidgets('shows "?" for "Unknown" name', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.text('?'), findsOneWidget);
    });

    testWidgets('shows initials from first and last word for 3-word name',
        (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Jean Claude Van',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.text('JV'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // Card structure
  // ---------------------------------------------------------------------------

  group('ProviderCard structure', () {
    testWidgets('renders a GestureDetector for navigation', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Test',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.byType(GestureDetector), findsWidgets);
    });

    testWidgets('renders CircleAvatar', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'user_id': '123',
        'display_name': 'Test User',
        'role': 'provider',
      }));
      await tester.pumpAndSettle();

      expect(find.byType(CircleAvatar), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // Skills display — skills from the public profile payload
  // ---------------------------------------------------------------------------

  group('ProviderCard skills display', () {
    testWidgets('renders SkillsDisplayWidget when skills are provided',
        (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'organization_id': 'org-1',
        'name': 'Acme Agency',
        'org_type': 'agency',
        'skills': <Map<String, dynamic>>[
          <String, dynamic>{'skill_text': 'react', 'display_text': 'React'},
          <String, dynamic>{'skill_text': 'go', 'display_text': 'Go'},
        ],
      }));
      await tester.pumpAndSettle();

      expect(find.byType(SkillsDisplayWidget), findsOneWidget);
      expect(find.text('React'), findsOneWidget);
      expect(find.text('Go'), findsOneWidget);
    });

    testWidgets('does not render SkillsDisplayWidget when skills are empty',
        (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'organization_id': 'org-1',
        'name': 'Empty Org',
        'org_type': 'agency',
        'skills': <Map<String, dynamic>>[],
      }));
      await tester.pumpAndSettle();

      expect(find.byType(SkillsDisplayWidget), findsNothing);
    });

    testWidgets('does not render SkillsDisplayWidget when skills key absent',
        (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'organization_id': 'org-1',
        'name': 'No Skills Org',
        'org_type': 'agency',
      }));
      await tester.pumpAndSettle();

      expect(find.byType(SkillsDisplayWidget), findsNothing);
    });

    testWidgets('caps chips at 4 and shows +N overflow chip', (tester) async {
      await tester.pumpWidget(_buildTestableCard({
        'organization_id': 'org-1',
        'name': 'Busy Org',
        'org_type': 'agency',
        'skills': <Map<String, dynamic>>[
          <String, dynamic>{'skill_text': 'react', 'display_text': 'React'},
          <String, dynamic>{'skill_text': 'go', 'display_text': 'Go'},
          <String, dynamic>{'skill_text': 'rust', 'display_text': 'Rust'},
          <String, dynamic>{'skill_text': 'flutter', 'display_text': 'Flutter'},
          <String, dynamic>{'skill_text': 'docker', 'display_text': 'Docker'},
          <String, dynamic>{'skill_text': 'k8s', 'display_text': 'Kubernetes'},
        ],
      }));
      await tester.pumpAndSettle();

      expect(find.text('React'), findsOneWidget);
      expect(find.text('Flutter'), findsOneWidget);
      expect(find.text('Docker'), findsNothing);
      expect(find.text('Kubernetes'), findsNothing);
      expect(find.text('+2'), findsOneWidget);
    });
  });
}
