// Form-specific types for the Create Project flow.
//
// These are presentation-layer types used only by the form.
// They are separate from the domain [Project] entity which
// represents persisted data from the API.

enum PaymentType { invoice, escrow }

enum ProjectStructure { milestone, oneTime }

enum BillingType { fixed, hourly }

enum BillingFrequency { weekly, biWeekly, monthly }

enum ApplicantType { freelancersAndAgencies, freelancersOnly, agenciesOnly }

/// A single milestone within an escrow-based project.
class MilestoneData {
  MilestoneData({
    this.title = '',
    this.description = '',
    this.amount = 0,
    this.deadline,
  });

  String title;
  String description;
  double amount;
  DateTime? deadline;
}

/// Holds all data collected by the Create Project form.
class ProjectFormData {
  ProjectFormData({
    this.paymentType = PaymentType.escrow,
    this.structure = ProjectStructure.milestone,
    this.billingType = BillingType.fixed,
    this.frequency = BillingFrequency.monthly,
    this.title = '',
    this.description = '',
    List<String>? skills,
    List<MilestoneData>? milestones,
    this.amount = 0,
    this.rate = 0,
    this.startDate,
    this.deadline,
    this.ongoing = false,
    this.applicantType = ApplicantType.freelancersAndAgencies,
    this.negotiable = false,
  })  : skills = skills ?? [],
        milestones = milestones ?? [MilestoneData()];

  PaymentType paymentType;
  ProjectStructure structure;
  BillingType billingType;
  BillingFrequency frequency;
  String title;
  String description;
  List<String> skills;
  List<MilestoneData> milestones;
  double amount;
  double rate;
  DateTime? startDate;
  DateTime? deadline;
  bool ongoing;
  ApplicantType applicantType;
  bool negotiable;
}
