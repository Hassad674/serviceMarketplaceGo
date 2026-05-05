import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../../reporting/presentation/widgets/report_bottom_sheet.dart';
import '../../domain/entities/job_application_entity.dart';

/// M-08 candidate detail screen — Soleil v2.
///
/// Soleil ivoire surface, Fraunces section heads, corail-soft card
/// chrome. Top app bar keeps the existing prev/next pager when a
/// candidates list is provided. Bottom action row uses a corail
/// FilledButton for "Send message" and a soft Outlined for
/// "View profile".
class CandidateDetailScreen extends ConsumerStatefulWidget {
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
  ConsumerState<CandidateDetailScreen> createState() =>
      _CandidateDetailScreenState();
}

class _CandidateDetailScreenState
    extends ConsumerState<CandidateDetailScreen> {
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
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;
    final profile = _currentItem.profile;
    final application = _currentItem.application;
    final displayName = profile.name;
    final initials = _initialsFromName(displayName);

    return Scaffold(
      backgroundColor: cs.surface,
      appBar: AppBar(
        backgroundColor: cs.surfaceContainerLowest,
        scrolledUnderElevation: 0,
        elevation: 0,
        title: _hasNavigation
            ? _CandidateNavigator(
                currentIndex: _currentIndex,
                total: _total,
                canGoBack: _canGoBack,
                canGoForward: _canGoForward,
                onPrevious: _goToPrevious,
                onNext: _goToNext,
              )
            : Text(
                l10n.candidateDetail,
                style: SoleilTextStyles.titleLarge.copyWith(
                  color: cs.onSurface,
                  fontSize: 18,
                  fontWeight: FontWeight.w600,
                ),
              ),
        actions: [
          PopupMenuButton<String>(
            onSelected: (value) {
              if (value == 'report') {
                showReportBottomSheet(
                  context,
                  ref,
                  targetType: 'application',
                  targetId: _currentItem.application.id,
                  conversationId: '',
                );
              }
            },
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            itemBuilder: (_) => [
              PopupMenuItem(
                value: 'report',
                child: Row(
                  children: [
                    Icon(
                      Icons.flag_outlined,
                      size: 18,
                      color: cs.error,
                    ),
                    const SizedBox(width: 8),
                    Text(l10n.reportApplication),
                  ],
                ),
              ),
            ],
          ),
        ],
      ),
      body: SafeArea(
        top: false,
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _Eyebrow(label: l10n.jobDetail_m08_panelEyebrow),
              const SizedBox(height: 12),
              _ProfileHeader(
                displayName: displayName,
                initials: initials,
                photoUrl: profile.photoUrl,
                orgType: profile.orgType,
                title: profile.title,
                appliedRelative: l10n.jobDetail_m08_appliedRelative(
                  _formatDate(application.createdAt),
                ),
              ),
              const SizedBox(height: 20),
              _ActionButtons(
                orgId: profile.organizationId,
                displayName: displayName,
                orgType: profile.orgType,
              ),
              if (application.message.isNotEmpty) ...[
                const SizedBox(height: 24),
                _MessageSection(
                  label: l10n.jobDetail_m08_messageHeading,
                  message: application.message,
                ),
              ],
              if (application.videoUrl != null &&
                  application.videoUrl!.isNotEmpty) ...[
                const SizedBox(height: 20),
                _VideoSection(
                  label: l10n.jobDetail_m08_videoPitchHeading,
                  videoUrl: application.videoUrl!,
                ),
              ],
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

class _Eyebrow extends StatelessWidget {
  const _Eyebrow({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Text(
      label,
      style: SoleilTextStyles.mono.copyWith(
        color: theme.colorScheme.primary,
        fontSize: 10.5,
        fontWeight: FontWeight.w700,
        letterSpacing: 1.5,
      ),
    );
  }
}

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
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        IconButton(
          onPressed: canGoBack ? onPrevious : null,
          icon: const Icon(Icons.chevron_left),
          iconSize: 22,
          visualDensity: VisualDensity.compact,
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
        ),
        Text(
          l10n.candidateOf(currentIndex + 1, total),
          style: SoleilTextStyles.bodyEmphasis.copyWith(
            color: theme.colorScheme.onSurface,
            fontSize: 14,
          ),
        ),
        IconButton(
          onPressed: canGoForward ? onNext : null,
          icon: const Icon(Icons.chevron_right),
          iconSize: 22,
          visualDensity: VisualDensity.compact,
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
        ),
      ],
    );
  }
}

class _ProfileHeader extends StatelessWidget {
  const _ProfileHeader({
    required this.displayName,
    required this.initials,
    required this.photoUrl,
    required this.orgType,
    required this.title,
    required this.appliedRelative,
  });

  final String displayName;
  final String initials;
  final String photoUrl;
  final String orgType;
  final String title;
  final String appliedRelative;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>()!;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 60,
          height: 60,
          child: photoUrl.isNotEmpty
              ? ClipOval(
                  child: CachedNetworkImage(
                    imageUrl: photoUrl,
                    width: 60,
                    height: 60,
                    fit: BoxFit.cover,
                    memCacheWidth: 192,
                    memCacheHeight: 192,
                    errorWidget: (_, __, ___) => _InitialsDisc(
                      initials: initials,
                      background: soleil.accentSoft,
                      foreground: soleil.primaryDeep,
                    ),
                  ),
                )
              : _InitialsDisc(
                  initials: initials,
                  background: soleil.accentSoft,
                  foreground: soleil.primaryDeep,
                ),
        ),
        const SizedBox(width: 16),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                displayName,
                style: SoleilTextStyles.titleLarge.copyWith(
                  color: cs.onSurface,
                  fontSize: 22,
                  fontWeight: FontWeight.w600,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 4),
              _OrgTypeBadge(orgType: orgType),
              if (title.isNotEmpty) ...[
                const SizedBox(height: 4),
                Text(
                  title,
                  style: SoleilTextStyles.body.copyWith(
                    color: cs.onSurfaceVariant,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
              const SizedBox(height: 6),
              Text(
                appliedRelative,
                style: SoleilTextStyles.mono.copyWith(
                  color: soleil.subtleForeground,
                  fontSize: 10.5,
                  fontWeight: FontWeight.w600,
                  letterSpacing: 0.8,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _InitialsDisc extends StatelessWidget {
  const _InitialsDisc({
    required this.initials,
    required this.background,
    required this.foreground,
  });

  final String initials;
  final Color background;
  final Color foreground;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: background,
        shape: BoxShape.circle,
      ),
      alignment: Alignment.center,
      child: Text(
        initials,
        style: SoleilTextStyles.titleMedium.copyWith(
          color: foreground,
          fontSize: 18,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _OrgTypeBadge extends StatelessWidget {
  const _OrgTypeBadge({required this.orgType});

  final String orgType;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;

    final label = switch (orgType) {
      'agency' => l10n.jobDetail_m08_orgAgency,
      'enterprise' => l10n.jobDetail_m08_orgEnterprise,
      _ => l10n.jobDetail_m08_orgFreelance,
    };
    final isFreelance = orgType == 'provider_personal';
    final isAgency = orgType == 'agency';

    final Color background;
    final Color foreground;
    if (isFreelance) {
      background = soleil.accentSoft;
      foreground = soleil.primaryDeep;
    } else if (isAgency) {
      background = soleil.successSoft;
      foreground = soleil.success;
    } else {
      background = soleil.amberSoft;
      foreground = cs.onSurface;
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        label,
        style: SoleilTextStyles.caption.copyWith(
          color: foreground,
          fontSize: 11,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.4,
        ),
      ),
    );
  }
}

class _MessageSection extends StatelessWidget {
  const _MessageSection({required this.label, required this.message});

  final String label;
  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: cs.onSurface,
            fontSize: 16,
          ),
        ),
        const SizedBox(height: 10),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: cs.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(AppTheme.radiusXl),
            border: Border.all(color: cs.outline),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Text(
            message,
            style: SoleilTextStyles.bodyLarge.copyWith(
              color: cs.onSurface,
              height: 1.6,
            ),
          ),
        ),
      ],
    );
  }
}

class _VideoSection extends StatelessWidget {
  const _VideoSection({required this.label, required this.videoUrl});

  final String label;
  final String videoUrl;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: cs.onSurface,
            fontSize: 16,
          ),
        ),
        const SizedBox(height: 10),
        ClipRRect(
          borderRadius: BorderRadius.circular(AppTheme.radiusXl),
          child: VideoPlayerWidget(videoUrl: videoUrl),
        ),
      ],
    );
  }
}

class _ActionButtons extends StatelessWidget {
  const _ActionButtons({
    required this.orgId,
    required this.displayName,
    required this.orgType,
  });

  final String orgId;
  final String displayName;
  final String orgType;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;

    return Row(
      children: [
        Expanded(
          child: OutlinedButton.icon(
            onPressed: () => context.push(
              '/profiles/$orgId',
              extra: {
                'display_name': displayName,
                'org_type': orgType,
              },
            ),
            icon: const Icon(Icons.person_outline, size: 18),
            label: Text(l10n.jobDetail_m08_viewProfile),
            style: OutlinedButton.styleFrom(
              foregroundColor: cs.onSurface,
              side: BorderSide(color: cs.outlineVariant),
              minimumSize: const Size.fromHeight(46),
              shape: const StadiumBorder(),
              textStyle: SoleilTextStyles.button,
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: FilledButton.icon(
            onPressed: () => context.push(
              '${RoutePaths.newChat}/$orgId',
              extra: {'name': displayName},
            ),
            icon: const Icon(Icons.send_rounded, size: 18),
            label: Text(l10n.jobDetail_m08_sendMessage),
            style: FilledButton.styleFrom(
              backgroundColor: cs.primary,
              foregroundColor: cs.onPrimary,
              minimumSize: const Size.fromHeight(46),
              shape: const StadiumBorder(),
              textStyle: SoleilTextStyles.button,
            ),
          ),
        ),
      ],
    );
  }
}

String _initialsFromName(String name) {
  final trimmed = name.trim();
  if (trimmed.isEmpty) return '?';
  final parts = trimmed.split(RegExp(r'\s+'));
  if (parts.length == 1) return parts.first[0].toUpperCase();
  return '${parts.first[0]}${parts.last[0]}'.toUpperCase();
}
