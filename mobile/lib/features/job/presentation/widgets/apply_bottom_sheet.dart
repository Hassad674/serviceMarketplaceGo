import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'dart:io';
import 'package:dio/dio.dart';

import '../../../../core/network/api_client.dart';
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
  String? _videoName;
  bool _isUploading = false;

  @override
  void dispose() {
    _messageController.dispose();
    super.dispose();
  }

  Future<void> _pickVideo() async {
    final picker = ImagePicker();
    final file = await picker.pickVideo(source: ImageSource.gallery);
    if (file == null) return;

    setState(() { _isUploading = true; _videoName = file.name; });
    try {
      final apiClient = ref.read(apiClientProvider);
      final formData = FormData.fromMap({
        'file': await MultipartFile.fromFile(file.path, filename: file.name),
      });
      final response = await apiClient.upload('/api/v1/upload/video', data: formData);
      final url = response.data?['url'] as String?;
      if (url != null) setState(() => _videoUrl = url);
    } catch (e) {
      debugPrint('[ApplyBottomSheet] video upload error: $e');
    } finally {
      setState(() => _isUploading = false);
    }
  }

  void _removeVideo() {
    setState(() { _videoUrl = null; _videoName = null; });
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

    if (result != null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Candidature envoy\u00e9e !'), backgroundColor: Color(0xFFF43F5E)),
      );
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Erreur lors de l\u2019envoi'), backgroundColor: Colors.red),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
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
          Text('Postuler', style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          const SizedBox(height: 16),

          // Message (optional)
          TextField(
            controller: _messageController,
            maxLines: 5,
            maxLength: 5000,
            decoration: const InputDecoration(
              labelText: 'Votre message (optionnel)',
              hintText: 'Pourquoi \u00eates-vous le bon candidat ?',
              border: OutlineInputBorder(),
              alignLabelWithHint: true,
            ),
          ),
          const SizedBox(height: 16),

          // Video upload (optional)
          if (_videoUrl == null && !_isUploading)
            OutlinedButton.icon(
              onPressed: _pickVideo,
              icon: const Icon(Icons.videocam_outlined),
              label: const Text('Ajouter une vid\u00e9o'),
              style: OutlinedButton.styleFrom(minimumSize: const Size.fromHeight(44)),
            ),
          if (_isUploading)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 8),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2)), SizedBox(width: 8), Text('Envoi en cours...')],
              ),
            ),
          if (_videoUrl != null)
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(color: Colors.green.shade50, borderRadius: BorderRadius.circular(8)),
              child: Row(
                children: [
                  const Icon(Icons.check_circle, color: Colors.green, size: 20),
                  const SizedBox(width: 8),
                  Expanded(child: Text(_videoName ?? 'Vid\u00e9o', overflow: TextOverflow.ellipsis, style: const TextStyle(fontSize: 13))),
                  IconButton(icon: const Icon(Icons.close, size: 18), onPressed: _removeVideo, padding: EdgeInsets.zero, constraints: const BoxConstraints()),
                ],
              ),
            ),
          const SizedBox(height: 16),

          // Submit
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: (_isSubmitting || _isUploading) ? null : _submit,
              style: FilledButton.styleFrom(backgroundColor: const Color(0xFFF43F5E)),
              child: _isSubmitting
                  ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                  : const Text('Envoyer ma candidature'),
            ),
          ),
        ],
      ),
    );
  }
}
