import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../../core/network/api_client.dart';
import '../../../../l10n/app_localizations.dart';
import '../../data/dispute_uploader.dart';
import '../providers/dispute_provider.dart';

/// Form screen to send a counter-proposal on a dispute.
class CounterProposeScreen extends ConsumerStatefulWidget {
  const CounterProposeScreen({
    super.key,
    required this.disputeId,
    required this.proposalAmount,
  });

  final String disputeId;
  final int proposalAmount;

  @override
  ConsumerState<CounterProposeScreen> createState() =>
      _CounterProposeScreenState();
}

class _CounterProposeScreenState extends ConsumerState<CounterProposeScreen> {
  final _messageController = TextEditingController();
  int _clientAmount = 0;
  List<File> _files = [];
  bool _isSubmitting = false;

  @override
  void dispose() {
    _messageController.dispose();
    super.dispose();
  }

  Future<void> _onAddFiles() async {
    final result = await FilePicker.platform.pickFiles(allowMultiple: true);
    if (result == null) return;
    setState(() {
      _files.addAll(
        result.files.where((f) => f.path != null).map((f) => File(f.path!)),
      );
    });
  }

  void _removeFile(int index) {
    setState(() => _files.removeAt(index));
  }

  Future<void> _onSubmit() async {
    setState(() => _isSubmitting = true);

    try {
      final apiClient = ref.read(apiClientProvider);
      final attachments = _files.isEmpty
          ? <Map<String, dynamic>>[]
          : await uploadDisputeFiles(apiClient, _files);

      final providerAmount = widget.proposalAmount - _clientAmount;
      final ok = await counterPropose(
        ref,
        disputeId: widget.disputeId,
        amountClient: _clientAmount,
        amountProvider: providerAmount,
        message: _messageController.text.trim(),
        attachments: attachments,
      );

      if (!mounted) return;

      if (!ok) {
        setState(() => _isSubmitting = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(AppLocalizations.of(context)!.unexpectedError)),
        );
        return;
      }

      GoRouter.of(context).pop(true);
    } catch (e) {
      if (mounted) {
        setState(() => _isSubmitting = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              '${AppLocalizations.of(context)!.unexpectedError}: $e',
            ),
          ),
        );
      }
    }
  }

  String _formatEur(int centimes) {
    return NumberFormat.currency(locale: 'fr_FR', symbol: '€', decimalDigits: 2)
        .format(centimes / 100);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final providerAmount = widget.proposalAmount - _clientAmount;

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.disputeCounterPropose),
        actions: [
          TextButton(
            onPressed: _isSubmitting ? null : _onSubmit,
            child: Text(
              l10n.disputeCounterSubmit,
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
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Text(
              l10n.disputeCounterSplitLabel,
              style: theme.textTheme.bodyMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            Slider(
              min: 0,
              max: widget.proposalAmount.toDouble(),
              divisions: 100,
              value: _clientAmount.toDouble(),
              onChanged: (v) => setState(() => _clientAmount = v.round()),
            ),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text('${l10n.disputeClient}: ${_formatEur(_clientAmount)}'),
                Text('${l10n.disputeProvider}: ${_formatEur(providerAmount)}'),
              ],
            ),
            const SizedBox(height: 24),
            TextFormField(
              controller: _messageController,
              decoration: InputDecoration(
                labelText: l10n.disputeCounterMessageLabel,
                hintText: l10n.disputeCounterMessagePlaceholder,
                alignLabelWithHint: true,
              ),
              maxLines: 4,
              maxLength: 2000,
            ),
            const SizedBox(height: 8),
            _AttachmentsRow(
              files: _files,
              onAdd: _onAddFiles,
              onRemove: _removeFile,
              addLabel: l10n.disputeAddFiles,
            ),
            const SizedBox(height: 24),
            if (_isSubmitting)
              Center(
                child: CircularProgressIndicator(
                  color: theme.colorScheme.primary,
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _AttachmentsRow extends StatelessWidget {
  const _AttachmentsRow({
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
