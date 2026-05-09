import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:path_provider/path_provider.dart';
import 'package:share_plus/share_plus.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../profile_completion/presentation/widgets/profile_completion_bar.dart';
import '../../../security/presentation/widgets/security_activity_section.dart';
import '../../data/gdpr_repository_impl.dart';

/// AccountScreen — Soleil v2 mobile mirror of the web /account page.
///
/// Surfaces the user's preferences as a stack of section cards:
///
///   1. Préférences de notification — placeholder. The mobile data
///      hook for notification preferences does not exist yet (only
///      the data-layer repository is present), so the section is
///      shown but disabled with a "Bientôt disponible" pill. Building
///      the toggle list would require introducing a new Riverpod
///      provider, which is out of scope for this design batch.
///   2. Adresse email — read-only display + CTA routing to
///      [ChangeEmailScreen]. Backend bumps the session version on
///      success, so that flow logs the user out and routes to /login.
///   3. Mot de passe — CTA routing to [ChangePasswordScreen]. Same
///      session-version semantics as the email change.
///   4. Données et suppression — entry points to the existing
///      [DeleteAccountScreen] / [CancelDeletionScreen] flows
///      (already wired backend-side).
///
/// The screen lives inside the [DashboardShell], so the bottom-nav
/// stays visible and "Mon compte" is reached through the lateral
/// drawer rather than the bottom bar (it is not one of the four
/// primary destinations).
class AccountScreen extends ConsumerWidget {
  const AccountScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final authState = ref.watch(authProvider);
    final email = authState.user?['email'] as String? ?? '—';
    // Issue 3 — enterprise users no longer see the completion nudge.
    // The 4-section enterprise checklist is short and the marketplace
    // acquisition incentive does not apply to clients. Decided
    // 2026-05-09 with the avatar-refresh batch.
    final role = authState.user?['role'] as String? ?? '';
    final showCompletionBar = role != 'enterprise';

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.menu),
          onPressed: openShellDrawer,
          tooltip: MaterialLocalizations.of(context).openAppDrawerTooltip,
        ),
        title: Text(l10n.accountTitle),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (showCompletionBar) ...[
                const ProfileCompletionBar(hideWhenComplete: true),
                const SizedBox(height: 16),
              ],
              _AccountSection(
                icon: Icons.notifications_outlined,
                title: l10n.accountSectionNotifications,
                description: l10n.accountSectionNotificationsDesc,
                child: _ComingSoonPill(label: l10n.accountComingSoon),
              ),
              const SizedBox(height: 16),
              _AccountSection(
                icon: Icons.email_outlined,
                title: l10n.accountSectionEmail,
                description: l10n.accountSectionEmailDesc,
                child: _EmailSectionBody(
                  label: l10n.accountCurrentEmail,
                  email: email,
                  changeEmailCta: l10n.accountChangeEmailCta,
                ),
              ),
              const SizedBox(height: 16),
              _AccountSection(
                icon: Icons.lock_outline,
                title: l10n.accountSectionPassword,
                description: l10n.accountSectionPasswordDesc,
                child: Align(
                  alignment: Alignment.centerLeft,
                  child: OutlinedButton.icon(
                    onPressed: () =>
                        context.push(RoutePaths.accountChangePassword),
                    icon: const Icon(Icons.password_outlined, size: 18),
                    label: Text(l10n.accountChangePasswordCta),
                  ),
                ),
              ),
              const SizedBox(height: 16),
              _AccountSection(
                icon: Icons.verified_user_outlined,
                title: l10n.accountSectionSecurity,
                description: l10n.accountSectionSecurityDesc,
                child: const SecurityActivitySection(),
              ),
              const SizedBox(height: 16),
              _AccountSection(
                icon: Icons.shield_outlined,
                title: l10n.accountSectionDataAndDeletion,
                description: l10n.accountSectionDataAndDeletionDesc,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    const _ExportDataButton(),
                    const SizedBox(height: 12),
                    OutlinedButton.icon(
                      onPressed: () => context.push(
                        RoutePaths.accountCancelDeletion,
                      ),
                      icon: const Icon(Icons.restore_outlined, size: 18),
                      label: Text(l10n.accountCancelDeletionAction),
                    ),
                    const SizedBox(height: 12),
                    FilledButton.icon(
                      style: FilledButton.styleFrom(
                        backgroundColor: theme.colorScheme.error,
                      ),
                      onPressed: () =>
                          context.push(RoutePaths.accountDelete),
                      icon: const Icon(Icons.delete_outline, size: 18),
                      label: Text(l10n.accountDeleteAccountAction),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Soleil v2 section card — corail icon square, title, optional helper
/// caption, then the body widget. Mirrors the web `account-settings`
/// shell card without depending on the freelance widget surface.
class _AccountSection extends StatelessWidget {
  const _AccountSection({
    required this.icon,
    required this.title,
    required this.description,
    required this.child,
  });

  final IconData icon;
  final String title;
  final String description;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: colors?.border ?? theme.dividerColor),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                  color: colors?.accentSoft ??
                      theme.colorScheme.primaryContainer,
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
                alignment: Alignment.center,
                child: Icon(
                  icon,
                  size: 18,
                  color: theme.colorScheme.primary,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  title,
                  style: SoleilTextStyles.headlineMedium.copyWith(
                    color: theme.colorScheme.onSurface,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            description,
            style: SoleilTextStyles.body.copyWith(
              color: colors?.mutedForeground ??
                  theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }
}

/// Signature for the platform-specific "share these bytes as a file"
/// hook used by [_ExportDataButton]. Splitting this out lets widget
/// tests stub the share sheet (which depends on `share_plus` +
/// `path_provider` platform channels that throw in unit-test mode)
/// while keeping production wired to the real plugins.
typedef ExportShareSink = Future<void> Function(
  List<int> bytes,
  String filename,
);

Future<void> _defaultExportShareSink(List<int> bytes, String filename) async {
  final dir = await getTemporaryDirectory();
  final file = File('${dir.path}/$filename');
  await file.writeAsBytes(bytes, flush: true);
  await Share.shareXFiles(
    [XFile(file.path, mimeType: 'application/zip', name: filename)],
  );
}

/// Riverpod seam used by [_ExportDataButton]. Production resolves to
/// [_defaultExportShareSink]; widget tests override it with a fake
/// that records the bytes and never touches the file system.
final exportShareSinkProvider = Provider<ExportShareSink>(
  (_) => _defaultExportShareSink,
);

/// Mobile mirror of the web "Télécharger mes données" button. Pulls
/// the export ZIP through [GDPRRepositoryImpl.exportMyData], persists
/// it to the temp dir, then hands it off to the system share sheet
/// (`share_plus`) so the user can save it to Files / iCloud / Drive.
///
/// Stateful only for the local loading flag — the underlying call is
/// idempotent and re-entrancy is prevented via [_isExporting].
class _ExportDataButton extends ConsumerStatefulWidget {
  const _ExportDataButton();

  @override
  ConsumerState<_ExportDataButton> createState() => _ExportDataButtonState();
}

class _ExportDataButtonState extends ConsumerState<_ExportDataButton> {
  bool _isExporting = false;

  Future<void> _onPressed() async {
    if (_isExporting) return;
    setState(() => _isExporting = true);
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    try {
      final repo = ref.read(gdprRepositoryProvider);
      final share = ref.read(exportShareSinkProvider);
      final bytes = await repo.exportMyData();
      final filename =
          'marketplace-export-${DateTime.now().toUtc().millisecondsSinceEpoch}.zip';
      await share(bytes, filename);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.accountExportDataSuccess)),
      );
    } catch (_) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.accountExportDataError)),
      );
    } finally {
      if (mounted) {
        setState(() => _isExporting = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final label = _isExporting
        ? l10n.accountExportDataPreparing
        : l10n.accountExportDataAction;
    return OutlinedButton.icon(
      onPressed: _isExporting ? null : _onPressed,
      icon: _isExporting
          ? const SizedBox(
              width: 16,
              height: 16,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.download_outlined, size: 18),
      label: Text(label),
    );
  }
}

/// Soleil v2 corail-soft pill that signals a feature is wired on the
/// web side but not yet on mobile. Identical to the web `comingSoon`
/// affordance for visual parity.
class _ComingSoonPill extends StatelessWidget {
  const _ComingSoonPill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: colors?.accentSoft ?? theme.colorScheme.primaryContainer,
          borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        ),
        child: Text(
          label,
          style: SoleilTextStyles.caption.copyWith(
            color: colors?.primaryDeep ?? theme.colorScheme.primary,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    );
  }
}

/// Read-only display of the current email + CTA routing to the
/// [ChangeEmailScreen]. Mirrors the web layout: labelled value, then
/// a button below.
class _EmailSectionBody extends StatelessWidget {
  const _EmailSectionBody({
    required this.label,
    required this.email,
    required this.changeEmailCta,
  });

  final String label;
  final String email;
  final String changeEmailCta;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label.toUpperCase(),
          style: SoleilTextStyles.caption.copyWith(
            color: colors?.mutedForeground ??
                theme.colorScheme.onSurfaceVariant,
            fontWeight: FontWeight.w600,
            letterSpacing: 1.2,
          ),
        ),
        const SizedBox(height: 6),
        SelectableText(
          email,
          style: SoleilTextStyles.monoLarge.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        const SizedBox(height: 12),
        Align(
          alignment: Alignment.centerLeft,
          child: OutlinedButton.icon(
            onPressed: () => context.push(RoutePaths.accountChangeEmail),
            icon: const Icon(Icons.alternate_email_outlined, size: 18),
            label: Text(changeEmailCta),
          ),
        ),
      ],
    );
  }
}
