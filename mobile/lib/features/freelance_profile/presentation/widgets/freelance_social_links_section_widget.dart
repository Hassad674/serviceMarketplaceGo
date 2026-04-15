import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../shared/widgets/social_links_card.dart';
import '../providers/freelance_social_links_providers.dart';

/// FreelanceSocialLinksSectionWidget mounts the shared social-links
/// card on the owner's freelance profile screen. It owns the
/// Riverpod plumbing so the shared widget stays decoupled from the
/// persona's data layer.
class FreelanceSocialLinksSectionWidget extends ConsumerWidget {
  const FreelanceSocialLinksSectionWidget({
    super.key,
    required this.canEdit,
  });

  final bool canEdit;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final linksAsync = ref.watch(freelanceSocialLinksProvider);
    final repo = ref.read(freelanceSocialLinksRepositoryProvider);

    return linksAsync.when(
      loading: () => const SocialLinksCard(links: [], isLoading: true),
      error: (_, __) => const SizedBox.shrink(),
      data: (raw) {
        final links = raw
            .map(
              (entry) => SocialLinkEntry(
                platform: (entry['platform'] as String?) ?? '',
                url: (entry['url'] as String?) ?? '',
              ),
            )
            .where((e) => e.platform.isNotEmpty && e.url.isNotEmpty)
            .toList();
        return SocialLinksCard(
          links: links,
          editor: canEdit
              ? SocialLinksEditorConfig(
                  onUpsert: (platform, url) async {
                    await repo.upsert(platform, url);
                    ref.invalidate(freelanceSocialLinksProvider);
                  },
                  onDelete: (platform) async {
                    await repo.delete(platform);
                    ref.invalidate(freelanceSocialLinksProvider);
                  },
                )
              : null,
        );
      },
    );
  }
}

/// PublicFreelanceSocialLinksWidget is the read-only variant used on
/// the `/freelancers/:id` public profile screen.
class PublicFreelanceSocialLinksWidget extends ConsumerWidget {
  const PublicFreelanceSocialLinksWidget({
    super.key,
    required this.organizationId,
  });

  final String organizationId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final linksAsync =
        ref.watch(publicFreelanceSocialLinksProvider(organizationId));

    return linksAsync.when(
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
      data: (raw) {
        final links = raw
            .map(
              (entry) => SocialLinkEntry(
                platform: (entry['platform'] as String?) ?? '',
                url: (entry['url'] as String?) ?? '',
              ),
            )
            .where((e) => e.platform.isNotEmpty && e.url.isNotEmpty)
            .toList();
        return SocialLinksCard(links: links);
      },
    );
  }
}
