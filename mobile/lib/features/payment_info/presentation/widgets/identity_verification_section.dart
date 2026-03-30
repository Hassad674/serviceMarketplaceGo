import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/identity_document_entity.dart';
import '../providers/identity_document_provider.dart';
import 'payment_info_widgets.dart';

class _DocType {
  const _DocType(this.value, this.sides);
  final String value;
  final List<String> sides;
}

const _docTypes = [
  _DocType('passport', ['single']),
  _DocType('id_card', ['front', 'back']),
  _DocType('driving_license', ['front', 'back']),
];

/// Identity verification section matching the web UX flow.
class IdentityVerificationSection extends ConsumerStatefulWidget {
  const IdentityVerificationSection({super.key});

  @override
  ConsumerState<IdentityVerificationSection> createState() =>
      _IdentityVerificationSectionState();
}

class _IdentityVerificationSectionState
    extends ConsumerState<IdentityVerificationSection> {
  bool _uploading = false;

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
        asyncDocs.when(
          loading: () =>
              const Center(child: CircularProgressIndicator()),
          error: (_, __) => Text(l10n.somethingWentWrong),
          data: (docs) => _buildContent(docs, l10n),
        ),
      ],
    );
  }

  Widget _buildContent(List<IdentityDocument> docs, AppLocalizations l10n) {
    final identityDocs =
        docs.where((d) => d.category == 'identity').toList();
    final status = _overallStatus(identityDocs);

    if (_uploading) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.all(16),
          child: CircularProgressIndicator(),
        ),
      );
    }

    return switch (status) {
      'none' => _buildUploadPrompt(l10n),
      'pending' => _buildStatusBanner(
          identityDocs, l10n, _pendingColors,
          l10n.identityDocPendingBanner, Icons.schedule,),
      'verified' => _buildStatusBanner(
          identityDocs, l10n, _verifiedColors,
          l10n.identityDocVerifiedBanner, Icons.check_circle_outline,),
      'rejected' => _buildRejectedState(identityDocs, l10n),
      _ => _buildUploadPrompt(l10n),
    };
  }

  Widget _buildUploadPrompt(AppLocalizations l10n) {
    return InkWell(
      onTap: () => _showTypeSelector(),
      borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(vertical: 32),
        decoration: BoxDecoration(
          border: Border.all(
            color: Theme.of(context).dividerColor,
            width: 2,
            strokeAlign: BorderSide.strokeAlignInside,
          ),
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        ),
        child: Column(
          children: [
            Icon(
              Icons.upload_file,
              size: 40,
              color: Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.4),
            ),
            const SizedBox(height: 8),
            Text(
              l10n.identityDocUpload,
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w500,
                color: Theme.of(context)
                    .colorScheme
                    .onSurface
                    .withValues(alpha: 0.6),
              ),
            ),
            const SizedBox(height: 4),
            Text(
              l10n.identityDocUploadDesc,
              style: TextStyle(
                fontSize: 12,
                color: Theme.of(context)
                    .colorScheme
                    .onSurface
                    .withValues(alpha: 0.4),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStatusBanner(
    List<IdentityDocument> docs,
    AppLocalizations l10n,
    _StatusColors colors,
    String message,
    IconData icon,
  ) {
    final doc = docs.first;
    final typeLabel = _docTypeLabel(doc.documentType, l10n);

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: colors.background,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Row(
        children: [
          Icon(icon, size: 20, color: colors.foreground),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  message,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                    color: colors.foreground,
                  ),
                ),
                Text(
                  typeLabel,
                  style: TextStyle(
                    fontSize: 11,
                    color: colors.foreground.withValues(alpha: 0.7),
                  ),
                ),
              ],
            ),
          ),
          TextButton.icon(
            onPressed: () => _showTypeSelector(),
            icon: const Icon(Icons.edit_outlined, size: 16),
            label: Text(l10n.identityDocReplace),
            style: TextButton.styleFrom(
              foregroundColor: colors.foreground,
              textStyle: const TextStyle(fontSize: 12),
              padding: const EdgeInsets.symmetric(horizontal: 8),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildRejectedState(
    List<IdentityDocument> docs,
    AppLocalizations l10n,
  ) {
    final doc = docs.first;

    return Column(
      children: [
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: const Color(0xFFFEF2F2),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
          child: Row(
            children: [
              const Icon(
                  Icons.warning_amber,
                  size: 20, color: Color(0xFFDC2626),),
              const SizedBox(width: 8),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.identityDocRejectedBanner,
                      style: const TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w500,
                        color: Color(0xFFDC2626),
                      ),
                    ),
                    if (doc.rejectionReason.isNotEmpty)
                      Text(
                        doc.rejectionReason,
                        style: const TextStyle(
                          fontSize: 11,
                          color: Color(0xFFDC2626),
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        SizedBox(
          width: double.infinity,
          child: ElevatedButton.icon(
            onPressed: () => _showTypeSelector(),
            icon: const Icon(Icons.upload_file, size: 18),
            label: Text(l10n.identityDocUpload),
            style: ElevatedButton.styleFrom(
              backgroundColor: const Color(0xFFF43F5E),
              foregroundColor: Colors.white,
              shape: RoundedRectangleBorder(
                borderRadius:
                    BorderRadius.circular(AppTheme.radiusLg),
              ),
            ),
          ),
        ),
      ],
    );
  }

  void _showTypeSelector() {
    final l10n = AppLocalizations.of(context)!;

    showModalBottomSheet<void>(
      context: context,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(
          top: Radius.circular(AppTheme.radiusXl),
        ),
      ),
      builder: (ctx) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Center(
                child: Container(
                  width: 32,
                  height: 4,
                  decoration: BoxDecoration(
                    color: Theme.of(ctx).dividerColor,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Text(
                l10n.identityDocSelectType,
                style: Theme.of(ctx)
                    .textTheme
                    .titleMedium
                    ?.copyWith(fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 4),
              Text(
                l10n.identityDocUploadDesc,
                style: Theme.of(ctx).textTheme.bodySmall?.copyWith(
                      color: Theme.of(ctx)
                          .colorScheme
                          .onSurface
                          .withValues(alpha: 0.6),
                    ),
              ),
              const SizedBox(height: 16),
              _TypeOption(
                icon: Icons.menu_book_outlined,
                label: l10n.identityDocPassport,
                subtitle: l10n.identityDocSinglePage,
                onTap: () {
                  Navigator.pop(ctx);
                  _startUploadFlow('passport');
                },
              ),
              _TypeOption(
                icon: Icons.badge_outlined,
                label: l10n.identityDocIdCard,
                subtitle: l10n.identityDocFrontAndBack,
                onTap: () {
                  Navigator.pop(ctx);
                  _startUploadFlow('id_card');
                },
              ),
              _TypeOption(
                icon: Icons.directions_car_outlined,
                label: l10n.identityDocDrivingLicense,
                subtitle: l10n.identityDocFrontAndBack,
                onTap: () {
                  Navigator.pop(ctx);
                  _startUploadFlow('driving_license');
                },
              ),
              const SizedBox(height: 8),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _startUploadFlow(String docType) async {
    final dt = _docTypes.firstWhere((d) => d.value == docType);

    for (int i = 0; i < dt.sides.length; i++) {
      final side = dt.sides[i];
      final image = await _pickImage(side, docType, i, dt.sides.length);
      if (image == null) return; // User cancelled
      await _uploadFile(image, docType, side);
    }
  }

  Future<XFile?> _pickImage(
    String side,
    String docType,
    int index,
    int total,
  ) async {
    final l10n = AppLocalizations.of(context)!;
    final sideLabel = side == 'front'
        ? l10n.identityDocFrontSide
        : side == 'back'
            ? l10n.identityDocBackSide
            : _docTypeLabel(docType, l10n);

    final title = total > 1
        ? '$sideLabel (${index + 1}/$total)'
        : sideLabel;

    final source = await showModalBottomSheet<ImageSource>(
      context: context,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(
          top: Radius.circular(AppTheme.radiusXl),
        ),
      ),
      builder: (ctx) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Center(
                child: Container(
                  width: 32,
                  height: 4,
                  decoration: BoxDecoration(
                    color: Theme.of(ctx).dividerColor,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Text(
                title,
                style: Theme.of(ctx)
                    .textTheme
                    .titleMedium
                    ?.copyWith(fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: () =>
                          Navigator.pop(ctx, ImageSource.camera),
                      icon: const Icon(
                          Icons.camera_alt_outlined, size: 18,),
                      label: Text(l10n.takePhoto),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: () =>
                          Navigator.pop(ctx, ImageSource.gallery),
                      icon: const Icon(
                          Icons.photo_library_outlined, size: 18,),
                      label: Text(l10n.chooseFromGallery),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
            ],
          ),
        ),
      ),
    );

    if (source == null) return null;

    final picker = ImagePicker();
    return picker.pickImage(
      source: source,
      maxWidth: 2000,
      maxHeight: 2000,
      imageQuality: 85,
    );
  }

  Future<void> _uploadFile(XFile image, String docType, String side) async {
    setState(() => _uploading = true);
    try {
      final repo = ref.read(identityDocumentRepositoryProvider);
      await repo.uploadDocument(
        documentType: docType,
        side: side,
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

  String _overallStatus(List<IdentityDocument> docs) {
    if (docs.isEmpty) return 'none';
    if (docs.any((d) => d.isRejected)) return 'rejected';
    if (docs.any((d) => d.isVerified)) return 'verified';
    return 'pending';
  }

  String _docTypeLabel(String type, AppLocalizations l10n) {
    return switch (type) {
      'passport' => l10n.identityDocPassport,
      'id_card' => l10n.identityDocIdCard,
      'driving_license' => l10n.identityDocDrivingLicense,
      _ => type,
    };
  }
}

class _TypeOption extends StatelessWidget {
  const _TypeOption({
    required this.icon,
    required this.label,
    required this.subtitle,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        child: Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            border: Border.all(
              color: theme.dividerColor.withValues(alpha: 0.5),
            ),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
          child: Row(
            children: [
              Icon(
                  icon, size: 24,
                  color: theme.colorScheme.onSurface
                      .withValues(alpha: 0.5),),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      label,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    Text(
                      subtitle,
                      style: TextStyle(
                        fontSize: 12,
                        color: theme.colorScheme.onSurface
                            .withValues(alpha: 0.5),
                      ),
                    ),
                  ],
                ),
              ),
              Icon(
                  Icons.chevron_right,
                  size: 20,
                  color: theme.colorScheme.onSurface
                      .withValues(alpha: 0.3),),
            ],
          ),
        ),
      ),
    );
  }
}

class _StatusColors {
  const _StatusColors(this.background, this.foreground);
  final Color background;
  final Color foreground;
}

const _pendingColors = _StatusColors(Color(0xFFFFFBEB), Color(0xFFD97706));
const _verifiedColors = _StatusColors(Color(0xFFECFDF5), Color(0xFF15803D));
