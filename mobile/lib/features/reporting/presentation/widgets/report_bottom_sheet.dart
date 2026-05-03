import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../data/report_repository_impl.dart';
import '../../../../core/theme/app_palette.dart';

/// Reason options for reporting a message.
const _messageReasons = [
  'harassment',
  'fraud',
  'off_platform_payment',
  'spam',
  'inappropriate_content',
  'other',
];

/// Reason options for reporting a user.
const _userReasons = [
  'harassment',
  'fraud',
  'off_platform_payment',
  'spam',
  'fake_profile',
  'unprofessional_behavior',
  'other',
];

/// Reason options for reporting a job.
const _jobReasons = [
  'fraud_or_scam',
  'misleading_description',
  'inappropriate_content',
  'spam',
  'other',
];

/// Reason options for reporting an application.
const _applicationReasons = [
  'fraud_or_scam',
  'spam',
  'inappropriate_content',
  'other',
];

/// Shows the report bottom sheet as a modal.
Future<void> showReportBottomSheet(
  BuildContext context,
  WidgetRef ref, {
  required String targetType,
  required String targetId,
  required String conversationId,
}) async {
  await showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => _ReportBottomSheetContent(
      targetType: targetType,
      targetId: targetId,
      conversationId: conversationId,
      ref: ref,
    ),
  );
}

class _ReportBottomSheetContent extends StatefulWidget {
  const _ReportBottomSheetContent({
    required this.targetType,
    required this.targetId,
    required this.conversationId,
    required this.ref,
  });

  final String targetType;
  final String targetId;
  final String conversationId;
  final WidgetRef ref;

  @override
  State<_ReportBottomSheetContent> createState() =>
      _ReportBottomSheetContentState();
}

class _ReportBottomSheetContentState
    extends State<_ReportBottomSheetContent> {
  final _descriptionController = TextEditingController();
  String? _selectedReason;
  bool _isSubmitting = false;

  List<String> get _reasons => switch (widget.targetType) {
        'user' => _userReasons,
        'job' => _jobReasons,
        'application' => _applicationReasons,
        _ => _messageReasons,
      };

  @override
  void dispose() {
    _descriptionController.dispose();
    super.dispose();
  }

  String _reasonLabel(AppLocalizations l10n, String reason) {
    return switch (reason) {
      'harassment' => l10n.reasonHarassment,
      'fraud' => l10n.reasonFraud,
      'fraud_or_scam' => l10n.reasonFraudOrScam,
      'off_platform_payment' => l10n.reasonOffPlatformPayment,
      'spam' => l10n.reasonSpam,
      'inappropriate_content' => l10n.reasonInappropriateContent,
      'fake_profile' => l10n.reasonFakeProfile,
      'unprofessional_behavior' => l10n.reasonUnprofessionalBehavior,
      'misleading_description' => l10n.reasonMisleadingDescription,
      'other' => l10n.reasonOther,
      _ => reason,
    };
  }

  Future<void> _submit() async {
    if (_selectedReason == null) return;
    final description = _descriptionController.text.trim();
    if (description.isEmpty) return;

    setState(() => _isSubmitting = true);

    try {
      final repo = widget.ref.read(reportRepositoryProvider);
      await repo.createReport(
        targetType: widget.targetType,
        targetId: widget.targetId,
        conversationId: widget.conversationId,
        reason: _selectedReason!,
        description: description,
      );

      if (!mounted) return;
      Navigator.pop(context);
      final l10n = AppLocalizations.of(context)!;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.reportSent)),
      );
    } catch (_) {
      if (!mounted) return;
      setState(() => _isSubmitting = false);
      final l10n = AppLocalizations.of(context)!;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.reportError)),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final title = switch (widget.targetType) {
      'user' => l10n.reportUser,
      'job' => l10n.reportJob,
      'application' => l10n.reportApplication,
      _ => l10n.reportMessage,
    };
    final canSubmit = _selectedReason != null &&
        _descriptionController.text.trim().isNotEmpty &&
        !_isSubmitting;

    return Padding(
      padding: EdgeInsets.only(
        left: 24,
        right: 24,
        top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 24,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Drag handle
          Center(
            child: Container(
              width: 40,
              height: 4,
              decoration: BoxDecoration(
                color: theme.dividerColor,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ),
          const SizedBox(height: 16),

          // Title
          Text(title, style: theme.textTheme.titleLarge),
          const SizedBox(height: 16),

          // Reason selection label
          Text(
            l10n.selectReason,
            style: theme.textTheme.titleSmall,
          ),
          const SizedBox(height: 8),

          // Reason chips
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: _reasons.map((reason) {
              final selected = _selectedReason == reason;
              return ChoiceChip(
                label: Text(_reasonLabel(l10n, reason)),
                selected: selected,
                onSelected: _isSubmitting
                    ? null
                    : (val) {
                        setState(() {
                          _selectedReason = val ? reason : null;
                        });
                      },
                selectedColor:
                    AppPalette.rose500.withValues(alpha: 0.15),
                labelStyle: TextStyle(
                  fontSize: 13,
                  color: selected
                      ? AppPalette.rose500
                      : theme.colorScheme.onSurface,
                ),
              );
            }).toList(),
          ),
          const SizedBox(height: 16),

          // Description field
          Text(
            l10n.reportDescription,
            style: theme.textTheme.titleSmall,
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _descriptionController,
            maxLines: 4,
            maxLength: 2000,
            enabled: !_isSubmitting,
            decoration: InputDecoration(
              hintText: l10n.reportDescriptionHint,
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
              ),
            ),
            onChanged: (_) => setState(() {}),
          ),
          const SizedBox(height: 16),

          // Submit button
          SizedBox(
            width: double.infinity,
            height: 48,
            child: ElevatedButton(
              onPressed: canSubmit ? _submit : null,
              style: ElevatedButton.styleFrom(
                backgroundColor: AppPalette.rose500,
                foregroundColor: Colors.white,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
              child: _isSubmitting
                  ? const SizedBox(
                      width: 22,
                      height: 22,
                      child: CircularProgressIndicator(
                        strokeWidth: 2.5,
                        color: Colors.white,
                      ),
                    )
                  : Text(l10n.submitReport),
            ),
          ),

          // Cancel
          const SizedBox(height: 8),
          SizedBox(
            width: double.infinity,
            height: 48,
            child: TextButton(
              onPressed:
                  _isSubmitting ? null : () => Navigator.of(context).pop(),
              child: Text(l10n.cancel),
            ),
          ),
        ],
      ),
    );
  }
}
