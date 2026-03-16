import 'package:freezed_annotation/freezed_annotation.dart';

part 'project.freezed.dart';
part 'project.g.dart';

enum BudgetType { fixed, hourly, monthly }

@freezed
class Project with _$Project {
  const factory Project({
    required String id,
    required String title,
    required String description,
    @Default([]) List<String> skills,
    @Default(BudgetType.fixed) BudgetType budgetType,
    double? minBudget,
    double? maxBudget,
    required DateTime createdAt,
  }) = _Project;

  factory Project.fromJson(Map<String, dynamic> json) =>
      _$ProjectFromJson(json);
}
