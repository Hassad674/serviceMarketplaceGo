import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/shared/widgets/drawer/drawer_items.dart';

void main() {
  group('drawerPrimaryItems', () {
    test('contains the dashboard entry available to all roles', () {
      final dashboard = drawerPrimaryItems.firstWhere(
        (item) => item.labelKey == 'drawerDashboard',
      );
      expect(dashboard.roles, ['agency', 'enterprise', 'provider']);
    });

    test('jobs entry is gated to enterprise + agency', () {
      final jobs = drawerPrimaryItems.firstWhere(
        (item) => item.labelKey == 'drawerJobs',
      );
      expect(jobs.roles, containsAll(['enterprise', 'agency']));
      expect(jobs.roles.contains('provider'), isFalse);
    });

    test('opportunities entry is gated to provider + agency', () {
      final opp = drawerPrimaryItems.firstWhere(
        (item) => item.labelKey == 'drawerOpportunities',
      );
      expect(opp.roles, containsAll(['provider', 'agency']));
      expect(opp.roles.contains('enterprise'), isFalse);
    });

    test('client profile entry has both role+orgType gates', () {
      final clientProfile = drawerPrimaryItems.firstWhere(
        (item) => item.labelKey == 'navClientProfile',
      );
      expect(clientProfile.roles, containsAll(['agency', 'enterprise']));
      expect(clientProfile.orgTypes, containsAll(['agency', 'enterprise']));
    });

    test('invoicing entry is restricted to provider + agency', () {
      final invoices = drawerPrimaryItems.firstWhere(
        (item) => item.labelKey == 'drawerInvoices',
      );
      expect(invoices.roles, ['agency', 'provider']);
    });
  });

  group('drawerSearchItems', () {
    test('agency search is enterprise-only', () {
      final agencies = drawerSearchItems.firstWhere(
        (item) => item.labelKey == 'drawerFindAgencies',
      );
      expect(agencies.roles, ['enterprise']);
    });

    test('freelancer search is for buyers (agency + enterprise)', () {
      final freelancers = drawerSearchItems.firstWhere(
        (item) => item.labelKey == 'drawerFindFreelancers',
      );
      expect(freelancers.roles, ['agency', 'enterprise']);
    });

    test('referrer search is for agency + enterprise', () {
      final referrers = drawerSearchItems.firstWhere(
        (item) => item.labelKey == 'drawerFindReferrers',
      );
      expect(referrers.roles, ['agency', 'enterprise']);
    });
  });

  group('drawerRoleBadgeColors', () {
    testWidgets('contains entries for the three primary roles', (tester) async {
      late Map<String, (Color, Color)> colors;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.light,
          home: Builder(
            builder: (context) {
              colors = drawerRoleBadgeColors(context);
              return const SizedBox.shrink();
            },
          ),
        ),
      );
      expect(colors.containsKey('agency'), isTrue);
      expect(colors.containsKey('enterprise'), isTrue);
      expect(colors.containsKey('provider'), isTrue);
    });
  });
}
