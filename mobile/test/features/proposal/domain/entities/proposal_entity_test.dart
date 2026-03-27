import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/proposal/domain/entities/proposal_entity.dart';

void main() {
  group('ProposalEntity', () {
    test('creates with all required fields', () {
      const proposal = ProposalEntity(
        id: 'proposal-1',
        conversationId: 'conv-1',
        senderId: 'user-1',
        recipientId: 'user-2',
        title: 'Website redesign',
        description: 'Full redesign of the corporate website',
        amount: 500000,
        status: 'pending',
        version: 1,
        clientId: 'user-2',
        providerId: 'user-1',
        createdAt: '2026-03-27T10:00:00Z',
      );

      expect(proposal.id, 'proposal-1');
      expect(proposal.conversationId, 'conv-1');
      expect(proposal.senderId, 'user-1');
      expect(proposal.recipientId, 'user-2');
      expect(proposal.title, 'Website redesign');
      expect(proposal.description, 'Full redesign of the corporate website');
      expect(proposal.amount, 500000);
      expect(proposal.status, 'pending');
      expect(proposal.version, 1);
      expect(proposal.clientId, 'user-2');
      expect(proposal.providerId, 'user-1');
      expect(proposal.createdAt, '2026-03-27T10:00:00Z');
      expect(proposal.deadline, isNull);
      expect(proposal.parentId, isNull);
      expect(proposal.documents, isEmpty);
      expect(proposal.acceptedAt, isNull);
      expect(proposal.paidAt, isNull);
    });

    test('amountInEuros converts centimes to euros', () {
      const proposal = ProposalEntity(
        id: 'p-1',
        conversationId: 'c-1',
        senderId: 's-1',
        recipientId: 'r-1',
        title: 'Test',
        description: 'Test',
        amount: 150050,
        status: 'pending',
        version: 1,
        clientId: 'c-1',
        providerId: 'p-1',
        createdAt: '2026-03-27T10:00:00Z',
      );

      expect(proposal.amountInEuros, 1500.50);
    });

    test('amountInEuros returns 0.0 for zero centimes', () {
      const proposal = ProposalEntity(
        id: 'p-1',
        conversationId: 'c-1',
        senderId: 's-1',
        recipientId: 'r-1',
        title: 'Test',
        description: 'Test',
        amount: 0,
        status: 'pending',
        version: 1,
        clientId: 'c-1',
        providerId: 'p-1',
        createdAt: '2026-03-27T10:00:00Z',
      );

      expect(proposal.amountInEuros, 0.0);
    });

    test('amountInEuros handles small amounts correctly', () {
      const proposal = ProposalEntity(
        id: 'p-1',
        conversationId: 'c-1',
        senderId: 's-1',
        recipientId: 'r-1',
        title: 'Test',
        description: 'Test',
        amount: 99,
        status: 'pending',
        version: 1,
        clientId: 'c-1',
        providerId: 'p-1',
        createdAt: '2026-03-27T10:00:00Z',
      );

      expect(proposal.amountInEuros, 0.99);
    });

    test('fromJson parses all fields correctly', () {
      final json = {
        'id': 'proposal-10',
        'conversation_id': 'conv-5',
        'sender_id': 'user-1',
        'recipient_id': 'user-2',
        'title': 'Mobile app development',
        'description': 'Build a Flutter mobile app',
        'amount': 800000,
        'deadline': '2026-06-15T00:00:00Z',
        'status': 'accepted',
        'parent_id': 'proposal-9',
        'version': 2,
        'client_id': 'user-2',
        'provider_id': 'user-1',
        'documents': [
          {
            'id': 'doc-1',
            'filename': 'spec.pdf',
            'url': 'https://storage.example.com/spec.pdf',
            'size': 2048,
            'mime_type': 'application/pdf',
          },
        ],
        'accepted_at': '2026-03-28T12:00:00Z',
        'paid_at': null,
        'created_at': '2026-03-27T10:00:00Z',
      };

      final proposal = ProposalEntity.fromJson(json);

      expect(proposal.id, 'proposal-10');
      expect(proposal.conversationId, 'conv-5');
      expect(proposal.senderId, 'user-1');
      expect(proposal.recipientId, 'user-2');
      expect(proposal.title, 'Mobile app development');
      expect(proposal.description, 'Build a Flutter mobile app');
      expect(proposal.amount, 800000);
      expect(proposal.deadline, '2026-06-15T00:00:00Z');
      expect(proposal.status, 'accepted');
      expect(proposal.parentId, 'proposal-9');
      expect(proposal.version, 2);
      expect(proposal.clientId, 'user-2');
      expect(proposal.providerId, 'user-1');
      expect(proposal.documents.length, 1);
      expect(proposal.documents[0].filename, 'spec.pdf');
      expect(proposal.acceptedAt, '2026-03-28T12:00:00Z');
      expect(proposal.paidAt, isNull);
      expect(proposal.createdAt, '2026-03-27T10:00:00Z');
    });

    test('fromJson handles missing optional fields', () {
      final json = {
        'id': 'proposal-11',
        'conversation_id': 'conv-5',
        'sender_id': 'user-1',
        'recipient_id': 'user-2',
        'title': 'Quick task',
        'description': 'A simple task',
        'amount': 10000,
        'status': 'pending',
        'client_id': 'user-2',
        'provider_id': 'user-1',
        'created_at': '2026-03-27T10:00:00Z',
      };

      final proposal = ProposalEntity.fromJson(json);

      expect(proposal.id, 'proposal-11');
      expect(proposal.deadline, isNull);
      expect(proposal.parentId, isNull);
      expect(proposal.version, 1);
      expect(proposal.documents, isEmpty);
      expect(proposal.acceptedAt, isNull);
      expect(proposal.paidAt, isNull);
    });

    test('fromJson handles null documents list', () {
      final json = {
        'id': 'proposal-12',
        'conversation_id': 'conv-5',
        'sender_id': 'user-1',
        'recipient_id': 'user-2',
        'title': 'No docs',
        'description': 'No documents attached',
        'amount': 5000,
        'status': 'pending',
        'client_id': 'user-2',
        'provider_id': 'user-1',
        'documents': null,
        'created_at': '2026-03-27T10:00:00Z',
      };

      final proposal = ProposalEntity.fromJson(json);

      expect(proposal.documents, isEmpty);
    });

    test('fromJson parses amount as num (int)', () {
      final json = {
        'id': 'p-1',
        'conversation_id': 'c-1',
        'sender_id': 's-1',
        'recipient_id': 'r-1',
        'title': 'Test',
        'description': 'Test',
        'amount': 150000,
        'status': 'pending',
        'client_id': 'c-1',
        'provider_id': 'p-1',
        'created_at': '2026-03-27T10:00:00Z',
      };

      final proposal = ProposalEntity.fromJson(json);

      expect(proposal.amount, 150000);
      expect(proposal.amountInEuros, 1500.0);
    });

    test('fromJson parses amount as num (double)', () {
      final json = {
        'id': 'p-1',
        'conversation_id': 'c-1',
        'sender_id': 's-1',
        'recipient_id': 'r-1',
        'title': 'Test',
        'description': 'Test',
        'amount': 150000.0,
        'status': 'pending',
        'client_id': 'c-1',
        'provider_id': 'p-1',
        'created_at': '2026-03-27T10:00:00Z',
      };

      final proposal = ProposalEntity.fromJson(json);

      expect(proposal.amount, 150000);
    });
  });

  group('ProposalDocumentEntity', () {
    test('creates with all required fields', () {
      const doc = ProposalDocumentEntity(
        id: 'doc-1',
        filename: 'contract.pdf',
        url: 'https://storage.example.com/contract.pdf',
        size: 4096,
        mimeType: 'application/pdf',
      );

      expect(doc.id, 'doc-1');
      expect(doc.filename, 'contract.pdf');
      expect(doc.url, 'https://storage.example.com/contract.pdf');
      expect(doc.size, 4096);
      expect(doc.mimeType, 'application/pdf');
    });

    test('fromJson parses all fields correctly', () {
      final json = {
        'id': 'doc-2',
        'filename': 'specs.xlsx',
        'url': 'https://storage.example.com/specs.xlsx',
        'size': 8192,
        'mime_type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      };

      final doc = ProposalDocumentEntity.fromJson(json);

      expect(doc.id, 'doc-2');
      expect(doc.filename, 'specs.xlsx');
      expect(doc.url, 'https://storage.example.com/specs.xlsx');
      expect(doc.size, 8192);
      expect(doc.mimeType, 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet');
    });

    test('fromJson handles size as double', () {
      final json = {
        'id': 'doc-3',
        'filename': 'large.zip',
        'url': 'https://storage.example.com/large.zip',
        'size': 1048576.0,
        'mime_type': 'application/zip',
      };

      final doc = ProposalDocumentEntity.fromJson(json);

      expect(doc.size, 1048576);
    });
  });

  group('ProposalEntity status values', () {
    final statuses = [
      'pending',
      'accepted',
      'declined',
      'withdrawn',
      'paid',
      'active',
      'completed',
    ];

    for (final status in statuses) {
      test('fromJson parses status "$status"', () {
        final json = {
          'id': 'p-1',
          'conversation_id': 'c-1',
          'sender_id': 's-1',
          'recipient_id': 'r-1',
          'title': 'Test',
          'description': 'Test',
          'amount': 1000,
          'status': status,
          'client_id': 'c-1',
          'provider_id': 'p-1',
          'created_at': '2026-03-27T10:00:00Z',
        };

        final proposal = ProposalEntity.fromJson(json);

        expect(proposal.status, status);
      });
    }
  });

  group('ProposalEntity with multiple documents', () {
    test('fromJson parses multiple documents', () {
      final json = {
        'id': 'p-1',
        'conversation_id': 'c-1',
        'sender_id': 's-1',
        'recipient_id': 'r-1',
        'title': 'Test',
        'description': 'Test',
        'amount': 1000,
        'status': 'pending',
        'client_id': 'c-1',
        'provider_id': 'p-1',
        'documents': [
          {
            'id': 'doc-1',
            'filename': 'a.pdf',
            'url': 'https://example.com/a.pdf',
            'size': 1024,
            'mime_type': 'application/pdf',
          },
          {
            'id': 'doc-2',
            'filename': 'b.png',
            'url': 'https://example.com/b.png',
            'size': 2048,
            'mime_type': 'image/png',
          },
          {
            'id': 'doc-3',
            'filename': 'c.docx',
            'url': 'https://example.com/c.docx',
            'size': 4096,
            'mime_type': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
          },
        ],
        'created_at': '2026-03-27T10:00:00Z',
      };

      final proposal = ProposalEntity.fromJson(json);

      expect(proposal.documents.length, 3);
      expect(proposal.documents[0].filename, 'a.pdf');
      expect(proposal.documents[1].filename, 'b.png');
      expect(proposal.documents[2].filename, 'c.docx');
    });
  });
}
