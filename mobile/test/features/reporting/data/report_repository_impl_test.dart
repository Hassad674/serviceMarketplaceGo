import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/reporting/data/report_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ReportRepository repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ReportRepository(fakeApi);
  });

  group('createReport', () {
    test('POSTs to /api/v1/reports with the body', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/reports'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'id': 'r-1'});
      };

      await repo.createReport(
        targetType: 'user',
        targetId: 'u-1',
        conversationId: 'c-1',
        reason: 'harassment',
        description: 'mean',
      );

      expect(captured!['target_type'], 'user');
      expect(captured!['target_id'], 'u-1');
      expect(captured!['conversation_id'], 'c-1');
      expect(captured!['reason'], 'harassment');
      expect(captured!['description'], 'mean');
    });

    test('handles message reports', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/reports'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.createReport(
        targetType: 'message',
        targetId: 'm-1',
        conversationId: 'c-1',
        reason: 'spam',
        description: '',
      );
      expect(captured!['target_type'], 'message');
      expect(captured!['description'], '');
    });

    test('handles job reports with empty conversation_id', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/reports'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({});
      };
      await repo.createReport(
        targetType: 'job',
        targetId: 'j-1',
        conversationId: '',
        reason: 'fraud_or_scam',
        description: 'scam',
      );
      expect(captured!['conversation_id'], '');
    });
  });
}
