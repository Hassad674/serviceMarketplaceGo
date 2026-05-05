import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../widgets/billing_profile_form.dart';

// Allow-list of paths the screen will route to after save. Mirrors
// the web safeReturnTo guard so a malicious deep link can't redirect
// the user out of the app to an attacker-controlled URL.
const _allowedReturnTo = <String>{
  '/wallet',
  '/payment-info',
  '/invoices',
  '/settings',
};

String? _safeReturnTo(String? raw) {
  if (raw == null || raw.isEmpty) return null;
  if (!raw.startsWith('/')) return null;
  if (!_allowedReturnTo.contains(raw)) return null;
  return raw;
}

/// Soleil v2 page-level wrapper for [BillingProfileForm].
///
/// AppBar Fraunces "Profil de facturation" + editorial header (corail
/// mono eyebrow + Fraunces italic-corail title + tabac subtitle), then
/// the existing Riverpod-backed form embedded as-is.
///
/// Reached either from the drawer ("Mes factures" → invoice page →
/// form link) or from the gate modal that pops on wallet/subscribe
/// flows. When the gate-modal route includes `?return_to=/wallet`,
/// the screen pops back to that destination once the profile passes
/// completeness — that flow is preserved verbatim from the legacy
/// implementation.
class BillingProfileScreen extends StatelessWidget {
  const BillingProfileScreen({super.key, this.returnTo});

  final String? returnTo;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final safe = _safeReturnTo(returnTo);
    return Scaffold(
      backgroundColor: colorScheme.surface,
      appBar: AppBar(
        backgroundColor: colorScheme.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: Text(
          'Profil de facturation',
          style: SoleilTextStyles.titleMedium.copyWith(
            color: colorScheme.onSurface,
          ),
        ),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const _EditorialHeader(),
              const SizedBox(height: 20),
              BillingProfileForm(
                onSaved: safe == null
                    ? null
                    : () {
                        if (!context.mounted) return;
                        context.go(safe);
                      },
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _EditorialHeader extends StatelessWidget {
  const _EditorialHeader();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final primary = colorScheme.primary;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'ATELIER · PROFIL DE FACTURATION',
          style: SoleilTextStyles.mono.copyWith(
            color: primary,
            fontSize: 11,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.4,
          ),
        ),
        const SizedBox(height: 8),
        RichText(
          text: TextSpan(
            style: SoleilTextStyles.headlineLarge.copyWith(
              color: colorScheme.onSurface,
            ),
            children: [
              const TextSpan(text: 'Renseigne ton '),
              TextSpan(
                text: 'profil de facturation.',
                style: SoleilTextStyles.headlineLarge.copyWith(
                  color: primary,
                  fontStyle: FontStyle.italic,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        Text(
          "Ces informations apparaissent sur les factures que la plateforme "
          "émet à ton organisation. Elles doivent être complètes pour "
          "pouvoir retirer ton solde et souscrire à un abonnement Premium.",
          style: SoleilTextStyles.bodyLarge.copyWith(
            color: colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}
