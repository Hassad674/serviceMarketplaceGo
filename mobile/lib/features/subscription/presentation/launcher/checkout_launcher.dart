import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

/// Contract for launching a Stripe hosted URL (Checkout session or
/// Billing Portal) from within the app.
///
/// Kept as a thin abstraction so tests can inject a deterministic mock
/// without touching `url_launcher` — see agent 5D's widget tests.
abstract class CheckoutLauncher {
  /// Opens [url] in an in-app browser tab when available so the user
  /// never fully leaves the Flutter app while paying. Returns `true`
  /// when the URL was successfully handed off to the platform, `false`
  /// otherwise. Implementations must NOT throw — surface failure via
  /// the boolean so callers can show a SnackBar.
  Future<bool> launch(String url);
}

/// Production [CheckoutLauncher] backed by the `url_launcher` plugin.
///
/// Strategy:
///  1. Try `LaunchMode.inAppBrowserView` — Chrome Custom Tabs on
///     Android and `SFSafariViewController` on iOS. Keeps the user
///     inside the app context so the return trip via universal /
///     App Links opens the Flutter app directly.
///  2. On failure (some devices do not ship an in-app tab provider,
///     e.g. AOSP without Chrome), fall back to
///     `LaunchMode.externalApplication`.
///
/// Never logs the URL itself — Stripe portal URLs are one-time
/// credentials and must be treated as sensitive.
class UrlLauncherCheckoutLauncher implements CheckoutLauncher {
  const UrlLauncherCheckoutLauncher();

  @override
  Future<bool> launch(String url) async {
    final Uri? uri = Uri.tryParse(url);
    if (uri == null || !uri.hasScheme) {
      debugPrint('[checkout_launcher] invalid URL, refusing to launch');
      return false;
    }
    try {
      final bool ok = await launchUrl(
        uri,
        mode: LaunchMode.inAppBrowserView,
      );
      if (ok) return true;
      debugPrint(
        '[checkout_launcher] in-app browser unavailable, '
        'falling back to externalApplication',
      );
    } catch (e) {
      debugPrint(
        '[checkout_launcher] in-app browser threw: $e — '
        'falling back to externalApplication',
      );
    }
    try {
      return await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (e) {
      debugPrint('[checkout_launcher] external launch failed: $e');
      return false;
    }
  }
}

/// Riverpod handle for the [CheckoutLauncher]. Production wires the
/// `url_launcher` impl; tests override this provider with a mock.
final checkoutLauncherProvider = Provider<CheckoutLauncher>((ref) {
  return const UrlLauncherCheckoutLauncher();
});
