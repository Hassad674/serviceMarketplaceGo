import 'package:freezed_annotation/freezed_annotation.dart';

import 'security_event.dart';

part 'security_activity_page.freezed.dart';

/// One paginated page of security events.
///
/// `nextCursor` is null when [data] is the last page — the UI hides
/// the "Voir plus" pill in that case. Cursors are opaque base64
/// strings; the presentation layer must echo them back unchanged
/// when fetching the next page.
@freezed
class SecurityActivityPage with _$SecurityActivityPage {
  const factory SecurityActivityPage({
    required List<SecurityEvent> data,
    String? nextCursor,
  }) = _SecurityActivityPage;
}
