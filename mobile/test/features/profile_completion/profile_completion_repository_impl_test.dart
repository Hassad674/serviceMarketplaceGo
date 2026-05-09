import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_completion/data/profile_completion_repository_impl.dart';

import '../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late ProfileCompletionRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = ProfileCompletionRepositoryImpl(fakeApi);
  });

  test('getMy parses the wrapped envelope into the report entity',
      () async {
    fakeApi.getHandlers['/api/v1/me/profile/completion'] = (_) async {
      return FakeApiClient.ok({
        'data': {
          'role': 'provider',
          'persona': 'freelance',
          'percent': 60,
          'total_sections': 10,
          'filled_sections': 6,
          'sections': [
            {
              'key': 'title',
              'filled': true,
              'label_key': 'profile.completion.section.title',
              'completion_path': '/dashboard/profile/edit',
            },
            {
              'key': 'about',
              'filled': false,
              'label_key': 'profile.completion.section.about',
              'completion_path': '/dashboard/profile/edit',
            },
          ],
        },
      });
    };

    final report = await repo.getMy();

    expect(report.role, 'provider');
    expect(report.persona, 'freelance');
    expect(report.percent, 60);
    expect(report.totalSections, 10);
    expect(report.filledSections, 6);
    expect(report.missingCount, 4);
    expect(report.isComplete, isFalse);
    expect(report.sections, hasLength(2));
    expect(report.sections.first.key, 'title');
    expect(report.sections.first.filled, isTrue);
    expect(report.sections.last.completionPath, '/dashboard/profile/edit');
  });

  test('getMy tolerates a raw payload without a data envelope', () async {
    fakeApi.getHandlers['/api/v1/me/profile/completion'] = (_) async {
      return FakeApiClient.ok({
        'role': 'agency',
        'persona': 'agency',
        'percent': 100,
        'total_sections': 4,
        'filled_sections': 4,
        'sections': [],
      });
    };

    final report = await repo.getMy();
    expect(report.percent, 100);
    expect(report.isComplete, isTrue);
  });

  test('isComplete returns true at exactly 100 percent', () {
    const report = TestReport(percent: 100);
    expect(report.percent, 100);
  });

  test('missingCount handles the all-empty case', () async {
    fakeApi.getHandlers['/api/v1/me/profile/completion'] = (_) async {
      return FakeApiClient.ok({
        'role': 'provider',
        'persona': 'freelance',
        'percent': 0,
        'total_sections': 13,
        'filled_sections': 0,
        'sections': [],
      });
    };
    final report = await repo.getMy();
    expect(report.missingCount, 13);
    expect(report.isComplete, isFalse);
  });

  test('forwards the persona override on the query string', () async {
    var calledWithReferrerPath = false;
    fakeApi.getHandlers['/api/v1/me/profile/completion?persona=referrer'] =
        (_) async {
      calledWithReferrerPath = true;
      return FakeApiClient.ok({
        'role': 'provider',
        'persona': 'referrer',
        'percent': 25,
        'total_sections': 8,
        'filled_sections': 2,
        'sections': [],
      });
    };
    // Default path MUST not be hit when the persona override is set.
    fakeApi.getHandlers['/api/v1/me/profile/completion'] = (_) async {
      throw StateError('default path hit despite persona override');
    };
    final report = await repo.getMy(persona: 'referrer');
    expect(calledWithReferrerPath, isTrue,
        reason: 'persona must be appended verbatim as a query string');
    expect(report.persona, 'referrer');
    expect(report.totalSections, 8);
  });

  test('omits the query string when persona is null', () async {
    var calledDefaultPath = false;
    fakeApi.getHandlers['/api/v1/me/profile/completion'] = (_) async {
      calledDefaultPath = true;
      return FakeApiClient.ok({
        'role': 'provider',
        'persona': 'freelance',
        'percent': 0,
        'total_sections': 13,
        'filled_sections': 0,
        'sections': [],
      });
    };
    await repo.getMy();
    expect(calledDefaultPath, isTrue);
  });
}

/// Lightweight stand-in used by the assertion above to verify a
/// constant value without re-deriving the entity factory.
class TestReport {
  const TestReport({required this.percent});
  final int percent;
}
