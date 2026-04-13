import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/expertise_catalog.dart';
import '../providers/expertise_provider.dart';
import '../utils/expertise_labels.dart';
import 'expertise_picker_bottom_sheet.dart';

/// Editable "Areas of expertise" section rendered on the operator's
/// own profile screen.
///
/// The section is a controlled widget: it does NOT own the current
/// list of domains. The parent (profile screen) reads them from the
/// profile provider, passes them in as [initialDomains], and
/// invalidates the profile provider after a successful save so the
/// whole card refreshes with server truth.
///
/// Optimism lives locally: we track a `_pendingDomains` buffer so
/// the pills update the instant the bottom sheet closes, even
/// before the PUT round-trip is done. On save failure we roll back
/// to the original list and surface a SnackBar.
class ExpertiseSectionWidget extends ConsumerStatefulWidget {
  const ExpertiseSectionWidget({
    super.key,
    required this.orgType,
    required this.initialDomains,
    required this.canEdit,
    required this.onSaved,
  });

  /// The operator's current org type. Controls the `maxSelection`
  /// and the feature-flag check (enterprise hides the whole card).
  final String? orgType;

  /// Domains currently stored on the profile. The widget keeps a
  /// local copy so edits feel instant.
  final List<String> initialDomains;

  /// Whether the current operator has permission to edit the org
  /// profile. When `false`, the card still renders (if there are
  /// domains to show) but the "Add domains" button and pill remove
  /// chips are hidden.
  final bool canEdit;

  /// Invoked after a successful save so the parent can invalidate
  /// the profile provider and trigger a refetch.
  final VoidCallback onSaved;

  @override
  ConsumerState<ExpertiseSectionWidget> createState() =>
      _ExpertiseSectionWidgetState();
}

class _ExpertiseSectionWidgetState
    extends ConsumerState<ExpertiseSectionWidget> {
  late List<String> _pending;

  @override
  void initState() {
    super.initState();
    _pending = _sanitize(widget.initialDomains);
  }

  @override
  void didUpdateWidget(covariant ExpertiseSectionWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    // Parent re-fetched the profile — sync the local buffer to the
    // new server truth.
    if (!_sameList(oldWidget.initialDomains, widget.initialDomains)) {
      _pending = _sanitize(widget.initialDomains);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (!ExpertiseCatalog.isFeatureEnabledForOrgType(widget.orgType)) {
      return const SizedBox.shrink();
    }

    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final max = ExpertiseCatalog.maxForOrgType(widget.orgType);
    final hasDomains = _pending.isNotEmpty;

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
          _buildHeader(theme, appColors, l10n, max),
          const SizedBox(height: 12),
          if (hasDomains)
            _SelectedPillsWrap(
              domains: _pending,
              canRemove: widget.canEdit,
              onRemove: _removeDomain,
            )
          else
            _EmptyState(max: max),
          if (widget.canEdit) ...[
            const SizedBox(height: 12),
            _AddDomainsButton(onTap: _openPicker),
          ],
        ],
      ),
    );
  }

  // --------------------------------------------------------------------------
  // Header — title + counter
  // --------------------------------------------------------------------------

  Widget _buildHeader(
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
    int max,
  ) {
    return Row(
      children: [
        Icon(
          Icons.auto_awesome_outlined,
          size: 20,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            l10n.expertiseSectionTitle,
            style: theme.textTheme.titleMedium,
          ),
        ),
        Container(
          padding: const EdgeInsets.symmetric(
            horizontal: 10,
            vertical: 4,
          ),
          decoration: BoxDecoration(
            color: theme.colorScheme.primary.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(12),
          ),
          child: Text(
            l10n.expertiseCounter(_pending.length, max),
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

  // --------------------------------------------------------------------------
  // Interaction handlers
  // --------------------------------------------------------------------------

  Future<void> _openPicker() async {
    final max = ExpertiseCatalog.maxForOrgType(widget.orgType);
    if (max == 0) return;

    final selection = await showExpertisePickerBottomSheet(
      context: context,
      initialDomains: _pending,
      maxSelection: max,
    );
    if (selection == null) return;
    if (_sameList(selection, _pending)) return;

    await _persist(selection);
  }

  Future<void> _removeDomain(String key) async {
    final next = [..._pending]..remove(key);
    await _persist(next);
  }

  Future<void> _persist(List<String> next) async {
    final previous = List<String>.unmodifiable(_pending);
    setState(() => _pending = List<String>.unmodifiable(next));

    final saved =
        await ref.read(expertiseEditorProvider.notifier).save(next);

    if (!mounted) return;
    if (saved == null) {
      setState(() => _pending = previous);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(AppLocalizations.of(context)!.expertiseErrorGeneric),
          behavior: SnackBarBehavior.floating,
        ),
      );
      return;
    }

    setState(() => _pending = _sanitize(saved));
    widget.onSaved();
  }

  // --------------------------------------------------------------------------
  // Pure helpers
  // --------------------------------------------------------------------------

  List<String> _sanitize(List<String> domains) {
    final seen = <String>{};
    final ordered = <String>[];
    for (final key in ExpertiseCatalog.allKeys) {
      if (domains.contains(key) && seen.add(key)) {
        ordered.add(key);
      }
    }
    return List<String>.unmodifiable(ordered);
  }

  bool _sameList(List<String> a, List<String> b) {
    if (a.length != b.length) return false;
    for (var i = 0; i < a.length; i++) {
      if (a[i] != b[i]) return false;
    }
    return true;
  }
}

// ---------------------------------------------------------------------------
// Private "Add domains" CTA
// ---------------------------------------------------------------------------

class _AddDomainsButton extends StatelessWidget {
  const _AddDomainsButton({required this.onTap});

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return SizedBox(
      width: double.infinity,
      child: OutlinedButton.icon(
        onPressed: onTap,
        icon: const Icon(Icons.add, size: 18),
        label: Text(l10n.expertiseAddDomains),
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

// ---------------------------------------------------------------------------
// Selected pills — removable chips
// ---------------------------------------------------------------------------

class _SelectedPillsWrap extends StatelessWidget {
  const _SelectedPillsWrap({
    required this.domains,
    required this.canRemove,
    required this.onRemove,
  });

  final List<String> domains;
  final bool canRemove;
  final ValueChanged<String> onRemove;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        for (final key in domains)
          _SelectedPill(
            label: localizedExpertiseLabel(context, key),
            onRemove: canRemove ? () => onRemove(key) : null,
          ),
      ],
    );
  }
}

class _SelectedPill extends StatelessWidget {
  const _SelectedPill({
    required this.label,
    this.onRemove,
  });

  final String label;
  final VoidCallback? onRemove;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;

    return Semantics(
      label: label,
      button: onRemove != null,
      child: Container(
        padding: EdgeInsets.only(
          left: 12,
          right: onRemove != null ? 4 : 12,
          top: 6,
          bottom: 6,
        ),
        decoration: BoxDecoration(
          color: primary.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: primary.withValues(alpha: 0.2)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              label,
              style: TextStyle(
                color: primary,
                fontWeight: FontWeight.w600,
                fontSize: 13,
              ),
            ),
            if (onRemove != null) ...[
              const SizedBox(width: 4),
              InkWell(
                onTap: onRemove,
                customBorder: const CircleBorder(),
                child: Padding(
                  padding: const EdgeInsets.all(4),
                  child: Icon(
                    Icons.close,
                    size: 14,
                    color: primary,
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Empty state — placeholder copy
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.max});

  final int max;

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
          Icon(
            Icons.info_outline,
            size: 18,
            color: appColors?.mutedForeground,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              '${l10n.expertiseEmptyPrivate} ${l10n.expertiseSectionSubtitle(max)}',
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
