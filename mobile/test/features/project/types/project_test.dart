import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/project/types/project.dart';

void main() {
  group('ProjectFormData', () {
    test('creates with default values', () {
      final form = ProjectFormData();

      expect(form.paymentType, PaymentType.escrow);
      expect(form.structure, ProjectStructure.milestone);
      expect(form.billingType, BillingType.fixed);
      expect(form.frequency, BillingFrequency.monthly);
      expect(form.title, '');
      expect(form.description, '');
      expect(form.skills, isEmpty);
      expect(form.milestones, hasLength(1));
      expect(form.amount, 0);
      expect(form.rate, 0);
      expect(form.startDate, isNull);
      expect(form.deadline, isNull);
      expect(form.ongoing, false);
      expect(form.applicantType, ApplicantType.freelancersAndAgencies);
      expect(form.negotiable, false);
    });

    test('creates with custom values', () {
      final startDate = DateTime(2026, 4, 1);
      final deadline = DateTime(2026, 6, 30);

      final form = ProjectFormData(
        paymentType: PaymentType.invoice,
        structure: ProjectStructure.oneTime,
        billingType: BillingType.hourly,
        frequency: BillingFrequency.weekly,
        title: 'Website Redesign',
        description: 'Complete redesign of the company website',
        skills: ['Flutter', 'Dart', 'Firebase'],
        amount: 5000,
        rate: 75,
        startDate: startDate,
        deadline: deadline,
        ongoing: false,
        applicantType: ApplicantType.freelancersOnly,
        negotiable: true,
      );

      expect(form.paymentType, PaymentType.invoice);
      expect(form.structure, ProjectStructure.oneTime);
      expect(form.billingType, BillingType.hourly);
      expect(form.frequency, BillingFrequency.weekly);
      expect(form.title, 'Website Redesign');
      expect(form.description, 'Complete redesign of the company website');
      expect(form.skills, ['Flutter', 'Dart', 'Firebase']);
      expect(form.amount, 5000);
      expect(form.rate, 75);
      expect(form.startDate, startDate);
      expect(form.deadline, deadline);
      expect(form.ongoing, false);
      expect(form.applicantType, ApplicantType.freelancersOnly);
      expect(form.negotiable, true);
    });

    test('default milestones list has one empty milestone', () {
      final form = ProjectFormData();

      expect(form.milestones, hasLength(1));
      expect(form.milestones.first.title, '');
      expect(form.milestones.first.description, '');
      expect(form.milestones.first.amount, 0);
      expect(form.milestones.first.deadline, isNull);
    });

    test('custom milestones override default', () {
      final form = ProjectFormData(
        milestones: [
          MilestoneData(title: 'Phase 1', amount: 1000),
          MilestoneData(title: 'Phase 2', amount: 2000),
        ],
      );

      expect(form.milestones, hasLength(2));
      expect(form.milestones[0].title, 'Phase 1');
      expect(form.milestones[0].amount, 1000);
      expect(form.milestones[1].title, 'Phase 2');
      expect(form.milestones[1].amount, 2000);
    });

    test('skills default to empty list', () {
      final form = ProjectFormData();
      expect(form.skills, isEmpty);
    });

    test('custom skills are preserved', () {
      final form = ProjectFormData(
        skills: ['Go', 'PostgreSQL', 'Docker'],
      );

      expect(form.skills, hasLength(3));
      expect(form.skills, contains('Go'));
      expect(form.skills, contains('PostgreSQL'));
      expect(form.skills, contains('Docker'));
    });

    test('fields are mutable', () {
      final form = ProjectFormData();

      form.title = 'Updated Title';
      form.description = 'Updated Description';
      form.paymentType = PaymentType.invoice;
      form.ongoing = true;

      expect(form.title, 'Updated Title');
      expect(form.description, 'Updated Description');
      expect(form.paymentType, PaymentType.invoice);
      expect(form.ongoing, true);
    });
  });

  group('MilestoneData', () {
    test('creates with default values', () {
      final milestone = MilestoneData();

      expect(milestone.title, '');
      expect(milestone.description, '');
      expect(milestone.amount, 0);
      expect(milestone.deadline, isNull);
    });

    test('creates with custom values', () {
      final deadline = DateTime(2026, 5, 15);
      final milestone = MilestoneData(
        title: 'Design Phase',
        description: 'Complete all wireframes and mockups',
        amount: 2500,
        deadline: deadline,
      );

      expect(milestone.title, 'Design Phase');
      expect(milestone.description, 'Complete all wireframes and mockups');
      expect(milestone.amount, 2500);
      expect(milestone.deadline, deadline);
    });

    test('fields are mutable', () {
      final milestone = MilestoneData();

      milestone.title = 'Updated';
      milestone.amount = 999;
      milestone.deadline = DateTime(2026, 12, 25);

      expect(milestone.title, 'Updated');
      expect(milestone.amount, 999);
      expect(milestone.deadline, DateTime(2026, 12, 25));
    });
  });

  group('Enums', () {
    test('PaymentType has exactly 2 values', () {
      expect(PaymentType.values, hasLength(2));
      expect(PaymentType.values, contains(PaymentType.invoice));
      expect(PaymentType.values, contains(PaymentType.escrow));
    });

    test('ProjectStructure has exactly 2 values', () {
      expect(ProjectStructure.values, hasLength(2));
      expect(ProjectStructure.values, contains(ProjectStructure.milestone));
      expect(ProjectStructure.values, contains(ProjectStructure.oneTime));
    });

    test('BillingType has exactly 2 values', () {
      expect(BillingType.values, hasLength(2));
      expect(BillingType.values, contains(BillingType.fixed));
      expect(BillingType.values, contains(BillingType.hourly));
    });

    test('BillingFrequency has exactly 3 values', () {
      expect(BillingFrequency.values, hasLength(3));
      expect(BillingFrequency.values, contains(BillingFrequency.weekly));
      expect(BillingFrequency.values, contains(BillingFrequency.biWeekly));
      expect(BillingFrequency.values, contains(BillingFrequency.monthly));
    });

    test('ApplicantType has exactly 3 values', () {
      expect(ApplicantType.values, hasLength(3));
      expect(ApplicantType.values, contains(ApplicantType.freelancersAndAgencies));
      expect(ApplicantType.values, contains(ApplicantType.freelancersOnly));
      expect(ApplicantType.values, contains(ApplicantType.agenciesOnly));
    });
  });
}
