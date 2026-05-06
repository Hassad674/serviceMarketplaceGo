import 'package:flutter/material.dart';

/// AppPalette — legacy Tailwind-equivalent palette.
///
/// **Deprecated since Soleil v2 (2026-05-05).** New code should pull
/// colors from the active [Theme]:
///
///   - Material 3 tokens via `Theme.of(context).colorScheme.*`
///     (`primary` for corail, `error` for corail-deep, `outline` for
///     borders, `onSurface` for encre, `onSurfaceVariant` for tabac).
///   - Soleil-specific tokens via
///     `Theme.of(context).extension<AppColors>()!.*` (`accentSoft`,
///     `successSoft`, `warning`, `pinkSoft`, `amberSoft`,
///     `primaryDeep`, etc.).
///
/// Each accessor below is annotated `@Deprecated` so the analyzer
/// surfaces a hint when legacy widgets still pull from this palette.
/// The class itself is kept (not deleted) so any straggler reference
/// keeps compiling — a follow-up PR can drop the file once the
/// repo-wide grep returns zero.
class AppPalette {
  AppPalette._();

  // --- Primary brand (Rose, matches web --primary) -------------------------
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer or AppColors.accentSoft — Soleil v2 migration')
  static const Color rose50 = Color(0xFFFFF1F2);
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer or AppColors.accentSoft — Soleil v2 migration')
  static const Color rose100 = Color(0xFFFFE4E6);
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer — Soleil v2 migration')
  static const Color rose200 = Color(0xFFFECDD3);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 migration')
  static const Color rose300 = Color(0xFFFDA4AF);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 migration')
  static const Color rose400 = Color(0xFFFB7185);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 migration')
  static const Color rose500 = Color(0xFFF43F5E);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.primaryDeep ?? Theme.of(context).colorScheme.error) — Soleil v2 migration')
  static const Color rose600 = Color(0xFFE11D48);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.primaryDeep ?? Theme.of(context).colorScheme.error) — Soleil v2 migration')
  static const Color rose700 = Color(0xFFBE123C);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.primaryDeep ?? Theme.of(context).colorScheme.error) — Soleil v2 migration')
  static const Color rose950 = Color(0xFF4C0519);

  // --- Slate (gray) --------------------------------------------------------
  @Deprecated('Use Theme.of(context).colorScheme.surface — Soleil v2 migration')
  static const Color slate50 = Color(0xFFF8FAFC);
  @Deprecated('Use Theme.of(context).colorScheme.surface — Soleil v2 migration')
  static const Color slate100 = Color(0xFFF1F5F9);
  @Deprecated('Use Theme.of(context).colorScheme.outline — Soleil v2 migration')
  static const Color slate200 = Color(0xFFE2E8F0);
  @Deprecated('Use Theme.of(context).colorScheme.outline — Soleil v2 migration')
  static const Color slate300 = Color(0xFFCBD5E1);
  @Deprecated('Use Theme.of(context).colorScheme.outline — Soleil v2 migration')
  static const Color slate400 = Color(0xFF94A3B8);
  @Deprecated('Use Theme.of(context).colorScheme.onSurfaceVariant — Soleil v2 migration')
  static const Color slate500 = Color(0xFF64748B);
  @Deprecated('Use Theme.of(context).colorScheme.onSurfaceVariant — Soleil v2 migration')
  static const Color slate600 = Color(0xFF475569);
  @Deprecated('Use Theme.of(context).colorScheme.onSurfaceVariant — Soleil v2 migration')
  static const Color slate700 = Color(0xFF334155);
  @Deprecated('Use Theme.of(context).colorScheme.onSurface — Soleil v2 migration')
  static const Color slate800 = Color(0xFF1E293B);
  @Deprecated('Use Theme.of(context).colorScheme.onSurface — Soleil v2 migration')
  static const Color slate900 = Color(0xFF0F172A);

  // --- Red / Destructive ---------------------------------------------------
  @Deprecated('Use Theme.of(context).colorScheme.errorContainer — Soleil v2 migration')
  static const Color red100 = Color(0xFFFEE2E2);
  @Deprecated('Use Theme.of(context).colorScheme.errorContainer — Soleil v2 migration')
  static const Color red200 = Color(0xFFFECACA);
  @Deprecated('Use Theme.of(context).colorScheme.error — Soleil v2 migration')
  static const Color red300 = Color(0xFFFCA5A5);
  @Deprecated('Use Theme.of(context).colorScheme.error — Soleil v2 migration')
  static const Color red400 = Color(0xFFF87171);
  @Deprecated('Use Theme.of(context).colorScheme.error — Soleil v2 migration')
  static const Color red500 = Color(0xFFEF4444);
  @Deprecated('Use Theme.of(context).colorScheme.error — Soleil v2 migration')
  static const Color red600 = Color(0xFFDC2626);
  @Deprecated('Use Theme.of(context).colorScheme.error — Soleil v2 migration')
  static const Color red700 = Color(0xFFB91C1C);
  @Deprecated('Use Theme.of(context).colorScheme.error — Soleil v2 migration')
  static const Color red800 = Color(0xFF991B1B);
  @Deprecated('Use Theme.of(context).colorScheme.errorContainer — Soleil v2 migration')
  static const Color red50 = Color(0xFFFEF2F2);

  // --- Amber / Warning -----------------------------------------------------
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.amberSoft ?? Theme.of(context).colorScheme.secondaryContainer) — Soleil v2 migration')
  static const Color amber50 = Color(0xFFFFFBEB);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.amberSoft ?? Theme.of(context).colorScheme.secondaryContainer) — Soleil v2 migration')
  static const Color amber100 = Color(0xFFFEF3C7);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.amberSoft ?? Theme.of(context).colorScheme.secondaryContainer) — Soleil v2 migration')
  static const Color amber200 = Color(0xFFFDE68A);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color amber300 = Color(0xFFFCD34D);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color amber400 = Color(0xFFFBBF24);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color amber500 = Color(0xFFF59E0B);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color amber600 = Color(0xFFD97706);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color amber700 = Color(0xFFB45309);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color amber800 = Color(0xFF92400E);

  // --- Green / Success -----------------------------------------------------
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.successSoft ?? Theme.of(context).colorScheme.primaryContainer) — Soleil v2 migration')
  static const Color green100 = Color(0xFFDCFCE7);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 migration')
  static const Color green500 = Color(0xFF22C55E);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 migration')
  static const Color green600 = Color(0xFF16A34A);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 migration')
  static const Color green700 = Color(0xFF15803D);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 migration')
  static const Color green800 = Color(0xFF166534);

  // --- Emerald (success accent) -------------------------------------------
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.successSoft ?? Theme.of(context).colorScheme.primaryContainer) — Soleil v2 migration')
  static const Color emerald100 = Color(0xFFD1FAE5);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.successSoft ?? Theme.of(context).colorScheme.primaryContainer) — Soleil v2 migration')
  static const Color emerald200 = Color(0xFFA7F3D0);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 migration')
  static const Color emerald300 = Color(0xFF6EE7B7);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 migration')
  static const Color emerald500 = Color(0xFF10B981);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 migration')
  static const Color emerald600 = Color(0xFF059669);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 migration')
  static const Color emerald700 = Color(0xFF047857);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 migration')
  static const Color emerald800 = Color(0xFF065F46);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.successSoft ?? Theme.of(context).colorScheme.primaryContainer) — Soleil v2 migration')
  static const Color emerald50 = Color(0xFFECFDF5);

  // --- Blue -----------------------------------------------------------------
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer — Soleil v2 has no cool tones')
  static const Color blue50 = Color(0xFFEFF6FF);
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer — Soleil v2 has no cool tones')
  static const Color blue100 = Color(0xFFDBEAFE);
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer — Soleil v2 has no cool tones')
  static const Color blue200 = Color(0xFFBFDBFE);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color blue500 = Color(0xFF3B82F6);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color blue600 = Color(0xFF2563EB);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color blue700 = Color(0xFF1D4ED8);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color blue800 = Color(0xFF1E40AF);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color blue900 = Color(0xFF1E3A8A);

  // --- Sky (cyan-ish blue) -------------------------------------------------
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer — Soleil v2 has no cool tones')
  static const Color sky100 = Color(0xFFE0F2FE);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color sky800 = Color(0xFF075985);

  // --- Indigo ---------------------------------------------------------------
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer — Soleil v2 has no cool tones')
  static const Color indigo100 = Color(0xFFE0E7FF);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color indigo700 = Color(0xFF4338CA);

  // --- Violet / Purple -----------------------------------------------------
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer — Soleil v2 has no cool tones')
  static const Color violet50 = Color(0xFFFAF5FF);
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer — Soleil v2 has no cool tones')
  static const Color violet100 = Color(0xFFEDE9FE);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color violet500 = Color(0xFF8B5CF6);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color violet700 = Color(0xFF6D28D9);
  @Deprecated('Use Theme.of(context).colorScheme.primaryContainer — Soleil v2 has no cool tones')
  static const Color purple100 = Color(0xFFF3E8FF);
  @Deprecated('Use Theme.of(context).colorScheme.primary — Soleil v2 has no cool tones')
  static const Color purple700 = Color(0xFF7E22CE);

  // --- Teal -----------------------------------------------------------------
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.success ?? Theme.of(context).colorScheme.primary) — Soleil v2 has no cool tones')
  static const Color teal500 = Color(0xFF14B8A6);

  // --- Orange ---------------------------------------------------------------
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.amberSoft ?? Theme.of(context).colorScheme.secondaryContainer) — Soleil v2 migration')
  static const Color orange100 = Color(0xFFFFEDD5);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.amberSoft ?? Theme.of(context).colorScheme.secondaryContainer) — Soleil v2 migration')
  static const Color orange200 = Color(0xFFFED7AA);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color orange300 = Color(0xFFFDBA74);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.amberSoft ?? Theme.of(context).colorScheme.secondaryContainer) — Soleil v2 migration')
  static const Color orange50 = Color(0xFFFFF7ED);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color orange600 = Color(0xFFEA580C);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color orange700 = Color(0xFFC2410C);
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.warning ?? Theme.of(context).colorScheme.tertiary) — Soleil v2 migration')
  static const Color orange800 = Color(0xFF9A3412);

  // --- Pink (very pale) ---------------------------------------------------
  @Deprecated('Use (Theme.of(context).extension<AppColors>()?.pinkSoft ?? Theme.of(context).colorScheme.surfaceContainerHigh) — Soleil v2 migration')
  static const Color pink100 = Color(0xFFFCE4EC);

  // --- Neutrals -------------------------------------------------------------
  @Deprecated('Use Colors.white directly — Soleil v2 migration')
  static const Color white = Color(0xFFFFFFFF);
  @Deprecated('Use Colors.black directly — Soleil v2 migration')
  static const Color black = Color(0xFF000000);
  @Deprecated('Use Theme.of(context).colorScheme.surface — Soleil v2 migration')
  static const Color neutral50 = Color(0xFFFAFAFA);
  @Deprecated('Use Theme.of(context).colorScheme.onSurface — Soleil v2 migration')
  static const Color gray800 = Color(0xFF333333);

  // --- Brand (third-party social platforms) -------------------------------
  // Note: brand colors stay literal since LinkedIn, Twitter, and
  // Instagram have a single corporate identity not captured by any
  // semantic theme token. Inline as `Color(0xFF...)` per platform.
  @Deprecated('Move inline as Color(0xFF0A66C2) — brand color, not a theme token')
  static const Color linkedinBlue = Color(0xFF0A66C2);
  @Deprecated('Move inline as Color(0xFF1DA1F2) — brand color, not a theme token')
  static const Color twitterBlue = Color(0xFF1DA1F2);
  @Deprecated('Move inline as Color(0xFFE4405F) — brand color, not a theme token')
  static const Color instagramPink = Color(0xFFE4405F);

  // --- Status / Misc -------------------------------------------------------
  @Deprecated('Use Theme.of(context).colorScheme.error — Soleil v2 migration')
  static const Color pureRed = Color(0xFFFF0000);

  // --- Translucent overlays (alpha-aware) ---------------------------------
  /// Black at 80% opacity — used for drop shadows under glassmorphic surfaces.
  @Deprecated('Use Colors.black87 (~85% black) or Colors.black.withValues(alpha:.8)')
  static const Color black80 = Color(0xCC000000);

  /// Black at 95% opacity — used for full-bleed call/video backgrounds.
  @Deprecated('Use Colors.black or Colors.black.withValues(alpha:.95)')
  static const Color black95 = Color(0xF2000000);
}
