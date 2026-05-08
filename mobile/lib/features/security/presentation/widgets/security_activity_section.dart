import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/security_event.dart';
import '../providers/security_providers.dart';

/// SecurityActivitySection — Soleil v2 inline list of recent
/// authentication events for the current user, embedded inside the
/// account screen.
///
/// The section follows the same cursor-pagination pattern as the
/// receipts tab: a list of fetched cursors, one FutureProvider per
/// page, a "Voir plus" pill that appends the next cursor.
///
/// Empty / error states are first-class — every async surface in the
/// app ships with both, this one is no exception.
class SecurityActivitySection extends ConsumerStatefulWidget {
  const SecurityActivitySection({super.key});

  @override
  ConsumerState<SecurityActivitySection> createState() =>
      _SecurityActivitySectionState();
}

class _SecurityActivitySectionState
    extends ConsumerState<SecurityActivitySection> {
  // List of cursors fetched so far. The first entry is `null` (initial
  // page); each successful page push appends its `nextCursor` (when
  // non-null) so subsequent watches resolve all cached pages.
  final List<String?> _cursors = [null];

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final pages = _cursors
        .map((c) => ref.watch(securityActivityProvider(c)))
        .toList(growable: false);

    final firstLoading = pages.first.isLoading && !pages.first.hasValue;
    if (firstLoading) {
      return const _SecurityLoading();
    }
    final allLoaded = pages.every((p) => p.hasValue);
    final anyError = pages.any((p) => p.hasError);
    if (anyError && !allLoaded) {
      return _SecurityErrorState(
        message: l10n.accountSecurityError,
        retryLabel: l10n.accountSecurityRetry,
        onRetry: () {
          for (final cursor in _cursors) {
            ref.invalidate(securityActivityProvider(cursor));
          }
        },
      );
    }

    final events = pages
        .expand((p) => p.requireValue.data)
        .toList(growable: false);
    if (events.isEmpty) {
      return _SecurityEmptyState(label: l10n.accountSecurityEmpty);
    }

    final latestPage = pages.last;
    final nextCursor = latestPage.value?.nextCursor;
    final fetchingNext =
        latestPage.isLoading && _cursors.length > 1 && !latestPage.hasValue;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final event in events) _SecurityEventRow(event: event),
        if (nextCursor != null && nextCursor.isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(top: 12),
            child: Center(
              child: OutlinedButton(
                onPressed: fetchingNext
                    ? null
                    : () => setState(() => _cursors.add(nextCursor)),
                child: Text(
                  fetchingNext
                      ? l10n.accountSecurityLoadingMore
                      : l10n.accountSecurityLoadMore,
                ),
              ),
            ),
          ),
      ],
    );
  }
}

class _SecurityLoading extends StatelessWidget {
  const _SecurityLoading();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 24),
      child: Center(
        child: CircularProgressIndicator(color: theme.colorScheme.primary),
      ),
    );
  }
}

class _SecurityEmptyState extends StatelessWidget {
  const _SecurityEmptyState({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 16),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: theme.dividerColor),
        color: theme.colorScheme.surface,
      ),
      child: Center(
        child: Text(
          label,
          textAlign: TextAlign.center,
          style: SoleilTextStyles.body.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ),
    );
  }
}

class _SecurityErrorState extends StatelessWidget {
  const _SecurityErrorState({
    required this.message,
    required this.retryLabel,
    required this.onRetry,
  });

  final String message;
  final String retryLabel;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: theme.colorScheme.error.withValues(alpha: 0.4)),
        color: theme.colorScheme.errorContainer.withValues(alpha: 0.2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            message,
            style: SoleilTextStyles.body.copyWith(
              color: theme.colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 12),
          Align(
            alignment: Alignment.centerLeft,
            child: OutlinedButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh, size: 18),
              label: Text(retryLabel),
            ),
          ),
        ],
      ),
    );
  }
}

class _SecurityEventRow extends StatelessWidget {
  const _SecurityEventRow({required this.event});

  final SecurityEvent event;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final colors = theme.extension<AppColors>();

    final deviceLabel = event.userAgentSummary.isNotEmpty
        ? event.userAgentSummary
        : l10n.accountSecurityUnknownDevice;
    final actionLabel = _localisedActionFor(l10n, event.action);
    final dateLabel = _formatDateTime(event.createdAt);

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
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
              _iconFor(event.accessKind),
              size: 18,
              color: theme.colorScheme.primary,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  deviceLabel,
                  style: SoleilTextStyles.body.copyWith(
                    fontWeight: FontWeight.w600,
                    color: theme.colorScheme.onSurface,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  _detailsFor(event, actionLabel),
                  style: SoleilTextStyles.caption.copyWith(
                    color: colors?.mutedForeground ??
                        theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 12),
          Text(
            dateLabel,
            style: SoleilTextStyles.caption.copyWith(
              color: colors?.mutedForeground ??
                  theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}

IconData _iconFor(SecurityAccessKind kind) {
  switch (kind) {
    case SecurityAccessKind.desktop:
      return Icons.desktop_windows_outlined;
    case SecurityAccessKind.mobile:
      return Icons.smartphone_outlined;
    case SecurityAccessKind.tablet:
      return Icons.tablet_mac_outlined;
    case SecurityAccessKind.unknown:
      return Icons.help_outline;
  }
}

String _localisedActionFor(AppLocalizations l10n, String action) {
  switch (action) {
    case 'auth.login_success':
      return l10n.accountSecurityActionLoginSuccess;
    case 'auth.logout':
      return l10n.accountSecurityActionLogout;
    case 'auth.token_refresh':
      return l10n.accountSecurityActionTokenRefresh;
    case 'auth.password_reset_request':
      return l10n.accountSecurityActionPasswordResetRequest;
    case 'auth.password_reset_complete':
      return l10n.accountSecurityActionPasswordResetComplete;
    default:
      return l10n.accountSecurityActionUnknown;
  }
}

String _detailsFor(SecurityEvent event, String actionLabel) {
  final parts = <String>[actionLabel];
  if (event.ipAddress != null && event.ipAddress!.isNotEmpty) {
    parts.add(event.ipAddress!);
  }
  if (event.countryHint != null && event.countryHint!.isNotEmpty) {
    parts.add(event.countryHint!);
  }
  return parts.join(' · ');
}

String _formatDateTime(DateTime dt) {
  // Local-time formatting; the backend stores UTC and we render in the
  // device's timezone so "il y a 2h" maps to the user's wall clock.
  final local = dt.toLocal();
  return DateFormat('d MMM, HH:mm').format(local);
}
