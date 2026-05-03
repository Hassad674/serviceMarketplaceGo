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
    this.paymentFrequency,
    this.durationWeeks,
    this.isIndefinite = false,
    this.descriptionType = 'text',
    this.videoUrl,
    this.totalApplicants = 0,
    this.newApplicants = 0,
  });

  final String id;
  final String creatorId;
  final String title;
  final String description;
  final List<String> skills;
  final String applicantType;
  final String budgetType;
  final int minBudget;
  final int maxBudget;
  final String status;
  final String createdAt;
  final String updatedAt;
  final String? closedAt;
  final String? paymentFrequency;
  final int? durationWeeks;
  final bool isIndefinite;
  final String descriptionType;
  final String? videoUrl;
  final int totalApplicants;
  final int newApplicants;

  bool get isOpen => status == 'open';

  factory JobEntity.fromJson(Map<String, dynamic> json) {
    return JobEntity(
      id: json['id'] as String,
      creatorId: json['creator_id'] as String,
      title: json['title'] as String,
      description: (json['description'] as String?) ?? '',
      skills: (json['skills'] as List?)?.map((e) => e as String).toList() ?? [],
      applicantType: json['applicant_type'] as String,
      budgetType: json['budget_type'] as String,
      minBudget: json['min_budget'] as int,
      maxBudget: json['max_budget'] as int,
      status: json['status'] as String,
      createdAt: json['created_at'] as String,
      updatedAt: json['updated_at'] as String,
      closedAt: json['closed_at'] as String?,
      paymentFrequency: json['payment_frequency'] as String?,
      durationWeeks: json['duration_weeks'] as int?,
      isIndefinite: (json['is_indefinite'] as bool?) ?? false,
      descriptionType: (json['description_type'] as String?) ?? 'text',
      videoUrl: json['video_url'] as String?,
      totalApplicants: (json['total_applicants'] as int?) ?? 0,
      newApplicants: (json['new_applicants'] as int?) ?? 0,
    );
  }
}
