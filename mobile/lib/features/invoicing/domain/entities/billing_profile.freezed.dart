// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'billing_profile.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$BillingProfile {
  String get organizationId => throw _privateConstructorUsedError;
  ProfileType get profileType => throw _privateConstructorUsedError;
  String get legalName => throw _privateConstructorUsedError;
  String get tradingName => throw _privateConstructorUsedError;
  String get legalForm => throw _privateConstructorUsedError;
  String get taxId => throw _privateConstructorUsedError;
  String get vatNumber => throw _privateConstructorUsedError;
  DateTime? get vatValidatedAt => throw _privateConstructorUsedError;
  String get addressLine1 => throw _privateConstructorUsedError;
  String get addressLine2 => throw _privateConstructorUsedError;
  String get postalCode => throw _privateConstructorUsedError;
  String get city => throw _privateConstructorUsedError;
  String get country => throw _privateConstructorUsedError;
  String get invoicingEmail => throw _privateConstructorUsedError;
  DateTime? get syncedFromKycAt => throw _privateConstructorUsedError;

  /// Create a copy of BillingProfile
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $BillingProfileCopyWith<BillingProfile> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $BillingProfileCopyWith<$Res> {
  factory $BillingProfileCopyWith(
          BillingProfile value, $Res Function(BillingProfile) then) =
      _$BillingProfileCopyWithImpl<$Res, BillingProfile>;
  @useResult
  $Res call(
      {String organizationId,
      ProfileType profileType,
      String legalName,
      String tradingName,
      String legalForm,
      String taxId,
      String vatNumber,
      DateTime? vatValidatedAt,
      String addressLine1,
      String addressLine2,
      String postalCode,
      String city,
      String country,
      String invoicingEmail,
      DateTime? syncedFromKycAt});
}

/// @nodoc
class _$BillingProfileCopyWithImpl<$Res, $Val extends BillingProfile>
    implements $BillingProfileCopyWith<$Res> {
  _$BillingProfileCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of BillingProfile
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? organizationId = null,
    Object? profileType = null,
    Object? legalName = null,
    Object? tradingName = null,
    Object? legalForm = null,
    Object? taxId = null,
    Object? vatNumber = null,
    Object? vatValidatedAt = freezed,
    Object? addressLine1 = null,
    Object? addressLine2 = null,
    Object? postalCode = null,
    Object? city = null,
    Object? country = null,
    Object? invoicingEmail = null,
    Object? syncedFromKycAt = freezed,
  }) {
    return _then(_value.copyWith(
      organizationId: null == organizationId
          ? _value.organizationId
          : organizationId // ignore: cast_nullable_to_non_nullable
              as String,
      profileType: null == profileType
          ? _value.profileType
          : profileType // ignore: cast_nullable_to_non_nullable
              as ProfileType,
      legalName: null == legalName
          ? _value.legalName
          : legalName // ignore: cast_nullable_to_non_nullable
              as String,
      tradingName: null == tradingName
          ? _value.tradingName
          : tradingName // ignore: cast_nullable_to_non_nullable
              as String,
      legalForm: null == legalForm
          ? _value.legalForm
          : legalForm // ignore: cast_nullable_to_non_nullable
              as String,
      taxId: null == taxId
          ? _value.taxId
          : taxId // ignore: cast_nullable_to_non_nullable
              as String,
      vatNumber: null == vatNumber
          ? _value.vatNumber
          : vatNumber // ignore: cast_nullable_to_non_nullable
              as String,
      vatValidatedAt: freezed == vatValidatedAt
          ? _value.vatValidatedAt
          : vatValidatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
      addressLine1: null == addressLine1
          ? _value.addressLine1
          : addressLine1 // ignore: cast_nullable_to_non_nullable
              as String,
      addressLine2: null == addressLine2
          ? _value.addressLine2
          : addressLine2 // ignore: cast_nullable_to_non_nullable
              as String,
      postalCode: null == postalCode
          ? _value.postalCode
          : postalCode // ignore: cast_nullable_to_non_nullable
              as String,
      city: null == city
          ? _value.city
          : city // ignore: cast_nullable_to_non_nullable
              as String,
      country: null == country
          ? _value.country
          : country // ignore: cast_nullable_to_non_nullable
              as String,
      invoicingEmail: null == invoicingEmail
          ? _value.invoicingEmail
          : invoicingEmail // ignore: cast_nullable_to_non_nullable
              as String,
      syncedFromKycAt: freezed == syncedFromKycAt
          ? _value.syncedFromKycAt
          : syncedFromKycAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$BillingProfileImplCopyWith<$Res>
    implements $BillingProfileCopyWith<$Res> {
  factory _$$BillingProfileImplCopyWith(_$BillingProfileImpl value,
          $Res Function(_$BillingProfileImpl) then) =
      __$$BillingProfileImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String organizationId,
      ProfileType profileType,
      String legalName,
      String tradingName,
      String legalForm,
      String taxId,
      String vatNumber,
      DateTime? vatValidatedAt,
      String addressLine1,
      String addressLine2,
      String postalCode,
      String city,
      String country,
      String invoicingEmail,
      DateTime? syncedFromKycAt});
}

/// @nodoc
class __$$BillingProfileImplCopyWithImpl<$Res>
    extends _$BillingProfileCopyWithImpl<$Res, _$BillingProfileImpl>
    implements _$$BillingProfileImplCopyWith<$Res> {
  __$$BillingProfileImplCopyWithImpl(
      _$BillingProfileImpl _value, $Res Function(_$BillingProfileImpl) _then)
      : super(_value, _then);

  /// Create a copy of BillingProfile
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? organizationId = null,
    Object? profileType = null,
    Object? legalName = null,
    Object? tradingName = null,
    Object? legalForm = null,
    Object? taxId = null,
    Object? vatNumber = null,
    Object? vatValidatedAt = freezed,
    Object? addressLine1 = null,
    Object? addressLine2 = null,
    Object? postalCode = null,
    Object? city = null,
    Object? country = null,
    Object? invoicingEmail = null,
    Object? syncedFromKycAt = freezed,
  }) {
    return _then(_$BillingProfileImpl(
      organizationId: null == organizationId
          ? _value.organizationId
          : organizationId // ignore: cast_nullable_to_non_nullable
              as String,
      profileType: null == profileType
          ? _value.profileType
          : profileType // ignore: cast_nullable_to_non_nullable
              as ProfileType,
      legalName: null == legalName
          ? _value.legalName
          : legalName // ignore: cast_nullable_to_non_nullable
              as String,
      tradingName: null == tradingName
          ? _value.tradingName
          : tradingName // ignore: cast_nullable_to_non_nullable
              as String,
      legalForm: null == legalForm
          ? _value.legalForm
          : legalForm // ignore: cast_nullable_to_non_nullable
              as String,
      taxId: null == taxId
          ? _value.taxId
          : taxId // ignore: cast_nullable_to_non_nullable
              as String,
      vatNumber: null == vatNumber
          ? _value.vatNumber
          : vatNumber // ignore: cast_nullable_to_non_nullable
              as String,
      vatValidatedAt: freezed == vatValidatedAt
          ? _value.vatValidatedAt
          : vatValidatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
      addressLine1: null == addressLine1
          ? _value.addressLine1
          : addressLine1 // ignore: cast_nullable_to_non_nullable
              as String,
      addressLine2: null == addressLine2
          ? _value.addressLine2
          : addressLine2 // ignore: cast_nullable_to_non_nullable
              as String,
      postalCode: null == postalCode
          ? _value.postalCode
          : postalCode // ignore: cast_nullable_to_non_nullable
              as String,
      city: null == city
          ? _value.city
          : city // ignore: cast_nullable_to_non_nullable
              as String,
      country: null == country
          ? _value.country
          : country // ignore: cast_nullable_to_non_nullable
              as String,
      invoicingEmail: null == invoicingEmail
          ? _value.invoicingEmail
          : invoicingEmail // ignore: cast_nullable_to_non_nullable
              as String,
      syncedFromKycAt: freezed == syncedFromKycAt
          ? _value.syncedFromKycAt
          : syncedFromKycAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ));
  }
}

/// @nodoc

class _$BillingProfileImpl implements _BillingProfile {
  const _$BillingProfileImpl(
      {required this.organizationId,
      required this.profileType,
      required this.legalName,
      required this.tradingName,
      required this.legalForm,
      required this.taxId,
      required this.vatNumber,
      this.vatValidatedAt,
      required this.addressLine1,
      required this.addressLine2,
      required this.postalCode,
      required this.city,
      required this.country,
      required this.invoicingEmail,
      this.syncedFromKycAt});

  @override
  final String organizationId;
  @override
  final ProfileType profileType;
  @override
  final String legalName;
  @override
  final String tradingName;
  @override
  final String legalForm;
  @override
  final String taxId;
  @override
  final String vatNumber;
  @override
  final DateTime? vatValidatedAt;
  @override
  final String addressLine1;
  @override
  final String addressLine2;
  @override
  final String postalCode;
  @override
  final String city;
  @override
  final String country;
  @override
  final String invoicingEmail;
  @override
  final DateTime? syncedFromKycAt;

  @override
  String toString() {
    return 'BillingProfile(organizationId: $organizationId, profileType: $profileType, legalName: $legalName, tradingName: $tradingName, legalForm: $legalForm, taxId: $taxId, vatNumber: $vatNumber, vatValidatedAt: $vatValidatedAt, addressLine1: $addressLine1, addressLine2: $addressLine2, postalCode: $postalCode, city: $city, country: $country, invoicingEmail: $invoicingEmail, syncedFromKycAt: $syncedFromKycAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$BillingProfileImpl &&
            (identical(other.organizationId, organizationId) ||
                other.organizationId == organizationId) &&
            (identical(other.profileType, profileType) ||
                other.profileType == profileType) &&
            (identical(other.legalName, legalName) ||
                other.legalName == legalName) &&
            (identical(other.tradingName, tradingName) ||
                other.tradingName == tradingName) &&
            (identical(other.legalForm, legalForm) ||
                other.legalForm == legalForm) &&
            (identical(other.taxId, taxId) || other.taxId == taxId) &&
            (identical(other.vatNumber, vatNumber) ||
                other.vatNumber == vatNumber) &&
            (identical(other.vatValidatedAt, vatValidatedAt) ||
                other.vatValidatedAt == vatValidatedAt) &&
            (identical(other.addressLine1, addressLine1) ||
                other.addressLine1 == addressLine1) &&
            (identical(other.addressLine2, addressLine2) ||
                other.addressLine2 == addressLine2) &&
            (identical(other.postalCode, postalCode) ||
                other.postalCode == postalCode) &&
            (identical(other.city, city) || other.city == city) &&
            (identical(other.country, country) || other.country == country) &&
            (identical(other.invoicingEmail, invoicingEmail) ||
                other.invoicingEmail == invoicingEmail) &&
            (identical(other.syncedFromKycAt, syncedFromKycAt) ||
                other.syncedFromKycAt == syncedFromKycAt));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType,
      organizationId,
      profileType,
      legalName,
      tradingName,
      legalForm,
      taxId,
      vatNumber,
      vatValidatedAt,
      addressLine1,
      addressLine2,
      postalCode,
      city,
      country,
      invoicingEmail,
      syncedFromKycAt);

  /// Create a copy of BillingProfile
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$BillingProfileImplCopyWith<_$BillingProfileImpl> get copyWith =>
      __$$BillingProfileImplCopyWithImpl<_$BillingProfileImpl>(
          this, _$identity);
}

abstract class _BillingProfile implements BillingProfile {
  const factory _BillingProfile(
      {required final String organizationId,
      required final ProfileType profileType,
      required final String legalName,
      required final String tradingName,
      required final String legalForm,
      required final String taxId,
      required final String vatNumber,
      final DateTime? vatValidatedAt,
      required final String addressLine1,
      required final String addressLine2,
      required final String postalCode,
      required final String city,
      required final String country,
      required final String invoicingEmail,
      final DateTime? syncedFromKycAt}) = _$BillingProfileImpl;

  @override
  String get organizationId;
  @override
  ProfileType get profileType;
  @override
  String get legalName;
  @override
  String get tradingName;
  @override
  String get legalForm;
  @override
  String get taxId;
  @override
  String get vatNumber;
  @override
  DateTime? get vatValidatedAt;
  @override
  String get addressLine1;
  @override
  String get addressLine2;
  @override
  String get postalCode;
  @override
  String get city;
  @override
  String get country;
  @override
  String get invoicingEmail;
  @override
  DateTime? get syncedFromKycAt;

  /// Create a copy of BillingProfile
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$BillingProfileImplCopyWith<_$BillingProfileImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
mixin _$UpdateBillingProfileInput {
  ProfileType get profileType => throw _privateConstructorUsedError;
  String get legalName => throw _privateConstructorUsedError;
  String get tradingName => throw _privateConstructorUsedError;
  String get legalForm => throw _privateConstructorUsedError;
  String get taxId => throw _privateConstructorUsedError;
  String get vatNumber => throw _privateConstructorUsedError;
  String get addressLine1 => throw _privateConstructorUsedError;
  String get addressLine2 => throw _privateConstructorUsedError;
  String get postalCode => throw _privateConstructorUsedError;
  String get city => throw _privateConstructorUsedError;
  String get country => throw _privateConstructorUsedError;
  String get invoicingEmail => throw _privateConstructorUsedError;

  /// Create a copy of UpdateBillingProfileInput
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UpdateBillingProfileInputCopyWith<UpdateBillingProfileInput> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UpdateBillingProfileInputCopyWith<$Res> {
  factory $UpdateBillingProfileInputCopyWith(UpdateBillingProfileInput value,
          $Res Function(UpdateBillingProfileInput) then) =
      _$UpdateBillingProfileInputCopyWithImpl<$Res, UpdateBillingProfileInput>;
  @useResult
  $Res call(
      {ProfileType profileType,
      String legalName,
      String tradingName,
      String legalForm,
      String taxId,
      String vatNumber,
      String addressLine1,
      String addressLine2,
      String postalCode,
      String city,
      String country,
      String invoicingEmail});
}

/// @nodoc
class _$UpdateBillingProfileInputCopyWithImpl<$Res,
        $Val extends UpdateBillingProfileInput>
    implements $UpdateBillingProfileInputCopyWith<$Res> {
  _$UpdateBillingProfileInputCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UpdateBillingProfileInput
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? profileType = null,
    Object? legalName = null,
    Object? tradingName = null,
    Object? legalForm = null,
    Object? taxId = null,
    Object? vatNumber = null,
    Object? addressLine1 = null,
    Object? addressLine2 = null,
    Object? postalCode = null,
    Object? city = null,
    Object? country = null,
    Object? invoicingEmail = null,
  }) {
    return _then(_value.copyWith(
      profileType: null == profileType
          ? _value.profileType
          : profileType // ignore: cast_nullable_to_non_nullable
              as ProfileType,
      legalName: null == legalName
          ? _value.legalName
          : legalName // ignore: cast_nullable_to_non_nullable
              as String,
      tradingName: null == tradingName
          ? _value.tradingName
          : tradingName // ignore: cast_nullable_to_non_nullable
              as String,
      legalForm: null == legalForm
          ? _value.legalForm
          : legalForm // ignore: cast_nullable_to_non_nullable
              as String,
      taxId: null == taxId
          ? _value.taxId
          : taxId // ignore: cast_nullable_to_non_nullable
              as String,
      vatNumber: null == vatNumber
          ? _value.vatNumber
          : vatNumber // ignore: cast_nullable_to_non_nullable
              as String,
      addressLine1: null == addressLine1
          ? _value.addressLine1
          : addressLine1 // ignore: cast_nullable_to_non_nullable
              as String,
      addressLine2: null == addressLine2
          ? _value.addressLine2
          : addressLine2 // ignore: cast_nullable_to_non_nullable
              as String,
      postalCode: null == postalCode
          ? _value.postalCode
          : postalCode // ignore: cast_nullable_to_non_nullable
              as String,
      city: null == city
          ? _value.city
          : city // ignore: cast_nullable_to_non_nullable
              as String,
      country: null == country
          ? _value.country
          : country // ignore: cast_nullable_to_non_nullable
              as String,
      invoicingEmail: null == invoicingEmail
          ? _value.invoicingEmail
          : invoicingEmail // ignore: cast_nullable_to_non_nullable
              as String,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$UpdateBillingProfileInputImplCopyWith<$Res>
    implements $UpdateBillingProfileInputCopyWith<$Res> {
  factory _$$UpdateBillingProfileInputImplCopyWith(
          _$UpdateBillingProfileInputImpl value,
          $Res Function(_$UpdateBillingProfileInputImpl) then) =
      __$$UpdateBillingProfileInputImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {ProfileType profileType,
      String legalName,
      String tradingName,
      String legalForm,
      String taxId,
      String vatNumber,
      String addressLine1,
      String addressLine2,
      String postalCode,
      String city,
      String country,
      String invoicingEmail});
}

/// @nodoc
class __$$UpdateBillingProfileInputImplCopyWithImpl<$Res>
    extends _$UpdateBillingProfileInputCopyWithImpl<$Res,
        _$UpdateBillingProfileInputImpl>
    implements _$$UpdateBillingProfileInputImplCopyWith<$Res> {
  __$$UpdateBillingProfileInputImplCopyWithImpl(
      _$UpdateBillingProfileInputImpl _value,
      $Res Function(_$UpdateBillingProfileInputImpl) _then)
      : super(_value, _then);

  /// Create a copy of UpdateBillingProfileInput
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? profileType = null,
    Object? legalName = null,
    Object? tradingName = null,
    Object? legalForm = null,
    Object? taxId = null,
    Object? vatNumber = null,
    Object? addressLine1 = null,
    Object? addressLine2 = null,
    Object? postalCode = null,
    Object? city = null,
    Object? country = null,
    Object? invoicingEmail = null,
  }) {
    return _then(_$UpdateBillingProfileInputImpl(
      profileType: null == profileType
          ? _value.profileType
          : profileType // ignore: cast_nullable_to_non_nullable
              as ProfileType,
      legalName: null == legalName
          ? _value.legalName
          : legalName // ignore: cast_nullable_to_non_nullable
              as String,
      tradingName: null == tradingName
          ? _value.tradingName
          : tradingName // ignore: cast_nullable_to_non_nullable
              as String,
      legalForm: null == legalForm
          ? _value.legalForm
          : legalForm // ignore: cast_nullable_to_non_nullable
              as String,
      taxId: null == taxId
          ? _value.taxId
          : taxId // ignore: cast_nullable_to_non_nullable
              as String,
      vatNumber: null == vatNumber
          ? _value.vatNumber
          : vatNumber // ignore: cast_nullable_to_non_nullable
              as String,
      addressLine1: null == addressLine1
          ? _value.addressLine1
          : addressLine1 // ignore: cast_nullable_to_non_nullable
              as String,
      addressLine2: null == addressLine2
          ? _value.addressLine2
          : addressLine2 // ignore: cast_nullable_to_non_nullable
              as String,
      postalCode: null == postalCode
          ? _value.postalCode
          : postalCode // ignore: cast_nullable_to_non_nullable
              as String,
      city: null == city
          ? _value.city
          : city // ignore: cast_nullable_to_non_nullable
              as String,
      country: null == country
          ? _value.country
          : country // ignore: cast_nullable_to_non_nullable
              as String,
      invoicingEmail: null == invoicingEmail
          ? _value.invoicingEmail
          : invoicingEmail // ignore: cast_nullable_to_non_nullable
              as String,
    ));
  }
}

/// @nodoc

class _$UpdateBillingProfileInputImpl implements _UpdateBillingProfileInput {
  const _$UpdateBillingProfileInputImpl(
      {required this.profileType,
      required this.legalName,
      required this.tradingName,
      required this.legalForm,
      required this.taxId,
      required this.vatNumber,
      required this.addressLine1,
      required this.addressLine2,
      required this.postalCode,
      required this.city,
      required this.country,
      required this.invoicingEmail});

  @override
  final ProfileType profileType;
  @override
  final String legalName;
  @override
  final String tradingName;
  @override
  final String legalForm;
  @override
  final String taxId;
  @override
  final String vatNumber;
  @override
  final String addressLine1;
  @override
  final String addressLine2;
  @override
  final String postalCode;
  @override
  final String city;
  @override
  final String country;
  @override
  final String invoicingEmail;

  @override
  String toString() {
    return 'UpdateBillingProfileInput(profileType: $profileType, legalName: $legalName, tradingName: $tradingName, legalForm: $legalForm, taxId: $taxId, vatNumber: $vatNumber, addressLine1: $addressLine1, addressLine2: $addressLine2, postalCode: $postalCode, city: $city, country: $country, invoicingEmail: $invoicingEmail)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UpdateBillingProfileInputImpl &&
            (identical(other.profileType, profileType) ||
                other.profileType == profileType) &&
            (identical(other.legalName, legalName) ||
                other.legalName == legalName) &&
            (identical(other.tradingName, tradingName) ||
                other.tradingName == tradingName) &&
            (identical(other.legalForm, legalForm) ||
                other.legalForm == legalForm) &&
            (identical(other.taxId, taxId) || other.taxId == taxId) &&
            (identical(other.vatNumber, vatNumber) ||
                other.vatNumber == vatNumber) &&
            (identical(other.addressLine1, addressLine1) ||
                other.addressLine1 == addressLine1) &&
            (identical(other.addressLine2, addressLine2) ||
                other.addressLine2 == addressLine2) &&
            (identical(other.postalCode, postalCode) ||
                other.postalCode == postalCode) &&
            (identical(other.city, city) || other.city == city) &&
            (identical(other.country, country) || other.country == country) &&
            (identical(other.invoicingEmail, invoicingEmail) ||
                other.invoicingEmail == invoicingEmail));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType,
      profileType,
      legalName,
      tradingName,
      legalForm,
      taxId,
      vatNumber,
      addressLine1,
      addressLine2,
      postalCode,
      city,
      country,
      invoicingEmail);

  /// Create a copy of UpdateBillingProfileInput
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UpdateBillingProfileInputImplCopyWith<_$UpdateBillingProfileInputImpl>
      get copyWith => __$$UpdateBillingProfileInputImplCopyWithImpl<
          _$UpdateBillingProfileInputImpl>(this, _$identity);
}

abstract class _UpdateBillingProfileInput implements UpdateBillingProfileInput {
  const factory _UpdateBillingProfileInput(
      {required final ProfileType profileType,
      required final String legalName,
      required final String tradingName,
      required final String legalForm,
      required final String taxId,
      required final String vatNumber,
      required final String addressLine1,
      required final String addressLine2,
      required final String postalCode,
      required final String city,
      required final String country,
      required final String invoicingEmail}) = _$UpdateBillingProfileInputImpl;

  @override
  ProfileType get profileType;
  @override
  String get legalName;
  @override
  String get tradingName;
  @override
  String get legalForm;
  @override
  String get taxId;
  @override
  String get vatNumber;
  @override
  String get addressLine1;
  @override
  String get addressLine2;
  @override
  String get postalCode;
  @override
  String get city;
  @override
  String get country;
  @override
  String get invoicingEmail;

  /// Create a copy of UpdateBillingProfileInput
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UpdateBillingProfileInputImplCopyWith<_$UpdateBillingProfileInputImpl>
      get copyWith => throw _privateConstructorUsedError;
}
