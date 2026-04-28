import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/router/app_router.dart';

/// In-app WebView hosting a Stripe-hosted URL (Checkout Session or
/// Customer Portal). The screen owns navigation plumbing so the caller
/// only has to push it with the URL.
///
/// Why this exists: Stripe does NOT publish a Flutter SDK for Checkout.
/// Custom Tabs feel like leaving the app because of the Chrome chrome
/// at the top. A plain in-app WebView keeps the user inside our app
/// chrome AND lets us intercept the success/cancel return trip in-code
/// instead of relying on Android App Links (which require a hosted
/// assetlinks.json that we do not yet publish).
///
/// The screen is deliberately defensive:
///
///   * Every navigation to our `/billing/success` or `/billing/cancel`
///     return URL is intercepted BEFORE the browser loads it — we
///     close the WebView and call `GoRouter.go(...)` into the in-app
///     route so the user lands on the existing BillingSuccessScreen
///     polling widget, not a blank web page.
///   * Load errors (offline, TLS, Stripe outage) surface a fallback
///     "Ouvrir dans le navigateur" button that hands the URL off to
///     `url_launcher` — the external browser is the last resort for
///     3DS redirects or payment methods that refuse to run in a
///     WebView (Apple Pay / Google Pay / some bank apps).
///   * The back button on the AppBar maps to "cancel": Stripe expects
///     a round-trip and our BillingCancelScreen shows a stable UX.
class CheckoutWebViewScreen extends StatefulWidget {
  const CheckoutWebViewScreen({super.key, required this.url});

  final String url;

  @override
  State<CheckoutWebViewScreen> createState() => _CheckoutWebViewScreenState();
}

class _CheckoutWebViewScreenState extends State<CheckoutWebViewScreen> {
  bool _loading = true;
  bool _errored = false;

  /// Paths we watch for to close the WebView and hand control back to
  /// GoRouter. Both are configured on the backend as the Checkout
  /// Session `success_url` / `cancel_url`, so Stripe guarantees one of
  /// them resolves at the end of the flow.
  static const _successPath = '/billing/success';
  static const _cancelPath = '/billing/cancel';

  Future<NavigationActionPolicy> _onNavigation(
    InAppWebViewController _,
    NavigationAction action,
  ) async {
    final uri = action.request.url;
    if (uri == null) return NavigationActionPolicy.ALLOW;
    final path = uri.path;

    if (path.startsWith(_successPath)) {
      _routeIntoApp(RoutePaths.billingSuccess);
      return NavigationActionPolicy.CANCEL;
    }
    if (path.startsWith(_cancelPath)) {
      _routeIntoApp(RoutePaths.billingCancel);
      return NavigationActionPolicy.CANCEL;
    }
    return NavigationActionPolicy.ALLOW;
  }

  /// Replaces the WebView in the navigation stack with the target
  /// in-app route. `pushReplacement` keeps the back button sane — the
  /// user can come back to their previous screen without bouncing back
  /// through Stripe.
  void _routeIntoApp(String route) {
    if (!mounted) return;
    GoRouter.of(context).pushReplacement(route);
  }

  Future<void> _openInExternalBrowser() async {
    final uri = Uri.tryParse(widget.url);
    if (uri == null) return;
    await launchUrl(uri, mode: LaunchMode.externalApplication);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Paiement sécurisé'),
        leading: IconButton(
          icon: const Icon(Icons.close),
          tooltip: 'Annuler le paiement',
          onPressed: () => _routeIntoApp(RoutePaths.billingCancel),
        ),
      ),
      body: Stack(
        children: [
          if (!_errored)
            InAppWebView(
              initialUrlRequest: URLRequest(url: WebUri(widget.url)),
              initialSettings: InAppWebViewSettings(
                javaScriptEnabled: true,
                useHybridComposition: true,
                transparentBackground: true,
                // Some 3DS / Payment Element iframes need third-party
                // cookies — enabled on Android, iOS handles it via
                // WKWebView defaults.
                thirdPartyCookiesEnabled: true,
                // Stripe advertises itself via user agent in some flows;
                // use the default mobile UA so it doesn't fall back to
                // desktop chrome.
                userAgent: '',
                // Payment methods like Apple Pay / Google Pay rely on
                // the browser supporting the Payment Request API. The
                // WebView does not, which is why we expose the
                // external-browser fallback on error.
              ),
              shouldOverrideUrlLoading: _onNavigation,
              onLoadStop: (_, __) {
                if (mounted) setState(() => _loading = false);
              },
              onReceivedError: (_, request, error) {
                if (kDebugMode) {
                  debugPrint(
                    'checkout webview error: ${error.description} '
                    'for=${request.url} mainFrame=${request.isForMainFrame}',
                  );
                }
                // Only flip into the error state when the MAIN frame
                // fails. Stripe Embedded Checkout loads a handful of
                // iframes (analytics, payment-method-specific runners)
                // that can 404 / connection-reset / blocked-by-policy
                // without breaking the actual checkout — flagging the
                // whole screen on any of those was the cause of the
                // false "Impossible de charger" error overlay.
                if (request.isForMainFrame != true) return;
                if (mounted) {
                  setState(() {
                    _loading = false;
                    _errored = true;
                  });
                }
              },
              onReceivedHttpError: (_, request, response) {
                // Same scoping rule: only the main-frame document's
                // HTTP errors should flip the screen into the error
                // state. Sub-resource 4xx/5xx are noise here — Stripe
                // analytics is famous for 404'ing in test mode without
                // breaking the checkout.
                if (request.isForMainFrame != true) return;
                if (response.statusCode != null &&
                    response.statusCode! >= 500) {
                  if (mounted) {
                    setState(() {
                      _loading = false;
                      _errored = true;
                    });
                  }
                }
              },
            ),
          if (_loading && !_errored)
            const Center(child: CircularProgressIndicator()),
          if (_errored)
            Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(
                      Icons.wifi_off,
                      size: 48,
                      color: theme.colorScheme.error,
                    ),
                    const SizedBox(height: 16),
                    Text(
                      'Impossible de charger le paiement',
                      style: theme.textTheme.titleMedium,
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 8),
                    Text(
                      "Vérifie ta connexion ou ouvre le paiement dans ton navigateur.",
                      style: theme.textTheme.bodyMedium,
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 20),
                    ElevatedButton.icon(
                      onPressed: _openInExternalBrowser,
                      icon: const Icon(Icons.open_in_browser),
                      label: const Text('Ouvrir dans le navigateur'),
                    ),
                    const SizedBox(height: 8),
                    OutlinedButton(
                      onPressed: () =>
                          _routeIntoApp(RoutePaths.billingCancel),
                      child: const Text('Retour'),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }

}
