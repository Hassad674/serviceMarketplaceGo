import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

/// Dialog shown when the backend rejects a commission retry with the
/// `kyc_required` code (D1+D2). Offers a deep-link to finish Stripe
/// onboarding when [onboardingUrl] is set, otherwise routes to the
/// in-app /payment-info screen for the same flow.
///
/// The dialog is intentionally compact — no logic beyond rendering and
/// CTA dispatch. The screen that opens it owns the navigation
/// fallback so this widget stays free of GoRouter coupling.
class CommissionKYCRequiredDialog extends StatelessWidget {
  const CommissionKYCRequiredDialog({
    super.key,
    this.onboardingUrl,
    this.onPaymentInfoTap,
  });

  /// Stripe Connect onboarding URL extracted from the 422 envelope.
  /// When null the dialog hides the "Finish KYC" CTA and surfaces the
  /// in-app fallback only.
  final String? onboardingUrl;

  /// Called when the user taps the in-app fallback CTA. The host
  /// screen is expected to `context.push('/payment-info')` (or its
  /// route equivalent) — this keeps the dialog free of GoRouter
  /// coupling so it can be reused in widget tests.
  final VoidCallback? onPaymentInfoTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasOnboarding = onboardingUrl != null && onboardingUrl!.isNotEmpty;
    return AlertDialog(
      title: Text(
        'Finish KYC to receive your commission',
        style: theme.textTheme.titleMedium,
      ),
      content: Text(
        'Your commission is ready to be paid out, but your Stripe Connect '
        'account has not enabled payouts yet. Finish onboarding to receive '
        'the transfer.',
        style: theme.textTheme.bodyMedium,
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Later'),
        ),
        if (hasOnboarding)
          FilledButton(
            onPressed: () async {
              final uri = Uri.parse(onboardingUrl!);
              if (await canLaunchUrl(uri)) {
                await launchUrl(uri, mode: LaunchMode.externalApplication);
              }
              if (context.mounted) {
                Navigator.of(context).pop();
              }
            },
            child: const Text('Finish KYC'),
          )
        else
          FilledButton(
            onPressed: () {
              Navigator.of(context).pop();
              onPaymentInfoTap?.call();
            },
            child: const Text('Open payment info'),
          ),
      ],
    );
  }
}
