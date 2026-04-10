import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/project_history_repository_impl.dart';
import '../../domain/entities/project_history_entry.dart';
import '../../domain/repositories/project_history_repository.dart';

/// Provides the [ProjectHistoryRepository] instance.
final projectHistoryRepositoryProvider =
    Provider<ProjectHistoryRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return ProjectHistoryRepositoryImpl(api);
});

/// Fetches the project history entries of a provider.
final projectHistoryProvider =
    FutureProvider.family<List<ProjectHistoryEntry>, String>((ref, userId) async {
  final repo = ref.watch(projectHistoryRepositoryProvider);
  return repo.getByProvider(userId);
});
