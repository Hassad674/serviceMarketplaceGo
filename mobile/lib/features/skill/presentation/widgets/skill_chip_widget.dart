import 'package:flutter/material.dart';

/// Removable chip used in the editor's "selected" Wrap.
///
/// Renders an [InputChip] so we get the Material ripple, 48dp
/// tap target, and accessibility labels for free. Tapping the
/// delete icon calls [onDeleted]; tapping the chip body does
/// nothing (selection is one-way — the chip is already selected).
class SkillChipWidget extends StatelessWidget {
  const SkillChipWidget({
    super.key,
    required this.label,
    this.onDeleted,
  });

  final String label;
  final VoidCallback? onDeleted;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    return InputChip(
      label: Text(label),
      labelStyle: TextStyle(
        color: primary,
        fontWeight: FontWeight.w600,
        fontSize: 13,
      ),
      backgroundColor: primary.withValues(alpha: 0.1),
      side: BorderSide(color: primary.withValues(alpha: 0.2)),
      deleteIconColor: primary,
      onDeleted: onDeleted,
      materialTapTargetSize: MaterialTapTargetSize.padded,
      visualDensity: VisualDensity.compact,
    );
  }
}

/// Tappable chip used in the catalog browser and the "popular"
/// row. Unlike [SkillChipWidget], this one toggles selection on
/// tap and has no delete icon.
class SelectableSkillChip extends StatelessWidget {
  const SelectableSkillChip({
    super.key,
    required this.label,
    required this.onTap,
    this.subtitle,
    this.disabled = false,
  });

  final String label;
  final String? subtitle;
  final VoidCallback onTap;
  final bool disabled;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    return Material(
      color: disabled
          ? colors.surfaceContainerHighest.withValues(alpha: 0.4)
          : colors.surface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: colors.outlineVariant),
      ),
      child: InkWell(
        onTap: disabled ? null : onTap,
        customBorder: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
        ),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                label,
                style: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w500,
                  color: disabled ? theme.disabledColor : null,
                ),
              ),
              if (subtitle != null) ...[
                const SizedBox(width: 6),
                Text(
                  subtitle!,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.hintColor,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
