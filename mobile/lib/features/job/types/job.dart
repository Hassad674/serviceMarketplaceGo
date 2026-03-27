enum BudgetType { oneShot, longTerm }
enum ApplicantType { all, freelancers, agencies }
enum PaymentFrequency { weekly, monthly }
enum DescriptionType { text, video, both }

class JobFormData {
  JobFormData({
    this.title = '',
    this.description = '',
    List<String>? skills,
    this.applicantType = ApplicantType.all,
    this.budgetType = BudgetType.oneShot,
    this.minBudget = '',
    this.maxBudget = '',
    this.paymentFrequency = PaymentFrequency.monthly,
    this.durationWeeks = '',
    this.isIndefinite = false,
    this.descriptionType = DescriptionType.text,
    this.videoUrl = '',
  }) : skills = skills ?? [];

  String title;
  String description;
  List<String> skills;
  ApplicantType applicantType;
  BudgetType budgetType;
  String minBudget;
  String maxBudget;
  PaymentFrequency paymentFrequency;
  String durationWeeks;
  bool isIndefinite;
  DescriptionType descriptionType;
  String videoUrl;
}
