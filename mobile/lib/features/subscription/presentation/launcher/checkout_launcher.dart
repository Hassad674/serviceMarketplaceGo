import 'package:flutter/widgets.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/network/api_client.dart';
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
///
/// When [sessionBridge] is provided, the launcher mints a web session
/// for the bearer-authenticated user and injects the session_id
/// cookie into the WebView's CookieManager BEFORE pushing the route,
/// so the embed page is already authenticated when it loads. Without
/// this, the Next.js middleware redirects the user to /login because
/// the WebView starts cookie-less. If the bridge call fails we still
/// open the WebView — the user just sees the login page and can
/// continue manually, instead of being blocked entirely.
class WebViewCheckoutLauncher implements CheckoutLauncher {
  const WebViewCheckoutLauncher({this.sessionBridge});

  final WebSessionBridge? sessionBridge;

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
    if (sessionBridge != null) {
      await sessionBridge!.injectInto(uri);
    }
    if (!context.mounted) return false;
    try {
      context.push(RoutePaths.checkoutWebview, extra: url);
      return true;
    } catch (e) {
      debugPrint('[checkout_launcher] navigation push failed: $e');
      return false;
    }
  }
}

/// Mints a fresh web session for the bearer-authenticated user and
/// installs the matching session_id cookie into the in-app WebView's
/// shared CookieManager so the WebView opens already authenticated
/// against the web app. Stateless — pulls a new session on every call.
class WebSessionBridge {
  WebSessionBridge(this._apiClient);

  final ApiClient _apiClient;

  /// Mints a session and writes the cookie scoped to [target.host].
  /// Errors are swallowed (logged in debug) — the launcher proceeds
  /// regardless so the user can still manually sign in within the
  /// WebView if the bridge transiently fails.
  Future<void> injectInto(Uri target) async {
    try {
      final response = await _apiClient.post<Map<String, dynamic>>(
        '/api/v1/auth/web-session',
      );
      final body = response.data;
      if (body == null) return;
      final sessionID = body['session_id'] as String?;
      final maxAge = body['max_age_seconds'] as int?;
      if (sessionID == null || sessionID.isEmpty) return;

      final manager = CookieManager.instance();
      // The session_id is httpOnly on the server response; setting it
      // from the WebView side is independent — this matches the value
      // the browser would have received via Set-Cookie.
      //
      // We deliberately DO NOT pass a `domain` argument when the host
      // is a raw IP. RFC 6265 forbids the `Domain` attribute on IP
      // addresses; browsers (and WebViews) reject the Set-Cookie
      // entirely, leaving the WebView cookie-less and the user stuck
      // on the login redirect. Omitting the parameter creates a
      // "host-only" cookie which is exactly what we want for the LAN
      // dev URL. For DNS hosts (production), the WebView still scopes
      // to the URL's host implicitly.
      await manager.setCookie(
        url: WebUri('${target.scheme}://${target.host}:${target.port}'),
        name: 'session_id',
        value: sessionID,
        path: '/',
        maxAge: maxAge,
        isSecure: target.scheme == 'https',
        isHttpOnly: true,
        sameSite: HTTPCookieSameSitePolicy.LAX,
      );
    } catch (e) {
      debugPrint('[checkout_launcher] web-session bridge failed: $e');
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
  final apiClient = ref.read(apiClientProvider);
  return WebViewCheckoutLauncher(
    sessionBridge: WebSessionBridge(apiClient),
  );
});
