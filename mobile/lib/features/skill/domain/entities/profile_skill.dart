/// One skill attached to the current operator's profile, as
/// returned by `GET /api/v1/profile/skills`.
///
/// [position] preserves the server-side ordering so the chips on
/// the profile render in the exact order the user arranged them.
/// The backend guarantees `position` starts at 0 and is contiguous
/// per-profile.
class ProfileSkill {
  /// Canonical key of the underlying catalog entry (lowercased,
  /// trimmed). This is the value sent back on save.
  final String skillText;

  /// Human-readable form used to render the chip label.
  final String displayText;

  /// Zero-based position in the user's ordered list.
  final int position;

  const ProfileSkill({
    required this.skillText,
    required this.displayText,
    required this.position,
  });

  factory ProfileSkill.fromJson(Map<String, dynamic> json) {
    return ProfileSkill(
      skillText: (json['skill_text'] as String?) ?? '',
      displayText: (json['display_text'] as String?) ??
          (json['skill_text'] as String?) ??
          '',
      position: _parseInt(json['position']),
    );
  }

  Map<String, dynamic> toJson() => <String, dynamic>{
        'skill_text': skillText,
        'display_text': displayText,
        'position': position,
      };

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ProfileSkill &&
          runtimeType == other.runtimeType &&
          skillText == other.skillText &&
          position == other.position;

  @override
  int get hashCode => Object.hash(skillText, position);

  static int _parseInt(dynamic raw) {
    if (raw is int) return raw;
    if (raw is num) return raw.toInt();
    if (raw is String) return int.tryParse(raw) ?? 0;
    return 0;
  }
}
