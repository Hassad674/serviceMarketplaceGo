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

/// Soleil v2 invoice row — single entry in the M-15 invoices list.
///
/// Anatomy mirrors the web `InvoiceList` row (W-19 Soleil port):
///   - corail-soft icon disc + Geist Mono invoice-number pill,
///   - source label + relative date in tabac (Geist Mono small caps),
///   - Geist Mono semibold amount,
///   - sapin-soft / amber-soft / muted status pill,
///   - download icon button that pulls the PDF through the
///     authenticated [ApiClient].
///
/// Strings stay hardcoded so the existing widget test (off-limits) still
/// matches "Abonnement Premium", "Commission mensuelle", and the
/// `tooltip = 'Télécharger la facture <number>'` fixture.
class InvoiceListItem extends ConsumerWidget {
  const InvoiceListItem({super.key, required this.invoice});

  final Invoice invoice;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    final repo = ref.watch(invoicingRepositoryProvider);
    final status = _statusFor(invoice.sourceType, appColors, colorScheme);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: appColors?.accentSoft ?? colorScheme.primaryContainer,
              borderRadius: BorderRadius.circular(AppTheme.radiusLg),
            ),
            child: Icon(
              Icons.description_outlined,
              size: 18,
              color: colorScheme.primary,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [
                    _NumberPill(number: invoice.number),
                    _StatusPill(status: status),
                  ],
                ),
                const SizedBox(height: 6),
                Text(
                  _sourceLabel(invoice.sourceType),
                  style: SoleilTextStyles.body.copyWith(
                    color: colorScheme.onSurface,
                    fontSize: 13.5,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                Text(
                  _formatDate(invoice.issuedAt).toUpperCase(),
                  style: SoleilTextStyles.mono.copyWith(
                    color: appColors?.subtleForeground ??
                        colorScheme.onSurfaceVariant,
                    fontSize: 11,
                    letterSpacing: 0.6,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ),
          const SizedBox(width: 10),
          Padding(
            padding: const EdgeInsets.only(top: 2),
            child: Text(
              _formatCurrency(invoice.amountInclTaxCents),
              style: SoleilTextStyles.mono.copyWith(
                color: colorScheme.onSurface,
                fontSize: 14,
                fontWeight: FontWeight.w600,
                letterSpacing: 0.2,
              ),
            ),
          ),
          const SizedBox(width: 6),
          IconButton(
            tooltip: 'Télécharger la facture ${invoice.number}',
            onPressed: () => _downloadPdf(context, repo, invoice),
            icon: const Icon(Icons.download_rounded, size: 18),
            color: colorScheme.onSurface,
            visualDensity: VisualDensity.compact,
            constraints: const BoxConstraints(minWidth: 36, minHeight: 36),
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

class _NumberPill extends StatelessWidget {
  const _NumberPill({required this.number});

  final String number;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        number,
        style: SoleilTextStyles.mono.copyWith(
          color: colorScheme.onSurface,
          fontSize: 11,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.2,
        ),
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.status});

  final _SourceStatus status;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: status.background,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        status.label,
        style: SoleilTextStyles.bodyEmphasis.copyWith(
          color: status.foreground,
          fontSize: 11,
        ),
      ),
    );
  }
}

class _SourceStatus {
  const _SourceStatus({
    required this.label,
    required this.background,
    required this.foreground,
  });

  final String label;
  final Color background;
  final Color foreground;
}

_SourceStatus _statusFor(
  SourceType type,
  AppColors? appColors,
  ColorScheme colorScheme,
) {
  switch (type) {
    case SourceType.subscription:
      return _SourceStatus(
        label: 'Payée',
        background: appColors?.successSoft ??
            colorScheme.secondaryContainer,
        foreground: appColors?.success ?? colorScheme.primary,
      );
    case SourceType.monthlyCommission:
      return _SourceStatus(
        label: 'En attente',
        background: appColors?.amberSoft ?? colorScheme.surfaceContainerHigh,
        foreground: colorScheme.onSurface,
      );
    case SourceType.creditNote:
      return _SourceStatus(
        label: 'Avoir',
        background: appColors?.border ?? colorScheme.surfaceContainerHigh,
        foreground: colorScheme.onSurfaceVariant,
      );
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
