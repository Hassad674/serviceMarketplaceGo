import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/theme/app_theme.dart';
import '../../l10n/app_localizations.dart';

/// Shape of a single link passed to the card. Kept intentionally
/// light so each persona feature can re-use the widget without
/// committing to a specific DTO/entity class.
class SocialLinkEntry {
  const SocialLinkEntry({required this.platform, required this.url});

  final String platform;
  final String url;
}

/// Editor wiring the parent injects when the card should be
/// editable. The card never knows about Riverpod providers or
/// repositories — it just fires callbacks.
class SocialLinksEditorConfig {
  const SocialLinksEditorConfig({
    required this.onUpsert,
    required this.onDelete,
  });

  final Future<void> Function(String platform, String url) onUpsert;
  final Future<void> Function(String platform) onDelete;
}

class _PlatformMeta {
  const _PlatformMeta(this.key, this.icon, this.color);
  final String key;
  final IconData icon;
  final Color color;
}

const _platforms = <_PlatformMeta>[
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

/// Shared social-links card used by both personas (freelance and
/// referrer) in mobile. Collapses to nothing in read-only mode when
/// the set is empty, matching the web behaviour.
class SocialLinksCard extends StatelessWidget {
  const SocialLinksCard({
    super.key,
    required this.links,
    this.isLoading = false,
    this.editor,
  });

  final List<SocialLinkEntry> links;
  final bool isLoading;
  final SocialLinksEditorConfig? editor;

  bool get _canEdit => editor != null;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    if (isLoading) {
      return _SkeletonCard();
    }

    if (!_canEdit && links.isEmpty) {
      return const SizedBox.shrink();
    }

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
              Icon(
                Icons.share_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  l10n.socialLinks,
                  style: theme.textTheme.titleMedium,
                ),
              ),
              if (_canEdit)
                IconButton(
                  icon: const Icon(Icons.edit_outlined, size: 20),
                  tooltip: l10n.editSocialLinks,
                  onPressed: () => _openEditor(context),
                ),
            ],
          ),
          const SizedBox(height: 12),
          if (links.isEmpty)
            _EmptyState(label: l10n.noSocialLinks)
          else
            _SocialLinksList(links: links),
        ],
      ),
    );
  }

  void _openEditor(BuildContext context) {
    final initial = <String, String>{
      for (final link in links) link.platform: link.url,
    };
    final editorConfig = editor;
    if (editorConfig == null) return;
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _SocialLinksEditorSheet(
        initial: initial,
        editor: editorConfig,
      ),
    );
  }
}

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
  final List<SocialLinkEntry> links;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: links.map((link) {
        final meta = _metaForPlatform(link.platform);
        if (meta == null) return const SizedBox.shrink();
        return _SocialLinkTile(meta: meta, url: link.url);
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

class _SkeletonCard extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: const Padding(
        padding: EdgeInsets.symmetric(vertical: 12),
        child: LinearProgressIndicator(),
      ),
    );
  }
}

class _SocialLinksEditorSheet extends StatefulWidget {
  const _SocialLinksEditorSheet({
    required this.initial,
    required this.editor,
  });

  final Map<String, String> initial;
  final SocialLinksEditorConfig editor;

  @override
  State<_SocialLinksEditorSheet> createState() =>
      _SocialLinksEditorSheetState();
}

class _SocialLinksEditorSheetState extends State<_SocialLinksEditorSheet> {
  late final Map<String, TextEditingController> _controllers;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _controllers = <String, TextEditingController>{
      for (final meta in _platforms)
        meta.key:
            TextEditingController(text: widget.initial[meta.key] ?? ''),
    };
  }

  @override
  void dispose() {
    for (final controller in _controllers.values) {
      controller.dispose();
    }
    super.dispose();
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    try {
      for (final meta in _platforms) {
        final next = _controllers[meta.key]!.text.trim();
        final had = (widget.initial[meta.key] ?? '').isNotEmpty;
        if (next.isNotEmpty) {
          await widget.editor.onUpsert(meta.key, next);
        } else if (had) {
          await widget.editor.onDelete(meta.key);
        }
      }
      if (!mounted) return;
      Navigator.pop(context);
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.socialLinksSaved)),
      );
    } catch (_) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.socialLinksSaveError)),
      );
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
              _EditorField(meta: meta, controller: _controllers[meta.key]!),
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
