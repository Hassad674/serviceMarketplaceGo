import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/proposal.dart';

/// Soleil v2 — Header card for the proposal detail screen.
/// Big icon plate + Fraunces title + Soleil status pill, plus an
/// optional one-line participants caption (Client / Prestataire) when
/// the backend has surfaced both display names.
class ProposalHeaderCard extends StatelessWidget {
  const ProposalHeaderCard({
    super.key,
    required this.title,
    required this.status,
    required this.version,
    this.clientName,
    this.providerName,
  });

  final String title;
  final ProposalStatus status;
  final int version;
  // Backend-resolved participant names (`client_name`/`provider_name`
  // on `ProposalResponse`). Optional because older messages and
  // legacy fixtures may omit them; rendering is suppressed when both
  // are null/empty so we never display `User <id>` placeholders.
  final String? clientName;
  final String? providerName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final (label, bgColor, fgColor) = _statusStyle(status, l10n, theme, appColors);
    final hasClientName = (clientName ?? '').trim().isNotEmpty;
    final hasProviderName = (providerName ?? '').trim().isNotEmpty;
    final showParticipants = hasClientName || hasProviderName;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: theme.colorScheme.primaryContainer,
              borderRadius: BorderRadius.circular(AppTheme.radiusLg),
            ),
            child: Icon(
              Icons.description_outlined,
              color: theme.colorScheme.primary,
              size: 24,
            ),
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
                  ),
                ),
                if (version > 1) ...[
                  const SizedBox(height: 4),
                  Text(
                    'v$version',
                    style: SoleilTextStyles.mono.copyWith(
                      color: appColors?.subtleForeground ??
                          theme.colorScheme.onSurfaceVariant,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ],
                const SizedBox(height: 8),
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: bgColor,
                    borderRadius: BorderRadius.circular(AppTheme.radiusFull),
                  ),
                  child: Text(
                    label,
                    style: SoleilTextStyles.mono.copyWith(
                      color: fgColor,
                      fontSize: 10.5,
                      fontWeight: FontWeight.w700,
                      letterSpacing: 0.6,
                    ),
                  ),
                ),
                if (showParticipants) ...[
                  const SizedBox(height: 12),
                  _ParticipantLine(
                    label: l10n.proposalClient,
                    name: hasClientName ? clientName! : '—',
                    appColors: appColors,
                    theme: theme,
                  ),
                  const SizedBox(height: 4),
                  _ParticipantLine(
                    label: l10n.proposalProvider,
                    name: hasProviderName ? providerName! : '—',
                    appColors: appColors,
                    theme: theme,
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }

  (String, Color, Color) _statusStyle(
    ProposalStatus status,
    AppLocalizations l10n,
    ThemeData theme,
    AppColors? appColors,
  ) {
    final corail = theme.colorScheme.primary;
    final corailSoft = theme.colorScheme.primaryContainer;
    final sapin = appColors?.success ?? corail;
    final sapinSoft = appColors?.successSoft ?? corailSoft;
    final ambre = appColors?.warning ?? corail;
    final ambreSoft = appColors?.amberSoft ?? corailSoft;
    final muted = theme.colorScheme.onSurfaceVariant;
    final mutedSoft = theme.colorScheme.outline.withValues(alpha: 0.2);

    return switch (status) {
      ProposalStatus.pending => (l10n.proposalPending, ambreSoft, ambre),
      ProposalStatus.accepted => (l10n.proposalAccepted, sapinSoft, sapin),
      ProposalStatus.declined => (
          l10n.proposalDeclined,
          theme.colorScheme.error.withValues(alpha: 0.1),
          theme.colorScheme.error,
        ),
      ProposalStatus.withdrawn => (l10n.proposalWithdrawn, mutedSoft, muted),
      ProposalStatus.paid ||
      ProposalStatus.active =>
        (l10n.projectStatusActive, sapinSoft, sapin),
      ProposalStatus.completionRequested => (
          l10n.proposalCompletionRequestedMessage,
          ambreSoft,
          ambre,
        ),
      ProposalStatus.completed => (l10n.projectStatusCompleted, corailSoft, corail),
    };
  }
}

/// One row of the participants caption: a tabac role label followed by
/// the encre display name. Sized to nest tightly inside the header
/// card column without dwarfing the title.
class _ParticipantLine extends StatelessWidget {
  const _ParticipantLine({
    required this.label,
    required this.name,
    required this.appColors,
    required this.theme,
  });

  final String label;
  final String name;
  final AppColors? appColors;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    final mutedColor = appColors?.subtleForeground ??
        theme.colorScheme.onSurfaceVariant;
    return Row(
      children: [
        Text(
          label,
          style: SoleilTextStyles.mono.copyWith(
            color: mutedColor,
            fontSize: 10.5,
            fontWeight: FontWeight.w700,
            letterSpacing: 0.6,
          ),
        ),
        const SizedBox(width: 6),
        Expanded(
          child: Text(
            name,
            overflow: TextOverflow.ellipsis,
            style: SoleilTextStyles.body.copyWith(
              color: theme.colorScheme.onSurface,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
      ],
    );
  }
}
