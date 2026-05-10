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
import '../widgets/milestone_editor_widget.dart';
import '../widgets/payment_mode_toggle_widget.dart';

/// Soleil v2 — Proposal creation form (M-09 mobile equivalent for proposals).
///
/// Editorial header (corail mono eyebrow + Fraunces italic-corail title +
/// tabac subtitle), Soleil card sections, Fraunces section heads, ivoire
/// inputs with corail focus, corail StadiumBorder pill submit.
///
/// Phase 6 (Contra-style milestones): when payment mode is [milestone],
/// the global title/description/amount/deadline inputs are HIDDEN. The
/// proposal-level fields are derived from the milestone slice at submit
/// time. Min [kMinMilestonesPerMilestoneProposal] entries — toggling
/// into milestone mode tops the slice up to 2 empty rows automatically.
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

  int? _debouncedAmountCents;
  Timer? _amountDebounce;

  bool get _isModifyMode => widget.existingProposal != null;
  bool get _isMilestoneMode =>
      _formData.paymentMode == ProposalPaymentMode.milestone;

  @override
  void initState() {
    super.initState();
    _formData = ProposalFormData(
      recipientId: widget.recipientId,
      conversationId: widget.conversationId,
      recipientName: widget.recipientName,
    );

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

  Future<void> _onDeadlinePicked() async {
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

  void _onPaymentModeChanged(ProposalPaymentMode mode) {
    setState(() {
      _formData.paymentMode = mode;
      if (mode == ProposalPaymentMode.milestone) {
        // Top up to the minimum so the editor never starts below 2
        // empty rows (mirrors the web `ensureMinimumMilestones`).
        while (_formData.milestones.length <
            kMinMilestonesPerMilestoneProposal) {
          _formData.milestones.add(MilestoneFormItem());
        }
      }
    });
  }

  bool _isValidMilestoneMode() {
    if (_formData.milestones.length < kMinMilestonesPerMilestoneProposal) {
      return false;
    }
    for (final m in _formData.milestones) {
      if (m.title.trim().isEmpty) return false;
      if (m.amount <= 0) return false;
    }
    return true;
  }

  /// Synthesises a proposal-level title from the milestones (Contra-
  /// style: the global title input is hidden in milestone mode).
  String _deriveMilestoneTitle(String fallback) {
    for (final m in _formData.milestones) {
      final trimmed = m.title.trim();
      if (trimmed.isNotEmpty) return trimmed;
    }
    return fallback;
  }

  /// Concatenates milestone titles into a proposal-level description.
  String _deriveMilestoneDescription() {
    final titles = _formData.milestones
        .map((m) => m.title.trim())
        .where((t) => t.isNotEmpty)
        .toList();
    if (titles.isEmpty) return '';
    final lines = <String>[];
    for (var i = 0; i < titles.length; i++) {
      lines.add('${i + 1}. ${titles[i]}');
    }
    return lines.join('\n');
  }

  /// Returns the latest milestone deadline as ISO string, or null if no
  /// milestone has a deadline.
  String? _latestMilestoneDeadlineIso() {
    DateTime? latest;
    for (final m in _formData.milestones) {
      if (m.deadline == null) continue;
      if (latest == null || m.deadline!.isAfter(latest)) {
        latest = m.deadline;
      }
    }
    return latest?.toUtc().toIso8601String();
  }

  /// Builds the milestone payload (sequence-indexed) for the API.
  List<MilestoneInputData> _buildMilestonePayload() {
    return [
      for (var i = 0; i < _formData.milestones.length; i++)
        MilestoneInputData(
          sequence: i + 1,
          title: _formData.milestones[i].title.trim(),
          description: _formData.milestones[i].description.trim(),
          amount: (_formData.milestones[i].amount * 100).round(),
          deadline: _formData.milestones[i].deadline != null
              ? _formatYmd(_formData.milestones[i].deadline!)
              : null,
        ),
    ];
  }

  Future<void> _onSend() async {
    final l10n = AppLocalizations.of(context)!;
    if (_isMilestoneMode) {
      if (!_isValidMilestoneMode()) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              l10n.milestoneEditorMinimumHint(
                kMinMilestonesPerMilestoneProposal,
              ),
            ),
          ),
        );
        return;
      }
    } else {
      _formData.title = _titleController.text.trim();
      _formData.description = _descriptionController.text.trim();
      _formData.amount = double.tryParse(_amountController.text) ?? 0;
      if (!_formKey.currentState!.validate()) return;
    }

    setState(() => _isSubmitting = true);

    try {
      final repo = ref.read(proposalRepositoryProvider);

      if (_isModifyMode) {
        final ModifyProposalData payload;
        if (_isMilestoneMode) {
          final milestonesPayload = _buildMilestonePayload();
          final totalCents =
              milestonesPayload.fold<int>(0, (sum, m) => sum + m.amount);
          payload = ModifyProposalData(
            title: _deriveMilestoneTitle(l10n.proposalTitleFallback),
            description: _deriveMilestoneDescription(),
            amount: totalCents,
            deadline: _latestMilestoneDeadlineIso(),
            paymentMode: 'milestone',
            milestones: milestonesPayload,
          );
        } else {
          payload = ModifyProposalData(
            title: _formData.title,
            description: _formData.description,
            amount: (_formData.amount * 100).round(),
            deadline: _formData.deadline?.toUtc().toIso8601String(),
            paymentMode: 'one_time',
          );
        }
        final modified =
            await repo.modifyProposal(widget.existingProposal!.id, payload);
        if (mounted) GoRouter.of(context).pop(modified);
      } else {
        final CreateProposalData payload;
        if (_isMilestoneMode) {
          final milestonesPayload = _buildMilestonePayload();
          final totalCents =
              milestonesPayload.fold<int>(0, (sum, m) => sum + m.amount);
          payload = CreateProposalData(
            recipientId: _formData.recipientId,
            conversationId: _formData.conversationId,
            title: _deriveMilestoneTitle(l10n.proposalTitleFallback),
            description: _deriveMilestoneDescription(),
            amount: totalCents,
            deadline: _latestMilestoneDeadlineIso(),
            paymentMode: 'milestone',
            milestones: milestonesPayload,
          );
        } else {
          payload = CreateProposalData(
            recipientId: _formData.recipientId,
            conversationId: _formData.conversationId,
            title: _formData.title,
            description: _formData.description,
            amount: (_formData.amount * 100).round(),
            deadline: _formData.deadline?.toUtc().toIso8601String(),
            paymentMode: 'one_time',
          );
        }
        final created = await repo.createProposal(payload);
        if (mounted) GoRouter.of(context).pop(created);
      }
    } catch (e) {
      if (mounted) {
        setState(() => _isSubmitting = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${l10n.unexpectedError}: $e')),
        );
      }
    }
  }

  String _formatYmd(DateTime d) {
    final dd = d.day.toString().padLeft(2, '0');
    final mm = d.month.toString().padLeft(2, '0');
    return '${d.year}-$mm-$dd';
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
    final sendLabel =
        _isModifyMode ? l10n.proposalModify : l10n.proposalSend;

    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        leading: IconButton(
          icon: const Icon(Icons.close_rounded),
          onPressed: () => GoRouter.of(context).pop(),
          color: theme.colorScheme.onSurface,
        ),
        title: Text(
          l10n.proposalCreate,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        actions: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            child: FilledButton(
              onPressed: _isSubmitting ? null : _onSend,
              style: FilledButton.styleFrom(
                shape: const StadiumBorder(),
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                textStyle: SoleilTextStyles.button,
              ),
              child: _isSubmitting
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        valueColor:
                            AlwaysStoppedAnimation<Color>(Colors.white),
                      ),
                    )
                  : Text(sendLabel),
            ),
          ),
        ],
      ),
      body: SafeArea(
        child: Form(
          key: _formKey,
          child: ListView(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
            children: [
              _Header(isModify: _isModifyMode),
              const SizedBox(height: 24),
              _BriefSection(
                titleController: _titleController,
                descriptionController: _descriptionController,
                recipientName: widget.recipientName.isNotEmpty
                    ? widget.recipientName
                    : 'User ${widget.recipientId.length > 8 ? widget.recipientId.substring(0, 8) : widget.recipientId}',
                amountController: _amountController,
                debouncedAmountCents: _debouncedAmountCents,
                recipientId: widget.recipientId,
                paymentMode: _formData.paymentMode,
                onPaymentModeChanged: _onPaymentModeChanged,
                isMilestoneMode: _isMilestoneMode,
              ),
              if (_isMilestoneMode) ...[
                const SizedBox(height: 16),
                _SoleilCard(
                  eyebrow: l10n.proposalSectionPayment,
                  child: MilestoneEditorWidget(
                    milestones: _formData.milestones,
                    onChanged: (next) =>
                        setState(() => _formData.milestones = next),
                    disabled: _isSubmitting,
                  ),
                ),
              ] else ...[
                const SizedBox(height: 16),
                _DeadlineSection(
                  deadline: _formData.deadline,
                  onPick: _onDeadlinePicked,
                  formatter: _formatDate,
                ),
              ],
              const SizedBox(height: 24),
              FilledButton(
                onPressed: _isSubmitting ? null : _onSend,
                style: FilledButton.styleFrom(
                  minimumSize: const Size.fromHeight(52),
                  shape: const StadiumBorder(),
                  textStyle: SoleilTextStyles.button,
                ),
                child: _isSubmitting
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor:
                              AlwaysStoppedAnimation<Color>(Colors.white),
                        ),
                      )
                    : Text(sendLabel),
              ),
              const SizedBox(height: 8),
              TextButton(
                onPressed: () => GoRouter.of(context).pop(),
                style: TextButton.styleFrom(
                  shape: const StadiumBorder(),
                  minimumSize: const Size.fromHeight(48),
                ),
                child: Text(l10n.cancel),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.isModify});

  final bool isModify;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final primary = theme.colorScheme.primary;

    final titleAccent = isModify
        ? l10n.proposalFlow_create_modifyTitleAccent
        : l10n.proposalFlow_create_titleAccent;
    final subtitle = isModify
        ? l10n.proposalFlow_create_modifySubtitle
        : l10n.proposalFlow_create_subtitle;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.proposalFlow_create_eyebrow,
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
            style: SoleilTextStyles.headlineLarge.copyWith(
              color: theme.colorScheme.onSurface,
            ),
            children: [
              TextSpan(text: '${l10n.proposalFlow_create_titlePrefix} '),
              TextSpan(
                text: titleAccent,
                style: SoleilTextStyles.headlineLarge.copyWith(
                  color: primary,
                  fontStyle: FontStyle.italic,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        Text(
          subtitle,
          style: SoleilTextStyles.bodyLarge.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

class _BriefSection extends StatelessWidget {
  const _BriefSection({
    required this.titleController,
    required this.descriptionController,
    required this.amountController,
    required this.recipientName,
    required this.debouncedAmountCents,
    required this.recipientId,
    required this.paymentMode,
    required this.onPaymentModeChanged,
    required this.isMilestoneMode,
  });

  final TextEditingController titleController;
  final TextEditingController descriptionController;
  final TextEditingController amountController;
  final String recipientName;
  final int? debouncedAmountCents;
  final String recipientId;
  final ProposalPaymentMode paymentMode;
  final ValueChanged<ProposalPaymentMode> onPaymentModeChanged;
  final bool isMilestoneMode;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();

    return _SoleilCard(
      eyebrow: l10n.proposalFlow_create_sectionBrief,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              CircleAvatar(
                radius: 18,
                backgroundColor: theme.colorScheme.primaryContainer,
                child: Icon(
                  Icons.person_rounded,
                  size: 18,
                  color: theme.colorScheme.primary,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.proposalRecipient,
                      style: SoleilTextStyles.mono.copyWith(
                        color: appColors?.subtleForeground ??
                            theme.colorScheme.onSurfaceVariant,
                        fontSize: 10.5,
                        fontWeight: FontWeight.w700,
                        letterSpacing: 0.8,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      recipientName,
                      style: SoleilTextStyles.bodyEmphasis.copyWith(
                        color: theme.colorScheme.onSurface,
                      ),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 18),
          PaymentModeToggleWidget(
            value: paymentMode,
            onChanged: onPaymentModeChanged,
          ),
          // Phase 6: in milestone mode the global title/description/amount
          // inputs are HIDDEN — the proposal-level fields derive from the
          // milestone slice at submit time.
          if (!isMilestoneMode) ...[
            const SizedBox(height: 18),
            TextFormField(
              controller: titleController,
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
            const SizedBox(height: 12),
            TextFormField(
              controller: descriptionController,
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
            const SizedBox(height: 12),
            TextFormField(
              controller: amountController,
              decoration: InputDecoration(
                labelText: l10n.proposalAmount,
                hintText: l10n.proposalAmountHint,
                prefixText: '€ ',
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
            if (debouncedAmountCents != null) ...[
              const SizedBox(height: 12),
              FeePreviewWidget(
                amountCents: debouncedAmountCents,
                recipientId: recipientId.isNotEmpty ? recipientId : null,
              ),
            ],
          ],
        ],
      ),
    );
  }
}

class _DeadlineSection extends StatelessWidget {
  const _DeadlineSection({
    required this.deadline,
    required this.onPick,
    required this.formatter,
  });

  final DateTime? deadline;
  final VoidCallback onPick;
  final String Function(DateTime) formatter;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return _SoleilCard(
      eyebrow: l10n.proposalFlow_create_sectionDeadline,
      child: GestureDetector(
        onTap: onPick,
        child: AbsorbPointer(
          child: TextFormField(
            decoration: InputDecoration(
              labelText: l10n.proposalDeadline,
              suffixIcon: Icon(
                Icons.calendar_today_rounded,
                size: 18,
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            controller: TextEditingController(
              text: deadline != null ? formatter(deadline!) : '',
            ),
          ),
        ),
      ),
    );
  }
}

class _SoleilCard extends StatelessWidget {
  const _SoleilCard({required this.eyebrow, required this.child});

  final String eyebrow;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            eyebrow,
            style: SoleilTextStyles.mono.copyWith(
              color: primary,
              fontSize: 10.5,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.2,
            ),
          ),
          const SizedBox(height: 14),
          child,
        ],
      ),
    );
  }
}
