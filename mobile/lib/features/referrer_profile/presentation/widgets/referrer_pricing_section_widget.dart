import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/profile/money_format.dart';
import '../../domain/entities/referrer_pricing.dart';
import '../providers/referrer_profile_providers.dart';

/// Pricing card rendered on the referrer edit screen.
///
/// V1 pricing simplification: the referrer persona is narrowed to
/// `commission_pct` only. The editor asks for ONE percentage in
/// [0..100] and persists a range-shaped row with min == max (the
/// backend validator requires both bounds). Legacy commission_flat
/// rows still render correctly on public cards via _formatReferrer;
/// only the editor is constrained.
class ReferrerPricingSectionWidget extends ConsumerWidget {
  const ReferrerPricingSectionWidget({super.key, required this.canEdit});

  final bool canEdit;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final pricingAsync = ref.watch(referrerPricingProvider);

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
                Icons.handshake_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Text(
                l10n.tier1PricingReferralSectionTitle,
                style: theme.textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 12),
          pricingAsync.when(
            loading: () => const Padding(
              padding: EdgeInsets.all(12),
              child: LinearProgressIndicator(),
            ),
            error: (_, __) => Text(l10n.tier1ErrorGeneric),
            data: (pricing) => _PricingBody(
              pricing: pricing,
              emptyLabel: l10n.tier1PricingEmpty,
              negotiableBadge: l10n.tier1PricingNegotiableBadge,
            ),
          ),
          if (canEdit) ...[
            const SizedBox(height: 12),
            OutlinedButton.icon(
              onPressed: () => _openEditor(context, ref),
              icon: const Icon(Icons.edit_outlined, size: 18),
              label: Text(l10n.tier1PricingEditButton),
              style: OutlinedButton.styleFrom(
                minimumSize: const Size(double.infinity, 48),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _openEditor(BuildContext context, WidgetRef ref) async {
    final current = ref.read(referrerPricingProvider).valueOrNull;
    final result = await showModalBottomSheet<ReferrerPricing?>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _ReferrerPricingEditor(initial: current),
    );
    if (result == null) return;
    if (!context.mounted) return;

    final errorLabel = AppLocalizations.of(context)!.tier1ErrorGeneric;
    final messenger = ScaffoldMessenger.of(context);
    final notifier = ref.read(referrerPricingEditorProvider.notifier);
    final ok = await notifier.upsert(result);
    if (!ok) {
      messenger.showSnackBar(SnackBar(content: Text(errorLabel)));
      return;
    }
    ref.invalidate(referrerPricingProvider);
    ref.invalidate(referrerProfileProvider);
  }
}

class _PricingBody extends StatelessWidget {
  const _PricingBody({
    required this.pricing,
    required this.emptyLabel,
    required this.negotiableBadge,
  });

  final ReferrerPricing? pricing;
  final String emptyLabel;
  final String negotiableBadge;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final locale = Localizations.localeOf(context).languageCode;
    final row = pricing;
    if (row == null) {
      return Text(
        emptyLabel,
        style: theme.textTheme.bodyMedium?.copyWith(
          color: theme.colorScheme.onSurfaceVariant,
          fontStyle: FontStyle.italic,
        ),
      );
    }
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Expanded(
          child: Text(
            _formatReferrer(row, locale),
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.w700,
            ),
          ),
        ),
        if (row.negotiable)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            decoration: BoxDecoration(
              color: theme.colorScheme.primary.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Text(
              negotiableBadge,
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w700,
                color: theme.colorScheme.primary,
              ),
            ),
          ),
      ],
    );
  }

  String _formatReferrer(ReferrerPricing p, String locale) {
    final isFrench = locale.startsWith('fr');
    switch (p.type) {
      case ReferrerPricingType.commissionPct:
        final min = formatBasisPoints(p.minAmount, isFrench: isFrench);
        // V1 headline shape: collapse "N – N %" into a single clean
        // "N % de commission" when min equals max (the shape the
        // editor now produces). Legacy multi-bound rows still render
        // as a range.
        if (p.maxAmount != null && p.maxAmount != p.minAmount) {
          final max = formatBasisPoints(p.maxAmount!, isFrench: isFrench);
          return '$min – $max';
        }
        return isFrench ? '$min de commission' : '$min commission';
      case ReferrerPricingType.commissionFlat:
        final amount = formatMoney(p.minAmount, p.currency, locale);
        return isFrench ? '$amount / deal' : '$amount per deal';
    }
  }
}

// ---------------------------------------------------------------------------
// Editor sheet
// ---------------------------------------------------------------------------

class _ReferrerPricingEditor extends StatefulWidget {
  const _ReferrerPricingEditor({required this.initial});

  final ReferrerPricing? initial;

  @override
  State<_ReferrerPricingEditor> createState() =>
      _ReferrerPricingEditorState();
}

class _ReferrerPricingEditorState extends State<_ReferrerPricingEditor> {
  // V1: the referrer editor collapses to a single commission percent
  // input. Type is locked to commission_pct, currency is locked to
  // "pct" (backend convention). Legacy commission_flat rows are
  // coerced on re-edit — the flat amount is discarded and the user
  // types a fresh percentage.
  late TextEditingController _pctController;
  late TextEditingController _noteController;
  late bool _negotiable;

  static const _v1Type = ReferrerPricingType.commissionPct;
  static const _v1Currency = 'pct';
  static const _pctMax = 100;

  @override
  void initState() {
    super.initState();
    final init = widget.initial;
    _pctController = TextEditingController(text: _initialPctText(init));
    _noteController = TextEditingController(text: init?.note ?? '');
    _negotiable = init?.negotiable ?? false;
  }

  @override
  void dispose() {
    _pctController.dispose();
    _noteController.dispose();
    super.dispose();
  }

  // _initialPctText chooses the best single-percent seed to display.
  // We prefer the max_amount of a commission_pct row (the user's
  // headline rate); legacy commission_flat rows have no percent, so
  // we fall back to empty and let the user type fresh.
  String _initialPctText(ReferrerPricing? init) {
    if (init == null || init.type != ReferrerPricingType.commissionPct) {
      return '';
    }
    final basisPoints = init.maxAmount ?? init.minAmount;
    final value = basisPoints / 100.0;
    if (value == value.roundToDouble()) return value.toInt().toString();
    return value.toStringAsFixed(2);
  }

  // _parsePctToBasisPoints clamps [0..100] and converts to the backend
  // basis-point representation (5.5 % -> 550).
  int _parsePctToBasisPoints(String raw) {
    final cleaned = raw.replaceAll(',', '.').trim();
    if (cleaned.isEmpty) return 0;
    final value = (double.tryParse(cleaned) ?? 0.0).clamp(0.0, _pctMax.toDouble());
    return (value * 100).round();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.tier1PricingReferralModalTitle,
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 16),
            // V1: single commission percentage field. Type is locked
            // to commission_pct and currency is locked to "pct" — no
            // dropdown, no max-amount bound.
            TextField(
              controller: _pctController,
              keyboardType: const TextInputType.numberWithOptions(
                decimal: true,
              ),
              decoration: InputDecoration(
                labelText: l10n.tier1PricingReferrerCommissionLabel,
                helperText: l10n.tier1PricingReferrerCommissionHint,
                suffixText: '%',
                border: const OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _noteController,
              maxLength: 160,
              decoration: InputDecoration(
                labelText: l10n.tier1PricingNoteLabel,
                hintText: l10n.tier1PricingNotePlaceholder,
                border: const OutlineInputBorder(),
              ),
            ),
            SwitchListTile.adaptive(
              contentPadding: EdgeInsets.zero,
              title: Text(l10n.tier1PricingNegotiableLabel),
              value: _negotiable,
              onChanged: (value) => setState(() => _negotiable = value),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => Navigator.of(context).pop(),
                    child: Text(l10n.tier1Cancel),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: ElevatedButton(
                    onPressed: _submit,
                    child: Text(l10n.tier1Save),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  void _submit() {
    // V1: echo the single percentage to BOTH min and max so the
    // backend validator (range required) accepts the payload, and
    // the formatter collapses it to a single "N % commission" on
    // the card.
    final basisPoints = _parsePctToBasisPoints(_pctController.text);
    final pricing = ReferrerPricing(
      type: _v1Type,
      minAmount: basisPoints,
      maxAmount: basisPoints,
      currency: _v1Currency,
      note: _noteController.text.trim(),
      negotiable: _negotiable,
    );
    Navigator.of(context).pop(pricing);
  }
}
