/// Abstract repository for expertise domain operations.
///
/// Only the "update" direction lives here — reads come directly
/// from the profile endpoint via `profileProvider` /
/// `publicProfileProvider`. Keeping the surface narrow follows the
/// Interface Segregation principle: consumers that only need to
/// save the list do not have to mock a getter too.
abstract class ExpertiseRepository {
  /// Replaces the caller's organization expertise selection with
  /// [domains]. Returns the server-echoed list on success so the UI
  /// can reconcile its optimistic state.
  ///
  /// Throws a [DioException] on network / validation errors; the
  /// presentation layer is responsible for mapping those to
  /// user-friendly messages.
  Future<List<String>> updateExpertise(List<String> domains);
}
