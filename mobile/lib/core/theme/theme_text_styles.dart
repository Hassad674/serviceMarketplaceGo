import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

/// Soleil v2 typography. Always go through these constants — never inline
/// `TextStyle(fontSize: ..., fontWeight: ...)` with magic numbers.
///
/// Fraunces for display/serif accents, Inter Tight for UI/body, Geist Mono
/// for numbers/IDs.
///
/// **Geist Mono** is bundled as an asset since 2026-05-05 (`assets/fonts/`),
/// referenced via `fontFamily: 'GeistMono'`. Other faces still come through
/// `google_fonts` (Fraunces, Inter Tight) — both are network-backed and
/// cached after the first hit, but Soleil expects them everywhere.
class SoleilTextStyles {
  SoleilTextStyles._();

  static TextStyle get displayL => GoogleFonts.fraunces(
    fontSize: 38,
    height: 1.05,
    fontWeight: FontWeight.w400,
    letterSpacing: -0.95, // -0.025em on 38px
  );

  static TextStyle get displayM => GoogleFonts.fraunces(
    fontSize: 30,
    height: 1.15,
    fontWeight: FontWeight.w400,
    letterSpacing: -0.6,
  );

  static TextStyle get headlineLarge => GoogleFonts.fraunces(
    fontSize: 28,
    height: 1.2,
    fontWeight: FontWeight.w500,
    letterSpacing: -0.5,
  );

  static TextStyle get headlineMedium => GoogleFonts.fraunces(
    fontSize: 22,
    height: 1.25,
    fontWeight: FontWeight.w500,
    letterSpacing: -0.3,
  );

  static TextStyle get titleLarge => GoogleFonts.fraunces(
    fontSize: 20,
    height: 1.3,
    fontWeight: FontWeight.w500,
  );

  static TextStyle get titleMedium => GoogleFonts.fraunces(
    fontSize: 18,
    height: 1.35,
    fontWeight: FontWeight.w600,
  );

  static TextStyle get body => GoogleFonts.interTight(
    fontSize: 14,
    height: 1.5,
    fontWeight: FontWeight.w400,
  );

  static TextStyle get bodyLarge => GoogleFonts.interTight(
    fontSize: 15,
    height: 1.6,
    fontWeight: FontWeight.w400,
  );

  static TextStyle get bodyEmphasis => GoogleFonts.interTight(
    fontSize: 14,
    fontWeight: FontWeight.w600,
  );

  static TextStyle get caption => GoogleFonts.interTight(
    fontSize: 12,
    height: 1.4,
    fontWeight: FontWeight.w500,
  );

  static TextStyle get button => GoogleFonts.interTight(
    fontSize: 14,
    fontWeight: FontWeight.w600,
    letterSpacing: 0.1,
  );

  /// Mono — Geist Mono bundled as a local asset (see `pubspec.yaml`). Used
  /// for numbers, IDs, dates metadata, mono labels (often UPPERCASE).
  ///
  /// Migrated from `GoogleFonts.jetBrainsMono(...)` on 2026-05-05 to remove
  /// the runtime network fetch and fix offline first-launch fallbacks.
  static const TextStyle mono = TextStyle(
    fontFamily: 'GeistMono',
    fontSize: 12,
    fontWeight: FontWeight.w500,
    letterSpacing: 0.6,
  );

  static const TextStyle monoLarge = TextStyle(
    fontFamily: 'GeistMono',
    fontSize: 16,
    fontWeight: FontWeight.w500,
  );

  /// Italic editorial accent — used in display headings on a key word
  /// to switch into corail. Compose as: `displayL.copyWith(fontStyle: italic, color: corail)`.
  static TextStyle italicAccent(TextStyle base, Color corail) =>
      base.copyWith(fontStyle: FontStyle.italic, color: corail);

  /// Builds the Material TextTheme with Fraunces+Inter Tight applied to
  /// the scope-appropriate styles.
  static TextTheme textTheme(Color foreground, Color muted) {
    return TextTheme(
      displayLarge: displayL.copyWith(color: foreground),
      displayMedium: displayM.copyWith(color: foreground),
      headlineLarge: headlineLarge.copyWith(color: foreground),
      headlineMedium: headlineMedium.copyWith(color: foreground),
      titleLarge: titleLarge.copyWith(color: foreground),
      titleMedium: titleMedium.copyWith(color: foreground),
      bodyLarge: bodyLarge.copyWith(color: foreground),
      bodyMedium: body.copyWith(color: foreground),
      bodySmall: caption.copyWith(color: muted),
      labelLarge: button.copyWith(color: foreground),
    );
  }
}
