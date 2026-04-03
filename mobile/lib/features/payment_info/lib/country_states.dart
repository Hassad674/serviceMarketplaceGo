/// State/province option for country dropdowns.
class StateOption {
  const StateOption(this.code, this.name);
  final String code;
  final String name;
}

/// Returns true if the given field should render as a state dropdown.
bool isStateField(String labelKey, String path) {
  return labelKey == 'state' ||
      labelKey == 'businessState' ||
      path.contains('.address.state');
}

/// Returns true if the country has a known list of states/provinces.
bool hasStates(String countryCode) {
  return countryStates.containsKey(countryCode);
}

/// Returns the list of states for a country, or empty if not supported.
List<StateOption> getStatesForCountry(String countryCode) {
  return countryStates[countryCode] ?? const [];
}

/// Static map of states per country code.
///
/// Data sourced from country-region-data (same as the web app).
/// Covers: US (50+DC), AU (8), CA (13), IN (36), BR (27).
const Map<String, List<StateOption>> countryStates = {
  'US': _usStates,
  'AU': _auStates,
  'CA': _caStates,
  'IN': _inStates,
  'BR': _brStates,
  'MX': _mxStates,
  'JP': _jpStates,
  'TH': _thStates,
  'MY': _myStates,
  'SG': _sgStates,
  'GB': _gbStates,
  'IE': _ieStates,
  'NZ': _nzStates,
  'IT': _itStates,
  'ES': _esStates,
};

// ---------------------------------------------------------------------------
// United States (50 states + DC)
// ---------------------------------------------------------------------------
const _usStates = [
  StateOption('AL', 'Alabama'),
  StateOption('AK', 'Alaska'),
  StateOption('AZ', 'Arizona'),
  StateOption('AR', 'Arkansas'),
  StateOption('CA', 'California'),
  StateOption('CO', 'Colorado'),
  StateOption('CT', 'Connecticut'),
  StateOption('DE', 'Delaware'),
  StateOption('DC', 'District of Columbia'),
  StateOption('FL', 'Florida'),
  StateOption('GA', 'Georgia'),
  StateOption('HI', 'Hawaii'),
  StateOption('ID', 'Idaho'),
  StateOption('IL', 'Illinois'),
  StateOption('IN', 'Indiana'),
  StateOption('IA', 'Iowa'),
  StateOption('KS', 'Kansas'),
  StateOption('KY', 'Kentucky'),
  StateOption('LA', 'Louisiana'),
  StateOption('ME', 'Maine'),
  StateOption('MD', 'Maryland'),
  StateOption('MA', 'Massachusetts'),
  StateOption('MI', 'Michigan'),
  StateOption('MN', 'Minnesota'),
  StateOption('MS', 'Mississippi'),
  StateOption('MO', 'Missouri'),
  StateOption('MT', 'Montana'),
  StateOption('NE', 'Nebraska'),
  StateOption('NV', 'Nevada'),
  StateOption('NH', 'New Hampshire'),
  StateOption('NJ', 'New Jersey'),
  StateOption('NM', 'New Mexico'),
  StateOption('NY', 'New York'),
  StateOption('NC', 'North Carolina'),
  StateOption('ND', 'North Dakota'),
  StateOption('OH', 'Ohio'),
  StateOption('OK', 'Oklahoma'),
  StateOption('OR', 'Oregon'),
  StateOption('PA', 'Pennsylvania'),
  StateOption('RI', 'Rhode Island'),
  StateOption('SC', 'South Carolina'),
  StateOption('SD', 'South Dakota'),
  StateOption('TN', 'Tennessee'),
  StateOption('TX', 'Texas'),
  StateOption('UT', 'Utah'),
  StateOption('VT', 'Vermont'),
  StateOption('VA', 'Virginia'),
  StateOption('WA', 'Washington'),
  StateOption('WV', 'West Virginia'),
  StateOption('WI', 'Wisconsin'),
  StateOption('WY', 'Wyoming'),
];

// ---------------------------------------------------------------------------
// Australia (8 states/territories)
// ---------------------------------------------------------------------------
const _auStates = [
  StateOption('ACT', 'Australian Capital Territory'),
  StateOption('NSW', 'New South Wales'),
  StateOption('NT', 'Northern Territory'),
  StateOption('QLD', 'Queensland'),
  StateOption('SA', 'South Australia'),
  StateOption('TAS', 'Tasmania'),
  StateOption('VIC', 'Victoria'),
  StateOption('WA', 'Western Australia'),
];

// ---------------------------------------------------------------------------
// Canada (13 provinces/territories)
// ---------------------------------------------------------------------------
const _caStates = [
  StateOption('AB', 'Alberta'),
  StateOption('BC', 'British Columbia'),
  StateOption('MB', 'Manitoba'),
  StateOption('NB', 'New Brunswick'),
  StateOption('NL', 'Newfoundland and Labrador'),
  StateOption('NS', 'Nova Scotia'),
  StateOption('NT', 'Northwest Territories'),
  StateOption('NU', 'Nunavut'),
  StateOption('ON', 'Ontario'),
  StateOption('PE', 'Prince Edward Island'),
  StateOption('QC', 'Quebec'),
  StateOption('SK', 'Saskatchewan'),
  StateOption('YT', 'Yukon'),
];

// ---------------------------------------------------------------------------
// India (36 states/union territories)
// ---------------------------------------------------------------------------
const _inStates = [
  StateOption('AN', 'Andaman and Nicobar Islands'),
  StateOption('AP', 'Andhra Pradesh'),
  StateOption('AR', 'Arunachal Pradesh'),
  StateOption('AS', 'Assam'),
  StateOption('BR', 'Bihar'),
  StateOption('CH', 'Chandigarh'),
  StateOption('CT', 'Chhattisgarh'),
  StateOption('DD', 'Dadra and Nagar Haveli and Daman and Diu'),
  StateOption('DL', 'Delhi'),
  StateOption('GA', 'Goa'),
  StateOption('GJ', 'Gujarat'),
  StateOption('HR', 'Haryana'),
  StateOption('HP', 'Himachal Pradesh'),
  StateOption('JK', 'Jammu and Kashmir'),
  StateOption('JH', 'Jharkhand'),
  StateOption('KA', 'Karnataka'),
  StateOption('KL', 'Kerala'),
  StateOption('LA', 'Ladakh'),
  StateOption('LD', 'Lakshadweep'),
  StateOption('MP', 'Madhya Pradesh'),
  StateOption('MH', 'Maharashtra'),
  StateOption('MN', 'Manipur'),
  StateOption('ML', 'Meghalaya'),
  StateOption('MZ', 'Mizoram'),
  StateOption('NL', 'Nagaland'),
  StateOption('OR', 'Odisha'),
  StateOption('PY', 'Puducherry'),
  StateOption('PB', 'Punjab'),
  StateOption('RJ', 'Rajasthan'),
  StateOption('SK', 'Sikkim'),
  StateOption('TN', 'Tamil Nadu'),
  StateOption('TS', 'Telangana'),
  StateOption('TR', 'Tripura'),
  StateOption('UP', 'Uttar Pradesh'),
  StateOption('UK', 'Uttarakhand'),
  StateOption('WB', 'West Bengal'),
];

// ---------------------------------------------------------------------------
// Brazil (27 states)
// ---------------------------------------------------------------------------
const _brStates = [
  StateOption('AC', 'Acre'),
  StateOption('AL', 'Alagoas'),
  StateOption('AP', 'Amapa'),
  StateOption('AM', 'Amazonas'),
  StateOption('BA', 'Bahia'),
  StateOption('CE', 'Ceara'),
  StateOption('DF', 'Distrito Federal'),
  StateOption('ES', 'Espirito Santo'),
  StateOption('GO', 'Goias'),
  StateOption('MA', 'Maranhao'),
  StateOption('MT', 'Mato Grosso'),
  StateOption('MS', 'Mato Grosso do Sul'),
  StateOption('MG', 'Minas Gerais'),
  StateOption('PA', 'Para'),
  StateOption('PB', 'Paraiba'),
  StateOption('PR', 'Parana'),
  StateOption('PE', 'Pernambuco'),
  StateOption('PI', 'Piaui'),
  StateOption('RJ', 'Rio de Janeiro'),
  StateOption('RN', 'Rio Grande do Norte'),
  StateOption('RS', 'Rio Grande do Sul'),
  StateOption('RO', 'Rondonia'),
  StateOption('RR', 'Roraima'),
  StateOption('SC', 'Santa Catarina'),
  StateOption('SP', 'Sao Paulo'),
  StateOption('SE', 'Sergipe'),
  StateOption('TO', 'Tocantins'),
];

// ---------------------------------------------------------------------------
// Mexico (32 states)
// ---------------------------------------------------------------------------
const _mxStates = [
  StateOption('AGU', 'Aguascalientes'),
  StateOption('BCN', 'Baja California'),
  StateOption('BCS', 'Baja California Sur'),
  StateOption('CAM', 'Campeche'),
  StateOption('CHP', 'Chiapas'),
  StateOption('CHH', 'Chihuahua'),
  StateOption('COA', 'Coahuila'),
  StateOption('COL', 'Colima'),
  StateOption('CMX', 'Ciudad de Mexico'),
  StateOption('DUR', 'Durango'),
  StateOption('GUA', 'Guanajuato'),
  StateOption('GRO', 'Guerrero'),
  StateOption('HID', 'Hidalgo'),
  StateOption('JAL', 'Jalisco'),
  StateOption('MEX', 'Mexico'),
  StateOption('MIC', 'Michoacan'),
  StateOption('MOR', 'Morelos'),
  StateOption('NAY', 'Nayarit'),
  StateOption('NLE', 'Nuevo Leon'),
  StateOption('OAX', 'Oaxaca'),
  StateOption('PUE', 'Puebla'),
  StateOption('QUE', 'Queretaro'),
  StateOption('ROO', 'Quintana Roo'),
  StateOption('SLP', 'San Luis Potosi'),
  StateOption('SIN', 'Sinaloa'),
  StateOption('SON', 'Sonora'),
  StateOption('TAB', 'Tabasco'),
  StateOption('TAM', 'Tamaulipas'),
  StateOption('TLA', 'Tlaxcala'),
  StateOption('VER', 'Veracruz'),
  StateOption('YUC', 'Yucatan'),
  StateOption('ZAC', 'Zacatecas'),
];

// ---------------------------------------------------------------------------
// Japan (47 prefectures)
// ---------------------------------------------------------------------------
const _jpStates = [
  StateOption('01', 'Hokkaido'),
  StateOption('02', 'Aomori'),
  StateOption('03', 'Iwate'),
  StateOption('04', 'Miyagi'),
  StateOption('05', 'Akita'),
  StateOption('06', 'Yamagata'),
  StateOption('07', 'Fukushima'),
  StateOption('08', 'Ibaraki'),
  StateOption('09', 'Tochigi'),
  StateOption('10', 'Gunma'),
  StateOption('11', 'Saitama'),
  StateOption('12', 'Chiba'),
  StateOption('13', 'Tokyo'),
  StateOption('14', 'Kanagawa'),
  StateOption('15', 'Niigata'),
  StateOption('16', 'Toyama'),
  StateOption('17', 'Ishikawa'),
  StateOption('18', 'Fukui'),
  StateOption('19', 'Yamanashi'),
  StateOption('20', 'Nagano'),
  StateOption('21', 'Gifu'),
  StateOption('22', 'Shizuoka'),
  StateOption('23', 'Aichi'),
  StateOption('24', 'Mie'),
  StateOption('25', 'Shiga'),
  StateOption('26', 'Kyoto'),
  StateOption('27', 'Osaka'),
  StateOption('28', 'Hyogo'),
  StateOption('29', 'Nara'),
  StateOption('30', 'Wakayama'),
  StateOption('31', 'Tottori'),
  StateOption('32', 'Shimane'),
  StateOption('33', 'Okayama'),
  StateOption('34', 'Hiroshima'),
  StateOption('35', 'Yamaguchi'),
  StateOption('36', 'Tokushima'),
  StateOption('37', 'Kagawa'),
  StateOption('38', 'Ehime'),
  StateOption('39', 'Kochi'),
  StateOption('40', 'Fukuoka'),
  StateOption('41', 'Saga'),
  StateOption('42', 'Nagasaki'),
  StateOption('43', 'Kumamoto'),
  StateOption('44', 'Oita'),
  StateOption('45', 'Miyazaki'),
  StateOption('46', 'Kagoshima'),
  StateOption('47', 'Okinawa'),
];

// ---------------------------------------------------------------------------
// Thailand (77 provinces)
// ---------------------------------------------------------------------------
const _thStates = [
  StateOption('10', 'Bangkok'),
  StateOption('11', 'Samut Prakan'),
  StateOption('12', 'Nonthaburi'),
  StateOption('13', 'Pathum Thani'),
  StateOption('14', 'Phra Nakhon Si Ayutthaya'),
  StateOption('15', 'Ang Thong'),
  StateOption('16', 'Lop Buri'),
  StateOption('17', 'Sing Buri'),
  StateOption('18', 'Chai Nat'),
  StateOption('19', 'Saraburi'),
  StateOption('20', 'Chon Buri'),
  StateOption('21', 'Rayong'),
  StateOption('22', 'Chanthaburi'),
  StateOption('23', 'Trat'),
  StateOption('24', 'Chachoengsao'),
  StateOption('25', 'Prachin Buri'),
  StateOption('26', 'Nakhon Nayok'),
  StateOption('27', 'Sa Kaeo'),
  StateOption('30', 'Nakhon Ratchasima'),
  StateOption('31', 'Buri Ram'),
  StateOption('32', 'Surin'),
  StateOption('33', 'Si Sa Ket'),
  StateOption('34', 'Ubon Ratchathani'),
  StateOption('35', 'Yasothon'),
  StateOption('36', 'Chaiyaphum'),
  StateOption('37', 'Amnat Charoen'),
  StateOption('38', 'Bueng Kan'),
  StateOption('39', 'Nong Bua Lam Phu'),
  StateOption('40', 'Khon Kaen'),
  StateOption('41', 'Udon Thani'),
  StateOption('42', 'Loei'),
  StateOption('43', 'Nong Khai'),
  StateOption('44', 'Maha Sarakham'),
  StateOption('45', 'Roi Et'),
  StateOption('46', 'Kalasin'),
  StateOption('47', 'Sakon Nakhon'),
  StateOption('48', 'Nakhon Phanom'),
  StateOption('49', 'Mukdahan'),
  StateOption('50', 'Chiang Mai'),
  StateOption('51', 'Lamphun'),
  StateOption('52', 'Lampang'),
  StateOption('53', 'Uttaradit'),
  StateOption('54', 'Phrae'),
  StateOption('55', 'Nan'),
  StateOption('56', 'Phayao'),
  StateOption('57', 'Chiang Rai'),
  StateOption('58', 'Mae Hong Son'),
  StateOption('60', 'Nakhon Sawan'),
  StateOption('61', 'Uthai Thani'),
  StateOption('62', 'Kamphaeng Phet'),
  StateOption('63', 'Tak'),
  StateOption('64', 'Sukhothai'),
  StateOption('65', 'Phitsanulok'),
  StateOption('66', 'Phichit'),
  StateOption('67', 'Phetchabun'),
  StateOption('70', 'Ratchaburi'),
  StateOption('71', 'Kanchanaburi'),
  StateOption('72', 'Suphan Buri'),
  StateOption('73', 'Nakhon Pathom'),
  StateOption('74', 'Samut Sakhon'),
  StateOption('75', 'Samut Songkhram'),
  StateOption('76', 'Phetchaburi'),
  StateOption('77', 'Prachuap Khiri Khan'),
  StateOption('80', 'Nakhon Si Thammarat'),
  StateOption('81', 'Krabi'),
  StateOption('82', 'Phang Nga'),
  StateOption('83', 'Phuket'),
  StateOption('84', 'Surat Thani'),
  StateOption('85', 'Ranong'),
  StateOption('86', 'Chumphon'),
  StateOption('90', 'Songkhla'),
  StateOption('91', 'Satun'),
  StateOption('92', 'Trang'),
  StateOption('93', 'Phatthalung'),
  StateOption('94', 'Pattani'),
  StateOption('95', 'Yala'),
  StateOption('96', 'Narathiwat'),
];

// ---------------------------------------------------------------------------
// Malaysia (16 states/territories)
// ---------------------------------------------------------------------------
const _myStates = [
  StateOption('JHR', 'Johor'),
  StateOption('KDH', 'Kedah'),
  StateOption('KTN', 'Kelantan'),
  StateOption('KUL', 'Kuala Lumpur'),
  StateOption('LBN', 'Labuan'),
  StateOption('MLK', 'Malacca'),
  StateOption('NSN', 'Negeri Sembilan'),
  StateOption('PHG', 'Pahang'),
  StateOption('PNP', 'Penang'),
  StateOption('PRK', 'Perak'),
  StateOption('PLS', 'Perlis'),
  StateOption('PJY', 'Putrajaya'),
  StateOption('SBH', 'Sabah'),
  StateOption('SWK', 'Sarawak'),
  StateOption('SGR', 'Selangor'),
  StateOption('TRG', 'Terengganu'),
];

// ---------------------------------------------------------------------------
// Singapore (5 districts)
// ---------------------------------------------------------------------------
const _sgStates = [
  StateOption('01', 'Central Singapore'),
  StateOption('02', 'North East'),
  StateOption('03', 'North West'),
  StateOption('04', 'South East'),
  StateOption('05', 'South West'),
];

// ---------------------------------------------------------------------------
// United Kingdom (4 nations + Crown Dependencies)
// ---------------------------------------------------------------------------
const _gbStates = [
  StateOption('ENG', 'England'),
  StateOption('NIR', 'Northern Ireland'),
  StateOption('SCT', 'Scotland'),
  StateOption('WLS', 'Wales'),
];

// ---------------------------------------------------------------------------
// Ireland (26 counties)
// ---------------------------------------------------------------------------
const _ieStates = [
  StateOption('CW', 'Carlow'),
  StateOption('CN', 'Cavan'),
  StateOption('CE', 'Clare'),
  StateOption('CO', 'Cork'),
  StateOption('DL', 'Donegal'),
  StateOption('D', 'Dublin'),
  StateOption('G', 'Galway'),
  StateOption('KY', 'Kerry'),
  StateOption('KE', 'Kildare'),
  StateOption('KK', 'Kilkenny'),
  StateOption('LS', 'Laois'),
  StateOption('LM', 'Leitrim'),
  StateOption('LK', 'Limerick'),
  StateOption('LD', 'Longford'),
  StateOption('LH', 'Louth'),
  StateOption('MO', 'Mayo'),
  StateOption('MH', 'Meath'),
  StateOption('MN', 'Monaghan'),
  StateOption('OY', 'Offaly'),
  StateOption('RN', 'Roscommon'),
  StateOption('SO', 'Sligo'),
  StateOption('TA', 'Tipperary'),
  StateOption('WD', 'Waterford'),
  StateOption('WH', 'Westmeath'),
  StateOption('WX', 'Wexford'),
  StateOption('WW', 'Wicklow'),
];

// ---------------------------------------------------------------------------
// New Zealand (16 regions)
// ---------------------------------------------------------------------------
const _nzStates = [
  StateOption('AUK', 'Auckland'),
  StateOption('BOP', 'Bay of Plenty'),
  StateOption('CAN', 'Canterbury'),
  StateOption('GIS', 'Gisborne'),
  StateOption('HKB', 'Hawke\'s Bay'),
  StateOption('MWT', 'Manawatu-Wanganui'),
  StateOption('MBH', 'Marlborough'),
  StateOption('NSN', 'Nelson'),
  StateOption('NTL', 'Northland'),
  StateOption('OTA', 'Otago'),
  StateOption('STL', 'Southland'),
  StateOption('TKI', 'Taranaki'),
  StateOption('TAS', 'Tasman'),
  StateOption('WKO', 'Waikato'),
  StateOption('WGN', 'Wellington'),
  StateOption('WTC', 'West Coast'),
];

// ---------------------------------------------------------------------------
// Italy (20 regions)
// ---------------------------------------------------------------------------
const _itStates = [
  StateOption('65', 'Abruzzo'),
  StateOption('77', 'Basilicata'),
  StateOption('78', 'Calabria'),
  StateOption('72', 'Campania'),
  StateOption('45', 'Emilia-Romagna'),
  StateOption('36', 'Friuli-Venezia Giulia'),
  StateOption('62', 'Lazio'),
  StateOption('42', 'Liguria'),
  StateOption('25', 'Lombardy'),
  StateOption('57', 'Marche'),
  StateOption('67', 'Molise'),
  StateOption('21', 'Piedmont'),
  StateOption('75', 'Puglia'),
  StateOption('88', 'Sardinia'),
  StateOption('82', 'Sicily'),
  StateOption('52', 'Tuscany'),
  StateOption('32', 'Trentino-South Tyrol'),
  StateOption('55', 'Umbria'),
  StateOption('23', 'Aosta Valley'),
  StateOption('34', 'Veneto'),
];

// ---------------------------------------------------------------------------
// Spain (17 autonomous communities + 2 cities)
// ---------------------------------------------------------------------------
const _esStates = [
  StateOption('AN', 'Andalusia'),
  StateOption('AR', 'Aragon'),
  StateOption('AS', 'Asturias'),
  StateOption('CN', 'Canary Islands'),
  StateOption('CB', 'Cantabria'),
  StateOption('CL', 'Castile and Leon'),
  StateOption('CM', 'Castile-La Mancha'),
  StateOption('CT', 'Catalonia'),
  StateOption('CE', 'Ceuta'),
  StateOption('EX', 'Extremadura'),
  StateOption('GA', 'Galicia'),
  StateOption('IB', 'Balearic Islands'),
  StateOption('RI', 'La Rioja'),
  StateOption('MD', 'Madrid'),
  StateOption('ML', 'Melilla'),
  StateOption('MC', 'Murcia'),
  StateOption('NC', 'Navarre'),
  StateOption('PV', 'Basque Country'),
  StateOption('VC', 'Valencia'),
];
