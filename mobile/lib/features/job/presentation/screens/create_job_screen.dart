import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/job.dart';
import '../widgets/budget_section.dart';
import '../widgets/job_details_section.dart';

/// Full-page scrollable form for creating a new job posting.
///
/// Composed of two expandable sections delegated to dedicated widget files.
/// Frontend-only: tapping "Continue" shows a snackbar (no backend call).
class CreateJobScreen extends StatefulWidget {
  const CreateJobScreen({super.key});

  @override
  State<CreateJobScreen> createState() => _CreateJobScreenState();
}

class _CreateJobScreenState extends State<CreateJobScreen> {
  final _formKey = GlobalKey<FormState>();

  // Text controllers
  final _titleController = TextEditingController();
  final _descriptionController = TextEditingController();
  final _minRateController = TextEditingController();
  final _maxRateController = TextEditingController();
  final _minBudgetController = TextEditingController();
  final _maxBudgetController = TextEditingController();
  final _durationController = TextEditingController();

  // Form data
  late final JobFormData _formData;

  // Expansion state
  bool _detailsExpanded = true;
  bool _budgetExpanded = false;

  @override
  void initState() {
    super.initState();
    _formData = JobFormData();
  }

  @override
  void dispose() {
    _titleController.dispose();
    _descriptionController.dispose();
    _minRateController.dispose();
    _maxRateController.dispose();
    _minBudgetController.dispose();
    _maxBudgetController.dispose();
    _durationController.dispose();
    super.dispose();
  }

  // -----------------------------------------------------------------------
  // Section 1 callbacks
  // -----------------------------------------------------------------------

  void _onSkillAdded(String skill) {
    setState(() => _formData.skills.add(skill));
  }

  void _onSkillRemoved(int index) {
    setState(() => _formData.skills.removeAt(index));
  }

  void _onToolAdded(String tool) {
    setState(() => _formData.tools.add(tool));
  }

  void _onToolRemoved(int index) {
    setState(() => _formData.tools.removeAt(index));
  }

  void _onContractorCountChanged(int count) {
    setState(() => _formData.contractorCount = count);
  }

  void _onApplicantTypeChanged(ApplicantType type) {
    setState(() => _formData.applicantType = type);
  }

  // -----------------------------------------------------------------------
  // Section 2 callbacks
  // -----------------------------------------------------------------------

  void _onBudgetTypeChanged(BudgetType type) {
    setState(() => _formData.budgetType = type);
  }

  void _onPaymentFrequencyChanged(PaymentFrequency frequency) {
    setState(() => _formData.paymentFrequency = frequency);
  }

  void _onMaxHoursChanged(int hours) {
    setState(() => _formData.maxHoursPerWeek = hours);
  }

  void _onDurationUnitChanged(DurationUnit unit) {
    setState(() => _formData.durationUnit = unit);
  }

  void _onIndefiniteChanged(bool value) {
    setState(() {
      _formData.isIndefinite = value;
      if (value) _durationController.clear();
    });
  }

  // -----------------------------------------------------------------------
  // Submit
  // -----------------------------------------------------------------------

  void _onContinue() {
    _formData.title = _titleController.text.trim();
    _formData.description = _descriptionController.text.trim();
    _formData.minRate = _minRateController.text.trim();
    _formData.maxRate = _maxRateController.text.trim();
    _formData.minBudget = _minBudgetController.text.trim();
    _formData.maxBudget = _maxBudgetController.text.trim();
    _formData.estimatedDuration = _durationController.text.trim();

    if (!_formKey.currentState!.validate()) return;

    final l10n = AppLocalizations.of(context)!;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(l10n.jobSave)),
    );
    Navigator.of(context).pop();
  }

  // -----------------------------------------------------------------------
  // Build
  // -----------------------------------------------------------------------

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
              onPressed: _onContinue,
              style: FilledButton.styleFrom(
                backgroundColor: theme.colorScheme.primary,
                foregroundColor: Colors.white,
                shape: RoundedRectangleBorder(
                  borderRadius:
                      BorderRadius.circular(AppTheme.radiusSm),
                ),
              ),
              child: Text(l10n.jobContinue),
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
                tools: _formData.tools,
                onToolAdded: _onToolAdded,
                onToolRemoved: _onToolRemoved,
                contractorCount: _formData.contractorCount,
                onContractorCountChanged: _onContractorCountChanged,
                applicantType: _formData.applicantType,
                onApplicantTypeChanged: _onApplicantTypeChanged,
                isExpanded: _detailsExpanded,
                onExpansionChanged: (expanded) {
                  setState(() => _detailsExpanded = expanded);
                },
              ),
              const SizedBox(height: 16),

              // Section 2: Budget and duration
              BudgetSection(
                budgetType: _formData.budgetType,
                onBudgetTypeChanged: _onBudgetTypeChanged,
                paymentFrequency: _formData.paymentFrequency,
                onPaymentFrequencyChanged: _onPaymentFrequencyChanged,
                minRateController: _minRateController,
                maxRateController: _maxRateController,
                maxHoursPerWeek: _formData.maxHoursPerWeek,
                onMaxHoursChanged: _onMaxHoursChanged,
                minBudgetController: _minBudgetController,
                maxBudgetController: _maxBudgetController,
                durationController: _durationController,
                durationUnit: _formData.durationUnit,
                onDurationUnitChanged: _onDurationUnitChanged,
                isIndefinite: _formData.isIndefinite,
                onIndefiniteChanged: _onIndefiniteChanged,
                isExpanded: _budgetExpanded,
                onExpansionChanged: (expanded) {
                  setState(() => _budgetExpanded = expanded);
                },
              ),
              const SizedBox(height: 32),

              // Continue button (bottom)
              ElevatedButton(
                onPressed: _onContinue,
                style: ElevatedButton.styleFrom(
                  backgroundColor: theme.colorScheme.primary,
                  foregroundColor: Colors.white,
                  minimumSize: const Size(double.infinity, 48),
                  shape: RoundedRectangleBorder(
                    borderRadius:
                        BorderRadius.circular(AppTheme.radiusMd),
                  ),
                ),
                child: Text(l10n.jobContinue),
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
