import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../../proposal/types/proposal.dart';
import '../../../../../core/theme/app_palette.dart';

/// Renders a proposal message as a rich Material card inside the chat.
///
/// Displayed when `message.type` is `proposal_sent` or `proposal_modified`.
/// Extracts metadata from `message.metadata` via [ProposalMessageMetadata].
///
/// Action buttons are conditionally shown based on proposal status,
/// ownership, and version (old versions are greyed out).
class ProposalCard extends StatelessWidget {
  const ProposalCard({
    super.key,
    required this.metadata,
    required this.isOwn,
    required this.currentUserId,
    this.isLatestVersion = true,
    this.onAccept,
    this.onDecline,
    this.onModify,
    this.onPay,
    this.onTap,
  });

  final ProposalMessageMetadata metadata;
  final bool isOwn;
  final String currentUserId;
  final bool isLatestVersion;
  final VoidCallback? onAccept;
  final VoidCallback? onDecline;
  final VoidCallback? onModify;
  final VoidCallback? onPay;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final isOldVersion = !isLatestVersion;

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Align(
        alignment: isOwn ? Alignment.centerRight : Alignment.centerLeft,
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxWidth: MediaQuery.sizeOf(context).width * 0.8,
          ),
          child: Opacity(
            opacity: isOldVersion ? 0.5 : 1.0,
            child: GestureDetector(
              onTap: onTap,
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
                    // "View details" link
                    if (onTap != null && !isOldVersion) ...[
                      Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 14,
                          vertical: 6,
                        ),
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.end,
                          children: [
                            Text(
                              l10n.proposalViewDetails,
                              style: TextStyle(
                                fontSize: 12,
                                fontWeight: FontWeight.w500,
                                color: theme.colorScheme.primary,
                              ),
                            ),
                            const SizedBox(width: 4),
                            Icon(
                              Icons.chevron_right,
                              size: 16,
                              color: theme.colorScheme.primary,
                            ),
                          ],
                        ),
                      ),
                    ],
                    if (_shouldShowActions && !isOldVersion) ...[
                      const Divider(height: 1),
                      _buildActions(theme, l10n),
                    ],
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  /// Determine whether to show action buttons.
  bool get _shouldShowActions {
    if (metadata.status == ProposalStatus.pending && !isOwn) {
      return true; // recipient can accept/decline/modify
    }
    // "Pay now" is shown to the client (payer), not just !isOwn.
    if (metadata.status == ProposalStatus.accepted) {
      final isClient = metadata.clientId == currentUserId;
      return isClient;
    }
    return false;
  }

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

  Widget _buildBody(
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
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
                '\u20AC ${metadata.amount.toStringAsFixed(2)}',
                style: theme.textTheme.titleMedium?.copyWith(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),

          // Deadline row (optional)
          if (metadata.deadline != null) ...[
            const SizedBox(height: 10),
            Row(
              children: [
                Icon(
                  Icons.calendar_today_outlined,
                  size: 18,
                  color: appColors?.mutedForeground,
                ),
                const SizedBox(width: 8),
                Text(l10n.proposalDeadline,
                    style: theme.textTheme.bodySmall),
                const Spacer(),
                Text(
                  metadata.deadline!,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ),
          ],

          // Documents count (optional)
          if (metadata.documentsCount > 0) ...[
            const SizedBox(height: 10),
            Row(
              children: [
                Icon(
                  Icons.attach_file,
                  size: 18,
                  color: appColors?.mutedForeground,
                ),
                const SizedBox(width: 8),
                Text(
                  '${metadata.documentsCount} documents',
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ),
          ],

          // Version indicator
          if (metadata.version > 1) ...[
            const SizedBox(height: 10),
            Row(
              children: [
                Icon(
                  Icons.history,
                  size: 18,
                  color: appColors?.mutedForeground,
                ),
                const SizedBox(width: 8),
                Text(
                  'v${metadata.version}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w500,
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

  Widget _buildActions(ThemeData theme, AppLocalizations l10n) {
    final appColors = theme.extension<AppColors>();

    // Accepted state: show "Pay now" button for the client (recipient).
    if (metadata.status == ProposalStatus.accepted) {
      return Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        child: SizedBox(
          width: double.infinity,
          child: ElevatedButton.icon(
            onPressed: onPay,
            icon: const Icon(Icons.payment_outlined, size: 18),
            label: Text(l10n.payNow),
            style: ElevatedButton.styleFrom(
              backgroundColor: theme.colorScheme.primary,
              foregroundColor: Colors.white,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(AppTheme.radiusSm),
              ),
              minimumSize: const Size(0, 38),
              elevation: 0,
            ),
          ),
        ),
      );
    }

    // Pending state: accept / decline / modify buttons.
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      child: Column(
        children: [
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: onDecline,
                  style: OutlinedButton.styleFrom(
                    foregroundColor: theme.colorScheme.error,
                    side: BorderSide(
                      color:
                          theme.colorScheme.error.withValues(alpha: 0.3),
                    ),
                    shape: RoundedRectangleBorder(
                      borderRadius:
                          BorderRadius.circular(AppTheme.radiusSm),
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
                    backgroundColor:
                        appColors?.success ?? Colors.green,
                    foregroundColor: Colors.white,
                    shape: RoundedRectangleBorder(
                      borderRadius:
                          BorderRadius.circular(AppTheme.radiusSm),
                    ),
                    minimumSize: const Size(0, 38),
                    elevation: 0,
                  ),
                  child: Text(l10n.proposalAccept),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          SizedBox(
            width: double.infinity,
            child: OutlinedButton.icon(
              onPressed: onModify,
              icon: const Icon(Icons.edit_outlined, size: 16),
              label: Text(l10n.proposalModify),
              style: OutlinedButton.styleFrom(
                shape: RoundedRectangleBorder(
                  borderRadius:
                      BorderRadius.circular(AppTheme.radiusSm),
                ),
                minimumSize: const Size(0, 36),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});

  final ProposalStatus status;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    final (label, bgColor, fgColor) = switch (status) {
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
