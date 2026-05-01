// Cat E (animation jank) source contract for the chat input bar.
//
// PERF-M-08: heavy animation children must be wrapped in
// RepaintBoundary so the 60fps repaint stays inside the animated
// layer instead of invalidating the rest of the bar chrome.
//
// The two animation sites in `MessageInputBar` are:
//   1. The pulsing red dot during voice recording (60fps Opacity).
//   2. The timer Text that ticks every second.
//
// Both should now sit inside their own RepaintBoundary.

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  group('MessageInputBar repaint boundary contract', () {
    final src = File(
      'lib/features/messaging/presentation/widgets/chat/message_input_bar.dart',
    ).readAsStringSync();

    test('pulsing dot is wrapped in a RepaintBoundary', () {
      final pulseIdx = src.indexOf('AnimatedBuilder');
      expect(pulseIdx, greaterThan(0));
      // Walk back ~150 chars and confirm RepaintBoundary appears
      // in the slice just before the AnimatedBuilder.
      final start = (pulseIdx - 150).clamp(0, pulseIdx);
      final slice = src.substring(start, pulseIdx);
      expect(
        slice.contains('RepaintBoundary'),
        isTrue,
        reason: 'The 60fps Opacity tween must be in its own '
            'RepaintBoundary so it doesn\'t invalidate the delete '
            'button / timer / send button raster layers',
      );
    });

    test('recording timer Text is wrapped in a RepaintBoundary', () {
      final timerIdx = src.indexOf('_formatDuration(_recordingDuration)');
      expect(timerIdx, greaterThan(0));
      // Look at the lines preceding the timer for RepaintBoundary.
      final start = (timerIdx - 200).clamp(0, timerIdx);
      final slice = src.substring(start, timerIdx);
      expect(
        slice.contains('RepaintBoundary'),
        isTrue,
        reason: 'The 1Hz timer must be inside a RepaintBoundary '
            'so its text repaints don\'t bubble to the bar chrome',
      );
    });
  });
}
