import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/proposal_entity.dart';
import '../../domain/repositories/proposal_repository.dart';
import '../providers/proposal_provider.dart';

// The three milestone actions exposed on the mobile detail screen.
// `fund` is wired through but not currently triggered by the detail
// screen — the accepted → paid transition still runs through the
// payment-simulation flow.
enum MilestoneAction { fund, submit, approve, reject }

/// A contextual confirmation sheet for milestone state transitions.
///
/// Renders the action title, a short description drawn from l10n, the
/// milestone summary, and primary / cancel buttons. On success it
/// invalidates [proposalByIdProvider] so the detail screen refetches
/// and pops back. Errors land in a SnackBar; the sheet stays open so
/// the user can retry.
class MilestoneActionBottomSheet extends ConsumerStatefulWidget {
  const MilestoneActionBottomSheet({
    super.key,
    required this.proposalId,
    required this.milestone,
    required this.action,
  });

  final String proposalId;
  final MilestoneEntity milestone;
  final MilestoneAction action;

  static Future<void> show(
    BuildContext context, {
    required String proposalId,
    required MilestoneEntity milestone,
    required MilestoneAction action,
  }) {
    return showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (_) => MilestoneActionBottomSheet(
        proposalId: proposalId,
        milestone: milestone,
        action: action,
      ),
    );
  }

  @override
  ConsumerState<MilestoneActionBottomSheet> createState() =>
      _MilestoneActionBottomSheetState();
}

class _MilestoneActionBottomSheetState
    extends ConsumerState<MilestoneActionBottomSheet> {
  bool _isSubmitting = false;

  Future<void> _confirm() async {
    if (_isSubmitting) return;
    setState(() => _isSubmitting = true);

    final repo = ref.read(proposalRepositoryProvider);
    bool ok;
    try {
      await _runAction(repo, widget.action);
      ok = true;
    } catch (_) {
      ok = false;
    }

    if (!mounted) return;

    if (ok) {
      ref.invalidate(proposalByIdProvider(widget.proposalId));
      Navigator.of(context).pop();
      return;
    }

    setState(() => _isSubmitting = false);
    final l10n = AppLocalizations.of(context)!;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(l10n.milestoneActionFailed)),
    );
  }

  Future<void> _runAction(
    ProposalRepository repo,
    MilestoneAction action,
  ) {
    switch (action) {
      case MilestoneAction.fund:
        return repo.fundMilestone(widget.proposalId, widget.milestone.id);
      case MilestoneAction.submit:
        return repo.submitMilestone(widget.proposalId, widget.milestone.id);
      case MilestoneAction.approve:
        return repo.approveMilestone(widget.proposalId, widget.milestone.id);
      case MilestoneAction.reject:
        return repo.rejectMilestone(widget.proposalId, widget.milestone.id);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final copy = _copyFor(widget.action, l10n);
    final insetBottom = MediaQuery.of(context).viewInsets.bottom;

    return Padding(
      padding: EdgeInsets.only(bottom: insetBottom),
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Drag handle
              Center(
                child: Container(
                  width: 36,
                  height: 4,
                  decoration: BoxDecoration(
                    color: theme.dividerColor,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              const SizedBox(height: 20),
              Text(
                copy.title,
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                copy.description,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.7),
                  height: 1.4,
                ),
              ),
              const SizedBox(height: 20),
              _MilestoneSummaryCard(milestone: widget.milestone),
              const SizedBox(height: 24),
              ElevatedButton(
                onPressed: _isSubmitting ? null : _confirm,
                style: ElevatedButton.styleFrom(
                  backgroundColor: copy.primaryColor(theme),
                  foregroundColor: Colors.white,
                  minimumSize: const Size(0, 48),
                  elevation: 0,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                  ),
                ),
                child: _isSubmitting
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor:
                              AlwaysStoppedAnimation<Color>(Colors.white),
                        ),
                      )
                    : Text(copy.primaryLabel),
              ),
              const SizedBox(height: 8),
              TextButton(
                onPressed: _isSubmitting ? null : () => Navigator.of(context).pop(),
                child: Text(l10n.cancel),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _MilestoneSummaryCard extends StatelessWidget {
  const _MilestoneSummaryCard({required this.milestone});

  final MilestoneEntity milestone;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: appColors?.muted ?? const Color(0xFFF1F5F9),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.milestoneSequenceLabel(milestone.sequence),
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
              fontWeight: FontWeight.w600,
              letterSpacing: 0.4,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            milestone.title,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w700,
            ),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
          const SizedBox(height: 6),
          Text(
            '\u20AC ${milestone.amountInEuros.toStringAsFixed(2)}',
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

class _ActionCopy {
  const _ActionCopy({
    required this.title,
    required this.description,
    required this.primaryLabel,
    required this.isDestructive,
  });

  final String title;
  final String description;
  final String primaryLabel;
  final bool isDestructive;

  Color primaryColor(ThemeData theme) {
    return isDestructive
        ? theme.colorScheme.error
        : theme.colorScheme.primary;
  }
}

_ActionCopy _copyFor(MilestoneAction action, AppLocalizations l10n) {
  switch (action) {
    case MilestoneAction.fund:
      return _ActionCopy(
        title: l10n.milestoneFundTitle,
        description: l10n.milestoneFundDescription,
        primaryLabel: l10n.milestoneFundConfirm,
        isDestructive: false,
      );
    case MilestoneAction.submit:
      return _ActionCopy(
        title: l10n.milestoneSubmitTitle,
        description: l10n.milestoneSubmitDescription,
        primaryLabel: l10n.milestoneSubmitConfirm,
        isDestructive: false,
      );
    case MilestoneAction.approve:
      return _ActionCopy(
        title: l10n.milestoneApproveTitle,
        description: l10n.milestoneApproveDescription,
        primaryLabel: l10n.milestoneApproveConfirm,
        isDestructive: false,
      );
    case MilestoneAction.reject:
      return _ActionCopy(
        title: l10n.milestoneRejectTitle,
        description: l10n.milestoneRejectDescription,
        primaryLabel: l10n.milestoneRejectConfirm,
        isDestructive: true,
      );
  }
}
