// Form-specific types for the Create Job flow.
//
// These are presentation-layer types used only by the form.
// They are separate from any future domain [Job] entity which
// would represent persisted data from the API.

enum BudgetType { ongoing, oneTime }

enum PaymentFrequency { hourly, weekly, monthly }

enum ApplicantType { all, freelancers, agencies }

/// Duration unit for estimated project length.
enum DurationUnit { weeks, months }

/// Holds all data collected by the Create Job form.
class JobFormData {
  JobFormData({
    this.title = '',
    this.description = '',
    List<String>? skills,
    List<String>? tools,
    this.contractorCount = 1,
    this.applicantType = ApplicantType.all,
    this.budgetType = BudgetType.ongoing,
    this.paymentFrequency = PaymentFrequency.hourly,
    this.minRate = '',
    this.maxRate = '',
    this.maxHoursPerWeek = 20,
    this.minBudget = '',
    this.maxBudget = '',
    this.estimatedDuration = '',
    this.durationUnit = DurationUnit.weeks,
    this.isIndefinite = false,
  })  : skills = skills ?? [],
        tools = tools ?? [];

  String title;
  String description;
  List<String> skills;
  List<String> tools;
  int contractorCount;
  ApplicantType applicantType;
  BudgetType budgetType;
  PaymentFrequency paymentFrequency;
  String minRate;
  String maxRate;
  int maxHoursPerWeek;
  String minBudget;
  String maxBudget;
  String estimatedDuration;
  DurationUnit durationUnit;
  bool isIndefinite;
}
