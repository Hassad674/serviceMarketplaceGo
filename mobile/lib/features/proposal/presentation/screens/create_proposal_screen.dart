import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../billing/presentation/widgets/fee_preview_widget.dart';
import '../../domain/entities/proposal_entity.dart';
import '../../domain/repositories/proposal_repository.dart';
import '../../types/proposal.dart';
import '../providers/proposal_provider.dart';

/// Full-page form for creating or modifying a proposal.
///
/// Receives `recipientId`, `conversationId`, and `recipientName` via
/// route extras. When `existingProposal` is provided, pre-fills the form
/// for a counter-offer (modify mode).
class CreateProposalScreen extends ConsumerStatefulWidget {
  const CreateProposalScreen({
    super.key,
    this.recipientId = '',
    this.conversationId = '',
    this.recipientName = '',
    this.existingProposal,
  });

  final String recipientId;
  final String conversationId;
  final String recipientName;

  /// When non-null, the screen acts in "modify" mode (counter-offer).
  final ProposalEntity? existingProposal;

  @override
  ConsumerState<CreateProposalScreen> createState() =>
      _CreateProposalScreenState();
}

class _CreateProposalScreenState extends ConsumerState<CreateProposalScreen> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _descriptionController = TextEditingController();
  final _amountController = TextEditingController();

  late final ProposalFormData _formData;
  bool _isSubmitting = false;

  /// Debounced amount in cents used to drive the fee preview. Null when the
  /// input is empty or non-positive so the widget renders nothing.
  int? _debouncedAmountCents;
  Timer? _amountDebounce;

  bool get _isModifyMode => widget.existingProposal != null;

  @override
  void initState() {
    super.initState();
    _formData = ProposalFormData(
      recipientId: widget.recipientId,
      conversationId: widget.conversationId,
      recipientName: widget.recipientName,
    );

    // Pre-fill form in modify mode.
    if (_isModifyMode) {
      final p = widget.existingProposal!;
      _titleController.text = p.title;
      _descriptionController.text = p.description;
      _amountController.text = p.amountInEuros.toStringAsFixed(2);
      _debouncedAmountCents =
          (p.amountInEuros * 100).round().clamp(0, 1 << 31);
      if (p.deadline != null) {
        try {
          _formData.deadline = DateTime.parse(p.deadline!);
        } catch (_) {}
      }
    }

    _amountController.addListener(_onAmountChanged);
  }

  @override
  void dispose() {
    _amountDebounce?.cancel();
    _amountController.removeListener(_onAmountChanged);
    _titleController.dispose();
    _descriptionController.dispose();
    _amountController.dispose();
    super.dispose();
  }

  /// Debounces the amount field (300ms) before refreshing the fee preview.
  /// A shorter window makes the preview flicker while typing; a longer one
  /// feels unresponsive.
  void _onAmountChanged() {
    _amountDebounce?.cancel();
    _amountDebounce = Timer(const Duration(milliseconds: 300), () {
      final parsed = double.tryParse(_amountController.text.trim());
      final cents =
          parsed == null || parsed <= 0 ? null : (parsed * 100).round();
      if (!mounted) return;
      if (cents == _debouncedAmountCents) return;
      setState(() => _debouncedAmountCents = cents);
    });
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

  Future<void> _onSend() async {
    _formData.title = _titleController.text.trim();
    _formData.description = _descriptionController.text.trim();
    _formData.amount = double.tryParse(_amountController.text) ?? 0;

    if (!_formKey.currentState!.validate()) return;

    setState(() => _isSubmitting = true);

    final amountCentimes = (_formData.amount * 100).round();
    final deadlineIso = _formData.deadline?.toUtc().toIso8601String();

    try {
      final repo = ref.read(proposalRepositoryProvider);

      if (_isModifyMode) {
        final modified = await repo.modifyProposal(
          widget.existingProposal!.id,
          ModifyProposalData(
            title: _formData.title,
            description: _formData.description,
            amount: amountCentimes,
            deadline: deadlineIso,
          ),
        );
        if (mounted) {
          GoRouter.of(context).pop(modified);
        }
      } else {
        final created = await repo.createProposal(
          CreateProposalData(
            recipientId: _formData.recipientId,
            conversationId: _formData.conversationId,
            title: _formData.title,
            description: _formData.description,
            amount: amountCentimes,
            deadline: deadlineIso,
          ),
        );
        if (mounted) {
          GoRouter.of(context).pop(created);
        }
      }
    } catch (e) {
      if (mounted) {
        setState(() => _isSubmitting = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${AppLocalizations.of(context)!.unexpectedError}: $e')),
        );
      }
    }
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
    final sendLabel =
        _isModifyMode ? l10n.proposalModify : l10n.proposalSend;

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.proposalCreate),
        actions: [
          TextButton(
            onPressed: _isSubmitting ? null : _onSend,
            child: Text(
              sendLabel,
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

              // Platform fee preview — debounced, only renders once the
              // amount is parseable and positive. The FeePreviewWidget
              // enforces its own role gate: when the backend resolves
              // the viewer as client-side (via recipientId), it renders
              // nothing at all. No extra check needed here.
              if (_debouncedAmountCents != null) ...[
                FeePreviewWidget(
                  amountCents: _debouncedAmountCents,
                  recipientId: widget.recipientId.isNotEmpty
                      ? widget.recipientId
                      : null,
                ),
                const SizedBox(height: 16),
              ],

              // Deadline (optional)
              _buildDeadlineField(theme, appColors, l10n),
              const SizedBox(height: 32),

              // Send / Modify button
              ElevatedButton(
                onPressed: _isSubmitting ? null : _onSend,
                style: ElevatedButton.styleFrom(
                  backgroundColor: theme.colorScheme.primary,
                  foregroundColor: Colors.white,
                  minimumSize: const Size(double.infinity, 48),
                  shape: RoundedRectangleBorder(
                    borderRadius:
                        BorderRadius.circular(AppTheme.radiusMd),
                  ),
                ),
                child: _isSubmitting
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : Text(sendLabel),
              ),
              const SizedBox(height: 8),

              // Cancel button
              TextButton(
                onPressed: () => GoRouter.of(context).pop(),
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
