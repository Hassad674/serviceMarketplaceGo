import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/identity_document_entity.dart';

void main() {
  group('IdentityDocument.fromJson', () {
    test('parses complete JSON with all fields', () {
      final json = <String, dynamic>{
        'id': 'doc-123',
        'user_id': 'user-456',
        'category': 'identity',
        'document_type': 'passport',
        'side': 'single',
        'file_url': 'https://storage.example.com/docs/passport.jpg',
        'status': 'verified',
        'rejection_reason': '',
        'created_at': '2026-03-01T10:00:00Z',
        'updated_at': '2026-03-02T15:00:00Z',
      };

      final doc = IdentityDocument.fromJson(json);

      expect(doc.id, 'doc-123');
      expect(doc.userId, 'user-456');
      expect(doc.category, 'identity');
      expect(doc.documentType, 'passport');
      expect(doc.side, 'single');
      expect(doc.fileUrl, 'https://storage.example.com/docs/passport.jpg');
      expect(doc.status, 'verified');
      expect(doc.rejectionReason, '');
      expect(doc.createdAt, DateTime.parse('2026-03-01T10:00:00Z'));
      expect(doc.updatedAt, DateTime.parse('2026-03-02T15:00:00Z'));
    });

    test('uses defaults for missing optional fields', () {
      final json = <String, dynamic>{
        'id': 'doc-minimal',
        'user_id': 'user-789',
        'document_type': 'id_card',
        'created_at': '2026-03-10T00:00:00Z',
        'updated_at': '2026-03-10T00:00:00Z',
      };

      final doc = IdentityDocument.fromJson(json);

      expect(doc.category, 'identity');
      expect(doc.side, 'front');
      expect(doc.fileUrl, '');
      expect(doc.status, 'pending');
      expect(doc.rejectionReason, '');
    });

    test('parses rejected document with reason', () {
      final json = <String, dynamic>{
        'id': 'doc-rejected',
        'user_id': 'user-111',
        'document_type': 'driving_license',
        'status': 'rejected',
        'rejection_reason': 'Document is blurry and unreadable',
        'created_at': '2026-03-05T08:00:00Z',
        'updated_at': '2026-03-06T09:00:00Z',
      };

      final doc = IdentityDocument.fromJson(json);

      expect(doc.status, 'rejected');
      expect(doc.rejectionReason, 'Document is blurry and unreadable');
    });
  });

  group('IdentityDocument status helpers', () {
    IdentityDocument buildDoc({required String status}) {
      return IdentityDocument(
        id: 'test-id',
        userId: 'test-user',
        documentType: 'passport',
        createdAt: DateTime.now(),
        updatedAt: DateTime.now(),
        status: status,
      );
    }

    test('isPending returns true only for pending status', () {
      expect(buildDoc(status: 'pending').isPending, true);
      expect(buildDoc(status: 'verified').isPending, false);
      expect(buildDoc(status: 'rejected').isPending, false);
    });

    test('isVerified returns true only for verified status', () {
      expect(buildDoc(status: 'verified').isVerified, true);
      expect(buildDoc(status: 'pending').isVerified, false);
      expect(buildDoc(status: 'rejected').isVerified, false);
    });

    test('isRejected returns true only for rejected status', () {
      expect(buildDoc(status: 'rejected').isRejected, true);
      expect(buildDoc(status: 'pending').isRejected, false);
      expect(buildDoc(status: 'verified').isRejected, false);
    });

    test('unknown status returns false for all helpers', () {
      final doc = buildDoc(status: 'unknown');
      expect(doc.isPending, false);
      expect(doc.isVerified, false);
      expect(doc.isRejected, false);
    });
  });
}
