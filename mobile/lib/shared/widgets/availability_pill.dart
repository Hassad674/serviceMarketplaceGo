import 'package:flutter/material.dart';
import '../../core/theme/app_palette.dart';

/// Generic availability pill used by both freelance and referrer
/// profile headers. Renders a colored badge based on a wire value
/// (e.g. `available_now`, `available_soon`, `not_available`). Pure
/// display widget — no business logic, no repository access, no
/// feature imports.
///
/// Callers pass the wire value and the localized label so the pill
/// can stay free of the l10n generated code and be reused from any
/// feature that needs a compact availability badge.
class AvailabilityPill extends StatelessWidget {
  const AvailabilityPill({
    super.key,
    required this.wireValue,
    required this.label,
    this.compact = false,
  });

  /// Backend-facing string (`available_now`, `available_soon`,
  /// `not_available`). Any unknown value falls back to the "now"
  /// styling so the pill never renders an empty badge.
  final String wireValue;

  /// Localized user-facing label (e.g. "Available now").
  final String label;

  /// When true the pill uses a more compact padding — useful in
  /// dense headers where horizontal space is tight.
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final tone = _toneFor(wireValue);
    final horizontalPadding = compact ? 8.0 : 12.0;
    final verticalPadding = compact ? 4.0 : 6.0;

    return Container(
      padding: EdgeInsets.symmetric(
        horizontal: horizontalPadding,
        vertical: verticalPadding,
      ),
      decoration: BoxDecoration(
        color: tone.background,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: tone.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(
              color: tone.dot,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 6),
          Text(
            label,
            style: theme.textTheme.labelSmall?.copyWith(
              color: tone.foreground,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }

  _PillTone _toneFor(String wireValue) {
    switch (wireValue) {
      case 'available_soon':
        return const _PillTone(
          background: AppPalette.amber50, // amber-50
          border: AppPalette.amber200, // amber-200
          foreground: AppPalette.amber700, // amber-700
          dot: AppPalette.amber500, // amber-500
        );
      case 'not_available':
        return const _PillTone(
          background: AppPalette.red50, // red-50
          border: AppPalette.red200, // red-200
          foreground: AppPalette.red700, // red-700
          dot: AppPalette.red500, // red-500
        );
      case 'available_now':
      default:
        return const _PillTone(
          background: AppPalette.emerald50, // emerald-50
          border: AppPalette.emerald200, // emerald-200
          foreground: AppPalette.emerald700, // emerald-700
          dot: AppPalette.emerald500, // emerald-500
        );
    }
  }
}

class _PillTone {
  const _PillTone({
    required this.background,
    required this.border,
    required this.foreground,
    required this.dot,
  });

  final Color background;
  final Color border;
  final Color foreground;
  final Color dot;
}
