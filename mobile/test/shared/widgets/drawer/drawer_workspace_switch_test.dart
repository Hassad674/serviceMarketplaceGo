import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';
import 'package:marketplace_mobile/shared/widgets/drawer/drawer_workspace_switch.dart';
import 'package:shared_preferences/shared_preferences.dart';

Widget _wrap(Widget child) {
  final router = GoRouter(
    initialLocation: '/dashboard',
    routes: [
      GoRoute(
        path: '/dashboard',
        builder: (context, state) => Scaffold(
          // The widget calls Navigator.pop() to dismiss the drawer
          // before navigating. Provide a Drawer wrapper so that pop
          // has something to close.
          drawer: Drawer(
            // Disable automatic edge-of-screen sizing so the drawer
            // gets enough horizontal room for the workspace pill.
            width: 480,
            child: SafeArea(child: child),
          ),
          body: Builder(
            builder: (ctx) => TextButton(
              onPressed: () => Scaffold.of(ctx).openDrawer(),
              child: const Text('Open drawer'),
            ),
          ),
        ),
      ),
      GoRoute(
        path: '/dashboard/referrer',
        builder: (context, state) =>
            const Scaffold(body: Text('Referrer dashboard')),
      ),
    ],
  );
  return MaterialApp.router(
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en')],
    routerConfig: router,
  );
}

Future<AppLocalizations> _loadEn() async {
  return AppLocalizations.delegate.load(const Locale('en'));
}

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  setUpAll(() {
    TestWidgetsFlutterBinding.ensureInitialized();
  });

  Future<void> _setup(WidgetTester tester) async {
    // Widen the test viewport so the workspace pill (with French
    // label) does not overflow the default 800x600 surface.
    tester.view.physicalSize = const Size(1600, 1600);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });
  }

  Future<void> _openDrawer(WidgetTester tester) async {
    await tester.tap(find.text('Open drawer'));
    await tester.pumpAndSettle();
  }

  testWidgets('renders the "switch to referrer" label by default',
      (tester) async {
    await _setup(tester);
    final l10n = await _loadEn();
    await tester.pumpWidget(_wrap(DrawerWorkspaceSwitch(l10n: l10n)));
    await _openDrawer(tester);
    expect(find.text(l10n.drawerSwitchToReferrer), findsOneWidget);
    expect(find.byIcon(Icons.auto_awesome), findsOneWidget);
  });

  testWidgets('renders the "switch to freelance" label when in referrer mode',
      (tester) async {
    await _setup(tester);
    SharedPreferences.setMockInitialValues({
      'workspace_mode': 'referrer',
    });
    final l10n = await _loadEn();
    await tester.pumpWidget(_wrap(DrawerWorkspaceSwitch(l10n: l10n)));
    await _openDrawer(tester);
    expect(find.text(l10n.drawerSwitchToFreelance), findsOneWidget);
    expect(find.byIcon(Icons.swap_horiz), findsOneWidget);
  });

  testWidgets('persists referrer mode after tap', (tester) async {
    await _setup(tester);
    final l10n = await _loadEn();
    await tester.pumpWidget(_wrap(DrawerWorkspaceSwitch(l10n: l10n)));
    await _openDrawer(tester);

    await tester.tap(find.byType(DrawerWorkspaceSwitch));
    await tester.pumpAndSettle();

    final prefs = await SharedPreferences.getInstance();
    expect(prefs.getString('workspace_mode'), 'referrer');
  });
}
