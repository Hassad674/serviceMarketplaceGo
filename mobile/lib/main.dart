import 'dart:async';

import 'package:firebase_core/firebase_core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'l10n/app_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/theme/app_theme.dart';
import 'core/theme/theme_provider.dart';
import 'core/router/app_router.dart';
import 'features/call/presentation/widgets/call_event_listener.dart';

/// Whether Firebase has finished initializing. Exposed so the FCM
/// service (and any future Firebase-dependent feature) can wait on
/// readiness without blocking the splash screen.
///
/// `Future<void>` rather than a `bool` so callers can `await`
/// without polling. Resolves at most once per process — the
/// underlying [Firebase.initializeApp] is idempotent.
Future<void> firebaseReady = Future<void>.value();

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Lock to portrait on phones.
  SystemChrome.setPreferredOrientations([
    DeviceOrientation.portraitUp,
    DeviceOrientation.portraitDown,
  ]);

  // Transparent status bar on Android.
  SystemChrome.setSystemUIOverlayStyle(
    const SystemUiOverlayStyle(
      statusBarColor: Colors.transparent,
      statusBarIconBrightness: Brightness.dark,
      statusBarBrightness: Brightness.light,
    ),
  );

  // Defer Firebase initialization off the cold-start critical path.
  // Anything Firebase-dependent (FCM, Crashlytics) awaits
  // [firebaseReady] before touching the SDK. Saves 200-500 ms TTI
  // on iOS where Firebase setup is the slowest non-Flutter step.
  firebaseReady = _initFirebase();

  runApp(
    const ProviderScope(
      child: MarketplaceApp(),
    ),
  );
}

/// Initializes Firebase off the splash-blocking path. Errors are
/// swallowed and logged: a Firebase outage must not crash the app
/// — push notifications simply degrade until the next cold start.
Future<void> _initFirebase() async {
  try {
    await Firebase.initializeApp();
  } catch (e, st) {
    if (kDebugMode) {
      debugPrint('Firebase.initializeApp failed: $e\n$st');
    }
  }
}

class MarketplaceApp extends ConsumerWidget {
  const MarketplaceApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(appRouterProvider);
    final themeMode = ref.watch(themeModeProvider);

    return MaterialApp.router(
      title: 'Marketplace Service',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      themeMode: themeMode,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [
        Locale('en'),
        Locale('fr'),
      ],
      routerConfig: router,
      builder: (context, child) {
        return CallEventListener(child: child ?? const SizedBox.shrink());
      },
    );
  }
}
