import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../domain/entities/portfolio_item.dart';
import '../../domain/repositories/portfolio_repository.dart';
import '../providers/portfolio_provider.dart';
import 'portfolio_form_chrome.dart';
import 'portfolio_form_footer.dart';
import 'portfolio_form_media.dart';
import 'portfolio_form_text_fields.dart';

/// Bottom sheet for creating or editing a portfolio item.
///
/// Owns form state (controllers + media drafts + saving flag) and composes
/// stateless sub-widgets for each section.
class PortfolioFormSheet extends ConsumerStatefulWidget {
  const PortfolioFormSheet({
    super.key,
    required this.orgId,
    this.item,
    required this.nextPosition,
  });

  final String orgId;
  final PortfolioItem? item;
  final int nextPosition;

  @override
  ConsumerState<PortfolioFormSheet> createState() => _PortfolioFormSheetState();
}

class _PortfolioFormSheetState extends ConsumerState<PortfolioFormSheet> {
  late final TextEditingController _titleController;
  late final TextEditingController _descController;
  late final TextEditingController _linkController;
  late final List<PortfolioMediaDraft> _media;
  bool _uploadingMedia = false;
  bool _saving = false;

  bool get _isEdit => widget.item != null;
  PortfolioRepository get _repo => ref.read(portfolioRepositoryProvider);

  @override
  void initState() {
    super.initState();
    _titleController = TextEditingController(text: widget.item?.title ?? '');
    _descController =
        TextEditingController(text: widget.item?.description ?? '');
    _linkController = TextEditingController(text: widget.item?.linkUrl ?? '');
    _media = widget.item?.media
            .map(
              (m) => PortfolioMediaDraft(
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

  // -- Media picking + upload ------------------------------------------------

  Future<void> _pickAndUploadImage() async {
    if (_media.length >= kPortfolioMaxMedia) return;
    final picked = await ImagePicker().pickImage(
      source: ImageSource.gallery,
      maxWidth: 2400,
      imageQuality: 85,
    );
    if (picked == null) return;
    await _uploadMedia(File(picked.path), 'image');
  }

  Future<void> _pickAndUploadVideo() async {
    if (_media.length >= kPortfolioMaxMedia) return;
    final result = await FilePicker.platform.pickFiles(
      type: FileType.video,
      allowMultiple: false,
    );
    final path = result?.files.single.path;
    if (path == null) return;
    await _uploadMedia(File(path), 'video');
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
          PortfolioMediaDraft(
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
    final picked = await ImagePicker().pickImage(
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

  void _revertCustomThumbnail(int idx) =>
      setState(() => _media[idx].thumbnailUrl = '');

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

    final result = _isEdit
        ? await notifier.updateItem(
            orgId: widget.orgId,
            id: widget.item!.id,
            title: title,
            description: _descController.text.trim(),
            linkUrl: _normalizeUrl(_linkController.text),
            media: mediaPayload,
          )
        : await notifier.createItem(
            orgId: widget.orgId,
            title: title,
            description: _descController.text.trim(),
            linkUrl: _normalizeUrl(_linkController.text),
            position: widget.nextPosition,
            media: mediaPayload,
          );

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

  void _showAddMediaSheet() {
    showModalBottomSheet(
      context: context,
      builder: (_) => PortfolioAddMediaSheet(
        onPickImage: _pickAndUploadImage,
        onPickVideo: _pickAndUploadVideo,
      ),
    );
  }

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
              PortfolioFormChrome(
                isEdit: _isEdit,
                onClose: () => Navigator.of(context).pop(),
              ),
              const Divider(height: 1),
              Expanded(
                child: ListView(
                  controller: scrollController,
                  padding: const EdgeInsets.fromLTRB(20, 16, 20, 16),
                  children: [
                    PortfolioFormTitleField(
                      controller: _titleController,
                      onChanged: () => setState(() {}),
                    ),
                    const SizedBox(height: 20),
                    PortfolioFormMediaSection(
                      media: _media,
                      uploadingMedia: _uploadingMedia,
                      onShowAddSheet: _showAddMediaSheet,
                      onRemoveMedia: _removeMedia,
                      onPickCustomThumbnail: _pickCustomThumbnail,
                      onRevertCustomThumbnail: _revertCustomThumbnail,
                    ),
                    const SizedBox(height: 20),
                    PortfolioFormDescriptionField(
                      controller: _descController,
                      onChanged: () => setState(() {}),
                    ),
                    const SizedBox(height: 20),
                    PortfolioFormLinkField(controller: _linkController),
                    const SizedBox(height: 24),
                  ],
                ),
              ),
              PortfolioFormFooter(
                isEdit: _isEdit,
                saving: _saving,
                canSave: _titleController.text.trim().isNotEmpty,
                onCancel: () => Navigator.of(context).pop(),
                onSave: _save,
              ),
            ],
          ),
        );
      },
    );
  }
}
