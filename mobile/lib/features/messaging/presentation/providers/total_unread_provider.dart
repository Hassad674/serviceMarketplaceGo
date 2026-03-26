import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'conversations_provider.dart';

// ---------------------------------------------------------------------------
// Total unread count provider
// ---------------------------------------------------------------------------

/// Provides the total unread count across all conversations for badge display.
///
/// Recomputed from the conversations state whenever it changes.
final totalUnreadProvider = Provider<int>((ref) {
  final convState = ref.watch(conversationsProvider);
  return convState.conversations.fold<int>(
    0,
    (sum, c) => sum + c.unreadCount,
  );
});
