import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/proposal/data/proposal_repository_impl.dart';
import 'package:marketplace_mobile/features/proposal/domain/repositories/proposal_repository.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ProposalRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ProposalRepositoryImpl(apiClient: fakeApi);
  });

  final sampleProposal = {
    'id': 'prop-1',
    'conversation_id': 'conv-1',
    'sender_id': 'user-1',
    'recipient_id': 'user-2',
    'title': 'Web Development',
    'description': 'Build a website',
    'amount': 5000,
    'status': 'pending',
    'version': 1,
    'client_id': 'user-2',
    'provider_id': 'user-1',
    'documents': <dynamic>[],
    'created_at': '2026-03-27T10:00:00Z',
  };

  group('ProposalRepositoryImpl.createProposal', () {
    test('sends correct body and returns proposal', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/proposals'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': sampleProposal});
      };

      final result = await repo.createProposal(CreateProposalData(
        recipientId: 'user-2',
        conversationId: 'conv-1',
        title: 'Web Development',
        description: 'Build a website',
        amount: 5000,
      ));

      expect(result.id, 'prop-1');
      expect(result.title, 'Web Development');
      expect(capturedBody!['recipient_id'], 'user-2');
      expect(capturedBody!['conversation_id'], 'conv-1');
      expect(capturedBody!['amount'], 5000);
      expect(capturedBody!.containsKey('deadline'), false);
    });

    test('includes deadline when provided', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/proposals'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': sampleProposal});
      };

      await repo.createProposal(CreateProposalData(
        recipientId: 'user-2',
        conversationId: 'conv-1',
        title: 'Test',
        description: 'Desc',
        amount: 1000,
        deadline: '2026-04-15',
      ));

      expect(capturedBody!['deadline'], '2026-04-15');
    });
  });

  group('ProposalRepositoryImpl.getProposal', () {
    test('returns proposal from wrapped response', () async {
      fakeApi.getHandlers['/api/v1/proposals/prop-1'] = (_) async {
        return FakeApiClient.ok({'data': sampleProposal});
      };

      final result = await repo.getProposal('prop-1');

      expect(result.id, 'prop-1');
      expect(result.amount, 5000);
    });

    test('returns proposal from flat response', () async {
      fakeApi.getHandlers['/api/v1/proposals/prop-1'] = (_) async {
        return FakeApiClient.ok(sampleProposal);
      };

      final result = await repo.getProposal('prop-1');

      expect(result.id, 'prop-1');
    });
  });

  group('ProposalRepositoryImpl.acceptProposal', () {
    test('calls correct endpoint', () async {
      var called = false;

      fakeApi.postHandlers['/api/v1/proposals/prop-1/accept'] = (_) async {
        called = true;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.acceptProposal('prop-1');

      expect(called, true);
    });
  });

  group('ProposalRepositoryImpl.declineProposal', () {
    test('calls correct endpoint', () async {
      var called = false;

      fakeApi.postHandlers['/api/v1/proposals/prop-1/decline'] = (_) async {
        called = true;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.declineProposal('prop-1');

      expect(called, true);
    });
  });

  group('ProposalRepositoryImpl.modifyProposal', () {
    test('sends modified data', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/proposals/prop-1/modify'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': sampleProposal});
      };

      await repo.modifyProposal(
        'prop-1',
        ModifyProposalData(
          title: 'Updated Title',
          description: 'Updated desc',
          amount: 7000,
        ),
      );

      expect(capturedBody!['title'], 'Updated Title');
      expect(capturedBody!['amount'], 7000);
      expect(capturedBody!.containsKey('deadline'), false);
    });

    test('includes deadline in modification', () async {
      Map<String, dynamic>? capturedBody;

      fakeApi.postHandlers['/api/v1/proposals/prop-1/modify'] = (data) async {
        capturedBody = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': sampleProposal});
      };

      await repo.modifyProposal(
        'prop-1',
        ModifyProposalData(
          title: 'T',
          description: 'D',
          amount: 100,
          deadline: '2026-05-01',
        ),
      );

      expect(capturedBody!['deadline'], '2026-05-01');
    });
  });

  group('ProposalRepositoryImpl.simulatePayment', () {
    test('calls correct endpoint', () async {
      var called = false;

      fakeApi.postHandlers['/api/v1/proposals/prop-1/pay'] = (_) async {
        called = true;
        return FakeApiClient.ok({'status': 'ok'});
      };

      await repo.simulatePayment('prop-1');

      expect(called, true);
    });
  });

  group('ProposalRepositoryImpl.listProjects', () {
    test('returns list from data array', () async {
      fakeApi.getHandlers['/api/v1/projects'] = (_) async {
        return FakeApiClient.ok({
          'data': [sampleProposal],
          'next_cursor': null,
          'has_more': false,
        });
      };

      final result = await repo.listProjects();

      expect(result.length, 1);
      expect(result[0].id, 'prop-1');
    });

    test('returns empty list when no data key', () async {
      fakeApi.getHandlers['/api/v1/projects'] = (_) async {
        return FakeApiClient.ok({'status': 'ok'});
      };

      final result = await repo.listProjects();

      expect(result, isEmpty);
    });

    test('returns empty list when data is not a list', () async {
      fakeApi.getHandlers['/api/v1/projects'] = (_) async {
        return FakeApiClient.ok({'data': 'not-a-list'});
      };

      final result = await repo.listProjects();

      expect(result, isEmpty);
    });
  });
}
