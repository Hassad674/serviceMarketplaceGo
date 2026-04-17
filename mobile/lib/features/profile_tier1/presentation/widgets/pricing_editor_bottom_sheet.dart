import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/pricing.dart';
import '../../domain/entities/pricing_kind.dart';
import '../providers/pricing_provider.dart';
import '../utils/pricing_format.dart';
import 'pricing_row_card.dart';

/// Opens the pricing editor as a modal bottom sheet for ONE kind.
/// The split across two screens (profile for direct, referral for
/// referral) means a single sheet instance never mixes both.
///
/// Resolves to `true` when the sheet closes after a successful
/// save or delete — the caller uses it to refresh the parent card.
Future<bool?> showPricingEditorBottomSheet({
  required BuildContext context,
  required PricingKind variant,
  required String? orgType,
  required bool referrerEnabled,
  required List<Pricing> initialPricings,
}) {
  return showModalBottomSheet<bool>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => PricingEditorBottomSheet(
      variant: variant,
      orgType: orgType,
      referrerEnabled: referrerEnabled,
      initialPricings: initialPricings,
    ),
  );
}

class PricingEditorBottomSheet extends ConsumerStatefulWidget {
  const PricingEditorBottomSheet({
    super.key,
    required this.variant,
    required this.orgType,
    required this.referrerEnabled,
    required this.initialPricings,
  });

  final PricingKind variant;
  final String? orgType;
  final bool referrerEnabled;
  final List<Pricing> initialPricings;

  @override
  ConsumerState<PricingEditorBottomSheet> createState() =>
      _PricingEditorBottomSheetState();
}

class _PricingEditorBottomSheetState
    extends ConsumerState<PricingEditorBottomSheet> {
  late PricingDraft _draft;
  late final List<PricingType> _allowedTypes;

  @override
  void initState() {
    super.initState();
    _draft = _findOrEmpty(widget.variant);
    _allowedTypes = _computeAllowedTypes();
    // If the persisted type is no longer allowed (e.g. an agency
    // loading a legacy daily row), fall back to the first allowed
    // type so the form never renders an invalid state.
    if (_draft.enabled && !_allowedTypes.contains(_draft.type)) {
      _draft.type = _allowedTypes.first;
    }
  }

  PricingDraft _findOrEmpty(PricingKind kind) {
    for (final p in widget.initialPricings) {
      if (p.kind == kind) return PricingDraft.fromPricing(p);
    }
    return PricingDraft.empty(kind: kind);
  }

  // _computeAllowedTypes mirrors the backend V1 whitelist: every
  // (kind, org) triplet narrows to the single allowed type.
  //   - referral -> commission_pct only
  //   - agency direct -> project_from only
  //   - provider direct -> project_from only (provider_personal
  //     direct pricing actually goes through freelancepricing /
  //     daily; this screen is used by agencies + legacy callers)
  //
  // The PricingType enum keeps every value so legacy rows
  // deserialise — but the editor only lets users create the one V1
  // type per persona. When the persisted type is no longer allowed,
  // initState() falls back to _allowedTypes.first.
  List<PricingType> _computeAllowedTypes() {
    if (widget.variant == PricingKind.referral) {
      return const <PricingType>[PricingType.commissionPct];
    }
    return const <PricingType>[PricingType.projectFrom];
  }

  Future<void> _save() async {
    final notifier = ref.read(pricingProvider.notifier);
    final l10n = AppLocalizations.of(context)!;

    final hadRow = widget.initialPricings.any((p) => p.kind == widget.variant);

    if (_draft.enabled) {
      final pricing = _draft.toPricing();
      if (pricing == null) {
        _showError(l10n.tier1ErrorPricingInvalidAmount);
        return;
      }
      final ok = await notifier.upsert(pricing);
      if (!ok) {
        _showError(l10n.tier1ErrorGeneric);
        return;
      }
    } else if (hadRow) {
      final ok = await notifier.remove(widget.variant);
      if (!ok) {
        _showError(l10n.tier1ErrorGeneric);
        return;
      }
    }

    if (!mounted) return;
    Navigator.of(context).pop(true);
  }

  void _showError(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final state = ref.watch(pricingProvider);

    final title = widget.variant == PricingKind.direct
        ? l10n.tier1PricingDirectModalTitle
        : l10n.tier1PricingReferralModalTitle;

    return SafeArea(
      top: false,
      child: Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.9,
          minChildSize: 0.5,
          maxChildSize: 0.97,
          builder: (_, scrollController) {
            return Column(
              children: [
                _SheetHeader(title: title),
                const Divider(height: 1),
                Expanded(
                  child: ListView(
                    controller: scrollController,
                    padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
                    children: _buildBody(context, l10n),
                  ),
                ),
                _SaveBar(
                  isSaving: state.isSaving,
                  onSave: _save,
                ),
              ],
            );
          },
        ),
      ),
    );
  }

  List<Widget> _buildBody(BuildContext context, AppLocalizations l10n) {
    final cardTitle = widget.variant == PricingKind.direct
        ? l10n.tier1PricingKindDirect
        : l10n.tier1PricingKindReferral;
    return <Widget>[
      _PreviewStrip(
        heading: l10n.tier1PricingPreviewHeading,
        value: _draftPreview(context),
      ),
      const SizedBox(height: 20),
      PricingRowCard(
        title: cardTitle,
        draft: _draft,
        allowedTypes: _allowedTypes,
        onChanged: () => setState(() {}),
      ),
    ];
  }

  String _draftPreview(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    if (!_draft.enabled) return l10n.tier1PricingEmptyPreview;
    final pricing = _draft.toPricing();
    if (pricing == null) return l10n.tier1PricingEmptyPreview;
    final locale = Localizations.localeOf(context).languageCode;
    return formatPricing(pricing, locale: locale);
  }
}

// ---------------------------------------------------------------------------
// Preview strip — live card-like preview of the current draft
// ---------------------------------------------------------------------------

class _PreviewStrip extends StatelessWidget {
  const _PreviewStrip({
    required this.heading,
    required this.value,
  });

  final String heading;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.primary.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: theme.colorScheme.primary.withValues(alpha: 0.2),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            heading,
            style: theme.textTheme.labelMedium?.copyWith(
              color: appColors?.mutedForeground,
              letterSpacing: 0.5,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            value,
            style: theme.textTheme.titleMedium?.copyWith(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Sheet chrome
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
    required this.isSaving,
    required this.onSave,
  });

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
          onPressed: isSaving ? null : onSave,
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
