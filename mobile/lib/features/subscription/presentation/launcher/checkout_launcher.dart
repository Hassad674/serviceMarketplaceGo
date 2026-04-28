import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/router/app_router.dart';

/// Contract for launching a Stripe-hosted URL (Checkout Session or
/// Billing Portal) from within the app.
///
/// Takes a [BuildContext] so the default implementation can push the
/// in-app WebView route via `GoRouter`. The context is intentionally
/// NOT stored — implementations read it once and discard it.
///
/// Kept as a thin abstraction so tests can inject a deterministic mock
/// without pulling in flutter_inappwebview or url_launcher — see
/// `test/features/subscription/helpers/subscription_test_helpers.dart`.
abstract class CheckoutLauncher {
  /// Opens [url] keeping the user inside the app as much as possible.
  /// Returns `true` when the URL was handed off successfully, `false`
  /// otherwise. Implementations must NOT throw — surface failure via
  /// the boolean so callers can show a SnackBar.
  Future<bool> launch(BuildContext context, String url);
}

/// Production [CheckoutLauncher]. Navigates to the in-app WebView
/// screen that hosts the Stripe page. The WebView intercepts the
/// return URL (`/billing/success` or `/billing/cancel`) and routes
/// back into the Flutter app — no external browser, no reliance on
/// Android App Links / iOS Universal Links.
///
/// If the URL is malformed we do not even attempt the push — callers
/// get `false` and surface their standard error UI.
///
/// Never logs the URL itself — Stripe Checkout / Portal URLs are
/// one-time credentials and must be treated as sensitive.
class WebViewCheckoutLauncher implements CheckoutLauncher {
  const WebViewCheckoutLauncher();

  @override
  Future<bool> launch(BuildContext context, String url) async {
    final Uri? uri = Uri.tryParse(url);
    if (uri == null || !uri.hasScheme) {
      debugPrint('[checkout_launcher] refused malformed URL');
      return false;
    }
    if (!_isAllowedScheme(uri)) {
      debugPrint('[checkout_launcher] refused non-https URL');
      return false;
    }
    try {
      context.push(RoutePaths.checkoutWebview, extra: url);
      return true;
    } catch (e) {
      debugPrint('[checkout_launcher] navigation push failed: $e');
      return false;
    }
  }
}

// Allow HTTPS unconditionally (production) AND allow HTTP for
// localhost + private LAN IPs (RFC1918) so a flutter run pointed at
// `http://192.168.1.156:3001` works during development without
// loosening the production posture. Public HTTP URLs are still
// refused — Checkout Sessions / Portal URLs are one-time credentials
// that must travel over TLS in prod.
bool _isAllowedScheme(Uri uri) {
  if (uri.isScheme('https')) return true;
  if (!uri.isScheme('http')) return false;
  final host = uri.host.toLowerCase();
  if (host == 'localhost' || host == '127.0.0.1' || host == '::1') {
    return true;
  }
  // RFC1918 private ranges: 10/8, 172.16/12, 192.168/16.
  if (host.startsWith('10.') || host.startsWith('192.168.')) return true;
  if (host.startsWith('172.')) {
    final parts = host.split('.');
    if (parts.length >= 2) {
      final second = int.tryParse(parts[1]);
      if (second != null && second >= 16 && second <= 31) return true;
    }
  }
  return false;
}

/// Legacy [CheckoutLauncher] that opens the URL in the system browser
/// via `url_launcher`. Kept available for the fallback path the
/// WebView screen exposes when the embedded renderer errors (3DS,
/// Apple Pay / Google Pay, TLS failures). Not the default production
/// binding.
class ExternalBrowserCheckoutLauncher implements CheckoutLauncher {
  const ExternalBrowserCheckoutLauncher();

  @override
  Future<bool> launch(BuildContext context, String url) async {
    final Uri? uri = Uri.tryParse(url);
    if (uri == null || !uri.hasScheme) return false;
    try {
      return await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (e) {
      debugPrint('[checkout_launcher] external launch failed: $e');
      return false;
    }
  }
}

/// Riverpod handle for the [CheckoutLauncher]. Production wires the
/// WebView impl; tests override this provider with a recording mock.
final checkoutLauncherProvider = Provider<CheckoutLauncher>((ref) {
  return const WebViewCheckoutLauncher();
});
