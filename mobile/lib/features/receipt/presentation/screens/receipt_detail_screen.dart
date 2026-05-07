import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';
import 'package:share_plus/share_plus.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/receipt.dart';
import '../../domain/entities/receipt_party.dart';
import '../../domain/repositories/receipt_repository.dart';
import '../providers/receipt_providers.dart';
import '../widgets/receipt_card.dart' show formatReceiptAmount;

/// Soleil v2 receipt detail screen.
///
/// Pulls the full snapshot through [receiptDetailProvider] and renders
/// three Soleil cards (Client / Prestataire / Apporteur) with their
/// billing identity. The bottom CTA pulls the PDF through the
/// authenticated [ApiClient] then hands the file to the system share
/// sheet so the user can save / forward / open it.
///
/// When `snapshotAvailable: false`, the parties are absent — we render
/// the corail-soft "Reçu antérieur" notice in place of the detail
/// cards. The PDF download remains available since the backend can
/// still render a redacted template for legacy rows.
class ReceiptDetailScreen extends ConsumerWidget {
  const ReceiptDetailScreen({super.key, required this.receiptId});

  final String receiptId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final asyncReceipt = ref.watch(receiptDetailProvider(receiptId));

    return Scaffold(
      backgroundColor: colorScheme.surface,
      appBar: AppBar(
        backgroundColor: colorScheme.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: Text(
          'Reçu',
          style: SoleilTextStyles.titleMedium.copyWith(
            color: colorScheme.onSurface,
          ),
        ),
      ),
      body: SafeArea(
        child: asyncReceipt.when(
          data: (receipt) => _ReceiptDetailBody(receipt: receipt),
          loading: () => Center(
            child: CircularProgressIndicator(color: colorScheme.primary),
          ),
          error: (_, __) => _DetailErrorState(
            onRetry: () => ref.invalidate(receiptDetailProvider(receiptId)),
          ),
        ),
      ),
    );
  }
}

class _ReceiptDetailBody extends ConsumerWidget {
  const _ReceiptDetailBody({required this.receipt});

  final Receipt receipt;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _SummaryCard(receipt: receipt, appColors: appColors),
          const SizedBox(height: 16),
          if (!receipt.snapshotAvailable)
            _LegacyNotice(appColors: appColors, colorScheme: colorScheme)
          else ...[
            if (receipt.client != null) ...[
              _PartyCard(title: 'Client', party: receipt.client!),
              const SizedBox(height: 12),
            ],
            if (receipt.provider != null) ...[
              _PartyCard(title: 'Prestataire', party: receipt.provider!),
              const SizedBox(height: 12),
            ],
            if (receipt.referrer != null) ...[
              _PartyCard(
                title: 'Apporteur',
                party: receipt.referrer!,
                commissionAmountCents:
                    receipt.referrerCommissionAmountCents,
                currency: receipt.currency,
              ),
              const SizedBox(height: 12),
            ],
          ],
          const SizedBox(height: 8),
          _DownloadPdfButton(receipt: receipt),
        ],
      ),
    );
  }
}

class _SummaryCard extends StatelessWidget {
  const _SummaryCard({required this.receipt, required this.appColors});

  final Receipt receipt;
  final AppColors? appColors;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'MONTANT',
            style: SoleilTextStyles.mono.copyWith(
              color: appColors?.subtleForeground ??
                  colorScheme.onSurfaceVariant,
              fontSize: 11,
              letterSpacing: 1.4,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            formatReceiptAmount(receipt.amountCents, receipt.currency),
            style: SoleilTextStyles.headlineLarge.copyWith(
              color: colorScheme.onSurface,
              fontFamily: 'GeistMono',
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 12),
          Text(
            'Reçu émis le ${_formatLongDate(receipt.createdAt)}',
            style: SoleilTextStyles.body.copyWith(
              color: appColors?.subtleForeground ??
                  colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}

class _PartyCard extends StatelessWidget {
  const _PartyCard({
    required this.title,
    required this.party,
    this.commissionAmountCents,
    this.currency,
  });

  final String title;
  final ReceiptParty party;
  final int? commissionAmountCents;
  final String? currency;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title.toUpperCase(),
            style: SoleilTextStyles.mono.copyWith(
              color: colorScheme.primary,
              fontSize: 11,
              letterSpacing: 1.4,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 8),
          _LineValue(value: party.name, emphasis: true),
          if (party.siret.isNotEmpty)
            _LineLabelValue(label: 'SIRET', value: party.siret),
          if (party.vat.isNotEmpty)
            _LineLabelValue(label: 'TVA', value: party.vat),
          if (party.addressLine1.isNotEmpty)
            _LineValue(value: party.addressLine1),
          if (party.addressLine2.isNotEmpty)
            _LineValue(value: party.addressLine2),
          if (_cityLine(party).isNotEmpty)
            _LineValue(value: _cityLine(party)),
          if (party.country.isNotEmpty) _LineValue(value: party.country),
          if (commissionAmountCents != null && commissionAmountCents! > 0) ...[
            const SizedBox(height: 8),
            _LineLabelValue(
              label: 'Commission',
              value: formatReceiptAmount(
                commissionAmountCents!,
                currency ?? 'eur',
              ),
            ),
          ],
        ],
      ),
    );
  }
}

String _cityLine(ReceiptParty party) {
  final hasPostal = party.postalCode.isNotEmpty;
  final hasCity = party.city.isNotEmpty;
  if (!hasPostal && !hasCity) return '';
  if (!hasCity) return party.postalCode;
  if (!hasPostal) return party.city;
  return '${party.postalCode} ${party.city}';
}

class _LineValue extends StatelessWidget {
  const _LineValue({required this.value, this.emphasis = false});

  final String value;
  final bool emphasis;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Padding(
      padding: const EdgeInsets.only(top: 2),
      child: Text(
        value,
        style: emphasis
            ? SoleilTextStyles.bodyEmphasis.copyWith(
                color: colorScheme.onSurface,
              )
            : SoleilTextStyles.body.copyWith(
                color: colorScheme.onSurface,
              ),
      ),
    );
  }
}

class _LineLabelValue extends StatelessWidget {
  const _LineLabelValue({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Padding(
      padding: const EdgeInsets.only(top: 2),
      child: RichText(
        text: TextSpan(
          children: [
            TextSpan(
              text: '$label · ',
              style: SoleilTextStyles.mono.copyWith(
                color: appColors?.subtleForeground ??
                    colorScheme.onSurfaceVariant,
                fontSize: 12,
                letterSpacing: 0.6,
              ),
            ),
            TextSpan(
              text: value,
              style: SoleilTextStyles.body.copyWith(
                color: colorScheme.onSurface,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _LegacyNotice extends StatelessWidget {
  const _LegacyNotice({required this.appColors, required this.colorScheme});

  final AppColors? appColors;
  final ColorScheme colorScheme;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: appColors?.accentSoft ?? colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.info_outline_rounded,
            size: 18,
            color: colorScheme.primary,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              'Ce reçu est antérieur à la fonctionnalité de snapshot — '
              'les détails de facturation des parties ne sont pas '
              'disponibles. Le PDF reste téléchargeable.',
              style: SoleilTextStyles.body.copyWith(
                color: colorScheme.onSurface,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _DownloadPdfButton extends ConsumerWidget {
  const _DownloadPdfButton({required this.receipt});

  final Receipt receipt;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return FilledButton.icon(
      onPressed: () => _downloadPdf(context, ref),
      icon: const Icon(Icons.download_rounded, size: 18),
      label: const Text('Télécharger le PDF'),
      style: FilledButton.styleFrom(
        minimumSize: const Size(double.infinity, 48),
        backgroundColor: colorScheme.primary,
        foregroundColor: colorScheme.onPrimary,
        shape: const StadiumBorder(),
        textStyle: SoleilTextStyles.button,
      ),
    );
  }

  Future<void> _downloadPdf(BuildContext context, WidgetRef ref) async {
    final repo = ref.read(receiptRepositoryProvider);
    final messenger = ScaffoldMessenger.maybeOf(context);
    messenger?.showSnackBar(
      const SnackBar(
        content: Text('Téléchargement du reçu…'),
        duration: Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
    try {
      await _shareReceiptPdf(repo, receipt);
      if (!context.mounted) return;
    } catch (e) {
      debugPrint('receipt pdf download failed: $e');
      if (!context.mounted) return;
      messenger?.showSnackBar(
        const SnackBar(content: Text('Impossible de télécharger le PDF.')),
      );
    }
  }
}

Future<void> _shareReceiptPdf(
  ReceiptRepository repo,
  Receipt receipt,
) async {
  final bytes = await repo.downloadPdfBytes(receipt.id);
  final dir = await getTemporaryDirectory();
  final filename = 'recu-${receipt.id}.pdf';
  final path = '${dir.path}/$filename';
  final file = File(path);
  await file.writeAsBytes(bytes, flush: true);
  await Share.shareXFiles(
    [XFile(path, mimeType: 'application/pdf', name: filename)],
    subject: 'Reçu ${receipt.id}',
  );
}

class _DetailErrorState extends StatelessWidget {
  const _DetailErrorState({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(
            'Impossible de charger ce reçu.',
            textAlign: TextAlign.center,
            style: SoleilTextStyles.bodyLarge.copyWith(
              color: colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 16),
          OutlinedButton(
            onPressed: onRetry,
            style: OutlinedButton.styleFrom(
              minimumSize: const Size(double.infinity, 44),
              foregroundColor: colorScheme.onSurface,
              side: BorderSide(
                color: appColors?.borderStrong ?? theme.dividerColor,
              ),
              shape: const StadiumBorder(),
              textStyle: SoleilTextStyles.button,
            ),
            child: const Text('Réessayer'),
          ),
        ],
      ),
    );
  }
}

String _formatLongDate(DateTime d) {
  final months = [
    'janvier',
    'février',
    'mars',
    'avril',
    'mai',
    'juin',
    'juillet',
    'août',
    'septembre',
    'octobre',
    'novembre',
    'décembre',
  ];
  final monthIdx = d.month - 1;
  final month = (monthIdx >= 0 && monthIdx < months.length)
      ? months[monthIdx]
      : d.month.toString();
  return '${d.day} $month ${d.year}';
}
