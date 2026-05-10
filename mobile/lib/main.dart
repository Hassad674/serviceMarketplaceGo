import 'dart:async';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_crashlytics/firebase_crashlytics.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'l10n/app_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/analytics/posthog_service.dart';
import 'core/lifecycle/app_lifecycle_observer.dart';
import 'core/theme/app_theme.dart';
import 'core/theme/theme_provider.dart';
import 'core/router/app_router.dart';
import 'features/call/presentation/widgets/call_event_listener.dart';

/// Whether Firebase has finished initializing. Exposed so the FCM
/// service (and any future Firebase-dependent feature, including
/// Crashlytics) can wait on readiness without blocking the splash
/// screen.
///
/// `Future<void>` rather than a `bool` so callers can `await`
/// without polling. Resolves at most once per process — the
/// underlying [Firebase.initializeApp] is idempotent.
Future<void> firebaseReady = Future<void>.value();

Future<void> main() async {
  // H2/M4: every uncaught error path goes through Crashlytics in
  // release mode and through `debugPrint` in debug mode. Three
  // independent error surfaces are wired:
  //
  //   1. `FlutterError.onError`              — framework errors
  //      (build failures, layout asserts, gesture exceptions).
  //   2. `PlatformDispatcher.instance.onError` — uncaught async
  //      errors that bubble out of a zone (Dart 2.18+ replacement
  //      for the old `Isolate.current.addErrorListener` pattern).
  //   3. `runZonedGuarded`                    — synchronous + async
  //      errors that escape `runApp`'s own zone, e.g. raised inside
  //      a `Future` whose error nobody listens to.
  //
  // All three forward to the same sink so we never lose a crash
  // regardless of which surface fires first. The sink is gated on
  // `kReleaseMode`: in debug we just log so a noisy local crash
  // does not pollute Crashlytics' release dashboard.
  await runZonedGuarded<Future<void>>(() async {
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

    // Initialise PostHog analytics in the same off-critical-path
    // pattern. The SDK setup hits the platform method channel for a
    // ~50 ms init; we fire-and-forget so the splash never blocks on
    // it. Analytics failures are swallowed — observability must
    // never crash the boot.
    unawaited(PostHogService.instance.initialize());

    // Wire framework + platform error surfaces. Both must be set
    // BEFORE runApp so the very first frame's errors are caught.
    FlutterError.onError = _onFlutterError;
    PlatformDispatcher.instance.onError = _onPlatformError;

    // H2/M5: install the global lifecycle observer so features can
    // subscribe via `ref.watch(appLifecycleProvider)` instead of
    // each feature setting up its own AppLifecycleListener. The
    // observer is registered with WidgetsBinding right after the
    // binding is initialised so early lifecycle transitions are
    // captured.
    final lifecycleObserver = AppLifecycleObserver();
    setAppLifecycleObserver(lifecycleObserver);
    WidgetsBinding.instance.addObserver(lifecycleObserver);

    runApp(
      const ProviderScope(
        child: MarketplaceApp(),
      ),
    );
  }, _onZoneError,);
}

/// Initializes Firebase off the splash-blocking path. Errors are
/// swallowed and logged: a Firebase outage must not crash the app
/// — push notifications and crash reporting simply degrade until
/// the next cold start.
///
/// Once Firebase is ready, opt the Crashlytics SDK in (or out, in
/// debug) for crash collection.
Future<void> _initFirebase() async {
  try {
    await Firebase.initializeApp();
    // Crashlytics collection mirrors build mode: enabled in release,
    // disabled in debug. We still wire `recordError` so devs see the
    // Crashlytics-bound errors in the console via the [_logCrash]
    // sink, but uploads are gated.
    await FirebaseCrashlytics.instance
        .setCrashlyticsCollectionEnabled(kReleaseMode);
  } catch (e, st) {
    if (kDebugMode) {
      debugPrint('Firebase.initializeApp failed: $e\n$st');
    }
  }
}

/// Sink for an uncaught Flutter framework error.
///
/// Marked `fatal: true` because by the time the framework surfaces an
/// error here, the affected widget tree is in an undefined state —
/// continuing to render against it can cause cascading failures, so
/// the user-visible outcome is effectively a crash even if the
/// process keeps running.
void _onFlutterError(FlutterErrorDetails details) {
  if (kReleaseMode) {
    FirebaseCrashlytics.instance.recordFlutterFatalError(details);
  } else {
    // Use the default presenter so devs see the familiar red banner
    // and a stack in the console.
    FlutterError.presentError(details);
    debugPrint('FlutterError.onError: ${details.exceptionAsString()}');
  }
}

/// Sink for an uncaught platform-level async error (Dart 2.18+).
///
/// Returning `true` tells the engine the error has been handled and
/// suppresses the default print-to-stderr behaviour, which would
/// otherwise interleave noisily with our structured Crashlytics
/// upload.
bool _onPlatformError(Object error, StackTrace stack) {
  if (kReleaseMode) {
    FirebaseCrashlytics.instance.recordError(error, stack, fatal: true);
  } else {
    debugPrint('PlatformDispatcher.onError: $error\n$stack');
  }
  return true;
}

/// Sink for an uncaught error that escaped [runZonedGuarded] — this
/// catches anything `FlutterError` and `PlatformDispatcher` missed,
/// typically `Future`s with no error handler attached.
void _onZoneError(Object error, StackTrace stack) {
  if (kReleaseMode) {
    FirebaseCrashlytics.instance.recordError(error, stack, fatal: true);
  } else {
    debugPrint('runZonedGuarded onError: $error\n$stack');
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
