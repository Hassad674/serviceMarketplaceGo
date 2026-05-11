import 'package:flutter/material.dart';

/// Tiny line chart used inside the stat cards. CustomPainter with no
/// external dep — the project doesn't ship `fl_chart`, and the V1
/// stats screen only needs a non-interactive sparkline.
///
/// Renders the polyline through the supplied [values] mapped to the
/// available size, with an optional soft fill underneath. Empty / single-
/// point series degrade to a flat baseline so the card doesn't look
/// broken when the org has no signal yet.
///
/// Wrap in a [RepaintBoundary] at call site if rebuilding adjacent
/// widgets — the painter itself is cheap (≤ 90 points) but the gradient
/// fill is the expensive part.
class SparklinePainter extends CustomPainter {
  SparklinePainter({
    required this.values,
    required this.lineColor,
    required this.fillColor,
    this.secondaryValues,
  });

  final List<int> values;
  final Color lineColor;
  final Color fillColor;

  /// Optional overlay series, drawn on top of [values] as a faded
  /// dashed line. Used by the visibility card to surface total views
  /// (overlay) alongside unique viewers (primary). Both series share
  /// the same Y axis — the painter scales to the combined max.
  final List<int>? secondaryValues;

  @override
  void paint(Canvas canvas, Size size) {
    if (size.width <= 0 || size.height <= 0) return;
    final pointCount = values.length;

    // Empty / single-point: draw a flat baseline so the card does not
    // look broken (the empty-state copy in the parent communicates the
    // "no data" status — the painter just keeps the layout stable).
    if (pointCount < 2) {
      _paintBaseline(canvas, size);
      return;
    }

    // Y axis spans the combined max of primary + secondary so neither
    // line clips when the user enables the secondary overlay.
    var maxValue = values.reduce((a, b) => a > b ? a : b);
    if (secondaryValues != null && secondaryValues!.isNotEmpty) {
      final secMax = secondaryValues!.reduce((a, b) => a > b ? a : b);
      if (secMax > maxValue) maxValue = secMax;
    }
    if (maxValue <= 0) {
      _paintBaseline(canvas, size);
      return;
    }

    final stepX = size.width / (pointCount - 1);
    // Reserve 4px headroom so the line never clips the top edge.
    final usableHeight = size.height - 4;

    final path = Path();
    final fillPath = Path()..moveTo(0, size.height);
    for (var i = 0; i < pointCount; i++) {
      final x = stepX * i;
      final ratio = values[i] / maxValue;
      final y = size.height - (ratio * usableHeight);
      if (i == 0) {
        path.moveTo(x, y);
      } else {
        path.lineTo(x, y);
      }
      fillPath.lineTo(x, y);
    }
    fillPath
      ..lineTo(size.width, size.height)
      ..close();

    final fillPaint = Paint()
      ..style = PaintingStyle.fill
      ..color = fillColor;
    canvas.drawPath(fillPath, fillPaint);

    final linePaint = Paint()
      ..style = PaintingStyle.stroke
      ..color = lineColor
      ..strokeWidth = 2
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;
    canvas.drawPath(path, linePaint);

    _paintSecondary(canvas, size, stepX, usableHeight, maxValue);
  }

  void _paintSecondary(
    Canvas canvas,
    Size size,
    double stepX,
    double usableHeight,
    int maxValue,
  ) {
    final sec = secondaryValues;
    if (sec == null || sec.length < 2) return;

    // Build the secondary polyline path on the same Y scale as primary.
    final secPath = Path();
    for (var i = 0; i < sec.length; i++) {
      final x = stepX * i;
      final ratio = sec[i] / maxValue;
      final y = size.height - (ratio * usableHeight);
      if (i == 0) {
        secPath.moveTo(x, y);
      } else {
        secPath.lineTo(x, y);
      }
    }

    // Render as dashed segments via PathMetric walk — visually reads
    // as the dashed corail line in the web chart (4 / 4 pattern).
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..color = lineColor.withValues(alpha: 0.55)
      ..strokeWidth = 1.5
      ..strokeCap = StrokeCap.round;
    const dash = 4.0;
    const gap = 4.0;
    for (final metric in secPath.computeMetrics()) {
      var distance = 0.0;
      while (distance < metric.length) {
        final next = distance + dash;
        final extracted = metric.extractPath(
          distance,
          next > metric.length ? metric.length : next,
        );
        canvas.drawPath(extracted, paint);
        distance = next + gap;
      }
    }
  }

  void _paintBaseline(Canvas canvas, Size size) {
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..color = lineColor.withValues(alpha: 0.3)
      ..strokeWidth = 1.5;
    final y = size.height - 2;
    canvas.drawLine(Offset(0, y), Offset(size.width, y), paint);
  }

  @override
  bool shouldRepaint(covariant SparklinePainter old) {
    return old.lineColor != lineColor ||
        old.fillColor != fillColor ||
        !_listEquals(old.values, values) ||
        !_listEqualsNullable(old.secondaryValues, secondaryValues);
  }

  bool _listEquals(List<int> a, List<int> b) {
    if (identical(a, b)) return true;
    if (a.length != b.length) return false;
    for (var i = 0; i < a.length; i++) {
      if (a[i] != b[i]) return false;
    }
    return true;
  }

  bool _listEqualsNullable(List<int>? a, List<int>? b) {
    if (a == null && b == null) return true;
    if (a == null || b == null) return false;
    return _listEquals(a, b);
  }
}
