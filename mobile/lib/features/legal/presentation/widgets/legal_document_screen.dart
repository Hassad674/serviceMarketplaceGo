import 'package:flutter/material.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter_markdown/flutter_markdown.dart';

import '../../../../core/theme/app_theme.dart';

/// Shared scaffolding for the 6 long-form legal document screens
/// (registre, AIPD, DPA template, politique de confidentialité, CGU,
/// CGV).
///
/// The 6 wrapper screens each instantiate this widget with their own
/// (title, subtitle, asset path) — the 200+ LOC of theming, async asset
/// loading, English-notice banner, and "last updated" footer live here
/// once rather than being duplicated six times.
///
/// The markdown source is loaded through [rootBundle] (bundled at
/// build time — see `pubspec.yaml`), so the screen renders deterministically
/// offline. A small [_assetLoaderOverride] hook is exposed for widget
/// tests that want to skip the platform channel and inject content
/// directly.
class LegalDocumentScreen extends StatelessWidget {
  const LegalDocumentScreen({
    super.key,
    required this.title,
    required this.subtitle,
    required this.assetPath,
    required this.englishNotice,
    required this.lastUpdatedLabel,
    Future<String> Function(String path)? assetLoader,
  }) : _assetLoaderOverride = assetLoader;

  /// AppBar title (resolved from i18n by the caller).
  final String title;

  /// One-line subtitle rendered above the markdown body.
  final String subtitle;

  /// Path of the bundled markdown asset — e.g. `assets/legal/registre.md`.
  final String assetPath;

  /// "English version available on request…" banner copy (i18n).
  final String englishNotice;

  /// Footer copy — "Dernière mise à jour : 2026-05-11" (i18n with date
  /// interpolated by the caller, mirrors web `legal.docs.lastUpdatedISO`).
  final String lastUpdatedLabel;

  /// Test seam — lets widget tests inject a fake asset string without
  /// touching the asset bundle platform channel.
  final Future<String> Function(String path)? _assetLoaderOverride;

  Future<String> _loadMarkdown() {
    final loader = _assetLoaderOverride;
    if (loader != null) return loader(assetPath);
    return rootBundle.loadString(assetPath);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: SafeArea(
        child: FutureBuilder<String>(
          future: _loadMarkdown(),
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done) {
              return const Center(child: CircularProgressIndicator());
            }
            if (snapshot.hasError || snapshot.data == null) {
              return _LegalLoadError(message: snapshot.error?.toString());
            }
            return _LegalDocumentBody(
              subtitle: subtitle,
              markdown: snapshot.data!,
              englishNotice: englishNotice,
              lastUpdatedLabel: lastUpdatedLabel,
            );
          },
        ),
      ),
    );
  }
}

/// Inline error rendered when the asset fails to load. Should never
/// happen in prod (the asset is bundled at build time) but covered
/// here so a missing pubspec entry surfaces visibly in dev/QA.
class _LegalLoadError extends StatelessWidget {
  const _LegalLoadError({this.message});

  final String? message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.error_outline,
            color: theme.colorScheme.error,
            size: 32,
          ),
          const SizedBox(height: 12),
          Text(
            'Document indisponible.',
            style: SoleilTextStyles.bodyEmphasis.copyWith(
              color: theme.colorScheme.onSurface,
            ),
          ),
          if (message != null) ...[
            const SizedBox(height: 4),
            Text(
              message!,
              style: SoleilTextStyles.caption.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
          ],
        ],
      ),
    );
  }
}

/// Body of a loaded legal document: subtitle, English-notice banner,
/// markdown body, last-updated footer. Pulled out so the FutureBuilder
/// can remain a thin async switch and the layout is easy to test in
/// isolation (Soleil v2 typography lives here).
class _LegalDocumentBody extends StatelessWidget {
  const _LegalDocumentBody({
    required this.subtitle,
    required this.markdown,
    required this.englishNotice,
    required this.lastUpdatedLabel,
  });

  final String subtitle;
  final String markdown;
  final String englishNotice;
  final String lastUpdatedLabel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            subtitle,
            style: SoleilTextStyles.body.copyWith(
              color: colors?.mutedForeground ??
                  theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 16),
          _EnglishNoticeBanner(notice: englishNotice),
          const SizedBox(height: 16),
          MarkdownBody(
            data: markdown,
            selectable: true,
            styleSheet: _soleilMarkdownStyle(context),
          ),
          const SizedBox(height: 24),
          Text(
            lastUpdatedLabel,
            style: SoleilTextStyles.caption.copyWith(
              color: colors?.mutedForeground ??
                  theme.colorScheme.onSurfaceVariant,
              fontStyle: FontStyle.italic,
            ),
          ),
        ],
      ),
    );
  }
}

/// Corail-soft banner mirroring the web `englishNotice` block — single
/// sentence, no CTA. Visual parity with the web /legal/* pages.
class _EnglishNoticeBanner extends StatelessWidget {
  const _EnglishNoticeBanner({required this.notice});

  final String notice;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: colors?.accentSoft ?? theme.colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Text(
        notice,
        style: SoleilTextStyles.caption.copyWith(
          color: colors?.primaryDeep ?? theme.colorScheme.primary,
        ),
      ),
    );
  }
}

/// Builds the [MarkdownStyleSheet] applying Soleil v2 typography
/// (Fraunces for headings, Inter Tight for body, corail for links and
/// strong emphasis). Extracted so widget tests can assert against a
/// known style surface.
MarkdownStyleSheet _soleilMarkdownStyle(BuildContext context) {
  final theme = Theme.of(context);
  final colors = theme.extension<AppColors>();
  final fg = theme.colorScheme.onSurface;
  final mute =
      colors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;
  final accent = theme.colorScheme.primary;
  final accentSoft =
      colors?.accentSoft ?? theme.colorScheme.primaryContainer;
  return MarkdownStyleSheet(
    h1: SoleilTextStyles.displayM.copyWith(color: fg),
    h2: SoleilTextStyles.headlineLarge.copyWith(color: fg),
    h3: SoleilTextStyles.headlineMedium.copyWith(color: fg),
    h4: SoleilTextStyles.titleLarge.copyWith(color: fg),
    h5: SoleilTextStyles.titleMedium.copyWith(color: fg),
    h6: SoleilTextStyles.titleMedium.copyWith(color: fg),
    p: SoleilTextStyles.bodyLarge.copyWith(color: fg),
    blockquote: SoleilTextStyles.body.copyWith(
      color: mute,
      fontStyle: FontStyle.italic,
    ),
    blockquoteDecoration: BoxDecoration(
      color: accentSoft,
      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
    ),
    blockquotePadding: const EdgeInsets.all(12),
    listBullet: SoleilTextStyles.bodyLarge.copyWith(color: fg),
    strong: SoleilTextStyles.bodyEmphasis.copyWith(color: fg),
    em: SoleilTextStyles.bodyLarge.copyWith(
      color: fg,
      fontStyle: FontStyle.italic,
    ),
    a: SoleilTextStyles.bodyLarge.copyWith(
      color: accent,
      decoration: TextDecoration.underline,
    ),
    code: SoleilTextStyles.mono.copyWith(color: fg),
    codeblockDecoration: BoxDecoration(
      color: accentSoft,
      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
    ),
    codeblockPadding: const EdgeInsets.all(12),
    h1Padding: const EdgeInsets.only(top: 16, bottom: 8),
    h2Padding: const EdgeInsets.only(top: 16, bottom: 8),
    h3Padding: const EdgeInsets.only(top: 12, bottom: 6),
    h4Padding: const EdgeInsets.only(top: 12, bottom: 6),
    blockSpacing: 12,
  );
}
