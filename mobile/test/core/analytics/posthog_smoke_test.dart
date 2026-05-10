// PostHogService smoke test — validates that the analytics surface
// is safe to call even when no project key is configured.
//
// In a unit test the const `String.fromEnvironment('POSTHOG_PROJECT_KEY')`
// resolves to "", which makes [PostHogService.isEnabled] false. The
// service must therefore short-circuit every public method without
// touching the platform method channel — calling the underlying SDK
// without a setup() would crash the test runner.
//
// Tests:
//   1. isEnabled is false when no key is set at compile time.
//   2. initialize() is a no-op when disabled (no MissingPluginError).
//   3. capture / identify / group / reset are all no-ops, never throw.
//   4. The singleton stays a singleton across calls.

import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/analytics/posthog_service.dart';

void main() {
  group('PostHogService', () {
    test('isEnabled is false when POSTHOG_PROJECT_KEY is absent', () {
      expect(PostHogService.instance.isEnabled, isFalse);
    });

    test('initialize is a no-op when disabled — never throws', () async {
      // No platform method channel is registered in unit tests so
      // calling Posthog().setup() would throw MissingPluginError.
      // The service must short-circuit before that.
      await expectLater(PostHogService.instance.initialize(), completes);
    });

    test('capture is a no-op when disabled', () async {
      await expectLater(
        PostHogService.instance.capture(
          'mobile.smoke_test',
          properties: const {'foo': 'bar'},
        ),
        completes,
      );
    });

    test('identify is a no-op when disabled', () async {
      await expectLater(
        PostHogService.instance.identify(
          'user-123',
          properties: const {'role': 'agency'},
        ),
        completes,
      );
    });

    test('group is a no-op when disabled', () async {
      await expectLater(
        PostHogService.instance.group(
          'organization',
          'org-42',
          properties: const {'plan': 'premium'},
        ),
        completes,
      );
    });

    test('reset is a no-op when disabled', () async {
      await expectLater(PostHogService.instance.reset(), completes);
    });

    test('optIn / optOut are no-ops when disabled', () async {
      await expectLater(PostHogService.instance.optIn(), completes);
      await expectLater(PostHogService.instance.optOut(), completes);
    });

    test('singleton is stable — same instance across calls', () {
      final a = PostHogService.instance;
      final b = PostHogService.instance;
      expect(identical(a, b), isTrue);
    });
  });
}
