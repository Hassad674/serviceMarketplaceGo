import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';

void main() {
  // ---------------------------------------------------------------------------
  // Light theme
  // ---------------------------------------------------------------------------

  group('AppTheme.light', () {
    test('returns a valid ThemeData', () {
      final theme = AppTheme.light;
      expect(theme, isA<ThemeData>());
    });

    test('uses rose-500 as primary color', () {
      final theme = AppTheme.light;
      expect(theme.colorScheme.primary, equals(const Color(0xFFF43F5E)));
    });

    test('uses slate-50 as scaffold background', () {
      final theme = AppTheme.light;
      expect(
        theme.scaffoldBackgroundColor,
        equals(const Color(0xFFF8FAFC)),
      );
    });

    test('includes AppColors extension', () {
      final theme = AppTheme.light;
      final appColors = theme.extension<AppColors>();
      expect(appColors, isNotNull);
    });

    test('AppColors has correct light values', () {
      final theme = AppTheme.light;
      final appColors = theme.extension<AppColors>()!;

      expect(appColors.muted, equals(const Color(0xFFF1F5F9)));
      expect(appColors.mutedForeground, equals(const Color(0xFF64748B)));
      expect(appColors.border, equals(const Color(0xFFE2E8F0)));
      expect(appColors.success, equals(const Color(0xFF22C55E)));
      expect(appColors.warning, equals(const Color(0xFFF59E0B)));
    });

    test('uses Material 3', () {
      final theme = AppTheme.light;
      expect(theme.useMaterial3, isTrue);
    });

    test('app bar has zero elevation', () {
      final theme = AppTheme.light;
      expect(theme.appBarTheme.elevation, equals(0));
    });

    test('elevated button has rose primary background', () {
      final theme = AppTheme.light;
      final style = theme.elevatedButtonTheme.style!;
      final bgColor = style.backgroundColor!.resolve({});
      expect(bgColor, equals(const Color(0xFFF43F5E)));
    });
  });

  // ---------------------------------------------------------------------------
  // Dark theme
  // ---------------------------------------------------------------------------

  group('AppTheme.dark', () {
    test('returns a valid ThemeData', () {
      final theme = AppTheme.dark;
      expect(theme, isA<ThemeData>());
    });

    test('uses rose-400 as primary color for dark mode', () {
      final theme = AppTheme.dark;
      expect(theme.colorScheme.primary, equals(const Color(0xFFFB7185)));
    });

    test('uses slate-900 as scaffold background', () {
      final theme = AppTheme.dark;
      expect(
        theme.scaffoldBackgroundColor,
        equals(const Color(0xFF0F172A)),
      );
    });

    test('includes AppColors extension', () {
      final theme = AppTheme.dark;
      final appColors = theme.extension<AppColors>();
      expect(appColors, isNotNull);
    });

    test('AppColors has correct dark values', () {
      final theme = AppTheme.dark;
      final appColors = theme.extension<AppColors>()!;

      expect(appColors.muted, equals(const Color(0xFF334155)));
      expect(appColors.mutedForeground, equals(const Color(0xFF94A3B8)));
      expect(appColors.border, equals(const Color(0xFF334155)));
    });
  });

  // ---------------------------------------------------------------------------
  // AppColors extension
  // ---------------------------------------------------------------------------

  group('AppColors', () {
    test('copyWith returns new instance with overridden values', () {
      const original = AppColors(
        muted: Colors.grey,
        mutedForeground: Colors.grey,
        accent: Colors.pink,
        border: Colors.grey,
        success: Colors.green,
        warning: Colors.amber,
      );

      final modified = original.copyWith(success: Colors.blue);

      expect(modified.success, equals(Colors.blue));
      expect(modified.muted, equals(Colors.grey));
      expect(modified.warning, equals(Colors.amber));
    });

    test('copyWith with no arguments returns identical values', () {
      const original = AppColors(
        muted: Colors.grey,
        mutedForeground: Colors.grey,
        accent: Colors.pink,
        border: Colors.grey,
        success: Colors.green,
        warning: Colors.amber,
      );

      final copy = original.copyWith();

      expect(copy.muted, equals(original.muted));
      expect(copy.mutedForeground, equals(original.mutedForeground));
      expect(copy.accent, equals(original.accent));
      expect(copy.border, equals(original.border));
      expect(copy.success, equals(original.success));
      expect(copy.warning, equals(original.warning));
    });

    test('lerp interpolates between two AppColors', () {
      const start = AppColors(
        muted: Color(0xFF000000),
        mutedForeground: Color(0xFF000000),
        accent: Color(0xFF000000),
        border: Color(0xFF000000),
        success: Color(0xFF000000),
        warning: Color(0xFF000000),
      );
      const end = AppColors(
        muted: Color(0xFFFFFFFF),
        mutedForeground: Color(0xFFFFFFFF),
        accent: Color(0xFFFFFFFF),
        border: Color(0xFFFFFFFF),
        success: Color(0xFFFFFFFF),
        warning: Color(0xFFFFFFFF),
      );

      final mid = start.lerp(end, 0.5);

      // At t=0.5 between black and white, we get mid-grey
      expect(mid.muted.red, closeTo(128, 1));
      expect(mid.muted.green, closeTo(128, 1));
      expect(mid.muted.blue, closeTo(128, 1));
    });

    test('lerp returns this when other is not AppColors', () {
      const start = AppColors(
        muted: Colors.grey,
        mutedForeground: Colors.grey,
        accent: Colors.pink,
        border: Colors.grey,
        success: Colors.green,
        warning: Colors.amber,
      );

      final result = start.lerp(null, 0.5);
      expect(result.muted, equals(start.muted));
    });
  });

  // ---------------------------------------------------------------------------
  // Radius and shadow constants
  // ---------------------------------------------------------------------------

  group('AppTheme constants', () {
    test('radius values are correct', () {
      expect(AppTheme.radiusSm, equals(8.0));
      expect(AppTheme.radiusMd, equals(12.0));
      expect(AppTheme.radiusLg, equals(16.0));
      expect(AppTheme.radiusXl, equals(20.0));
    });

    test('cardShadow returns non-empty list', () {
      expect(AppTheme.cardShadow, isNotEmpty);
      expect(AppTheme.cardShadow.length, equals(2));
    });

    test('cardShadowHover returns non-empty list', () {
      expect(AppTheme.cardShadowHover, isNotEmpty);
      expect(AppTheme.cardShadowHover.length, equals(2));
    });
  });
}
