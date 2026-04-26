import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/invoice.dart';
import '../providers/invoicing_providers.dart';

/// Single row in the invoices list.
///
/// Displays the invoice number, the issue date, the source motif (FR
/// label), the inclusive amount and a trailing "Télécharger" icon
/// button. Tapping the button opens the PDF via `launchUrl` with
/// `LaunchMode.externalApplication`. The endpoint behind the URL
/// responds with HTTP 302 to a 5-minute presigned URL — the OS
/// browser handles the redirect and download natively.
class InvoiceListItem extends ConsumerWidget {
  const InvoiceListItem({super.key, required this.invoice});

  final Invoice invoice;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final repo = ref.watch(invoicingRepositoryProvider);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: const Color(0xFFFFE4E6), // rose-100
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: const Icon(
              Icons.description_outlined,
              size: 18,
              color: Color(0xFFBE123C), // rose-700
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  invoice.number,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                Text(
                  '${_formatDate(invoice.issuedAt)} · '
                  '${_sourceLabel(invoice.sourceType)}',
                  style: theme.textTheme.bodySmall,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Text(
            _formatCurrency(invoice.amountInclTaxCents),
            style: theme.textTheme.bodyMedium?.copyWith(
              fontFamily: 'monospace',
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(width: 8),
          IconButton(
            tooltip: 'Télécharger la facture ${invoice.number}',
            onPressed: () => _downloadPdf(context, repo.getInvoicePDFURL(invoice.id)),
            icon: const Icon(Icons.download_rounded, size: 20),
            color: theme.colorScheme.onSurface.withValues(alpha: 0.7),
          ),
        ],
      ),
    );
  }

  Future<void> _downloadPdf(BuildContext context, String url) async {
    try {
      final ok = await launchUrl(
        Uri.parse(url),
        mode: LaunchMode.externalApplication,
      );
      if (!ok && context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Impossible d\'ouvrir le PDF.')),
        );
      }
    } catch (e) {
      debugPrint('invoice pdf launch failed: $e');
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Impossible d\'ouvrir le PDF.')),
        );
      }
    }
  }
}

String _formatDate(DateTime d) => DateFormat('dd/MM/yyyy').format(d);

String _formatCurrency(int cents) {
  final amount = cents / 100;
  try {
    return NumberFormat.currency(
      locale: 'fr_FR',
      symbol: '€',
      decimalDigits: 2,
    ).format(amount);
  } catch (_) {
    final euros = cents ~/ 100;
    final remainder = (cents.abs() % 100).toString().padLeft(2, '0');
    final sign = cents < 0 ? '-' : '';
    return '$sign$euros,$remainder €';
  }
}

String _sourceLabel(SourceType type) {
  switch (type) {
    case SourceType.subscription:
      return 'Abonnement Premium';
    case SourceType.monthlyCommission:
      return 'Commission mensuelle';
    case SourceType.creditNote:
      return 'Avoir';
  }
}
