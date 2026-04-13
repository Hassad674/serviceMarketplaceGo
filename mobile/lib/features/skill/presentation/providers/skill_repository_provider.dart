import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/skill_repository_impl.dart';
import '../../domain/repositories/skill_repository.dart';

/// Dependency-injection seam for the skill feature.
///
/// Presentation code depends on the abstract [SkillRepository] via
/// this Provider. The concrete [SkillRepositoryImpl] is wired here
/// and nowhere else — swapping the backend (e.g. to a local mock in
/// tests) is a one-line override of this provider.
final skillRepositoryProvider = Provider<SkillRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return SkillRepositoryImpl(api);
});
