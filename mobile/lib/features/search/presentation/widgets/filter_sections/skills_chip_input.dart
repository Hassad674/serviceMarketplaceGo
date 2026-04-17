/// skills_chip_input.dart — free-text chip input with popular-skills
/// quick-add chips. Mirrors the web `SkillsSection` component.
///
/// Features:
///   - Enter or comma commits the draft as a chip.
///   - Backspace on empty draft removes the last chip.
///   - Dedupe is case-insensitive ("React" and "react" do not stack).
///   - Popular skills appear below; tapping one adds the skill and
///     removes it from the popular list until removed.
///   - Haptic feedback on chip add/remove.
///
/// Scope: presentational only. All state lives in the parent filter
/// sheet; this widget receives `selected` + two callbacks.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../../../../shared/search/search_filters.dart';
import 'filter_primitives.dart';

typedef SkillsChanged = void Function(List<String> next);

class SkillsChipInput extends StatefulWidget {
  const SkillsChipInput({
    super.key,
    required this.selected,
    required this.onChanged,
    required this.placeholder,
    required this.semanticsPlaceholder,
  });

  final List<String> selected;
  final SkillsChanged onChanged;
  final String placeholder;
  final String semanticsPlaceholder;

  @override
  State<SkillsChipInput> createState() => _SkillsChipInputState();
}

class _SkillsChipInputState extends State<SkillsChipInput> {
  final TextEditingController _controller = TextEditingController();
  final FocusNode _focus = FocusNode();

  @override
  void dispose() {
    _controller.dispose();
    _focus.dispose();
    super.dispose();
  }

  void _addSkill(String raw) {
    final trimmed = raw.trim();
    if (trimmed.isEmpty) return;
    final lower = trimmed.toLowerCase();
    final alreadyPresent =
        widget.selected.any((s) => s.toLowerCase() == lower);
    if (alreadyPresent) return;
    HapticFeedback.selectionClick();
    widget.onChanged(<String>[...widget.selected, trimmed]);
    _controller.clear();
  }

  void _removeSkill(String value) {
    HapticFeedback.selectionClick();
    widget.onChanged(widget.selected.where((s) => s != value).toList());
  }

  void _handleKey(KeyEvent event) {
    if (event is! KeyDownEvent) return;
    if (event.logicalKey == LogicalKeyboardKey.backspace &&
        _controller.text.isEmpty &&
        widget.selected.isNotEmpty) {
      _removeSkill(widget.selected.last);
    }
  }

  void _handleSubmit(String value) {
    _addSkill(value);
  }

  void _handleChange(String value) {
    // Comma commits as chip (mirrors web).
    if (value.contains(',')) {
      final parts = value.split(',');
      for (final p in parts.take(parts.length - 1)) {
        _addSkill(p);
      }
      _controller.text = parts.last;
      _controller.selection = TextSelection.collapsed(
        offset: _controller.text.length,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (widget.selected.isNotEmpty) ...[
          _SelectedSkillsChips(
            selected: widget.selected,
            onRemove: _removeSkill,
          ),
          const SizedBox(height: 8),
        ],
        KeyboardListener(
          focusNode: FocusNode(skipTraversal: true),
          onKeyEvent: _handleKey,
          child: Semantics(
            textField: true,
            label: widget.semanticsPlaceholder,
            child: TextField(
              controller: _controller,
              focusNode: _focus,
              textInputAction: TextInputAction.done,
              decoration: InputDecoration(
                hintText: widget.placeholder,
                border: const OutlineInputBorder(),
                isDense: true,
              ),
              onChanged: _handleChange,
              onSubmitted: _handleSubmit,
            ),
          ),
        ),
        const SizedBox(height: 8),
        _PopularSkillsChips(
          selected: widget.selected,
          onPick: _addSkill,
        ),
      ],
    );
  }
}

class _SelectedSkillsChips extends StatelessWidget {
  const _SelectedSkillsChips({
    required this.selected,
    required this.onRemove,
  });

  final List<String> selected;
  final ValueChanged<String> onRemove;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: selected
          .map(
            (skill) => InputChip(
              key: ValueKey('selected-skill-$skill'),
              label: Text(skill),
              backgroundColor: kFilterRose100,
              labelStyle: const TextStyle(
                color: kFilterRose700,
                fontWeight: FontWeight.w600,
                fontSize: 12,
              ),
              onDeleted: () => onRemove(skill),
              deleteIconColor: kFilterRose700,
            ),
          )
          .toList(growable: false),
    );
  }
}

class _PopularSkillsChips extends StatelessWidget {
  const _PopularSkillsChips({
    required this.selected,
    required this.onPick,
  });

  final List<String> selected;
  final ValueChanged<String> onPick;

  @override
  Widget build(BuildContext context) {
    final selectedLower = selected.map((s) => s.toLowerCase()).toSet();
    final available = kMobilePopularSkills
        .where((s) => !selectedLower.contains(s.toLowerCase()))
        .toList(growable: false);
    if (available.isEmpty) return const SizedBox.shrink();
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: available
          .map(
            (skill) => FilterPillButton(
              key: ValueKey('popular-skill-$skill'),
              label: skill,
              selected: false,
              onPressed: () => onPick(skill),
              prefix: '+',
              semanticsLabel: 'Add skill $skill',
            ),
          )
          .toList(growable: false),
    );
  }
}
