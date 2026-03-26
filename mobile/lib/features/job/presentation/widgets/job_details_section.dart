import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/job.dart';
import 'applicant_type_selector.dart';
import 'contractor_counter.dart';

/// Section 1: Job details — title, description, skills, tools,
/// contractor count, and applicant type.
class JobDetailsSection extends StatelessWidget {
  const JobDetailsSection({
    super.key,
    required this.titleController,
    required this.descriptionController,
    required this.skills,
    required this.onSkillAdded,
    required this.onSkillRemoved,
    required this.tools,
    required this.onToolAdded,
    required this.onToolRemoved,
    required this.contractorCount,
    required this.onContractorCountChanged,
    required this.applicantType,
    required this.onApplicantTypeChanged,
    required this.isExpanded,
    required this.onExpansionChanged,
  });

  final TextEditingController titleController;
  final TextEditingController descriptionController;
  final List<String> skills;
  final ValueChanged<String> onSkillAdded;
  final ValueChanged<int> onSkillRemoved;
  final List<String> tools;
  final ValueChanged<String> onToolAdded;
  final ValueChanged<int> onToolRemoved;
  final int contractorCount;
  final ValueChanged<int> onContractorCountChanged;
  final ApplicantType applicantType;
  final ValueChanged<ApplicantType> onApplicantTypeChanged;
  final bool isExpanded;
  final ValueChanged<bool> onExpansionChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return _buildExpandableContainer(
      context: context,
      theme: theme,
      appColors: appColors,
      title: l10n.jobDetails,
      icon: Icons.description_outlined,
      isExpanded: isExpanded,
      onExpansionChanged: onExpansionChanged,
      children: [
        // Title
        TextFormField(
          controller: titleController,
          decoration: InputDecoration(
            labelText: l10n.jobTitle,
            hintText: l10n.jobTitleHint,
          ),
          maxLength: 100,
          textInputAction: TextInputAction.next,
          validator: (value) {
            if (value == null || value.trim().isEmpty) {
              return l10n.fieldRequired;
            }
            return null;
          },
        ),
        const SizedBox(height: 16),

        // Description
        TextFormField(
          controller: descriptionController,
          decoration: InputDecoration(
            labelText: l10n.jobDescription,
            alignLabelWithHint: true,
          ),
          maxLines: 5,
          textInputAction: TextInputAction.newline,
          validator: (value) {
            if (value == null || value.trim().isEmpty) {
              return l10n.fieldRequired;
            }
            return null;
          },
        ),
        const SizedBox(height: 20),

        // Skills
        _ChipInput(
          label: l10n.jobSkills,
          hintText: l10n.jobSkillsHint,
          items: skills,
          maxItems: 5,
          onAdded: onSkillAdded,
          onRemoved: onSkillRemoved,
        ),
        const SizedBox(height: 20),

        // Tools
        _ChipInput(
          label: l10n.jobTools,
          hintText: l10n.jobToolsHint,
          items: tools,
          maxItems: 5,
          onAdded: onToolAdded,
          onRemoved: onToolRemoved,
        ),
        const SizedBox(height: 20),

        // Contractor count
        ContractorCounter(
          value: contractorCount,
          onChanged: onContractorCountChanged,
        ),
        const SizedBox(height: 20),

        // Applicant type
        ApplicantTypeSelector(
          selected: applicantType,
          onChanged: onApplicantTypeChanged,
        ),
      ],
    );
  }

  Widget _buildExpandableContainer({
    required BuildContext context,
    required ThemeData theme,
    required AppColors? appColors,
    required String title,
    required IconData icon,
    required bool isExpanded,
    required ValueChanged<bool> onExpansionChanged,
    required List<Widget> children,
  }) {
    final primary = theme.colorScheme.primary;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      curve: Curves.easeOut,
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: isExpanded
              ? primary.withValues(alpha: 0.3)
              : appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: isExpanded ? AppTheme.cardShadow : null,
      ),
      child: Column(
        children: [
          // Header
          InkWell(
            onTap: () => onExpansionChanged(!isExpanded),
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  Container(
                    width: 40,
                    height: 40,
                    decoration: BoxDecoration(
                      color: primary.withValues(alpha: 0.1),
                      borderRadius:
                          BorderRadius.circular(AppTheme.radiusSm),
                    ),
                    child: Icon(icon, color: primary, size: 20),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      title,
                      style: theme.textTheme.titleMedium,
                    ),
                  ),
                  AnimatedRotation(
                    turns: isExpanded ? 0.5 : 0,
                    duration: const Duration(milliseconds: 200),
                    child: Icon(
                      Icons.keyboard_arrow_down,
                      color: appColors?.mutedForeground ??
                          theme.colorScheme.onSurface,
                    ),
                  ),
                ],
              ),
            ),
          ),
          // Content
          AnimatedCrossFade(
            firstChild: const SizedBox.shrink(),
            secondChild: Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: children,
              ),
            ),
            crossFadeState: isExpanded
                ? CrossFadeState.showSecond
                : CrossFadeState.showFirst,
            duration: const Duration(milliseconds: 200),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Reusable chip input for skills and tools
// ---------------------------------------------------------------------------

class _ChipInput extends StatefulWidget {
  const _ChipInput({
    required this.label,
    required this.hintText,
    required this.items,
    required this.maxItems,
    required this.onAdded,
    required this.onRemoved,
  });

  final String label;
  final String hintText;
  final List<String> items;
  final int maxItems;
  final ValueChanged<String> onAdded;
  final ValueChanged<int> onRemoved;

  @override
  State<_ChipInput> createState() => _ChipInputState();
}

class _ChipInputState extends State<_ChipInput> {
  final _controller = TextEditingController();

  void _addItem() {
    final text = _controller.text.trim();
    if (text.isEmpty) return;
    if (widget.items.contains(text)) return;
    if (widget.items.length >= widget.maxItems) return;
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
    final primary = theme.colorScheme.primary;
    final canAdd = widget.items.length < widget.maxItems;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(widget.label, style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        Row(
          children: [
            Expanded(
              child: TextFormField(
                controller: _controller,
                enabled: canAdd,
                decoration: InputDecoration(
                  hintText: canAdd
                      ? widget.hintText
                      : '${widget.maxItems} max',
                  contentPadding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 12,
                  ),
                ),
                textInputAction: TextInputAction.done,
                onFieldSubmitted: (_) => _addItem(),
              ),
            ),
            const SizedBox(width: 8),
            IconButton.filled(
              onPressed: canAdd ? _addItem : null,
              icon: const Icon(Icons.add, size: 20),
              style: IconButton.styleFrom(
                backgroundColor: canAdd ? primary : primary.withValues(alpha: 0.3),
                foregroundColor: Colors.white,
              ),
            ),
          ],
        ),
        if (widget.items.isNotEmpty) ...[
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (int i = 0; i < widget.items.length; i++)
                Chip(
                  label: Text(
                    widget.items[i],
                    style: TextStyle(
                      color: primary,
                      fontSize: 13,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  deleteIcon: Icon(
                    Icons.close,
                    size: 16,
                    color: primary,
                  ),
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
