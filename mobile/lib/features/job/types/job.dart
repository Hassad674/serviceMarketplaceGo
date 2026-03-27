// Form-specific types for the Create Job flow.
//
// These are presentation-layer types used only by the form.
// They are separate from the domain [JobEntity] which
// represents persisted data from the API.

enum BudgetType { oneShot, longTerm }

enum ApplicantType { all, freelancers, agencies }

/// Holds all data collected by the Create Job form.
class JobFormData {
  JobFormData({
    this.title = '',
    this.description = '',
    List<String>? skills,
    this.applicantType = ApplicantType.all,
    this.budgetType = BudgetType.oneShot,
    this.minBudget = '',
    this.maxBudget = '',
  }) : skills = skills ?? [];

  String title;
  String description;
  List<String> skills;
  ApplicantType applicantType;
  BudgetType budgetType;
  String minBudget;
  String maxBudget;
}
