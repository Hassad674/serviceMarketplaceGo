import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../domain/entities/portfolio_item.dart';
import '../../domain/repositories/portfolio_repository.dart';
import '../providers/portfolio_provider.dart';
import 'portfolio_video_thumbnail.dart';

const int _kMaxMedia = 8;
const int _kMaxTitleLen = 200;
const int _kMaxDescLen = 2000;

/// Internal local representation of a media item being edited.
class _LocalMedia {
  String mediaUrl;
  String mediaType; // 'image' or 'video'
  String thumbnailUrl;
  int position;

  _LocalMedia({
    required this.mediaUrl,
    required this.mediaType,
    this.thumbnailUrl = '',
    required this.position,
  });

  bool get isVideo => mediaType == 'video';
}

/// Bottom sheet for creating or editing a portfolio item.
class PortfolioFormSheet extends ConsumerStatefulWidget {
  const PortfolioFormSheet({
    super.key,
    required this.userId,
    this.item,
    required this.nextPosition,
  });

  final String userId;
  final PortfolioItem? item;
  final int nextPosition;

  @override
  ConsumerState<PortfolioFormSheet> createState() => _PortfolioFormSheetState();
}

class _PortfolioFormSheetState extends ConsumerState<PortfolioFormSheet> {
  late TextEditingController _titleController;
  late TextEditingController _descController;
  late TextEditingController _linkController;
  late List<_LocalMedia> _media;
  bool _uploadingMedia = false;
  bool _saving = false;

  bool get _isEdit => widget.item != null;

  @override
  void initState() {
    super.initState();
    _titleController = TextEditingController(text: widget.item?.title ?? '');
    _descController =
        TextEditingController(text: widget.item?.description ?? '');
    _linkController = TextEditingController(text: widget.item?.linkUrl ?? '');
    _media = widget.item?.media
            .map(
              (m) => _LocalMedia(
                mediaUrl: m.mediaUrl,
                mediaType: m.mediaType,
                thumbnailUrl: m.thumbnailUrl,
                position: m.position,
              ),
            )
            .toList() ??
        [];
  }

  @override
  void dispose() {
    _titleController.dispose();
    _descController.dispose();
    _linkController.dispose();
    super.dispose();
  }

  PortfolioRepository get _repo => ref.read(portfolioRepositoryProvider);

  // ---------------------------------------------------------------------------
  // Media picking + upload
  // ---------------------------------------------------------------------------

  Future<void> _pickAndUploadImage() async {
    if (_media.length >= _kMaxMedia) return;
    final picker = ImagePicker();
    final picked = await picker.pickImage(
      source: ImageSource.gallery,
      maxWidth: 2400,
      imageQuality: 85,
    );
    if (picked == null) return;
    await _uploadMedia(File(picked.path), 'image');
  }

  Future<void> _pickAndUploadVideo() async {
    if (_media.length >= _kMaxMedia) return;
    final result = await FilePicker.platform.pickFiles(
      type: FileType.video,
      allowMultiple: false,
    );
    if (result == null || result.files.single.path == null) return;
    await _uploadMedia(File(result.files.single.path!), 'video');
  }

  Future<void> _uploadMedia(File file, String type) async {
    setState(() => _uploadingMedia = true);
    try {
      final url = type == 'video'
          ? await _repo.uploadPortfolioVideo(file.path)
          : await _repo.uploadPortfolioImage(file.path);
      if (!mounted) return;
      setState(() {
        _media.add(
          _LocalMedia(
            mediaUrl: url,
            mediaType: type,
            position: _media.length,
          ),
        );
      });
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Upload failed: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _uploadingMedia = false);
    }
  }

  Future<void> _pickCustomThumbnail(int idx) async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(
      source: ImageSource.gallery,
      maxWidth: 1600,
      imageQuality: 85,
    );
    if (picked == null) return;
    setState(() => _uploadingMedia = true);
    try {
      final url = await _repo.uploadPortfolioImage(picked.path);
      if (!mounted) return;
      setState(() => _media[idx].thumbnailUrl = url);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Upload failed: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _uploadingMedia = false);
    }
  }

  void _removeMedia(int idx) {
    setState(() {
      _media.removeAt(idx);
      for (var i = 0; i < _media.length; i++) {
        _media[i].position = i;
      }
    });
  }

  void _revertCustomThumbnail(int idx) {
    setState(() => _media[idx].thumbnailUrl = '');
  }

  // ---------------------------------------------------------------------------
  // Save
  // ---------------------------------------------------------------------------

  String _normalizeUrl(String raw) {
    final t = raw.trim();
    if (t.isEmpty) return '';
    if (t.startsWith('http://') || t.startsWith('https://')) return t;
    return 'https://$t';
  }

  Future<void> _save() async {
    final title = _titleController.text.trim();
    if (title.isEmpty) return;

    setState(() => _saving = true);
    final notifier = ref.read(portfolioMutationProvider.notifier);
    final mediaPayload = _media
        .asMap()
        .entries
        .map(
          (e) => <String, dynamic>{
            'media_url': e.value.mediaUrl,
            'media_type': e.value.mediaType,
            if (e.value.thumbnailUrl.isNotEmpty)
              'thumbnail_url': e.value.thumbnailUrl,
            'position': e.key,
          },
        )
        .toList();

    PortfolioItem? result;
    if (_isEdit) {
      result = await notifier.updateItem(
        userId: widget.userId,
        id: widget.item!.id,
        title: title,
        description: _descController.text.trim(),
        linkUrl: _normalizeUrl(_linkController.text),
        media: mediaPayload,
      );
    } else {
      result = await notifier.createItem(
        userId: widget.userId,
        title: title,
        description: _descController.text.trim(),
        linkUrl: _normalizeUrl(_linkController.text),
        position: widget.nextPosition,
        media: mediaPayload,
      );
    }

    if (!mounted) return;
    setState(() => _saving = false);
    if (result != null) {
      Navigator.of(context).pop();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to save project')),
      );
    }
  }

  // ---------------------------------------------------------------------------
  // Build
  // ---------------------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return DraggableScrollableSheet(
      initialChildSize: 0.92,
      maxChildSize: 0.95,
      minChildSize: 0.5,
      builder: (context, scrollController) {
        return Container(
          decoration: BoxDecoration(
            color: theme.cardColor,
            borderRadius: const BorderRadius.vertical(top: Radius.circular(20)),
          ),
          child: Column(
            children: [
              // Drag handle
              Center(
                child: Container(
                  margin: const EdgeInsets.only(top: 12, bottom: 8),
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                    color: theme.dividerColor,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),

              // Header
              Padding(
                padding: const EdgeInsets.fromLTRB(20, 4, 12, 8),
                child: Row(
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            _isEdit ? 'Edit project' : 'Add project',
                            style: theme.textTheme.titleLarge?.copyWith(
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          Text(
                            _isEdit
                                ? 'Update your project details'
                                : 'Showcase a project with images, videos and a link',
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.close),
                      onPressed: () => Navigator.of(context).pop(),
                    ),
                  ],
                ),
              ),
              const Divider(height: 1),

              // Body
              Expanded(
                child: ListView(
                  controller: scrollController,
                  padding: const EdgeInsets.fromLTRB(20, 16, 20, 16),
                  children: [
                    _buildTitleField(theme),
                    const SizedBox(height: 20),
                    _buildMediaSection(theme),
                    const SizedBox(height: 20),
                    _buildDescriptionField(theme),
                    const SizedBox(height: 20),
                    _buildLinkField(theme),
                    const SizedBox(height: 24),
                  ],
                ),
              ),

              // Footer with save button
              SafeArea(
                top: false,
                child: Container(
                  padding: const EdgeInsets.fromLTRB(20, 12, 20, 12),
                  decoration: BoxDecoration(
                    border: Border(
                      top: BorderSide(color: theme.dividerColor),
                    ),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: TextButton(
                          onPressed: _saving
                              ? null
                              : () => Navigator.of(context).pop(),
                          child: const Text('Cancel'),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        flex: 2,
                        child: FilledButton(
                          onPressed: (_saving ||
                                  _titleController.text.trim().isEmpty)
                              ? null
                              : _save,
                          style: FilledButton.styleFrom(
                            backgroundColor: const Color(0xFFE11D48),
                            foregroundColor: Colors.white,
                            padding: const EdgeInsets.symmetric(vertical: 12),
                          ),
                          child: _saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                    color: Colors.white,
                                  ),
                                )
                              : Text(_isEdit ? 'Save changes' : 'Create'),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildTitleField(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(
              'Title *',
              style: theme.textTheme.labelLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const Spacer(),
            Text(
              '${_titleController.text.length}/$_kMaxTitleLen',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        TextField(
          controller: _titleController,
          maxLength: _kMaxTitleLen,
          onChanged: (_) => setState(() {}),
          decoration: InputDecoration(
            hintText: 'e.g. E-commerce Redesign for Nike',
            counterText: '',
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 14,
              vertical: 12,
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildMediaSection(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(
              'Media',
              style: theme.textTheme.labelLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const Spacer(),
            Text(
              '${_media.length}/$_kMaxMedia',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        if (_media.isEmpty)
          _buildEmptyMediaUploader(theme)
        else
          _buildMediaGrid(theme),
        if (_media.isNotEmpty) ...[
          const SizedBox(height: 6),
          Text(
            'The first media will be used as the cover.',
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ],
    );
  }

  Widget _buildEmptyMediaUploader(ThemeData theme) {
    return InkWell(
      onTap: _uploadingMedia ? null : _showAddMediaSheet,
      borderRadius: BorderRadius.circular(16),
      child: Container(
        height: 160,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: theme.colorScheme.outlineVariant,
            width: 2,
            style: BorderStyle.solid,
          ),
          color: theme.colorScheme.surfaceContainerLow,
        ),
        child: Center(
          child: _uploadingMedia
              ? const CircularProgressIndicator()
              : Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Container(
                      width: 56,
                      height: 56,
                      decoration: BoxDecoration(
                        gradient: const LinearGradient(
                          colors: [Color(0xFFFFE4E6), Color(0xFFFEF2F2)],
                        ),
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: const Icon(
                        Icons.add_photo_alternate_outlined,
                        color: Color(0xFFE11D48),
                        size: 26,
                      ),
                    ),
                    const SizedBox(height: 10),
                    Text(
                      'Tap to add images or videos',
                      style: theme.textTheme.bodyMedium?.copyWith(
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Up to $_kMaxMedia files',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
        ),
      ),
    );
  }

  Widget _buildMediaGrid(ThemeData theme) {
    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 3,
        crossAxisSpacing: 8,
        mainAxisSpacing: 8,
        childAspectRatio: 1,
      ),
      itemCount: _media.length + (_media.length < _kMaxMedia ? 1 : 0),
      itemBuilder: (context, index) {
        if (index == _media.length) {
          return _buildAddMediaButton(theme);
        }
        return _buildMediaThumb(theme, index);
      },
    );
  }

  Widget _buildAddMediaButton(ThemeData theme) {
    return InkWell(
      onTap: _uploadingMedia ? null : _showAddMediaSheet,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: theme.colorScheme.outlineVariant,
            width: 2,
          ),
          color: theme.colorScheme.surfaceContainerLow,
        ),
        child: Center(
          child: _uploadingMedia
              ? const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Icon(
                  Icons.add,
                  color: theme.colorScheme.onSurfaceVariant,
                  size: 28,
                ),
        ),
      ),
    );
  }

  Widget _buildMediaThumb(ThemeData theme, int index) {
    final m = _media[index];
    final isFirst = index == 0;

    return Stack(
      fit: StackFit.expand,
      children: [
        // Thumbnail content
        ClipRRect(
          borderRadius: BorderRadius.circular(12),
          child: Container(
            color: const Color(0xFF0F172A),
            child: m.isVideo
                ? (m.thumbnailUrl.isNotEmpty
                    ? Image.network(m.thumbnailUrl, fit: BoxFit.cover)
                    : PortfolioVideoThumbnail(videoUrl: m.mediaUrl))
                : Image.network(m.mediaUrl, fit: BoxFit.cover),
          ),
        ),

        // Border (cover ring on first)
        if (isFirst)
          Positioned.fill(
            child: IgnorePointer(
              child: Container(
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: const Color(0xFFE11D48),
                    width: 2,
                  ),
                ),
              ),
            ),
          ),

        // Cover badge
        if (isFirst)
          Positioned(
            top: 4,
            left: 4,
            child: Container(
              padding:
                  const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                gradient: const LinearGradient(
                  colors: [Color(0xFFF43F5E), Color(0xFFE11D48)],
                ),
                borderRadius: BorderRadius.circular(99),
              ),
              child: const Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.star, color: Colors.white, size: 10),
                  SizedBox(width: 2),
                  Text(
                    'Cover',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 9,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ],
              ),
            ),
          ),

        // Play icon overlay on videos
        if (m.isVideo)
          const Center(
            child: Icon(
              Icons.play_circle_fill,
              color: Colors.white70,
              size: 28,
            ),
          ),

        // Delete button (top-right)
        Positioned(
          top: 4,
          right: 4,
          child: GestureDetector(
            onTap: () => _removeMedia(index),
            child: Container(
              width: 22,
              height: 22,
              decoration: BoxDecoration(
                color: Colors.red.withValues(alpha: 0.9),
                shape: BoxShape.circle,
              ),
              child: const Icon(
                Icons.close,
                color: Colors.white,
                size: 14,
              ),
            ),
          ),
        ),

        // Custom thumbnail bar (videos only) - always visible at bottom
        if (m.isVideo)
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: GestureDetector(
              onTap: m.thumbnailUrl.isNotEmpty
                  ? () => _revertCustomThumbnail(index)
                  : () => _pickCustomThumbnail(index),
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 5),
                decoration: BoxDecoration(
                  color: m.thumbnailUrl.isNotEmpty
                      ? const Color(0xFFE11D48).withValues(alpha: 0.92)
                      : Colors.black.withValues(alpha: 0.7),
                  borderRadius: const BorderRadius.only(
                    bottomLeft: Radius.circular(12),
                    bottomRight: Radius.circular(12),
                  ),
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(
                      m.thumbnailUrl.isNotEmpty
                          ? Icons.refresh
                          : Icons.camera_alt_outlined,
                      color: Colors.white,
                      size: 11,
                    ),
                    const SizedBox(width: 3),
                    Text(
                      m.thumbnailUrl.isNotEmpty ? 'Custom' : 'Cover perso',
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 9,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
      ],
    );
  }

  void _showAddMediaSheet() {
    showModalBottomSheet(
      context: context,
      builder: (sheetContext) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.image_outlined),
              title: const Text('Add an image'),
              onTap: () {
                Navigator.of(sheetContext).pop();
                _pickAndUploadImage();
              },
            ),
            ListTile(
              leading: const Icon(Icons.videocam_outlined),
              title: const Text('Add a video'),
              onTap: () {
                Navigator.of(sheetContext).pop();
                _pickAndUploadVideo();
              },
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }

  Widget _buildDescriptionField(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(
              'Description',
              style: theme.textTheme.labelLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const Spacer(),
            Text(
              '${_descController.text.length}/$_kMaxDescLen',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        TextField(
          controller: _descController,
          maxLength: _kMaxDescLen,
          maxLines: 4,
          onChanged: (_) => setState(() {}),
          decoration: InputDecoration(
            hintText: 'Describe your role, the challenge and the results...',
            counterText: '',
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 14,
              vertical: 12,
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildLinkField(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Project link',
          style: theme.textTheme.labelLarge?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 6),
        TextField(
          controller: _linkController,
          keyboardType: TextInputType.url,
          decoration: InputDecoration(
            hintText: 'example.com',
            prefixIcon: const Icon(Icons.link, size: 18),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 14,
              vertical: 12,
            ),
          ),
        ),
        const SizedBox(height: 4),
        Text(
          "We'll automatically add https:// if you forget",
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}
