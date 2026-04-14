import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/availability_status.dart';
import '../providers/profile_tier1_providers.dart';

/// Which availability slot this card owns. The direct variant is
/// rendered on the freelance profile screen; the referrer variant
/// is rendered on the referrer profile screen. Each variant saves
/// only its own slot — the other column is untouched, on both the
/// optimistic cache and the backend.
enum AvailabilityVariant { direct, referrer }

/// Compact availability card rendered on the profile edit screen.
///
/// Shows a single colored badge for the slot owned by [variant] and
/// opens a bottom-sheet radio group for editing. The referrer
/// variant self-hides unless the operator is a `provider_personal`
/// with `referrer_enabled == true` — that way the same widget is
/// safe to drop into any screen that may not gate it upstream.
class AvailabilitySectionWidget extends ConsumerStatefulWidget {
  const AvailabilitySectionWidget({
    super.key,
    required this.variant,
    required this.initialDirect,
    required this.initialReferrer,
    required this.referrerEnabled,
    required this.canEdit,
    required this.onSaved,
  });

  final AvailabilityVariant variant;
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

  bool get _isDirect => widget.variant == AvailabilityVariant.direct;

  @override
  Widget build(BuildContext context) {
    if (!_isDirect && !widget.referrerEnabled) {
      return const SizedBox.shrink();
    }
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
          _buildBadge(),
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

  Widget _buildBadge() {
    if (_isDirect) {
      return AvailabilityBadge(status: _direct);
    }
    final referrer = _referrer ?? AvailabilityStatus.availableNow;
    return AvailabilityBadge(status: referrer);
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
        variant: widget.variant,
        initialDirect: _direct,
        initialReferrer: _referrer,
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
    required this.variant,
    required this.initialDirect,
    required this.initialReferrer,
  });

  final AvailabilityVariant variant;
  final AvailabilityStatus initialDirect;
  final AvailabilityStatus? initialReferrer;

  @override
  ConsumerState<_AvailabilityEditorSheet> createState() =>
      _AvailabilityEditorSheetState();
}

class _AvailabilityEditorSheetState
    extends ConsumerState<_AvailabilityEditorSheet> {
  late AvailabilityStatus _direct;
  late AvailabilityStatus _referrer;

  @override
  void initState() {
    super.initState();
    _direct = widget.initialDirect;
    _referrer = widget.initialReferrer ?? AvailabilityStatus.availableNow;
  }

  bool get _isDirect => widget.variant == AvailabilityVariant.direct;

  bool get _hasChanges => _isDirect
      ? _direct != widget.initialDirect
      : _referrer != (widget.initialReferrer ?? AvailabilityStatus.availableNow);

  Future<void> _save() async {
    final notifier = ref.read(availabilityEditorProvider.notifier);
    final ok = await notifier.save(
      direct: _isDirect ? _direct : null,
      referrer: _isDirect ? null : _referrer,
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
    final label = _isDirect
        ? l10n.tier1AvailabilityDirectLabel
        : l10n.tier1AvailabilityReferrerTitle;

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
                      Text(label, style: theme.textTheme.titleSmall),
                      const SizedBox(height: 8),
                      _AvailabilityRadioGroup(
                        value: _isDirect ? _direct : _referrer,
                        onChanged: (s) => setState(() {
                          if (_isDirect) {
                            _direct = s;
                          } else {
                            _referrer = s;
                          }
                        }),
                      ),
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
