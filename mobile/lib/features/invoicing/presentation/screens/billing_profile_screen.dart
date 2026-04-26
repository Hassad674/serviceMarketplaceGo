import 'package:flutter/material.dart';

import '../widgets/billing_profile_form.dart';

/// Page-level wrapper for [BillingProfileForm].
///
/// Standard scaffold + AppBar + back button + scrollable body. Reached
/// either from the drawer ("Mes factures" → invoice page → form link)
/// or from the gate modal that pops on wallet/subscribe flows.
class BillingProfileScreen extends StatelessWidget {
  const BillingProfileScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Profil de facturation'),
      ),
      body: const SafeArea(
        child: SingleChildScrollView(
          padding: EdgeInsets.all(16),
          child: BillingProfileForm(),
        ),
      ),
    );
  }
}
