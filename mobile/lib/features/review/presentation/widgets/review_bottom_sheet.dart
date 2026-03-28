import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:video_player/video_player.dart';

import '../providers/review_provider.dart';

/// Bottom sheet for leaving a review after a completed mission.
class ReviewBottomSheet extends ConsumerStatefulWidget {
  final String proposalId;
  final String proposalTitle;
  final VoidCallback? onSubmitted;

  const ReviewBottomSheet({
    super.key,
    required this.proposalId,
    required this.proposalTitle,
    this.onSubmitted,
  });

  static Future<void> show(
    BuildContext context, {
    required String proposalId,
    required String proposalTitle,
    VoidCallback? onSubmitted,
  }) {
    return showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (_) => ReviewBottomSheet(
        proposalId: proposalId,
        proposalTitle: proposalTitle,
        onSubmitted: onSubmitted,
      ),
    );
  }

  @override
  ConsumerState<ReviewBottomSheet> createState() => _ReviewBottomSheetState();
}

class _ReviewBottomSheetState extends ConsumerState<ReviewBottomSheet> {
  int _globalRating = 0;
  int _timeliness = 0;
  int _communication = 0;
  int _quality = 0;
  final _commentController = TextEditingController();
  bool _isSubmitting = false;
  bool _isUploadingVideo = false;
  String? _videoUrl;
  VideoPlayerController? _videoController;

  @override
  void dispose() {
    _commentController.dispose();
    _videoController?.dispose();
    super.dispose();
  }

  Future<void> _pickVideo() async {
    final picker = ImagePicker();
    final video = await picker.pickVideo(
      source: ImageSource.gallery,
      maxDuration: const Duration(minutes: 5),
    );
    if (video == null) return;

    setState(() {
      _isUploadingVideo = true;
    });

    try {
      final repo = ref.read(reviewRepositoryProvider);
      final url = await repo.uploadReviewVideo(video.path);
      if (mounted) {
        _initVideoPreview(video.path);
        setState(() {
          _videoUrl = url;
          _isUploadingVideo = false;
        });
      }
    } catch (_) {
      if (mounted) {
        setState(() => _isUploadingVideo = false);
      }
    }
  }

  void _initVideoPreview(String path) {
    _videoController?.dispose();
    _videoController = VideoPlayerController.file(File(path))
      ..initialize().then((_) {
        if (mounted) setState(() {});
      });
  }

  void _removeVideo() {
    _videoController?.dispose();
    _videoController = null;
    setState(() => _videoUrl = null);
  }

  Future<void> _submit() async {
    if (_globalRating == 0) return;
    setState(() => _isSubmitting = true);

    try {
      final repo = ref.read(reviewRepositoryProvider);
      await repo.createReview(
        proposalId: widget.proposalId,
        globalRating: _globalRating,
        timeliness: _timeliness > 0 ? _timeliness : null,
        communication: _communication > 0 ? _communication : null,
        quality: _quality > 0 ? _quality : null,
        comment: _commentController.text.trim(),
        videoUrl: _videoUrl,
      );
      if (mounted) {
        Navigator.of(context).pop();
        widget.onSubmitted?.call();
      }
    } catch (_) {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
        left: 20,
        right: 20,
        top: 20,
      ),
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildHeader(theme),
            const SizedBox(height: 20),
            _buildStarRow('Overall rating *', _globalRating, (v) {
              setState(() => _globalRating = v);
            }),
            const Divider(height: 32),
            Text(
              'Detailed criteria (optional)',
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            _buildStarRow('Timeliness', _timeliness, (v) {
              setState(() => _timeliness = v);
            }),
            const SizedBox(height: 8),
            _buildStarRow('Communication', _communication, (v) {
              setState(() => _communication = v);
            }),
            const SizedBox(height: 8),
            _buildStarRow('Quality', _quality, (v) {
              setState(() => _quality = v);
            }),
            const SizedBox(height: 20),
            TextField(
              controller: _commentController,
              maxLines: 3,
              maxLength: 2000,
              decoration: const InputDecoration(
                labelText: 'Written review',
                hintText: 'Describe your experience...',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            _buildVideoSection(theme),
            const SizedBox(height: 16),
            _buildActions(theme),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }

  Widget _buildHeader(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Center(
          child: Container(
            width: 40,
            height: 4,
            decoration: BoxDecoration(
              color: theme.dividerColor,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
        ),
        const SizedBox(height: 16),
        Text('Leave a review', style: theme.textTheme.titleLarge),
        const SizedBox(height: 4),
        Text(widget.proposalTitle, style: theme.textTheme.bodySmall),
      ],
    );
  }

  Widget _buildStarRow(String label, int value, ValueChanged<int> onChanged) {
    return Row(
      children: [
        Expanded(
          child: Text(label, style: Theme.of(context).textTheme.bodyMedium),
        ),
        for (int i = 1; i <= 5; i++)
          GestureDetector(
            onTap: () => onChanged(i),
            child: Icon(
              i <= value ? Icons.star : Icons.star_border,
              color: const Color(0xFFFBBF24),
              size: 28,
            ),
          ),
      ],
    );
  }

  Widget _buildVideoSection(ThemeData theme) {
    if (_videoUrl != null && _videoController != null) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          ClipRRect(
            borderRadius: BorderRadius.circular(8),
            child: AspectRatio(
              aspectRatio: _videoController!.value.isInitialized
                  ? _videoController!.value.aspectRatio
                  : 16 / 9,
              child: VideoPlayer(_videoController!),
            ),
          ),
          const SizedBox(height: 8),
          TextButton.icon(
            onPressed: _isSubmitting ? null : _removeVideo,
            icon: const Icon(Icons.delete_outline, size: 18),
            label: const Text('Remove video'),
            style: TextButton.styleFrom(
              foregroundColor: theme.colorScheme.error,
            ),
          ),
        ],
      );
    }

    return OutlinedButton.icon(
      onPressed: (_isSubmitting || _isUploadingVideo) ? null : _pickVideo,
      icon: _isUploadingVideo
          ? const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.videocam_outlined, size: 20),
      label: Text(_isUploadingVideo ? 'Uploading video...' : 'Add a video'),
    );
  }

  Widget _buildActions(ThemeData theme) {
    final isBusy = _isSubmitting || _isUploadingVideo;
    return Row(
      children: [
        Expanded(
          child: FilledButton(
            onPressed: (_globalRating == 0 || isBusy) ? null : _submit,
            child: _isSubmitting
                ? const SizedBox(
                    height: 20,
                    width: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Text('Submit review'),
          ),
        ),
        const SizedBox(width: 12),
        OutlinedButton(
          onPressed: isBusy ? null : () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
      ],
    );
  }
}
