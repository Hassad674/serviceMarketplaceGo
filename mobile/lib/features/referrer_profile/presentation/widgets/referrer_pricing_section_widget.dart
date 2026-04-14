import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/profile/money_format.dart';
import '../../domain/entities/referrer_pricing.dart';
import '../providers/referrer_profile_providers.dart';

/// Pricing card rendered on the referrer edit screen. The editor
/// only offers the two referrer-legal types (commission_pct,
/// commission_flat) — the backend rejects anything else.
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
        if (p.maxAmount != null) {
          final max = formatBasisPoints(p.maxAmount!, isFrench: isFrench);
          return '$min – $max';
        }
        return min;
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
  late ReferrerPricingType _type;
  late TextEditingController _minController;
  late TextEditingController _maxController;
  late TextEditingController _noteController;
  late String _currency;
  late bool _negotiable;

  static const List<String> _currencies = ['EUR', 'USD', 'GBP', 'CAD', 'AUD'];

  @override
  void initState() {
    super.initState();
    final init = widget.initial;
    _type = init?.type ?? ReferrerPricingType.commissionPct;
    _minController = TextEditingController(
      text: init != null ? _amountToText(init.minAmount, init.type) : '',
    );
    _maxController = TextEditingController(
      text: init?.maxAmount != null
          ? _amountToText(init!.maxAmount!, init.type)
          : '',
    );
    _noteController = TextEditingController(text: init?.note ?? '');
    _currency =
        init?.currency ?? (_type.isMonetary ? 'EUR' : 'pct');
    _negotiable = init?.negotiable ?? false;
  }

  @override
  void dispose() {
    _minController.dispose();
    _maxController.dispose();
    _noteController.dispose();
    super.dispose();
  }

  String _amountToText(int rawAmount, ReferrerPricingType type) {
    // Commission percent: basis points -> percent float.
    // Commission flat: centimes -> currency float.
    final value = rawAmount / 100.0;
    if (value == value.roundToDouble()) return value.toInt().toString();
    return value.toStringAsFixed(2);
  }

  int _parseAmount(String raw) {
    final cleaned = raw.replaceAll(',', '.').trim();
    if (cleaned.isEmpty) return 0;
    final value = double.tryParse(cleaned) ?? 0.0;
    return (value * 100).round();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final isPct = _type == ReferrerPricingType.commissionPct;

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
            DropdownButtonFormField<ReferrerPricingType>(
              initialValue: _type,
              decoration: InputDecoration(
                labelText: l10n.tier1PricingKindReferral,
                border: const OutlineInputBorder(),
              ),
              items: [
                DropdownMenuItem(
                  value: ReferrerPricingType.commissionPct,
                  child: Text(l10n.tier1PricingTypeCommissionPct),
                ),
                DropdownMenuItem(
                  value: ReferrerPricingType.commissionFlat,
                  child: Text(l10n.tier1PricingTypeCommissionFlat),
                ),
              ],
              onChanged: (value) {
                if (value == null) return;
                setState(() {
                  _type = value;
                  _currency = value.isMonetary ? 'EUR' : 'pct';
                });
              },
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _minController,
              keyboardType: TextInputType.number,
              decoration: InputDecoration(
                labelText: l10n.tier1PricingMinLabel,
                border: const OutlineInputBorder(),
              ),
            ),
            if (isPct) ...[
              const SizedBox(height: 12),
              TextField(
                controller: _maxController,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(
                  labelText: l10n.tier1PricingMaxLabel,
                  border: const OutlineInputBorder(),
                ),
              ),
            ],
            if (!isPct) ...[
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: _currency,
                decoration: InputDecoration(
                  labelText: l10n.tier1PricingCurrencyLabel,
                  border: const OutlineInputBorder(),
                ),
                items: [
                  for (final c in _currencies)
                    DropdownMenuItem(value: c, child: Text(c)),
                ],
                onChanged: (value) {
                  if (value != null) setState(() => _currency = value);
                },
              ),
            ],
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
    final minAmount = _parseAmount(_minController.text);
    final maxAmount = _type == ReferrerPricingType.commissionPct &&
            _maxController.text.isNotEmpty
        ? _parseAmount(_maxController.text)
        : null;
    final pricing = ReferrerPricing(
      type: _type,
      minAmount: minAmount,
      maxAmount: maxAmount,
      currency: _currency,
      note: _noteController.text.trim(),
      negotiable: _negotiable,
    );
    Navigator.of(context).pop(pricing);
  }
}
