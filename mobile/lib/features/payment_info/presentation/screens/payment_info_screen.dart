import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/storage/secure_storage.dart';
import '../../../../l10n/app_localizations.dart';

/// Stripe account status fetched from GET /payment-info/account-status.
final _accountStatusProvider = FutureProvider.autoDispose<_AccountStatus?>((ref) async {
  try {
    final client = ref.read(apiClientProvider);
    final response = await client.get('/api/v1/payment-info/account-status');
    return _AccountStatus.fromJson(response.data);
  } catch (_) {
    return null;
  }
});

class _AccountStatus {
  final bool chargesEnabled;
  final bool payoutsEnabled;
  final int requirementsCount;

  const _AccountStatus({
    required this.chargesEnabled,
    required this.payoutsEnabled,
    required this.requirementsCount,
  });

  bool get fullyActive => chargesEnabled && payoutsEnabled && requirementsCount == 0;

  factory _AccountStatus.fromJson(Map<String, dynamic> json) {
    return _AccountStatus(
      chargesEnabled: json['charges_enabled'] ?? false,
      payoutsEnabled: json['payouts_enabled'] ?? false,
      requirementsCount: json['requirements_count'] ?? 0,
    );
  }
}

/// Payment info screen — shows Stripe account status and opens a WebView
/// to the Next.js /payment-info page for KYC onboarding and management.
class PaymentInfoScreen extends ConsumerWidget {
  const PaymentInfoScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final statusAsync = ref.watch(_accountStatusProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.drawerPaymentInfo),
        elevation: 0,
      ),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(_accountStatusProvider),
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            statusAsync.when(
              loading: () => const Center(
                child: Padding(
                  padding: EdgeInsets.all(48),
                  child: CircularProgressIndicator(),
                ),
              ),
              error: (_, __) => _buildContent(context, ref, l10n, null),
              data: (status) => _buildContent(context, ref, l10n, status),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildContent(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
    _AccountStatus? status,
  ) {
    final theme = Theme.of(context);
    final hasAccount = status != null;
    final fullyActive = status?.fullyActive ?? false;

    return Column(
      children: [
        // Status card
        _StatusCard(status: status),
        const SizedBox(height: 24),

        // Action button
        SizedBox(
          width: double.infinity,
          height: 48,
          child: FilledButton.icon(
            onPressed: () => _openWebView(context, ref),
            icon: Icon(
              fullyActive ? Icons.edit_outlined : Icons.arrow_forward_rounded,
            ),
            label: Text(
              fullyActive
                  ? l10n.paymentInfoEdit
                  : hasAccount
                      ? l10n.paymentInfoComplete
                      : l10n.paymentInfoSetup,
            ),
            style: FilledButton.styleFrom(
              backgroundColor: theme.colorScheme.primary,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
              ),
            ),
          ),
        ),
      ],
    );
  }

  Future<void> _openWebView(BuildContext context, WidgetRef ref) async {
    final locale = Localizations.localeOf(context).languageCode;
    // Payment info WebView always uses the production web app so it works
    // from any device without local proxy issues. Override with WEB_PAYMENT_URL
    // for dev if needed.
    const webBaseUrl = String.fromEnvironment(
      'WEB_URL',
      defaultValue: 'http://192.168.1.156:3001',
    );

    // Pass JWT token via URL — the page reads it and uses Authorization
    // header for API calls. No cookie injection needed.
    final storage = ref.read(secureStorageProvider);
    final token = await storage.getAccessToken();

    if (!context.mounted) return;

    final query = token != null
        ? '?token=$token&embedded=true'
        : '?embedded=true';
    Navigator.of(context).push(
      MaterialPageRoute(
        fullscreenDialog: true,
        builder: (_) => _PaymentInfoWebView(
          url: '$webBaseUrl/$locale/payment-info$query',
          title: AppLocalizations.of(context)!.drawerPaymentInfo,
          onDone: () {
            Navigator.of(context).pop();
            ref.invalidate(_accountStatusProvider);
          },
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Status card
// ---------------------------------------------------------------------------

class _StatusCard extends StatelessWidget {
  final _AccountStatus? status;

  const _StatusCard({required this.status});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final hasAccount = status != null;
    final fullyActive = status?.fullyActive ?? false;

    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(16),
        gradient: LinearGradient(
          colors: fullyActive
              ? [Colors.green.shade500, Colors.green.shade600]
              : hasAccount
                  ? [Colors.orange.shade500, Colors.orange.shade600]
                  : [Colors.grey.shade400, Colors.grey.shade500],
        ),
      ),
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            fullyActive
                ? Icons.check_circle_rounded
                : hasAccount
                    ? Icons.warning_rounded
                    : Icons.credit_card_off_rounded,
            color: Colors.white,
            size: 32,
          ),
          const SizedBox(height: 12),
          Text(
            fullyActive
                ? l10n.paymentInfoActive
                : hasAccount
                    ? l10n.paymentInfoPending
                    : l10n.paymentInfoNotConfigured,
            style: theme.textTheme.titleLarge?.copyWith(
              color: Colors.white,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            fullyActive
                ? l10n.paymentInfoActiveDesc
                : hasAccount
                    ? l10n.paymentInfoPendingDesc(status!.requirementsCount)
                    : l10n.paymentInfoNotConfiguredDesc,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: Colors.white.withValues(alpha: 0.9),
            ),
          ),
          if (hasAccount) ...[
            const SizedBox(height: 16),
            Row(
              children: [
                _CapabilityChip(
                  label: l10n.paymentInfoCharges,
                  enabled: status!.chargesEnabled,
                ),
                const SizedBox(width: 8),
                _CapabilityChip(
                  label: l10n.paymentInfoPayouts,
                  enabled: status!.payoutsEnabled,
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

class _CapabilityChip extends StatelessWidget {
  final String label;
  final bool enabled;

  const _CapabilityChip({required this.label, required this.enabled});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.2),
        borderRadius: BorderRadius.circular(20),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: enabled ? Colors.greenAccent : Colors.orangeAccent,
            ),
          ),
          const SizedBox(width: 6),
          Text(
            label,
            style: const TextStyle(color: Colors.white, fontSize: 12),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// WebView screen (full-screen modal)
// ---------------------------------------------------------------------------

class _PaymentInfoWebView extends StatelessWidget {
  final String url;
  final String title;
  final VoidCallback onDone;

  const _PaymentInfoWebView({
    required this.url,
    required this.title,
    required this.onDone,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(title),
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: onDone,
        ),
      ),
      body: InAppWebView(
        initialUrlRequest: URLRequest(url: WebUri(url)),
        initialSettings: InAppWebViewSettings(
          javaScriptEnabled: true,
          supportZoom: false,
          transparentBackground: true,
        ),
      ),
    );
  }
}
