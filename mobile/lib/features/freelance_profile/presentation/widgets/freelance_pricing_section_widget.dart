import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/profile/money_format.dart';
import '../../domain/entities/freelance_pricing.dart';
import '../providers/freelance_profile_providers.dart';

/// Pricing card rendered on the freelance edit screen. Surfaces the
/// current row (or an empty hint) and opens a bottom sheet editor.
///
/// V1 pricing simplification: the freelance persona is narrowed to
/// `daily` (TJM) in EUR only. The editor is a single amount field
/// — no type dropdown, no currency picker. Legacy rows (hourly /
/// project_*) still render correctly on the public card via
/// _formatPricing below; only the editor is constrained.
class FreelancePricingSectionWidget extends ConsumerWidget {
  const FreelancePricingSectionWidget({
    super.key,
    required this.canEdit,
  });

  final bool canEdit;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final pricingAsync = ref.watch(freelancePricingProvider);

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
                Icons.paid_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Text(
                l10n.tier1PricingDirectSectionTitle,
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
    final current = ref.read(freelancePricingProvider).valueOrNull;
    final result = await showModalBottomSheet<FreelancePricing?>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _FreelancePricingEditor(initial: current),
    );
    if (result == null) return;
    if (!context.mounted) return;

    final errorLabel = AppLocalizations.of(context)!.tier1ErrorGeneric;
    final notifier = ref.read(freelancePricingEditorProvider.notifier);
    final messenger = ScaffoldMessenger.of(context);
    final ok = await notifier.upsert(result);
    if (!ok) {
      messenger.showSnackBar(SnackBar(content: Text(errorLabel)));
      return;
    }
    ref.invalidate(freelancePricingProvider);
    ref.invalidate(freelanceProfileProvider);
  }
}

class _PricingBody extends StatelessWidget {
  const _PricingBody({
    required this.pricing,
    required this.emptyLabel,
    required this.negotiableBadge,
  });

  final FreelancePricing? pricing;
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
            _formatPricing(row, locale),
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

  String _formatPricing(FreelancePricing p, String locale) {
    final isFrench = locale.startsWith('fr');
    switch (p.type) {
      case FreelancePricingType.daily:
        final amount = formatMoney(p.minAmount, p.currency, locale);
        return isFrench ? '$amount / j' : '$amount / day';
      case FreelancePricingType.hourly:
        final amount = formatMoney(p.minAmount, p.currency, locale);
        return isFrench ? '$amount / h' : '$amount / hr';
      case FreelancePricingType.projectFrom:
        final amount = formatMoney(p.minAmount, p.currency, locale);
        return isFrench ? 'À partir de $amount' : 'From $amount';
      case FreelancePricingType.projectRange:
        final min = formatMoney(p.minAmount, p.currency, locale);
        final max = p.maxAmount != null
            ? formatMoney(p.maxAmount!, p.currency, locale)
            : null;
        return max != null ? '$min – $max' : min;
    }
  }
}

// ---------------------------------------------------------------------------
// Editor sheet
// ---------------------------------------------------------------------------

class _FreelancePricingEditor extends StatefulWidget {
  const _FreelancePricingEditor({required this.initial});

  final FreelancePricing? initial;

  @override
  State<_FreelancePricingEditor> createState() =>
      _FreelancePricingEditorState();
}

class _FreelancePricingEditorState extends State<_FreelancePricingEditor> {
  // V1 narrows the editor to a single TJM field. Type and currency
  // are no longer user-editable — always daily + EUR.
  late TextEditingController _amountController;
  late TextEditingController _noteController;
  late bool _negotiable;

  static const _v1Type = FreelancePricingType.daily;
  static const _v1Currency = 'EUR';

  @override
  void initState() {
    super.initState();
    final init = widget.initial;
    _amountController = TextEditingController(
      text: init != null ? _centimesToText(init.minAmount) : '',
    );
    _noteController = TextEditingController(text: init?.note ?? '');
    _negotiable = init?.negotiable ?? false;
  }

  @override
  void dispose() {
    _amountController.dispose();
    _noteController.dispose();
    super.dispose();
  }

  String _centimesToText(int centimes) {
    final value = centimes / 100.0;
    if (value == value.roundToDouble()) {
      return value.toInt().toString();
    }
    return value.toStringAsFixed(2);
  }

  int _parseToCentimes(String raw) {
    final cleaned = raw.replaceAll(',', '.').trim();
    if (cleaned.isEmpty) return 0;
    final value = double.tryParse(cleaned) ?? 0.0;
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
              l10n.tier1PricingDirectModalTitle,
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 16),
            // V1: single TJM amount field (€/j). The type dropdown
            // and currency picker are removed — we always persist
            // daily + EUR.
            TextField(
              controller: _amountController,
              keyboardType: const TextInputType.numberWithOptions(
                decimal: true,
              ),
              decoration: InputDecoration(
                labelText: l10n.tier1PricingFreelanceDailyLabel,
                helperText: l10n.tier1PricingFreelanceDailyHint,
                suffixText: '€/j',
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
    // V1: always daily + EUR, no max amount. Currency / type are
    // not user-editable — the backend whitelist would reject any
    // other combination anyway.
    final amount = _parseToCentimes(_amountController.text);
    final pricing = FreelancePricing(
      type: _v1Type,
      minAmount: amount,
      maxAmount: null,
      currency: _v1Currency,
      note: _noteController.text.trim(),
      negotiable: _negotiable,
    );
    Navigator.of(context).pop(pricing);
  }
}
