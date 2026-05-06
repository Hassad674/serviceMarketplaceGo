// H2/M5 — Unit tests for the global app lifecycle observer.
//
// The observer is a thin wrapper over [WidgetsBindingObserver] that
// rebroadcasts events through a Riverpod-friendly stream. Tests
// pin the four contract guarantees:
//
//   1. didChangeAppLifecycleState updates currentState.
//   2. didChangeAppLifecycleState pushes onto the stream.
//   3. dispose closes the stream and silences future events.
//   4. setAppLifecycleObserver wires the Riverpod provider to the
//      live instance (not a fresh empty one).

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/lifecycle/app_lifecycle_observer.dart';

void main() {
  group('AppLifecycleObserver', () {
    test('initial currentState is null until first event', () {
      final observer = AppLifecycleObserver();
      addTearDown(observer.dispose);
      expect(observer.currentState, isNull);
    });

    test('didChangeAppLifecycleState updates currentState', () {
      final observer = AppLifecycleObserver();
      addTearDown(observer.dispose);

      observer.didChangeAppLifecycleState(AppLifecycleState.paused);
      expect(observer.currentState, equals(AppLifecycleState.paused));

      observer.didChangeAppLifecycleState(AppLifecycleState.resumed);
      expect(observer.currentState, equals(AppLifecycleState.resumed));
    });

    test('didChangeAppLifecycleState pushes events onto the stream',
        () async {
      final observer = AppLifecycleObserver();
      addTearDown(observer.dispose);

      // Use expectLater so the matcher subscribes BEFORE the events
      // fire — broadcast streams drop events delivered before
      // anyone is listening.
      final fut = expectLater(
        observer.stream,
        emitsInOrder([
          AppLifecycleState.paused,
          AppLifecycleState.resumed,
          AppLifecycleState.inactive,
        ]),
      );

      // Schedule the events on the next microtask so the matcher's
      // subscription is wired first.
      Future<void>.microtask(() {
        observer.didChangeAppLifecycleState(AppLifecycleState.paused);
        observer.didChangeAppLifecycleState(AppLifecycleState.resumed);
        observer.didChangeAppLifecycleState(AppLifecycleState.inactive);
      });

      await fut;
    });

    test('multiple subscribers each receive the same events', () async {
      final observer = AppLifecycleObserver();
      addTearDown(observer.dispose);

      final fut1 = observer.stream.first;
      final fut2 = observer.stream.first;

      Future<void>.microtask(() {
        observer.didChangeAppLifecycleState(AppLifecycleState.hidden);
      });

      expect(await fut1, equals(AppLifecycleState.hidden));
      expect(await fut2, equals(AppLifecycleState.hidden));
    });

    test('dispose closes the stream and silences subsequent events',
        () async {
      final observer = AppLifecycleObserver();

      // Capture all events before dispose.
      final events = <AppLifecycleState>[];
      final sub = observer.stream.listen(events.add);

      observer.didChangeAppLifecycleState(AppLifecycleState.paused);
      // Yield so the controller drains.
      await Future<void>.delayed(Duration.zero);
      expect(events, equals([AppLifecycleState.paused]));

      // Dispose — stream should close.
      await observer.dispose();

      // Subsequent events must NOT throw (we silently swallow once
      // closed) and must NOT reach listeners.
      observer.didChangeAppLifecycleState(AppLifecycleState.resumed);
      await Future<void>.delayed(Duration.zero);
      expect(events, equals([AppLifecycleState.paused]));

      await sub.cancel();
    });

    test('dispose is idempotent — calling twice does not throw', () async {
      final observer = AppLifecycleObserver();
      await observer.dispose();
      // Second dispose must be a no-op.
      await observer.dispose();
    });
  });

  group('appLifecycleProvider wiring', () {
    setUp(resetAppLifecycleObserverForTest);
    tearDown(resetAppLifecycleObserverForTest);

    test('without setAppLifecycleObserver: provider returns a fresh '
        'no-op observer', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final observer = container.read(appLifecycleProvider);
      // Fresh observer — currentState starts null.
      expect(observer.currentState, isNull);
    });

    test('after setAppLifecycleObserver: provider returns the SAME '
        'instance the app boot installed', () {
      final boot = AppLifecycleObserver();
      addTearDown(boot.dispose);
      setAppLifecycleObserver(boot);

      final container = ProviderContainer();
      addTearDown(container.dispose);

      // Two reads must return the same instance — and that instance
      // must be the one main.dart installed, otherwise consumers
      // would never see the live lifecycle events.
      final fromProvider1 = container.read(appLifecycleProvider);
      final fromProvider2 = container.read(appLifecycleProvider);
      expect(identical(fromProvider1, boot), isTrue,
          reason: 'provider must hand out the installed observer');
      expect(identical(fromProvider1, fromProvider2), isTrue,
          reason: 'provider must be a singleton per container');
    });

    test('setAppLifecycleObserver with the same instance is idempotent', () {
      final boot = AppLifecycleObserver();
      addTearDown(boot.dispose);
      setAppLifecycleObserver(boot);
      // Calling twice with the same instance must be a safe no-op.
      setAppLifecycleObserver(boot);

      final container = ProviderContainer();
      addTearDown(container.dispose);
      expect(identical(container.read(appLifecycleProvider), boot), isTrue);
    });

    test('appLifecycleStreamProvider exposes the observer stream', () async {
      final boot = AppLifecycleObserver();
      addTearDown(boot.dispose);
      setAppLifecycleObserver(boot);

      final container = ProviderContainer();
      addTearDown(container.dispose);

      // Subscribe via the stream provider; emit one event; assert
      // the AsyncValue resolves with the right state.
      final sub = container.listen<AsyncValue<AppLifecycleState>>(
        appLifecycleStreamProvider,
        (_, __) {},
      );
      // Initial state is loading (no event yet).
      expect(sub.read(), isA<AsyncLoading<AppLifecycleState>>());

      boot.didChangeAppLifecycleState(AppLifecycleState.paused);
      // Pump to give the StreamProvider a chance to drain.
      await Future<void>.delayed(Duration.zero);
      expect(sub.read().value, equals(AppLifecycleState.paused));
    });
  });
}
