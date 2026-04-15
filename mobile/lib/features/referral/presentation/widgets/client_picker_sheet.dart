import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../messaging/data/messaging_repository_impl.dart';
import '../../../messaging/domain/entities/conversation_entity.dart';
import 'picker_selection.dart';

/// showClientPickerSheet opens a modal bottom sheet that lists the user's
/// existing conversations filtered to enterprises — the only legitimate
/// way to pick a client for a business referral.
///
/// Cold-introducing a stranger is not supported by design (see
/// feedback_b2b_confidentiality.md and the referral plan).
Future<ClientPickerSelection?> showClientPickerSheet(
  BuildContext context,
) async {
  return showModalBottomSheet<ClientPickerSelection?>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
    ),
    builder: (ctx) => const _ClientPickerSheet(),
  );
}

class _ClientPickerSheet extends ConsumerStatefulWidget {
  const _ClientPickerSheet();

  @override
  ConsumerState<_ClientPickerSheet> createState() => _ClientPickerSheetState();
}

class _ClientPickerSheetState extends ConsumerState<_ClientPickerSheet> {
  bool _loading = true;
  String? _error;
  List<ConversationEntity> _conversations = const [];

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final repo = ref.read(messagingRepositoryProvider);
      final response = await repo.getConversations(limit: 50);
      if (!mounted) return;
      setState(() {
        _conversations = response.data;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = 'Impossible de charger vos conversations.';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final enterprises = _conversations.where((c) => c.otherOrgType == 'enterprise').toList();

    return SizedBox(
      height: MediaQuery.of(context).size.height * 0.75,
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 8, 8),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Choisir un client',
                        style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
                      ),
                      Text(
                        'Uniquement les entreprises avec qui vous avez déjà une conversation.',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.close),
                  onPressed: () => Navigator.of(context).pop(),
                  tooltip: 'Fermer',
                ),
              ],
            ),
          ),
          Divider(height: 1, color: theme.colorScheme.outlineVariant),
          if (_loading)
            const Expanded(child: Center(child: CircularProgressIndicator()))
          else if (_error != null)
            Expanded(
              child: Center(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text(_error!, textAlign: TextAlign.center),
                ),
              ),
            )
          else if (enterprises.isEmpty)
            Expanded(
              child: Center(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text(
                    'Aucune conversation avec un client.\nCommencez par échanger avec un prospect avant de le présenter.',
                    textAlign: TextAlign.center,
                    style: theme.textTheme.bodySmall,
                  ),
                ),
              ),
            )
          else
            Expanded(
              child: ListView.separated(
                itemCount: enterprises.length,
                separatorBuilder: (_, __) => Divider(height: 1, color: theme.colorScheme.outlineVariant),
                itemBuilder: (ctx, i) {
                  final c = enterprises[i];
                  final initial = c.otherOrgName.isNotEmpty ? c.otherOrgName.substring(0, 1).toUpperCase() : '?';
                  return ListTile(
                    leading: CircleAvatar(
                      radius: 18,
                      backgroundColor: theme.colorScheme.secondaryContainer,
                      child: Text(
                        initial,
                        style: TextStyle(
                          color: theme.colorScheme.onSecondaryContainer,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    title: Text(
                      c.otherOrgName,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontWeight: FontWeight.w600),
                    ),
                    subtitle: c.lastMessage != null && c.lastMessage!.isNotEmpty
                        ? Text(
                            c.lastMessage!,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          )
                        : const Text('Entreprise'),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => Navigator.of(context).pop(
                      ClientPickerSelection(
                        userId: c.otherUserId,
                        orgId: c.otherOrgId,
                        name: c.otherOrgName,
                      ),
                    ),
                  );
                },
              ),
            ),
        ],
      ),
    );
  }
}
