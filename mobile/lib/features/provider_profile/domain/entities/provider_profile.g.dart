// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'provider_profile.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$ProviderProfileImpl _$$ProviderProfileImplFromJson(
        Map<String, dynamic> json) =>
    _$ProviderProfileImpl(
      id: json['id'] as String,
      userId: json['userId'] as String,
      name: json['name'] as String,
      title: json['title'] as String?,
      city: json['city'] as String?,
      country: json['country'] as String?,
      about: json['about'] as String?,
      logoUrl: json['logoUrl'] as String?,
      skills: (json['skills'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          const [],
      expertises: (json['expertises'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          const [],
    );

Map<String, dynamic> _$$ProviderProfileImplToJson(
        _$ProviderProfileImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'userId': instance.userId,
      'name': instance.name,
      'title': instance.title,
      'city': instance.city,
      'country': instance.country,
      'about': instance.about,
      'logoUrl': instance.logoUrl,
      'skills': instance.skills,
      'expertises': instance.expertises,
    };
