import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/pricing.dart';
import '../../domain/entities/pricing_kind.dart';
import '../providers/pricing_provider.dart';
import '../utils/pricing_format.dart';
import 'pricing_editor_bottom_sheet.dart';

/// Inline pricing card rendered on the profile / referral screens.
///
/// The section is gated at the parent level — enterprise orgs never
/// render this card. Each instance of the widget is bound to a
/// single [variant] (direct on the profile screen, referral on the
/// referral screen) and the editor it opens only edits that one
/// kind.
class PricingSectionWidget extends ConsumerWidget {
  const PricingSectionWidget({
    super.key,
    required this.variant,
    required this.orgType,
    required this.referrerEnabled,
    required this.canEdit,
    required this.onSaved,
  });

  final PricingKind variant;
  final String? orgType;
  final bool referrerEnabled;
  final bool canEdit;
  final VoidCallback onSaved;

  bool get _isVisible {
    if (orgType == 'enterprise') return false;
    if (variant == PricingKind.direct) {
      return orgType == 'agency' || orgType == 'provider_personal';
    }
    return orgType == 'provider_personal' && referrerEnabled;
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (!_isVisible) return const SizedBox.shrink();

    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final state = ref.watch(pricingProvider);

    final title = variant == PricingKind.direct
        ? l10n.tier1PricingDirectSectionTitle
        : l10n.tier1PricingReferralSectionTitle;

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
              Text(title, style: theme.textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 12),
          state.pricings.when(
            loading: () => const _LoadingState(),
            error: (_, __) => _ErrorState(text: l10n.tier1ErrorGeneric),
            data: (list) => _VariantSummary(
              variant: variant,
              pricings: list,
            ),
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
      variant: variant,
      orgType: orgType,
      referrerEnabled: referrerEnabled,
      initialPricings: current,
    );
    if (saved == true) onSaved();
  }
}

class _VariantSummary extends StatelessWidget {
  const _VariantSummary({required this.variant, required this.pricings});

  final PricingKind variant;
  final List<Pricing> pricings;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;

    Pricing? row;
    for (final p in pricings) {
      if (p.kind == variant) {
        row = p;
        break;
      }
    }

    if (row == null) {
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
        Text(
          formatPricing(row, locale: locale),
          style: theme.textTheme.titleSmall?.copyWith(
            color: appColors?.mutedForeground,
            fontFeatures: const [FontFeature.tabularFigures()],
          ),
        ),
        if (row.negotiable) ...[
          const SizedBox(height: 6),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: theme.colorScheme.primary.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Text(
              l10n.tier1PricingNegotiableBadge,
              style: TextStyle(
                color: theme.colorScheme.primary,
                fontWeight: FontWeight.w600,
                fontSize: 11,
              ),
            ),
          ),
        ],
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
