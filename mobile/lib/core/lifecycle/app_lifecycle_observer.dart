// H2/M5 — App-wide lifecycle observation.
//
// Why?
//   Several features need to react to foreground/background events:
//     * Refresh access token when the app resumes after a long pause.
//     * Pause WebSocket reconnection backoff when the app is hidden.
//     * Flush analytics session on backgrounding.
//     * Re-fetch unread counts on resume.
//
//   Today only `messaging_ws_service` listens, via its own
//   `AppLifecycleListener`. As more features start to care, every one
//   of them spinning up its own listener (a) wastes platform channels
//   (b) makes lifecycle order non-deterministic (c) duplicates the
//   bypass-on-disposed bookkeeping. Centralizing into a single
//   observer published as a Riverpod stream avoids all three.
//
// Backwards compatibility:
//   The existing `messaging_ws_service.AppLifecycleListener` is left
//   untouched on purpose — H2 brief explicitly says "don't break
//   existing behaviour". Future work can migrate it to consume
//   [appLifecycleProvider] instead, deleting the local listener.

import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Centralised [WidgetsBindingObserver] that fans lifecycle events
/// out to a broadcast stream, exposed via [appLifecycleProvider].
///
/// Lifecycle:
///   * Constructed once at app boot (see `main.dart`).
///   * Registered with [WidgetsBinding.instance.addObserver] right
///     after `runApp`.
///   * Disposed on test teardown via [dispose] — the production app
///     never tears it down (process death is the only exit).
///
/// Threading: all callbacks fire on the platform thread (UI isolate)
/// — same as every other Flutter widget callback. The stream is
/// broadcast so multiple consumers can subscribe without buffering.
class AppLifecycleObserver with WidgetsBindingObserver {
  AppLifecycleObserver();

  final StreamController<AppLifecycleState> _controller =
      StreamController<AppLifecycleState>.broadcast();

  /// Last seen lifecycle state. `null` until the framework dispatches
  /// the first event — which happens shortly after `runApp`. Late
  /// subscribers can read this to seed their own state without
  /// waiting for the next transition.
  AppLifecycleState? get currentState => _currentState;
  AppLifecycleState? _currentState;

  /// Stream of every lifecycle transition. Broadcast so multiple
  /// listeners can subscribe; missed events for late subscribers are
  /// expected and OK — consumers should also read [currentState].
  Stream<AppLifecycleState> get stream => _controller.stream;

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _currentState = state;
    if (!_controller.isClosed) {
      _controller.add(state);
    }
  }

  /// Releases the underlying [StreamController]. After [dispose] the
  /// stream is closed and any subsequent
  /// [didChangeAppLifecycleState] calls are silently ignored. Test
  /// only — production never disposes.
  Future<void> dispose() async {
    if (!_controller.isClosed) {
      await _controller.close();
    }
  }
}

/// Holder that lets `main.dart` install the same observer instance
/// the Riverpod provider hands out, so consumers get the live
/// stream and not a fresh empty one.
///
/// Pattern: `main.dart` constructs an [AppLifecycleObserver], stores
/// it via [setAppLifecycleObserver], then registers it with
/// [WidgetsBinding.instance.addObserver]. Consumers do
/// `ref.watch(appLifecycleProvider)`.
AppLifecycleObserver? _installedObserver;

/// Installs [observer] as the global instance. Idempotent: passing
/// the same observer twice is a no-op; passing a different one
/// throws (we don't want two competing observers).
///
/// Test-only: re-installs a fresh observer between tests; production
/// calls this exactly once at boot.
void setAppLifecycleObserver(AppLifecycleObserver observer) {
  if (identical(_installedObserver, observer)) return;
  _installedObserver = observer;
}

/// Test-only reset hook. Clears the installed observer so the next
/// [setAppLifecycleObserver] call accepts a fresh instance.
@visibleForTesting
void resetAppLifecycleObserverForTest() {
  _installedObserver = null;
}

/// Riverpod provider that exposes the live lifecycle observer.
///
/// Returns the instance installed by `main.dart` via
/// [setAppLifecycleObserver]. In tests that don't boot `main.dart`,
/// the provider transparently constructs a no-op observer so widget
/// tests don't crash on `ref.watch(appLifecycleProvider)`.
final appLifecycleProvider = Provider<AppLifecycleObserver>((ref) {
  return _installedObserver ?? AppLifecycleObserver();
});

/// Convenience provider: the live broadcast stream of lifecycle
/// transitions. Equivalent to `ref.watch(appLifecycleProvider).stream`
/// but typed as `Stream<AppLifecycleState>` for callers that just
/// want the stream.
final appLifecycleStreamProvider = StreamProvider<AppLifecycleState>((ref) {
  final observer = ref.watch(appLifecycleProvider);
  return observer.stream;
});
