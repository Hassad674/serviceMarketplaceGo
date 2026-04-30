import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../expertise/domain/entities/expertise_catalog.dart';
import '../../../expertise/presentation/widgets/expertise_section_widget.dart';
import '../../../portfolio/presentation/widgets/portfolio_grid_widget.dart';
import '../../../profile_tier1/domain/entities/availability_status.dart';
import '../../../profile_tier1/domain/entities/languages.dart';
import '../../../profile_tier1/domain/entities/location.dart';
import '../../../profile_tier1/domain/entities/pricing_kind.dart';
import '../../../profile_tier1/presentation/widgets/availability_section_widget.dart';
import '../../../profile_tier1/presentation/widgets/languages_section_widget.dart';
import '../../../profile_tier1/presentation/widgets/location_section_widget.dart';
import '../../../profile_tier1/presentation/widgets/pricing_section_widget.dart';
import '../../../project_history/presentation/widgets/project_history_widget.dart';
import '../../../skill/domain/entities/skill_limits.dart';
import '../../../skill/presentation/widgets/skills_section_widget.dart';
import '../providers/profile_provider.dart';
import '../widgets/profile_about_section.dart';
import '../widgets/profile_edit_dialogs.dart';
import '../widgets/profile_header_card.dart';
import '../widgets/profile_screen_actions.dart';
import '../widgets/profile_video_section.dart';

/// LEGACY AGENCY-ONLY profile screen. Kept for the agency org type
/// until the agency path migrates to the split-profile architecture.
/// `provider_personal` users now hit [FreelanceProfileScreen] via the
/// router dispatcher. Do not add features to this file — create them
/// on the split-profile modules instead.
class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final profileAsync = ref.watch(profileProvider);
    final l10n = AppLocalizations.of(context)!;
    final canEditProfile = ref.watch(
      hasPermissionProvider(OrgPermission.orgProfileEdit),
    );

    final user = authState.user;
    final displayName = user?['display_name'] as String? ??
        user?['first_name'] as String? ??
        '';
    final email = user?['email'] as String? ?? '';
    final role = user?['role'] as String?;
    final initials = _buildInitials(displayName);

    final profileTitle = profileAsync.whenOrNull(
      data: (p) => p['title'] as String?,
    );
    final profileAbout = profileAsync.whenOrNull(
      data: (p) => p['about'] as String?,
    );
    final profileOrgId = profileAsync.whenOrNull(
      data: (p) => p['organization_id'] as String?,
    );
    final profileExpertise = profileAsync.whenOrNull(
          data: (p) => (p['expertise_domains'] as List<dynamic>?)
              ?.whereType<String>()
              .toList(),
        ) ??
        const <String>[];
    final orgType = authState.organization?['type'] as String?;
    final referrerEnabled =
        (authState.user?['referrer_enabled'] as bool?) ?? false;
    final tier1Enabled = _isTier1EnabledForOrgType(orgType);

    final currentLocation = profileAsync.whenOrNull(
          data: (p) => Location.fromJson(p),
        ) ??
        Location.empty;
    final currentLanguages = profileAsync.whenOrNull(
          data: (p) => Languages.fromJson(p),
        ) ??
        Languages.empty;
    final currentAvailability = AvailabilityStatus.fromWire(
      profileAsync.whenOrNull(
        data: (p) => p['availability_status'] as String?,
      ),
    );
    final currentReferrerAvailability = AvailabilityStatus.fromWireOrNull(
      profileAsync.whenOrNull(
        data: (p) => p['referrer_availability_status'] as String?,
      ),
    );

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.myProfile),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              ProfileHeaderCard(
                initials: initials,
                displayName: displayName,
                email: email,
                role: role,
                photoUrl: profileAsync.whenOrNull(
                  data: (p) => p['photo_url'] as String?,
                ),
                onPhotoTap: canEditProfile
                    ? () => openProfilePhotoUpload(context, ref)
                    : null,
              ),
              const SizedBox(height: 16),

              // Tier 1 completion — availability / pricing / location / languages
              // ordered to match the harmonized agency layout.
              if (tier1Enabled) ...[
                AvailabilitySectionWidget(
                  variant: AvailabilityVariant.direct,
                  initialDirect: currentAvailability,
                  initialReferrer: currentReferrerAvailability,
                  referrerEnabled: referrerEnabled,
                  canEdit: canEditProfile,
                  onSaved: () => ref.invalidate(profileProvider),
                ),
                const SizedBox(height: 16),
                PricingSectionWidget(
                  variant: PricingKind.direct,
                  orgType: orgType,
                  referrerEnabled: referrerEnabled,
                  canEdit: canEditProfile,
                  onSaved: () => ref.invalidate(profileProvider),
                ),
                if (referrerEnabled) ...[
                  const SizedBox(height: 16),
                  PricingSectionWidget(
                    variant: PricingKind.referral,
                    orgType: orgType,
                    referrerEnabled: referrerEnabled,
                    canEdit: canEditProfile,
                    onSaved: () => ref.invalidate(profileProvider),
                  ),
                ],
                const SizedBox(height: 16),
                LocationSectionWidget(
                  initialLocation: currentLocation,
                  orgType: orgType,
                  canEdit: canEditProfile,
                  onSaved: () => ref.invalidate(profileProvider),
                ),
                const SizedBox(height: 16),
                LanguagesSectionWidget(
                  initialLanguages: currentLanguages,
                  canEdit: canEditProfile,
                  onSaved: () => ref.invalidate(profileProvider),
                ),
                const SizedBox(height: 16),
              ],

              // Title
              ProfileTitleSection(title: profileTitle),
              const SizedBox(height: 16),

              // About
              ProfileAboutSection(
                about: profileAbout,
                onTap: canEditProfile
                    ? () => openProfileAboutEditor(context, ref, profileAbout)
                    : null,
              ),
              const SizedBox(height: 16),

              // Presentation video
              ProfileVideoSection(
                videoUrl: profileAsync.whenOrNull(
                  data: (p) => p['presentation_video_url'] as String?,
                ),
                onUploadTap: canEditProfile
                    ? () => openProfileVideoUpload(context, ref)
                    : null,
                onDeleteTap: canEditProfile
                    ? () => confirmDeleteProfileVideo(context, ref)
                    : null,
              ),
              const SizedBox(height: 16),

              // Expertise — hidden for enterprise org type.
              ExpertiseSectionWidget(
                orgType: orgType,
                initialDomains: profileExpertise,
                canEdit: canEditProfile,
                onSaved: () => ref.invalidate(profileProvider),
              ),
              if (ExpertiseCatalog.isFeatureEnabledForOrgType(orgType))
                const SizedBox(height: 16),

              // Skills — hidden for enterprise org type.
              SkillsSectionWidget(
                orgType: orgType,
                expertiseKeys: profileExpertise,
                canEdit: canEditProfile,
                onSaved: () => ref.invalidate(profileProvider),
              ),
              if (SkillLimits.isFeatureEnabledForOrgType(orgType))
                const SizedBox(height: 16),

              // Portfolio + project history
              if (profileOrgId != null) ...[
                PortfolioGridWidget(
                  orgId: profileOrgId,
                  readOnly: false,
                ),
                const SizedBox(height: 16),
                ProjectHistoryWidget(orgId: profileOrgId),
                const SizedBox(height: 16),
              ],

              const ProfileDarkModeToggle(),
              const SizedBox(height: 24),

              ProfileLogoutButton(
                onPressed: () async {
                  await ref.read(authProvider.notifier).logout();
                  if (context.mounted) context.go(RoutePaths.login);
                },
              ),
              const SizedBox(height: 24),
            ],
          ),
        ),
      ),
    );
  }

  String _buildInitials(String name) {
    if (name.isEmpty) return '?';
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }

  bool _isTier1EnabledForOrgType(String? orgType) {
    switch (orgType) {
      case 'agency':
      case 'provider_personal':
        return true;
      default:
        return false;
    }
  }
}
