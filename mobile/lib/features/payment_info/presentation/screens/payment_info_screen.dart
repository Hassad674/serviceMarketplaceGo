import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/storage/secure_storage.dart';
import '../../../../core/theme/app_theme.dart';
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
///
/// Soleil v2 (M-W-05): editorial header + ivoire/corail tokens. The WebView
/// shell stays untouched — controller, navigation handlers and postMessage
/// flow are identical to before the visual port.
class PaymentInfoScreen extends ConsumerWidget {
  const PaymentInfoScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final statusAsync = ref.watch(_accountStatusProvider);

    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      appBar: AppBar(
        title: Text(
          l10n.drawerPaymentInfo,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        backgroundColor: theme.colorScheme.surfaceContainerLowest,
        elevation: 0,
        scrolledUnderElevation: 0,
      ),
      body: RefreshIndicator(
        color: theme.colorScheme.primary,
        backgroundColor: theme.colorScheme.surfaceContainerLowest,
        onRefresh: () async => ref.invalidate(_accountStatusProvider),
        child: ListView(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 24),
          children: [
            _EditorialHeader(l10n: l10n),
            const SizedBox(height: 24),
            statusAsync.when(
              loading: () => Center(
                child: Padding(
                  padding: const EdgeInsets.all(48),
                  child: CircularProgressIndicator(
                    color: theme.colorScheme.primary,
                  ),
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
    final hasAccount = status != null;
    final fullyActive = status?.fullyActive ?? false;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _StatusCard(status: status),
        const SizedBox(height: 16),
        _WebViewHintCard(
          message: l10n.kycW05OpenWebViewHint,
        ),
        const SizedBox(height: 20),
        _PrimaryActionButton(
          fullyActive: fullyActive,
          hasAccount: hasAccount,
          label: fullyActive
              ? l10n.paymentInfoEdit
              : hasAccount
                  ? l10n.paymentInfoComplete
                  : l10n.paymentInfoSetup,
          onPressed: () => _openWebView(context, ref),
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
// Editorial header — Soleil eyebrow + Fraunces italic accent
// ---------------------------------------------------------------------------

class _EditorialHeader extends StatelessWidget {
  final AppLocalizations l10n;

  const _EditorialHeader({required this.l10n});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final onSurface = theme.colorScheme.onSurface;
    final mute = theme.colorScheme.onSurfaceVariant;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.kycW05Eyebrow,
          style: SoleilTextStyles.mono.copyWith(
            color: primary,
            fontWeight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 8),
        RichText(
          text: TextSpan(
            style: SoleilTextStyles.headlineLarge.copyWith(color: onSurface),
            children: [
              TextSpan(text: '${l10n.kycW05TitlePart1} '),
              TextSpan(
                text: l10n.kycW05TitleAccent,
                style: SoleilTextStyles.headlineLarge.copyWith(
                  fontStyle: FontStyle.italic,
                  color: primary,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 10),
        Text(
          l10n.kycW05Subtitle,
          style: SoleilTextStyles.body.copyWith(color: mute, fontSize: 13.5),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Status card — Soleil tinted band, no flashy gradient
// ---------------------------------------------------------------------------

class _StatusCard extends StatelessWidget {
  final _AccountStatus? status;

  const _StatusCard({required this.status});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    final hasAccount = status != null;
    final fullyActive = status?.fullyActive ?? false;

    final Color tintBg;
    final Color tintFg;
    final IconData icon;
    final String title;
    final String subtitle;

    if (fullyActive) {
      tintBg = colors.successSoft;
      tintFg = colors.success;
      icon = Icons.check_circle_rounded;
      title = l10n.paymentInfoActive;
      subtitle = l10n.paymentInfoActiveDesc;
    } else if (hasAccount) {
      tintBg = colors.amberSoft;
      tintFg = colors.warning;
      icon = Icons.schedule_rounded;
      title = l10n.paymentInfoPending;
      subtitle = l10n.paymentInfoPendingDesc(status!.requirementsCount);
    } else {
      tintBg = colors.accentSoft;
      tintFg = theme.colorScheme.primary;
      icon = Icons.credit_card_off_rounded;
      title = l10n.paymentInfoNotConfigured;
      subtitle = l10n.paymentInfoNotConfiguredDesc;
    }

    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: colors.border),
        boxShadow: AppTheme.cardShadow,
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Tinted header band
          Container(
            color: tintBg,
            padding: const EdgeInsets.fromLTRB(18, 18, 18, 18),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: theme.colorScheme.surfaceContainerLowest,
                    borderRadius: BorderRadius.circular(14),
                    boxShadow: AppTheme.cardShadow,
                  ),
                  child: Icon(icon, color: tintFg, size: 22),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        style: SoleilTextStyles.titleLarge.copyWith(
                          color: theme.colorScheme.onSurface,
                          fontSize: 18,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        subtitle,
                        style: SoleilTextStyles.body.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                          fontSize: 13,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          if (hasAccount)
            Padding(
              padding: const EdgeInsets.fromLTRB(18, 14, 18, 16),
              child: Row(
                children: [
                  Expanded(
                    child: _CapabilityRow(
                      label: l10n.paymentInfoCharges,
                      enabled: status!.chargesEnabled,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: _CapabilityRow(
                      label: l10n.paymentInfoPayouts,
                      enabled: status!.payoutsEnabled,
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

class _CapabilityRow extends StatelessWidget {
  final String label;
  final bool enabled;

  const _CapabilityRow({required this.label, required this.enabled});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    final tone = enabled ? colors.success : colors.warning;
    final toneSoft = enabled ? colors.successSoft : colors.amberSoft;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: toneSoft,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        border: Border.all(color: tone.withValues(alpha: 0.25)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: tone,
            ),
          ),
          const SizedBox(width: 8),
          Flexible(
            child: Text(
              label,
              overflow: TextOverflow.ellipsis,
              style: SoleilTextStyles.caption.copyWith(
                color: tone,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Hint card — Soleil ivory surface, sable border, mono caption
// ---------------------------------------------------------------------------

class _WebViewHintCard extends StatelessWidget {
  final String message;

  const _WebViewHintCard({required this.message});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: colors.border),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.lock_outline_rounded,
            size: 18,
            color: theme.colorScheme.primary,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              style: SoleilTextStyles.body.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                fontSize: 13,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Primary action — Soleil pill (corail filled)
// ---------------------------------------------------------------------------

class _PrimaryActionButton extends StatelessWidget {
  final bool fullyActive;
  final bool hasAccount;
  final String label;
  final VoidCallback onPressed;

  const _PrimaryActionButton({
    required this.fullyActive,
    required this.hasAccount,
    required this.label,
    required this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return SizedBox(
      width: double.infinity,
      height: 52,
      child: FilledButton.icon(
        onPressed: onPressed,
        icon: Icon(
          fullyActive ? Icons.edit_outlined : Icons.arrow_forward_rounded,
          size: 18,
        ),
        label: Text(
          label,
          style: SoleilTextStyles.button.copyWith(fontSize: 15),
        ),
        style: FilledButton.styleFrom(
          backgroundColor: theme.colorScheme.primary,
          foregroundColor: theme.colorScheme.onPrimary,
          shape: const StadiumBorder(),
          elevation: 0,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// WebView screen (full-screen modal) — Soleil chrome, untouched controller
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
    final theme = Theme.of(context);

    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surfaceContainerLowest,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: Text(
          title,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        leading: IconButton(
          icon: Icon(
            Icons.close_rounded,
            color: theme.colorScheme.onSurface,
          ),
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
