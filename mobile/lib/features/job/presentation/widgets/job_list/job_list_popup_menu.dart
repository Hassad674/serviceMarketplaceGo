import 'package:flutter/material.dart';

import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/job_entity.dart';

enum _JobMenuAction { edit, closeOrReopen, delete }

/// 3-dot popup menu rendered on each job tile in the "My jobs" list.
class JobListPopupMenu extends StatelessWidget {
  const JobListPopupMenu({
    super.key,
    required this.job,
    required this.canEdit,
    required this.canDelete,
    required this.onEdit,
    required this.onClose,
    required this.onReopen,
    required this.onDelete,
  });

  final JobEntity job;
  final bool canEdit;
  final bool canDelete;
  final VoidCallback onEdit;
  final VoidCallback onClose;
  final VoidCallback onReopen;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return PopupMenuButton<_JobMenuAction>(
      icon: const Icon(Icons.more_vert, size: 20),
      padding: EdgeInsets.zero,
      constraints: const BoxConstraints(),
      splashRadius: 18,
      onSelected: (action) {
        switch (action) {
          case _JobMenuAction.edit:
            onEdit();
          case _JobMenuAction.closeOrReopen:
            job.isOpen ? onClose() : onReopen();
          case _JobMenuAction.delete:
            onDelete();
        }
      },
      itemBuilder: (context) => [
        if (canEdit)
          PopupMenuItem(
            value: _JobMenuAction.edit,
            child: Row(
              children: [
                const Icon(Icons.edit_outlined, size: 18),
                const SizedBox(width: 8),
                Text(l10n.jobEditJob),
              ],
            ),
          ),
        if (canEdit)
          PopupMenuItem(
            value: _JobMenuAction.closeOrReopen,
            child: Row(
              children: [
                Icon(
                  job.isOpen ? Icons.lock_outline : Icons.lock_open_outlined,
                  size: 18,
                ),
                const SizedBox(width: 8),
                Text(job.isOpen ? l10n.jobClose : l10n.jobReopen),
              ],
            ),
          ),
        if (canDelete)
          PopupMenuItem(
            value: _JobMenuAction.delete,
            child: Row(
              children: [
                Icon(
                  Icons.delete_outline,
                  size: 18,
                  color: Theme.of(context).colorScheme.error,
                ),
                const SizedBox(width: 8),
                Text(
                  l10n.jobDelete,
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.error,
                  ),
                ),
              ],
            ),
          ),
      ],
    );
  }
}
