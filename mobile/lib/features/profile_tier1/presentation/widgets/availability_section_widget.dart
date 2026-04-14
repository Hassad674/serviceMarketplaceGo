import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/availability_status.dart';
import '../providers/profile_tier1_providers.dart';

/// Compact availability card rendered on the profile edit screen.
///
/// Shows two colored badges — one for the direct service line and,
/// when the operator has `referrer_enabled == true`, a second one
/// for the business-referrer line. Tapping the edit button opens
/// a modal bottom sheet with radio groups for each track.
///
/// The widget is gated at the parent level: enterprise orgs never
/// render it (no availability concept for client-side orgs).
class AvailabilitySectionWidget extends ConsumerStatefulWidget {
  const AvailabilitySectionWidget({
    super.key,
    required this.initialDirect,
    required this.initialReferrer,
    required this.referrerEnabled,
    required this.canEdit,
    required this.onSaved,
  });

  final AvailabilityStatus initialDirect;
  final AvailabilityStatus? initialReferrer;
  final bool referrerEnabled;
  final bool canEdit;
  final VoidCallback onSaved;

  @override
  ConsumerState<AvailabilitySectionWidget> createState() =>
      _AvailabilitySectionWidgetState();
}

class _AvailabilitySectionWidgetState
    extends ConsumerState<AvailabilitySectionWidget> {
  late AvailabilityStatus _direct;
  late AvailabilityStatus? _referrer;

  @override
  void initState() {
    super.initState();
    _direct = widget.initialDirect;
    _referrer = widget.initialReferrer;
  }

  @override
  void didUpdateWidget(covariant AvailabilitySectionWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initialDirect != widget.initialDirect) {
      _direct = widget.initialDirect;
    }
    if (oldWidget.initialReferrer != widget.initialReferrer) {
      _referrer = widget.initialReferrer;
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
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
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
              Text(
                l10n.tier1AvailabilitySectionTitle,
                style: theme.textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 12),
          _buildBadgeRow(context, l10n),
          if (widget.canEdit) ...[
            const SizedBox(height: 12),
            _EditButton(
              label: l10n.tier1AvailabilityEditButton,
              onTap: _openEditor,
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildBadgeRow(BuildContext context, AppLocalizations l10n) {
    final showReferrer = widget.referrerEnabled && _referrer != null;
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        AvailabilityBadge(
          status: _direct,
          prefix: showReferrer ? l10n.tier1AvailabilityDirectLabel : null,
        ),
        if (showReferrer)
          AvailabilityBadge(
            status: _referrer!,
            prefix: l10n.tier1AvailabilityReferrerLabel,
          ),
      ],
    );
  }

  Future<void> _openEditor() async {
    final saved = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (_) => _AvailabilityEditorSheet(
        initialDirect: _direct,
        initialReferrer: _referrer,
        referrerEnabled: widget.referrerEnabled,
      ),
    );
    if (saved == true) {
      widget.onSaved();
    }
  }
}

// ---------------------------------------------------------------------------
// Colored availability badge — exported for reuse by the identity strip
// ---------------------------------------------------------------------------

class AvailabilityBadge extends StatelessWidget {
  const AvailabilityBadge({
    super.key,
    required this.status,
    this.prefix,
  });

  final AvailabilityStatus status;
  final String? prefix;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final color = availabilityColor(status);
    final label = availabilityLabel(status, l10n);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: color.withValues(alpha: 0.3)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 6),
          Flexible(
            child: Text(
              prefix == null ? label : '$prefix · $label',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: color,
                fontWeight: FontWeight.w600,
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

Color availabilityColor(AvailabilityStatus status) {
  switch (status) {
    case AvailabilityStatus.availableNow:
      return const Color(0xFF22C55E); // green-500
    case AvailabilityStatus.availableSoon:
      return const Color(0xFFF59E0B); // amber-500
    case AvailabilityStatus.notAvailable:
      return const Color(0xFFEF4444); // red-500
  }
}

String availabilityLabel(AvailabilityStatus status, AppLocalizations l10n) {
  switch (status) {
    case AvailabilityStatus.availableNow:
      return l10n.tier1AvailabilityStatusAvailableNow;
    case AvailabilityStatus.availableSoon:
      return l10n.tier1AvailabilityStatusAvailableSoon;
    case AvailabilityStatus.notAvailable:
      return l10n.tier1AvailabilityStatusNotAvailable;
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

// ---------------------------------------------------------------------------
// Bottom-sheet editor — two radio groups
// ---------------------------------------------------------------------------

class _AvailabilityEditorSheet extends ConsumerStatefulWidget {
  const _AvailabilityEditorSheet({
    required this.initialDirect,
    required this.initialReferrer,
    required this.referrerEnabled,
  });

  final AvailabilityStatus initialDirect;
  final AvailabilityStatus? initialReferrer;
  final bool referrerEnabled;

  @override
  ConsumerState<_AvailabilityEditorSheet> createState() =>
      _AvailabilityEditorSheetState();
}

class _AvailabilityEditorSheetState
    extends ConsumerState<_AvailabilityEditorSheet> {
  late AvailabilityStatus _direct;
  late AvailabilityStatus? _referrer;

  @override
  void initState() {
    super.initState();
    _direct = widget.initialDirect;
    _referrer = widget.initialReferrer ??
        (widget.referrerEnabled ? AvailabilityStatus.availableNow : null);
  }

  bool get _hasChanges =>
      _direct != widget.initialDirect || _referrer != widget.initialReferrer;

  Future<void> _save() async {
    final notifier = ref.read(availabilityEditorProvider.notifier);
    final ok = await notifier.save(
      _direct,
      widget.referrerEnabled ? _referrer : null,
    );
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
    final editorState = ref.watch(availabilityEditorProvider);
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return SafeArea(
      top: false,
      child: Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.7,
          minChildSize: 0.4,
          maxChildSize: 0.92,
          builder: (_, scrollController) {
            return Column(
              children: [
                _SheetHeader(title: l10n.tier1AvailabilitySectionTitle),
                const Divider(height: 1),
                Expanded(
                  child: ListView(
                    controller: scrollController,
                    padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
                    children: [
                      Text(
                        l10n.tier1AvailabilityDirectLabel,
                        style: theme.textTheme.titleSmall,
                      ),
                      const SizedBox(height: 8),
                      _AvailabilityRadioGroup(
                        value: _direct,
                        onChanged: (s) => setState(() => _direct = s),
                      ),
                      if (widget.referrerEnabled) ...[
                        const SizedBox(height: 24),
                        Text(
                          l10n.tier1AvailabilityReferrerTitle,
                          style: theme.textTheme.titleSmall,
                        ),
                        const SizedBox(height: 8),
                        _AvailabilityRadioGroup(
                          value: _referrer ?? AvailabilityStatus.availableNow,
                          onChanged: (s) => setState(() => _referrer = s),
                        ),
                      ],
                    ],
                  ),
                ),
                _SaveBar(
                  hasChanges: _hasChanges,
                  isSaving: editorState.isSaving,
                  onSave: _save,
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}

class _AvailabilityRadioGroup extends StatelessWidget {
  const _AvailabilityRadioGroup({
    required this.value,
    required this.onChanged,
  });

  final AvailabilityStatus value;
  final ValueChanged<AvailabilityStatus> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Column(
      children: [
        for (final status in AvailabilityStatus.values)
          RadioListTile<AvailabilityStatus>(
            value: status,
            groupValue: value,
            onChanged: (v) {
              if (v != null) onChanged(v);
            },
            title: Text(availabilityLabel(status, l10n)),
            secondary: Container(
              width: 12,
              height: 12,
              decoration: BoxDecoration(
                color: availabilityColor(status),
                shape: BoxShape.circle,
              ),
            ),
            contentPadding: EdgeInsets.zero,
          ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Shared sheet chrome (header + save bar)
// ---------------------------------------------------------------------------

class _SheetHeader extends StatelessWidget {
  const _SheetHeader({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 12, 12),
      child: Row(
        children: [
          Expanded(
            child: Text(title, style: theme.textTheme.titleLarge),
          ),
          IconButton(
            tooltip: MaterialLocalizations.of(context).closeButtonTooltip,
            icon: const Icon(Icons.close),
            onPressed: () => Navigator.of(context).pop(false),
          ),
        ],
      ),
    );
  }
}

class _SaveBar extends StatelessWidget {
  const _SaveBar({
    required this.hasChanges,
    required this.isSaving,
    required this.onSave,
  });

  final bool hasChanges;
  final bool isSaving;
  final VoidCallback onSave;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          top: BorderSide(color: appColors?.border ?? theme.dividerColor),
        ),
      ),
      child: SizedBox(
        width: double.infinity,
        child: ElevatedButton(
          onPressed: (!hasChanges || isSaving) ? null : onSave,
          style: ElevatedButton.styleFrom(
            minimumSize: const Size(double.infinity, 48),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
          ),
          child: Text(isSaving ? l10n.tier1Saving : l10n.tier1Save),
        ),
      ),
    );
  }
}
