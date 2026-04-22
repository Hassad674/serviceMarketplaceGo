import 'package:flutter/material.dart';

import '../../../../shared/widgets/project_history_card.dart';
import '../../domain/entities/project_history_entry.dart';

/// Feature-level wrapper that adapts a [ProjectHistoryEntry] to the
/// shared [ProjectHistoryCard]. Keeps the provider-profile call sites
/// untouched while the underlying visual is shared with the client
/// profile project history.
class ProjectHistoryEntryCard extends StatelessWidget {
  final ProjectHistoryEntry entry;

  const ProjectHistoryEntryCard({super.key, required this.entry});

  @override
  Widget build(BuildContext context) {
    return ProjectHistoryCard(
      title: entry.title,
      amountCents: entry.amount,
      completedAt: entry.completedAt,
      review: entry.review,
    );
  }
}
