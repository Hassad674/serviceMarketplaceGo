import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';

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
///   2. Adresse email — read-only display of the authenticated email.
///      The "change email" flow is also a future feature, hence the
///      "Bientôt disponible" pill on the new-email field, mirroring
///      the web counterpart.
///   3. Mot de passe — placeholder, same rationale as web.
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
                child: _CurrentEmailRow(
                  label: l10n.accountCurrentEmail,
                  email: email,
                  comingSoon: l10n.accountComingSoon,
                ),
              ),
              const SizedBox(height: 16),
              _AccountSection(
                icon: Icons.lock_outline,
                title: l10n.accountSectionPassword,
                description: l10n.accountSectionPasswordDesc,
                child: _ComingSoonPill(label: l10n.accountComingSoon),
              ),
              const SizedBox(height: 16),
              _AccountSection(
                icon: Icons.shield_outlined,
                title: l10n.accountSectionDataAndDeletion,
                description: l10n.accountSectionDataAndDeletionDesc,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
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

/// Read-only display of the current email + "coming soon" pill for the
/// not-yet-implemented change-email flow. Mirrors the web layout:
/// labelled value, then a pill below.
class _CurrentEmailRow extends StatelessWidget {
  const _CurrentEmailRow({
    required this.label,
    required this.email,
    required this.comingSoon,
  });

  final String label;
  final String email;
  final String comingSoon;

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
        _ComingSoonPill(label: comingSoon),
      ],
    );
  }
}
