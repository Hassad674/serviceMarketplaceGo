import 'package:flutter/material.dart';

import 'theme_colors.dart';
import 'theme_text_styles.dart';

// Re-export so external code can keep importing `app_theme.dart` and still
// see `AppColors` + `SoleilTextStyles`. This keeps the file split internal.
export 'theme_colors.dart' show AppColors;
export 'theme_text_styles.dart' show SoleilTextStyles;

/// DesignedTrust Services — Direction Soleil v2 mobile theme.
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
