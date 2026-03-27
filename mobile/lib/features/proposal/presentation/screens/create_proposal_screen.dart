import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/proposal.dart';

/// Full-page form for creating a new proposal.
///
/// Receives `recipientId`, `conversationId`, and `recipientName` via
/// route extras. Collects title, description, amount, deadline.
/// Frontend-only: tapping "Send" logs data and pops the screen.
class CreateProposalScreen extends StatefulWidget {
  const CreateProposalScreen({
    super.key,
    this.recipientId = '',
    this.conversationId = '',
    this.recipientName = '',
  });

  final String recipientId;
  final String conversationId;
  final String recipientName;

  @override
  State<CreateProposalScreen> createState() => _CreateProposalScreenState();
}

class _CreateProposalScreenState extends State<CreateProposalScreen> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _descriptionController = TextEditingController();
  final _amountController = TextEditingController();

  late final ProposalFormData _formData;

  @override
  void initState() {
    super.initState();
    _formData = ProposalFormData(
      recipientId: widget.recipientId,
      conversationId: widget.conversationId,
      recipientName: widget.recipientName,
    );
  }

  @override
  void dispose() {
    _titleController.dispose();
    _descriptionController.dispose();
    _amountController.dispose();
    super.dispose();
  }

  void _onDeadlinePicked() async {
    final now = DateTime.now();
    final picked = await showDatePicker(
      context: context,
      initialDate: _formData.deadline ?? now,
      firstDate: now,
      lastDate: now.add(const Duration(days: 730)),
    );
    if (picked != null) {
      setState(() => _formData.deadline = picked);
    }
  }

  void _onSend() {
    _formData.title = _titleController.text.trim();
    _formData.description = _descriptionController.text.trim();
    _formData.amount = double.tryParse(_amountController.text) ?? 0;

    if (!_formKey.currentState!.validate()) return;

    // Mock: just log the data. Backend integration will come later.
    debugPrint('Sending proposal: ${_formData.title}, '
        '${_formData.amount} EUR, '
        'to ${_formData.recipientId}');

    Navigator.of(context).pop(_formData);
  }

  String _formatDate(DateTime date) {
    final d = date.day.toString().padLeft(2, '0');
    final m = date.month.toString().padLeft(2, '0');
    return '$d/$m/${date.year}';
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.proposalCreate),
        actions: [
          TextButton(
            onPressed: _onSend,
            child: Text(
              l10n.proposalSend,
              style: TextStyle(
                color: theme.colorScheme.primary,
                fontWeight: FontWeight.w600,
              ),
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
              // Recipient (read-only)
              _buildRecipientField(theme, appColors, l10n),
              const SizedBox(height: 20),
              const Divider(),
              const SizedBox(height: 20),

              // Title
              TextFormField(
                controller: _titleController,
                decoration: InputDecoration(
                  labelText: l10n.proposalTitle,
                  hintText: l10n.proposalTitleHint,
                ),
                maxLength: 100,
                textInputAction: TextInputAction.next,
                validator: (value) {
                  if (value == null || value.trim().isEmpty) {
                    return l10n.fieldRequired;
                  }
                  return null;
                },
              ),
              const SizedBox(height: 16),

              // Description
              TextFormField(
                controller: _descriptionController,
                decoration: InputDecoration(
                  labelText: l10n.proposalDescription,
                  hintText: l10n.proposalDescriptionHint,
                  alignLabelWithHint: true,
                ),
                maxLines: 5,
                textInputAction: TextInputAction.newline,
                validator: (value) {
                  if (value == null || value.trim().isEmpty) {
                    return l10n.fieldRequired;
                  }
                  return null;
                },
              ),
              const SizedBox(height: 16),

              // Amount
              TextFormField(
                controller: _amountController,
                decoration: InputDecoration(
                  labelText: l10n.proposalAmount,
                  hintText: l10n.proposalAmountHint,
                  prefixText: '\u20AC ',
                ),
                keyboardType: TextInputType.number,
                validator: (value) {
                  if (value == null || value.trim().isEmpty) {
                    return l10n.fieldRequired;
                  }
                  final parsed = double.tryParse(value);
                  if (parsed == null || parsed <= 0) {
                    return l10n.fieldRequired;
                  }
                  return null;
                },
              ),
              const SizedBox(height: 16),

              // Deadline (optional)
              _buildDeadlineField(theme, appColors, l10n),
              const SizedBox(height: 32),

              // Send button
              ElevatedButton(
                onPressed: _onSend,
                style: ElevatedButton.styleFrom(
                  backgroundColor: theme.colorScheme.primary,
                  foregroundColor: Colors.white,
                  minimumSize: const Size(double.infinity, 48),
                  shape: RoundedRectangleBorder(
                    borderRadius:
                        BorderRadius.circular(AppTheme.radiusMd),
                  ),
                ),
                child: Text(l10n.proposalSend),
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

  Widget _buildRecipientField(
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    final name = widget.recipientName.isNotEmpty
        ? widget.recipientName
        : 'User ${widget.recipientId.length > 8 ? widget.recipientId.substring(0, 8) : widget.recipientId}';

    final initials = name
        .split(' ')
        .map((w) => w.isNotEmpty ? w[0] : '')
        .join('')
        .toUpperCase();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.proposalRecipient,
          style: theme.textTheme.bodySmall?.copyWith(
            fontWeight: FontWeight.w500,
          ),
        ),
        const SizedBox(height: 8),
        Row(
          children: [
            CircleAvatar(
              radius: 20,
              backgroundColor: theme.colorScheme.primary,
              child: Text(
                initials.isNotEmpty ? initials : '?',
                style: const TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.w600,
                  fontSize: 14,
                ),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Text(
                name,
                style: theme.textTheme.titleMedium,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildDeadlineField(
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    return GestureDetector(
      onTap: _onDeadlinePicked,
      child: AbsorbPointer(
        child: TextFormField(
          decoration: InputDecoration(
            labelText: l10n.proposalDeadline,
            suffixIcon: const Icon(
              Icons.calendar_today_outlined,
              size: 18,
            ),
          ),
          controller: TextEditingController(
            text: _formData.deadline != null
                ? _formatDate(_formData.deadline!)
                : '',
          ),
        ),
      ),
    );
  }
}
