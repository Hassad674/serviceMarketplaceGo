import 'package:flutter/material.dart';

/// Read-only display of a profile's declared skills as pill chips.
///
/// Owned by the search feature — NOT the skill feature — so search
/// results and public profiles can render skills without pulling in
/// the skill editor package. This preserves feature independence:
/// the `skill` feature remains fully removable even though skills
/// are shown on search surfaces.
///
/// The widget accepts the raw JSON-shaped list from the API
/// (`{skill_text, display_text}`) to minimize DTO conversion at the
/// call site. When the list is null or empty it collapses to a
/// `SizedBox.shrink()` — consumers hide the whole section.
class SkillsDisplayWidget extends StatelessWidget {
  const SkillsDisplayWidget({
    super.key,
    required this.skills,
    this.maxVisible,
  });

  /// Flat JSON-shaped skills from the public profile payload. Each
  /// entry is `{skill_text, display_text}`. `display_text` is shown
  /// to the user; `skill_text` is the lowercase key used as a
  /// fallback when `display_text` is missing.
  final List<Map<String, dynamic>>? skills;

  /// Maximum number of chips to render. When the list exceeds this
  /// cap, the first [maxVisible] skills are shown followed by a
  /// muted "+N" overflow chip. Pass `null` to render every skill —
  /// the public profile uses null, provider cards use 4.
  final int? maxVisible;

  @override
  Widget build(BuildContext context) {
    final list = skills;
    if (list == null || list.isEmpty) return const SizedBox.shrink();

    final cap = maxVisible;
    final hasOverflow = cap != null && list.length > cap;
    final visible = hasOverflow ? list.take(cap).toList() : list;
    final overflowCount = hasOverflow ? list.length - cap : 0;

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        for (final skill in visible) _SkillPill(label: _labelFor(skill)),
        if (overflowCount > 0) _OverflowPill(count: overflowCount),
      ],
    );
  }

  /// Resolves the user-facing label for a single skill entry.
  /// Prefers `display_text`, falls back to `skill_text`, and finally
  /// to an empty string so the widget never throws on a malformed
  /// payload during a rolling deploy.
  String _labelFor(Map<String, dynamic> skill) {
    final displayText = skill['display_text'] as String?;
    if (displayText != null && displayText.isNotEmpty) return displayText;
    final skillText = skill['skill_text'] as String?;
    return skillText ?? '';
  }
}

// ---------------------------------------------------------------------------
// Private pill — rose-tinted, read-only chip
// Matches ExpertiseDisplayWidget's pill style to keep the two sections
// visually consistent on the public profile.
// ---------------------------------------------------------------------------

class _SkillPill extends StatelessWidget {
  const _SkillPill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;

    return Semantics(
      label: label,
      container: true,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: primary.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: primary.withValues(alpha: 0.2)),
        ),
        child: Text(
          label,
          style: TextStyle(
            color: primary,
            fontWeight: FontWeight.w600,
            fontSize: 13,
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Overflow pill — neutral-tinted "+N" marker shown on capped lists
// ---------------------------------------------------------------------------

class _OverflowPill extends StatelessWidget {
  const _OverflowPill({required this.count});

  final int count;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final onSurfaceVariant = theme.colorScheme.onSurfaceVariant;
    final label = '+$count';

    return Semantics(
      label: label,
      container: true,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: onSurfaceVariant.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: onSurfaceVariant.withValues(alpha: 0.2),
          ),
        ),
        child: Text(
          label,
          style: TextStyle(
            color: onSurfaceVariant,
            fontWeight: FontWeight.w600,
            fontSize: 13,
          ),
        ),
      ),
    );
  }
}
