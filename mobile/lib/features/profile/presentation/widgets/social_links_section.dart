import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../providers/social_links_provider.dart';

/// Platform metadata used to render icons and labels.
class _PlatformMeta {
  const _PlatformMeta(this.key, this.icon, this.color);
  final String key;
  final IconData icon;
  final Color color;
}

const _platforms = [
  _PlatformMeta('linkedin', Icons.business, Color(0xFF0A66C2)),
  _PlatformMeta('instagram', Icons.camera_alt, Color(0xFFE4405F)),
  _PlatformMeta('youtube', Icons.play_circle_fill, Color(0xFFFF0000)),
  _PlatformMeta('twitter', Icons.alternate_email, Color(0xFF1DA1F2)),
  _PlatformMeta('github', Icons.code, Color(0xFF333333)),
  _PlatformMeta('website', Icons.language, Color(0xFFF43F5E)),
];

String _platformLabel(String key, AppLocalizations l10n) {
  switch (key) {
    case 'linkedin':
      return l10n.socialLinkLinkedin;
    case 'instagram':
      return l10n.socialLinkInstagram;
    case 'youtube':
      return l10n.socialLinkYoutube;
    case 'twitter':
      return l10n.socialLinkTwitter;
    case 'github':
      return l10n.socialLinkGithub;
    case 'website':
      return l10n.socialLinkWebsite;
    default:
      return key;
  }
}

_PlatformMeta? _metaForPlatform(String key) {
  for (final meta in _platforms) {
    if (meta.key == key) return meta;
  }
  return null;
}

/// Displays social links for the authenticated user's own profile.
/// Includes an edit button that opens a bottom sheet editor.
class SocialLinksSection extends ConsumerWidget {
  const SocialLinksSection({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final linksAsync = ref.watch(socialLinksProvider);
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.share_outlined, size: 20, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Expanded(
                child: Text(l10n.socialLinks, style: theme.textTheme.titleMedium),
              ),
              IconButton(
                icon: const Icon(Icons.edit_outlined, size: 20),
                onPressed: () => _openEditor(context, ref),
                tooltip: l10n.editSocialLinks,
              ),
            ],
          ),
          const SizedBox(height: 12),
          linksAsync.when(
            data: (links) => links.isEmpty
                ? _EmptyState(label: l10n.noSocialLinks)
                : _SocialLinksList(links: links),
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (_, __) => _EmptyState(label: l10n.noSocialLinks),
          ),
        ],
      ),
    );
  }

  void _openEditor(BuildContext context, WidgetRef ref) {
    final linksAsync = ref.read(socialLinksProvider);
    final existing = linksAsync.valueOrNull ?? [];
    final initial = <String, String>{};
    for (final link in existing) {
      initial[link['platform'] as String] = link['url'] as String;
    }

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _SocialLinksEditor(
        initial: initial,
        onSaved: () => ref.invalidate(socialLinksProvider),
      ),
    );
  }
}

/// Read-only display of social links for a public profile.
class PublicSocialLinksSection extends ConsumerWidget {
  const PublicSocialLinksSection({super.key, required this.userId});

  final String userId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final linksAsync = ref.watch(publicSocialLinksProvider(userId));
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return linksAsync.when(
      data: (links) {
        if (links.isEmpty) return const SizedBox.shrink();
        return Container(
          width: double.infinity,
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(Icons.share_outlined, size: 20, color: theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Text(l10n.socialLinks, style: theme.textTheme.titleMedium),
                ],
              ),
              const SizedBox(height: 12),
              _SocialLinksList(links: links),
            ],
          ),
        );
      },
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
    );
  }
}

// ---------------------------------------------------------------------------
// Private widgets
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) {
    final appColors = Theme.of(context).extension<AppColors>();
    return Text(
      label,
      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
        color: appColors?.mutedForeground,
        fontStyle: FontStyle.italic,
      ),
    );
  }
}

class _SocialLinksList extends StatelessWidget {
  const _SocialLinksList({required this.links});
  final List<Map<String, dynamic>> links;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: links.map((link) {
        final platform = link['platform'] as String;
        final url = link['url'] as String;
        final meta = _metaForPlatform(platform);
        if (meta == null) return const SizedBox.shrink();
        return _SocialLinkTile(meta: meta, url: url);
      }).toList(),
    );
  }
}

class _SocialLinkTile extends StatelessWidget {
  const _SocialLinkTile({required this.meta, required this.url});
  final _PlatformMeta meta;
  final String url;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return InkWell(
      onTap: () => _launchUrl(url),
      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 4),
        child: Row(
          children: [
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: meta.color.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(AppTheme.radiusMd),
              ),
              child: Icon(meta.icon, size: 20, color: meta.color),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    _platformLabel(meta.key, l10n),
                    style: theme.textTheme.bodyMedium?.copyWith(
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  Text(
                    url.replaceAll(RegExp(r'(^\w+:|^)//'), ''),
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.extension<AppColors>()?.mutedForeground,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            Icon(
              Icons.open_in_new,
              size: 16,
              color: theme.extension<AppColors>()?.mutedForeground,
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _launchUrl(String rawUrl) async {
    final uri = Uri.tryParse(rawUrl);
    if (uri != null && await canLaunchUrl(uri)) {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    }
  }
}

// ---------------------------------------------------------------------------
// Editor bottom sheet
// ---------------------------------------------------------------------------

class _SocialLinksEditor extends ConsumerStatefulWidget {
  const _SocialLinksEditor({required this.initial, required this.onSaved});
  final Map<String, String> initial;
  final VoidCallback onSaved;

  @override
  ConsumerState<_SocialLinksEditor> createState() => _SocialLinksEditorState();
}

class _SocialLinksEditorState extends ConsumerState<_SocialLinksEditor> {
  late final Map<String, TextEditingController> _controllers;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _controllers = {
      for (final meta in _platforms)
        meta.key: TextEditingController(text: widget.initial[meta.key] ?? ''),
    };
  }

  @override
  void dispose() {
    for (final c in _controllers.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final api = ref.read(apiClientProvider);
      for (final meta in _platforms) {
        final url = _controllers[meta.key]!.text.trim();
        final hadBefore = (widget.initial[meta.key] ?? '').isNotEmpty;

        if (url.isNotEmpty) {
          await api.put(
            '/api/v1/profile/social-links',
            data: {'platform': meta.key, 'url': url},
          );
        } else if (hadBefore) {
          await api.delete('/api/v1/profile/social-links/${meta.key}');
        }
      }
      widget.onSaved();
      if (mounted) {
        final l10n = AppLocalizations.of(context)!;
        Navigator.pop(context);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.socialLinksSaved)),
        );
      }
    } catch (_) {
      if (mounted) {
        final l10n = AppLocalizations.of(context)!;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.socialLinksSaveError)),
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.editSocialLinks,
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 20),
            for (final meta in _platforms) ...[
              _EditorField(
                meta: meta,
                controller: _controllers[meta.key]!,
              ),
              const SizedBox(height: 12),
            ],
            const SizedBox(height: 8),
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: _saving ? null : _save,
                child: _saving
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : Text(l10n.save),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _EditorField extends StatelessWidget {
  const _EditorField({required this.meta, required this.controller});
  final _PlatformMeta meta;
  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return TextField(
      controller: controller,
      keyboardType: TextInputType.url,
      decoration: InputDecoration(
        labelText: _platformLabel(meta.key, l10n),
        hintText: l10n.socialLinkEnterUrl,
        prefixIcon: Icon(meta.icon, color: meta.color, size: 20),
        border: const OutlineInputBorder(),
      ),
    );
  }
}
