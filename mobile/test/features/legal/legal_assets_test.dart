import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter_test/flutter_test.dart';

/// Asset bundle regression suite — guards against forgetting to add a
/// `legal/<doc>.md` entry to `pubspec.yaml` under `flutter.assets:` or
/// to delete the markdown file by mistake. Each of the 6 docs must
/// load as a long string (> 1000 chars) so a truncated copy is also
/// caught.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const expectedAssets = [
    'assets/legal/registre.md',
    'assets/legal/aipd.md',
    'assets/legal/dpa-template.md',
    'assets/legal/politique-confidentialite.md',
    'assets/legal/cgu.md',
    'assets/legal/cgv.md',
  ];

  for (final path in expectedAssets) {
    test('$path loads from rootBundle as a long FR document', () async {
      final content = await rootBundle.loadString(path);
      expect(
        content.length,
        greaterThan(1000),
        reason: '$path should be a long-form FR markdown document',
      );
      // Sanity check on the file extension contract — every doc opens
      // with a level-1 markdown heading.
      expect(content.trimLeft().startsWith('# '), isTrue);
    });
  }
}
