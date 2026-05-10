// PostHog analytics wrapper for the Flutter app.
//
// Mirrors the web/backend pattern: every feature dials a single
// surface (PostHogService) instead of importing posthog_flutter
// directly. This keeps the adapter swappable (a future Mixpanel
// migration changes ONE file) and makes the call sites mockable in
// widget tests via a Riverpod override.
//
// Init lifecycle:
//   1. main.dart calls PostHogService.instance.initialize() AFTER
//      WidgetsFlutterBinding.ensureInitialized(). Safe to await on
//      the splash-blocking path — the SDK boot is ~50 ms and is
//      cheaper than holding a stale identify until first use.
//   2. The auth state notifier calls identify() on login / reset()
//      on logout. Reset always runs locally, even if the SDK is
//      disabled (no key) — keeps the call site shape simple.
//   3. Feature widgets call capture() on user actions (search,
//      apply, message sent…). The wrapper no-ops gracefully when
//      analytics is disabled.

import 'package:flutter/foundation.dart';
import 'package:posthog_flutter/posthog_flutter.dart';

/// Reads the PostHog key from the dart-define provided at build time:
///
///   flutter run --dart-define=POSTHOG_PROJECT_KEY=phc_xxx
///   flutter build apk --dart-define=POSTHOG_PROJECT_KEY=phc_xxx
///
/// When the key is empty the service flips into a no-op mode — every
/// public method short-circuits without invoking the SDK.
const String _projectKey = String.fromEnvironment('POSTHOG_PROJECT_KEY');

/// Same convention as web: defaults to the EU host so Quebec / EU
/// users keep their personal data inside the EU region.
const String _host = String.fromEnvironment(
  'POSTHOG_HOST',
  defaultValue: 'https://eu.posthog.com',
);

/// Singleton-style facade over posthog_flutter. We use a static
/// instance because Riverpod providers can wrap it for testing, but
/// the underlying SDK is already a singleton at the platform layer
/// (iOS/Android plugin) so a second instance would be a footgun.
class PostHogService {
  PostHogService._();
  static final PostHogService instance = PostHogService._();

  bool _initialized = false;

  /// Whether the SDK was wired with a real project key. Lets call
  /// sites short-circuit without paying the method-channel hop when
  /// analytics is intentionally disabled in dev builds.
  bool get isEnabled => _projectKey.isNotEmpty;

  /// Idempotent init. Safe to call multiple times; subsequent calls
  /// are no-ops. Catches setup failures so a misconfigured key never
  /// crashes the app boot — analytics is observability, not load-
  /// bearing.
  Future<void> initialize() async {
    if (_initialized || !isEnabled) return;
    try {
      final config = PostHogConfig(_projectKey)
        ..host = _host
        ..captureApplicationLifecycleEvents = true
        ..debug = kDebugMode
        // RGPD: do not capture until consent is granted. Until the
        // user makes a choice the SDK queues nothing.
        ..optOut = true;
      await Posthog().setup(config);
      _initialized = true;
      if (kDebugMode) {
        debugPrint('[PostHog] initialized at $_host');
      }
    } catch (e, st) {
      if (kDebugMode) {
        debugPrint('[PostHog] init failed: $e\n$st');
      }
    }
  }

  /// Flip into capturing-on mode. Mobile equivalent of clicking
  /// "Accepter" in the web cookie banner. Persistence of the choice
  /// is the caller's responsibility (typically [SharedPreferences]).
  Future<void> optIn() async {
    if (!_initialized) return;
    try {
      await Posthog().enable();
    } catch (_) {
      // best-effort
    }
  }

  /// Flip into capturing-off mode.
  Future<void> optOut() async {
    if (!_initialized) return;
    try {
      await Posthog().disable();
    } catch (_) {
      // best-effort
    }
  }

  /// Capture a custom event. Drops silently when the SDK is not
  /// initialised so call sites can stay free of null checks.
  Future<void> capture(
    String eventName, {
    Map<String, Object>? properties,
  }) async {
    if (!_initialized) return;
    try {
      await Posthog().capture(eventName: eventName, properties: properties);
    } catch (e) {
      if (kDebugMode) {
        debugPrint('[PostHog] capture($eventName) failed: $e');
      }
    }
  }

  /// Attach profile attributes to the current distinct id.
  Future<void> identify(
    String userId, {
    Map<String, Object>? properties,
  }) async {
    if (!_initialized) return;
    try {
      await Posthog().identify(userId: userId, userProperties: properties);
    } catch (e) {
      if (kDebugMode) {
        debugPrint('[PostHog] identify($userId) failed: $e');
      }
    }
  }

  /// Attach attributes to a group (typically organization). Mirrors
  /// the backend + web wiring so dashboards filter consistently.
  Future<void> group(
    String groupType,
    String groupKey, {
    Map<String, Object>? properties,
  }) async {
    if (!_initialized) return;
    try {
      await Posthog().group(
        groupType: groupType,
        groupKey: groupKey,
        groupProperties: properties,
      );
    } catch (e) {
      if (kDebugMode) {
        debugPrint('[PostHog] group($groupKey) failed: $e');
      }
    }
  }

  /// Reset on logout — clears the distinct id so the next anonymous
  /// session does not pollute the previous user's timeline.
  Future<void> reset() async {
    if (!_initialized) return;
    try {
      await Posthog().reset();
    } catch (_) {
      // best-effort
    }
  }
}
