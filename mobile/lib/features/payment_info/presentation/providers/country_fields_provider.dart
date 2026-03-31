import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/country_field_spec.dart';
import 'payment_info_provider.dart';

/// Key for the country fields family provider.
class CountryFieldsKey {
  const CountryFieldsKey(this.country, this.businessType);

  final String country;
  final String businessType;

  @override
  bool operator ==(Object other) =>
      other is CountryFieldsKey &&
      other.country == country &&
      other.businessType == businessType;

  @override
  int get hashCode => Object.hash(country, businessType);
}

/// Fetches country-specific field requirements.
final countryFieldsProvider =
    FutureProvider.family<CountryFieldsResponse, CountryFieldsKey>(
  (ref, key) async {
    final repo = ref.watch(paymentInfoRepositoryProvider);
    return repo.getCountryFields(key.country, key.businessType);
  },
);
