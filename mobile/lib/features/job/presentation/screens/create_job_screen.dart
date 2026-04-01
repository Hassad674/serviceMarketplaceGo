import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/repositories/job_repository.dart';
import '../../types/job.dart';
import '../providers/job_provider.dart';
import '../widgets/budget_section.dart';
import '../widgets/job_details_section.dart';

/// Full-page scrollable form for creating a new job posting.
///
/// Composed of two expandable sections. Tapping "Publish" calls the
/// backend API to persist the job, then pops back to the jobs list.
class CreateJobScreen extends ConsumerStatefulWidget {
  const CreateJobScreen({super.key});

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

  // Video upload state
  bool _isUploadingVideo = false;
  String? _videoName;

  @override
  void initState() {
    super.initState();
    _formData = JobFormData();
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
      );
      final url = response.data?['url'] as String?;
      if (url != null) {
        setState(() => _formData.videoUrl = url);
      }
    } catch (e) {
      debugPrint('[CreateJobScreen] video upload error: $e');
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

    final result = await createJobAction(
      ref,
      CreateJobData(
        title: _formData.title,
        description: _formData.description,
        skills: _formData.skills,
        applicantType: applicantTypeStr,
        budgetType: budgetTypeStr,
        minBudget: minBudget,
        maxBudget: maxBudget,
        descriptionType: descriptionTypeStr,
        videoUrl: _formData.videoUrl.isNotEmpty ? _formData.videoUrl : null,
      ),
    );

    if (!mounted) return;
    setState(() => _submitting = false);

    if (result != null) {
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

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: () => Navigator.of(context).pop(),
        ),
        title: Text(l10n.jobCreateJob),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 8),
            child: FilledButton(
              onPressed: _submitting ? null : _onSubmit,
              style: FilledButton.styleFrom(
                backgroundColor: theme.colorScheme.primary,
                foregroundColor: Colors.white,
                shape: RoundedRectangleBorder(
                  borderRadius:
                      BorderRadius.circular(AppTheme.radiusSm),
                ),
              ),
              child: _submitting
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: Colors.white,
                      ),
                    )
                  : Text(l10n.jobPublish),
            ),
          ),
        ],
      ),
      body: SafeArea(
        child: Form(
          key: _formKey,
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              // Section 1: Job details
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
              ),
              const SizedBox(height: 16),

              // Section: Description type + video upload
              _DescriptionTypeSection(
                descriptionType: _formData.descriptionType,
                onDescriptionTypeChanged: _onDescriptionTypeChanged,
                videoUrl: _formData.videoUrl,
                videoName: _videoName,
                isUploading: _isUploadingVideo,
                onPickVideo: _pickVideo,
                onRemoveVideo: _removeVideo,
              ),
              const SizedBox(height: 16),

              // Section 2: Budget
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
              const SizedBox(height: 32),

              // Publish button (bottom)
              ElevatedButton(
                onPressed: _submitting ? null : _onSubmit,
                style: ElevatedButton.styleFrom(
                  backgroundColor: theme.colorScheme.primary,
                  foregroundColor: Colors.white,
                  minimumSize: const Size(double.infinity, 48),
                  shape: RoundedRectangleBorder(
                    borderRadius:
                        BorderRadius.circular(AppTheme.radiusMd),
                  ),
                ),
                child: _submitting
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : Text(l10n.jobPublish),
              ),
              const SizedBox(height: 8),

              // Cancel button
              TextButton(
                onPressed: () => Navigator.of(context).pop(),
                child: Text(l10n.jobCancel),
              ),
              const SizedBox(height: 16),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Description type selector + video upload
// ---------------------------------------------------------------------------

class _DescriptionTypeSection extends StatelessWidget {
  const _DescriptionTypeSection({
    required this.descriptionType,
    required this.onDescriptionTypeChanged,
    required this.videoUrl,
    required this.videoName,
    required this.isUploading,
    required this.onPickVideo,
    required this.onRemoveVideo,
  });

  final DescriptionType descriptionType;
  final ValueChanged<DescriptionType> onDescriptionTypeChanged;
  final String videoUrl;
  final String? videoName;
  final bool isUploading;
  final VoidCallback onPickVideo;
  final VoidCallback onRemoveVideo;

  bool get _showVideoUpload =>
      descriptionType == DescriptionType.video ||
      descriptionType == DescriptionType.both;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: primary.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                ),
                child: Icon(Icons.videocam_outlined, color: primary, size: 20),
              ),
              const SizedBox(width: 12),
              Text(
                l10n.jobDescriptionType,
                style: theme.textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 16),

          // Segmented button for description type
          SegmentedButton<DescriptionType>(
            segments: [
              ButtonSegment(
                value: DescriptionType.text,
                label: Text(l10n.jobDescriptionTypeText),
                icon: const Icon(Icons.text_fields, size: 18),
              ),
              ButtonSegment(
                value: DescriptionType.video,
                label: Text(l10n.jobDescriptionTypeVideo),
                icon: const Icon(Icons.videocam, size: 18),
              ),
              ButtonSegment(
                value: DescriptionType.both,
                label: Text(l10n.jobDescriptionTypeBoth),
                icon: const Icon(Icons.dashboard, size: 18),
              ),
            ],
            selected: {descriptionType},
            onSelectionChanged: (set) => onDescriptionTypeChanged(set.first),
            style: SegmentedButton.styleFrom(
              selectedBackgroundColor: primary.withValues(alpha: 0.12),
              selectedForegroundColor: primary,
            ),
          ),

          // Video upload area
          if (_showVideoUpload) ...[
            const SizedBox(height: 16),
            if (videoUrl.isEmpty && !isUploading)
              OutlinedButton.icon(
                onPressed: onPickVideo,
                icon: const Icon(Icons.videocam_outlined),
                label: Text(l10n.jobAddVideo),
                style: OutlinedButton.styleFrom(
                  minimumSize: const Size.fromHeight(44),
                ),
              ),
            if (isUploading)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                    const SizedBox(width: 8),
                    Text(l10n.jobVideoUploading),
                  ],
                ),
              ),
            if (videoUrl.isNotEmpty && !isUploading)
              Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: Colors.green.shade50,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    const Icon(
                      Icons.check_circle,
                      color: Colors.green,
                      size: 20,
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        videoName ?? l10n.jobVideoUploaded,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(fontSize: 13),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.close, size: 18),
                      onPressed: onRemoveVideo,
                      padding: EdgeInsets.zero,
                      constraints: const BoxConstraints(),
                    ),
                  ],
                ),
              ),
          ],
        ],
      ),
    );
  }
}

