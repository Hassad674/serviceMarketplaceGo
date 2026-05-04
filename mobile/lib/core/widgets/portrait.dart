import 'package:flutter/material.dart';

/// Atelier — stylized SVG portrait used everywhere a person is referenced.
///
/// 6 deterministic palettes selected by `id % 6`. Replaces ALL initials,
/// emojis, or generic gray placeholders. Drawn natively via [CustomPainter]
/// so even a long list of avatars stays cheap (no SVG parsing).
///
/// Reference implementation: design/assets/sources/phase1/soleil.jsx
/// lines 27-52. Keep the silhouette geometry in lock-step with that file.
class Portrait extends StatelessWidget {
  const Portrait({
    super.key,
    required this.id,
    this.size = 48,
    this.borderRadius,
    this.semanticLabel = 'Portrait',
  });

  /// Deterministic seed for palette selection (0-5 picked via `id % 6`).
  final int id;

  /// Width and height in logical pixels. Default 48.
  final double size;

  /// Custom border radius. Defaults to fully rounded (size / 2).
  final BorderRadius? borderRadius;

  /// Semantic label exposed to screen readers.
  final String semanticLabel;

  @override
  Widget build(BuildContext context) {
    final palette = _palettes[((id % _palettes.length) + _palettes.length) % _palettes.length];
    final radius = borderRadius ?? BorderRadius.circular(size / 2);

    return Semantics(
      label: semanticLabel,
      image: true,
      child: SizedBox(
        width: size,
        height: size,
        child: ClipRRect(
          borderRadius: radius,
          child: CustomPaint(
            painter: _PortraitPainter(palette),
            size: Size(size, size),
          ),
        ),
      ),
    );
  }
}

/// Number of distinct palettes — exposed for tests and consumers.
const int kPortraitPaletteCount = 6;

class _Palette {
  const _Palette({
    required this.bg,
    required this.skin,
    required this.hair,
    required this.shirt,
  });

  final Color bg;
  final Color skin;
  final Color hair;
  final Color shirt;
}

const List<_Palette> _palettes = [
  // 0 — corail
  _Palette(bg: Color(0xFFFDE9E3), skin: Color(0xFFE8A890), hair: Color(0xFF3D2618), shirt: Color(0xFFC43A26)),
  // 1 — vert olive
  _Palette(bg: Color(0xFFE8F2EB), skin: Color(0xFFD4A584), hair: Color(0xFF5A3A1F), shirt: Color(0xFF5A9670)),
  // 2 — rose
  _Palette(bg: Color(0xFFFDE6ED), skin: Color(0xFFD49A82), hair: Color(0xFF1A1A1A), shirt: Color(0xFFC84D72)),
  // 3 — ambre
  _Palette(bg: Color(0xFFFBF0DC), skin: Color(0xFFC4926E), hair: Color(0xFF8B4A1F), shirt: Color(0xFFD4924A)),
  // 4 — lilas
  _Palette(bg: Color(0xFFE8E4F4), skin: Color(0xFFD8A890), hair: Color(0xFF2A1F3A), shirt: Color(0xFF6B5B9A)),
  // 5 — bleu
  _Palette(bg: Color(0xFFDFECEF), skin: Color(0xFFC89478), hair: Color(0xFF3D2818), shirt: Color(0xFF3A6B7A)),
];

class _PortraitPainter extends CustomPainter {
  _PortraitPainter(this.palette);

  final _Palette palette;

  @override
  void paint(Canvas canvas, Size size) {
    // The SVG viewBox is 0..60, so we scale.
    final scaleX = size.width / 60;
    final scaleY = size.height / 60;
    canvas.scale(scaleX, scaleY);

    // Background fills the whole canvas (clipping handled by ClipRRect outside).
    final bgPaint = Paint()..color = palette.bg;
    canvas.drawRect(const Rect.fromLTWH(0, 0, 60, 60), bgPaint);

    // Cou — rect 24,38 12x10
    final skinPaint = Paint()..color = palette.skin;
    canvas.drawRect(const Rect.fromLTWH(24, 38, 12, 10), skinPaint);

    // Épaules / haut — path "M8 60 Q8 46 30 44 Q52 46 52 60 Z"
    final shirtPaint = Paint()..color = palette.shirt;
    final shoulders = Path()
      ..moveTo(8, 60)
      ..quadraticBezierTo(8, 46, 30, 44)
      ..quadraticBezierTo(52, 46, 52, 60)
      ..close();
    canvas.drawPath(shoulders, shirtPaint);

    // Tête — ellipse cx=30 cy=28 rx=11 ry=13
    canvas.drawOval(
      Rect.fromCenter(center: const Offset(30, 28), width: 22, height: 26),
      skinPaint,
    );

    // Cheveux — path "M19 24 Q19 13 30 13 Q41 13 41 24 Q41 21 36 19 Q30 17 24 19 Q19 21 19 28 Z"
    final hairPaint = Paint()..color = palette.hair;
    final hair = Path()
      ..moveTo(19, 24)
      ..quadraticBezierTo(19, 13, 30, 13)
      ..quadraticBezierTo(41, 13, 41, 24)
      ..quadraticBezierTo(41, 21, 36, 19)
      ..quadraticBezierTo(30, 17, 24, 19)
      ..quadraticBezierTo(19, 21, 19, 28)
      ..close();
    canvas.drawPath(hair, hairPaint);
  }

  @override
  bool shouldRepaint(covariant _PortraitPainter oldDelegate) {
    return oldDelegate.palette != palette;
  }
}
