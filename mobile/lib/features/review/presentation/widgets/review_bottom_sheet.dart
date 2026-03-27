import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers/review_provider.dart';

/// Bottom sheet for leaving a review after a completed mission.
class ReviewBottomSheet extends ConsumerStatefulWidget {
  final String proposalId;
  final String proposalTitle;
  final VoidCallback? onSubmitted;

  const ReviewBottomSheet({
    super.key,
    required this.proposalId,
    required this.proposalTitle,
    this.onSubmitted,
  });

  static Future<void> show(
    BuildContext context, {
    required String proposalId,
    required String proposalTitle,
    VoidCallback? onSubmitted,
  }) {
    return showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (_) => ReviewBottomSheet(
        proposalId: proposalId,
        proposalTitle: proposalTitle,
        onSubmitted: onSubmitted,
      ),
    );
  }

  @override
  ConsumerState<ReviewBottomSheet> createState() => _ReviewBottomSheetState();
}

class _ReviewBottomSheetState extends ConsumerState<ReviewBottomSheet> {
  int _globalRating = 0;
  int _timeliness = 0;
  int _communication = 0;
  int _quality = 0;
  final _commentController = TextEditingController();
  bool _isSubmitting = false;

  @override
  void dispose() {
    _commentController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_globalRating == 0) return;
    setState(() => _isSubmitting = true);

    try {
      final repo = ref.read(reviewRepositoryProvider);
      await repo.createReview(
        proposalId: widget.proposalId,
        globalRating: _globalRating,
        timeliness: _timeliness > 0 ? _timeliness : null,
        communication: _communication > 0 ? _communication : null,
        quality: _quality > 0 ? _quality : null,
        comment: _commentController.text.trim(),
      );
      if (mounted) {
        Navigator.of(context).pop();
        widget.onSubmitted?.call();
      }
    } catch (_) {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
        left: 20,
        right: 20,
        top: 20,
      ),
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildHeader(theme),
            const SizedBox(height: 20),
            _buildStarRow('Overall rating *', _globalRating, (v) {
              setState(() => _globalRating = v);
            }),
            const Divider(height: 32),
            Text(
              'Detailed criteria (optional)',
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            _buildStarRow('Timeliness', _timeliness, (v) {
              setState(() => _timeliness = v);
            }),
            const SizedBox(height: 8),
            _buildStarRow('Communication', _communication, (v) {
              setState(() => _communication = v);
            }),
            const SizedBox(height: 8),
            _buildStarRow('Quality', _quality, (v) {
              setState(() => _quality = v);
            }),
            const SizedBox(height: 20),
            TextField(
              controller: _commentController,
              maxLines: 3,
              maxLength: 2000,
              decoration: const InputDecoration(
                labelText: 'Written review',
                hintText: 'Describe your experience...',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            _buildActions(theme),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }

  Widget _buildHeader(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
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
        Text('Leave a review', style: theme.textTheme.titleLarge),
        const SizedBox(height: 4),
        Text(widget.proposalTitle, style: theme.textTheme.bodySmall),
      ],
    );
  }

  Widget _buildStarRow(String label, int value, ValueChanged<int> onChanged) {
    return Row(
      children: [
        Expanded(
          child: Text(label, style: Theme.of(context).textTheme.bodyMedium),
        ),
        for (int i = 1; i <= 5; i++)
          GestureDetector(
            onTap: () => onChanged(i),
            child: Icon(
              i <= value ? Icons.star : Icons.star_border,
              color: const Color(0xFFFBBF24),
              size: 28,
            ),
          ),
      ],
    );
  }

  Widget _buildActions(ThemeData theme) {
    return Row(
      children: [
        Expanded(
          child: FilledButton(
            onPressed: (_globalRating == 0 || _isSubmitting) ? null : _submit,
            child: _isSubmitting
                ? const SizedBox(
                    height: 20,
                    width: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Text('Submit review'),
          ),
        ),
        const SizedBox(width: 12),
        OutlinedButton(
          onPressed: _isSubmitting ? null : () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
      ],
    );
  }
}
