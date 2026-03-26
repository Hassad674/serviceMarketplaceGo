import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

/// Chip-based skill input for the proposal form.
///
/// Replicates the pattern from project creation but scoped to proposals.
class ProposalSkillsInput extends StatefulWidget {
  const ProposalSkillsInput({
    super.key,
    required this.skills,
    required this.onAdded,
    required this.onRemoved,
  });

  final List<String> skills;
  final ValueChanged<String> onAdded;
  final ValueChanged<int> onRemoved;

  @override
  State<ProposalSkillsInput> createState() => _ProposalSkillsInputState();
}

class _ProposalSkillsInputState extends State<ProposalSkillsInput> {
  final _controller = TextEditingController();

  void _addSkill() {
    final text = _controller.text.trim();
    if (text.isEmpty) return;
    if (widget.skills.contains(text)) return;
    widget.onAdded(text);
    _controller.clear();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final primary = theme.colorScheme.primary;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: TextFormField(
                controller: _controller,
                decoration: InputDecoration(
                  hintText: l10n.addSkillHint,
                  contentPadding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 12,
                  ),
                ),
                textInputAction: TextInputAction.done,
                onFieldSubmitted: (_) => _addSkill(),
              ),
            ),
            const SizedBox(width: 8),
            IconButton.filled(
              onPressed: _addSkill,
              icon: const Icon(Icons.add, size: 20),
              style: IconButton.styleFrom(
                backgroundColor: primary,
                foregroundColor: Colors.white,
              ),
            ),
          ],
        ),
        if (widget.skills.isNotEmpty) ...[
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (int i = 0; i < widget.skills.length; i++)
                Chip(
                  label: Text(
                    widget.skills[i],
                    style: TextStyle(
                      color: primary,
                      fontSize: 13,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  deleteIcon: Icon(Icons.close, size: 16, color: primary),
                  onDeleted: () => widget.onRemoved(i),
                  backgroundColor: primary.withValues(alpha: 0.08),
                  side: BorderSide(
                    color: primary.withValues(alpha: 0.2),
                  ),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                  ),
                ),
            ],
          ),
        ],
      ],
    );
  }
}
