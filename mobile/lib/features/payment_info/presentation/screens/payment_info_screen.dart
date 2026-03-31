import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:webview_flutter/webview_flutter.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/router/app_router.dart';
import '../../../../core/storage/secure_storage.dart';
import '../../../../l10n/app_localizations.dart';
import '../providers/payment_info_provider.dart';

// ---------------------------------------------------------------------------
// Screen
// ---------------------------------------------------------------------------

/// Payment info screen that uses Stripe Connect Embedded Components
/// via a WebView pointing to the hosted onboarding page.
class PaymentInfoScreen extends ConsumerStatefulWidget {
  const PaymentInfoScreen({super.key});

  @override
  ConsumerState<PaymentInfoScreen> createState() => _PaymentInfoScreenState();
}

class _PaymentInfoScreenState extends ConsumerState<PaymentInfoScreen> {
  bool _showWebView = false;

  void _openOnboarding() {
    setState(() => _showWebView = true);
  }

  void _onOnboardingComplete() {
    setState(() => _showWebView = false);
    ref.invalidate(paymentInfoProvider);
    ref.invalidate(paymentInfoStatusProvider);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final asyncInfo = ref.watch(paymentInfoProvider);

    if (_showWebView) {
      return _OnboardingWebView(
        onComplete: _onOnboardingComplete,
      );
    }

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.paymentInfoTitle),
      ),
      body: asyncInfo.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => _ErrorView(
          error: error.toString(),
          onRetry: () => ref.invalidate(paymentInfoProvider),
        ),
        data: (info) {
          if (info != null && info.stripeVerified) {
            return _VerifiedView(accountId: info.stripeAccountId);
          }

          if (info != null && info.stripeAccountId.isNotEmpty) {
            return _PendingView(onContinue: _openOnboarding);
          }

          return _SetupView(onSetup: _openOnboarding);
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Sub-views
// ---------------------------------------------------------------------------

class _VerifiedView extends StatelessWidget {
  const _VerifiedView({required this.accountId});
  final String accountId;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.check_circle, size: 64, color: Colors.green[600]),
            const SizedBox(height: 16),
            Text(
              'Account Verified',
              style: theme.textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            Text(
              'Your Stripe account is active. You can receive payments.',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
              ),
            ),
            const SizedBox(height: 8),
            Text(
              accountId,
              style: theme.textTheme.bodySmall?.copyWith(
                fontFamily: 'monospace',
                color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _PendingView extends StatelessWidget {
  const _PendingView({required this.onContinue});
  final VoidCallback onContinue;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.shield, size: 64, color: Colors.amber[600]),
            const SizedBox(height: 16),
            Text(
              'Verification Pending',
              style: theme.textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            Text(
              'Your Stripe account requires additional information.',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyMedium,
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: onContinue,
              child: const Text('Complete Verification'),
            ),
          ],
        ),
      ),
    );
  }
}

class _SetupView extends StatelessWidget {
  const _SetupView({required this.onSetup});
  final VoidCallback onSetup;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.credit_card, size: 64, color: Colors.pink[400]),
            const SizedBox(height: 16),
            Text(
              'Set Up Payments',
              style: theme.textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            Text(
              'To receive payments, set up your Stripe account. '
              'It only takes a few minutes.',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyMedium,
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: onSetup,
              child: const Text('Set Up My Account'),
            ),
          ],
        ),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final String error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text('Error: $error'),
          const SizedBox(height: 8),
          ElevatedButton(onPressed: onRetry, child: const Text('Retry')),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// WebView for Stripe onboarding
// ---------------------------------------------------------------------------

class _OnboardingWebView extends ConsumerStatefulWidget {
  const _OnboardingWebView({required this.onComplete});
  final VoidCallback onComplete;

  @override
  ConsumerState<_OnboardingWebView> createState() =>
      _OnboardingWebViewState();
}

class _OnboardingWebViewState extends ConsumerState<_OnboardingWebView> {
  late final WebViewController _controller;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _initWebView();
  }

  Future<void> _initWebView() async {
    final storage = ref.read(secureStorageProvider);
    final token = await storage.getAccessToken();

    // Build the onboarding URL
    final baseUrl = ApiClient.baseUrl.replaceAll('/api', '');
    final url = '${baseUrl.replaceAll(':8083', ':3000')}/en/onboarding-embed';

    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(NavigationDelegate(
        onPageFinished: (_) {
          if (mounted) setState(() => _loading = false);
        },
      ))
      ..addJavaScriptChannel(
        'FlutterChannel',
        onMessageReceived: (message) {
          if (message.message == 'onboarding-complete') {
            widget.onComplete();
          }
        },
      );

    // Set the auth cookie before loading the page
    if (token != null) {
      await _controller.runJavaScript('''
        document.cookie = "access_token=$token; path=/;";
      ''');
    }

    await _controller.loadRequest(Uri.parse(url));
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: widget.onComplete,
        ),
        title: const Text('Account Setup'),
      ),
      body: Stack(
        children: [
          WebViewWidget(controller: _controller),
          if (_loading)
            const Center(child: CircularProgressIndicator()),
        ],
      ),
    );
  }
}
