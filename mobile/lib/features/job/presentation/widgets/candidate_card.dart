import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../domain/entities/job_application_entity.dart';
import '../../../../core/theme/app_palette.dart';

/// Builds 1- or 2-letter initials from a display name. Returns "?" for
/// empty or whitespace-only names.
String _initialsFromName(String name) {
  final trimmed = name.trim();
  if (trimmed.isEmpty) return '?';
  final parts = trimmed.split(RegExp(r'\s+'));
  if (parts.length == 1) return parts.first[0].toUpperCase();
  return '${parts.first[0]}${parts.last[0]}'.toUpperCase();
}

/// Compact candidate card displayed in the candidates list.
///
/// Shows avatar, org name, org-type badge, truncated message, and date.
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
    final displayName = profile.name;
    final initials = _initialsFromName(displayName);

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
                    backgroundColor: AppPalette.rose50,
                    // 24 lp radius × 3x DPR = ~144 px raster. Cap
                    // disk + memory cache at 128 (PERF-M-05).
                    backgroundImage: profile.photoUrl.isNotEmpty
                        ? CachedNetworkImageProvider(
                            profile.photoUrl,
                            maxWidth: 128,
                            maxHeight: 128,
                          )
                        : null,
                    child: profile.photoUrl.isEmpty
                        ? Text(
                            initials,
                            style: const TextStyle(
                              color: AppPalette.rose500,
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
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            Container(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 6,
                                vertical: 1,
                              ),
                              decoration: BoxDecoration(
                                color: profile.orgType == 'provider_personal'
                                    ? AppPalette.rose50
                                    : Colors.blue.shade50,
                                borderRadius: BorderRadius.circular(8),
                              ),
                              child: Text(
                                profile.orgType == 'provider_personal'
                                    ? 'Freelance'
                                    : 'Agency',
                                style: TextStyle(
                                  fontSize: 10,
                                  fontWeight: FontWeight.w500,
                                  color: profile.orgType == 'provider_personal'
                                      ? AppPalette.rose500
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
