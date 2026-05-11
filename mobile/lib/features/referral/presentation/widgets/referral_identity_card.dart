import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/referral_entity.dart';
import 'anonymized_client_card.dart';
import 'anonymized_provider_card.dart';

/// ReferralIdentityCard — WALLET-UNIFY Run D parity with web Run C
/// `referral-detail-view`. When the viewer is the apporteur (the
/// "is_owner" branch on web — `viewerId == referral.referrerId`) we
/// reveal the clear provider + client names with tap-to-profile links;
/// otherwise we fall back to the existing anonymized cards.
///
/// The decision is computed by the caller and forwarded via [isOwner]
/// so this widget stays pure for testing.
class ReferralIdentityCard extends StatelessWidget {
  const ReferralIdentityCard({
    super.key,
    required this.referral,
    required this.isOwner,
    this.providerName,
    this.clientName,
    this.onProviderTap,
    this.onClientTap,
  });

  final Referral referral;
  final bool isOwner;
  final String? providerName;
  final String? clientName;
  final VoidCallback? onProviderTap;
  final VoidCallback? onClientTap;

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
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _RevealedIdentityTile(
          key: const ValueKey('referral-identity-provider'),
          icon: Icons.work_outline,
          label: AppLocalizations.of(context)!
              .referralIdentityRevealProviderLink,
          name: providerName,
          fallback:
              referral.introSnapshot.provider.expertiseDomains.isNotEmpty
                  ? referral.introSnapshot.provider.expertiseDomains.first
                  : null,
          onTap: onProviderTap,
        ),
        const SizedBox(height: 12),
        _RevealedIdentityTile(
          key: const ValueKey('referral-identity-client'),
          icon: Icons.business_outlined,
          label: AppLocalizations.of(context)!
              .referralIdentityRevealClientLink,
          name: clientName,
          fallback: referral.introSnapshot.client.industry,
          onTap: onClientTap,
        ),
      ],
    );
  }
}

class _RevealedIdentityTile extends StatelessWidget {
  const _RevealedIdentityTile({
    super.key,
    required this.icon,
    required this.label,
    required this.name,
    required this.fallback,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final String? name;
  final String? fallback;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final displayName = (name != null && name!.isNotEmpty)
        ? name!
        : (fallback ?? '');
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      child: Container(
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
                borderRadius:
                    BorderRadius.circular(AppTheme.radiusMd),
              ),
              child: Icon(icon, color: theme.colorScheme.primary, size: 20),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (displayName.isNotEmpty)
                    Text(
                      displayName,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                        color: theme.colorScheme.onSurface,
                      ),
                    ),
                  const SizedBox(height: 2),
                  Text(
                    label,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.primary,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ],
              ),
            ),
            if (onTap != null)
              Icon(
                Icons.arrow_forward_ios,
                size: 14,
                color: theme.colorScheme.primary,
              ),
          ],
        ),
      ),
    );
  }
}
