// Lint-as-perf-budget guard for the mobile app (Phase 4-O cat A).
//
// Each `prefer_const_constructors` info hit in `lib/` represents a
// missed const-canonicalisation opportunity — the widget gets
// reconstructed on every rebuild instead of being shared with
// itself across rebuilds. The fix is mechanical (`dart fix --apply
// --code=prefer_const_constructors`) and the savings are real but
// small per site, so we cap the count at the post-Phase-4-O budget.
//
// Going OVER this budget means a future change reintroduced a
// non-const widget or theme literal where one was avoidable; the
// fix is to either add `const` or document why the value can't be
// const (e.g. a runtime-derived parameter).

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  group('lib/ const-canonicalisation budget', () {
    test('prefer_const_constructors stays at 0 (post-Phase-4-O)', () {
      // We invoke `flutter analyze` to count remaining hits in
      // lib/. Anything > 0 means a regression from this commit's
      // baseline.
      final flutterBin = '/home/hassad/flutter/bin/flutter';
      final result = Process.runSync(
        flutterBin,
        ['analyze', 'lib'],
        workingDirectory: '.',
      );
      // Combine stdout + stderr; analyze prints to stdout.
      final out = '${result.stdout}\n${result.stderr}';
      final lines = out.split('\n').where(
            (l) => l.contains('prefer_const_constructors'),
          );
      final count = lines.length;
      expect(
        count,
        equals(0),
        reason: 'Phase 4-O eliminated every prefer_const_constructors '
            'hit in lib/. Found $count regression(s):\n'
            '${lines.join('\n')}',
      );
    });
  });
}
