import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../data/gdpr_repository_impl.dart';

/// CancelDeletionScreen lets a soft-deleted user roll back the
/// 30-day cooldown without leaving the app. Shows a single CTA + a
/// success state. Errors surface inline.
class CancelDeletionScreen extends ConsumerStatefulWidget {
  const CancelDeletionScreen({super.key});

  @override
  ConsumerState<CancelDeletionScreen> createState() =>
      _CancelDeletionScreenState();
}

class _CancelDeletionScreenState extends ConsumerState<CancelDeletionScreen> {
  bool _submitting = false;
  bool _done = false;
  String? _error;

  Future<void> _cancel() async {
    final l10n = AppLocalizations.of(context)!;
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final repo = ref.read(gdprRepositoryProvider);
      await repo.cancelDeletion();
      if (!mounted) return;
      setState(() => _done = true);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = l10n.gdprCancelGenericError);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.gdprCancelTitle)),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (_done) ...[
                Text(
                  l10n.gdprCancelDoneTitle,
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 8),
                Text(l10n.gdprCancelDoneBody),
              ] else ...[
                Text(l10n.gdprCancelBody),
                const SizedBox(height: 16),
                if (_error != null) ...[
                  Text(_error!, style: const TextStyle(color: Colors.red)),
                  const SizedBox(height: 12),
                ],
                FilledButton(
                  onPressed: _submitting ? null : _cancel,
                  child: _submitting
                      ? const SizedBox(
                          height: 16,
                          width: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Text(l10n.gdprCancelButton),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
