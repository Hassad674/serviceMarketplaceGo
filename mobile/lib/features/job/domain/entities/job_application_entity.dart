import 'job_entity.dart';

class JobApplicationEntity {
  const JobApplicationEntity({
    required this.id,
    required this.jobId,
    required this.applicantId,
    required this.message,
    this.videoUrl,
    required this.createdAt,
  });

  final String id;
  final String jobId;
  final String applicantId;
  final String message;
  final String? videoUrl;
  final String createdAt;

  factory JobApplicationEntity.fromJson(Map<String, dynamic> json) {
    return JobApplicationEntity(
      id: json['id'] as String,
      jobId: json['job_id'] as String,
      applicantId: json['applicant_id'] as String,
      message: (json['message'] as String?) ?? '',
      videoUrl: json['video_url'] as String?,
      createdAt: json['created_at'] as String,
    );
  }
}

class PublicProfileSummary {
  const PublicProfileSummary({
    required this.userId,
    required this.displayName,
    required this.firstName,
    required this.lastName,
    required this.role,
    required this.title,
    required this.photoUrl,
    required this.referrerEnabled,
  });

  final String userId;
  final String displayName;
  final String firstName;
  final String lastName;
  final String role;
  final String title;
  final String photoUrl;
  final bool referrerEnabled;

  factory PublicProfileSummary.fromJson(Map<String, dynamic> json) {
    return PublicProfileSummary(
      userId: json['user_id'] as String,
      displayName: (json['display_name'] as String?) ?? '',
      firstName: (json['first_name'] as String?) ?? '',
      lastName: (json['last_name'] as String?) ?? '',
      role: (json['role'] as String?) ?? '',
      title: (json['title'] as String?) ?? '',
      photoUrl: (json['photo_url'] as String?) ?? '',
      referrerEnabled: (json['referrer_enabled'] as bool?) ?? false,
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
