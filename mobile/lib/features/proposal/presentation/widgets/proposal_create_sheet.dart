import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/proposal.dart';
import 'proposal_milestone_card.dart';
import 'proposal_payment_selector.dart';
import 'proposal_skills_input.dart';

/// Full-screen modal bottom sheet for creating a new proposal.
///
/// Collects title, description, payment type, milestones/amount,
/// skills, dates, and negotiable flag. Mock-only — no backend call.
class ProposalCreateSheet extends StatefulWidget {
  const ProposalCreateSheet({super.key, this.onSend});

  /// Called with the completed form data when the user taps "Send".
  final ValueChanged<ProposalFormData>? onSend;

  /// Shows the proposal creation sheet as a full-screen modal.
  static Future<ProposalFormData?> show(BuildContext context) {
    return showModalBottomSheet<ProposalFormData>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(
          top: Radius.circular(AppTheme.radiusXl),
        ),
      ),
      builder: (_) => const ProposalCreateSheet(),
    );
  }

  @override
  State<ProposalCreateSheet> createState() => _ProposalCreateSheetState();
}

class _ProposalCreateSheetState extends State<ProposalCreateSheet> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _descriptionController = TextEditingController();
  final _amountController = TextEditingController();

  late final ProposalFormData _formData;

  @override
  void initState() {
    super.initState();
    _formData = ProposalFormData();
  }

  @override
  void dispose() {
    _titleController.dispose();
    _descriptionController.dispose();
    _amountController.dispose();
    super.dispose();
  }

  // -----------------------------------------------------------------------
  // Callbacks
  // -----------------------------------------------------------------------

  void _onPaymentTypeChanged(ProposalPaymentType type) {
    setState(() => _formData.paymentType = type);
  }

  void _onMilestoneAdded() {
    setState(() => _formData.milestones.add(ProposalMilestone()));
  }

  void _onMilestoneRemoved(int index) {
    setState(() => _formData.milestones.removeAt(index));
  }

  void _onMilestoneChanged() {
    setState(() {});
  }

  void _onSkillAdded(String skill) {
    setState(() => _formData.skills.add(skill));
  }

  void _onSkillRemoved(int index) {
    setState(() => _formData.skills.removeAt(index));
  }

  void _onStartDatePicked() => _pickDate(
        initial: _formData.startDate,
        onPicked: (d) => setState(() => _formData.startDate = d),
      );

  void _onDeadlinePicked() => _pickDate(
        initial: _formData.deadline,
        onPicked: (d) => setState(() => _formData.deadline = d),
      );

  void _onNegotiableChanged(bool value) {
    setState(() => _formData.negotiable = value);
  }

  void _onSend() {
    _formData.title = _titleController.text.trim();
    _formData.description = _descriptionController.text.trim();
    if (_formData.paymentType == ProposalPaymentType.invoice) {
      _formData.amount = double.tryParse(_amountController.text) ?? 0;
    }

    if (!_formKey.currentState!.validate()) return;

    widget.onSend?.call(_formData);
    Navigator.of(context).pop(_formData);
  }

  Future<void> _pickDate({
    DateTime? initial,
    required ValueChanged<DateTime> onPicked,
  }) async {
    final now = DateTime.now();
    final picked = await showDatePicker(
      context: context,
      initialDate: initial ?? now,
      firstDate: now,
      lastDate: now.add(const Duration(days: 730)),
    );
    if (picked != null) onPicked(picked);
  }

  String _formatDate(DateTime date) {
    final d = date.day.toString().padLeft(2, '0');
    final m = date.month.toString().padLeft(2, '0');
    return '$d/$m/${date.year}';
  }

  // -----------------------------------------------------------------------
  // Build
  // -----------------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return DraggableScrollableSheet(
      initialChildSize: 0.92,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (context, scrollController) {
        return Column(
          children: [
            _buildHandle(appColors),
            _buildHeader(l10n, theme),
            const Divider(height: 1),
            Expanded(
              child: Form(
                key: _formKey,
                child: ListView(
                  controller: scrollController,
                  padding: const EdgeInsets.all(16),
                  children: [
                    _buildTitleField(l10n),
                    const SizedBox(height: 16),
                    _buildDescriptionField(l10n),
                    const SizedBox(height: 24),
                    _buildPaymentSection(l10n, theme, appColors),
                    const SizedBox(height: 24),
                    _buildSkillsSection(l10n, theme),
                    const SizedBox(height: 24),
                    _buildDatesSection(l10n, theme, appColors),
                    const SizedBox(height: 16),
                    _buildNegotiableToggle(l10n, theme),
                    const SizedBox(height: 32),
                    _buildSendButton(l10n, theme),
                    const SizedBox(height: 16),
                  ],
                ),
              ),
            ),
          ],
        );
      },
    );
  }

  Widget _buildHandle(AppColors? appColors) {
    return Padding(
      padding: const EdgeInsets.only(top: 12, bottom: 4),
      child: Container(
        width: 40,
        height: 4,
        decoration: BoxDecoration(
          color: appColors?.border ?? Colors.grey[300],
          borderRadius: BorderRadius.circular(2),
        ),
      ),
    );
  }

  Widget _buildHeader(AppLocalizations l10n, ThemeData theme) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        children: [
          Icon(
            Icons.description_outlined,
            color: theme.colorScheme.primary,
            size: 24,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              l10n.proposalCreate,
              style: theme.textTheme.titleLarge,
            ),
          ),
          IconButton(
            onPressed: () => Navigator.of(context).pop(),
            icon: const Icon(Icons.close, size: 24),
          ),
        ],
      ),
    );
  }

  Widget _buildTitleField(AppLocalizations l10n) {
    return TextFormField(
      controller: _titleController,
      decoration: InputDecoration(labelText: l10n.proposalTitle),
      maxLength: 100,
      textInputAction: TextInputAction.next,
      validator: (value) {
        if (value == null || value.trim().isEmpty) {
          return l10n.fieldRequired;
        }
        return null;
      },
    );
  }

  Widget _buildDescriptionField(AppLocalizations l10n) {
    return TextFormField(
      controller: _descriptionController,
      decoration: InputDecoration(
        labelText: l10n.projectDescription,
        alignLabelWithHint: true,
      ),
      maxLines: 4,
      textInputAction: TextInputAction.newline,
    );
  }

  Widget _buildPaymentSection(
    AppLocalizations l10n,
    ThemeData theme,
    AppColors? appColors,
  ) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.paymentType, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        ProposalPaymentSelector(
          selected: _formData.paymentType,
          onChanged: _onPaymentTypeChanged,
        ),
        const SizedBox(height: 16),
        if (_formData.paymentType == ProposalPaymentType.escrow)
          _buildMilestonesSection(l10n, theme, appColors)
        else
          _buildInvoiceAmount(l10n),
      ],
    );
  }

  Widget _buildMilestonesSection(
    AppLocalizations l10n,
    ThemeData theme,
    AppColors? appColors,
  ) {
    return Column(
      children: [
        for (int i = 0; i < _formData.milestones.length; i++)
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: ProposalMilestoneCard(
              index: i,
              milestone: _formData.milestones[i],
              canDelete: _formData.milestones.length > 1,
              onDelete: () => _onMilestoneRemoved(i),
              onChanged: _onMilestoneChanged,
            ),
          ),
        // Total display
        if (_formData.milestones.isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  l10n.proposalTotalAmount,
                  style: theme.textTheme.titleMedium,
                ),
                Text(
                  '\u20AC ${_formData.totalMilestoneAmount.toStringAsFixed(2)}',
                  style: theme.textTheme.titleMedium?.copyWith(
                    color: theme.colorScheme.primary,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
          ),
        SizedBox(
          width: double.infinity,
          child: OutlinedButton.icon(
            onPressed: _onMilestoneAdded,
            icon: const Icon(Icons.add, size: 18),
            label: Text(l10n.addMilestone),
            style: OutlinedButton.styleFrom(
              foregroundColor: theme.colorScheme.primary,
              side: BorderSide(
                color: theme.colorScheme.primary.withValues(alpha: 0.3),
              ),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildInvoiceAmount(AppLocalizations l10n) {
    return TextFormField(
      controller: _amountController,
      decoration: InputDecoration(
        labelText: l10n.proposalTotalAmount,
        prefixText: '\u20AC ',
      ),
      keyboardType: TextInputType.number,
      validator: (value) {
        if (value == null || value.trim().isEmpty) {
          return l10n.fieldRequired;
        }
        return null;
      },
    );
  }

  Widget _buildSkillsSection(AppLocalizations l10n, ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.requiredSkills, style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        ProposalSkillsInput(
          skills: _formData.skills,
          onAdded: _onSkillAdded,
          onRemoved: _onSkillRemoved,
        ),
      ],
    );
  }

  Widget _buildDatesSection(
    AppLocalizations l10n,
    ThemeData theme,
    AppColors? appColors,
  ) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.timeline, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: _buildDateField(
                label: l10n.startDate,
                value: _formData.startDate,
                onTap: _onStartDatePicked,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: _buildDateField(
                label: l10n.deadline,
                value: _formData.deadline,
                onTap: _onDeadlinePicked,
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildDateField({
    required String label,
    required DateTime? value,
    required VoidCallback onTap,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: AbsorbPointer(
        child: TextFormField(
          decoration: InputDecoration(
            labelText: label,
            suffixIcon: const Icon(
              Icons.calendar_today_outlined,
              size: 18,
            ),
          ),
          controller: TextEditingController(
            text: value != null ? _formatDate(value) : '',
          ),
        ),
      ),
    );
  }

  Widget _buildNegotiableToggle(AppLocalizations l10n, ThemeData theme) {
    return Row(
      children: [
        Expanded(
          child: Text(l10n.proposalNegotiable, style: theme.textTheme.bodyMedium),
        ),
        Switch(
          value: _formData.negotiable,
          onChanged: _onNegotiableChanged,
          activeTrackColor: theme.colorScheme.primary,
          activeThumbColor: Colors.white,
        ),
      ],
    );
  }

  Widget _buildSendButton(AppLocalizations l10n, ThemeData theme) {
    return ElevatedButton.icon(
      onPressed: _onSend,
      icon: const Icon(Icons.send, size: 18),
      label: Text(l10n.proposalSend),
      style: ElevatedButton.styleFrom(
        backgroundColor: theme.colorScheme.primary,
        foregroundColor: Colors.white,
        minimumSize: const Size(double.infinity, 48),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        ),
      ),
    );
  }
}
