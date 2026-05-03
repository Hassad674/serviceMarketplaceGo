import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../messaging/domain/entities/conversation_entity.dart';
import '../../../messaging/data/messaging_repository_impl.dart';
import 'picker_selection.dart';

/// showProviderPickerSheet opens the provider-picking modal bottom sheet.
/// The sheet has two tabs:
///
///   1. Rechercher — freelances + agences from /api/v1/profiles/search,
///      filtered client-side by name.
///   2. Depuis une conversation — the current user's conversations filtered
///      to provider_personal / agency (warm contacts).
///
/// Returns the selected provider (or null if the user dismissed the sheet).
Future<ProviderPickerSelection?> showProviderPickerSheet(
  BuildContext context,
) async {
  return showModalBottomSheet<ProviderPickerSelection?>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
    ),
    builder: (ctx) => const _ProviderPickerSheet(),
  );
}

class _ProviderPickerSheet extends ConsumerStatefulWidget {
  const _ProviderPickerSheet();

  @override
  ConsumerState<_ProviderPickerSheet> createState() => _ProviderPickerSheetState();
}

class _ProviderPickerSheetState extends ConsumerState<_ProviderPickerSheet>
    with SingleTickerProviderStateMixin {
  late final TabController _tabController;
  final _searchCtrl = TextEditingController();

  bool _loading = false;
  String? _error;
  List<Map<String, dynamic>> _freelancers = const [];
  List<Map<String, dynamic>> _agencies = const [];
  List<ConversationEntity> _conversations = const [];
  String _query = '';

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _loadAll();
  }

  @override
  void dispose() {
    _tabController.dispose();
    _searchCtrl.dispose();
    super.dispose();
  }

  Future<void> _loadAll() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = ref.read(apiClientProvider);
      final results = await Future.wait([
        api.get('/api/v1/profiles/search', queryParameters: {'type': 'freelancer', 'limit': '50'}),
        api.get('/api/v1/profiles/search', queryParameters: {'type': 'agency', 'limit': '50'}),
      ]);
      final msgRepo = ref.read(messagingRepositoryProvider);
      final convs = await msgRepo.getConversations(limit: 50);

      final freelancersData = (results[0].data as Map<String, dynamic>?)?['data'] as List? ?? const [];
      final agenciesData = (results[1].data as Map<String, dynamic>?)?['data'] as List? ?? const [];

      if (!mounted) return;
      setState(() {
        _freelancers = freelancersData.cast<Map<String, dynamic>>();
        _agencies = agenciesData.cast<Map<String, dynamic>>();
        _conversations = convs.data;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = 'Impossible de charger la liste. Réessayez dans un instant.';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SizedBox(
      height: MediaQuery.of(context).size.height * 0.85,
      child: Column(
        children: [
          _Header(
            title: 'Choisir un prestataire',
            onClose: () => Navigator.of(context).pop(),
          ),
          Container(
            color: theme.colorScheme.surfaceContainerHighest,
            child: TabBar(
              controller: _tabController,
              labelColor: theme.colorScheme.primary,
              unselectedLabelColor: theme.colorScheme.onSurfaceVariant,
              indicatorColor: theme.colorScheme.primary,
              tabs: const [
                Tab(text: 'Rechercher', icon: Icon(Icons.search, size: 18)),
                Tab(text: 'Conversations', icon: Icon(Icons.chat_bubble_outline, size: 18)),
              ],
            ),
          ),
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
          else
            Expanded(
              child: TabBarView(
                controller: _tabController,
                children: [
                  _buildSearchTab(theme),
                  _buildConversationsTab(theme),
                ],
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildSearchTab(ThemeData theme) {
    final all = [..._freelancers, ..._agencies];
    final q = _query.trim().toLowerCase();
    final filtered = q.isEmpty
        ? all
        : all.where((p) => ((p['name'] as String?) ?? '').toLowerCase().contains(q)).toList();

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(12),
          child: TextField(
            controller: _searchCtrl,
            onChanged: (v) => setState(() => _query = v),
            decoration: InputDecoration(
              hintText: 'Filtrer par nom…',
              prefixIcon: const Icon(Icons.search, size: 20),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(10),
                borderSide: BorderSide(color: theme.colorScheme.outlineVariant),
              ),
              contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
            ),
          ),
        ),
        Expanded(
          child: filtered.isEmpty
              ? const _EmptyState(message: 'Aucun résultat pour cette recherche.')
              : ListView.separated(
                  itemCount: filtered.length,
                  separatorBuilder: (_, __) => Divider(height: 1, color: theme.colorScheme.outlineVariant),
                  itemBuilder: (ctx, i) {
                    final p = filtered[i];
                    return _ProfileTile(
                      name: (p['name'] as String?) ?? '—',
                      subtitle: orgTypeLabel((p['org_type'] as String?) ?? ''),
                      onTap: () => Navigator.of(context).pop(
                        ProviderPickerSelection(
                          userId: (p['owner_user_id'] as String?) ?? '',
                          orgId: (p['organization_id'] as String?) ?? '',
                          name: (p['name'] as String?) ?? '—',
                          orgType: (p['org_type'] as String?) ?? '',
                        ),
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }

  Widget _buildConversationsTab(ThemeData theme) {
    final providers = _conversations
        .where((c) => c.otherOrgType == 'provider_personal' || c.otherOrgType == 'agency')
        .toList();

    if (providers.isEmpty) {
      return const _EmptyState(
        message:
            'Aucune conversation avec un freelance ou une agence.\nPassez par l\'onglet Rechercher pour choisir un prestataire du catalogue.',
      );
    }

    return ListView.separated(
      itemCount: providers.length,
      separatorBuilder: (_, __) => Divider(height: 1, color: theme.colorScheme.outlineVariant),
      itemBuilder: (ctx, i) {
        final c = providers[i];
        return _ProfileTile(
          name: c.otherOrgName,
          subtitle: orgTypeLabel(c.otherOrgType),
          onTap: () => Navigator.of(context).pop(
            ProviderPickerSelection(
              userId: c.otherUserId,
              orgId: c.otherOrgId,
              name: c.otherOrgName,
              orgType: c.otherOrgType,
            ),
          ),
        );
      },
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.title, required this.onClose});

  final String title;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 8, 12),
      child: Row(
        children: [
          Expanded(
            child: Text(
              title,
              style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.close),
            onPressed: onClose,
            tooltip: 'Fermer',
          ),
        ],
      ),
    );
  }
}

class _ProfileTile extends StatelessWidget {
  const _ProfileTile({
    required this.name,
    required this.subtitle,
    required this.onTap,
  });

  final String name;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final initial = name.isNotEmpty ? name.substring(0, 1).toUpperCase() : '?';
    return ListTile(
      leading: CircleAvatar(
        radius: 18,
        backgroundColor: theme.colorScheme.primaryContainer,
        child: Text(
          initial,
          style: TextStyle(
            color: theme.colorScheme.onPrimaryContainer,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      title: Text(
        name,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: const TextStyle(fontWeight: FontWeight.w600),
      ),
      subtitle: Text(subtitle),
      trailing: const Icon(Icons.chevron_right),
      onTap: onTap,
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Text(
          message,
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.bodySmall,
        ),
      ),
    );
  }
}
