import '../../domain/entities/message_entity.dart';

/// Removes stale "proposal_completion_requested" cards once the
/// proposal moves past that state.
///
/// A completion request is resolved by any of: `proposal_completed`,
/// `proposal_completion_rejected`, `milestone_released`,
/// `milestone_auto_approved`, `proposal_cancelled`,
/// `proposal_auto_closed`. Once any of these is in the conversation
/// for a given `proposal_id`, the earlier yellow "Completion
/// requested" card becomes noise and is hidden.
///
/// Multi-milestone proposals emit a fresh `completion_requested`
/// card for each milestone, so hiding is keyed on the proposal id
/// AND the relative order — we compute the set of resolved proposal
/// ids once per render and drop any `completion_requested` that
/// belongs to them.
List<MessageEntity> filterVisibleChatMessages(List<MessageEntity> messages) {
  const resolverTypes = <String>{
    'proposal_completed',
    'proposal_completion_rejected',
    'milestone_released',
    'milestone_auto_approved',
    'proposal_cancelled',
    'proposal_auto_closed',
  };

  final resolved = <String>{};
  for (final m in messages) {
    if (!resolverTypes.contains(m.type)) continue;
    final meta = m.metadata;
    if (meta is Map<String, dynamic>) {
      final pid = meta['proposal_id'];
      if (pid is String) resolved.add(pid);
    }
  }
  if (resolved.isEmpty) return messages;
  return messages.where((m) {
    if (m.type != 'proposal_completion_requested') return true;
    final meta = m.metadata;
    if (meta is Map<String, dynamic>) {
      final pid = meta['proposal_id'];
      if (pid is String) return !resolved.contains(pid);
    }
    return true;
  }).toList();
}
