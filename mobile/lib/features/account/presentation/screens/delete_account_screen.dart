import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../data/gdpr_repository_impl.dart';
import '../../domain/entities/deletion_status.dart';

/// DeleteAccountScreen is the mobile counterpart of the web flow at
/// /account?section=data-and-deletion. It exposes the password
/// re-prompt + confirm checkbox + 409 owner-block panel + success
/// confirmation.
///
/// Once submitted successfully, the user is told to check their email
/// for the confirmation link. Clicking the link in the email opens
/// the web confirm-deletion page (universal links land directly in
/// the Flutter app or fall back to the browser depending on platform
/// configuration; this is out of P5 scope).
class DeleteAccountScreen extends ConsumerStatefulWidget {
  const DeleteAccountScreen({super.key});

  @override
  ConsumerState<DeleteAccountScreen> createState() =>
      _DeleteAccountScreenState();
}

class _DeleteAccountScreenState extends ConsumerState<DeleteAccountScreen> {
  final _passwordCtrl = TextEditingController();
  bool _confirmed = false;
  bool _submitting = false;
  String? _error;
  RequestDeletionResult? _success;
  List<BlockedOrg>? _blocked;

  @override
  void dispose() {
    _passwordCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_confirmed || _passwordCtrl.text.isEmpty) return;
    final l10n = AppLocalizations.of(context)!;
    setState(() {
      _submitting = true;
      _error = null;
      _blocked = null;
    });

    try {
      final repo = ref.read(gdprRepositoryProvider);
      final res = await repo.requestDeletion(_passwordCtrl.text);
      if (!mounted) return;
      setState(() {
        _success = res;
      });
    } on OwnerBlockedException catch (e) {
      if (!mounted) return;
      setState(() {
        _blocked = e.blockedOrgs;
      });
    } catch (err) {
      if (!mounted) return;
      setState(() {
        _error = l10n.gdprDeleteGenericError;
      });
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.gdprDeleteTitle)),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: _success != null
              ? _SuccessPanel(emailSentTo: _success!.emailSentTo)
              : _blocked != null
                  ? _BlockedPanel(orgs: _blocked!)
                  : _Form(
                      passwordCtrl: _passwordCtrl,
                      confirmed: _confirmed,
                      submitting: _submitting,
                      error: _error,
                      onConfirmedChanged: (v) =>
                          setState(() => _confirmed = v ?? false),
                      onSubmit: _submit,
                    ),
        ),
      ),
    );
  }
}

class _Form extends StatelessWidget {
  final TextEditingController passwordCtrl;
  final bool confirmed;
  final bool submitting;
  final String? error;
  final ValueChanged<bool?> onConfirmedChanged;
  final VoidCallback onSubmit;

  const _Form({
    required this.passwordCtrl,
    required this.confirmed,
    required this.submitting,
    required this.error,
    required this.onConfirmedChanged,
    required this.onSubmit,
  });

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(l10n.gdprDeleteIntro),
        const SizedBox(height: 12),
        Text(l10n.gdprDeleteBullet1),
        const SizedBox(height: 4),
        Text(l10n.gdprDeleteBullet2),
        const SizedBox(height: 4),
        Text(l10n.gdprDeleteBullet3),
        const SizedBox(height: 16),
        TextField(
          controller: passwordCtrl,
          obscureText: true,
          decoration: InputDecoration(
            labelText: l10n.gdprDeletePasswordLabel,
            border: const OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 12),
        CheckboxListTile(
          title: Text(l10n.gdprDeleteConfirmCheckbox),
          value: confirmed,
          onChanged: onConfirmedChanged,
          controlAffinity: ListTileControlAffinity.leading,
        ),
        if (error != null) ...[
          const SizedBox(height: 8),
          Text(error!, style: const TextStyle(color: Colors.red)),
        ],
        const SizedBox(height: 16),
        FilledButton(
          style: FilledButton.styleFrom(
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
          onPressed: confirmed && !submitting ? onSubmit : null,
          child: submitting
              ? const SizedBox(
                  height: 16,
                  width: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Text(l10n.gdprDeleteSubmit),
        ),
      ],
    );
  }
}

class _SuccessPanel extends StatelessWidget {
  final String emailSentTo;
  const _SuccessPanel({required this.emailSentTo});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.gdprDeleteSuccessTitle,
          style: Theme.of(context).textTheme.titleLarge,
        ),
        const SizedBox(height: 8),
        Text(l10n.gdprDeleteSuccessIntro),
        const SizedBox(height: 8),
        Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Colors.pink.shade50,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Text(
            emailSentTo,
            style: const TextStyle(fontWeight: FontWeight.bold),
          ),
        ),
        const SizedBox(height: 12),
        Text(l10n.gdprDeleteSuccessTtl),
      ],
    );
  }
}

class _BlockedPanel extends StatelessWidget {
  final List<BlockedOrg> orgs;
  const _BlockedPanel({required this.orgs});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.gdprDeleteBlockedTitle,
          style: Theme.of(context).textTheme.titleLarge,
        ),
        const SizedBox(height: 8),
        Text(l10n.gdprDeleteBlockedIntro),
        const SizedBox(height: 12),
        for (final org in orgs)
          Card(
            color: Colors.amber.shade50,
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    org.orgName,
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                  Text(
                    l10n.gdprDeleteBlockedMemberCount(org.memberCount),
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}
