import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../domain/entities/referral_entity.dart';
import '../providers/referral_provider.dart';
import '../widgets/referral_status_chip.dart';

/// ReferralDashboardScreen is the apporteur's home for the business
/// referral feature. Mirrors the web /referrals dashboard:
/// stat cards on top, then sections (À traiter / Pending / Active /
/// History) listing the referrals.
class ReferralDashboardScreen extends ConsumerWidget {
  const ReferralDashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final myAsync = ref.watch(myReferralsProvider);
    final incomingAsync = ref.watch(incomingReferralsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Referrals'),
        elevation: 0,
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/referrals/new'),
        icon: const Icon(Icons.add),
        label: const Text('New intro'),
      ),
      body: RefreshIndicator(
        onRefresh: () async {
          ref.invalidate(myReferralsProvider);
          ref.invalidate(incomingReferralsProvider);
          await Future.wait([
            ref.read(myReferralsProvider.future),
            ref.read(incomingReferralsProvider.future),
          ]);
        },
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            myAsync.when(
              loading: () => const _LoadingStats(),
              error: (e, _) => const _ErrorBanner(message: 'Could not load your referrals.'),
              data: (referrals) => _StatRow(referrals: referrals),
            ),
            const SizedBox(height: 24),
            _Section(
              title: 'Inbox',
              description: 'Intros where you must take action.',
              child: incomingAsync.when(
                loading: () => const _SectionLoading(),
                error: (_, __) => const SizedBox.shrink(),
                data: (items) => items.isEmpty
                    ? const _EmptyState(message: 'No incoming intros.')
                    : _ReferralList(items: items),
              ),
            ),
            const SizedBox(height: 16),
            _Section(
              title: 'Pending',
              description: 'Your intros waiting on another party.',
              child: myAsync.when(
                loading: () => const _SectionLoading(),
                error: (_, __) => const SizedBox.shrink(),
                data: (items) {
                  final pending = items.where((r) => r.isPending).toList();
                  return pending.isEmpty
                      ? const _EmptyState(message: 'No pending intros.')
                      : _ReferralList(items: pending);
                },
              ),
            ),
            const SizedBox(height: 16),
            _Section(
              title: 'Active',
              description: 'Activated, in their exclusivity window.',
              child: myAsync.when(
                loading: () => const _SectionLoading(),
                error: (_, __) => const SizedBox.shrink(),
                data: (items) {
                  final active = items.where((r) => r.status == 'active').toList();
                  return active.isEmpty
                      ? const _EmptyState(message: 'No active referrals yet.')
                      : _ReferralList(items: active);
                },
              ),
            ),
            const SizedBox(height: 16),
            _Section(
              title: 'History',
              description: 'Terminated, expired, cancelled, rejected.',
              child: myAsync.when(
                loading: () => const _SectionLoading(),
                error: (_, __) => const SizedBox.shrink(),
                data: (items) {
                  final history = items.where((r) => r.isTerminal).toList();
                  return history.isEmpty
                      ? const _EmptyState(message: 'History will appear here.')
                      : _ReferralList(items: history);
                },
              ),
            ),
            const SizedBox(height: 80), // breathing room above FAB
          ],
        ),
      ),
    );
  }
}

class _StatRow extends StatelessWidget {
  const _StatRow({required this.referrals});

  final List<Referral> referrals;

  @override
  Widget build(BuildContext context) {
    final pending = referrals.where((r) => r.isPending).length;
    final active = referrals.where((r) => r.status == 'active').length;
    return Row(
      children: [
        Expanded(child: _StatCard(label: 'Pending', value: '$pending', icon: Icons.access_time, tone: Colors.amber.shade100, iconColor: Colors.amber.shade700)),
        const SizedBox(width: 12),
        Expanded(child: _StatCard(label: 'Active', value: '$active', icon: Icons.check_circle, tone: Colors.green.shade100, iconColor: Colors.green.shade700)),
        const SizedBox(width: 12),
        Expanded(child: _StatCard(label: 'Total', value: '${referrals.length}', icon: Icons.auto_awesome, tone: Colors.pink.shade100, iconColor: Colors.pink.shade700)),
      ],
    );
  }
}

class _StatCard extends StatelessWidget {
  const _StatCard({
    required this.label,
    required this.value,
    required this.icon,
    required this.tone,
    required this.iconColor,
  });

  final String label;
  final String value;
  final IconData icon;
  final Color tone;
  final Color iconColor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(color: theme.colorScheme.outlineVariant),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: tone,
              shape: BoxShape.circle,
            ),
            child: Icon(icon, size: 18, color: iconColor),
          ),
          const SizedBox(height: 8),
          Text(
            label,
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 2),
          Text(
            value,
            style: theme.textTheme.headlineSmall?.copyWith(
              fontWeight: FontWeight.w700,
              fontFeatures: const [FontFeature.tabularFigures()],
            ),
          ),
        ],
      ),
    );
  }
}

class _Section extends StatelessWidget {
  const _Section({
    required this.title,
    required this.description,
    required this.child,
  });

  final String title;
  final String description;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
        ),
        Text(
          description,
          style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant),
        ),
        const SizedBox(height: 8),
        child,
      ],
    );
  }
}

class _ReferralList extends StatelessWidget {
  const _ReferralList({required this.items});

  final List<Referral> items;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [for (final r in items) _ReferralTile(referral: r)],
    );
  }
}

class _ReferralTile extends StatelessWidget {
  const _ReferralTile({required this.referral});

  final Referral referral;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final rateLabel = referral.ratePct == null
        ? '—'
        : '${referral.ratePct!.toStringAsFixed(referral.ratePct! % 1 == 0 ? 0 : 1)}%';
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      elevation: 0,
      shape: RoundedRectangleBorder(
        side: BorderSide(color: theme.colorScheme.outlineVariant),
        borderRadius: BorderRadius.circular(12),
      ),
      child: ListTile(
        title: Row(
          children: [
            ReferralStatusChip(status: referral.status),
            const SizedBox(width: 8),
            Text(
              'v${referral.version} · $rateLabel · ${referral.durationMonths}mo',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Text(
            'Provider ${referral.providerId.substring(0, 8)} → Client ${referral.clientId.substring(0, 8)}',
            style: theme.textTheme.bodySmall,
          ),
        ),
        trailing: const Icon(Icons.chevron_right),
        onTap: () => context.push('/referrals/${referral.id}'),
      ),
    );
  }
}

class _LoadingStats extends StatelessWidget {
  const _LoadingStats();

  @override
  Widget build(BuildContext context) {
    return Row(
      children: List.generate(
        3,
        (i) => Expanded(
          child: Container(
            margin: EdgeInsets.only(right: i < 2 ? 12 : 0),
            height: 96,
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(16),
            ),
          ),
        ),
      ),
    );
  }
}

class _SectionLoading extends StatelessWidget {
  const _SectionLoading();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 24),
      child: Center(
        child: SizedBox(
          height: 24,
          width: 24,
          child: CircularProgressIndicator(strokeWidth: 2),
        ),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 24),
      alignment: Alignment.center,
      decoration: BoxDecoration(
        border: Border.all(
          color: theme.colorScheme.outlineVariant,
          style: BorderStyle.solid,
        ),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(message, style: theme.textTheme.bodySmall),
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.errorContainer.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(message, style: theme.textTheme.bodySmall),
    );
  }
}
