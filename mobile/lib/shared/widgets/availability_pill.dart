import 'package:flutter/material.dart';

import '../../core/theme/app_theme.dart';
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
    final tone = _toneFor(context, wireValue);
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

  _PillTone _toneFor(BuildContext context, String wireValue) {
    final cs = Theme.of(context).colorScheme;
    final ext = Theme.of(context).extension<AppColors>();
    final amberSoft = ext?.amberSoft ?? cs.secondaryContainer;
    final warning = ext?.warning ?? cs.tertiary;
    final successSoft = ext?.successSoft ?? cs.primaryContainer;
    final success = ext?.success ?? cs.primary;
    switch (wireValue) {
      case 'available_soon':
        return _PillTone(
          background: amberSoft,
          border: amberSoft,
          foreground: warning,
          dot: warning,
        );
      case 'not_available':
        return _PillTone(
          background: cs.errorContainer,
          border: cs.errorContainer,
          foreground: cs.error,
          dot: cs.error,
        );
      case 'available_now':
      default:
        return _PillTone(
          background: successSoft,
          border: successSoft,
          foreground: success,
          dot: success,
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
