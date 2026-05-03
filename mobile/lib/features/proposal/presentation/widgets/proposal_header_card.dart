import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/proposal.dart';
import '../../../../core/theme/app_palette.dart';

/// Big icon + title + status pill block sitting at the top of the
/// proposal detail screen.
class ProposalHeaderCard extends StatelessWidget {
  const ProposalHeaderCard({
    super.key,
    required this.title,
    required this.status,
    required this.version,
  });

  final String title;
  final ProposalStatus status;
  final int version;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final (label, bgColor, fgColor) = _statusStyle(status, l10n);

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 44,
          height: 44,
          decoration: BoxDecoration(
            color: theme.colorScheme.primary.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
          child: Icon(
            Icons.description_outlined,
            color: theme.colorScheme.primary,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 6),
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 4,
                ),
                decoration: BoxDecoration(
                  color: bgColor,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  label,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: fgColor,
                  ),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  (String, Color, Color) _statusStyle(
    ProposalStatus status,
    AppLocalizations l10n,
  ) {
    return switch (status) {
      ProposalStatus.pending => (
          l10n.proposalPending,
          AppPalette.amber100,
          AppPalette.amber800,
        ),
      ProposalStatus.accepted => (
          l10n.proposalAccepted,
          AppPalette.green100,
          AppPalette.green800,
        ),
      ProposalStatus.declined => (
          l10n.proposalDeclined,
          AppPalette.red100,
          AppPalette.red800,
        ),
      ProposalStatus.withdrawn => (
          l10n.proposalWithdrawn,
          AppPalette.slate100,
          AppPalette.slate600,
        ),
      ProposalStatus.paid || ProposalStatus.active => (
          l10n.projectStatusActive,
          AppPalette.green100,
          AppPalette.green800,
        ),
      ProposalStatus.completionRequested => (
          l10n.proposalCompletionRequestedMessage,
          AppPalette.amber100,
          AppPalette.amber800,
        ),
      ProposalStatus.completed => (
          l10n.projectStatusCompleted,
          AppPalette.sky100,
          AppPalette.sky800,
        ),
    };
  }
}
