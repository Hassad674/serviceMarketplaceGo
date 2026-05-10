/// Persona key emitted by the search index — `freelance`, `agency`,
/// `referrer`, `client`. Used to decide which public-profile route the
/// search result card should land on.
///
/// Aligned with `SearchDocumentPersona.name` so callers can pass
/// `document.persona.name` directly without an extra mapping.
typedef SearchPersonaKey = String;

/// Builds the public-profile URL for a tapped search result, optionally
/// appending the originating search query and the 1-based result
/// position so the destination page can hydrate breadcrumbs and the
/// backend can attribute the click to a specific keyword/position pair.
///
/// Output examples:
/// * `/freelancers/abc?q=designer&pos=3`
/// * `/profiles/xyz` (no query, persona unmapped)
///
/// Contract:
/// * [query] is lowercased + trimmed before encoding. Empty → omitted.
/// * [position] is 1-based; `< 1` is normalised to `null` (omitted).
/// * Persona mapping: `freelance` → `/freelancers/`, `agency` →
///   `/profiles/` (legacy), `referrer` → `/referrers/`, `client` →
///   `/clients/`. Anything else falls back to `/profiles/`.
///
/// Pure function — no dependency on `BuildContext`, `GoRouter`, or
/// network. The unit test in
/// `test/features/search/profile_route_builder_test.dart` exercises the
/// matrix of inputs without spinning up a widget tree.
String buildProfileRouteFromSearch({
  required String orgId,
  required SearchPersonaKey persona,
  String? query,
  int? position,
}) {
  final base = _basePathForPersona(persona);
  final id = Uri.encodeComponent(orgId);
  final params = <String, String>{};

  final cleanQuery = (query ?? '').trim().toLowerCase();
  if (cleanQuery.isNotEmpty) {
    params['q'] = cleanQuery;
  }
  if (position != null && position >= 1) {
    params['pos'] = position.toString();
  }

  if (params.isEmpty) return '$base$id';

  final encoded = params.entries
      .map((e) => '${e.key}=${Uri.encodeQueryComponent(e.value)}')
      .join('&');
  return '$base$id?$encoded';
}

String _basePathForPersona(SearchPersonaKey persona) {
  switch (persona) {
    case 'freelance':
      return '/freelancers/';
    case 'referrer':
      return '/referrers/';
    case 'client':
      return '/clients/';
    case 'agency':
    default:
      // Agencies still ride the legacy combined `/profiles/:id` route —
      // mirror what `search_result_card.dart` did before the handoff
      // change.
      return '/profiles/';
  }
}
