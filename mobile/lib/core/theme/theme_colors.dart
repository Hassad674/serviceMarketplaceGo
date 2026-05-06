import 'package:flutter/material.dart';

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
