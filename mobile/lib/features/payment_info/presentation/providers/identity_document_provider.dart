import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/identity_document_repository_impl.dart';
import '../../domain/entities/identity_document_entity.dart';

/// Provides the [IdentityDocumentRepository] instance.
final identityDocumentRepositoryProvider =
    Provider<IdentityDocumentRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return IdentityDocumentRepositoryImpl(api);
});

/// Fetches the current user's identity documents.
final identityDocumentsProvider =
    FutureProvider<List<IdentityDocument>>((ref) async {
  final repo = ref.watch(identityDocumentRepositoryProvider);
  return repo.listDocuments();
});
