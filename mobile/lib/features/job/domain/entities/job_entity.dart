/// Domain entity representing a job posting.
///
/// Maps to the backend `JobResponse` from
/// `GET /api/v1/jobs/{id}` and `GET /api/v1/jobs/mine`.
class JobEntity {
  const JobEntity({
    required this.id,
    required this.creatorId,
    required this.title,
    required this.description,
    required this.skills,
    required this.applicantType,
    required this.budgetType,
    required this.minBudget,
    required this.maxBudget,
    required this.status,
    required this.createdAt,
    required this.updatedAt,
    this.closedAt,
  });

  final String id;
  final String creatorId;
  final String title;
  final String description;
  final List<String> skills;
  final String applicantType; // all | freelancers | agencies
  final String budgetType; // one_shot | long_term
  final int minBudget;
  final int maxBudget;
  final String status; // open | closed
  final String createdAt;
  final String updatedAt;
  final String? closedAt;

  bool get isOpen => status == 'open';

  factory JobEntity.fromJson(Map<String, dynamic> json) {
    return JobEntity(
      id: json['id'] as String,
      creatorId: json['creator_id'] as String,
      title: json['title'] as String,
      description: json['description'] as String,
      skills: (json['skills'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          [],
      applicantType: json['applicant_type'] as String,
      budgetType: json['budget_type'] as String,
      minBudget: json['min_budget'] as int,
      maxBudget: json['max_budget'] as int,
      status: json['status'] as String,
      createdAt: json['created_at'] as String,
      updatedAt: json['updated_at'] as String,
      closedAt: json['closed_at'] as String?,
    );
  }
}
