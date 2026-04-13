import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/profile_skill.dart';
import '../../domain/entities/skill_limits.dart';
import '../providers/profile_skills_provider.dart';
import 'skills_editor_bottom_sheet.dart';

/// Inline skills card rendered on the profile screen, right after
/// the expertise card.
///
/// Responsibilities:
/// - Hide itself entirely when the feature is disabled (enterprise).
/// - Render a loading skeleton while the first fetch is in flight.
/// - Render a read-only list of rose-tinted chips for the current
///   list of profile skills (or an empty-state placeholder).
/// - Offer an "Edit my skills" button that opens
///   [SkillsEditorBottomSheet] when the operator can edit.
///
/// Holds NO local draft state — the editor manages its own buffer
/// and the StateNotifier fetches the fresh list after a save. The
/// parent profile screen invalidates its own provider via [onSaved]
/// so the header card refreshes in lockstep.
class SkillsSectionWidget extends ConsumerWidget {
  const SkillsSectionWidget({
    super.key,
    required this.orgType,
    required this.expertiseKeys,
    required this.canEdit,
    this.onSaved,
  });

  /// Current org type — drives the feature gate and max count.
  final String? orgType;

  /// Expertise domain keys declared on the profile. Passed to the
  /// editor so it can render popular / browse-by-domain sections.
  final List<String> expertiseKeys;

  /// Whether the operator has permission to modify the profile.
  /// When `false`, the read-only chip list still renders but the
  /// edit button is hidden.
  final bool canEdit;

  /// Invoked after a successful save so the parent can invalidate
  /// its profile provider. Optional so tests can skip it.
  final VoidCallback? onSaved;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (!SkillLimits.isFeatureEnabledForOrgType(orgType)) {
      return const SizedBox.shrink();
    }

    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final max = SkillLimits.maxForOrgType(orgType);
    final state = ref.watch(profileSkillsProvider);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _Header(
            max: max,
            count: state.skills.maybeWhen(
              data: (list) => list.length,
              orElse: () => 0,
            ),
          ),
          const SizedBox(height: 12),
          state.skills.when(
            loading: () => const _LoadingState(),
            error: (_, __) => const _ErrorState(),
            data: (list) => _DataState(skills: list),
          ),
          if (canEdit) ...[
            const SizedBox(height: 12),
            _EditButton(
              label: l10n.skillsEditButton,
              onTap: () => _openEditor(
                context,
                ref,
                state.skills.maybeWhen(
                  data: (list) => list,
                  orElse: () => const <ProfileSkill>[],
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _openEditor(
    BuildContext context,
    WidgetRef ref,
    List<ProfileSkill> currentSkills,
  ) async {
    final saved = await showSkillsEditorBottomSheet(
      context: context,
      orgType: orgType,
      expertiseKeys: expertiseKeys,
      initialSkills: currentSkills,
    );
    if (!saved) return;
    onSaved?.call();
  }
}

// ---------------------------------------------------------------------------
// Header
// ---------------------------------------------------------------------------

class _Header extends StatelessWidget {
  const _Header({required this.max, required this.count});

  final int max;
  final int count;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Row(
      children: [
        Icon(
          Icons.stars_outlined,
          size: 20,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            l10n.skillsSectionTitle,
            style: theme.textTheme.titleMedium,
          ),
        ),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
          decoration: BoxDecoration(
            color: theme.colorScheme.primary.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(12),
          ),
          child: Text(
            l10n.skillsCounter(count, max),
            style: TextStyle(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w600,
              fontSize: 12,
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Data state — read-only rose chips
// ---------------------------------------------------------------------------

class _DataState extends StatelessWidget {
  const _DataState({required this.skills});

  final List<ProfileSkill> skills;

  @override
  Widget build(BuildContext context) {
    if (skills.isEmpty) return const _EmptyState();
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        for (final s in skills) _ReadOnlyPill(label: s.displayText),
      ],
    );
  }
}

class _ReadOnlyPill extends StatelessWidget {
  const _ReadOnlyPill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    return Container(
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
    );
  }
}

// ---------------------------------------------------------------------------
// Empty / loading / error states
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 14),
      decoration: BoxDecoration(
        color: appColors?.muted,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Row(
        children: [
          Icon(Icons.info_outline, size: 18, color: appColors?.mutedForeground),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              l10n.skillsEmpty,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
                height: 1.4,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _LoadingState extends StatelessWidget {
  const _LoadingState();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 12),
      child: Center(child: CircularProgressIndicator.adaptive()),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Text(
        l10n.skillsErrorGeneric,
        style:
            theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.error),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Edit button
// ---------------------------------------------------------------------------

class _EditButton extends StatelessWidget {
  const _EditButton({required this.label, required this.onTap});

  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: double.infinity,
      child: OutlinedButton.icon(
        onPressed: onTap,
        icon: const Icon(Icons.edit_outlined, size: 18),
        label: Text(label),
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(double.infinity, 48),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
        ),
      ),
    );
  }
}
