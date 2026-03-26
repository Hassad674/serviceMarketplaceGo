import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../mock_data.dart';
import '../../types/conversation.dart';

// ---------------------------------------------------------------------------
// Role color mapping — matches web role badge colors
// ---------------------------------------------------------------------------

const _roleColors = {
  'agency': Color(0xFF2563EB), // blue-600
  'freelancer': Color(0xFFF43F5E), // rose-500
  'enterprise': Color(0xFF8B5CF6), // purple-500
};

// ---------------------------------------------------------------------------
// Messaging screen — conversation list
// ---------------------------------------------------------------------------

/// Displays a searchable, role-filterable list of conversations.
class MessagingScreen extends StatefulWidget {
  const MessagingScreen({super.key});

  @override
  State<MessagingScreen> createState() => _MessagingScreenState();
}

class _MessagingScreenState extends State<MessagingScreen> {
  String _searchQuery = '';
  String _roleFilter = 'all';

  List<Conversation> get _filteredConversations {
    return mockConversations.where((c) {
      final matchesRole = _roleFilter == 'all' || c.role == _roleFilter;
      final matchesSearch = _searchQuery.isEmpty ||
          c.name.toLowerCase().contains(_searchQuery.toLowerCase());
      return matchesRole && matchesSearch;
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();

    return Scaffold(
      appBar: AppBar(title: Text(l10n.messages)),
      body: Column(
        children: [
          // Search bar
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
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
                  vertical: 10,
                ),
              ),
            ),
          ),

          // Role filter chips
          _RoleFilterRow(
            selected: _roleFilter,
            onChanged: (role) => setState(() => _roleFilter = role),
          ),

          const SizedBox(height: 4),

          // Conversation list
          Expanded(
            child: _filteredConversations.isEmpty
                ? _EmptyState(message: l10n.messagingNoConversations)
                : ListView.builder(
                    itemCount: _filteredConversations.length,
                    itemBuilder: (context, index) {
                      final conversation = _filteredConversations[index];
                      return _ConversationTile(
                        conversation: conversation,
                        onTap: () => context.push(
                          '/chat/${conversation.id}',
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

// ---------------------------------------------------------------------------
// Role filter row
// ---------------------------------------------------------------------------

class _RoleFilterRow extends StatelessWidget {
  const _RoleFilterRow({
    required this.selected,
    required this.onChanged,
  });

  final String selected;
  final void Function(String) onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final filters = [
      ('all', l10n.messagingAllRoles, null),
      ('agency', l10n.messagingAgency, const Color(0xFF2563EB)),
      ('freelancer', l10n.messagingFreelancer, const Color(0xFFF43F5E)),
      ('enterprise', l10n.messagingEnterprise, const Color(0xFF8B5CF6)),
    ];

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: filters.map((filter) {
          final (key, label, color) = filter;
          final isSelected = selected == key;

          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: FilterChip(
              label: Text(
                label,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: isSelected
                      ? Colors.white
                      : (color ?? Theme.of(context).colorScheme.onSurface),
                ),
              ),
              selected: isSelected,
              onSelected: (_) => onChanged(key),
              backgroundColor: color?.withValues(alpha: 0.08) ??
                  Theme.of(context)
                      .colorScheme
                      .onSurface
                      .withValues(alpha: 0.06),
              selectedColor: color ?? Theme.of(context).colorScheme.onSurface,
              side: BorderSide.none,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(20),
              ),
              showCheckmark: false,
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            ),
          );
        }).toList(),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Conversation tile
// ---------------------------------------------------------------------------

class _ConversationTile extends StatelessWidget {
  const _ConversationTile({
    required this.conversation,
    required this.onTap,
  });

  final Conversation conversation;
  final VoidCallback onTap;

  String get _initials {
    return conversation.name
        .split(' ')
        .map((w) => w.isNotEmpty ? w[0] : '')
        .join()
        .substring(0, conversation.name.split(' ').length >= 2 ? 2 : 1)
        .toUpperCase();
  }

  Color get _roleColor => _roleColors[conversation.role] ?? Colors.grey;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          border: Border(
            left: BorderSide(
              color: conversation.unread > 0
                  ? _roleColor
                  : Colors.transparent,
              width: 3,
            ),
            bottom: BorderSide(
              color: appColors?.border ?? theme.dividerColor,
              width: 0.5,
            ),
          ),
        ),
        child: Row(
          children: [
            // Avatar with online indicator
            _Avatar(
              initials: _initials,
              online: conversation.online,
            ),
            const SizedBox(width: 12),

            // Name + last message
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          conversation.name,
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontSize: 14,
                            fontWeight: conversation.unread > 0
                                ? FontWeight.w700
                                : FontWeight.w600,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      if (conversation.lastMessageAt != null)
                        Text(
                          conversation.lastMessageAt!,
                          style: theme.textTheme.bodySmall?.copyWith(
                            fontSize: 11,
                            color: appColors?.mutedForeground,
                          ),
                        ),
                    ],
                  ),
                  const SizedBox(height: 2),
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          conversation.lastMessage ??
                              AppLocalizations.of(context)!.messagingNoMessages,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: appColors?.mutedForeground,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      if (conversation.unread > 0)
                        Container(
                          margin: const EdgeInsets.only(left: 8),
                          padding: const EdgeInsets.symmetric(
                            horizontal: 7,
                            vertical: 2,
                          ),
                          decoration: BoxDecoration(
                            color: const Color(0xFFF43F5E),
                            borderRadius: BorderRadius.circular(10),
                          ),
                          child: Text(
                            '${conversation.unread}',
                            style: const TextStyle(
                              color: Colors.white,
                              fontSize: 10,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Avatar widget
// ---------------------------------------------------------------------------

class _Avatar extends StatelessWidget {
  const _Avatar({
    required this.initials,
    required this.online,
  });

  final String initials;
  final bool online;

  @override
  Widget build(BuildContext context) {
    return Stack(
      clipBehavior: Clip.none,
      children: [
        Container(
          width: 44,
          height: 44,
          decoration: const BoxDecoration(
            shape: BoxShape.circle,
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [
                Color(0xFFF43F5E), // rose-500
                Color(0xFF8B5CF6), // purple-600
              ],
            ),
          ),
          child: Center(
            child: Text(
              initials,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 14,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ),
        if (online)
          Positioned(
            bottom: 0,
            right: 0,
            child: Container(
              width: 12,
              height: 12,
              decoration: BoxDecoration(
                color: const Color(0xFF22C55E), // emerald-500
                shape: BoxShape.circle,
                border: Border.all(
                  color: Theme.of(context).colorScheme.surface,
                  width: 2,
                ),
              ),
            ),
          ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Empty state
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.chat_outlined,
            size: 48,
            color: theme.colorScheme.onSurface.withValues(alpha: 0.2),
          ),
          const SizedBox(height: 12),
          Text(
            message,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
            ),
          ),
        ],
      ),
    );
  }
}
