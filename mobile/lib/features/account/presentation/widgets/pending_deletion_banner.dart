import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/deletion_status.dart';

/// PendingDeletionBanner is the dashboard alert that surfaces when
/// the user is in their 30-day GDPR cooldown. Tapping the cancel
/// button takes them to [CancelDeletionScreen]. Hidden when the
/// supplied status has [DeletionStatus.isPending] == false.
class PendingDeletionBanner extends StatelessWidget {
  final DeletionStatus status;
  final VoidCallback onTapCancel;

  const PendingDeletionBanner({
    super.key,
    required this.status,
    required this.onTapCancel,
  });

  @override
  Widget build(BuildContext context) {
    if (!status.isPending) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context)!;
    final hard = status.hardDeleteAt;
    final body = hard != null
        ? l10n.gdprBannerBody(MaterialLocalizations.of(context).formatShortDate(hard))
        : l10n.gdprBannerBodyNoDate;

    return Material(
      color: Colors.amber.shade50,
      child: InkWell(
        onTap: onTapCancel,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          child: Row(
            children: [
              const Icon(Icons.warning_amber_rounded, color: Colors.orange),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.gdprBannerTitle,
                      style: const TextStyle(fontWeight: FontWeight.bold),
                    ),
                    const SizedBox(height: 4),
                    Text(body),
                  ],
                ),
              ),
              const Icon(Icons.chevron_right),
            ],
          ),
        ),
      ),
    );
  }
}
