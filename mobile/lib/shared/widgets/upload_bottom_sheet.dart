import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';

/// The type of media being uploaded.
enum UploadMediaType { photo, video }

/// A bottom sheet that lets the user pick a photo (camera or gallery) or a
/// video (gallery via file_picker), preview the selection, and trigger upload.
///
/// All strings are in English.
class UploadBottomSheet extends StatefulWidget {
  const UploadBottomSheet({
    super.key,
    required this.type,
    required this.onUpload,
    this.maxSizeBytes = 10 * 1024 * 1024, // 10 MB default
  });

  final UploadMediaType type;
  final Future<void> Function(File file) onUpload;
  final int maxSizeBytes;

  @override
  State<UploadBottomSheet> createState() => _UploadBottomSheetState();
}

class _UploadBottomSheetState extends State<UploadBottomSheet> {
  File? _selectedFile;
  bool _isUploading = false;
  String? _errorMessage;

  bool get _isPhoto => widget.type == UploadMediaType.photo;

  String get _title =>
      _isPhoto ? 'Add a photo' : 'Add a video';

  // --------------------------------------------------------------------------
  // Pickers
  // --------------------------------------------------------------------------

  Future<void> _pickFromCamera() async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(
      source: ImageSource.camera,
      maxWidth: 1200,
      maxHeight: 1200,
      imageQuality: 85,
    );
    if (picked != null) _setFile(File(picked.path));
  }

  Future<void> _pickFromGallery() async {
    if (_isPhoto) {
      final picker = ImagePicker();
      final picked = await picker.pickImage(
        source: ImageSource.gallery,
        maxWidth: 1200,
        maxHeight: 1200,
        imageQuality: 85,
      );
      if (picked != null) _setFile(File(picked.path));
    } else {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.video,
      );
      if (result != null && result.files.single.path != null) {
        _setFile(File(result.files.single.path!));
      }
    }
  }

  void _setFile(File file) {
    final sizeBytes = file.lengthSync();
    if (sizeBytes > widget.maxSizeBytes) {
      final maxMB = (widget.maxSizeBytes / (1024 * 1024)).toStringAsFixed(0);
      setState(() {
        _errorMessage = 'File too large. Maximum size is $maxMB MB';
        _selectedFile = null;
      });
      return;
    }
    setState(() {
      _selectedFile = file;
      _errorMessage = null;
    });
  }

  // --------------------------------------------------------------------------
  // Upload
  // --------------------------------------------------------------------------

  Future<void> _handleUpload() async {
    if (_selectedFile == null) return;
    setState(() => _isUploading = true);
    try {
      await widget.onUpload(_selectedFile!);
      if (mounted) Navigator.of(context).pop(true);
    } catch (e) {
      if (mounted) {
        setState(() {
          _isUploading = false;
          _errorMessage = 'Upload failed. Please try again.';
        });
      }
    }
  }

  // --------------------------------------------------------------------------
  // UI helpers
  // --------------------------------------------------------------------------

  String _formatFileSize(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }

  // --------------------------------------------------------------------------
  // Build
  // --------------------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;

    return Padding(
      padding: EdgeInsets.only(
        left: 24,
        right: 24,
        top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 24,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Drag handle
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

          // Title
          Text(_title, style: theme.textTheme.titleLarge),
          const SizedBox(height: 16),

          // Option: Camera (photo only)
          if (_isPhoto)
            _OptionTile(
              icon: Icons.camera_alt_outlined,
              title: 'Take a photo',
              subtitle: 'Use the camera',
              onTap: _isUploading ? null : _pickFromCamera,
            ),

          // Option: Gallery
          _OptionTile(
            icon: _isPhoto
                ? Icons.photo_library_outlined
                : Icons.video_library_outlined,
            title: 'Choose from gallery',
            subtitle: _isPhoto
                ? 'Select an image'
                : 'Select a video',
            onTap: _isUploading ? null : _pickFromGallery,
          ),
          const SizedBox(height: 12),

          // Preview
          if (_selectedFile != null) ...[
            if (_isPhoto)
              ClipRRect(
                borderRadius: BorderRadius.circular(12),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxHeight: 200),
                  child: Image.file(
                    _selectedFile!,
                    width: double.infinity,
                    fit: BoxFit.cover,
                  ),
                ),
              )
            else
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: primary.withValues(alpha: 0.05),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: primary.withValues(alpha: 0.2)),
                ),
                child: Row(
                  children: [
                    Icon(Icons.videocam, color: primary, size: 32),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            _selectedFile!.path.split('/').last,
                            style: theme.textTheme.bodyMedium,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                          const SizedBox(height: 2),
                          Text(
                            _formatFileSize(_selectedFile!.lengthSync()),
                            style: theme.textTheme.bodySmall,
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            const SizedBox(height: 16),
          ],

          // Error message
          if (_errorMessage != null) ...[
            Text(
              _errorMessage!,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.error,
              ),
            ),
            const SizedBox(height: 12),
          ],

          // Upload button
          if (_selectedFile != null)
            SizedBox(
              width: double.infinity,
              height: 48,
              child: ElevatedButton(
                onPressed: _isUploading ? null : _handleUpload,
                child: _isUploading
                    ? SizedBox(
                        width: 22,
                        height: 22,
                        child: CircularProgressIndicator(
                          strokeWidth: 2.5,
                          color: theme.colorScheme.onPrimary,
                        ),
                      )
                    : const Text('Upload'),
              ),
            ),

          // Cancel button
          const SizedBox(height: 8),
          SizedBox(
            width: double.infinity,
            height: 48,
            child: TextButton(
              onPressed: _isUploading ? null : () => Navigator.of(context).pop(),
              child: const Text('Cancel'),
            ),
          ),
        ],
      ),
    );
  }
}

// ----------------------------------------------------------------------------
// Option tile (private helper widget)
// ----------------------------------------------------------------------------

class _OptionTile extends StatelessWidget {
  const _OptionTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;

    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Row(
          children: [
            Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(
                color: primary.withValues(alpha: 0.1),
                shape: BoxShape.circle,
              ),
              child: Icon(icon, color: primary, size: 22),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title, style: theme.textTheme.titleMedium),
                  Text(subtitle, style: theme.textTheme.bodySmall),
                ],
              ),
            ),
            Icon(
              Icons.chevron_right,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
            ),
          ],
        ),
      ),
    );
  }
}

/// Shows the [UploadBottomSheet] as a modal bottom sheet.
///
/// Returns `true` if the upload completed successfully, `null` if dismissed.
Future<bool?> showUploadBottomSheet({
  required BuildContext context,
  required UploadMediaType type,
  required Future<void> Function(File file) onUpload,
  int maxSizeBytes = 10 * 1024 * 1024,
}) {
  return showModalBottomSheet<bool>(
    context: context,
    isScrollControlled: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => UploadBottomSheet(
      type: type,
      onUpload: onUpload,
      maxSizeBytes: maxSizeBytes,
    ),
  );
}
