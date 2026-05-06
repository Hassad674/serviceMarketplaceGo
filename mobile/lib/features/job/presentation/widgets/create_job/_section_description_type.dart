import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../../../shared/widgets/video_player_widget.dart';
import '../../../types/job.dart';

/// Description-type segmented selector + optional text description and
/// video upload preview, all wrapped in a Soleil ivoire surface card.
///
/// Pure presentation widget — receives every piece of state from the
/// parent screen (controllers, callbacks, upload progress). Extracted
/// from `create_job_screen.dart` as part of the NF-9 file split
/// (V7 audit). Behaviour is unchanged.
class CreateJobDescriptionTypeSection extends StatelessWidget {
  const CreateJobDescriptionTypeSection({
    super.key,
    required this.descriptionType,
    required this.onDescriptionTypeChanged,
    required this.descriptionController,
    required this.videoUrl,
    required this.videoName,
    required this.isUploading,
    required this.uploadProgress,
    required this.onPickVideo,
    required this.onRemoveVideo,
  });

  final DescriptionType descriptionType;
  final ValueChanged<DescriptionType> onDescriptionTypeChanged;
  final TextEditingController descriptionController;
  final String videoUrl;
  final String? videoName;
  final bool isUploading;
  final double uploadProgress;
  final VoidCallback onPickVideo;
  final VoidCallback onRemoveVideo;

  bool get _showVideoUpload =>
      descriptionType == DescriptionType.video ||
      descriptionType == DescriptionType.both;

  bool get _showTextDescription =>
      descriptionType == DescriptionType.text ||
      descriptionType == DescriptionType.both;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final accentSoft = appColors?.accentSoft ?? theme.colorScheme.primaryContainer;
    final border = appColors?.border ?? theme.colorScheme.outline;
    final borderStrong = appColors?.borderStrong ?? theme.colorScheme.outline;
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(color: border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.jobDescriptionType.toUpperCase(),
            style: SoleilTextStyles.mono.copyWith(
              color: mute,
              fontSize: 11,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.8,
            ),
          ),
          const SizedBox(height: 12),
          // Soleil pill segmented selector
          Container(
            padding: const EdgeInsets.all(4),
            decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
              border: Border.all(color: border),
            ),
            child: Row(
              children: [
                Expanded(
                  child: _DescTypePill(
                    label: l10n.jobDescriptionTypeText,
                    icon: Icons.text_fields,
                    selected: descriptionType == DescriptionType.text,
                    onTap: () => onDescriptionTypeChanged(DescriptionType.text),
                  ),
                ),
                Expanded(
                  child: _DescTypePill(
                    label: l10n.jobDescriptionTypeVideo,
                    icon: Icons.videocam_outlined,
                    selected: descriptionType == DescriptionType.video,
                    onTap: () => onDescriptionTypeChanged(DescriptionType.video),
                  ),
                ),
                Expanded(
                  child: _DescTypePill(
                    label: l10n.jobDescriptionTypeBoth,
                    icon: Icons.dashboard_outlined,
                    selected: descriptionType == DescriptionType.both,
                    onTap: () => onDescriptionTypeChanged(DescriptionType.both),
                  ),
                ),
              ],
            ),
          ),
          if (_showTextDescription) ...[
            const SizedBox(height: 18),
            Text(
              l10n.jobDescription.toUpperCase(),
              style: SoleilTextStyles.mono.copyWith(
                color: mute,
                fontSize: 11,
                fontWeight: FontWeight.w700,
                letterSpacing: 0.8,
              ),
            ),
            const SizedBox(height: 8),
            TextFormField(
              controller: descriptionController,
              decoration: const InputDecoration(
                alignLabelWithHint: true,
              ),
              maxLines: 5,
              textInputAction: TextInputAction.newline,
            ),
          ],
          if (_showVideoUpload) ...[
            const SizedBox(height: 18),
            if (videoUrl.isEmpty && !isUploading)
              InkWell(
                onTap: onPickVideo,
                borderRadius: BorderRadius.circular(AppTheme.radiusLg),
                child: Container(
                  padding: const EdgeInsets.all(18),
                  decoration: BoxDecoration(
                    border: Border.all(
                      color: borderStrong,
                      width: 1.5,
                      style: BorderStyle.solid,
                    ),
                    borderRadius: BorderRadius.circular(AppTheme.radiusLg),
                    color: theme.colorScheme.surface,
                  ),
                  child: Row(
                    children: [
                      Container(
                        width: 56,
                        height: 44,
                        decoration: BoxDecoration(
                          gradient: LinearGradient(
                            colors: [
                              appColors?.amberSoft ?? accentSoft,
                              appColors?.pinkSoft ?? accentSoft,
                            ],
                            begin: Alignment.topLeft,
                            end: Alignment.bottomRight,
                          ),
                          borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                        ),
                        alignment: Alignment.center,
                        child: Container(
                          width: 26,
                          height: 26,
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            color: theme.colorScheme.surfaceContainerLowest,
                          ),
                          alignment: Alignment.center,
                          child: Icon(
                            Icons.play_arrow,
                            size: 14,
                            color: primary,
                          ),
                        ),
                      ),
                      const SizedBox(width: 14),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              l10n.jobAddVideo,
                              style: SoleilTextStyles.bodyEmphasis,
                            ),
                            const SizedBox(height: 2),
                            Text(
                              l10n.createJob_m09_subtitle,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: SoleilTextStyles.caption.copyWith(
                                color: mute,
                                fontStyle: FontStyle.italic,
                              ),
                            ),
                          ],
                        ),
                      ),
                      Icon(Icons.add, size: 18, color: mute),
                    ],
                  ),
                ),
              ),
            if (isUploading)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Column(
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(
                          l10n.jobVideoUploading,
                          style: SoleilTextStyles.body,
                        ),
                        Text(
                          l10n.uploadProgress((uploadProgress * 100).round()),
                          style: SoleilTextStyles.mono.copyWith(
                            fontSize: 12,
                            color: primary,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    ClipRRect(
                      borderRadius: BorderRadius.circular(AppTheme.radiusFull),
                      child: LinearProgressIndicator(
                        value: uploadProgress,
                        minHeight: 6,
                        backgroundColor: accentSoft,
                        valueColor: AlwaysStoppedAnimation<Color>(primary),
                      ),
                    ),
                  ],
                ),
              ),
            if (videoUrl.isNotEmpty && !isUploading) ...[
              VideoPlayerWidget(videoUrl: videoUrl),
              const SizedBox(height: 8),
              TextButton.icon(
                onPressed: onRemoveVideo,
                icon: const Icon(Icons.delete_outline, size: 18),
                label: Text(videoName ?? l10n.jobVideoUploaded),
                style: TextButton.styleFrom(
                  foregroundColor: theme.colorScheme.error,
                ),
              ),
            ],
          ],
        ],
      ),
    );
  }
}

class _DescTypePill extends StatelessWidget {
  const _DescTypePill({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          curve: Curves.easeOut,
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
          decoration: BoxDecoration(
            color: selected ? primary : Colors.transparent,
            borderRadius: BorderRadius.circular(AppTheme.radiusFull),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                icon,
                size: 14,
                color: selected ? theme.colorScheme.onPrimary : mute,
              ),
              const SizedBox(width: 4),
              Flexible(
                child: Text(
                  label,
                  textAlign: TextAlign.center,
                  overflow: TextOverflow.ellipsis,
                  style: SoleilTextStyles.button.copyWith(
                    color: selected ? theme.colorScheme.onPrimary : mute,
                    fontSize: 12,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
