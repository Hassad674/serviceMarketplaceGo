import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:video_player/video_player.dart';

import '../../../../core/models/review.dart';
import '../../../../core/network/api_exception.dart';
import '../../../../l10n/app_localizations.dart';
import '../providers/review_provider.dart';
import '../../../../core/theme/app_palette.dart';

/// Bottom sheet for leaving a review after a completed mission.
///
/// Supports both review directions (client -> provider and
/// provider -> client). When [side] is [ReviewSide.providerToClient] the
/// three sub-criteria rows (timeliness / communication / quality) are
/// hidden — providers only rate clients on the global axis plus comment
/// and optional video.
class ReviewBottomSheet extends ConsumerStatefulWidget {
  final String proposalId;
  final String proposalTitle;
  final String side;
  final VoidCallback? onSubmitted;

  const ReviewBottomSheet({
    super.key,
    required this.proposalId,
    required this.proposalTitle,
    this.side = ReviewSide.clientToProvider,
    this.onSubmitted,
  });

  static Future<void> show(
    BuildContext context, {
    required String proposalId,
    required String proposalTitle,
    String side = ReviewSide.clientToProvider,
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
        side: side,
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
  bool _titleVisible = true;

  bool get _isProviderSide => widget.side == ReviewSide.providerToClient;

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
      // Provider-side reviews intentionally omit the three sub-criteria —
      // providers only evaluate clients on the global axis (plus comment
      // and optional video).
      await repo.createReview(
        proposalId: widget.proposalId,
        globalRating: _globalRating,
        timeliness: _isProviderSide || _timeliness == 0 ? null : _timeliness,
        communication:
            _isProviderSide || _communication == 0 ? null : _communication,
        quality: _isProviderSide || _quality == 0 ? null : _quality,
        comment: _commentController.text.trim(),
        videoUrl: _videoUrl,
        titleVisible: _titleVisible,
      );
      if (mounted) {
        Navigator.of(context).pop();
        widget.onSubmitted?.call();
      }
    } catch (error) {
      if (!mounted) return;
      setState(() => _isSubmitting = false);
      _showSubmitError(error);
    }
  }

  /// Surfaces backend errors as a snack bar.
  ///
  /// Known domain errors (`review_window_closed`, `not_participant`) get
  /// their own localized message. Everything else falls through to the
  /// generic helper so the user is never shown a raw exception string.
  void _showSubmitError(Object error) {
    final l10n = AppLocalizations.of(context)!;
    final apiError = error is DioException
        ? ApiException.fromDioException(error)
        : (error is ApiException ? error : null);

    String message;
    switch (apiError?.code) {
      case 'review_window_closed':
        message = l10n.reviewErrorWindowClosed;
        break;
      case 'not_participant':
        message = l10n.reviewErrorNotParticipant;
        break;
      default:
        message = apiError?.localizedMessage(context) ?? l10n.unexpectedError;
    }

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message)),
    );
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
            if (!_isProviderSide) ...[
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
            ],
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
            _buildTitleVisibilityToggle(theme),
            const SizedBox(height: 16),
            _buildActions(theme),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }

  Widget _buildTitleVisibilityToggle(ThemeData theme) {
    return InkWell(
      onTap: _isSubmitting
          ? null
          : () => setState(() => _titleVisible = !_titleVisible),
      borderRadius: BorderRadius.circular(12),
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: theme.dividerColor),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Checkbox(
              value: _titleVisible,
              onChanged: _isSubmitting
                  ? null
                  : (v) => setState(() => _titleVisible = v ?? true),
              materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Show the mission title on the provider\'s public profile',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    'Uncheck to keep the mission title private. Your rating and comment will still be visible.',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildHeader(ThemeData theme) {
    final l10n = AppLocalizations.of(context)!;
    final title = _isProviderSide
        ? l10n.reviewTitleProviderToClient
        : l10n.reviewTitleClientToProvider;
    // For the provider side we replace the proposal title subtitle with a
    // question-shaped prompt — the mission title alone is redundant when
    // the provider is reviewing the client, not the other way around.
    final subtitle = _isProviderSide
        ? l10n.reviewSubtitleProviderToClient
        : widget.proposalTitle;

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
        Text(title, style: theme.textTheme.titleLarge),
        const SizedBox(height: 4),
        Text(subtitle, style: theme.textTheme.bodySmall),
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
              color: AppPalette.amber400,
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
