// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'Marketplace Service';

  @override
  String get signIn => 'Sign In';

  @override
  String get signUp => 'Sign Up';

  @override
  String get signOut => 'Sign Out';

  @override
  String get email => 'Email';

  @override
  String get emailHint => 'you@example.com';

  @override
  String get password => 'Password';

  @override
  String get passwordHint => 'Your password';

  @override
  String get confirmPassword => 'Confirm password';

  @override
  String get confirmPasswordHint => 'Confirm your password';

  @override
  String get passwordRequirements =>
      'Minimum 8 characters with uppercase, lowercase and digit';

  @override
  String get forgotPassword => 'Forgot password?';

  @override
  String get noAccount => 'No account yet?';

  @override
  String get alreadyRegistered => 'Already registered?';

  @override
  String get changeProfile => 'Change profile';

  @override
  String get signingIn => 'Signing in...';

  @override
  String get signingUp => 'Signing up...';

  @override
  String get agencyName => 'Agency name';

  @override
  String get agencyNameHint => 'Commercial name of your agency';

  @override
  String get companyName => 'Company name';

  @override
  String get companyNameHint => 'Name of your company';

  @override
  String get firstName => 'First name';

  @override
  String get firstNameHint => 'John';

  @override
  String get lastName => 'Last name';

  @override
  String get lastNameHint => 'Doe';

  @override
  String get createAgencyAccount => 'Create my agency account';

  @override
  String get createFreelanceAccount => 'Create my freelance account';

  @override
  String get createEnterpriseAccount => 'Create my enterprise account';

  @override
  String get roleSelectionTitle => 'Join the marketplace';

  @override
  String get roleSelectionSubtitle => 'Choose your professional profile';

  @override
  String get roleAgency => 'Agency';

  @override
  String get roleAgencyDesc =>
      'Manage your missions, your team and your visibility.';

  @override
  String get roleFreelance => 'Freelance / Business Referrer';

  @override
  String get roleFreelanceDesc =>
      'Manage your missions and grow your activity.';

  @override
  String get roleEnterprise => 'Enterprise';

  @override
  String get roleEnterpriseDesc => 'Find the best providers for your projects.';

  @override
  String get welcomeBack => 'Welcome back,';

  @override
  String get dashboard => 'Dashboard';

  @override
  String get home => 'Home';

  @override
  String get messages => 'Messages';

  @override
  String get missions => 'Missions';

  @override
  String get profile => 'Profile';

  @override
  String get myProfile => 'My Profile';

  @override
  String get settings => 'Settings';

  @override
  String get activeMissions => 'Active Missions';

  @override
  String get activeContracts => 'Active contracts';

  @override
  String get unreadMessages => 'Unread Messages';

  @override
  String get conversations => 'Conversations';

  @override
  String get monthlyRevenue => 'Monthly Revenue';

  @override
  String get thisMonth => 'This month';

  @override
  String get activeProjects => 'Active Projects';

  @override
  String get totalBudget => 'Total Budget';

  @override
  String get spentThisMonth => 'Spent this month';

  @override
  String get referrals => 'Referrals';

  @override
  String get pendingResponse => 'Pending response';

  @override
  String get completedMissions => 'Completed Missions';

  @override
  String get totalHistory => 'Total history';

  @override
  String get commissions => 'Commissions';

  @override
  String get totalEarned => 'Total earned';

  @override
  String get businessReferrerMode => 'Business Referrer Mode';

  @override
  String get freelanceDashboard => 'Freelance Dashboard';

  @override
  String get referrerMode => 'Referrer Mode';

  @override
  String get presentationVideo => 'Presentation Video';

  @override
  String get noVideo => 'No presentation video';

  @override
  String get addVideo => 'Add a video';

  @override
  String get videoUpdated => 'Video updated';

  @override
  String get photoUpdated => 'Photo updated';

  @override
  String get addPhoto => 'Add a photo';

  @override
  String get takePhoto => 'Take a photo';

  @override
  String get chooseFromGallery => 'Choose from gallery';

  @override
  String get chooseFile => 'Choose a file';

  @override
  String get upload => 'Upload';

  @override
  String get cancel => 'Cancel';

  @override
  String get save => 'Save';

  @override
  String get fileTooLarge => 'File too large';

  @override
  String get uploadError => 'Upload failed';

  @override
  String maxSize(String size) {
    return 'Maximum size: $size';
  }

  @override
  String get about => 'About';

  @override
  String get professionalTitle => 'Professional Title';

  @override
  String get noTitle => 'No title added';

  @override
  String get unexpectedError => 'An unexpected error occurred';

  @override
  String get connectionError => 'Connection error. Check your internet.';

  @override
  String get timeoutError => 'Request timed out. Try again.';

  @override
  String get serverError => 'Server error. Try again later.';

  @override
  String get comingSoon => 'Coming soon';

  @override
  String get fieldRequired => 'This field is required';

  @override
  String get invalidEmail => 'Invalid email address';

  @override
  String get passwordTooShort => 'Minimum 8 characters';

  @override
  String get passwordNoUppercase => 'At least one uppercase letter';

  @override
  String get passwordNoLowercase => 'At least one lowercase letter';

  @override
  String get passwordNoDigit => 'At least one digit';

  @override
  String get passwordsDoNotMatch => 'Passwords do not match';
}
