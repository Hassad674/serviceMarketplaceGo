/// Static app-level configuration. Values are wired through
/// `--dart-define=KEY=VALUE` flags at build time (or `.env` via the
/// `flutter_dotenv` package if we adopt it later) so the same APK can
/// target different environments without code changes.
class AppConfig {
  /// Base URL of the marketing / web app. Used by features that open
  /// an in-app WebView pointing at a web page we control — currently
  /// the embedded subscribe flow (`/subscribe/embed`) which renders
  /// our country-aware billing form + Stripe Embedded Checkout.
  ///
  /// Override at build time with:
  ///   flutter run --dart-define=WEB_ORIGIN_URL=https://app.example.com
  static const String webOriginUrl = String.fromEnvironment(
    'WEB_ORIGIN_URL',
    defaultValue: 'http://192.168.1.156:3001',
  );

  /// App locale segment baked into URLs we hand off to the WebView.
  /// next-intl uses `/[locale]/...` so we have to provide one — this
  /// matches the default locale of the web build. Override only when
  /// shipping a localized variant.
  static const String webLocaleSegment = String.fromEnvironment(
    'WEB_LOCALE_SEGMENT',
    defaultValue: 'fr',
  );
}
