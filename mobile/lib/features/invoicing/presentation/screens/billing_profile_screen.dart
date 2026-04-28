import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

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

/// Page-level wrapper for [BillingProfileForm].
///
/// Standard scaffold + AppBar + back button + scrollable body. Reached
/// either from the drawer ("Mes factures" → invoice page → form link)
/// or from the gate modal that pops on wallet/subscribe flows. When
/// the gate-modal route includes `?return_to=/wallet`, the screen
/// pops back to that destination once the profile passes completeness.
class BillingProfileScreen extends StatelessWidget {
  const BillingProfileScreen({super.key, this.returnTo});

  final String? returnTo;

  @override
  Widget build(BuildContext context) {
    final safe = _safeReturnTo(returnTo);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Profil de facturation'),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: BillingProfileForm(
            onSaved: safe == null
                ? null
                : () {
                    if (!context.mounted) return;
                    context.go(safe);
                  },
          ),
        ),
      ),
    );
  }
}
