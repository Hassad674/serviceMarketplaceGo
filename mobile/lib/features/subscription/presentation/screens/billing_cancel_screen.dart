import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// Landing screen for a cancelled or abandoned Stripe Checkout session.
/// Static — no async work.
class BillingCancelScreen extends StatelessWidget {
  const BillingCancelScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('Premium')),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(
                  Icons.info_outline,
                  size: 56,
                  color: theme.colorScheme.primary,
                ),
                const SizedBox(height: 16),
                Text(
                  'Abonnement non confirmé',
                  style: theme.textTheme.headlineMedium,
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 8),
                Text(
                  "Le paiement n'a pas été finalisé. Aucun montant n'a été prélevé.",
                  style: theme.textTheme.bodyMedium,
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 24),
                SizedBox(
                  width: double.infinity,
                  child: ElevatedButton(
                    onPressed: () => context.go('/pricing'),
                    child: const Text('Réessayer'),
                  ),
                ),
                const SizedBox(height: 8),
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton(
                    onPressed: () => context.go('/dashboard'),
                    child: const Text('Retour'),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
