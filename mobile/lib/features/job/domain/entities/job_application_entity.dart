import 'job_entity.dart';

class JobApplicationEntity {
  const JobApplicationEntity({
    required this.id,
    required this.jobId,
    required this.applicantOrgId,
    required this.message,
    this.videoUrl,
    required this.createdAt,
  });

  final String id;
  final String jobId;

  /// Since phase R3 the applicant is the organization that applied,
  /// not an individual user. The field is still surfaced as
  /// `applicant_id` on the wire for backwards-compatible JSON, but it
  /// holds the applicant org id.
  final String applicantOrgId;
  final String message;
  final String? videoUrl;
  final String createdAt;

  factory JobApplicationEntity.fromJson(Map<String, dynamic> json) {
    return JobApplicationEntity(
      id: json['id'] as String,
      jobId: json['job_id'] as String,
      applicantOrgId: json['applicant_id'] as String,
      message: (json['message'] as String?) ?? '',
      videoUrl: json['video_url'] as String?,
      createdAt: json['created_at'] as String,
    );
  }
}

/// Public profile summary as surfaced by GET /api/v1/profiles/search
/// and embedded in job application lists. Since phase R2/R6 this row
/// describes an organization — every operator of the team shares the
/// same summary.
class PublicProfileSummary {
  const PublicProfileSummary({
    required this.organizationId,
    required this.name,
    required this.orgType,
    required this.title,
    required this.photoUrl,
    required this.referrerEnabled,
    this.averageRating = 0,
    this.reviewCount = 0,
  });

  final String organizationId;
  final String name;
  final String orgType;
  final String title;
  final String photoUrl;
  final bool referrerEnabled;
  final double averageRating;
  final int reviewCount;

  factory PublicProfileSummary.fromJson(Map<String, dynamic> json) {
    return PublicProfileSummary(
      organizationId: json['organization_id'] as String,
      name: (json['name'] as String?) ?? '',
      orgType: (json['org_type'] as String?) ?? '',
      title: (json['title'] as String?) ?? '',
      photoUrl: (json['photo_url'] as String?) ?? '',
      referrerEnabled: (json['referrer_enabled'] as bool?) ?? false,
      averageRating:
          ((json['average_rating'] as num?) ?? 0).toDouble(),
      reviewCount: (json['review_count'] as int?) ?? 0,
    );
  }
}

class ApplicationWithProfile {
  const ApplicationWithProfile({
    required this.application,
    required this.profile,
  });

  final JobApplicationEntity application;
  final PublicProfileSummary profile;

  factory ApplicationWithProfile.fromJson(Map<String, dynamic> json) {
    return ApplicationWithProfile(
      application: JobApplicationEntity.fromJson(json['application'] as Map<String, dynamic>),
      profile: PublicProfileSummary.fromJson(json['profile'] as Map<String, dynamic>),
    );
  }
}

class ApplicationWithJob {
  const ApplicationWithJob({
    required this.application,
    required this.job,
  });

  final JobApplicationEntity application;
  final JobEntity job;

  factory ApplicationWithJob.fromJson(Map<String, dynamic> json) {
    return ApplicationWithJob(
      application: JobApplicationEntity.fromJson(json['application'] as Map<String, dynamic>),
      job: JobEntity.fromJson(json['job'] as Map<String, dynamic>),
    );
  }
}
