import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../reporting/presentation/widgets/report_bottom_sheet.dart';
import '../../domain/entities/job_entity.dart';

class OpportunityCard extends ConsumerWidget {
  const OpportunityCard({super.key, required this.job, this.hasApplied = false});

  final JobEntity job;
  final bool hasApplied;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => context.push('/opportunities/detail', extra: job.id),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      job.title,
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w600,
                        color: theme.colorScheme.onSurface,
                      ),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  const SizedBox(width: 8),
                  if (hasApplied)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: Colors.grey.shade100,
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        l10n.alreadyApplied,
                        style: TextStyle(fontSize: 11, color: Colors.grey.shade600, fontWeight: FontWeight.w500),
                      ),
                    ),
                  SizedBox(
                    width: 28,
                    height: 28,
                    child: PopupMenuButton<String>(
                      padding: EdgeInsets.zero,
                      iconSize: 18,
                      icon: Icon(Icons.more_vert, size: 18, color: Colors.grey.shade400),
                      onSelected: (value) {
                        if (value == 'report') {
                          showReportBottomSheet(
                            context,
                            ref,
                            targetType: 'job',
                            targetId: job.id,
                            conversationId: '',
                          );
                        }
                      },
                      itemBuilder: (_) => [
                        PopupMenuItem(
                          value: 'report',
                          child: Row(
                            children: [
                              const Icon(Icons.flag_outlined, size: 18, color: Colors.red),
                              const SizedBox(width: 8),
                              Text(l10n.report),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
              if (job.description.isNotEmpty) ...[
                const SizedBox(height: 6),
                Text(job.description, maxLines: 2, overflow: TextOverflow.ellipsis, style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
              ],
              if (job.skills.isNotEmpty) ...[
                const SizedBox(height: 8),
                Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  children: job.skills.take(3).map((s) => Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(color: const Color(0xFFFFF1F2), borderRadius: BorderRadius.circular(12)),
                    child: Text(s, style: const TextStyle(fontSize: 11, color: Color(0xFFF43F5E), fontWeight: FontWeight.w500)),
                  )).toList(),
                ),
              ],
              const SizedBox(height: 10),
              Row(
                children: [
                  const Icon(Icons.euro, size: 14, color: Colors.grey),
                  const SizedBox(width: 4),
                  Text('${job.minBudget}\u20ac - ${job.maxBudget}\u20ac', style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurface)),
                  const Spacer(),
                  const Icon(Icons.calendar_today, size: 14, color: Colors.grey),
                  const SizedBox(width: 4),
                  Text(_formatDate(job.createdAt), style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  String _formatDate(String dateStr) {
    try {
      final d = DateTime.parse(dateStr);
      return '${d.day}/${d.month}/${d.year}';
    } catch (_) {
      return dateStr;
    }
  }
}
