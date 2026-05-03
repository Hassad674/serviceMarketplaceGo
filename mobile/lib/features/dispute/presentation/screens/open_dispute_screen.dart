import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../proposal/presentation/providers/proposal_provider.dart';
import '../../data/dispute_uploader.dart';
import '../../types/dispute.dart';
import '../providers/dispute_provider.dart';
import '../../../../core/theme/app_palette.dart';

/// Form screen for opening a dispute on a proposal.
///
/// Adapted from `create_proposal_screen.dart` patterns.
class OpenDisputeScreen extends ConsumerStatefulWidget {
  const OpenDisputeScreen({
    super.key,
    required this.proposalId,
    required this.proposalAmount,
    required this.userRole,
  });

  final String proposalId;
  final int proposalAmount;
  final String userRole; // 'client' | 'provider'

  @override
  ConsumerState<OpenDisputeScreen> createState() => _OpenDisputeScreenState();
}

class _OpenDisputeScreenState extends ConsumerState<OpenDisputeScreen> {
  final _formKey = GlobalKey<FormState>();
  final _messageController = TextEditingController();
  final _descriptionController = TextEditingController();
  final _partialAmountController = TextEditingController();

  late final DisputeFormData _formData;
  bool _isSubmitting = false;

  @override
  void initState() {
    super.initState();
    _formData = DisputeFormData();
  }

  @override
  void dispose() {
    _messageController.dispose();
    _descriptionController.dispose();
    _partialAmountController.dispose();
    super.dispose();
  }

  Future<void> _onAddFiles() async {
    final result = await FilePicker.platform.pickFiles(allowMultiple: true);
    if (result == null) return;
    setState(() {
      _formData.attachments.addAll(
        result.files.where((f) => f.path != null).map((f) => File(f.path!)),
      );
    });
  }

  void _removeFile(int index) {
    setState(() => _formData.attachments.removeAt(index));
  }

  Future<void> _onSubmit() async {
    if (!_formKey.currentState!.validate()) return;
    if (_formData.reason == null) {
      _showError(AppLocalizations.of(context)!.disputeReasonPlaceholder);
      return;
    }

    setState(() => _isSubmitting = true);

    final requestedAmount = _formData.amountType == AmountType.total
        ? widget.proposalAmount
        : (double.tryParse(_partialAmountController.text) ?? 0).round() * 100;

    try {
      // 1. Upload files
      final apiClient = ref.read(apiClientProvider);
      final attachments = _formData.attachments.isEmpty
          ? <Map<String, dynamic>>[]
          : await uploadDisputeFiles(apiClient, _formData.attachments);

      // 2. Open the dispute
      final dispute = await openDispute(
        ref,
        proposalId: widget.proposalId,
        reason: _formData.reason!.value,
        description: _descriptionController.text.trim(),
        messageToParty: _messageController.text.trim(),
        requestedAmount: requestedAmount,
        attachments: attachments,
      );

      if (!mounted) return;

      if (dispute == null) {
        setState(() => _isSubmitting = false);
        _showError(AppLocalizations.of(context)!.unexpectedError);
        return;
      }

      // 3. Refresh the proposal so the banner shows
      ref.invalidate(proposalByIdProvider(widget.proposalId));
      ref.invalidate(projectsProvider);

      GoRouter.of(context).pop(true);
    } catch (e) {
      if (mounted) {
        setState(() => _isSubmitting = false);
        _showError('${AppLocalizations.of(context)!.unexpectedError}: $e');
      }
    }
  }

  void _showError(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  String _formatEur(int centimes) {
    return NumberFormat.currency(locale: 'fr_FR', symbol: '€', decimalDigits: 2)
        .format(centimes / 100);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final reasons = DisputeReason.forRole(widget.userRole);

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.disputeReportProblem),
        actions: [
          TextButton(
            onPressed: _isSubmitting ? null : _onSubmit,
            child: Text(
              l10n.disputeSubmit,
              style: TextStyle(
                color: _isSubmitting
                    ? theme.disabledColor
                    : theme.colorScheme.primary,
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
              _WarningBanner(text: l10n.disputeFormWarning),
              const SizedBox(height: 16),
              _buildReasonField(l10n, reasons),
              const SizedBox(height: 16),
              _buildAmountField(l10n),
              const SizedBox(height: 16),
              _buildMessageField(l10n),
              const SizedBox(height: 8),
              _AttachmentsSection(
                files: _formData.attachments,
                onAdd: _onAddFiles,
                onRemove: _removeFile,
                addLabel: l10n.disputeAddFiles,
              ),
              const SizedBox(height: 20),
              _buildDescriptionField(l10n),
              const SizedBox(height: 24),
              if (_isSubmitting)
                Center(
                  child: Padding(
                    padding: const EdgeInsets.all(8.0),
                    child: CircularProgressIndicator(
                      color: theme.colorScheme.primary,
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildReasonField(
    AppLocalizations l10n,
    List<DisputeReason> reasons,
  ) {
    return DropdownButtonFormField<DisputeReason>(
      initialValue: _formData.reason,
      decoration: InputDecoration(labelText: l10n.disputeReasonLabel),
      hint: Text(l10n.disputeReasonPlaceholder),
      items: reasons
          .map((r) => DropdownMenuItem(
                value: r,
                child: Text(_reasonLabel(l10n, r)),
              ))
          .toList(),
      onChanged: (v) => setState(() => _formData.reason = v),
    );
  }

  Widget _buildAmountField(AppLocalizations l10n) {
    final fullStr = _formatEur(widget.proposalAmount);
    final fullLabel = widget.userRole == 'client'
        ? l10n.disputeTotalRefund(fullStr)
        : l10n.disputeTotalRelease(fullStr);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.disputeAmountLabel,
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
        ),
        RadioListTile<AmountType>(
          contentPadding: EdgeInsets.zero,
          dense: true,
          title: Text(fullLabel),
          value: AmountType.total,
          groupValue: _formData.amountType,
          onChanged: (v) =>
              setState(() => _formData.amountType = v ?? AmountType.total),
        ),
        RadioListTile<AmountType>(
          contentPadding: EdgeInsets.zero,
          dense: true,
          title: Text(l10n.disputePartialAmount),
          value: AmountType.partial,
          groupValue: _formData.amountType,
          onChanged: (v) =>
              setState(() => _formData.amountType = v ?? AmountType.partial),
        ),
        if (_formData.amountType == AmountType.partial)
          Padding(
            padding: const EdgeInsets.only(left: 16, top: 4),
            child: TextFormField(
              controller: _partialAmountController,
              decoration: const InputDecoration(prefixText: '€ '),
              keyboardType: TextInputType.number,
              validator: (v) {
                if (_formData.amountType != AmountType.partial) return null;
                final n = double.tryParse(v ?? '');
                if (n == null || n <= 0) return l10n.fieldRequired;
                if ((n * 100) > widget.proposalAmount) {
                  return l10n.unexpectedError;
                }
                return null;
              },
            ),
          ),
      ],
    );
  }

  Widget _buildMessageField(AppLocalizations l10n) {
    return TextFormField(
      controller: _messageController,
      decoration: InputDecoration(
        labelText: l10n.disputeMessageToPartyLabel,
        hintText: l10n.disputeMessageToPartyPlaceholder,
        helperText: l10n.disputeMessageToPartyHint,
        helperMaxLines: 2,
        alignLabelWithHint: true,
      ),
      maxLines: 4,
      maxLength: 2000,
      validator: (value) {
        if (value == null || value.trim().isEmpty) return l10n.fieldRequired;
        return null;
      },
    );
  }

  Widget _buildDescriptionField(AppLocalizations l10n) {
    return TextFormField(
      controller: _descriptionController,
      decoration: InputDecoration(
        labelText: l10n.disputeDescriptionLabel,
        hintText: l10n.disputeDescriptionPlaceholder,
        helperText: l10n.disputeDescriptionHint,
        helperMaxLines: 2,
        alignLabelWithHint: true,
      ),
      maxLines: 4,
      maxLength: 5000,
    );
  }

  String _reasonLabel(AppLocalizations l10n, DisputeReason reason) {
    switch (reason) {
      case DisputeReason.workNotConforming:
        return l10n.disputeReasonWorkNotConforming;
      case DisputeReason.nonDelivery:
        return l10n.disputeReasonNonDelivery;
      case DisputeReason.insufficientQuality:
        return l10n.disputeReasonInsufficientQuality;
      case DisputeReason.clientGhosting:
        return l10n.disputeReasonClientGhosting;
      case DisputeReason.scopeCreep:
        return l10n.disputeReasonScopeCreep;
      case DisputeReason.refusalToValidate:
        return l10n.disputeReasonRefusalToValidate;
      case DisputeReason.harassment:
        return l10n.disputeReasonHarassment;
      case DisputeReason.other:
        return l10n.disputeReasonOther;
    }
  }
}

// ---------------------------------------------------------------------------
// Sub-widgets
// ---------------------------------------------------------------------------

class _WarningBanner extends StatelessWidget {
  const _WarningBanner({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppPalette.amber100,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: Border.all(color: AppPalette.amber300),
      ),
      child: Row(
        children: [
          const Icon(Icons.warning_amber_rounded,
              color: AppPalette.amber800, size: 20),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              text,
              style: const TextStyle(color: AppPalette.amber800, fontSize: 13),
            ),
          ),
        ],
      ),
    );
  }
}

class _AttachmentsSection extends StatelessWidget {
  const _AttachmentsSection({
    required this.files,
    required this.onAdd,
    required this.onRemove,
    required this.addLabel,
  });

  final List<File> files;
  final VoidCallback onAdd;
  final ValueChanged<int> onRemove;
  final String addLabel;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (files.isNotEmpty)
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: List.generate(files.length, (i) {
              final f = files[i];
              final name = f.path.split('/').last;
              return Chip(
                avatar: const Icon(Icons.insert_drive_file, size: 14),
                label: Text(name, style: const TextStyle(fontSize: 11)),
                onDeleted: () => onRemove(i),
                deleteIconColor: Colors.grey,
              );
            }),
          ),
        TextButton.icon(
          onPressed: onAdd,
          icon: const Icon(Icons.attach_file, size: 16),
          label: Text(addLabel),
        ),
      ],
    );
  }
}
