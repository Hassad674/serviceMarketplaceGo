/// filter_primitives.dart — tiny, design-system-friendly shared
/// widgets used across every section of the mobile filter sheet.
///
/// Kept private to the search feature so theming changes stay
/// local. Tests instantiate these directly via their public
/// constructors.
library;

import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../core/theme/app_palette.dart';

/// Rose-500 — the marketplace primary color. Kept local to the
/// filter sheet so we do not accidentally couple unrelated widgets
/// to a hardcoded constant — consumers should always prefer
/// `Theme.of(context).colorScheme.primary`. Used here only because
/// AppColors does not expose a `primary` slot.
const Color kFilterRose500 = AppPalette.rose500;
const Color kFilterRose100 = AppPalette.rose100;
const Color kFilterRose700 = AppPalette.rose700;

/// Section shell with a 12 px caps header + the sub-widget. Spacing
/// matches the 4 px design grid (8/16/20 increments only).
class FilterSectionShell extends StatelessWidget {
  const FilterSectionShell({
    super.key,
    required this.title,
    required this.child,
  });

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final appColors = Theme.of(context).extension<AppColors>();
    return Padding(
      padding: const EdgeInsets.only(bottom: 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              title.toUpperCase(),
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w700,
                letterSpacing: 0.4,
                color: appColors?.mutedForeground,
              ),
            ),
          ),
          child,
        ],
      ),
    );
  }
}

/// Pill button — rose background when selected, neutral outline
/// when not. Used by availability, work mode, languages, popular
/// skills sections.
class FilterPillButton extends StatelessWidget {
  const FilterPillButton({
    super.key,
    required this.label,
    required this.selected,
    required this.onPressed,
    this.prefix,
    this.semanticsLabel,
  });

  final String label;
  final bool selected;
  final VoidCallback onPressed;
  final String? prefix;
  final String? semanticsLabel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Semantics(
      button: true,
      selected: selected,
      label: semanticsLabel ?? label,
      child: Material(
        color: selected ? kFilterRose100 : theme.colorScheme.surface,
        shape: RoundedRectangleBorder(
          side: BorderSide(
            color: selected
                ? kFilterRose500
                : (appColors?.border ?? theme.dividerColor),
          ),
          borderRadius: BorderRadius.circular(999),
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(999),
          onTap: onPressed,
          child: Padding(
            padding:
                const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
            child: Text(
              prefix != null ? '$prefix $label' : label,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w600,
                color: selected
                    ? kFilterRose700
                    : theme.colorScheme.onSurface,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Checkbox row used by the expertise section.
class FilterCheckboxRow extends StatelessWidget {
  const FilterCheckboxRow({
    super.key,
    required this.label,
    required this.checked,
    required this.onChanged,
  });

  final String label;
  final bool checked;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: () => onChanged(!checked),
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(
          children: [
            SizedBox(
              width: 24,
              height: 24,
              child: Checkbox(
                value: checked,
                onChanged: (v) => onChanged(v ?? false),
                activeColor: kFilterRose500,
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                label,
                style: const TextStyle(fontSize: 14),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Themed number input — integer-only, clamped to [0, 9999999].
class FilterNumberField extends StatefulWidget {
  const FilterNumberField({
    super.key,
    required this.value,
    required this.onChanged,
    required this.label,
    required this.semanticsLabel,
  });

  final int? value;
  final ValueChanged<int?> onChanged;
  final String label;
  final String semanticsLabel;

  @override
  State<FilterNumberField> createState() => _FilterNumberFieldState();
}

class _FilterNumberFieldState extends State<FilterNumberField> {
  late final TextEditingController _controller;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController(
      text: widget.value?.toString() ?? '',
    );
  }

  @override
  void didUpdateWidget(covariant FilterNumberField oldWidget) {
    super.didUpdateWidget(oldWidget);
    final newText = widget.value?.toString() ?? '';
    if (_controller.text != newText) {
      _controller.value = TextEditingValue(
        text: newText,
        selection: TextSelection.collapsed(offset: newText.length),
      );
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: widget.semanticsLabel,
      textField: true,
      child: TextField(
        controller: _controller,
        keyboardType: TextInputType.number,
        decoration: InputDecoration(
          labelText: widget.label,
          border: const OutlineInputBorder(),
          isDense: true,
        ),
        onChanged: (raw) {
          final trimmed = raw.trim();
          if (trimmed.isEmpty) {
            widget.onChanged(null);
            return;
          }
          final n = int.tryParse(trimmed);
          if (n == null || n < 0) return;
          widget.onChanged(n.clamp(0, 9999999));
        },
      ),
    );
  }
}
