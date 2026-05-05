import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:dio/dio.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../providers/job_provider.dart';

void showApplyBottomSheet(BuildContext context, WidgetRef ref, String jobId) {
  showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    backgroundColor: Theme.of(context).colorScheme.surfaceContainerLowest,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => _ApplyForm(jobId: jobId),
  );
}

/// W-13 mobile · Apply form bottom sheet — Soleil v2.
///
/// Behaviour preserved exactly: video upload, message length cap,
/// applyToJobAction, status code → user-friendly message. Only the
/// chrome is updated (Fraunces title, corail FilledButton with
/// StadiumBorder, Soleil-aware progress bar).
class _ApplyForm extends ConsumerStatefulWidget {
  const _ApplyForm({required this.jobId});

  final String jobId;

  @override
  ConsumerState<_ApplyForm> createState() => _ApplyFormState();
}

class _ApplyFormState extends ConsumerState<_ApplyForm> {
  final _messageController = TextEditingController();
  bool _isSubmitting = false;
  String? _videoUrl;
  bool _isUploading = false;
  double _uploadProgress = 0;
  int _messageLength = 0;

  @override
  void initState() {
    super.initState();
    _messageController.addListener(_onMessageChanged);
  }

  void _onMessageChanged() {
    setState(() => _messageLength = _messageController.text.length);
  }

  @override
  void dispose() {
    _messageController.removeListener(_onMessageChanged);
    _messageController.dispose();
    super.dispose();
  }

  Future<void> _pickVideo() async {
    final picker = ImagePicker();
    final file = await picker.pickVideo(source: ImageSource.gallery);
    if (file == null) return;

    setState(() {
      _isUploading = true;
      _uploadProgress = 0;
    });
    try {
      final apiClient = ref.read(apiClientProvider);
      final formData = FormData.fromMap({
        'file': await MultipartFile.fromFile(file.path, filename: file.name),
      });
      final response = await apiClient.upload(
        '/api/v1/upload/video',
        data: formData,
        onSendProgress: (sent, total) {
          if (mounted && total > 0) {
            setState(() => _uploadProgress = sent / total);
          }
        },
      );
      final url = response.data?['url'] as String?;
      if (url != null) setState(() => _videoUrl = url);
    } catch (e) {
      debugPrint('[ApplyBottomSheet] video upload error: $e');
      if (mounted) {
        final l10n = AppLocalizations.of(context)!;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(l10n.videoUploadFailed),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    } finally {
      setState(() => _isUploading = false);
    }
  }

  void _removeVideo() {
    setState(() => _videoUrl = null);
  }

  Future<void> _submit() async {
    setState(() => _isSubmitting = true);
    final result = await applyToJobAction(
      ref,
      widget.jobId,
      message: _messageController.text.trim(),
      videoUrl: _videoUrl,
    );
    setState(() => _isSubmitting = false);

    if (!mounted) return;
    Navigator.pop(context);

    final l10n = AppLocalizations.of(context)!;
    final cs = Theme.of(context).colorScheme;
    if (result.success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(l10n.applicationSent),
          backgroundColor: cs.primary,
        ),
      );
    } else {
      final msg = switch (result.statusCode) {
        403 => l10n.applicantTypeMismatch,
        409 => l10n.alreadyApplied,
        _ => l10n.applicationSendError,
      };
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(msg), backgroundColor: cs.error),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final cs = Theme.of(context).colorScheme;
    final soleil = Theme.of(context).extension<AppColors>()!;

    return Padding(
      padding: EdgeInsets.only(
        left: 20,
        right: 20,
        top: 20,
        bottom: MediaQuery.of(context).viewInsets.bottom + 20,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              width: 40,
              height: 4,
              decoration: BoxDecoration(
                color: cs.outline,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ),
          const SizedBox(height: 20),
          Text(
            l10n.applyTitle,
            style: SoleilTextStyles.headlineMedium.copyWith(
              color: cs.onSurface,
              fontSize: 22,
            ),
          ),
          const SizedBox(height: 16),

          // Message (optional)
          TextField(
            controller: _messageController,
            maxLines: 5,
            maxLength: 5000,
            buildCounter:
                (
                  context, {
                  required currentLength,
                  required isFocused,
                  required maxLength,
                }) => null,
            decoration: InputDecoration(
              labelText: l10n.applyMessageLabel,
              hintText: l10n.applyMessageHint,
              alignLabelWithHint: true,
            ),
          ),
          // Character counter
          Align(
            alignment: Alignment.centerRight,
            child: Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                '$_messageLength/5000',
                style: SoleilTextStyles.mono.copyWith(
                  color: soleil.subtleForeground,
                  fontSize: 11,
                ),
              ),
            ),
          ),
          const SizedBox(height: 16),

          // Video upload (optional)
          if (_videoUrl == null && !_isUploading)
            OutlinedButton.icon(
              onPressed: _pickVideo,
              icon: const Icon(Icons.videocam_rounded),
              label: Text(l10n.applyAddVideo),
              style: OutlinedButton.styleFrom(
                minimumSize: const Size.fromHeight(48),
                shape: const StadiumBorder(),
                side: BorderSide(color: soleil.borderStrong),
                foregroundColor: cs.onSurface,
              ),
            ),
          if (_isUploading)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Column(
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(
                        l10n.applyUploading,
                        style: SoleilTextStyles.body.copyWith(
                          color: cs.onSurface,
                        ),
                      ),
                      Text(
                        l10n.uploadProgress(
                          (_uploadProgress * 100).round(),
                        ),
                        style: SoleilTextStyles.bodyEmphasis.copyWith(
                          color: cs.primary,
                          fontSize: 12,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  ClipRRect(
                    borderRadius: BorderRadius.circular(4),
                    child: LinearProgressIndicator(
                      value: _uploadProgress,
                      minHeight: 6,
                      backgroundColor: soleil.accentSoft,
                      valueColor: AlwaysStoppedAnimation<Color>(cs.primary),
                    ),
                  ),
                ],
              ),
            ),
          if (_videoUrl != null) ...[
            // Video player preview
            ClipRRect(
              borderRadius: BorderRadius.circular(AppTheme.radiusLg),
              child: SizedBox(
                height: 200,
                width: double.infinity,
                child: VideoPlayerWidget(videoUrl: _videoUrl!),
              ),
            ),
            const SizedBox(height: 8),
            // Remove video button
            SizedBox(
              width: double.infinity,
              child: OutlinedButton.icon(
                onPressed: _removeVideo,
                icon: Icon(
                  Icons.delete_outline_rounded,
                  size: 18,
                  color: cs.error,
                ),
                label: Text(
                  l10n.applyRemoveVideo,
                  style: TextStyle(color: cs.error),
                ),
                style: OutlinedButton.styleFrom(
                  shape: const StadiumBorder(),
                  side: BorderSide(color: cs.error.withValues(alpha: 0.5)),
                ),
              ),
            ),
          ],
          const SizedBox(height: 16),

          // Submit — corail pill
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: (_isSubmitting || _isUploading) ? null : _submit,
              style: FilledButton.styleFrom(
                backgroundColor: cs.primary,
                foregroundColor: cs.onPrimary,
                disabledBackgroundColor: soleil.borderStrong,
                disabledForegroundColor: cs.onSurfaceVariant,
                minimumSize: const Size.fromHeight(48),
                shape: const StadiumBorder(),
                textStyle: SoleilTextStyles.button,
              ),
              child: _isSubmitting
                  ? SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: cs.onPrimary,
                      ),
                    )
                  : Text(l10n.applySubmit),
            ),
          ),
        ],
      ),
    );
  }
}
