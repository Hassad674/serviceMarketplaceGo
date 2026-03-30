import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/identity_document_entity.dart';
import '../providers/identity_document_provider.dart';
import 'payment_info_widgets.dart';

// ---------------------------------------------------------------------------
// Document type options
// ---------------------------------------------------------------------------

const _documentTypes = [
  ('passport', 'Passport', 'Passeport'),
  ('id_card', 'ID Card', "Carte d'identit\u00e9"),
  ('driving_license', 'Driving License', 'Permis de conduire'),
];

// ---------------------------------------------------------------------------
// Main section widget
// ---------------------------------------------------------------------------

/// Identity verification section integrated into the payment info screen.
///
/// Shows existing documents with status badges, and allows uploading
/// new documents via camera or gallery.
class IdentityVerificationSection extends ConsumerStatefulWidget {
  const IdentityVerificationSection({super.key});

  @override
  ConsumerState<IdentityVerificationSection> createState() =>
      _IdentityVerificationSectionState();
}

class _IdentityVerificationSectionState
    extends ConsumerState<IdentityVerificationSection> {
  String _selectedType = 'passport';
  bool _uploading = false;

  Future<void> _pickAndUpload(ImageSource source) async {
    final picker = ImagePicker();
    final image = await picker.pickImage(
      source: source,
      maxWidth: 2000,
      maxHeight: 2000,
      imageQuality: 85,
    );
    if (image == null) return;

    setState(() => _uploading = true);
    try {
      final repo = ref.read(identityDocumentRepositoryProvider);
      await repo.uploadDocument(
        documentType: _selectedType,
        side: 'front',
        filePath: image.path,
        fileName: image.name,
      );
      ref.invalidate(identityDocumentsProvider);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              AppLocalizations.of(context)!.identityDocUploaded,
            ),
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Upload failed: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final asyncDocs = ref.watch(identityDocumentsProvider);

    return PaymentSectionCard(
      title: l10n.identityDocTitle,
      children: [
        Text(
          l10n.identityDocSubtitle,
          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color: Theme.of(context)
                    .colorScheme
                    .onSurface
                    .withValues(alpha: 0.6),
              ),
        ),
        const SizedBox(height: 12),

        // Existing documents
        asyncDocs.when(
          loading: () =>
              const Center(child: CircularProgressIndicator()),
          error: (_, __) => Text(l10n.somethingWentWrong),
          data: (docs) => _DocumentsList(
            documents: docs,
            onDelete: _handleDelete,
          ),
        ),

        const SizedBox(height: 12),

        // Document type selector
        _DocumentTypeSelector(
          value: _selectedType,
          onChanged: (v) => setState(() => _selectedType = v),
        ),
        const SizedBox(height: 12),

        // Upload buttons
        _UploadButtons(
          uploading: _uploading,
          onCamera: () => _pickAndUpload(ImageSource.camera),
          onGallery: () => _pickAndUpload(ImageSource.gallery),
        ),
      ],
    );
  }

  Future<void> _handleDelete(String id) async {
    try {
      final repo = ref.read(identityDocumentRepositoryProvider);
      await repo.deleteDocument(id);
      ref.invalidate(identityDocumentsProvider);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Delete failed: $e')),
        );
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Documents list with status badges
// ---------------------------------------------------------------------------

class _DocumentsList extends StatelessWidget {
  const _DocumentsList({
    required this.documents,
    required this.onDelete,
  });

  final List<IdentityDocument> documents;
  final ValueChanged<String> onDelete;

  @override
  Widget build(BuildContext context) {
    if (documents.isEmpty) return const SizedBox.shrink();

    return Column(
      children: documents
          .map((doc) => _DocumentTile(doc: doc, onDelete: onDelete))
          .toList(),
    );
  }
}

class _DocumentTile extends StatelessWidget {
  const _DocumentTile({
    required this.doc,
    required this.onDelete,
  });

  final IdentityDocument doc;
  final ValueChanged<String> onDelete;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final locale = Localizations.localeOf(context).languageCode;

    final typeLabel = _documentTypes
        .where((t) => t.$1 == doc.documentType)
        .map((t) => locale == 'fr' ? t.$3 : t.$2)
        .firstOrNull ?? doc.documentType;

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
      ),
      child: Row(
        children: [
          Icon(Icons.description_outlined,
              size: 20,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.6)),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  typeLabel,
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(fontWeight: FontWeight.w500),
                ),
                if (doc.isRejected && doc.rejectionReason.isNotEmpty)
                  Text(
                    doc.rejectionReason,
                    style: TextStyle(
                      fontSize: 12,
                      color: theme.colorScheme.error,
                    ),
                  ),
              ],
            ),
          ),
          _StatusBadge(status: doc.status, l10n: l10n),
          const SizedBox(width: 4),
          IconButton(
            icon: const Icon(Icons.close, size: 18),
            onPressed: () => onDelete(doc.id),
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Status badge
// ---------------------------------------------------------------------------

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status, required this.l10n});

  final String status;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final (Color bg, Color fg, String label) = switch (status) {
      'verified' => (
        const Color(0xFFECFDF5),
        const Color(0xFF15803D),
        l10n.identityDocVerified,
      ),
      'rejected' => (
        const Color(0xFFFEF2F2),
        const Color(0xFFDC2626),
        l10n.identityDocRejected,
      ),
      _ => (
        const Color(0xFFFFFBEB),
        const Color(0xFFD97706),
        l10n.identityDocPending,
      ),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: fg,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Document type selector
// ---------------------------------------------------------------------------

class _DocumentTypeSelector extends StatelessWidget {
  const _DocumentTypeSelector({
    required this.value,
    required this.onChanged,
  });

  final String value;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;

    return DropdownButtonFormField<String>(
      value: value,
      decoration: InputDecoration(
        labelText: l10n.identityDocType,
      ),
      items: _documentTypes.map((t) {
        final label = locale == 'fr' ? t.$3 : t.$2;
        return DropdownMenuItem(value: t.$1, child: Text(label));
      }).toList(),
      onChanged: (v) {
        if (v != null) onChanged(v);
      },
    );
  }
}

// ---------------------------------------------------------------------------
// Upload buttons
// ---------------------------------------------------------------------------

class _UploadButtons extends StatelessWidget {
  const _UploadButtons({
    required this.uploading,
    required this.onCamera,
    required this.onGallery,
  });

  final bool uploading;
  final VoidCallback onCamera;
  final VoidCallback onGallery;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    if (uploading) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.all(16),
          child: CircularProgressIndicator(),
        ),
      );
    }

    return Row(
      children: [
        Expanded(
          child: OutlinedButton.icon(
            onPressed: onCamera,
            icon: const Icon(Icons.camera_alt_outlined, size: 18),
            label: Text(l10n.takePhoto),
          ),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: OutlinedButton.icon(
            onPressed: onGallery,
            icon: const Icon(Icons.photo_library_outlined, size: 18),
            label: Text(l10n.chooseFromGallery),
          ),
        ),
      ],
    );
  }
}
