import 'package:flutter/material.dart';

/// Premium B2B theme aligned with the web app's redesigned design tokens.
///
/// Access custom semantic colors via `Theme.of(context).extension<AppColors>()!`.
class AppTheme {
  AppTheme._();

  // ---------------------------------------------------------------------------
  // Color palette — matches web design tokens (Tailwind / shadcn)
  // ---------------------------------------------------------------------------

  // Primary — Rose (matches web --primary)
  static const Color _primaryLight = Color(0xFFF43F5E); // rose-500
  static const Color _primaryDark = Color(0xFFFB7185); // rose-400 (lighter for dark)
  static const Color _onPrimary = Color(0xFFFFFFFF);

  // Backgrounds
  static const Color _backgroundLight = Color(0xFFF8FAFC); // slate-50 (gray-50)
  static const Color _backgroundDark = Color(0xFF0F172A); // slate-900

  // Foregrounds (body text)
  static const Color _foregroundLight = Color(0xFF0F172A); // slate-900
  static const Color _foregroundDark = Color(0xFFF8FAFC); // slate-50

  // Cards
  static const Color _cardLight = Color(0xFFFFFFFF);
  static const Color _cardDark = Color(0xFF1E293B); // slate-800

  // Muted (subtle backgrounds, disabled)
  static const Color _mutedLight = Color(0xFFF1F5F9); // slate-100
  static const Color _mutedDark = Color(0xFF334155); // slate-700
  static const Color _mutedForegroundLight = Color(0xFF64748B); // slate-500
  static const Color _mutedForegroundDark = Color(0xFF94A3B8); // slate-400

  // Borders
  static const Color _borderLight = Color(0xFFE2E8F0); // slate-200
  static const Color _borderDark = Color(0xFF334155); // slate-700

  // Semantic
  static const Color _destructive = Color(0xFFEF4444); // red-500
  static const Color _success = Color(0xFF22C55E); // green-500
  static const Color _warning = Color(0xFFF59E0B); // amber-500

  // Accent
  static const Color _accentLight = Color(0xFFFFF1F2); // rose-50
  static const Color _accentDark = Color(0xFF4C0519); // rose-950

  // ---------------------------------------------------------------------------
  // Radii — premium feel with larger corners
  // ---------------------------------------------------------------------------

  static const double radiusSm = 8.0;
  static const double radiusMd = 12.0;
  static const double radiusLg = 16.0;
  static const double radiusXl = 20.0;

  // ---------------------------------------------------------------------------
  // Shadows — subtle, premium box shadows
  // ---------------------------------------------------------------------------

  static List<BoxShadow> get cardShadow => [
    BoxShadow(
      color: const Color(0xFF0F172A).withValues(alpha: 0.04),
      blurRadius: 8,
      offset: const Offset(0, 2),
    ),
    BoxShadow(
      color: const Color(0xFF0F172A).withValues(alpha: 0.02),
      blurRadius: 4,
      offset: const Offset(0, 1),
    ),
  ];

  static List<BoxShadow> get cardShadowHover => [
    BoxShadow(
      color: const Color(0xFF0F172A).withValues(alpha: 0.08),
      blurRadius: 16,
      offset: const Offset(0, 4),
    ),
    BoxShadow(
      color: const Color(0xFF0F172A).withValues(alpha: 0.04),
      blurRadius: 8,
      offset: const Offset(0, 2),
    ),
  ];

  // ---------------------------------------------------------------------------
  // Input decoration — rounded 12px, rose focus
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
      hintStyle: TextStyle(color: hintColor, fontSize: 15),
      labelStyle: TextStyle(color: hintColor, fontSize: 15),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
      border: border,
      enabledBorder: border,
      focusedBorder: border.copyWith(
        borderSide: BorderSide(color: focusBorderColor, width: 2),
      ),
      errorBorder: border.copyWith(
        borderSide: const BorderSide(color: _destructive),
      ),
      focusedErrorBorder: border.copyWith(
        borderSide: const BorderSide(color: _destructive, width: 2),
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Button themes — rounded 12px, full width, 48px height
  // ---------------------------------------------------------------------------

  static ElevatedButtonThemeData _elevatedButton(Color primary) {
    return ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: primary,
        foregroundColor: _onPrimary,
        minimumSize: const Size(double.infinity, 48),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusMd),
        ),
        textStyle: const TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.2,
        ),
        elevation: 0,
      ),
    );
  }

  static OutlinedButtonThemeData _outlinedButton(Color borderColor) {
    return OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        minimumSize: const Size(double.infinity, 48),
        side: BorderSide(color: borderColor),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusMd),
        ),
        textStyle: const TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  static TextButtonThemeData _textButton(Color primary) {
    return TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: primary,
        textStyle: const TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Card theme — rounded 16px, no elevation, subtle shadow via BoxDecoration
  // ---------------------------------------------------------------------------

  static CardThemeData _card(Color color, Color borderColor) {
    return CardThemeData(
      color: color,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radiusLg),
        side: BorderSide(color: borderColor),
      ),
      margin: EdgeInsets.zero,
    );
  }

  // ---------------------------------------------------------------------------
  // App bar — clean, no elevation, white bg
  // ---------------------------------------------------------------------------

  static AppBarTheme _appBar({
    required Color background,
    required Color foreground,
    required Color borderColor,
  }) {
    return AppBarTheme(
      backgroundColor: background,
      foregroundColor: foreground,
      elevation: 0,
      scrolledUnderElevation: 0,
      surfaceTintColor: Colors.transparent,
      centerTitle: false,
      titleTextStyle: TextStyle(
        color: foreground,
        fontSize: 20,
        fontWeight: FontWeight.bold,
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Navigation bar — premium bottom nav
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
        if (states.contains(WidgetState.selected)) {
          return TextStyle(
            color: selected,
            fontSize: 12,
            fontWeight: FontWeight.w600,
          );
        }
        return TextStyle(color: unselected, fontSize: 12);
      }),
      iconTheme: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) {
          return IconThemeData(color: selected, size: 24);
        }
        return IconThemeData(color: unselected, size: 24);
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
        primary: _primaryLight,
        onPrimary: _onPrimary,
        secondary: _accentLight,
        surface: _cardLight,
        onSurface: _foregroundLight,
        error: _destructive,
      ),
      scaffoldBackgroundColor: _backgroundLight,
      appBarTheme: _appBar(
        background: _cardLight,
        foreground: _foregroundLight,
        borderColor: _borderLight,
      ),
      cardTheme: _card(_cardLight, _borderLight),
      elevatedButtonTheme: _elevatedButton(_primaryLight),
      outlinedButtonTheme: _outlinedButton(_borderLight),
      textButtonTheme: _textButton(_primaryLight),
      inputDecorationTheme: _inputDecoration(
        fillColor: _cardLight,
        borderColor: _borderLight,
        focusBorderColor: _primaryLight,
        hintColor: _mutedForegroundLight,
      ),
      dividerColor: _borderLight,
      dividerTheme: const DividerThemeData(color: _borderLight, thickness: 1),
      navigationBarTheme: _navigationBar(
        background: _cardLight,
        indicator: _primaryLight.withValues(alpha: 0.1),
        selected: _primaryLight,
        unselected: _mutedForegroundLight,
      ),
      textTheme: const TextTheme(
        headlineLarge: TextStyle(
          fontSize: 28,
          fontWeight: FontWeight.bold,
          color: _foregroundLight,
          letterSpacing: -0.5,
        ),
        headlineMedium: TextStyle(
          fontSize: 22,
          fontWeight: FontWeight.bold,
          color: _foregroundLight,
          letterSpacing: -0.3,
        ),
        titleLarge: TextStyle(
          fontSize: 20,
          fontWeight: FontWeight.bold,
          color: _foregroundLight,
        ),
        titleMedium: TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w600,
          color: _foregroundLight,
        ),
        bodyLarge: TextStyle(fontSize: 16, color: _foregroundLight),
        bodyMedium: TextStyle(fontSize: 15, color: _foregroundLight),
        bodySmall: TextStyle(fontSize: 13, color: _mutedForegroundLight),
      ),
      bottomNavigationBarTheme: const BottomNavigationBarThemeData(
        backgroundColor: _cardLight,
        selectedItemColor: _primaryLight,
        unselectedItemColor: _mutedForegroundLight,
        type: BottomNavigationBarType.fixed,
        elevation: 0,
      ),
      chipTheme: ChipThemeData(
        backgroundColor: _mutedLight,
        labelStyle: const TextStyle(color: _foregroundLight),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusSm),
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusMd),
        ),
      ),
      extensions: <ThemeExtension<dynamic>>[
        const AppColors(
          muted: _mutedLight,
          mutedForeground: _mutedForegroundLight,
          accent: _accentLight,
          border: _borderLight,
          success: _success,
          warning: _warning,
        ),
      ],
    );
  }

  static ThemeData get dark {
    final base = ThemeData.dark(useMaterial3: true);

    return base.copyWith(
      colorScheme: const ColorScheme.dark(
        primary: _primaryDark,
        onPrimary: _onPrimary,
        secondary: _accentDark,
        surface: _cardDark,
        onSurface: _foregroundDark,
        error: _destructive,
      ),
      scaffoldBackgroundColor: _backgroundDark,
      appBarTheme: _appBar(
        background: _cardDark,
        foreground: _foregroundDark,
        borderColor: _borderDark,
      ),
      cardTheme: _card(_cardDark, _borderDark),
      elevatedButtonTheme: _elevatedButton(_primaryDark),
      outlinedButtonTheme: _outlinedButton(_borderDark),
      textButtonTheme: _textButton(_primaryDark),
      inputDecorationTheme: _inputDecoration(
        fillColor: _cardDark,
        borderColor: _borderDark,
        focusBorderColor: _primaryDark,
        hintColor: _mutedForegroundDark,
      ),
      dividerColor: _borderDark,
      dividerTheme: const DividerThemeData(color: _borderDark, thickness: 1),
      navigationBarTheme: _navigationBar(
        background: _cardDark,
        indicator: _primaryDark.withValues(alpha: 0.15),
        selected: _primaryDark,
        unselected: _mutedForegroundDark,
      ),
      textTheme: const TextTheme(
        headlineLarge: TextStyle(
          fontSize: 28,
          fontWeight: FontWeight.bold,
          color: _foregroundDark,
          letterSpacing: -0.5,
        ),
        headlineMedium: TextStyle(
          fontSize: 22,
          fontWeight: FontWeight.bold,
          color: _foregroundDark,
          letterSpacing: -0.3,
        ),
        titleLarge: TextStyle(
          fontSize: 20,
          fontWeight: FontWeight.bold,
          color: _foregroundDark,
        ),
        titleMedium: TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w600,
          color: _foregroundDark,
        ),
        bodyLarge: TextStyle(fontSize: 16, color: _foregroundDark),
        bodyMedium: TextStyle(fontSize: 15, color: _foregroundDark),
        bodySmall: TextStyle(fontSize: 13, color: _mutedForegroundDark),
      ),
      bottomNavigationBarTheme: const BottomNavigationBarThemeData(
        backgroundColor: _cardDark,
        selectedItemColor: _primaryDark,
        unselectedItemColor: _mutedForegroundDark,
        type: BottomNavigationBarType.fixed,
        elevation: 0,
      ),
      chipTheme: ChipThemeData(
        backgroundColor: _mutedDark,
        labelStyle: const TextStyle(color: _foregroundDark),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusSm),
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusMd),
        ),
      ),
      extensions: <ThemeExtension<dynamic>>[
        const AppColors(
          muted: _mutedDark,
          mutedForeground: _mutedForegroundDark,
          accent: _accentDark,
          border: _borderDark,
          success: _success,
          warning: _warning,
        ),
      ],
    );
  }
}

/// Custom theme extension for semantic colors not covered by [ColorScheme].
///
/// Access via `Theme.of(context).extension<AppColors>()!`.
@immutable
class AppColors extends ThemeExtension<AppColors> {
  const AppColors({
    required this.muted,
    required this.mutedForeground,
    required this.accent,
    required this.border,
    required this.success,
    required this.warning,
  });

  final Color muted;
  final Color mutedForeground;
  final Color accent;
  final Color border;
  final Color success;
  final Color warning;

  @override
  AppColors copyWith({
    Color? muted,
    Color? mutedForeground,
    Color? accent,
    Color? border,
    Color? success,
    Color? warning,
  }) {
    return AppColors(
      muted: muted ?? this.muted,
      mutedForeground: mutedForeground ?? this.mutedForeground,
      accent: accent ?? this.accent,
      border: border ?? this.border,
      success: success ?? this.success,
      warning: warning ?? this.warning,
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
    );
  }
}
