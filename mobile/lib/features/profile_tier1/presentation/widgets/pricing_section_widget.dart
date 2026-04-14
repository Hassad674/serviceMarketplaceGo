import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/pricing.dart';
import '../../domain/entities/pricing_kind.dart';
import '../providers/pricing_provider.dart';
import '../utils/pricing_format.dart';
import 'pricing_editor_bottom_sheet.dart';

/// Inline pricing card rendered on the profile edit screen.
///
/// The section is gated at the parent level — enterprise orgs
/// never render this card. For provider / agency orgs, the card
/// shows the 0..2 current pricing rows as one-line summaries and
/// an "Update pricing" button that opens [PricingEditorBottomSheet].
class PricingSectionWidget extends ConsumerWidget {
  const PricingSectionWidget({
    super.key,
    required this.orgType,
    required this.referrerEnabled,
    required this.canEdit,
    required this.onSaved,
  });

  final String? orgType;
  final bool referrerEnabled;
  final bool canEdit;
  final VoidCallback onSaved;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final state = ref.watch(pricingProvider);

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
                Icons.euro_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Text(
                l10n.tier1PricingSectionTitle,
                style: theme.textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 12),
          state.pricings.when(
            loading: () => const _LoadingState(),
            error: (_, __) => _ErrorState(text: l10n.tier1ErrorGeneric),
            data: (list) => _PricingSummary(pricings: list),
          ),
          if (canEdit) ...[
            const SizedBox(height: 12),
            _EditButton(
              label: l10n.tier1PricingEditButton,
              onTap: () => _openEditor(
                context,
                ref,
                state.pricings.maybeWhen(
                  data: (list) => list,
                  orElse: () => const <Pricing>[],
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _openEditor(
    BuildContext context,
    WidgetRef ref,
    List<Pricing> current,
  ) async {
    final saved = await showPricingEditorBottomSheet(
      context: context,
      orgType: orgType,
      referrerEnabled: referrerEnabled,
      initialPricings: current,
    );
    if (saved == true) onSaved();
  }
}

// ---------------------------------------------------------------------------
// Read-only summary
// ---------------------------------------------------------------------------

class _PricingSummary extends StatelessWidget {
  const _PricingSummary({required this.pricings});

  final List<Pricing> pricings;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;

    if (pricings.isEmpty) {
      return Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 14),
        decoration: BoxDecoration(
          color: appColors?.muted,
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          border: Border.all(color: appColors?.border ?? theme.dividerColor),
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
                l10n.tier1PricingEmpty,
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

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final p in pricings) ...[
          _PricingRow(
            kindLabel: p.kind == PricingKind.direct
                ? l10n.tier1PricingKindDirect
                : l10n.tier1PricingKindReferral,
            value: formatPricing(p, locale: locale),
          ),
          const SizedBox(height: 6),
        ],
      ],
    );
  }
}

class _PricingRow extends StatelessWidget {
  const _PricingRow({required this.kindLabel, required this.value});

  final String kindLabel;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Row(
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
          decoration: BoxDecoration(
            color: theme.colorScheme.primary.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Text(
            kindLabel,
            style: TextStyle(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w600,
              fontSize: 11,
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            value,
            style: theme.textTheme.titleSmall?.copyWith(
              color: appColors?.mutedForeground,
              fontFeatures: const [FontFeature.tabularFigures()],
            ),
          ),
        ),
      ],
    );
  }
}

class _LoadingState extends StatelessWidget {
  const _LoadingState();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 12),
      child: Center(child: CircularProgressIndicator.adaptive()),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Text(
        text,
        style:
            theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.error),
      ),
    );
  }
}

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
