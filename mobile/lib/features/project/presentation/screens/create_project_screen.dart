import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/project.dart';
import '../widgets/applicant_section.dart';
import '../widgets/escrow_structure_section.dart';
import '../widgets/invoice_billing_section.dart';
import '../widgets/payment_type_selector.dart';
import '../widgets/project_details_section.dart';
import '../widgets/timeline_section.dart';

/// Full-page scrollable form for creating a new project.
///
/// Composed of five sections delegated to dedicated widget files.
/// Frontend-only: tapping "Publish" shows a snackbar (no backend call).
class CreateProjectScreen extends StatefulWidget {
  const CreateProjectScreen({super.key});

  @override
  State<CreateProjectScreen> createState() => _CreateProjectScreenState();
}

class _CreateProjectScreenState extends State<CreateProjectScreen> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _descriptionController = TextEditingController();

  late final ProjectFormData _formData;

  @override
  void initState() {
    super.initState();
    _formData = ProjectFormData();
  }

  @override
  void dispose() {
    _titleController.dispose();
    _descriptionController.dispose();
    super.dispose();
  }

  // -----------------------------------------------------------------------
  // Form callbacks
  // -----------------------------------------------------------------------

  void _onPaymentTypeChanged(PaymentType type) {
    setState(() => _formData.paymentType = type);
  }

  void _onStructureChanged(ProjectStructure structure) {
    setState(() => _formData.structure = structure);
  }

  void _onBillingTypeChanged(BillingType type) {
    setState(() => _formData.billingType = type);
  }

  void _onRateChanged(double rate) {
    _formData.rate = rate;
  }

  void _onFrequencyChanged(BillingFrequency frequency) {
    setState(() => _formData.frequency = frequency);
  }

  void _onAmountChanged(double amount) {
    _formData.amount = amount;
  }

  void _onMilestoneAdded() {
    setState(() => _formData.milestones.add(MilestoneData()));
  }

  void _onMilestoneRemoved(int index) {
    setState(() => _formData.milestones.removeAt(index));
  }

  void _onMilestoneChanged() {
    // Trigger rebuild for total calculation if needed.
    setState(() {});
  }

  void _onSkillAdded(String skill) {
    setState(() => _formData.skills.add(skill));
  }

  void _onSkillRemoved(int index) {
    setState(() => _formData.skills.removeAt(index));
  }

  void _onStartDateChanged(DateTime? date) {
    setState(() => _formData.startDate = date);
  }

  void _onDeadlineChanged(DateTime? date) {
    setState(() => _formData.deadline = date);
  }

  void _onOngoingChanged(bool value) {
    setState(() {
      _formData.ongoing = value;
      if (value) _formData.deadline = null;
    });
  }

  void _onApplicantTypeChanged(ApplicantType type) {
    setState(() => _formData.applicantType = type);
  }

  void _onNegotiableChanged(bool value) {
    setState(() => _formData.negotiable = value);
  }

  void _onPublish() {
    _formData.title = _titleController.text.trim();
    _formData.description = _descriptionController.text.trim();

    if (!_formKey.currentState!.validate()) return;

    final l10n = AppLocalizations.of(context)!;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(l10n.projectPublished)),
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
      appBar: AppBar(title: Text(l10n.createProject)),
      body: SafeArea(
        child: Form(
          key: _formKey,
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              // Section 1: Payment type
              PaymentTypeSelector(
                selected: _formData.paymentType,
                onChanged: _onPaymentTypeChanged,
              ),
              const SizedBox(height: 24),

              // Section 2: Structure (conditional)
              if (_formData.paymentType == PaymentType.escrow)
                EscrowStructureSection(
                  structure: _formData.structure,
                  onStructureChanged: _onStructureChanged,
                  milestones: _formData.milestones,
                  amount: _formData.amount,
                  onAmountChanged: _onAmountChanged,
                  onMilestoneAdded: _onMilestoneAdded,
                  onMilestoneRemoved: _onMilestoneRemoved,
                  onMilestoneChanged: _onMilestoneChanged,
                )
              else
                InvoiceBillingSection(
                  billingType: _formData.billingType,
                  onBillingTypeChanged: _onBillingTypeChanged,
                  rate: _formData.rate,
                  onRateChanged: _onRateChanged,
                  frequency: _formData.frequency,
                  onFrequencyChanged: _onFrequencyChanged,
                ),
              const SizedBox(height: 24),
              const Divider(),
              const SizedBox(height: 24),

              // Section 3: Details
              ProjectDetailsSection(
                titleController: _titleController,
                descriptionController: _descriptionController,
                skills: _formData.skills,
                onSkillAdded: _onSkillAdded,
                onSkillRemoved: _onSkillRemoved,
              ),
              const SizedBox(height: 24),
              const Divider(),
              const SizedBox(height: 24),

              // Section 4: Timeline
              TimelineSection(
                startDate: _formData.startDate,
                deadline: _formData.deadline,
                ongoing: _formData.ongoing,
                onStartDateChanged: _onStartDateChanged,
                onDeadlineChanged: _onDeadlineChanged,
                onOngoingChanged: _onOngoingChanged,
              ),
              const SizedBox(height: 24),
              const Divider(),
              const SizedBox(height: 24),

              // Section 5: Who can apply
              ApplicantSection(
                applicantType: _formData.applicantType,
                onApplicantTypeChanged: _onApplicantTypeChanged,
                negotiable: _formData.negotiable,
                onNegotiableChanged: _onNegotiableChanged,
              ),
              const SizedBox(height: 32),

              // Publish button
              ElevatedButton(
                onPressed: _onPublish,
                style: ElevatedButton.styleFrom(
                  backgroundColor: theme.colorScheme.primary,
                  foregroundColor: Colors.white,
                  minimumSize: const Size(double.infinity, 48),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                  ),
                ),
                child: Text(l10n.publishProject),
              ),
              const SizedBox(height: 8),

              // Cancel button
              TextButton(
                onPressed: () => Navigator.of(context).pop(),
                child: Text(l10n.cancel),
              ),
              const SizedBox(height: 16),
            ],
          ),
        ),
      ),
    );
  }
}
