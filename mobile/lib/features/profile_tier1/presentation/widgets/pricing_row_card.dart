import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/pricing.dart';
import '../../domain/entities/pricing_kind.dart';

/// Currencies surfaced by the mobile UI. The backend accepts more,
/// but 5 covers every market this app targets.
const List<String> kPricingCurrencies = <String>[
  'EUR',
  'USD',
  'GBP',
  'CAD',
  'AUD',
];

/// Editable representation of one pricing row. Values are kept in
/// "display units" (500 = 500€, 5.5 = 5.5%) — the save routine
/// converts to centimes / basis points at the boundary.
class PricingDraft {
  PricingDraft({
    required this.kind,
    required this.type,
    required this.min,
    required this.max,
    required this.currency,
    required this.note,
    required this.enabled,
  });

  PricingKind kind;
  PricingType type;
  String min;
  String max;
  String currency;
  String note;

  /// When `false`, save will issue a DELETE for this kind.
  bool enabled;

  factory PricingDraft.empty({required PricingKind kind}) {
    return PricingDraft(
      kind: kind,
      type: kind == PricingKind.direct
          ? PricingType.daily
          : PricingType.commissionPct,
      min: '',
      max: '',
      currency: kind == PricingKind.direct ? 'EUR' : 'pct',
      note: '',
      enabled: false,
    );
  }

  factory PricingDraft.fromPricing(Pricing p) {
    return PricingDraft(
      kind: p.kind,
      type: p.type,
      min: _formatInput(p.minAmount, p.type),
      max: p.maxAmount == null ? '' : _formatInput(p.maxAmount!, p.type),
      currency: p.currency,
      note: p.note,
      enabled: true,
    );
  }

  /// Build a [Pricing] from the current form values, converting
  /// display units to storage units. Returns `null` when the
  /// required `min` field cannot be parsed.
  Pricing? toPricing() {
    final minParsed = _parseInput(min);
    if (minParsed == null) return null;
    int? maxParsed;
    if (type.supportsMax && max.trim().isNotEmpty) {
      maxParsed = _parseInput(max);
      if (maxParsed == null) return null;
    }
    return Pricing(
      kind: kind,
      type: type,
      minAmount: minParsed,
      maxAmount: maxParsed,
      currency: type == PricingType.commissionPct ? 'pct' : currency,
      note: note.trim(),
    );
  }

  static String _formatInput(int raw, PricingType type) {
    final v = raw / 100.0;
    if (v == v.roundToDouble()) return v.toInt().toString();
    return v
        .toStringAsFixed(2)
        .replaceFirst(RegExp(r'0+$'), '')
        .replaceFirst(RegExp(r'\.$'), '');
  }

  static int? _parseInput(String text) {
    final trimmed = text.trim().replaceAll(',', '.');
    if (trimmed.isEmpty) return null;
    final value = double.tryParse(trimmed);
    if (value == null || value < 0) return null;
    return (value * 100).round();
  }
}

/// Per-kind form card. Contains a title row + enable switch, and
/// — when the switch is on — a type picker, the amount fields, a
/// currency dropdown (except for commissionPct), and a note.
class PricingRowCard extends StatefulWidget {
  const PricingRowCard({
    super.key,
    required this.title,
    required this.draft,
    required this.allowedTypes,
    required this.onChanged,
  });

  final String title;
  final PricingDraft draft;
  final List<PricingType> allowedTypes;
  final VoidCallback onChanged;

  @override
  State<PricingRowCard> createState() => _PricingRowCardState();
}

class _PricingRowCardState extends State<PricingRowCard> {
  late final TextEditingController _min;
  late final TextEditingController _max;
  late final TextEditingController _note;

  @override
  void initState() {
    super.initState();
    _min = TextEditingController(text: widget.draft.min);
    _max = TextEditingController(text: widget.draft.max);
    _note = TextEditingController(text: widget.draft.note);

    _min.addListener(() {
      widget.draft.min = _min.text;
      widget.onChanged();
    });
    _max.addListener(() {
      widget.draft.max = _max.text;
      widget.onChanged();
    });
    _note.addListener(() {
      widget.draft.note = _note.text;
      widget.onChanged();
    });
  }

  @override
  void dispose() {
    _min.dispose();
    _max.dispose();
    _note.dispose();
    super.dispose();
  }

  void _setEnabled(bool enabled) {
    setState(() {
      widget.draft.enabled = enabled;
      if (enabled && !widget.allowedTypes.contains(widget.draft.type)) {
        widget.draft.type = widget.allowedTypes.first;
      }
    });
    widget.onChanged();
  }

  void _setType(PricingType type) {
    setState(() => widget.draft.type = type);
    widget.onChanged();
  }

  void _setCurrency(String currency) {
    setState(() => widget.draft.currency = currency);
    widget.onChanged();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final draft = widget.draft;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: appColors?.muted,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: appColors?.border ?? theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(widget.title, style: theme.textTheme.titleSmall),
              ),
              Switch.adaptive(
                value: draft.enabled,
                onChanged: _setEnabled,
              ),
            ],
          ),
          if (draft.enabled) ...[
            const SizedBox(height: 12),
            _TypeRadioGroup(
              selected: draft.type,
              allowed: widget.allowedTypes,
              onChanged: _setType,
            ),
            const SizedBox(height: 12),
            _AmountFields(
              type: draft.type,
              minController: _min,
              maxController: _max,
              minLabel: l10n.tier1PricingMinLabel,
              maxLabel: l10n.tier1PricingMaxLabel,
            ),
            if (draft.type != PricingType.commissionPct) ...[
              const SizedBox(height: 12),
              _CurrencyDropdown(
                value: draft.currency,
                onChanged: _setCurrency,
                label: l10n.tier1PricingCurrencyLabel,
              ),
            ],
            const SizedBox(height: 12),
            TextField(
              controller: _note,
              maxLength: 160,
              maxLines: 2,
              decoration: InputDecoration(
                labelText: l10n.tier1PricingNoteLabel,
                hintText: l10n.tier1PricingNotePlaceholder,
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _TypeRadioGroup extends StatelessWidget {
  const _TypeRadioGroup({
    required this.selected,
    required this.allowed,
    required this.onChanged,
  });

  final PricingType selected;
  final List<PricingType> allowed;
  final ValueChanged<PricingType> onChanged;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        for (final type in allowed)
          RadioListTile<PricingType>(
            value: type,
            groupValue: selected,
            onChanged: (v) {
              if (v != null) onChanged(v);
            },
            title: Text(_labelFor(context, type)),
            contentPadding: EdgeInsets.zero,
            dense: true,
          ),
      ],
    );
  }

  String _labelFor(BuildContext context, PricingType type) {
    final l10n = AppLocalizations.of(context)!;
    switch (type) {
      case PricingType.daily:
        return l10n.tier1PricingTypeDaily;
      case PricingType.hourly:
        return l10n.tier1PricingTypeHourly;
      case PricingType.projectFrom:
        return l10n.tier1PricingTypeProjectFrom;
      case PricingType.projectRange:
        return l10n.tier1PricingTypeProjectRange;
      case PricingType.commissionPct:
        return l10n.tier1PricingTypeCommissionPct;
      case PricingType.commissionFlat:
        return l10n.tier1PricingTypeCommissionFlat;
    }
  }
}

class _AmountFields extends StatelessWidget {
  const _AmountFields({
    required this.type,
    required this.minController,
    required this.maxController,
    required this.minLabel,
    required this.maxLabel,
  });

  final PricingType type;
  final TextEditingController minController;
  final TextEditingController maxController;
  final String minLabel;
  final String maxLabel;

  @override
  Widget build(BuildContext context) {
    final minField = _numberField(minLabel, minController);
    if (!type.supportsMax) return minField;
    return Row(
      children: [
        Expanded(child: minField),
        const SizedBox(width: 12),
        Expanded(child: _numberField(maxLabel, maxController)),
      ],
    );
  }

  Widget _numberField(String label, TextEditingController controller) {
    return TextField(
      controller: controller,
      keyboardType: const TextInputType.numberWithOptions(decimal: true),
      inputFormatters: [
        FilteringTextInputFormatter.allow(RegExp(r'[0-9.,]')),
      ],
      decoration: InputDecoration(
        labelText: label,
        border: const OutlineInputBorder(),
      ),
    );
  }
}

class _CurrencyDropdown extends StatelessWidget {
  const _CurrencyDropdown({
    required this.value,
    required this.onChanged,
    required this.label,
  });

  final String value;
  final ValueChanged<String> onChanged;
  final String label;

  @override
  Widget build(BuildContext context) {
    return InputDecorator(
      decoration: InputDecoration(
        labelText: label,
        border: const OutlineInputBorder(),
      ),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<String>(
          value: kPricingCurrencies.contains(value) ? value : 'EUR',
          isDense: true,
          isExpanded: true,
          onChanged: (v) {
            if (v != null) onChanged(v);
          },
          items: [
            for (final currency in kPricingCurrencies)
              DropdownMenuItem<String>(
                value: currency,
                child: Text(currency),
              ),
          ],
        ),
      ),
    );
  }
}
