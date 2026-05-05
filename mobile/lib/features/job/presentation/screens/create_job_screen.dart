import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../../../core/network/api_client.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../domain/entities/job_entity.dart';
import '../../domain/repositories/job_repository.dart';
import '../../types/job.dart';
import '../providers/job_provider.dart';
import '../widgets/budget_section.dart';
import '../widgets/job_details_section.dart';

/// M-09 — Soleil v2 full-page form for creating or editing a job posting.
///
/// Composed of two expandable sections (details + budget) plus the
/// description-type / video upload sub-section. When [jobId] is provided,
/// the form loads the existing job and pre-fills all fields (edit mode).
/// All API + form behaviour is unchanged from the previous version — this
/// is a purely visual port to the ivoire/corail palette.
class CreateJobScreen extends ConsumerStatefulWidget {
  const CreateJobScreen({super.key, this.jobId});

  /// When non-null the screen operates in edit mode: fetches the job,
  /// pre-fills all controllers, and calls updateJob on submit.
  final String? jobId;

  bool get isEditMode => jobId != null;

  @override
  ConsumerState<CreateJobScreen> createState() => _CreateJobScreenState();
}

class _CreateJobScreenState extends ConsumerState<CreateJobScreen> {
  final _formKey = GlobalKey<FormState>();

  // Text controllers
  final _titleController = TextEditingController();
  final _descriptionController = TextEditingController();
  final _minBudgetController = TextEditingController();
  final _maxBudgetController = TextEditingController();

  // Form data
  late final JobFormData _formData;

  // Expansion state
  bool _detailsExpanded = true;
  bool _budgetExpanded = false;
  bool _submitting = false;

  // Loading state for edit mode
  bool _loadingJob = false;

  // Video upload state
  bool _isUploadingVideo = false;
  double _uploadProgress = 0;
  String? _videoName;

  bool get _isEditMode => widget.jobId != null;

  bool get _isAgency {
    final authState = ref.read(authProvider);
    return authState.user?['role'] == 'agency';
  }

  @override
  void initState() {
    super.initState();
    _formData = JobFormData();
    if (_isEditMode) {
      _loadExistingJob();
    }
    // Agency can only hire freelancers
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_isAgency) {
        setState(() => _formData.applicantType = ApplicantType.freelancers);
      }
    });
  }

  /// Fetches the existing job and pre-fills all form fields.
  Future<void> _loadExistingJob() async {
    setState(() => _loadingJob = true);
    try {
      final repo = ref.read(jobRepositoryProvider);
      final job = await repo.getJob(widget.jobId!);
      if (!mounted) return;
      _prefillFromJob(job);
    } catch (e) {
      debugPrint('[CreateJobScreen] loadExistingJob error: $e');
    } finally {
      if (mounted) setState(() => _loadingJob = false);
    }
  }

  void _prefillFromJob(JobEntity job) {
    _titleController.text = job.title;
    _descriptionController.text = job.description;
    _minBudgetController.text = job.minBudget.toString();
    _maxBudgetController.text = job.maxBudget.toString();

    _formData.skills
      ..clear()
      ..addAll(job.skills);

    _formData.applicantType = switch (job.applicantType) {
      'freelancers' => ApplicantType.freelancers,
      'agencies' => ApplicantType.agencies,
      _ => ApplicantType.all,
    };

    _formData.budgetType = switch (job.budgetType) {
      'long_term' => BudgetType.longTerm,
      _ => BudgetType.oneShot,
    };

    _formData.descriptionType = switch (job.descriptionType) {
      'video' => DescriptionType.video,
      'both' => DescriptionType.both,
      _ => DescriptionType.text,
    };

    _formData.videoUrl = job.videoUrl ?? '';

    setState(() {});
  }

  @override
  void dispose() {
    _titleController.dispose();
    _descriptionController.dispose();
    _minBudgetController.dispose();
    _maxBudgetController.dispose();
    super.dispose();
  }

  // Section 1 callbacks
  void _onSkillAdded(String skill) {
    setState(() => _formData.skills.add(skill));
  }

  void _onSkillRemoved(int index) {
    setState(() => _formData.skills.removeAt(index));
  }

  void _onApplicantTypeChanged(ApplicantType type) {
    setState(() => _formData.applicantType = type);
  }

  // Description type callback
  void _onDescriptionTypeChanged(DescriptionType type) {
    setState(() => _formData.descriptionType = type);
  }

  // Video picker
  Future<void> _pickVideo() async {
    final picker = ImagePicker();
    final file = await picker.pickVideo(source: ImageSource.gallery);
    if (file == null) return;

    setState(() {
      _isUploadingVideo = true;
      _uploadProgress = 0;
      _videoName = file.name;
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
      if (url != null) {
        setState(() => _formData.videoUrl = url);
      }
    } catch (e) {
      debugPrint('[CreateJobScreen] video upload error: $e');
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
      if (mounted) setState(() => _isUploadingVideo = false);
    }
  }

  void _removeVideo() {
    setState(() {
      _formData.videoUrl = '';
      _videoName = null;
    });
  }

  // Section 2 callbacks
  void _onBudgetTypeChanged(BudgetType type) {
    setState(() => _formData.budgetType = type);
  }

  // Submit
  Future<void> _onSubmit() async {
    _formData.title = _titleController.text.trim();
    _formData.description = _descriptionController.text.trim();
    _formData.minBudget = _minBudgetController.text.trim();
    _formData.maxBudget = _maxBudgetController.text.trim();

    if (!_formKey.currentState!.validate()) return;

    final minBudget = int.tryParse(_formData.minBudget) ?? 0;
    final maxBudget = int.tryParse(_formData.maxBudget) ?? 0;

    if (minBudget <= 0 || maxBudget <= 0) return;

    setState(() => _submitting = true);

    final budgetTypeStr = _formData.budgetType == BudgetType.oneShot
        ? 'one_shot'
        : 'long_term';

    final applicantTypeStr = switch (_formData.applicantType) {
      ApplicantType.all => 'all',
      ApplicantType.freelancers => 'freelancers',
      ApplicantType.agencies => 'agencies',
    };

    final descriptionTypeStr = switch (_formData.descriptionType) {
      DescriptionType.text => 'text',
      DescriptionType.video => 'video',
      DescriptionType.both => 'both',
    };

    final jobData = CreateJobData(
      title: _formData.title,
      description: _formData.description,
      skills: _formData.skills,
      applicantType: applicantTypeStr,
      budgetType: budgetTypeStr,
      minBudget: minBudget,
      maxBudget: maxBudget,
      descriptionType: descriptionTypeStr,
      videoUrl: _formData.videoUrl.isNotEmpty ? _formData.videoUrl : null,
    );

    final JobEntity? result;
    if (_isEditMode) {
      result = await updateJobAction(ref, widget.jobId!, jobData);
    } else {
      result = await createJobAction(ref, jobData);
    }

    if (!mounted) return;
    setState(() => _submitting = false);

    if (result != null) {
      if (_isEditMode) {
        final l10n = AppLocalizations.of(context)!;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.jobUpdateSuccess)),
        );
      }
      Navigator.of(context).pop();
    } else {
      final l10n = AppLocalizations.of(context)!;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.unexpectedError)),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;
    final border = appColors?.border ?? theme.colorScheme.outline;

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: () => Navigator.of(context).pop(),
        ),
        title: Text(
          _isEditMode ? l10n.createJob_m09_titleEdit : l10n.createJob_m09_title,
          style: SoleilTextStyles.titleLarge,
        ),
      ),
      body: _loadingJob
          ? const Center(child: CircularProgressIndicator())
          : SafeArea(
              child: Column(
                children: [
                  Expanded(
                    child: Form(
                      key: _formKey,
                      child: ListView(
                        padding: const EdgeInsets.fromLTRB(16, 4, 16, 16),
                        children: [
                          _SoleilHero(
                            eyebrow: l10n.createJob_m09_eyebrow,
                            titlePrefix: l10n.createJob_m09_heroPrefix,
                            titleAccent: l10n.createJob_m09_heroAccent,
                            subtitle: l10n.createJob_m09_subtitle,
                          ),
                          const SizedBox(height: 22),
                          JobDetailsSection(
                            titleController: _titleController,
                            descriptionController: _descriptionController,
                            skills: _formData.skills,
                            onSkillAdded: _onSkillAdded,
                            onSkillRemoved: _onSkillRemoved,
                            applicantType: _formData.applicantType,
                            onApplicantTypeChanged: _onApplicantTypeChanged,
                            isExpanded: _detailsExpanded,
                            onExpansionChanged: (expanded) {
                              setState(() => _detailsExpanded = expanded);
                            },
                            showDescription: false,
                            hideApplicantType: _isAgency,
                          ),
                          const SizedBox(height: 14),
                          _DescriptionTypeSection(
                            descriptionType: _formData.descriptionType,
                            onDescriptionTypeChanged: _onDescriptionTypeChanged,
                            descriptionController: _descriptionController,
                            videoUrl: _formData.videoUrl,
                            videoName: _videoName,
                            isUploading: _isUploadingVideo,
                            uploadProgress: _uploadProgress,
                            onPickVideo: _pickVideo,
                            onRemoveVideo: _removeVideo,
                          ),
                          const SizedBox(height: 14),
                          BudgetSection(
                            budgetType: _formData.budgetType,
                            onBudgetTypeChanged: _onBudgetTypeChanged,
                            minBudgetController: _minBudgetController,
                            maxBudgetController: _maxBudgetController,
                            isExpanded: _budgetExpanded,
                            onExpansionChanged: (expanded) {
                              setState(() => _budgetExpanded = expanded);
                            },
                          ),
                          const SizedBox(height: 12),
                        ],
                      ),
                    ),
                  ),
                  // Sticky bottom CTA bar
                  Container(
                    padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.surfaceContainerLowest,
                      border: Border(top: BorderSide(color: border)),
                    ),
                    child: Row(
                      children: [
                        OutlinedButton(
                          onPressed: () => Navigator.of(context).pop(),
                          style: OutlinedButton.styleFrom(
                            minimumSize: const Size(0, 48),
                            padding: const EdgeInsets.symmetric(horizontal: 20),
                            foregroundColor: mute,
                            side: BorderSide(color: border),
                            shape: const StadiumBorder(),
                          ),
                          child: Text(
                            l10n.jobCancel,
                            style: SoleilTextStyles.button.copyWith(color: mute),
                          ),
                        ),
                        const SizedBox(width: 10),
                        Expanded(
                          child: FilledButton(
                            onPressed: _submitting ? null : _onSubmit,
                            style: FilledButton.styleFrom(
                              backgroundColor: primary,
                              foregroundColor: theme.colorScheme.onPrimary,
                              minimumSize: const Size.fromHeight(48),
                              shape: const StadiumBorder(),
                              textStyle: SoleilTextStyles.button,
                            ),
                            child: _submitting
                                ? SizedBox(
                                    width: 18,
                                    height: 18,
                                    child: CircularProgressIndicator(
                                      strokeWidth: 2,
                                      color: theme.colorScheme.onPrimary,
                                    ),
                                  )
                                : Row(
                                    mainAxisAlignment: MainAxisAlignment.center,
                                    children: [
                                      Text(
                                        _isEditMode
                                            ? l10n.jobSave
                                            : l10n.createJob_m09_publishCta,
                                      ),
                                      const SizedBox(width: 6),
                                      const Icon(Icons.arrow_forward, size: 16),
                                    ],
                                  ),
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
}

// ---------------------------------------------------------------------------
// Soleil editorial hero (corail eyebrow + Fraunces display title with italic
// corail accent + tabac subtitle)
// ---------------------------------------------------------------------------

class _SoleilHero extends StatelessWidget {
  const _SoleilHero({
    required this.eyebrow,
    required this.titlePrefix,
    required this.titleAccent,
    required this.subtitle,
  });

  final String eyebrow;
  final String titlePrefix;
  final String titleAccent;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return Padding(
      padding: const EdgeInsets.only(top: 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            eyebrow,
            style: SoleilTextStyles.mono.copyWith(
              color: primary,
              fontSize: 11,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.4,
            ),
          ),
          const SizedBox(height: 8),
          RichText(
            text: TextSpan(
              style: SoleilTextStyles.displayM.copyWith(
                color: theme.colorScheme.onSurface,
              ),
              children: [
                TextSpan(text: '$titlePrefix '),
                TextSpan(
                  text: titleAccent,
                  style: SoleilTextStyles.displayM.copyWith(
                    fontStyle: FontStyle.italic,
                    color: primary,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
          Text(
            subtitle,
            style: SoleilTextStyles.body.copyWith(color: mute, fontSize: 13.5),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Description type selector + video upload — Soleil-styled inside a card
// matching the same ivoire surface as the other sections.
// ---------------------------------------------------------------------------

class _DescriptionTypeSection extends StatelessWidget {
  const _DescriptionTypeSection({
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
