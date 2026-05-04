import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

/// Atelier — Direction Soleil v2 mobile theme.
///
/// Palette ivoire & corail. Typographie Fraunces (display) + Inter Tight
/// (UI) + Geist Mono (numbers/IDs). Source-of-truth: design/DESIGN_SYSTEM.md.
///
/// Access:
/// - Material 3 colors via `Theme.of(context).colorScheme`
/// - Soleil-specific tokens via `Theme.of(context).extension<AppColors>()!`
/// - Typography via `SoleilTextStyles.*` constants (preferred over inline
///   `TextStyle(...)` with magic numbers)
class AppTheme {
  AppTheme._();

  // ---------------------------------------------------------------------------
  // Soleil v2 palette — see design/DESIGN_SYSTEM.md §1
  // ---------------------------------------------------------------------------

  // Surfaces
  static const Color _ivoire = Color(0xFFFFFBF5);          // bg
  static const Color _surface = Color(0xFFFFFFFF);         // cards
  static const Color _encre = Color(0xFF2A1F15);           // primary text
  static const Color _tabac = Color(0xFF7A6850);           // secondary text
  static const Color _sable = Color(0xFFA89679);           // mono labels, tertiary

  // Accents
  static const Color _corail = Color(0xFFE85D4A);          // CTAs
  static const Color _corailSoft = Color(0xFFFDE9E3);      // soft bg, active pill
  static const Color _corailDeep = Color(0xFFC43A26);      // hover, error

  // Decorative
  static const Color _pink = Color(0xFFF08AA8);
  static const Color _pinkSoft = Color(0xFFFDE6ED);
  static const Color _amberSoft = Color(0xFFFBF0DC);

  // Semantic
  static const Color _sapin = Color(0xFF5A9670);           // success
  static const Color _sapinSoft = Color(0xFFE8F2EB);
  static const Color _ambre = Color(0xFFD4924A);           // warning

  // Borders
  static const Color _borderLight = Color(0xFFF0E6D8);     // sable clair
  static const Color _borderStrong = Color(0xFFE0D3BC);    // sable foncé

  // Dark variant (calibrated to keep the warm identity, no cold blue)
  static const Color _ivoireDark = Color(0xFF1A1410);
  static const Color _surfaceDark = Color(0xFF251C16);
  static const Color _encreDark = Color(0xFFFBF3E4);
  static const Color _tabacDark = Color(0xFFA89679);
  static const Color _sableDark = Color(0xFF7A6850);
  static const Color _corailDark = Color(0xFFFB7D68);
  static const Color _corailSoftDark = Color(0xFF3D201B);
  static const Color _corailDeepDark = Color(0xFFFF9784);
  static const Color _sapinDark = Color(0xFF6FAE87);
  static const Color _sapinSoftDark = Color(0xFF1F3026);
  static const Color _borderDarkLight = Color(0xFF3A2E23);
  static const Color _borderDarkStrong = Color(0xFF4D3E2F);
  static const Color _onPrimary = Color(0xFFFFFBF5);

  // ---------------------------------------------------------------------------
  // Radii — Soleil signature: pills/buttons fully rounded
  // ---------------------------------------------------------------------------

  static const double radiusSm = 6.0;
  static const double radiusMd = 10.0;
  static const double radiusLg = 14.0;
  static const double radiusXl = 18.0;
  static const double radius2xl = 20.0;
  static const double radiusFull = 999.0;

  // ---------------------------------------------------------------------------
  // Shadows — calm, no glow (Soleil)
  // ---------------------------------------------------------------------------

  static List<BoxShadow> get cardShadow => [
    BoxShadow(
      color: _encre.withValues(alpha: 0.04),
      blurRadius: 24,
      offset: const Offset(0, 4),
    ),
  ];

  static List<BoxShadow> get cardShadowStrong => [
    BoxShadow(
      color: const Color(0xFF000000).withValues(alpha: 0.12),
      blurRadius: 24,
      offset: const Offset(0, 8),
    ),
  ];

  static List<BoxShadow> get portraitShadow => [
    BoxShadow(
      color: _encre.withValues(alpha: 0.06),
      blurRadius: 12,
      offset: const Offset(0, 2),
    ),
  ];

  // ---------------------------------------------------------------------------
  // Input decoration — radius 10, corail focus
  // ---------------------------------------------------------------------------

  static InputDecorationTheme _inputDecoration({
    required Color fillColor,
    required Color borderColor,
    required Color focusBorderColor,
    required Color hintColor,
  }) {
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(radiusMd),
      borderSide: BorderSide(color: borderColor),
    );

    return InputDecorationTheme(
      filled: true,
      fillColor: fillColor,
      hintStyle: SoleilTextStyles.body.copyWith(color: hintColor),
      labelStyle: SoleilTextStyles.body.copyWith(color: hintColor),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      border: border,
      enabledBorder: border,
      focusedBorder: border.copyWith(
        borderSide: BorderSide(color: focusBorderColor, width: 2),
      ),
      errorBorder: border.copyWith(
        borderSide: const BorderSide(color: _corailDeep),
      ),
      focusedErrorBorder: border.copyWith(
        borderSide: const BorderSide(color: _corailDeep, width: 2),
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Buttons — fully rounded pills (Soleil signature)
  // ---------------------------------------------------------------------------

  static ElevatedButtonThemeData _elevatedButton(Color primary) {
    return ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: primary,
        foregroundColor: _onPrimary,
        minimumSize: const Size(double.infinity, 48),
        shape: const StadiumBorder(),
        textStyle: SoleilTextStyles.button,
        elevation: 0,
      ),
    );
  }

  static OutlinedButtonThemeData _outlinedButton(Color borderColor, Color foreground) {
    return OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: foreground,
        minimumSize: const Size(double.infinity, 48),
        side: BorderSide(color: borderColor),
        shape: const StadiumBorder(),
        textStyle: SoleilTextStyles.button,
      ),
    );
  }

  static TextButtonThemeData _textButton(Color primary) {
    return TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: primary,
        textStyle: SoleilTextStyles.bodyEmphasis,
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Card — radius 18, no elevation, subtle border
  // ---------------------------------------------------------------------------

  static CardThemeData _card(Color color, Color borderColor) {
    return CardThemeData(
      color: color,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radiusXl),
        side: BorderSide(color: borderColor),
      ),
      margin: EdgeInsets.zero,
    );
  }

  // ---------------------------------------------------------------------------
  // App bar — flat ivoire, encre text
  // ---------------------------------------------------------------------------

  static AppBarTheme _appBar({
    required Color background,
    required Color foreground,
  }) {
    return AppBarTheme(
      backgroundColor: background,
      foregroundColor: foreground,
      elevation: 0,
      scrolledUnderElevation: 0,
      surfaceTintColor: Colors.transparent,
      centerTitle: false,
      titleTextStyle: SoleilTextStyles.titleLarge.copyWith(color: foreground),
    );
  }

  // ---------------------------------------------------------------------------
  // Navigation bar — soft active background
  // ---------------------------------------------------------------------------

  static NavigationBarThemeData _navigationBar({
    required Color background,
    required Color indicator,
    required Color selected,
    required Color unselected,
  }) {
    return NavigationBarThemeData(
      backgroundColor: background,
      elevation: 0,
      indicatorColor: indicator,
      labelTextStyle: WidgetStateProperty.resolveWith((states) {
        final selectedNow = states.contains(WidgetState.selected);
        return SoleilTextStyles.caption.copyWith(
          color: selectedNow ? selected : unselected,
          fontWeight: selectedNow ? FontWeight.w600 : FontWeight.w500,
        );
      }),
      iconTheme: WidgetStateProperty.resolveWith((states) {
        final selectedNow = states.contains(WidgetState.selected);
        return IconThemeData(
          color: selectedNow ? selected : unselected,
          size: 24,
        );
      }),
    );
  }

  // ---------------------------------------------------------------------------
  // Public theme getters
  // ---------------------------------------------------------------------------

  static ThemeData get light {
    final base = ThemeData.light(useMaterial3: true);
    return base.copyWith(
      colorScheme: const ColorScheme.light(
        primary: _corail,
        onPrimary: _onPrimary,
        primaryContainer: _corailSoft,
        onPrimaryContainer: _corailDeep,
        secondary: _corailSoft,
        onSecondary: _corailDeep,
        surface: _ivoire,
        onSurface: _encre,
        surfaceContainerLowest: _surface,
        surfaceContainerHigh: _surface,
        onSurfaceVariant: _tabac,
        outline: _borderLight,
        outlineVariant: _borderStrong,
        error: _corailDeep,
        onError: _onPrimary,
      ),
      scaffoldBackgroundColor: _ivoire,
      appBarTheme: _appBar(background: _surface, foreground: _encre),
      cardTheme: _card(_surface, _borderLight),
      elevatedButtonTheme: _elevatedButton(_corail),
      outlinedButtonTheme: _outlinedButton(_borderStrong, _encre),
      textButtonTheme: _textButton(_corail),
      inputDecorationTheme: _inputDecoration(
        fillColor: _surface,
        borderColor: _borderLight,
        focusBorderColor: _corail,
        hintColor: _tabac,
      ),
      dividerColor: _borderLight,
      dividerTheme: const DividerThemeData(color: _borderLight, thickness: 1),
      navigationBarTheme: _navigationBar(
        background: _surface,
        indicator: _corailSoft,
        selected: _corail,
        unselected: _tabac,
      ),
      textTheme: SoleilTextStyles.textTheme(_encre, _tabac),
      bottomNavigationBarTheme: const BottomNavigationBarThemeData(
        backgroundColor: _surface,
        selectedItemColor: _corail,
        unselectedItemColor: _tabac,
        type: BottomNavigationBarType.fixed,
        elevation: 0,
      ),
      chipTheme: ChipThemeData(
        backgroundColor: _ivoire,
        labelStyle: SoleilTextStyles.caption.copyWith(color: _tabac),
        shape: const StadiumBorder(),
        side: const BorderSide(color: _borderLight),
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(radiusMd)),
      ),
      extensions: const <ThemeExtension<dynamic>>[
        AppColors(
          // Backward-compatible aliases (legacy code)
          muted: _ivoire,
          mutedForeground: _tabac,
          accent: _corailSoft,
          border: _borderLight,
          success: _sapin,
          warning: _ambre,
          // Soleil-specific tokens
          subtleForeground: _sable,
          primaryDeep: _corailDeep,
          accentSoft: _corailSoft,
          successSoft: _sapinSoft,
          pink: _pink,
          pinkSoft: _pinkSoft,
          amberSoft: _amberSoft,
          borderStrong: _borderStrong,
        ),
      ],
    );
  }

  static ThemeData get dark {
    final base = ThemeData.dark(useMaterial3: true);
    return base.copyWith(
      colorScheme: const ColorScheme.dark(
        primary: _corailDark,
        onPrimary: _ivoireDark,
        primaryContainer: _corailSoftDark,
        onPrimaryContainer: _corailDeepDark,
        secondary: _corailSoftDark,
        onSecondary: _corailDeepDark,
        surface: _ivoireDark,
        onSurface: _encreDark,
        surfaceContainerLowest: _surfaceDark,
        surfaceContainerHigh: _surfaceDark,
        onSurfaceVariant: _tabacDark,
        outline: _borderDarkLight,
        outlineVariant: _borderDarkStrong,
        error: _corailDeepDark,
        onError: _ivoireDark,
      ),
      scaffoldBackgroundColor: _ivoireDark,
      appBarTheme: _appBar(background: _surfaceDark, foreground: _encreDark),
      cardTheme: _card(_surfaceDark, _borderDarkLight),
      elevatedButtonTheme: _elevatedButton(_corailDark),
      outlinedButtonTheme: _outlinedButton(_borderDarkStrong, _encreDark),
      textButtonTheme: _textButton(_corailDark),
      inputDecorationTheme: _inputDecoration(
        fillColor: _surfaceDark,
        borderColor: _borderDarkLight,
        focusBorderColor: _corailDark,
        hintColor: _tabacDark,
      ),
      dividerColor: _borderDarkLight,
      dividerTheme: const DividerThemeData(color: _borderDarkLight, thickness: 1),
      navigationBarTheme: _navigationBar(
        background: _surfaceDark,
        indicator: _corailSoftDark,
        selected: _corailDark,
        unselected: _tabacDark,
      ),
      textTheme: SoleilTextStyles.textTheme(_encreDark, _tabacDark),
      bottomNavigationBarTheme: const BottomNavigationBarThemeData(
        backgroundColor: _surfaceDark,
        selectedItemColor: _corailDark,
        unselectedItemColor: _tabacDark,
        type: BottomNavigationBarType.fixed,
        elevation: 0,
      ),
      chipTheme: ChipThemeData(
        backgroundColor: _ivoireDark,
        labelStyle: SoleilTextStyles.caption.copyWith(color: _tabacDark),
        shape: const StadiumBorder(),
        side: const BorderSide(color: _borderDarkLight),
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(radiusMd)),
      ),
      extensions: const <ThemeExtension<dynamic>>[
        AppColors(
          muted: _ivoireDark,
          mutedForeground: _tabacDark,
          accent: _corailSoftDark,
          border: _borderDarkLight,
          success: _sapinDark,
          warning: _ambre,
          subtleForeground: _sableDark,
          primaryDeep: _corailDeepDark,
          accentSoft: _corailSoftDark,
          successSoft: _sapinSoftDark,
          pink: _pink,
          pinkSoft: Color(0xFF3D2230),
          amberSoft: Color(0xFF2D2418),
          borderStrong: _borderDarkStrong,
        ),
      ],
    );
  }
}

/// Soleil v2 typography. Always go through these constants — never inline
/// `TextStyle(fontSize: ..., fontWeight: ...)` with magic numbers.
///
/// Fraunces for display/serif accents, Inter Tight for UI/body, Geist Mono
/// for numbers/IDs. Loaded via google_fonts (cached after first network hit).
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

  /// Mono — for numbers, IDs, dates metadata, mono labels (often UPPERCASE).
  ///
  /// Implementation note: google_fonts ^6.2.1 does not yet ship Geist Mono.
  /// JetBrains Mono is the closest visual match and stays in the family
  /// (Geist Mono itself was inspired by JetBrains-style mono). Swap to
  /// `GoogleFonts.geistMono()` when the package version that supports
  /// it lands on the constraint we accept.
  static TextStyle get mono => GoogleFonts.jetBrainsMono(
    fontSize: 12,
    fontWeight: FontWeight.w500,
    letterSpacing: 0.6,
  );

  static TextStyle get monoLarge => GoogleFonts.jetBrainsMono(
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

/// Theme extension exposing the Soleil tokens not covered by Material 3
/// `ColorScheme`. Access via `Theme.of(context).extension<AppColors>()!`.
///
/// The legacy fields (muted, mutedForeground, accent, border, success,
/// warning) are kept for backward compatibility with widgets written
/// before Soleil v2. New widgets should prefer the Soleil-named fields
/// (subtleForeground, accentSoft, successSoft, pinkSoft, etc.) for clarity.
@immutable
class AppColors extends ThemeExtension<AppColors> {
  const AppColors({
    // Legacy aliases
    required this.muted,
    required this.mutedForeground,
    required this.accent,
    required this.border,
    required this.success,
    required this.warning,
    // Soleil-specific
    required this.subtleForeground,
    required this.primaryDeep,
    required this.accentSoft,
    required this.successSoft,
    required this.pink,
    required this.pinkSoft,
    required this.amberSoft,
    required this.borderStrong,
  });

  final Color muted;
  final Color mutedForeground;
  final Color accent;
  final Color border;
  final Color success;
  final Color warning;

  final Color subtleForeground;
  final Color primaryDeep;
  final Color accentSoft;
  final Color successSoft;
  final Color pink;
  final Color pinkSoft;
  final Color amberSoft;
  final Color borderStrong;

  @override
  AppColors copyWith({
    Color? muted,
    Color? mutedForeground,
    Color? accent,
    Color? border,
    Color? success,
    Color? warning,
    Color? subtleForeground,
    Color? primaryDeep,
    Color? accentSoft,
    Color? successSoft,
    Color? pink,
    Color? pinkSoft,
    Color? amberSoft,
    Color? borderStrong,
  }) {
    return AppColors(
      muted: muted ?? this.muted,
      mutedForeground: mutedForeground ?? this.mutedForeground,
      accent: accent ?? this.accent,
      border: border ?? this.border,
      success: success ?? this.success,
      warning: warning ?? this.warning,
      subtleForeground: subtleForeground ?? this.subtleForeground,
      primaryDeep: primaryDeep ?? this.primaryDeep,
      accentSoft: accentSoft ?? this.accentSoft,
      successSoft: successSoft ?? this.successSoft,
      pink: pink ?? this.pink,
      pinkSoft: pinkSoft ?? this.pinkSoft,
      amberSoft: amberSoft ?? this.amberSoft,
      borderStrong: borderStrong ?? this.borderStrong,
    );
  }

  @override
  AppColors lerp(ThemeExtension<AppColors>? other, double t) {
    if (other is! AppColors) return this;
    return AppColors(
      muted: Color.lerp(muted, other.muted, t)!,
      mutedForeground: Color.lerp(mutedForeground, other.mutedForeground, t)!,
      accent: Color.lerp(accent, other.accent, t)!,
      border: Color.lerp(border, other.border, t)!,
      success: Color.lerp(success, other.success, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      subtleForeground: Color.lerp(subtleForeground, other.subtleForeground, t)!,
      primaryDeep: Color.lerp(primaryDeep, other.primaryDeep, t)!,
      accentSoft: Color.lerp(accentSoft, other.accentSoft, t)!,
      successSoft: Color.lerp(successSoft, other.successSoft, t)!,
      pink: Color.lerp(pink, other.pink, t)!,
      pinkSoft: Color.lerp(pinkSoft, other.pinkSoft, t)!,
      amberSoft: Color.lerp(amberSoft, other.amberSoft, t)!,
      borderStrong: Color.lerp(borderStrong, other.borderStrong, t)!,
    );
  }
}
