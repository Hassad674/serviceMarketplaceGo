import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/conversation_entity.dart';
import '../providers/messaging_provider.dart';
import '../widgets/messaging_list/messaging_conversation_tile.dart';
import '../widgets/messaging_list/messaging_org_type_filter_row.dart';
import '../widgets/messaging_list/messaging_states.dart';

// ---------------------------------------------------------------------------
// Messaging screen -- conversation list
// ---------------------------------------------------------------------------

/// Displays a searchable, role-filterable list of real conversations
/// fetched from the backend via [conversationsProvider].
class MessagingScreen extends ConsumerStatefulWidget {
  const MessagingScreen({super.key});

  @override
  ConsumerState<MessagingScreen> createState() => _MessagingScreenState();
}

class _MessagingScreenState extends ConsumerState<MessagingScreen> {
  String _searchQuery = '';
  String _orgTypeFilter = 'all';

  List<ConversationEntity> _applyFilters(
    List<ConversationEntity> conversations,
  ) {
    return conversations.where((c) {
      final matchesType =
          _orgTypeFilter == 'all' || c.otherOrgType == _orgTypeFilter;
      final matchesSearch = _searchQuery.isEmpty ||
          c.otherOrgName
              .toLowerCase()
              .contains(_searchQuery.toLowerCase());
      return matchesType && matchesSearch;
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();
    final convState = ref.watch(conversationsProvider);

    return Scaffold(
      backgroundColor: theme.scaffoldBackgroundColor,
      appBar: AppBar(
        backgroundColor: theme.scaffoldBackgroundColor,
        elevation: 0,
        scrolledUnderElevation: 0,
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(
          l10n.messages,
          style: SoleilTextStyles.headlineMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
      ),
      body: Column(
        children: [
          // Search bar
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
            child: TextField(
              onChanged: (value) => setState(() => _searchQuery = value),
              decoration: InputDecoration(
                hintText: l10n.messagingSearchHint,
                prefixIcon: Icon(
                  Icons.search,
                  color: appColors?.mutedForeground,
                  size: 20,
                ),
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 12,
                ),
                fillColor: theme.colorScheme.surface,
                filled: true,
              ),
            ),
          ),

          // Org-type filter chips
          MessagingOrgTypeFilterRow(
            selected: _orgTypeFilter,
            onChanged: (type) => setState(() => _orgTypeFilter = type),
          ),

          const SizedBox(height: 4),

          // Conversation list
          Expanded(child: _buildBody(convState, l10n)),
        ],
      ),
    );
  }

  Widget _buildBody(ConversationsState convState, AppLocalizations l10n) {
    if (convState.isLoading) {
      return const ConversationListShimmer();
    }

    if (convState.error != null) {
      return MessagingErrorState(
        message: convState.error!,
        onRetry: () =>
            ref.read(conversationsProvider.notifier).loadConversations(),
      );
    }

    final filtered = _applyFilters(convState.conversations);

    if (filtered.isEmpty) {
      return MessagingEmptyState(message: l10n.messagingNoConversations);
    }

    final typingUsers = convState.typingUsers;

    return RefreshIndicator(
      onRefresh: () =>
          ref.read(conversationsProvider.notifier).loadConversations(),
      child: ListView.builder(
        itemCount: filtered.length + (convState.hasMore ? 1 : 0),
        // Match conversations to their tile by id so a new message
        // pushing a conversation to the top doesn't re-create every
        // tile (PERF-M-06).
        findChildIndexCallback: (key) {
          if (key is! ValueKey<String>) return null;
          final id = key.value;
          final idx = filtered.indexWhere((c) => c.id == id);
          return idx >= 0 ? idx : null;
        },
        cacheExtent: 600,
        addAutomaticKeepAlives: false,
        itemBuilder: (context, index) {
          if (index >= filtered.length) {
            // Load more trigger.
            ref.read(conversationsProvider.notifier).loadMore();
            return const Padding(
              key: ValueKey<String>('__messaging_load_more__'),
              padding: EdgeInsets.all(16),
              child: Center(
                child: SizedBox(
                  width: 24,
                  height: 24,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
            );
          }
          final conversation = filtered[index];
          return RepaintBoundary(
            key: ValueKey<String>(conversation.id),
            child: MessagingConversationTile(
              conversation: conversation,
              isTyping: typingUsers.containsKey(conversation.id),
              onTap: () {
                ref
                    .read(conversationsProvider.notifier)
                    .clearUnread(conversation.id);
                context.push('/chat/${conversation.id}');
              },
            ),
          );
        },
      ),
    );
  }
}
