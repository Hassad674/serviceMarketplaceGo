import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/proposal.dart';

/// Soleil v2 — Mobile milestone editor for the create proposal form.
///
/// Mirrors the web `MilestoneEditor` component:
///   * Repeatable card-per-milestone (title, optional description,
///     amount, due date).
///   * Min [kMinMilestonesPerMilestoneProposal] rows — the trash icon
///     is hidden below the floor.
///   * Max [kMaxMilestonesPerProposal] — the "Add" button disables
///     above the cap.
///   * Sticky footer summing milestone amounts in real time.
class MilestoneEditorWidget extends StatelessWidget {
  const MilestoneEditorWidget({
    super.key,
    required this.milestones,
    required this.onChanged,
    this.disabled = false,
  });

  final List<MilestoneFormItem> milestones;
  final ValueChanged<List<MilestoneFormItem>> onChanged;
  final bool disabled;

  void _updateAt(int index, void Function(MilestoneFormItem) patch) {
    final next = milestones.map((m) => m).toList();
    patch(next[index]);
    onChanged(next);
  }

  void _addMilestone() {
    if (milestones.length >= kMaxMilestonesPerProposal) return;
    onChanged([...milestones, MilestoneFormItem()]);
  }

  void _removeAt(int index) {
    if (milestones.length <= kMinMilestonesPerMilestoneProposal) return;
    final next = [...milestones]..removeAt(index);
    onChanged(next);
  }

  double get _totalEuros {
    var total = 0.0;
    for (final m in milestones) {
      if (m.amount > 0) total += m.amount;
    }
    return total;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final canAdd =
        milestones.length < kMaxMilestonesPerProposal && !disabled;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.milestoneEditorMinimumHint(
            kMinMilestonesPerMilestoneProposal,
          ),
          style: SoleilTextStyles.caption.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 16),
        for (var i = 0; i < milestones.length; i++) ...[
          _MilestoneCard(
            sequence: i + 1,
            milestone: milestones[i],
            disabled: disabled,
            canRemove:
                milestones.length > kMinMilestonesPerMilestoneProposal,
            onTitleChanged: (v) =>
                _updateAt(i, (m) => m.title = v),
            onDescriptionChanged: (v) =>
                _updateAt(i, (m) => m.description = v),
            onAmountChanged: (v) => _updateAt(i, (m) => m.amount = v),
            onDeadlineChanged: (v) =>
                _updateAt(i, (m) => m.deadline = v),
            onRemove: () => _removeAt(i),
          ),
          const SizedBox(height: 12),
        ],
        OutlinedButton.icon(
          onPressed: canAdd ? _addMilestone : null,
          style: OutlinedButton.styleFrom(
            shape: const StadiumBorder(),
            minimumSize: const Size.fromHeight(48),
            side: BorderSide(
              color: canAdd
                  ? theme.colorScheme.primary
                  : theme.dividerColor,
              width: 2,
              style: BorderStyle.solid,
            ),
          ),
          icon: const Icon(Icons.add_rounded),
          label: Text(l10n.milestoneEditorAdd),
        ),
        const SizedBox(height: 16),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
          decoration: BoxDecoration(
            color: theme.colorScheme.primaryContainer,
            borderRadius: BorderRadius.circular(AppTheme.radiusXl),
            border: Border.all(color: theme.colorScheme.primary),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                l10n.milestoneEditorTotal,
                style: SoleilTextStyles.mono.copyWith(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 1.2,
                ),
              ),
              Text(
                '${_totalEuros.toStringAsFixed(2)} €',
                style: SoleilTextStyles.mono.copyWith(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.w700,
                  fontSize: 22,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _MilestoneCard extends StatefulWidget {
  const _MilestoneCard({
    required this.sequence,
    required this.milestone,
    required this.disabled,
    required this.canRemove,
    required this.onTitleChanged,
    required this.onDescriptionChanged,
    required this.onAmountChanged,
    required this.onDeadlineChanged,
    required this.onRemove,
  });

  final int sequence;
  final MilestoneFormItem milestone;
  final bool disabled;
  final bool canRemove;
  final ValueChanged<String> onTitleChanged;
  final ValueChanged<String> onDescriptionChanged;
  final ValueChanged<double> onAmountChanged;
  final ValueChanged<DateTime?> onDeadlineChanged;
  final VoidCallback onRemove;

  @override
  State<_MilestoneCard> createState() => _MilestoneCardState();
}

class _MilestoneCardState extends State<_MilestoneCard> {
  late final TextEditingController _titleController;
  late final TextEditingController _descriptionController;
  late final TextEditingController _amountController;

  @override
  void initState() {
    super.initState();
    _titleController = TextEditingController(text: widget.milestone.title);
    _descriptionController =
        TextEditingController(text: widget.milestone.description);
    _amountController = TextEditingController(
      text: widget.milestone.amount > 0
          ? widget.milestone.amount.toStringAsFixed(2)
          : '',
    );
  }

  @override
  void dispose() {
    _titleController.dispose();
    _descriptionController.dispose();
    _amountController.dispose();
    super.dispose();
  }

  Future<void> _pickDeadline() async {
    final now = DateTime.now();
    final picked = await showDatePicker(
      context: context,
      initialDate: widget.milestone.deadline ?? now,
      firstDate: now,
      lastDate: now.add(const Duration(days: 730)),
    );
    if (picked != null) {
      widget.onDeadlineChanged(picked);
      setState(() {});
    }
  }

  String _formatDate(DateTime d) {
    final dd = d.day.toString().padLeft(2, '0');
    final mm = d.month.toString().padLeft(2, '0');
    return '$dd/$mm/${d.year}';
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                decoration: BoxDecoration(
                  color: theme.colorScheme.primaryContainer,
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  '${l10n.milestoneEditorMilestone} ${widget.sequence}',
                  style: SoleilTextStyles.mono.copyWith(
                    color: theme.colorScheme.primary,
                    fontWeight: FontWeight.w700,
                    fontSize: 11,
                    letterSpacing: 0.8,
                  ),
                ),
              ),
              if (widget.canRemove)
                IconButton(
                  onPressed: widget.disabled ? null : widget.onRemove,
                  icon: const Icon(Icons.delete_outline_rounded, size: 20),
                  color: theme.colorScheme.error,
                  tooltip: l10n.milestoneEditorRemove,
                ),
            ],
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _titleController,
            decoration: InputDecoration(
              labelText: l10n.milestoneEditorTitleLabel,
              hintText: l10n.milestoneEditorTitleHint,
            ),
            enabled: !widget.disabled,
            onChanged: widget.onTitleChanged,
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _descriptionController,
            decoration: InputDecoration(
              labelText: l10n.milestoneEditorDescriptionLabel,
              hintText: l10n.milestoneEditorDescriptionHint,
            ),
            enabled: !widget.disabled,
            maxLines: 2,
            onChanged: widget.onDescriptionChanged,
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _amountController,
                  decoration: InputDecoration(
                    labelText: l10n.milestoneEditorAmountLabel,
                    prefixText: '€ ',
                  ),
                  enabled: !widget.disabled,
                  keyboardType: TextInputType.number,
                  onChanged: (v) {
                    final parsed = double.tryParse(v) ?? 0;
                    widget.onAmountChanged(parsed);
                  },
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: GestureDetector(
                  onTap: widget.disabled ? null : _pickDeadline,
                  child: AbsorbPointer(
                    child: TextField(
                      decoration: InputDecoration(
                        labelText: l10n.milestoneEditorDeadlineLabel,
                        suffixIcon: Icon(
                          Icons.calendar_today_rounded,
                          size: 18,
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                      controller: TextEditingController(
                        text: widget.milestone.deadline != null
                            ? _formatDate(widget.milestone.deadline!)
                            : '',
                      ),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
