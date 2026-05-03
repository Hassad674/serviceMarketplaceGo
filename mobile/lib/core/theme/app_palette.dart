import 'package:flutter/material.dart';

/// AppPalette — single source of truth for all literal color values used
/// across the mobile UI.
///
/// **Why this file exists:** the codebase had ~550 hardcoded `Color(0xFF...)`
/// literals scattered across feature widgets, which made theming
/// inconsistent and hex values hard to track. This palette centralizes
/// the Tailwind-equivalent tokens we use.
///
/// **Naming convention:** `<family><shade>` matches Tailwind CSS:
///   - rose50, rose100, ..., rose900
///   - slate50, slate100, ..., slate900
///   - amber50, amber500, etc.
/// Plus a few brand colors (LinkedIn, Twitter) and pure black/white.
///
/// **Usage:**
/// ```dart
/// import 'package:marketplace_mobile/core/theme/app_palette.dart';
/// Container(color: AppPalette.rose500);
/// ```
///
/// Theme-aware semantic tokens belong in `AppColors` (theme extension).
/// This palette is the raw layer underneath that — direct hex tokens.
class AppPalette {
  AppPalette._();

  // --- Primary brand (Rose, matches web --primary) -------------------------
  static const Color rose50 = Color(0xFFFFF1F2);
  static const Color rose100 = Color(0xFFFFE4E6);
  static const Color rose200 = Color(0xFFFECDD3);
  static const Color rose300 = Color(0xFFFDA4AF);
  static const Color rose400 = Color(0xFFFB7185);
  static const Color rose500 = Color(0xFFF43F5E);
  static const Color rose600 = Color(0xFFE11D48);
  static const Color rose700 = Color(0xFFBE123C);
  static const Color rose950 = Color(0xFF4C0519);

  // --- Slate (gray) --------------------------------------------------------
  static const Color slate50 = Color(0xFFF8FAFC);
  static const Color slate100 = Color(0xFFF1F5F9);
  static const Color slate200 = Color(0xFFE2E8F0);
  static const Color slate300 = Color(0xFFCBD5E1);
  static const Color slate400 = Color(0xFF94A3B8);
  static const Color slate500 = Color(0xFF64748B);
  static const Color slate600 = Color(0xFF475569);
  static const Color slate700 = Color(0xFF334155);
  static const Color slate800 = Color(0xFF1E293B);
  static const Color slate900 = Color(0xFF0F172A);

  // --- Red / Destructive ---------------------------------------------------
  static const Color red100 = Color(0xFFFEE2E2);
  static const Color red200 = Color(0xFFFECACA);
  static const Color red300 = Color(0xFFFCA5A5);
  static const Color red400 = Color(0xFFF87171);
  static const Color red500 = Color(0xFFEF4444);
  static const Color red600 = Color(0xFFDC2626);
  static const Color red700 = Color(0xFFB91C1C);
  static const Color red800 = Color(0xFF991B1B);
  static const Color red50 = Color(0xFFFEF2F2);

  // --- Amber / Warning -----------------------------------------------------
  static const Color amber50 = Color(0xFFFFFBEB);
  static const Color amber100 = Color(0xFFFEF3C7);
  static const Color amber200 = Color(0xFFFDE68A);
  static const Color amber300 = Color(0xFFFCD34D);
  static const Color amber400 = Color(0xFFFBBF24);
  static const Color amber500 = Color(0xFFF59E0B);
  static const Color amber600 = Color(0xFFD97706);
  static const Color amber700 = Color(0xFFB45309);
  static const Color amber800 = Color(0xFF92400E);

  // --- Green / Success -----------------------------------------------------
  static const Color green100 = Color(0xFFDCFCE7);
  static const Color green500 = Color(0xFF22C55E);
  static const Color green600 = Color(0xFF16A34A);
  static const Color green700 = Color(0xFF15803D);
  static const Color green800 = Color(0xFF166534);

  // --- Emerald (success accent) -------------------------------------------
  static const Color emerald100 = Color(0xFFD1FAE5);
  static const Color emerald200 = Color(0xFFA7F3D0);
  static const Color emerald300 = Color(0xFF6EE7B7);
  static const Color emerald500 = Color(0xFF10B981);
  static const Color emerald600 = Color(0xFF059669);
  static const Color emerald700 = Color(0xFF047857);
  static const Color emerald800 = Color(0xFF065F46);
  static const Color emerald50 = Color(0xFFECFDF5);

  // --- Blue -----------------------------------------------------------------
  static const Color blue50 = Color(0xFFEFF6FF);
  static const Color blue100 = Color(0xFFDBEAFE);
  static const Color blue200 = Color(0xFFBFDBFE);
  static const Color blue500 = Color(0xFF3B82F6);
  static const Color blue600 = Color(0xFF2563EB);
  static const Color blue700 = Color(0xFF1D4ED8);
  static const Color blue800 = Color(0xFF1E40AF);
  static const Color blue900 = Color(0xFF1E3A8A);

  // --- Sky (cyan-ish blue) -------------------------------------------------
  static const Color sky100 = Color(0xFFE0F2FE);
  static const Color sky800 = Color(0xFF075985);

  // --- Indigo ---------------------------------------------------------------
  static const Color indigo100 = Color(0xFFE0E7FF);
  static const Color indigo700 = Color(0xFF4338CA);

  // --- Violet / Purple -----------------------------------------------------
  static const Color violet50 = Color(0xFFFAF5FF);
  static const Color violet100 = Color(0xFFEDE9FE);
  static const Color violet500 = Color(0xFF8B5CF6);
  static const Color violet700 = Color(0xFF6D28D9);
  static const Color purple100 = Color(0xFFF3E8FF);
  static const Color purple700 = Color(0xFF7E22CE);

  // --- Teal -----------------------------------------------------------------
  static const Color teal500 = Color(0xFF14B8A6);

  // --- Orange ---------------------------------------------------------------
  static const Color orange100 = Color(0xFFFFEDD5);
  static const Color orange200 = Color(0xFFFED7AA);
  static const Color orange300 = Color(0xFFFDBA74);
  static const Color orange50 = Color(0xFFFFF7ED);
  static const Color orange600 = Color(0xFFEA580C);
  static const Color orange700 = Color(0xFFC2410C);
  static const Color orange800 = Color(0xFF9A3412);

  // --- Pink (very pale) ---------------------------------------------------
  static const Color pink100 = Color(0xFFFCE4EC);

  // --- Neutrals -------------------------------------------------------------
  static const Color white = Color(0xFFFFFFFF);
  static const Color black = Color(0xFF000000);
  static const Color neutral50 = Color(0xFFFAFAFA);
  static const Color gray800 = Color(0xFF333333);

  // --- Brand (third-party social platforms) -------------------------------
  static const Color linkedinBlue = Color(0xFF0A66C2);
  static const Color twitterBlue = Color(0xFF1DA1F2);
  static const Color instagramPink = Color(0xFFE4405F);

  // --- Status / Misc -------------------------------------------------------
  static const Color pureRed = Color(0xFFFF0000);

  // --- Translucent overlays (alpha-aware) ---------------------------------
  /// Black at 80% opacity — used for drop shadows under glassmorphic surfaces.
  static const Color black80 = Color(0xCC000000);

  /// Black at 95% opacity — used for full-bleed call/video backgrounds.
  static const Color black95 = Color(0xF2000000);
}
