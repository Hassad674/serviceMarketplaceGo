import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/dispute/domain/entities/dispute_entity.dart';

void main() {
  group('Dispute.fromJson', () {
    final baseJson = {
      'id': 'd-1',
      'proposal_id': 'p-1',
      'conversation_id': 'c-1',
      'initiator_id': 'u-1',
      'respondent_id': 'u-2',
      'client_id': 'u-1',
      'provider_id': 'u-2',
      'reason': 'non_delivery',
      'description': 'nothing delivered',
      'requested_amount': 50000,
      'proposal_amount': 50000,
      'status': 'open',
      'initiator_role': 'client',
      'created_at': '2026-04-01T00:00:00Z',
    };

    test('parses required fields', () {
      final d = Dispute.fromJson(Map<String, dynamic>.from(baseJson));
      expect(d.id, 'd-1');
      expect(d.proposalId, 'p-1');
      expect(d.conversationId, 'c-1');
      expect(d.status, 'open');
      expect(d.initiatorRole, 'client');
      expect(d.requestedAmount, 50000);
    });

    test('description defaults to empty string when missing', () {
      final json = Map<String, dynamic>.from(baseJson)..remove('description');
      final d = Dispute.fromJson(json);
      expect(d.description, '');
    });

    test('handles null resolution fields', () {
      final d = Dispute.fromJson(Map<String, dynamic>.from(baseJson));
      expect(d.resolutionType, isNull);
      expect(d.resolutionAmountClient, isNull);
      expect(d.resolutionAmountProvider, isNull);
      expect(d.resolutionNote, isNull);
    });

    test('parses resolution fields when set', () {
      final json = Map<String, dynamic>.from(baseJson)
        ..['resolution_type'] = 'split'
        ..['resolution_amount_client'] = 25000
        ..['resolution_amount_provider'] = 25000
        ..['resolution_note'] = '50/50'
        ..['resolved_at'] = '2026-04-15T00:00:00Z';
      final d = Dispute.fromJson(json);
      expect(d.resolutionType, 'split');
      expect(d.resolutionAmountClient, 25000);
      expect(d.resolutionAmountProvider, 25000);
      expect(d.resolutionNote, '50/50');
      expect(d.resolvedAt, '2026-04-15T00:00:00Z');
    });

    test('parses cancellation request fields', () {
      final json = Map<String, dynamic>.from(baseJson)
        ..['cancellation_requested_by'] = 'u-1'
        ..['cancellation_requested_at'] = '2026-04-15T00:00:00Z';
      final d = Dispute.fromJson(json);
      expect(d.cancellationRequestedBy, 'u-1');
      expect(d.cancellationRequestedAt, '2026-04-15T00:00:00Z');
    });

    test('evidence list defaults to empty when key absent', () {
      final d = Dispute.fromJson(Map<String, dynamic>.from(baseJson));
      expect(d.evidence, isEmpty);
    });

    test('parses an evidence list', () {
      final json = Map<String, dynamic>.from(baseJson)
        ..['evidence'] = [
          {
            'id': 'e-1',
            'filename': 'doc.pdf',
            'url': 'https://x/d.pdf',
            'size': 1024,
            'mime_type': 'application/pdf',
          },
        ];
      final d = Dispute.fromJson(json);
      expect(d.evidence.length, 1);
      expect(d.evidence.first.filename, 'doc.pdf');
      expect(d.evidence.first.size, 1024);
    });

    test('counter proposals default to empty', () {
      final d = Dispute.fromJson(Map<String, dynamic>.from(baseJson));
      expect(d.counterProposals, isEmpty);
    });

    test('parses counter proposals', () {
      final json = Map<String, dynamic>.from(baseJson)
        ..['counter_proposals'] = [
          {
            'id': 'cp-1',
            'proposer_id': 'u-1',
            'amount_client': 30000,
            'amount_provider': 20000,
            'message': 'split',
            'status': 'pending',
            'created_at': '2026-04-01T00:00:00Z',
          },
        ];
      final d = Dispute.fromJson(json);
      expect(d.counterProposals.length, 1);
      expect(d.counterProposals.first.proposerId, 'u-1');
      expect(d.counterProposals.first.amountClient, 30000);
    });
  });

  group('DisputeEvidence', () {
    test('parses correctly', () {
      final e = DisputeEvidence.fromJson({
        'id': 'e-1',
        'filename': 'x.pdf',
        'url': 'https://x',
        'size': 100,
        'mime_type': 'application/pdf',
      });
      expect(e.id, 'e-1');
      expect(e.filename, 'x.pdf');
      expect(e.size, 100);
      expect(e.mimeType, 'application/pdf');
    });
  });

  group('CounterProposal', () {
    test('parses required fields', () {
      final cp = CounterProposal.fromJson({
        'id': 'cp-1',
        'proposer_id': 'u-1',
        'amount_client': 100,
        'amount_provider': 200,
        'status': 'pending',
        'created_at': '2026-04-01T00:00:00Z',
      });
      expect(cp.id, 'cp-1');
      expect(cp.amountClient, 100);
      expect(cp.amountProvider, 200);
      expect(cp.status, 'pending');
      expect(cp.message, '');
    });

    test('parses message when present', () {
      final cp = CounterProposal.fromJson({
        'id': 'cp-1',
        'proposer_id': 'u-1',
        'amount_client': 100,
        'amount_provider': 200,
        'message': 'split it',
        'status': 'pending',
        'created_at': '2026-04-01T00:00:00Z',
      });
      expect(cp.message, 'split it');
    });

    test('parses respondedAt when present', () {
      final cp = CounterProposal.fromJson({
        'id': 'cp-1',
        'proposer_id': 'u-1',
        'amount_client': 100,
        'amount_provider': 200,
        'status': 'accepted',
        'responded_at': '2026-04-15T00:00:00Z',
        'created_at': '2026-04-01T00:00:00Z',
      });
      expect(cp.respondedAt, '2026-04-15T00:00:00Z');
    });
  });
}
