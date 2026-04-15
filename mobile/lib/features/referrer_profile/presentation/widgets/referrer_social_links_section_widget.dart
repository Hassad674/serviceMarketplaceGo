import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../shared/widgets/social_links_card.dart';
import '../providers/referrer_social_links_providers.dart';

/// ReferrerSocialLinksSectionWidget mounts the shared social-links
/// card on the owner's referrer profile screen. The referrer
/// persona keeps its own set independent from the freelance one.
class ReferrerSocialLinksSectionWidget extends ConsumerWidget {
  const ReferrerSocialLinksSectionWidget({
    super.key,
    required this.canEdit,
  });

  final bool canEdit;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final linksAsync = ref.watch(referrerSocialLinksProvider);
    final repo = ref.read(referrerSocialLinksRepositoryProvider);

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
                    ref.invalidate(referrerSocialLinksProvider);
                  },
                  onDelete: (platform) async {
                    await repo.delete(platform);
                    ref.invalidate(referrerSocialLinksProvider);
                  },
                )
              : null,
        );
      },
    );
  }
}

/// PublicReferrerSocialLinksWidget is the read-only variant used on
/// the `/referrers/:id` public profile screen.
class PublicReferrerSocialLinksWidget extends ConsumerWidget {
  const PublicReferrerSocialLinksWidget({
    super.key,
    required this.organizationId,
  });

  final String organizationId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final linksAsync =
        ref.watch(publicReferrerSocialLinksProvider(organizationId));

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
