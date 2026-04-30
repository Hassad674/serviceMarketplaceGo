import 'package:flutter/material.dart';

/// Footer for the portfolio form: cancel + save buttons inside a SafeArea bar.
class PortfolioFormFooter extends StatelessWidget {
  const PortfolioFormFooter({
    super.key,
    required this.isEdit,
    required this.saving,
    required this.canSave,
    required this.onCancel,
    required this.onSave,
  });

  final bool isEdit;
  final bool saving;
  final bool canSave;
  final VoidCallback onCancel;
  final VoidCallback onSave;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SafeArea(
      top: false,
      child: Container(
        padding: const EdgeInsets.fromLTRB(20, 12, 20, 12),
        decoration: BoxDecoration(
          border: Border(
            top: BorderSide(color: theme.dividerColor),
          ),
        ),
        child: Row(
          children: [
            Expanded(
              child: TextButton(
                onPressed: saving ? null : onCancel,
                child: const Text('Cancel'),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              flex: 2,
              child: FilledButton(
                onPressed: (saving || !canSave) ? null : onSave,
                style: FilledButton.styleFrom(
                  backgroundColor: const Color(0xFFE11D48),
                  foregroundColor: Colors.white,
                  padding: const EdgeInsets.symmetric(vertical: 12),
                ),
                child: saving
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : Text(isEdit ? 'Save changes' : 'Create'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
