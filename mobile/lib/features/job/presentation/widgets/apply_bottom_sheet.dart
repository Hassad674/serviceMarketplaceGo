import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:dio/dio.dart';

import '../../../../core/network/api_client.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../providers/job_provider.dart';

void showApplyBottomSheet(BuildContext context, WidgetRef ref, String jobId) {
  showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
    builder: (_) => _ApplyForm(jobId: jobId),
  );
}

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
          SnackBar(content: Text(l10n.videoUploadFailed), backgroundColor: Colors.red),
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
    if (result != null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.applicationSent), backgroundColor: const Color(0xFFF43F5E)),
      );
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.applicationSendError), backgroundColor: Colors.red),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: EdgeInsets.only(
        left: 20, right: 20, top: 20,
        bottom: MediaQuery.of(context).viewInsets.bottom + 20,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(width: 40, height: 4, decoration: BoxDecoration(color: Colors.grey.shade300, borderRadius: BorderRadius.circular(2))),
          ),
          const SizedBox(height: 16),
          Text(l10n.applyTitle, style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          const SizedBox(height: 16),

          // Message (optional)
          TextField(
            controller: _messageController,
            maxLines: 5,
            maxLength: 5000,
            buildCounter: (context, {required currentLength, required isFocused, required maxLength}) => null,
            decoration: InputDecoration(
              labelText: l10n.applyMessageLabel,
              hintText: l10n.applyMessageHint,
              border: const OutlineInputBorder(),
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
                style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Colors.grey),
              ),
            ),
          ),
          const SizedBox(height: 16),

          // Video upload (optional)
          if (_videoUrl == null && !_isUploading)
            OutlinedButton.icon(
              onPressed: _pickVideo,
              icon: const Icon(Icons.videocam_outlined),
              label: Text(l10n.applyAddVideo),
              style: OutlinedButton.styleFrom(minimumSize: const Size.fromHeight(44)),
            ),
          if (_isUploading)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Column(
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(l10n.applyUploading),
                      Text(
                        l10n.uploadProgress(
                          (_uploadProgress * 100).round(),
                        ),
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              fontWeight: FontWeight.w600,
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
                      backgroundColor:
                          const Color(0xFFF43F5E).withValues(alpha: 0.12),
                      valueColor: const AlwaysStoppedAnimation<Color>(
                        Color(0xFFF43F5E),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          if (_videoUrl != null) ...[
            // Video player preview
            ClipRRect(
              borderRadius: BorderRadius.circular(12),
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
                icon: Icon(Icons.delete_outline, size: 18, color: Theme.of(context).colorScheme.error),
                label: Text(
                  l10n.applyRemoveVideo,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
                style: OutlinedButton.styleFrom(
                  side: BorderSide(color: Theme.of(context).colorScheme.error.withValues(alpha: 0.5)),
                ),
              ),
            ),
          ],
          const SizedBox(height: 16),

          // Submit
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: (_isSubmitting || _isUploading) ? null : _submit,
              style: FilledButton.styleFrom(backgroundColor: const Color(0xFFF43F5E)),
              child: _isSubmitting
                  ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                  : Text(l10n.applySubmit),
            ),
          ),
        ],
      ),
    );
  }
}
