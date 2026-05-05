import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/proposal_entity.dart';
import '../../domain/repositories/proposal_repository.dart';
import '../providers/proposal_provider.dart';

/// Soleil v2 — Milestone action bottom sheet.
/// Soleil card body, Fraunces title, corail StadiumBorder pill primary
/// button (or destructive outline for reject).
enum MilestoneAction { fund, submit, approve, reject }

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
      backgroundColor: Theme.of(context).colorScheme.surfaceContainerLowest,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
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
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final copy = _copyFor(widget.action, l10n);
    final insetBottom = MediaQuery.of(context).viewInsets.bottom;

    return Padding(
      padding: EdgeInsets.only(bottom: insetBottom),
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Center(
                child: Container(
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                    color: appColors?.borderStrong ??
                        theme.colorScheme.outline.withValues(alpha: 0.4),
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              const SizedBox(height: 22),
              Text(
                copy.title,
                style: SoleilTextStyles.headlineMedium.copyWith(
                  color: theme.colorScheme.onSurface,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                copy.description,
                style: SoleilTextStyles.bodyLarge.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 22),
              _MilestoneSummaryCard(milestone: widget.milestone),
              const SizedBox(height: 24),
              FilledButton(
                onPressed: _isSubmitting ? null : _confirm,
                style: FilledButton.styleFrom(
                  backgroundColor: copy.primaryColor(theme),
                  minimumSize: const Size(0, 52),
                  shape: const StadiumBorder(),
                  textStyle: SoleilTextStyles.button,
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
                style: TextButton.styleFrom(
                  shape: const StadiumBorder(),
                  minimumSize: const Size(0, 48),
                ),
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
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.milestoneSequenceLabel(milestone.sequence),
            style: SoleilTextStyles.mono.copyWith(
              color: appColors?.primaryDeep ?? theme.colorScheme.primary,
              fontSize: 10.5,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.0,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            milestone.title,
            style: SoleilTextStyles.titleMedium.copyWith(
              color: theme.colorScheme.onSurface,
            ),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
          const SizedBox(height: 8),
          Text(
            '€ ${milestone.amountInEuros.toStringAsFixed(2)}',
            style: SoleilTextStyles.monoLarge.copyWith(
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
