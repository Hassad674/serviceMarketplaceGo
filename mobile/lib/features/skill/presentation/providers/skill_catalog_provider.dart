import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/repositories/skill_repository.dart';
import 'skill_repository_provider.dart';

/// Fetches the curated skills catalog for one expertise domain.
///
/// Families are the natural fit here: each expertise key is a
/// separate cache slot, so switching between domain panels in the
/// editor does not trigger a refetch once the data is cached. The
/// family parameter is the raw expertise key (e.g. `development`).
///
/// The UI renders this with `.when(data, loading, error)` — there
/// is no optimistic state because the catalog is read-only.
final skillCatalogProvider =
    FutureProvider.family<SkillCatalogPage, String>((ref, expertiseKey) async {
  final repo = ref.watch(skillRepositoryProvider);
  return repo.fetchCatalog(expertiseKey);
});
