import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/job_entity.dart';
import '../../providers/job_provider.dart';

/// Popup menu rendered in the job detail app bar — close, reopen,
/// delete actions.
class JobDetailPopupMenu extends ConsumerWidget {
  const JobDetailPopupMenu({
    super.key,
    required this.job,
    required this.jobId,
    required this.canEdit,
    required this.canDelete,
    required this.onRefresh,
  });

  final JobEntity job;
  final String jobId;
  final bool canEdit;
  final bool canDelete;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return PopupMenuButton<String>(
      onSelected: (value) => _onSelected(context, ref, value),
      itemBuilder: (context) => [
        if (canEdit) ...[
          if (job.isOpen)
            PopupMenuItem(
              value: 'close',
              child: Row(
                children: [
                  const Icon(Icons.block, size: 18),
                  const SizedBox(width: 8),
                  Text(l10n.jobClose),
                ],
              ),
            )
          else
            PopupMenuItem(
              value: 'reopen',
              child: Row(
                children: [
                  const Icon(Icons.refresh, size: 18),
                  const SizedBox(width: 8),
                  Text(l10n.jobReopen),
                ],
              ),
            ),
        ],
        if (canDelete)
          PopupMenuItem(
            value: 'delete',
            child: Row(
              children: [
                Icon(
                  Icons.delete_outline,
                  size: 18,
                  color: theme.colorScheme.error,
                ),
                const SizedBox(width: 8),
                Text(
                  l10n.jobDelete,
                  style: TextStyle(color: theme.colorScheme.error),
                ),
              ],
            ),
          ),
      ],
    );
  }

  Future<void> _onSelected(
    BuildContext context,
    WidgetRef ref,
    String value,
  ) async {
    final l10n = AppLocalizations.of(context)!;

    switch (value) {
      case 'close':
        final ok = await closeJobAction(ref, jobId);
        if (!context.mounted) return;
        if (ok) {
          onRefresh();
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.unexpectedError)),
          );
        }
      case 'reopen':
        final ok = await reopenJobAction(ref, jobId);
        if (!context.mounted) return;
        if (ok) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.jobReopenSuccess)),
          );
          onRefresh();
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.unexpectedError)),
          );
        }
      case 'delete':
        await _confirmDelete(context, ref);
    }
  }

  Future<void> _confirmDelete(BuildContext context, WidgetRef ref) async {
    final l10n = AppLocalizations.of(context)!;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.jobDelete),
        content: Text(l10n.jobDeleteConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.jobCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            child: Text(l10n.jobDelete),
          ),
        ],
      ),
    );

    if (confirmed != true || !context.mounted) return;

    final ok = await deleteJobAction(ref, jobId);
    if (!context.mounted) return;

    if (ok) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.jobDeleteSuccess)),
      );
      context.pop();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.unexpectedError)),
      );
    }
  }
}
