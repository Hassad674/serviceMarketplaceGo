import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../domain/entities/job_application_entity.dart';

/// Full-page detail screen for a job application / candidate.
///
/// Shows profile header, application message, optional video,
/// and action buttons (view profile, send message).
/// Optionally accepts a full candidates list + index for prev/next
/// navigation between candidates.
class CandidateDetailScreen extends StatefulWidget {
  const CandidateDetailScreen({
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
  State<CandidateDetailScreen> createState() => _CandidateDetailScreenState();
}

class _CandidateDetailScreenState extends State<CandidateDetailScreen> {
  late int _currentIndex;
  late ApplicationWithProfile _currentItem;

  bool get _hasNavigation =>
      widget.candidates != null && widget.candidates!.length > 1;

  int get _total => widget.candidates?.length ?? 1;

  bool get _canGoBack => _currentIndex > 0;

  bool get _canGoForward => _currentIndex < _total - 1;

  @override
  void initState() {
    super.initState();
    _currentIndex = widget.candidateIndex ?? 0;
    _currentItem = widget.item;
  }

  void _goToPrevious() {
    if (!_canGoBack) return;
    setState(() {
      _currentIndex--;
      _currentItem = widget.candidates![_currentIndex];
    });
  }

  void _goToNext() {
    if (!_canGoForward) return;
    setState(() {
      _currentIndex++;
      _currentItem = widget.candidates![_currentIndex];
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final profile = _currentItem.profile;
    final application = _currentItem.application;
    final fullName = '${profile.firstName} ${profile.lastName}'.trim();
    final displayName =
        profile.displayName.isNotEmpty ? profile.displayName : fullName;
    final initials =
        '${profile.firstName.isNotEmpty ? profile.firstName[0] : ''}${profile.lastName.isNotEmpty ? profile.lastName[0] : ''}'
            .toUpperCase();

    return Scaffold(
      appBar: AppBar(
        title: _hasNavigation
            ? _CandidateNavigator(
                currentIndex: _currentIndex,
                total: _total,
                canGoBack: _canGoBack,
                canGoForward: _canGoForward,
                onPrevious: _goToPrevious,
                onNext: _goToNext,
              )
            : Text(l10n.candidateDetail),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Profile header
            _ProfileHeader(
              displayName: displayName,
              fullName: fullName,
              initials: initials,
              photoUrl: profile.photoUrl,
              role: profile.role,
              title: profile.title,
            ),

            const SizedBox(height: 16),

            // Application date
            _ApplicationDate(createdAt: application.createdAt),

            // Application message
            if (application.message.isNotEmpty) ...[
              const SizedBox(height: 20),
              _MessageSection(
                label: l10n.applicationMessage,
                message: application.message,
              ),
            ],

            // Application video
            if (application.videoUrl != null &&
                application.videoUrl!.isNotEmpty) ...[
              const SizedBox(height: 20),
              _VideoSection(
                label: l10n.applicationVideo,
                videoUrl: application.videoUrl!,
              ),
            ],

            const SizedBox(height: 28),

            // Action buttons
            _ActionButtons(
              userId: profile.userId,
              displayName: displayName,
              role: profile.role,
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Candidate navigator — prev/next arrows with counter
// ---------------------------------------------------------------------------

class _CandidateNavigator extends StatelessWidget {
  const _CandidateNavigator({
    required this.currentIndex,
    required this.total,
    required this.canGoBack,
    required this.canGoForward,
    required this.onPrevious,
    required this.onNext,
  });

  final int currentIndex;
  final int total;
  final bool canGoBack;
  final bool canGoForward;
  final VoidCallback onPrevious;
  final VoidCallback onNext;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        IconButton(
          onPressed: canGoBack ? onPrevious : null,
          icon: const Icon(Icons.chevron_left),
          iconSize: 24,
          visualDensity: VisualDensity.compact,
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
        ),
        Text(
          l10n.candidateOf(currentIndex + 1, total),
          style: Theme.of(context).textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w600,
              ),
        ),
        IconButton(
          onPressed: canGoForward ? onNext : null,
          icon: const Icon(Icons.chevron_right),
          iconSize: 24,
          visualDensity: VisualDensity.compact,
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Profile header — avatar + name + role badge + title
// ---------------------------------------------------------------------------

class _ProfileHeader extends StatelessWidget {
  const _ProfileHeader({
    required this.displayName,
    required this.fullName,
    required this.initials,
    required this.photoUrl,
    required this.role,
    required this.title,
  });

  final String displayName;
  final String fullName;
  final String initials;
  final String photoUrl;
  final String role;
  final String title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Row(
      children: [
        CircleAvatar(
          radius: 30,
          backgroundColor: const Color(0xFFFFF1F2),
          backgroundImage: photoUrl.isNotEmpty
              ? CachedNetworkImageProvider(photoUrl)
              : null,
          child: photoUrl.isEmpty
              ? Text(
                  initials.isNotEmpty ? initials : '?',
                  style: const TextStyle(
                    color: Color(0xFFF43F5E),
                    fontWeight: FontWeight.w600,
                    fontSize: 20,
                  ),
                )
              : null,
        ),
        const SizedBox(width: 16),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                displayName,
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              if (fullName.isNotEmpty && fullName != displayName) ...[
                const SizedBox(height: 2),
                Text(
                  fullName,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: Colors.grey.shade600,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
              const SizedBox(height: 6),
              Row(
                children: [
                  _RoleBadge(role: role),
                  if (title.isNotEmpty) ...[
                    const SizedBox(width: 8),
                    Flexible(
                      child: Text(
                        title,
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: Colors.grey,
                        ),
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
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Role badge
// ---------------------------------------------------------------------------

class _RoleBadge extends StatelessWidget {
  const _RoleBadge({required this.role});

  final String role;

  @override
  Widget build(BuildContext context) {
    final isProvider = role == 'provider';
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: isProvider ? const Color(0xFFFFF1F2) : Colors.blue.shade50,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        isProvider ? 'Freelance' : 'Agence',
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w500,
          color: isProvider ? const Color(0xFFF43F5E) : Colors.blue.shade700,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Application date
// ---------------------------------------------------------------------------

class _ApplicationDate extends StatelessWidget {
  const _ApplicationDate({required this.createdAt});

  final String createdAt;

  @override
  Widget build(BuildContext context) {
    return Text(
      _formatDate(createdAt),
      style: Theme.of(context).textTheme.bodySmall?.copyWith(
            color: Colors.grey.shade500,
            fontSize: 12,
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

// ---------------------------------------------------------------------------
// Message section — card with full application message
// ---------------------------------------------------------------------------

class _MessageSection extends StatelessWidget {
  const _MessageSection({
    required this.label,
    required this.message,
  });

  final String label;
  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: theme.textTheme.titleSmall?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: theme.dividerColor.withValues(alpha: 0.5),
            ),
          ),
          child: Text(
            message,
            style: theme.textTheme.bodyMedium,
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Video section — titled video player
// ---------------------------------------------------------------------------

class _VideoSection extends StatelessWidget {
  const _VideoSection({
    required this.label,
    required this.videoUrl,
  });

  final String label;
  final String videoUrl;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: theme.textTheme.titleSmall?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        ClipRRect(
          borderRadius: BorderRadius.circular(12),
          child: VideoPlayerWidget(videoUrl: videoUrl),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Action buttons — view profile + send message
// ---------------------------------------------------------------------------

class _ActionButtons extends StatelessWidget {
  const _ActionButtons({
    required this.userId,
    required this.displayName,
    required this.role,
  });

  final String userId;
  final String displayName;
  final String role;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: OutlinedButton.icon(
            onPressed: () => context.push(
              '/profiles/$userId',
              extra: {
                'display_name': displayName,
                'role': role,
              },
            ),
            icon: const Icon(Icons.person_outline, size: 18),
            label: const Text('Voir le profil'),
            style: OutlinedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 12),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(10),
              ),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: FilledButton.icon(
            onPressed: () => context.push(
              '${RoutePaths.newChat}/$userId',
              extra: {'name': displayName},
            ),
            icon: const Icon(Icons.send, size: 18),
            label: const Text('Message'),
            style: FilledButton.styleFrom(
              backgroundColor: const Color(0xFFF43F5E),
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(vertical: 12),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(10),
              ),
            ),
          ),
        ),
      ],
    );
  }
}
