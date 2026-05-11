import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/wallet_summary_entity.dart';
import '../providers/wallet_provider.dart';

/// WalletUnifiedHistory — the merged mission + commission timeline
/// below the WALLET-UNIFY Run D hero on mobile. Each row carries:
///
///   - a type icon (💼 missions / 🤝 commissions)
///   - the amount, right-aligned, monospaced
///   - a status badge in one of four tones (paid / pending /
///     escrowed / failed)
///
/// Pagination is cursor-driven: each "Charger plus" tap advances the
/// local cursor state which keys the family provider. Initial mount
/// hits the cursor-less first page.
class WalletUnifiedHistory extends ConsumerStatefulWidget {
  const WalletUnifiedHistory({super.key});

  @override
  ConsumerState<WalletUnifiedHistory> createState() =>
      _WalletUnifiedHistoryState();
}

class _WalletUnifiedHistoryState extends ConsumerState<WalletUnifiedHistory> {
  final List<WalletSummaryTransaction> _accumulated = [];
  String? _cursor;
  bool _seeded = false;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final asyncSummary = ref.watch(walletSummaryProvider(_cursor));

    return Container(
      key: const ValueKey('wallet-unified-history'),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border:
            Border.all(color: theme.dividerColor.withValues(alpha: 0.5)),
        boxShadow: AppTheme.cardShadow,
      ),
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _Header(
            title: l10n.walletUnifiedHistoryTitle,
            subtitle: l10n.walletUnifiedHistorySubtitle,
          ),
          const SizedBox(height: 12),
          Divider(
            height: 1,
            color: theme.dividerColor.withValues(alpha: 0.5),
          ),
          ...asyncSummary.when(
            loading: () => <Widget>[
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 24),
                child: Center(child: CircularProgressIndicator()),
              ),
            ],
            error: (e, _) => <Widget>[
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 24),
                child: Center(
                  child: Text(
                    l10n.walletUnifiedHistoryEmpty,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurface
                          .withValues(alpha: 0.5),
                    ),
                  ),
                ),
              ),
            ],
            data: (summary) {
              _seedIfNeeded(summary);
              final rows = _rowsToRender(summary);
              return _buildBody(theme, l10n, rows, summary);
            },
          ),
        ],
      ),
    );
  }

  void _seedIfNeeded(WalletSummary summary) {
    if (_seeded) return;
    if (_cursor != null) return;
    if (summary.recentTransactions.isEmpty) return;
    _seeded = true;
    _accumulated.addAll(summary.recentTransactions);
  }

  List<WalletSummaryTransaction> _rowsToRender(WalletSummary summary) {
    if (_accumulated.isNotEmpty) return _accumulated;
    return summary.recentTransactions;
  }

  List<Widget> _buildBody(
    ThemeData theme,
    AppLocalizations l10n,
    List<WalletSummaryTransaction> rows,
    WalletSummary summary,
  ) {
    if (rows.isEmpty) {
      return [
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 24),
          child: Center(
            child: Text(
              l10n.walletUnifiedHistoryEmpty,
              style: theme.textTheme.bodySmall?.copyWith(
                color:
                    theme.colorScheme.onSurface.withValues(alpha: 0.5),
              ),
            ),
          ),
        ),
      ];
    }
    final hasMore = (summary.nextCursor ?? '').isNotEmpty;
    return [
      const SizedBox(height: 4),
      ...rows.map((row) => _TransactionRow(row: row)),
      if (hasMore)
        Padding(
          padding: const EdgeInsets.only(top: 12),
          child: Center(
            child: OutlinedButton(
              key: const ValueKey('wallet-history-load-more'),
              onPressed: () => _loadMore(summary),
              child: Text(l10n.walletUnifiedHistoryLoadMore),
            ),
          ),
        ),
    ];
  }

  void _loadMore(WalletSummary summary) {
    final next = summary.nextCursor;
    if (next == null || next.isEmpty) return;
    setState(() {
      // Append the rows just rendered then advance to the next cursor
      // so the next FutureProvider entry replaces _accumulated.
      _cursor = next;
    });
    // The next page's data lives under walletSummaryProvider(next);
    // we don't pre-merge here — when the page resolves, its rows
    // will be added via _seedIfNeeded? No: _seeded gates only the
    // first page. Use a one-shot post-frame listen instead.
    Future.microtask(() async {
      final value =
          await ref.read(walletSummaryProvider(next).future);
      if (!mounted) return;
      setState(() {
        _accumulated.addAll(value.recentTransactions);
      });
    });
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.title, required this.subtitle});

  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
            color: theme.colorScheme.onSurface,
          ),
        ),
        const SizedBox(height: 2),
        Text(
          subtitle,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurface.withValues(alpha: 0.55),
          ),
        ),
      ],
    );
  }
}

class _TransactionRow extends StatelessWidget {
  const _TransactionRow({required this.row});

  final WalletSummaryTransaction row;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final isMission = row.isMission;
    final accent = theme.extension<AppColors>();
    final typeLabel = isMission
        ? l10n.walletUnifiedHistoryRowMission
        : l10n.walletUnifiedHistoryRowCommission;
    final title = (row.missionTitle ?? '').isNotEmpty
        ? row.missionTitle!
        : l10n.walletUnifiedHistoryRowUntitled;
    final amount = formatWalletSummaryCents(row.amountCents);
    final dateText = _formatDate(row.occurredAt);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10, horizontal: 2),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: isMission
                  ? theme.colorScheme.primary.withValues(alpha: 0.12)
                  : (accent?.success ?? theme.colorScheme.primary)
                      .withValues(alpha: 0.12),
              borderRadius:
                  BorderRadius.circular(AppTheme.radiusSm),
            ),
            child: Icon(
              isMission ? Icons.work_outline : Icons.handshake_outlined,
              size: 18,
              color: isMission
                  ? theme.colorScheme.primary
                  : (accent?.success ?? theme.colorScheme.primary),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                    color: theme.colorScheme.onSurface,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                Text(
                  '$typeLabel · $dateText',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurface
                        .withValues(alpha: 0.55),
                    fontSize: 11,
                  ),
                ),
              ],
            ),
          ),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                '+$amount',
                style: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                  fontFamily: 'monospace',
                  color: theme.colorScheme.onSurface,
                ),
              ),
              const SizedBox(height: 4),
              _StatusBadge(status: row.status),
            ],
          ),
        ],
      ),
    );
  }

  static String _formatDate(String iso) {
    try {
      final d = DateTime.parse(iso);
      final dd = d.day.toString().padLeft(2, '0');
      final months = [
        'janv.',
        'févr.',
        'mars',
        'avr.',
        'mai',
        'juin',
        'juil.',
        'août',
        'sept.',
        'oct.',
        'nov.',
        'déc.',
      ];
      final mm = months[(d.month - 1).clamp(0, 11)];
      return '$dd $mm ${d.year}';
    } catch (_) {
      return '';
    }
  }
}

/// Compact status badge for the merged transaction list. Tone palette
/// mirrors the web `WalletStatusBadge` (4 tones — paid / pending /
/// escrowed / failed).
class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final accent = theme.extension<AppColors>();
    final tone = walletStatusTone(status);
    Color bg;
    Color fg;
    String label;
    switch (tone) {
      case WalletStatusTone.paid:
        bg = (accent?.success ?? theme.colorScheme.primary)
            .withValues(alpha: 0.16);
        fg = accent?.success ?? theme.colorScheme.primary;
        label = l10n.walletUnifiedHistoryStatusPaid;
        break;
      case WalletStatusTone.pending:
        bg = (accent?.warning ?? theme.colorScheme.tertiary)
            .withValues(alpha: 0.18);
        fg = accent?.warning ?? theme.colorScheme.tertiary;
        label = l10n.walletUnifiedHistoryStatusPending;
        break;
      case WalletStatusTone.escrowed:
        bg = theme.colorScheme.onSurface.withValues(alpha: 0.08);
        fg = theme.colorScheme.onSurface.withValues(alpha: 0.7);
        label = l10n.walletUnifiedHistoryStatusEscrowed;
        break;
      case WalletStatusTone.failed:
        bg = theme.colorScheme.error.withValues(alpha: 0.14);
        fg = theme.colorScheme.error;
        label = l10n.walletUnifiedHistoryStatusFailed;
        break;
    }
    return Container(
      padding:
          const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          color: fg,
          fontWeight: FontWeight.w700,
          fontSize: 10,
        ),
      ),
    );
  }
}
