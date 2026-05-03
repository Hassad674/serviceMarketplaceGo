import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:path_provider/path_provider.dart';
import 'package:share_plus/share_plus.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/invoice.dart';
import '../../domain/repositories/invoicing_repository.dart';
import '../providers/invoicing_providers.dart';
import '../../../../core/theme/app_palette.dart';

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
              color: AppPalette.rose100, // rose-100
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: const Icon(
              Icons.description_outlined,
              size: 18,
              color: AppPalette.rose700, // rose-700
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
            onPressed: () => _downloadPdf(context, repo, invoice),
            icon: const Icon(Icons.download_rounded, size: 20),
            color: theme.colorScheme.onSurface.withValues(alpha: 0.7),
          ),
        ],
      ),
    );
  }

  Future<void> _downloadPdf(
    BuildContext context,
    InvoicingRepository repo,
    Invoice invoice,
  ) async {
    // Pull the PDF through the authenticated ApiClient (so the bearer
    // token is sent), persist it in the temp dir, then hand it to the
    // system share/save sheet. The previous launchUrl flow opened the
    // raw API URL in the system browser which has no auth cookie /
    // token and bounced with 401 unauthorized.
    final messenger = ScaffoldMessenger.maybeOf(context);
    messenger?.showSnackBar(
      SnackBar(
        content: Text('Téléchargement de ${invoice.number}…'),
        duration: const Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
    try {
      final bytes = await repo.downloadInvoicePDFBytes(invoice.id);
      final dir = await getTemporaryDirectory();
      final path = '${dir.path}/${invoice.number}.pdf';
      final file = File(path);
      await file.writeAsBytes(bytes, flush: true);
      if (!context.mounted) return;
      await Share.shareXFiles(
        [XFile(path, mimeType: 'application/pdf', name: '${invoice.number}.pdf')],
        subject: 'Facture ${invoice.number}',
      );
    } catch (e) {
      debugPrint('invoice pdf download failed: $e');
      if (!context.mounted) return;
      messenger?.showSnackBar(
        const SnackBar(content: Text("Impossible de télécharger le PDF.")),
      );
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
