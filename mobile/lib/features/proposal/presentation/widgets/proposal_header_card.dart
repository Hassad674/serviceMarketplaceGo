import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/proposal.dart';

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
          const Color(0xFFFEF3C7),
          const Color(0xFF92400E),
        ),
      ProposalStatus.accepted => (
          l10n.proposalAccepted,
          const Color(0xFFDCFCE7),
          const Color(0xFF166534),
        ),
      ProposalStatus.declined => (
          l10n.proposalDeclined,
          const Color(0xFFFEE2E2),
          const Color(0xFF991B1B),
        ),
      ProposalStatus.withdrawn => (
          l10n.proposalWithdrawn,
          const Color(0xFFF1F5F9),
          const Color(0xFF475569),
        ),
      ProposalStatus.paid || ProposalStatus.active => (
          l10n.projectStatusActive,
          const Color(0xFFDCFCE7),
          const Color(0xFF166534),
        ),
      ProposalStatus.completionRequested => (
          l10n.proposalCompletionRequestedMessage,
          const Color(0xFFFEF3C7),
          const Color(0xFF92400E),
        ),
      ProposalStatus.completed => (
          l10n.projectStatusCompleted,
          const Color(0xFFE0F2FE),
          const Color(0xFF075985),
        ),
    };
  }
}
