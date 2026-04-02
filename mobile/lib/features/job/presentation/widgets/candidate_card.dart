import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../domain/entities/job_application_entity.dart';

/// Compact candidate card displayed in the candidates list.
///
/// Shows avatar, name, role badge, truncated message, and date.
/// Tapping navigates to [CandidateDetailScreen] for full details.
class CandidateCard extends StatelessWidget {
  const CandidateCard({
    super.key,
    required this.item,
    required this.jobId,
    this.candidates,
    this.candidateIndex,
  });

  final ApplicationWithProfile item;
  final String jobId;
  final List<ApplicationWithProfile>? candidates;
  final int? candidateIndex;

  @override
  Widget build(BuildContext context) {
    final profile = item.profile;
    final application = item.application;
    final fullName = '${profile.firstName} ${profile.lastName}'.trim();
    final displayName = profile.displayName.isNotEmpty
        ? profile.displayName
        : fullName;
    final initials =
        '${profile.firstName.isNotEmpty ? profile.firstName[0] : ''}${profile.lastName.isNotEmpty ? profile.lastName[0] : ''}'
            .toUpperCase();

    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => context.push(
          RoutePaths.candidateDetail,
          extra: {
            'item': item,
            'jobId': jobId,
            if (candidates != null) 'candidates': candidates,
            if (candidateIndex != null) 'candidateIndex': candidateIndex,
          },
        ),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Header: avatar + name + role badge
              Row(
                children: [
                  CircleAvatar(
                    radius: 24,
                    backgroundColor: const Color(0xFFFFF1F2),
                    backgroundImage: profile.photoUrl.isNotEmpty
                        ? CachedNetworkImageProvider(profile.photoUrl)
                        : null,
                    child: profile.photoUrl.isEmpty
                        ? Text(
                            initials.isNotEmpty ? initials : '?',
                            style: const TextStyle(
                              color: Color(0xFFF43F5E),
                              fontWeight: FontWeight.w600,
                              fontSize: 16,
                            ),
                          )
                        : null,
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          displayName,
                          style: Theme.of(context)
                              .textTheme
                              .titleMedium
                              ?.copyWith(fontWeight: FontWeight.w700),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        if (fullName.isNotEmpty &&
                            fullName != displayName) ...[
                          const SizedBox(height: 1),
                          Text(
                            fullName,
                            style: Theme.of(context)
                                .textTheme
                                .bodySmall
                                ?.copyWith(color: Colors.grey.shade600),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ],
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            Container(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 6,
                                vertical: 1,
                              ),
                              decoration: BoxDecoration(
                                color: profile.role == 'provider'
                                    ? const Color(0xFFFFF1F2)
                                    : Colors.blue.shade50,
                                borderRadius: BorderRadius.circular(8),
                              ),
                              child: Text(
                                profile.role == 'provider'
                                    ? 'Freelance'
                                    : 'Agence',
                                style: TextStyle(
                                  fontSize: 10,
                                  fontWeight: FontWeight.w500,
                                  color: profile.role == 'provider'
                                      ? const Color(0xFFF43F5E)
                                      : Colors.blue.shade700,
                                ),
                              ),
                            ),
                            if (profile.title.isNotEmpty) ...[
                              const SizedBox(width: 6),
                              Flexible(
                                child: Text(
                                  profile.title,
                                  style: Theme.of(context)
                                      .textTheme
                                      .bodySmall
                                      ?.copyWith(color: Colors.grey),
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                            ],
                          ],
                        ),
                      ],
                    ),
                  ),
                  // Chevron to hint tappability
                  Icon(
                    Icons.chevron_right,
                    color: Colors.grey.shade400,
                    size: 20,
                  ),
                ],
              ),

              // Application message (truncated to 2 lines)
              if (application.message.isNotEmpty) ...[
                const SizedBox(height: 10),
                Text(
                  application.message,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],

              // Date
              const SizedBox(height: 8),
              Text(
                _formatDate(application.createdAt),
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Colors.grey.shade500,
                      fontSize: 11,
                    ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  String _formatDate(String isoDate) {
    try {
      final dt = DateTime.parse(isoDate);
      return '${dt.day.toString().padLeft(2, '0')}/${dt.month.toString().padLeft(2, '0')}/${dt.year}';
    } catch (_) {
      return isoDate;
    }
  }
}
