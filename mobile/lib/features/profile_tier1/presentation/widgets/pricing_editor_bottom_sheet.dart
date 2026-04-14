import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/pricing.dart';
import '../../domain/entities/pricing_kind.dart';
import '../providers/pricing_provider.dart';
import '../utils/pricing_format.dart';
import 'pricing_row_card.dart';

/// Opens the pricing editor as a modal bottom sheet. The sheet
/// handles both kinds at once — a direct row and (when referrers
/// are enabled) a referral row. Each kind has its own form; the
/// save button upserts the enabled rows and deletes any the user
/// turned off.
///
/// Resolves to `true` when the sheet closes after at least one
/// successful save operation — the caller uses it to refresh the
/// parent card.
Future<bool?> showPricingEditorBottomSheet({
  required BuildContext context,
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
      orgType: orgType,
      referrerEnabled: referrerEnabled,
      initialPricings: initialPricings,
    ),
  );
}

class PricingEditorBottomSheet extends ConsumerStatefulWidget {
  const PricingEditorBottomSheet({
    super.key,
    required this.orgType,
    required this.referrerEnabled,
    required this.initialPricings,
  });

  final String? orgType;
  final bool referrerEnabled;
  final List<Pricing> initialPricings;

  @override
  ConsumerState<PricingEditorBottomSheet> createState() =>
      _PricingEditorBottomSheetState();
}

class _PricingEditorBottomSheetState
    extends ConsumerState<PricingEditorBottomSheet> {
  late PricingDraft _direct;
  late PricingDraft _referral;

  @override
  void initState() {
    super.initState();
    _direct = _findOrEmpty(PricingKind.direct);
    _referral = _findOrEmpty(PricingKind.referral);
  }

  PricingDraft _findOrEmpty(PricingKind kind) {
    for (final p in widget.initialPricings) {
      if (p.kind == kind) return PricingDraft.fromPricing(p);
    }
    return PricingDraft.empty(kind: kind);
  }

  bool get _showReferralRow => widget.referrerEnabled;

  List<PricingType> get _directTypes => const <PricingType>[
        PricingType.daily,
        PricingType.hourly,
        PricingType.projectFrom,
        PricingType.projectRange,
      ];

  List<PricingType> get _referralTypes => const <PricingType>[
        PricingType.commissionPct,
        PricingType.commissionFlat,
      ];

  Future<void> _save() async {
    final notifier = ref.read(pricingProvider.notifier);
    final l10n = AppLocalizations.of(context)!;

    final hadDirect =
        widget.initialPricings.any((p) => p.kind == PricingKind.direct);
    final hadReferral =
        widget.initialPricings.any((p) => p.kind == PricingKind.referral);

    var anyChange = false;

    // -- Direct row --
    if (_direct.enabled) {
      final pricing = _direct.toPricing();
      if (pricing == null) {
        _showError(l10n.tier1ErrorPricingInvalidAmount);
        return;
      }
      final ok = await notifier.upsert(pricing);
      if (!ok) return _showError(l10n.tier1ErrorGeneric);
      anyChange = true;
    } else if (hadDirect) {
      final ok = await notifier.remove(PricingKind.direct);
      if (!ok) return _showError(l10n.tier1ErrorGeneric);
      anyChange = true;
    }

    // -- Referral row (only when gated on) --
    if (_showReferralRow) {
      if (_referral.enabled) {
        final pricing = _referral.toPricing();
        if (pricing == null) {
          _showError(l10n.tier1ErrorPricingInvalidAmount);
          return;
        }
        final ok = await notifier.upsert(pricing);
        if (!ok) return _showError(l10n.tier1ErrorGeneric);
        anyChange = true;
      } else if (hadReferral) {
        final ok = await notifier.remove(PricingKind.referral);
        if (!ok) return _showError(l10n.tier1ErrorGeneric);
        anyChange = true;
      }
    }

    if (!mounted) return;
    Navigator.of(context).pop(anyChange);
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
                _SheetHeader(title: l10n.tier1PricingModalTitle),
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
    return <Widget>[
      _PreviewStrip(
        heading: l10n.tier1PricingPreviewHeading,
        direct: _directPreview(context),
        referral: _showReferralRow ? _referralPreview(context) : null,
      ),
      const SizedBox(height: 20),
      PricingRowCard(
        title: l10n.tier1PricingKindDirect,
        draft: _direct,
        allowedTypes: _directTypes,
        onChanged: () => setState(() {}),
      ),
      if (_showReferralRow) ...[
        const SizedBox(height: 16),
        PricingRowCard(
          title: l10n.tier1PricingKindReferral,
          draft: _referral,
          allowedTypes: _referralTypes,
          onChanged: () => setState(() {}),
        ),
      ],
    ];
  }

  String _directPreview(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    if (!_direct.enabled) return l10n.tier1PricingEmptyPreview;
    final pricing = _direct.toPricing();
    if (pricing == null) return l10n.tier1PricingEmptyPreview;
    final locale = Localizations.localeOf(context).languageCode;
    return formatPricing(pricing, locale: locale);
  }

  String _referralPreview(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    if (!_referral.enabled) return l10n.tier1PricingEmptyPreview;
    final pricing = _referral.toPricing();
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
    required this.direct,
    required this.referral,
  });

  final String heading;
  final String direct;
  final String? referral;

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
            direct,
            style: theme.textTheme.titleMedium?.copyWith(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w700,
            ),
          ),
          if (referral != null) ...[
            const SizedBox(height: 4),
            Text(
              referral!,
              style: theme.textTheme.titleSmall?.copyWith(
                color: theme.colorScheme.primary,
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
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
