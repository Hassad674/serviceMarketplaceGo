import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/job_application_entity.dart';

/// Builds 1- or 2-letter initials from a display name. Returns "?" for
/// empty or whitespace-only names.
String _initialsFromName(String name) {
  final trimmed = name.trim();
  if (trimmed.isEmpty) return '?';
  final parts = trimmed.split(RegExp(r'\s+'));
  if (parts.length == 1) return parts.first[0].toUpperCase();
  return '${parts.first[0]}${parts.last[0]}'.toUpperCase();
}

/// M-08 Soleil v2 candidate card.
///
/// Ivoire surface, soft border, no elevation. 40 dp avatar (CDN-cached
/// when present, corail-soft initials disc otherwise). Soleil pills:
/// corail-soft (freelance), sapin-soft (agency), amber-soft
/// (enterprise). Date is rendered as a mono uppercase label. Tapping
/// pushes [CandidateDetailScreen] with the same arguments shape as
/// before.
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
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;
    final profile = item.profile;
    final application = item.application;
    final displayName = profile.name;
    final initials = _initialsFromName(displayName);
    final orgLabel = _orgLabel(profile.orgType, l10n);

    return Material(
      color: cs.surfaceContainerLowest,
      borderRadius: BorderRadius.circular(AppTheme.radiusXl),
      child: InkWell(
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        onTap: () => context.push(
          RoutePaths.candidateDetail,
          extra: {
            'item': item,
            'jobId': jobId,
            if (candidates != null) 'candidates': candidates,
            if (candidateIndex != null) 'candidateIndex': candidateIndex,
          },
        ),
        child: Ink(
          decoration: BoxDecoration(
            color: cs.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(AppTheme.radiusXl),
            border: Border.all(color: cs.outline),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Padding(
            padding: const EdgeInsets.all(18),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    _Avatar(photoUrl: profile.photoUrl, initials: initials),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            displayName,
                            style: SoleilTextStyles.bodyEmphasis.copyWith(
                              color: cs.onSurface,
                              fontSize: 14.5,
                            ),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                          const SizedBox(height: 4),
                          Wrap(
                            spacing: 6,
                            runSpacing: 4,
                            crossAxisAlignment: WrapCrossAlignment.center,
                            children: [
                              _OrgPill(orgType: profile.orgType, label: orgLabel),
                              if (application.videoUrl != null &&
                                  application.videoUrl!.isNotEmpty)
                                _VideoBadge(label: l10n.jobDetail_m08_videoBadge),
                              if (profile.title.isNotEmpty)
                                Text(
                                  profile.title,
                                  style: SoleilTextStyles.caption.copyWith(
                                    color: cs.onSurfaceVariant,
                                  ),
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                ),
                            ],
                          ),
                        ],
                      ),
                    ),
                    Icon(
                      Icons.chevron_right,
                      color: cs.onSurfaceVariant,
                      size: 20,
                    ),
                  ],
                ),
                if (application.message.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  Padding(
                    padding: const EdgeInsets.only(left: 52),
                    child: Text(
                      application.message,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: SoleilTextStyles.body.copyWith(
                        color: cs.onSurfaceVariant,
                        height: 1.5,
                      ),
                    ),
                  ),
                ],
                const SizedBox(height: 8),
                Padding(
                  padding: const EdgeInsets.only(left: 52),
                  child: Text(
                    l10n.jobDetail_m08_appliedRelative(
                      _formatDate(application.createdAt),
                    ),
                    style: SoleilTextStyles.mono.copyWith(
                      color: theme
                              .extension<AppColors>()
                              ?.subtleForeground ??
                          cs.onSurfaceVariant,
                      fontSize: 10.5,
                      fontWeight: FontWeight.w600,
                      letterSpacing: 0.8,
                    ),
                  ),
                ),
              ],
            ),
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

  String _orgLabel(String orgType, AppLocalizations l10n) {
    return switch (orgType) {
      'agency' => l10n.jobDetail_m08_orgAgency,
      'enterprise' => l10n.jobDetail_m08_orgEnterprise,
      _ => l10n.jobDetail_m08_orgFreelance,
    };
  }
}

class _Avatar extends StatelessWidget {
  const _Avatar({required this.photoUrl, required this.initials});

  final String photoUrl;
  final String initials;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final soleil = theme.extension<AppColors>()!;

    return SizedBox(
      width: 40,
      height: 40,
      child: photoUrl.isNotEmpty
          ? ClipOval(
              child: CachedNetworkImage(
                imageUrl: photoUrl,
                width: 40,
                height: 40,
                fit: BoxFit.cover,
                memCacheWidth: 128,
                memCacheHeight: 128,
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
        style: SoleilTextStyles.bodyEmphasis.copyWith(
          color: foreground,
          fontSize: 13,
        ),
      ),
    );
  }
}

class _OrgPill extends StatelessWidget {
  const _OrgPill({required this.orgType, required this.label});

  final String orgType;
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final soleil = theme.extension<AppColors>()!;
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
      foreground = theme.colorScheme.onSurface;
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        label,
        style: SoleilTextStyles.caption.copyWith(
          color: foreground,
          fontSize: 10.5,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.4,
        ),
      ),
    );
  }
}

class _VideoBadge extends StatelessWidget {
  const _VideoBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        border: Border.all(color: cs.outline),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.play_arrow_rounded,
            size: 11,
            color: cs.onSurfaceVariant,
          ),
          const SizedBox(width: 2),
          Text(
            label,
            style: SoleilTextStyles.caption.copyWith(
              color: cs.onSurfaceVariant,
              fontSize: 10.5,
              fontWeight: FontWeight.w600,
              letterSpacing: 0.3,
            ),
          ),
        ],
      ),
    );
  }
}
