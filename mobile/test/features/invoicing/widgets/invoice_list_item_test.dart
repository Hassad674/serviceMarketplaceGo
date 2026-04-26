import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/invoice.dart';
import 'package:marketplace_mobile/features/invoicing/domain/repositories/invoicing_repository.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/providers/invoicing_providers.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/invoice_list_item.dart';

import '../helpers/invoicing_test_helpers.dart';

void main() {
  testWidgets(
      'renders number + date + motif label + amount and exposes a download button',
      (tester) async {
    final repo = RecordingInvoicingRepository();
    final invoice = buildInvoice(
      id: 'inv_42',
      number: 'INV-2026-0042',
      issuedAt: DateTime.utc(2026, 4, 15),
      sourceType: SourceType.subscription,
      amountInclTaxCents: 1900,
    );

    await tester.pumpWidget(
      wrapInvoicingWidget(
        overrides: [
          invoicingRepositoryProvider
              .overrideWithValue(repo as InvoicingRepository),
        ],
        child: InvoiceListItem(invoice: invoice),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('INV-2026-0042'), findsOneWidget);
    expect(find.textContaining('15/04/2026'), findsOneWidget);
    // Subscription motif label.
    expect(find.textContaining('Abonnement Premium'), findsOneWidget);
    // Amount formatted in EUR with comma separator.
    expect(find.textContaining('19,00'), findsOneWidget);
    // Download icon button visible.
    expect(
      find.byTooltip('Télécharger la facture INV-2026-0042'),
      findsOneWidget,
    );
  });

  testWidgets('tapping download invokes getInvoicePDFURL with the invoice id',
      (tester) async {
    final repo = RecordingInvoicingRepository();
    final invoice = buildInvoice(id: 'inv_99', number: 'INV-2026-0099');

    await tester.pumpWidget(
      wrapInvoicingWidget(
        overrides: [
          invoicingRepositoryProvider
              .overrideWithValue(repo as InvoicingRepository),
        ],
        child: InvoiceListItem(invoice: invoice),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Télécharger la facture INV-2026-0099'));
    // The widget kicks off launchUrl asynchronously — that platform call
    // throws under flutter_test (no plugin), but the snackbar surfaces
    // the failure. The repository call still happens synchronously
    // before launchUrl.
    await tester.pump();

    expect(repo.getInvoicePDFURLCalls, ['inv_99']);
  });

  testWidgets('renders monthly_commission motif label', (tester) async {
    final repo = RecordingInvoicingRepository();
    final invoice = buildInvoice(
      sourceType: SourceType.monthlyCommission,
    );

    await tester.pumpWidget(
      wrapInvoicingWidget(
        overrides: [
          invoicingRepositoryProvider
              .overrideWithValue(repo as InvoicingRepository),
        ],
        child: InvoiceListItem(invoice: invoice),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Commission mensuelle'), findsOneWidget);
  });
}
