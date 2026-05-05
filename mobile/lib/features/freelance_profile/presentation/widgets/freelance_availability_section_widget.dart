import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../providers/freelance_profile_providers.dart';

/// Soleil v2 W-16 v3 (BATCH-PROFIL-FIX item #4) — adds an editable
/// availability card on the freelance own-profile screen. The widget
/// renders a single Soleil card showing the three availability slots
/// as radio chips, plus a corail save button at the bottom. Tapping
/// `Save` calls the existing [freelanceAvailabilityEditorProvider] —
/// no new mutation hook is introduced.
///
/// The wire values match the backend `availability_status` enum:
/// `available_now` / `available_soon` / `not_available`. The brief
/// originally referenced `available_for_hire` + `available_from` —
/// neither field exists on the [FreelanceProfile] entity, so the
/// widget binds to the existing tri-state enum and the additional
/// flags are flagged as out-of-scope in the batch report.
class FreelanceAvailabilitySectionWidget extends ConsumerStatefulWidget {
  const FreelanceAvailabilitySectionWidget({
    super.key,
    required this.initialWireValue,
    required this.canEdit,
    required this.onSaved,
  });

  final String initialWireValue;
  final bool canEdit;
  final VoidCallback onSaved;

  @override
  ConsumerState<FreelanceAvailabilitySectionWidget> createState() =>
      _FreelanceAvailabilitySectionWidgetState();
}

class _FreelanceAvailabilitySectionWidgetState
    extends ConsumerState<FreelanceAvailabilitySectionWidget> {
  late String _draft;

  @override
  void initState() {
    super.initState();
    _draft = widget.initialWireValue;
  }

  @override
  void didUpdateWidget(
    covariant FreelanceAvailabilitySectionWidget oldWidget,
  ) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initialWireValue != widget.initialWireValue) {
      _draft = widget.initialWireValue;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: theme.colorScheme.outline),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.event_available_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  l10n.tier1AvailabilitySectionTitle,
                  style: theme.textTheme.titleMedium,
                ),
              ),
              if (widget.canEdit)
                IconButton(
                  tooltip: l10n.tier1AvailabilityEditButton,
                  icon: const Icon(Icons.edit_outlined, size: 18),
                  onPressed: _openEditor,
                ),
            ],
          ),
          const SizedBox(height: 12),
          _buildBadge(theme, l10n, widget.initialWireValue),
        ],
      ),
    );
  }

  Widget _buildBadge(ThemeData theme, AppLocalizations l10n, String wire) {
    final color = _badgeColor(theme, wire);
    final label = _statusLabel(l10n, wire);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        border: Border.all(color: color.withValues(alpha: 0.3)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
          const SizedBox(width: 6),
          Flexible(
            child: Text(
              label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: SoleilTextStyles.caption.copyWith(
                color: color,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Color _badgeColor(ThemeData theme, String wire) {
    final extension = theme.extension<AppColors>();
    switch (wire) {
      case 'available_soon':
        return extension?.warning ?? theme.colorScheme.error;
      case 'not_available':
        return extension?.subtleForeground ?? theme.colorScheme.onSurfaceVariant;
      case 'available_now':
      default:
        return extension?.success ?? theme.colorScheme.primary;
    }
  }

  String _statusLabel(AppLocalizations l10n, String wire) {
    switch (wire) {
      case 'available_soon':
        return l10n.tier1AvailabilityStatusAvailableSoon;
      case 'not_available':
        return l10n.tier1AvailabilityStatusNotAvailable;
      case 'available_now':
      default:
        return l10n.tier1AvailabilityStatusAvailableNow;
    }
  }

  Future<void> _openEditor() async {
    final saved = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surfaceContainerLowest,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (_) => _FreelanceAvailabilitySheet(
        initialWireValue: _draft,
      ),
    );
    if (saved == true) {
      widget.onSaved();
    }
  }
}

/// Bottom sheet body — three radio rows + Soleil corail save button.
/// Sealed inside this file because no other surface needs the same
/// freelance-scoped editor (the agency variant uses the tier1 widget).
class _FreelanceAvailabilitySheet extends ConsumerStatefulWidget {
  const _FreelanceAvailabilitySheet({required this.initialWireValue});

  final String initialWireValue;

  @override
  ConsumerState<_FreelanceAvailabilitySheet> createState() =>
      _FreelanceAvailabilitySheetState();
}

class _FreelanceAvailabilitySheetState
    extends ConsumerState<_FreelanceAvailabilitySheet> {
  late String _draft;

  static const List<String> _wireValues = <String>[
    'available_now',
    'available_soon',
    'not_available',
  ];

  @override
  void initState() {
    super.initState();
    _draft = widget.initialWireValue;
  }

  bool get _hasChanges => _draft != widget.initialWireValue;

  Future<void> _save() async {
    final notifier =
        ref.read(freelanceAvailabilityEditorProvider.notifier);
    final ok = await notifier.save(_draft);
    if (!mounted) return;
    if (ok) {
      Navigator.of(context).pop(true);
      return;
    }
    final l10n = AppLocalizations.of(context)!;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(l10n.tier1ErrorGeneric),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final editorState = ref.watch(freelanceAvailabilityEditorProvider);
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return SafeArea(
      top: false,
      child: Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 16, 12, 12),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      l10n.tier1AvailabilitySectionTitle,
                      style: theme.textTheme.titleLarge,
                    ),
                  ),
                  IconButton(
                    tooltip:
                        MaterialLocalizations.of(context).closeButtonTooltip,
                    icon: const Icon(Icons.close),
                    onPressed: () => Navigator.of(context).pop(false),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 16, 20, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  for (final wire in _wireValues)
                    RadioListTile<String>(
                      value: wire,
                      groupValue: _draft,
                      onChanged: (next) {
                        if (next != null) {
                          setState(() => _draft = next);
                        }
                      },
                      title: Text(_statusLabel(l10n, wire)),
                      contentPadding: EdgeInsets.zero,
                      activeColor: theme.colorScheme.primary,
                    ),
                ],
              ),
            ),
            Container(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerLowest,
                border: Border(
                  top: BorderSide(color: theme.colorScheme.outline),
                ),
              ),
              child: SizedBox(
                width: double.infinity,
                child: ElevatedButton(
                  onPressed:
                      (!_hasChanges || editorState.isSaving) ? null : _save,
                  child: Text(
                    editorState.isSaving ? l10n.tier1Saving : l10n.tier1Save,
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _statusLabel(AppLocalizations l10n, String wire) {
    switch (wire) {
      case 'available_soon':
        return l10n.tier1AvailabilityStatusAvailableSoon;
      case 'not_available':
        return l10n.tier1AvailabilityStatusNotAvailable;
      case 'available_now':
      default:
        return l10n.tier1AvailabilityStatusAvailableNow;
    }
  }
}
