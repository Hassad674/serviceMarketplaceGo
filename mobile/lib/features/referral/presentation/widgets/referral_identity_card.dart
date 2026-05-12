import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/referral_entity.dart';
import 'anonymized_client_card.dart';
import 'anonymized_provider_card.dart';

/// ReferralIdentityCard — apporteur owner sees a minimalist confirmation
/// card (just the display name + role label), other viewers keep the
/// masked snapshot. Decision is computed by the caller and forwarded
/// via [isOwner] so this widget stays pure for testing.
///
/// The apporteur view intentionally drops the legacy "Voir le profil"
/// CTA + chevron + masked-card eyebrow: the apporteur already knows
/// who they introduced; the card is a confirmation, not a discovery
/// surface.
class ReferralIdentityCard extends StatelessWidget {
  const ReferralIdentityCard({
    super.key,
    required this.referral,
    required this.isOwner,
    this.providerName,
    this.clientName,
  });

  final Referral referral;
  final bool isOwner;
  final String? providerName;
  final String? clientName;

  @override
  Widget build(BuildContext context) {
    if (!isOwner) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AnonymizedProviderCard(snapshot: referral.introSnapshot.provider),
          const SizedBox(height: 12),
          AnonymizedClientCard(snapshot: referral.introSnapshot.client),
        ],
      );
    }
    final l10n = AppLocalizations.of(context)!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _RevealedIdentityTile(
          key: const ValueKey('referral-identity-provider'),
          icon: Icons.work_outline,
          roleLabel: l10n.referralIdentityProviderTitle,
          name: providerName ?? referral.providerDisplayName,
        ),
        const SizedBox(height: 12),
        _RevealedIdentityTile(
          key: const ValueKey('referral-identity-client'),
          icon: Icons.business_outlined,
          roleLabel: l10n.referralIdentityClientTitle,
          name: clientName ?? referral.clientDisplayName,
        ),
      ],
    );
  }
}

/// _RevealedIdentityTile — minimalist apporteur-only tile. Shows the
/// role label as a small uppercase eyebrow ("Prestataire recommandé"),
/// then the display name in the title face below. NO CTA, NO chevron,
/// NO "Voir le profil" link.
class _RevealedIdentityTile extends StatelessWidget {
  const _RevealedIdentityTile({
    super.key,
    required this.icon,
    required this.roleLabel,
    required this.name,
  });

  final IconData icon;
  final String roleLabel;
  final String? name;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final displayName = (name != null && name!.isNotEmpty) ? name! : '—';
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Row(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: theme.colorScheme.primary.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: Icon(icon, color: theme.colorScheme.primary, size: 20),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  roleLabel,
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                    letterSpacing: 0.8,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  displayName,
                  key: const ValueKey('referral-identity-name'),
                  style: theme.textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.w600,
                    color: theme.colorScheme.onSurface,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
