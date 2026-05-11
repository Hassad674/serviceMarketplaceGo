import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/billing_profile.dart';

/// Compact read-only summary of the saved [BillingProfile], rendered
/// inside the proposal payment screen when the client's profile is
/// already complete. Mirrors the web `BillingProfileSummary` card:
/// pays + adresse + entité légale + identifiants fiscaux, with a
/// single "Modifier" CTA that flips the embed back into edit mode.
///
/// Soleil v2 styling: ivoire surface card (rounded-2xl), sable border,
/// Geist Mono used only for the SIRET / VAT identifiers.
class BillingProfileSummary extends StatelessWidget {
  const BillingProfileSummary({
    super.key,
    required this.profile,
    required this.onEdit,
  });

  final BillingProfile profile;
  final VoidCallback onEdit;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    final borderColor = appColors?.border ?? theme.dividerColor;

    final showTax = profile.profileType == ProfileType.business ||
        profile.taxId.isNotEmpty ||
        profile.vatNumber.isNotEmpty;

    return Semantics(
      label: l10n.billingEmbedSummaryTitle,
      container: true,
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: colorScheme.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(AppTheme.radius2xl),
          border: Border.all(color: borderColor),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _Header(onEdit: onEdit, l10n: l10n),
            const SizedBox(height: 12),
            _SummaryRow(
              label: l10n.billingEmbedEntity,
              value: _entityLine(profile),
            ),
            const SizedBox(height: 8),
            _SummaryRow(
              label: l10n.billingEmbedCountry,
              value: profile.country.isEmpty ? '—' : profile.country,
            ),
            const SizedBox(height: 8),
            _SummaryRow(
              label: l10n.billingEmbedAddress,
              value: _addressLine(profile),
            ),
            if (showTax) ...[
              const SizedBox(height: 8),
              _SummaryRow(
                label: l10n.billingEmbedTax,
                value: _taxLine(profile),
                monospace: true,
              ),
            ],
          ],
        ),
      ),
    );
  }

  static String _entityLine(BillingProfile p) {
    if (p.legalName.isEmpty) return '—';
    if (p.tradingName.isEmpty) return p.legalName;
    return '${p.legalName} · ${p.tradingName}';
  }

  static String _addressLine(BillingProfile p) {
    final parts = <String>[
      p.addressLine1,
      p.addressLine2,
      [p.postalCode, p.city].where((s) => s.isNotEmpty).join(' '),
    ].where((s) => s.isNotEmpty).toList();
    return parts.isEmpty ? '—' : parts.join(', ');
  }

  static String _taxLine(BillingProfile p) {
    if (p.taxId.isEmpty && p.vatNumber.isEmpty) return '—';
    if (p.vatNumber.isEmpty) return p.taxId;
    if (p.taxId.isEmpty) return p.vatNumber;
    return '${p.taxId} · ${p.vatNumber}';
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.onEdit, required this.l10n});

  final VoidCallback onEdit;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    return Row(
      children: [
        Icon(Icons.receipt_long_outlined, size: 16, color: primary),
        const SizedBox(width: 6),
        Expanded(
          child: Text(
            l10n.billingEmbedSummaryTitle.toUpperCase(),
            style: SoleilTextStyles.mono.copyWith(
              color: primary,
              fontSize: 11,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.4,
            ),
          ),
        ),
        TextButton.icon(
          onPressed: onEdit,
          icon: const Icon(Icons.edit_outlined, size: 14),
          label: Text(l10n.billingEmbedEditCta),
          style: TextButton.styleFrom(
            foregroundColor: theme.colorScheme.onSurfaceVariant,
            textStyle: SoleilTextStyles.body.copyWith(fontSize: 13),
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            minimumSize: const Size(0, 32),
          ),
        ),
      ],
    );
  }
}

class _SummaryRow extends StatelessWidget {
  const _SummaryRow({
    required this.label,
    required this.value,
    this.monospace = false,
  });

  final String label;
  final String value;
  final bool monospace;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label.toUpperCase(),
          style: SoleilTextStyles.mono.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
            fontSize: 10,
            fontWeight: FontWeight.w600,
            letterSpacing: 1.2,
          ),
        ),
        const SizedBox(height: 2),
        Text(
          value,
          style: monospace
              ? SoleilTextStyles.mono.copyWith(
                  color: theme.colorScheme.onSurface,
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                )
              : SoleilTextStyles.body.copyWith(
                  color: theme.colorScheme.onSurface,
                  fontSize: 13.5,
                ),
        ),
      ],
    );
  }
}
