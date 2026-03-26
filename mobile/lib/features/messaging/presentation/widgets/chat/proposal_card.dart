import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../../proposal/types/proposal.dart';

/// Renders a proposal message as a rich Material card inside the chat.
///
/// Displayed when `message.type == 'proposal_sent'`. Extracts metadata
/// from `message.metadata` via [ProposalMessageMetadata].
class ProposalCard extends StatelessWidget {
  const ProposalCard({
    super.key,
    required this.metadata,
    required this.isOwn,
    this.onAccept,
    this.onDecline,
  });

  final ProposalMessageMetadata metadata;
  final bool isOwn;
  final VoidCallback? onAccept;
  final VoidCallback? onDecline;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Align(
        alignment: isOwn ? Alignment.centerRight : Alignment.centerLeft,
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxWidth: MediaQuery.sizeOf(context).width * 0.8,
          ),
          child: Container(
            decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(AppTheme.radiusLg),
              border: Border.all(
                color: appColors?.border ?? theme.dividerColor,
              ),
              boxShadow: AppTheme.cardShadow,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                _buildHeader(theme, l10n),
                const Divider(height: 1),
                _buildBody(theme, appColors, l10n),
                if (_showActions) ...[
                  const Divider(height: 1),
                  _buildActions(theme, l10n),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  bool get _showActions =>
      !isOwn && metadata.status == ProposalStatus.pending;

  // -----------------------------------------------------------------------
  // Header: icon + "Proposal from [name]" + status badge
  // -----------------------------------------------------------------------

  Widget _buildHeader(ThemeData theme, AppLocalizations l10n) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      child: Row(
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: theme.colorScheme.primary.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(AppTheme.radiusSm),
            ),
            child: Icon(
              Icons.description_outlined,
              size: 20,
              color: theme.colorScheme.primary,
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '${l10n.proposalFrom} ${metadata.senderName}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w500,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                Text(
                  metadata.title,
                  style: theme.textTheme.titleMedium,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          _StatusBadge(status: metadata.status),
        ],
      ),
    );
  }

  // -----------------------------------------------------------------------
  // Body: amount, payment type, milestones, negotiable
  // -----------------------------------------------------------------------

  Widget _buildBody(
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    final paymentLabel = metadata.paymentType == ProposalPaymentType.escrow
        ? l10n.proposalEscrow
        : l10n.proposalInvoice;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      child: Column(
        children: [
          // Amount row
          Row(
            children: [
              Icon(
                Icons.euro_outlined,
                size: 18,
                color: appColors?.mutedForeground,
              ),
              const SizedBox(width: 8),
              Text(
                l10n.proposalTotalAmount,
                style: theme.textTheme.bodySmall,
              ),
              const Spacer(),
              Text(
                '\u20AC ${metadata.totalAmount.toStringAsFixed(2)}',
                style: theme.textTheme.titleMedium?.copyWith(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
          const SizedBox(height: 10),

          // Payment type row
          Row(
            children: [
              Icon(
                metadata.paymentType == ProposalPaymentType.escrow
                    ? Icons.lock_outline
                    : Icons.receipt_long_outlined,
                size: 18,
                color: appColors?.mutedForeground,
              ),
              const SizedBox(width: 8),
              Text(paymentLabel, style: theme.textTheme.bodySmall),
              if (metadata.paymentType == ProposalPaymentType.escrow) ...[
                const Spacer(),
                Text(
                  '${metadata.milestoneCount} ${l10n.proposalMilestones}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ],
          ),

          // Negotiable badge
          if (metadata.negotiable) ...[
            const SizedBox(height: 10),
            Row(
              children: [
                Icon(
                  Icons.swap_horiz,
                  size: 18,
                  color: appColors?.mutedForeground,
                ),
                const SizedBox(width: 8),
                Text(
                  l10n.proposalNegotiable,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontStyle: FontStyle.italic,
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }

  // -----------------------------------------------------------------------
  // Action buttons: Accept / Decline
  // -----------------------------------------------------------------------

  Widget _buildActions(ThemeData theme, AppLocalizations l10n) {
    final appColors = theme.extension<AppColors>();

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      child: Row(
        children: [
          Expanded(
            child: OutlinedButton(
              onPressed: onDecline,
              style: OutlinedButton.styleFrom(
                foregroundColor: theme.colorScheme.error,
                side: BorderSide(
                  color: theme.colorScheme.error.withValues(alpha: 0.3),
                ),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                ),
                minimumSize: const Size(0, 38),
              ),
              child: Text(l10n.proposalDecline),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: ElevatedButton(
              onPressed: onAccept,
              style: ElevatedButton.styleFrom(
                backgroundColor: appColors?.success ?? Colors.green,
                foregroundColor: Colors.white,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                ),
                minimumSize: const Size(0, 38),
                elevation: 0,
              ),
              child: Text(l10n.proposalAccept),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Status badge
// ---------------------------------------------------------------------------

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});

  final ProposalStatus status;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    final (label, bgColor, fgColor) = switch (status) {
      ProposalStatus.pending => (
          l10n.proposalPending,
          const Color(0xFFFEF3C7), // amber-100
          const Color(0xFF92400E), // amber-800
        ),
      ProposalStatus.accepted => (
          l10n.proposalAccepted,
          const Color(0xFFDCFCE7), // green-100
          const Color(0xFF166534), // green-800
        ),
      ProposalStatus.declined => (
          l10n.proposalDeclined,
          const Color(0xFFFEE2E2), // red-100
          const Color(0xFF991B1B), // red-800
        ),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: fgColor,
        ),
      ),
    );
  }
}
