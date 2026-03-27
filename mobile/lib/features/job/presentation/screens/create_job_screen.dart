import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

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
