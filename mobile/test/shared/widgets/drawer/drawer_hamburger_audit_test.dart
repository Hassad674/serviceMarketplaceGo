// Source-level guard: every primary destination reachable from the
// `AppDrawer` MUST surface the drawer-opening hamburger so the user
// never gets stuck on a screen with no menu access. The user-reported
// bug was that "Équipe" and "Mes candidatures" were missing the
// leading icon — this test makes the regression cheap to catch.

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  // The mapping pins each drawer-reachable screen to a fragment we
  // expect to find in its source. The fragment must match the Soleil
  // v2 hamburger pattern used elsewhere (see `jobs_screen.dart`).
  const screensWithDrawerEntry = {
    'features/dashboard/presentation/screens/dashboard_screen.dart':
        'openShellDrawer',
    'features/messaging/presentation/screens/messaging_screen.dart':
        'openShellDrawer',
    'features/proposal/presentation/screens/projects_list_screen.dart':
        'openShellDrawer',
    'features/job/presentation/screens/jobs_screen.dart': 'openShellDrawer',
    'features/job/presentation/screens/opportunities_screen.dart':
        'openShellDrawer',
    'features/team/presentation/screens/team_screen.dart': 'openShellDrawer',
    'features/profile/presentation/screens/profile_screen.dart':
        'openShellDrawer',
    'features/freelance_profile/presentation/screens/freelance_profile_screen.dart':
        'openShellDrawer',
    'features/client_profile/presentation/screens/client_profile_screen.dart':
        'openShellDrawer',
    'features/wallet/presentation/screens/wallet_screen.dart':
        'openShellDrawer',
    'features/account/presentation/screens/account_screen.dart':
        'openShellDrawer',
    'features/invoicing/presentation/screens/invoices_screen.dart':
        'openShellDrawer',
    'features/search/presentation/screens/search_screen.dart':
        'openShellDrawer',
    'features/notification/presentation/screens/notification_screen.dart':
        'openShellDrawer',
    'features/payment_info/presentation/screens/payment_info_screen.dart':
        'openShellDrawer',
  };

  test('every drawer destination wires the hamburger leading icon',
      () async {
    // Resolve the project root from the test runner cwd:
    // `flutter test` is invoked at `mobile/`, so `lib/` is one level
    // below. Manual `/`-joining keeps the test free of an extra
    // `path` dependency.
    final libDir = '${Directory.current.path}/lib';

    final missing = <String>[];
    for (final entry in screensWithDrawerEntry.entries) {
      final file = File('$libDir/${entry.key}');
      if (!file.existsSync()) {
        missing.add('${entry.key} (file does not exist)');
        continue;
      }
      final source = file.readAsStringSync();
      // Strip line comments + block comments before searching so the
      // marker ALWAYS reflects an actual call site.
      final code = source
          .replaceAll(RegExp(r'//.*\n'), '')
          .replaceAll(RegExp(r'/\*[\s\S]*?\*/'), '');
      if (!code.contains(entry.value)) {
        missing.add(entry.key);
      }
    }
    expect(
      missing,
      isEmpty,
      reason: 'These primary destinations no longer wire the drawer '
          'hamburger; users would get stuck without menu access:\n'
          '${missing.join('\n')}',
    );
  });
}
