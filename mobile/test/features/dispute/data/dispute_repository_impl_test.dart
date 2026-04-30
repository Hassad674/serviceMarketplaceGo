import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/dispute/data/dispute_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late DisputeRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = DisputeRepositoryImpl(apiClient: fakeApi);
  });

  Map<String, dynamic> sampleDispute() => {
        'id': 'd-1',
        'proposal_id': 'p-1',
        'conversation_id': 'c-1',
        'initiator_id': 'u-1',
        'respondent_id': 'u-2',
        'client_id': 'u-1',
        'provider_id': 'u-2',
        'reason': 'non_delivery',
        'description': '',
        'requested_amount': 50000,
        'proposal_amount': 50000,
        'status': 'open',
        'initiator_role': 'client',
        'created_at': '2026-04-01T00:00:00Z',
      };

  group('openDispute', () {
    test('POSTs the body and parses the response', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/disputes'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok(sampleDispute());
      };

      final dispute = await repo.openDispute(
        proposalId: 'p-1',
        reason: 'non_delivery',
        description: '',
        messageToParty: 'please respond',
        requestedAmount: 50000,
      );

      expect(dispute.id, 'd-1');
      expect(captured!['proposal_id'], 'p-1');
      expect(captured!['reason'], 'non_delivery');
      expect(captured!['message_to_party'], 'please respond');
      expect(captured!['attachments'], isEmpty);
    });

    test('passes attachments when provided', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/disputes'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok(sampleDispute());
      };

      await repo.openDispute(
        proposalId: 'p-1',
        reason: 'harassment',
        description: 'desc',
        messageToParty: 'stop',
        requestedAmount: 10000,
        attachments: [
          {'filename': 'x.pdf', 'url': 'https://x', 'size': 100, 'mime_type': 'application/pdf'},
        ],
      );

      expect((captured!['attachments'] as List).length, 1);
    });
  });

  group('getDispute', () {
    test('GETs and parses the dispute', () async {
      fakeApi.getHandlers['/api/v1/disputes/d-1'] = (params) async {
        return FakeApiClient.ok(sampleDispute());
      };
      final d = await repo.getDispute('d-1');
      expect(d.id, 'd-1');
    });
  });

  group('counterPropose', () {
    test('POSTs the amounts', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/disputes/d-1/counter-propose'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.counterPropose(
        disputeId: 'd-1',
        amountClient: 30000,
        amountProvider: 20000,
        message: 'split',
      );
      expect(captured!['amount_client'], 30000);
      expect(captured!['amount_provider'], 20000);
      expect(captured!['message'], 'split');
    });

    test('omits message when null', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/disputes/d-1/counter-propose'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.counterPropose(
        disputeId: 'd-1',
        amountClient: 100,
        amountProvider: 200,
      );
      expect(captured!.containsKey('message'), isFalse);
    });
  });

  group('respondToCounter', () {
    test('POSTs accept=true', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/disputes/d-1/counter-proposals/cp-1/respond'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.respondToCounter(
        disputeId: 'd-1',
        counterProposalId: 'cp-1',
        accept: true,
      );
      expect(captured!['accept'], true);
    });

    test('POSTs accept=false', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/disputes/d-1/counter-proposals/cp-1/respond'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.respondToCounter(
        disputeId: 'd-1',
        counterProposalId: 'cp-1',
        accept: false,
      );
      expect(captured!['accept'], false);
    });
  });

  group('cancelDispute', () {
    test('returns the status from the response', () async {
      fakeApi.postHandlers['/api/v1/disputes/d-1/cancel'] = (data) async {
        return FakeApiClient.ok({'status': 'cancellation_requested'});
      };
      final status = await repo.cancelDispute('d-1');
      expect(status, 'cancellation_requested');
    });

    test('falls back to cancelled when status is missing', () async {
      fakeApi.postHandlers['/api/v1/disputes/d-1/cancel'] = (data) async {
        return FakeApiClient.ok({});
      };
      final status = await repo.cancelDispute('d-1');
      expect(status, 'cancelled');
    });
  });

  group('respondToCancellation', () {
    test('POSTs accept=true', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/disputes/d-1/cancellation/respond'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.respondToCancellation(
        disputeId: 'd-1',
        accept: true,
      );
      expect(captured!['accept'], true);
    });

    test('POSTs accept=false', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/disputes/d-1/cancellation/respond'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.respondToCancellation(
        disputeId: 'd-1',
        accept: false,
      );
      expect(captured!['accept'], false);
    });
  });
}
