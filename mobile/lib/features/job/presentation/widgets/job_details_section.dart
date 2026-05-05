import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/job.dart';
import 'applicant_type_selector.dart';

/// M-09 — Soleil v2 job details section.
///
/// Public prop interface (controllers, callbacks, expanded state) is
/// unchanged. Visual identity ports to ivoire/corail with mono labels,
/// rounded inputs and a corail-on-soft chip palette.
class JobDetailsSection extends StatelessWidget {
  const JobDetailsSection({
    super.key,
    required this.titleController,
    required this.descriptionController,
    required this.skills,
    required this.onSkillAdded,
    required this.onSkillRemoved,
    required this.applicantType,
    required this.onApplicantTypeChanged,
    required this.isExpanded,
    required this.onExpansionChanged,
    this.showDescription = true,
    this.hideApplicantType = false,
  });

  final TextEditingController titleController;
  final TextEditingController descriptionController;
  final List<String> skills;
  final ValueChanged<String> onSkillAdded;
  final ValueChanged<int> onSkillRemoved;
  final ApplicantType applicantType;
  final ValueChanged<ApplicantType> onApplicantTypeChanged;
  final bool isExpanded;
  final ValueChanged<bool> onExpansionChanged;
  final bool showDescription;
  final bool hideApplicantType;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return _SoleilSectionCard(
      title: l10n.jobDetails,
      number: 1,
      isExpanded: isExpanded,
      onExpansionChanged: onExpansionChanged,
      children: [
        _MonoLabel(text: l10n.jobTitle),
        const SizedBox(height: 8),
        TextFormField(
          controller: titleController,
          decoration: InputDecoration(
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
        if (showDescription) ...[
          const SizedBox(height: 18),
          _MonoLabel(text: l10n.jobDescription),
          const SizedBox(height: 8),
          TextFormField(
            controller: descriptionController,
            decoration: const InputDecoration(
              alignLabelWithHint: true,
            ),
            maxLines: 5,
            textInputAction: TextInputAction.newline,
            validator: (value) {
              if (!showDescription) return null;
              if (value == null || value.trim().isEmpty) {
                return l10n.fieldRequired;
              }
              return null;
            },
          ),
          const SizedBox(height: 18),
        ] else
          const SizedBox(height: 18),
        _ChipInput(
          label: l10n.jobSkills,
          hintText: l10n.jobSkillsHint,
          items: skills,
          maxItems: 5,
          onAdded: onSkillAdded,
          onRemoved: onSkillRemoved,
        ),
        if (!hideApplicantType) ...[
          const SizedBox(height: 22),
          ApplicantTypeSelector(
            selected: applicantType,
            onChanged: onApplicantTypeChanged,
          ),
        ],
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Mono uppercase label (Soleil signature)
// ---------------------------------------------------------------------------

class _MonoLabel extends StatelessWidget {
  const _MonoLabel({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return Text(
      text.toUpperCase(),
      style: SoleilTextStyles.mono.copyWith(
        color: mute,
        fontSize: 11,
        fontWeight: FontWeight.w700,
        letterSpacing: 0.8,
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Soleil section card with numbered badge + chevron toggle
// ---------------------------------------------------------------------------

class _SoleilSectionCard extends StatelessWidget {
  const _SoleilSectionCard({
    required this.title,
    required this.number,
    required this.isExpanded,
    required this.onExpansionChanged,
    required this.children,
  });

  final String title;
  final int number;
  final bool isExpanded;
  final ValueChanged<bool> onExpansionChanged;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final accentSoft = appColors?.accentSoft ?? theme.colorScheme.primaryContainer;
    final primaryDeep = appColors?.primaryDeep ?? primary;
    final border = appColors?.border ?? theme.colorScheme.outline;
    final borderStrong = appColors?.borderStrong ?? theme.colorScheme.outline;
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      curve: Curves.easeOut,
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: isExpanded ? borderStrong : border,
        ),
        boxShadow: isExpanded ? AppTheme.cardShadow : null,
      ),
      child: Column(
        children: [
          InkWell(
            onTap: () => onExpansionChanged(!isExpanded),
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            child: Padding(
              padding: const EdgeInsets.fromLTRB(20, 18, 20, 18),
              child: Row(
                children: [
                  Container(
                    width: 32,
                    height: 32,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: isExpanded ? accentSoft : theme.colorScheme.surface,
                      border: Border.all(
                        color: isExpanded ? primary : borderStrong,
                      ),
                    ),
                    child: Text(
                      '$number',
                      style: SoleilTextStyles.mono.copyWith(
                        color: isExpanded ? primaryDeep : mute,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Text(
                      title,
                      style: SoleilTextStyles.titleMedium,
                    ),
                  ),
                  AnimatedRotation(
                    turns: isExpanded ? 0.25 : 0,
                    duration: const Duration(milliseconds: 200),
                    child: Icon(
                      Icons.chevron_right,
                      color: mute,
                    ),
                  ),
                ],
              ),
            ),
          ),
          AnimatedCrossFade(
            firstChild: const SizedBox.shrink(),
            secondChild: Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 22),
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
// Reusable chip input for skills — Soleil-styled (corail-soft chips on mute
// border, "+" button on corail surface)
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
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final primaryDeep = appColors?.primaryDeep ?? primary;
    final accentSoft = appColors?.accentSoft ?? theme.colorScheme.primaryContainer;
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;
    final canAdd = widget.items.length < widget.maxItems;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _MonoLabel(text: widget.label),
        const SizedBox(height: 10),
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
                ),
                textInputAction: TextInputAction.done,
                onFieldSubmitted: (_) => _addItem(),
              ),
            ),
            const SizedBox(width: 10),
            Material(
              color: canAdd ? primary : primary.withValues(alpha: 0.4),
              shape: const StadiumBorder(),
              child: InkWell(
                customBorder: const StadiumBorder(),
                onTap: canAdd ? _addItem : null,
                child: const SizedBox(
                  width: 48,
                  height: 48,
                  child: Icon(Icons.add, size: 22, color: Colors.white),
                ),
              ),
            ),
          ],
        ),
        if (widget.items.isNotEmpty) ...[
          const SizedBox(height: 14),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (int i = 0; i < widget.items.length; i++)
                Chip(
                  label: Text(
                    widget.items[i],
                    style: SoleilTextStyles.bodyEmphasis.copyWith(
                      color: primaryDeep,
                      fontSize: 12.5,
                    ),
                  ),
                  deleteIcon: Icon(
                    Icons.close,
                    size: 14,
                    color: primaryDeep,
                  ),
                  onDeleted: () => widget.onRemoved(i),
                  backgroundColor: accentSoft,
                  side: BorderSide(color: primary.withValues(alpha: 0.18)),
                  shape: const StadiumBorder(),
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 0),
                  visualDensity: VisualDensity.compact,
                ),
            ],
          ),
        ] else if (mute != Colors.transparent)
          const SizedBox.shrink(),
      ],
    );
  }
}
