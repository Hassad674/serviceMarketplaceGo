import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_fr.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
      : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations? of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations);
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
    delegate,
    GlobalMaterialLocalizations.delegate,
    GlobalCupertinoLocalizations.delegate,
    GlobalWidgetsLocalizations.delegate,
  ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('fr')
  ];

  /// No description provided for @appTitle.
  ///
  /// In en, this message translates to:
  /// **'Marketplace Service'**
  String get appTitle;

  /// No description provided for @signIn.
  ///
  /// In en, this message translates to:
  /// **'Sign In'**
  String get signIn;

  /// No description provided for @signUp.
  ///
  /// In en, this message translates to:
  /// **'Sign Up'**
  String get signUp;

  /// No description provided for @signOut.
  ///
  /// In en, this message translates to:
  /// **'Sign Out'**
  String get signOut;

  /// No description provided for @email.
  ///
  /// In en, this message translates to:
  /// **'Email'**
  String get email;

  /// No description provided for @emailHint.
  ///
  /// In en, this message translates to:
  /// **'you@example.com'**
  String get emailHint;

  /// No description provided for @password.
  ///
  /// In en, this message translates to:
  /// **'Password'**
  String get password;

  /// No description provided for @passwordHint.
  ///
  /// In en, this message translates to:
  /// **'Your password'**
  String get passwordHint;

  /// No description provided for @confirmPassword.
  ///
  /// In en, this message translates to:
  /// **'Confirm password'**
  String get confirmPassword;

  /// No description provided for @confirmPasswordHint.
  ///
  /// In en, this message translates to:
  /// **'Confirm your password'**
  String get confirmPasswordHint;

  /// No description provided for @passwordRequirements.
  ///
  /// In en, this message translates to:
  /// **'Minimum 8 characters with uppercase, lowercase and digit'**
  String get passwordRequirements;

  /// No description provided for @forgotPassword.
  ///
  /// In en, this message translates to:
  /// **'Forgot password?'**
  String get forgotPassword;

  /// No description provided for @noAccount.
  ///
  /// In en, this message translates to:
  /// **'No account yet?'**
  String get noAccount;

  /// No description provided for @alreadyRegistered.
  ///
  /// In en, this message translates to:
  /// **'Already registered?'**
  String get alreadyRegistered;

  /// No description provided for @changeProfile.
  ///
  /// In en, this message translates to:
  /// **'Change profile'**
  String get changeProfile;

  /// No description provided for @signingIn.
  ///
  /// In en, this message translates to:
  /// **'Signing in...'**
  String get signingIn;

  /// No description provided for @signingUp.
  ///
  /// In en, this message translates to:
  /// **'Signing up...'**
  String get signingUp;

  /// No description provided for @agencyName.
  ///
  /// In en, this message translates to:
  /// **'Agency name'**
  String get agencyName;

  /// No description provided for @agencyNameHint.
  ///
  /// In en, this message translates to:
  /// **'Commercial name of your agency'**
  String get agencyNameHint;

  /// No description provided for @companyName.
  ///
  /// In en, this message translates to:
  /// **'Company name'**
  String get companyName;

  /// No description provided for @companyNameHint.
  ///
  /// In en, this message translates to:
  /// **'Name of your company'**
  String get companyNameHint;

  /// No description provided for @firstName.
  ///
  /// In en, this message translates to:
  /// **'First name'**
  String get firstName;

  /// No description provided for @firstNameHint.
  ///
  /// In en, this message translates to:
  /// **'John'**
  String get firstNameHint;

  /// No description provided for @lastName.
  ///
  /// In en, this message translates to:
  /// **'Last name'**
  String get lastName;

  /// No description provided for @lastNameHint.
  ///
  /// In en, this message translates to:
  /// **'Doe'**
  String get lastNameHint;

  /// No description provided for @createAgencyAccount.
  ///
  /// In en, this message translates to:
  /// **'Create my agency account'**
  String get createAgencyAccount;

  /// No description provided for @createFreelanceAccount.
  ///
  /// In en, this message translates to:
  /// **'Create my freelance account'**
  String get createFreelanceAccount;

  /// No description provided for @createEnterpriseAccount.
  ///
  /// In en, this message translates to:
  /// **'Create my enterprise account'**
  String get createEnterpriseAccount;

  /// No description provided for @roleSelectionTitle.
  ///
  /// In en, this message translates to:
  /// **'Join the marketplace'**
  String get roleSelectionTitle;

  /// No description provided for @roleSelectionSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Choose your professional profile'**
  String get roleSelectionSubtitle;

  /// No description provided for @roleAgency.
  ///
  /// In en, this message translates to:
  /// **'Agency'**
  String get roleAgency;

  /// No description provided for @roleAgencyDesc.
  ///
  /// In en, this message translates to:
  /// **'Manage your missions, your team and your visibility.'**
  String get roleAgencyDesc;

  /// No description provided for @roleFreelance.
  ///
  /// In en, this message translates to:
  /// **'Freelance / Business Referrer'**
  String get roleFreelance;

  /// No description provided for @roleFreelanceDesc.
  ///
  /// In en, this message translates to:
  /// **'Manage your missions and grow your activity.'**
  String get roleFreelanceDesc;

  /// No description provided for @roleEnterprise.
  ///
  /// In en, this message translates to:
  /// **'Enterprise'**
  String get roleEnterprise;

  /// No description provided for @roleEnterpriseDesc.
  ///
  /// In en, this message translates to:
  /// **'Find the best providers for your projects.'**
  String get roleEnterpriseDesc;

  /// No description provided for @welcomeBack.
  ///
  /// In en, this message translates to:
  /// **'Welcome back,'**
  String get welcomeBack;

  /// No description provided for @dashboard.
  ///
  /// In en, this message translates to:
  /// **'Dashboard'**
  String get dashboard;

  /// No description provided for @home.
  ///
  /// In en, this message translates to:
  /// **'Home'**
  String get home;

  /// No description provided for @messages.
  ///
  /// In en, this message translates to:
  /// **'Messages'**
  String get messages;

  /// No description provided for @missions.
  ///
  /// In en, this message translates to:
  /// **'Missions'**
  String get missions;

  /// No description provided for @profile.
  ///
  /// In en, this message translates to:
  /// **'Profile'**
  String get profile;

  /// No description provided for @myProfile.
  ///
  /// In en, this message translates to:
  /// **'My Profile'**
  String get myProfile;

  /// No description provided for @settings.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settings;

  /// No description provided for @activeMissions.
  ///
  /// In en, this message translates to:
  /// **'Active Missions'**
  String get activeMissions;

  /// No description provided for @activeContracts.
  ///
  /// In en, this message translates to:
  /// **'Active contracts'**
  String get activeContracts;

  /// No description provided for @unreadMessages.
  ///
  /// In en, this message translates to:
  /// **'Unread Messages'**
  String get unreadMessages;

  /// No description provided for @conversations.
  ///
  /// In en, this message translates to:
  /// **'Conversations'**
  String get conversations;

  /// No description provided for @monthlyRevenue.
  ///
  /// In en, this message translates to:
  /// **'Monthly Revenue'**
  String get monthlyRevenue;

  /// No description provided for @thisMonth.
  ///
  /// In en, this message translates to:
  /// **'This month'**
  String get thisMonth;

  /// No description provided for @activeProjects.
  ///
  /// In en, this message translates to:
  /// **'Active Projects'**
  String get activeProjects;

  /// No description provided for @totalBudget.
  ///
  /// In en, this message translates to:
  /// **'Total Budget'**
  String get totalBudget;

  /// No description provided for @spentThisMonth.
  ///
  /// In en, this message translates to:
  /// **'Spent this month'**
  String get spentThisMonth;

  /// No description provided for @referrals.
  ///
  /// In en, this message translates to:
  /// **'Referrals'**
  String get referrals;

  /// No description provided for @pendingResponse.
  ///
  /// In en, this message translates to:
  /// **'Pending response'**
  String get pendingResponse;

  /// No description provided for @completedMissions.
  ///
  /// In en, this message translates to:
  /// **'Completed Missions'**
  String get completedMissions;

  /// No description provided for @totalHistory.
  ///
  /// In en, this message translates to:
  /// **'Total history'**
  String get totalHistory;

  /// No description provided for @commissions.
  ///
  /// In en, this message translates to:
  /// **'Commissions'**
  String get commissions;

  /// No description provided for @totalEarned.
  ///
  /// In en, this message translates to:
  /// **'Total earned'**
  String get totalEarned;

  /// No description provided for @businessReferrerMode.
  ///
  /// In en, this message translates to:
  /// **'Business Referrer Mode'**
  String get businessReferrerMode;

  /// No description provided for @freelanceDashboard.
  ///
  /// In en, this message translates to:
  /// **'Freelance Dashboard'**
  String get freelanceDashboard;

  /// No description provided for @referrerMode.
  ///
  /// In en, this message translates to:
  /// **'Referrer Mode'**
  String get referrerMode;

  /// No description provided for @presentationVideo.
  ///
  /// In en, this message translates to:
  /// **'Presentation Video'**
  String get presentationVideo;

  /// No description provided for @noVideo.
  ///
  /// In en, this message translates to:
  /// **'No presentation video'**
  String get noVideo;

  /// No description provided for @addVideo.
  ///
  /// In en, this message translates to:
  /// **'Add a video'**
  String get addVideo;

  /// No description provided for @videoUpdated.
  ///
  /// In en, this message translates to:
  /// **'Video updated'**
  String get videoUpdated;

  /// No description provided for @photoUpdated.
  ///
  /// In en, this message translates to:
  /// **'Photo updated'**
  String get photoUpdated;

  /// No description provided for @addPhoto.
  ///
  /// In en, this message translates to:
  /// **'Add a photo'**
  String get addPhoto;

  /// No description provided for @takePhoto.
  ///
  /// In en, this message translates to:
  /// **'Take a photo'**
  String get takePhoto;

  /// No description provided for @chooseFromGallery.
  ///
  /// In en, this message translates to:
  /// **'Choose from gallery'**
  String get chooseFromGallery;

  /// No description provided for @chooseFile.
  ///
  /// In en, this message translates to:
  /// **'Choose a file'**
  String get chooseFile;

  /// No description provided for @upload.
  ///
  /// In en, this message translates to:
  /// **'Upload'**
  String get upload;

  /// No description provided for @cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// No description provided for @save.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get save;

  /// No description provided for @fileTooLarge.
  ///
  /// In en, this message translates to:
  /// **'File too large'**
  String get fileTooLarge;

  /// No description provided for @uploadError.
  ///
  /// In en, this message translates to:
  /// **'Upload failed'**
  String get uploadError;

  /// No description provided for @maxSize.
  ///
  /// In en, this message translates to:
  /// **'Maximum size: {size}'**
  String maxSize(String size);

  /// No description provided for @about.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get about;

  /// No description provided for @professionalTitle.
  ///
  /// In en, this message translates to:
  /// **'Professional Title'**
  String get professionalTitle;

  /// No description provided for @noTitle.
  ///
  /// In en, this message translates to:
  /// **'No title added'**
  String get noTitle;

  /// No description provided for @unexpectedError.
  ///
  /// In en, this message translates to:
  /// **'An unexpected error occurred'**
  String get unexpectedError;

  /// No description provided for @connectionError.
  ///
  /// In en, this message translates to:
  /// **'Connection error. Check your internet.'**
  String get connectionError;

  /// No description provided for @timeoutError.
  ///
  /// In en, this message translates to:
  /// **'Request timed out. Try again.'**
  String get timeoutError;

  /// No description provided for @serverError.
  ///
  /// In en, this message translates to:
  /// **'Server error. Try again later.'**
  String get serverError;

  /// No description provided for @comingSoon.
  ///
  /// In en, this message translates to:
  /// **'Coming soon'**
  String get comingSoon;

  /// No description provided for @fieldRequired.
  ///
  /// In en, this message translates to:
  /// **'This field is required'**
  String get fieldRequired;

  /// No description provided for @invalidEmail.
  ///
  /// In en, this message translates to:
  /// **'Invalid email address'**
  String get invalidEmail;

  /// No description provided for @passwordTooShort.
  ///
  /// In en, this message translates to:
  /// **'Minimum 8 characters'**
  String get passwordTooShort;

  /// No description provided for @passwordNoUppercase.
  ///
  /// In en, this message translates to:
  /// **'At least one uppercase letter'**
  String get passwordNoUppercase;

  /// No description provided for @passwordNoLowercase.
  ///
  /// In en, this message translates to:
  /// **'At least one lowercase letter'**
  String get passwordNoLowercase;

  /// No description provided for @passwordNoDigit.
  ///
  /// In en, this message translates to:
  /// **'At least one digit'**
  String get passwordNoDigit;

  /// No description provided for @passwordsDoNotMatch.
  ///
  /// In en, this message translates to:
  /// **'Passwords do not match'**
  String get passwordsDoNotMatch;

  /// No description provided for @search.
  ///
  /// In en, this message translates to:
  /// **'Search'**
  String get search;

  /// No description provided for @findFreelancers.
  ///
  /// In en, this message translates to:
  /// **'Find Freelancers'**
  String get findFreelancers;

  /// No description provided for @findAgencies.
  ///
  /// In en, this message translates to:
  /// **'Find Agencies'**
  String get findAgencies;

  /// No description provided for @findReferrers.
  ///
  /// In en, this message translates to:
  /// **'Find Referrers'**
  String get findReferrers;

  /// No description provided for @noProfilesFound.
  ///
  /// In en, this message translates to:
  /// **'No profiles found'**
  String get noProfilesFound;

  /// No description provided for @searchTryAgain.
  ///
  /// In en, this message translates to:
  /// **'Try again later or adjust your search.'**
  String get searchTryAgain;

  /// No description provided for @couldNotLoadProfiles.
  ///
  /// In en, this message translates to:
  /// **'Could not load profiles. Check your connection.'**
  String get couldNotLoadProfiles;

  /// No description provided for @couldNotLoadProfile.
  ///
  /// In en, this message translates to:
  /// **'Could not load profile'**
  String get couldNotLoadProfile;

  /// No description provided for @checkConnectionRetry.
  ///
  /// In en, this message translates to:
  /// **'Check your connection and try again.'**
  String get checkConnectionRetry;

  /// No description provided for @somethingWentWrong.
  ///
  /// In en, this message translates to:
  /// **'Something went wrong'**
  String get somethingWentWrong;

  /// No description provided for @retry.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get retry;

  /// No description provided for @tapToPlay.
  ///
  /// In en, this message translates to:
  /// **'Tap to play'**
  String get tapToPlay;

  /// No description provided for @replaceVideo.
  ///
  /// In en, this message translates to:
  /// **'Replace video'**
  String get replaceVideo;

  /// No description provided for @removeVideo.
  ///
  /// In en, this message translates to:
  /// **'Remove video'**
  String get removeVideo;

  /// No description provided for @removeVideoConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Remove video'**
  String get removeVideoConfirmTitle;

  /// No description provided for @removeVideoConfirmMessage.
  ///
  /// In en, this message translates to:
  /// **'Are you sure you want to remove your presentation video?'**
  String get removeVideoConfirmMessage;

  /// No description provided for @remove.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get remove;

  /// No description provided for @darkMode.
  ///
  /// In en, this message translates to:
  /// **'Dark Mode'**
  String get darkMode;

  /// No description provided for @aboutPlaceholder.
  ///
  /// In en, this message translates to:
  /// **'Tell others about yourself and your expertise'**
  String get aboutPlaceholder;

  /// No description provided for @aboutEditHint.
  ///
  /// In en, this message translates to:
  /// **'Tell others about yourself...'**
  String get aboutEditHint;

  /// No description provided for @aboutUpdated.
  ///
  /// In en, this message translates to:
  /// **'About updated'**
  String get aboutUpdated;

  /// No description provided for @titlePlaceholder.
  ///
  /// In en, this message translates to:
  /// **'Add your professional title'**
  String get titlePlaceholder;

  /// No description provided for @videoRemoved.
  ///
  /// In en, this message translates to:
  /// **'Video removed'**
  String get videoRemoved;

  /// No description provided for @couldNotOpenVideo.
  ///
  /// In en, this message translates to:
  /// **'Could not open video'**
  String get couldNotOpenVideo;

  /// No description provided for @messagingSearchHint.
  ///
  /// In en, this message translates to:
  /// **'Search conversations...'**
  String get messagingSearchHint;

  /// No description provided for @messagingNoMessages.
  ///
  /// In en, this message translates to:
  /// **'No messages in this conversation'**
  String get messagingNoMessages;

  /// No description provided for @messagingNoConversations.
  ///
  /// In en, this message translates to:
  /// **'No conversations yet'**
  String get messagingNoConversations;

  /// No description provided for @messagingWriteMessage.
  ///
  /// In en, this message translates to:
  /// **'Write your message...'**
  String get messagingWriteMessage;

  /// No description provided for @messagingOnline.
  ///
  /// In en, this message translates to:
  /// **'Online'**
  String get messagingOnline;

  /// No description provided for @messagingOffline.
  ///
  /// In en, this message translates to:
  /// **'Offline'**
  String get messagingOffline;

  /// No description provided for @messagingAllRoles.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get messagingAllRoles;

  /// No description provided for @messagingAgency.
  ///
  /// In en, this message translates to:
  /// **'Agency'**
  String get messagingAgency;

  /// No description provided for @messagingFreelancer.
  ///
  /// In en, this message translates to:
  /// **'Freelance/Referrer'**
  String get messagingFreelancer;

  /// No description provided for @messagingEnterprise.
  ///
  /// In en, this message translates to:
  /// **'Enterprise'**
  String get messagingEnterprise;

  /// No description provided for @messagingConversationNotFound.
  ///
  /// In en, this message translates to:
  /// **'Conversation not found'**
  String get messagingConversationNotFound;

  /// No description provided for @messagingSendMessage.
  ///
  /// In en, this message translates to:
  /// **'Send a message'**
  String get messagingSendMessage;

  /// No description provided for @messagingTyping.
  ///
  /// In en, this message translates to:
  /// **'{name} is typing...'**
  String messagingTyping(String name);

  /// No description provided for @messagingEdited.
  ///
  /// In en, this message translates to:
  /// **'edited'**
  String get messagingEdited;

  /// No description provided for @messagingDeleted.
  ///
  /// In en, this message translates to:
  /// **'This message was deleted'**
  String get messagingDeleted;

  /// No description provided for @messagingDelivered.
  ///
  /// In en, this message translates to:
  /// **'Delivered'**
  String get messagingDelivered;

  /// No description provided for @messagingRead.
  ///
  /// In en, this message translates to:
  /// **'Read'**
  String get messagingRead;

  /// No description provided for @messagingSent.
  ///
  /// In en, this message translates to:
  /// **'Sent'**
  String get messagingSent;

  /// No description provided for @messagingSending.
  ///
  /// In en, this message translates to:
  /// **'Sending...'**
  String get messagingSending;

  /// No description provided for @messagingReconnecting.
  ///
  /// In en, this message translates to:
  /// **'Reconnecting...'**
  String get messagingReconnecting;

  /// No description provided for @messagingEditMessage.
  ///
  /// In en, this message translates to:
  /// **'Edit message'**
  String get messagingEditMessage;

  /// No description provided for @messagingDeleteMessage.
  ///
  /// In en, this message translates to:
  /// **'Delete message'**
  String get messagingDeleteMessage;

  /// No description provided for @messagingDeleteConfirm.
  ///
  /// In en, this message translates to:
  /// **'Are you sure you want to delete this message?'**
  String get messagingDeleteConfirm;

  /// No description provided for @messagingFileUpload.
  ///
  /// In en, this message translates to:
  /// **'Send a file'**
  String get messagingFileUpload;

  /// No description provided for @messagingStartConversation.
  ///
  /// In en, this message translates to:
  /// **'No messages yet. Start the conversation!'**
  String get messagingStartConversation;

  /// No description provided for @messagingLoadMore.
  ///
  /// In en, this message translates to:
  /// **'Load more'**
  String get messagingLoadMore;

  /// No description provided for @projects.
  ///
  /// In en, this message translates to:
  /// **'Projects'**
  String get projects;

  /// No description provided for @createProject.
  ///
  /// In en, this message translates to:
  /// **'Create Project'**
  String get createProject;

  /// No description provided for @noProjects.
  ///
  /// In en, this message translates to:
  /// **'No projects yet'**
  String get noProjects;

  /// No description provided for @noProjectsDesc.
  ///
  /// In en, this message translates to:
  /// **'Create your first project to get started.'**
  String get noProjectsDesc;

  /// No description provided for @paymentType.
  ///
  /// In en, this message translates to:
  /// **'Payment type'**
  String get paymentType;

  /// No description provided for @invoiceBilling.
  ///
  /// In en, this message translates to:
  /// **'Invoice billing'**
  String get invoiceBilling;

  /// No description provided for @invoiceBillingDesc.
  ///
  /// In en, this message translates to:
  /// **'Classic invoicing with flexible billing cycles.'**
  String get invoiceBillingDesc;

  /// No description provided for @escrowPayments.
  ///
  /// In en, this message translates to:
  /// **'Escrow payments'**
  String get escrowPayments;

  /// No description provided for @escrowPaymentsDesc.
  ///
  /// In en, this message translates to:
  /// **'Funds held securely until milestones are approved.'**
  String get escrowPaymentsDesc;

  /// No description provided for @projectStructure.
  ///
  /// In en, this message translates to:
  /// **'Structure'**
  String get projectStructure;

  /// No description provided for @milestone.
  ///
  /// In en, this message translates to:
  /// **'Milestone'**
  String get milestone;

  /// No description provided for @oneTime.
  ///
  /// In en, this message translates to:
  /// **'One-time'**
  String get oneTime;

  /// No description provided for @billingDetails.
  ///
  /// In en, this message translates to:
  /// **'Billing details'**
  String get billingDetails;

  /// No description provided for @fixed.
  ///
  /// In en, this message translates to:
  /// **'Fixed'**
  String get fixed;

  /// No description provided for @hourly.
  ///
  /// In en, this message translates to:
  /// **'Hourly'**
  String get hourly;

  /// No description provided for @rate.
  ///
  /// In en, this message translates to:
  /// **'Rate'**
  String get rate;

  /// No description provided for @frequency.
  ///
  /// In en, this message translates to:
  /// **'Frequency'**
  String get frequency;

  /// No description provided for @weekly.
  ///
  /// In en, this message translates to:
  /// **'Weekly'**
  String get weekly;

  /// No description provided for @biWeekly.
  ///
  /// In en, this message translates to:
  /// **'Bi-weekly'**
  String get biWeekly;

  /// No description provided for @monthly.
  ///
  /// In en, this message translates to:
  /// **'Monthly'**
  String get monthly;

  /// No description provided for @projectDetails.
  ///
  /// In en, this message translates to:
  /// **'Details'**
  String get projectDetails;

  /// No description provided for @projectTitle.
  ///
  /// In en, this message translates to:
  /// **'Project title'**
  String get projectTitle;

  /// No description provided for @projectDescription.
  ///
  /// In en, this message translates to:
  /// **'Description'**
  String get projectDescription;

  /// No description provided for @requiredSkills.
  ///
  /// In en, this message translates to:
  /// **'Required skills'**
  String get requiredSkills;

  /// No description provided for @addSkillHint.
  ///
  /// In en, this message translates to:
  /// **'Type a skill and press add'**
  String get addSkillHint;

  /// No description provided for @timeline.
  ///
  /// In en, this message translates to:
  /// **'Timeline'**
  String get timeline;

  /// No description provided for @startDate.
  ///
  /// In en, this message translates to:
  /// **'Start date'**
  String get startDate;

  /// No description provided for @deadline.
  ///
  /// In en, this message translates to:
  /// **'Deadline'**
  String get deadline;

  /// No description provided for @ongoing.
  ///
  /// In en, this message translates to:
  /// **'Ongoing'**
  String get ongoing;

  /// No description provided for @whoCanApply.
  ///
  /// In en, this message translates to:
  /// **'Who can apply'**
  String get whoCanApply;

  /// No description provided for @freelancersAndAgencies.
  ///
  /// In en, this message translates to:
  /// **'Freelancers & Agencies'**
  String get freelancersAndAgencies;

  /// No description provided for @freelancersOnly.
  ///
  /// In en, this message translates to:
  /// **'Freelancers only'**
  String get freelancersOnly;

  /// No description provided for @agenciesOnly.
  ///
  /// In en, this message translates to:
  /// **'Agencies only'**
  String get agenciesOnly;

  /// No description provided for @negotiable.
  ///
  /// In en, this message translates to:
  /// **'Budget is negotiable'**
  String get negotiable;

  /// No description provided for @milestoneTitle.
  ///
  /// In en, this message translates to:
  /// **'Title'**
  String get milestoneTitle;

  /// No description provided for @milestoneDescription.
  ///
  /// In en, this message translates to:
  /// **'Deliverables'**
  String get milestoneDescription;

  /// No description provided for @milestoneAmount.
  ///
  /// In en, this message translates to:
  /// **'Amount'**
  String get milestoneAmount;

  /// No description provided for @totalAmount.
  ///
  /// In en, this message translates to:
  /// **'Total amount'**
  String get totalAmount;

  /// No description provided for @addMilestone.
  ///
  /// In en, this message translates to:
  /// **'Add milestone'**
  String get addMilestone;

  /// No description provided for @publishProject.
  ///
  /// In en, this message translates to:
  /// **'Publish project'**
  String get publishProject;

  /// No description provided for @projectPublished.
  ///
  /// In en, this message translates to:
  /// **'Project published successfully'**
  String get projectPublished;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'fr'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'fr':
      return AppLocalizationsFr();
  }

  throw FlutterError(
      'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
      'an issue with the localizations generation tool. Please file an issue '
      'on GitHub with a reproducible sample app and the gen-l10n configuration '
      'that was used.');
}
