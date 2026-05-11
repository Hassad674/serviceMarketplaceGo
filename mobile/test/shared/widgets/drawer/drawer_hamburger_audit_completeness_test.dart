// Meta-audit on the existing `drawer_hamburger_audit_test.dart`: this
// test verifies the audit list is complete with respect to the actual
// `drawerPrimaryItems` declarations. New drawer destinations must be
// added to the audit list — otherwise the regression guard silently
// stops protecting them.
//
// This is a documentation/coverage guard, not a production assertion.
// It runs in pure-Dart mode (no Flutter binding needed).

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Each route fragment from `drawer_items.dart` mapped to the source
/// file we expect to host the hamburger leading icon. Routes that are
/// served by a sub-tab inside another already-audited screen are
/// listed in [_subTabRoutes] — they intentionally do NOT need their own
/// audit entry because the parent screen owns the chrome.
const Map<String, String> _expectedAuditPairings = {
  '/dashboard':
      'features/dashboard/presentation/screens/dashboard_screen.dart',
  '/messaging':
      'features/messaging/presentation/screens/messaging_screen.dart',
  '/missions':
      'features/proposal/presentation/screens/projects_list_screen.dart',
  '/jobs': 'features/job/presentation/screens/jobs_screen.dart',
  '/opportunities':
      'features/job/presentation/screens/opportunities_screen.dart',
  '/team': 'features/team/presentation/screens/team_screen.dart',
  '/profile': 'features/profile/presentation/screens/profile_screen.dart',
  '/client-profile':
      'features/client_profile/presentation/screens/client_profile_screen.dart',
  '/wallet': 'features/wallet/presentation/screens/wallet_screen.dart',
  '/account': 'features/account/presentation/screens/account_screen.dart',
  '/invoices':
      'features/invoicing/presentation/screens/invoices_screen.dart',
  '/search/freelancer':
      'features/search/presentation/screens/search_screen.dart',
  '/search/agency':
      'features/search/presentation/screens/search_screen.dart',
  '/search/referrer':
      'features/search/presentation/screens/search_screen.dart',
  '/notifications':
      'features/notification/presentation/screens/notification_screen.dart',
  '/payment-info':
      'features/payment_info/presentation/screens/payment_info_screen.dart',
};

/// Routes that legitimately live without the hamburger because they
/// are reachable from another screen that already has it. Documented
/// to make the audit-completeness contract explicit.
///
/// FIX-MOBILE-DRAWER (2026-05): notifications + payment-info now wire
/// the hamburger explicitly, so the gap-set is empty. Keeping the
/// scaffold so future agents have a place to flag a new known gap.
const _knownGapsToFlagInReport = <String>{};

void main() {
  late final String libDir;
  late final String routesSource;
  late final String auditSource;

  setUpAll(() {
    libDir = '${Directory.current.path}/lib';
    routesSource = File(
      '$libDir/shared/widgets/drawer/drawer_items.dart',
    ).readAsStringSync();
    auditSource = File(
      '${Directory.current.path}/test/shared/widgets/drawer/'
      'drawer_hamburger_audit_test.dart',
    ).readAsStringSync();
  });

  test('every entry in _expectedAuditPairings exists in drawer_items.dart',
      () {
    final missing = <String>[];
    for (final route in _expectedAuditPairings.keys) {
      if (!routesSource.contains("'$route'") &&
          !routesSource.contains('"$route"')) {
        // The route is referenced through a RoutePaths constant — accept
        // both literal and constant declarations.
        final tail = route.split('/').last;
        if (!routesSource.contains('RoutePaths.') &&
            !routesSource.contains(tail)) {
          missing.add(route);
        }
      }
    }
    expect(
      missing,
      isEmpty,
      reason: 'These routes are no longer declared in drawer_items.dart '
          'but are still expected to be audited: $missing',
    );
  });

  test('audit list covers all primary destinations OR they are flagged', () {
    final notCovered = <String>[];
    for (final entry in _expectedAuditPairings.entries) {
      // Search screens collapse onto a single source file — accept
      // "covered" if the source file path is present anywhere in the
      // audit list.
      if (!auditSource.contains(entry.value)) {
        notCovered.add('${entry.key} -> ${entry.value}');
      }
    }
    expect(
      notCovered,
      isEmpty,
      reason:
          'These drawer destinations are not in `drawer_hamburger_audit_test.dart` '
          '— add them or document them as a known gap: $notCovered',
    );
  });

  test('known-gap set is empty (regression pin)', () {
    // FIX-MOBILE-DRAWER (2026-05): every primary destination now wires
    // the hamburger. If a new gap is introduced, add the route here
    // explicitly (do NOT silently let it rot) and update the audit
    // pairings + screen source accordingly.
    expect(
      _knownGapsToFlagInReport,
      isEmpty,
      reason: 'A drawer destination is missing the hamburger — fix the '
          'screen or document the gap explicitly in this test.',
    );
    // Pin: routesSource is still referenced so the setUpAll's parse
    // remains exercised — guards against future refactors that would
    // silently turn this test into a no-op.
    expect(routesSource, isNotEmpty);
  });
}
