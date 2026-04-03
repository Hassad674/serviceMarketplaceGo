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
    Locale('fr'),
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

  /// No description provided for @messagingTypingShort.
  ///
  /// In en, this message translates to:
  /// **'typing...'**
  String get messagingTypingShort;

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

  /// No description provided for @messagingVoiceMessage.
  ///
  /// In en, this message translates to:
  /// **'Voice message'**
  String get messagingVoiceMessage;

  /// No description provided for @messagingRecording.
  ///
  /// In en, this message translates to:
  /// **'Recording...'**
  String get messagingRecording;

  /// No description provided for @messagingCancelRecording.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get messagingCancelRecording;

  /// No description provided for @messagingMicrophonePermission.
  ///
  /// In en, this message translates to:
  /// **'Microphone access required'**
  String get messagingMicrophonePermission;

  /// No description provided for @messagingReply.
  ///
  /// In en, this message translates to:
  /// **'Reply'**
  String get messagingReply;

  /// No description provided for @messagingReplyingTo.
  ///
  /// In en, this message translates to:
  /// **'Replying to {name}'**
  String messagingReplyingTo(String name);

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

  /// No description provided for @jobCreateJob.
  ///
  /// In en, this message translates to:
  /// **'Create job'**
  String get jobCreateJob;

  /// No description provided for @jobDetails.
  ///
  /// In en, this message translates to:
  /// **'Job details'**
  String get jobDetails;

  /// No description provided for @jobBudgetAndDuration.
  ///
  /// In en, this message translates to:
  /// **'Budget and duration'**
  String get jobBudgetAndDuration;

  /// No description provided for @jobTitle.
  ///
  /// In en, this message translates to:
  /// **'Job title'**
  String get jobTitle;

  /// No description provided for @jobTitleHint.
  ///
  /// In en, this message translates to:
  /// **'Add a descriptive title'**
  String get jobTitleHint;

  /// No description provided for @jobDescription.
  ///
  /// In en, this message translates to:
  /// **'Job description'**
  String get jobDescription;

  /// No description provided for @jobSkills.
  ///
  /// In en, this message translates to:
  /// **'Skills'**
  String get jobSkills;

  /// No description provided for @jobSkillsHint.
  ///
  /// In en, this message translates to:
  /// **'ex. UX Design, Web Development'**
  String get jobSkillsHint;

  /// No description provided for @jobTools.
  ///
  /// In en, this message translates to:
  /// **'Tools'**
  String get jobTools;

  /// No description provided for @jobToolsHint.
  ///
  /// In en, this message translates to:
  /// **'ex. Figma, Canva, Webflow'**
  String get jobToolsHint;

  /// No description provided for @jobContractorCount.
  ///
  /// In en, this message translates to:
  /// **'How many contractors?'**
  String get jobContractorCount;

  /// No description provided for @jobApplicantType.
  ///
  /// In en, this message translates to:
  /// **'Who can apply?'**
  String get jobApplicantType;

  /// No description provided for @jobApplicantAll.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get jobApplicantAll;

  /// No description provided for @jobApplicantFreelancers.
  ///
  /// In en, this message translates to:
  /// **'Freelancers'**
  String get jobApplicantFreelancers;

  /// No description provided for @jobApplicantAgencies.
  ///
  /// In en, this message translates to:
  /// **'Agencies'**
  String get jobApplicantAgencies;

  /// No description provided for @jobBudgetType.
  ///
  /// In en, this message translates to:
  /// **'Project type'**
  String get jobBudgetType;

  /// No description provided for @jobOngoing.
  ///
  /// In en, this message translates to:
  /// **'Ongoing'**
  String get jobOngoing;

  /// No description provided for @jobOneTime.
  ///
  /// In en, this message translates to:
  /// **'One-time'**
  String get jobOneTime;

  /// No description provided for @jobPaymentFrequency.
  ///
  /// In en, this message translates to:
  /// **'Payment frequency'**
  String get jobPaymentFrequency;

  /// No description provided for @jobHourly.
  ///
  /// In en, this message translates to:
  /// **'Hourly'**
  String get jobHourly;

  /// No description provided for @jobWeekly.
  ///
  /// In en, this message translates to:
  /// **'Weekly'**
  String get jobWeekly;

  /// No description provided for @jobMonthly.
  ///
  /// In en, this message translates to:
  /// **'Monthly'**
  String get jobMonthly;

  /// No description provided for @jobMinRate.
  ///
  /// In en, this message translates to:
  /// **'Min. rate'**
  String get jobMinRate;

  /// No description provided for @jobMaxRate.
  ///
  /// In en, this message translates to:
  /// **'Max. rate'**
  String get jobMaxRate;

  /// No description provided for @jobMinBudget.
  ///
  /// In en, this message translates to:
  /// **'Min. budget'**
  String get jobMinBudget;

  /// No description provided for @jobMaxBudget.
  ///
  /// In en, this message translates to:
  /// **'Max. budget'**
  String get jobMaxBudget;

  /// No description provided for @jobMaxHours.
  ///
  /// In en, this message translates to:
  /// **'Max. hours/week'**
  String get jobMaxHours;

  /// No description provided for @jobEstimatedDuration.
  ///
  /// In en, this message translates to:
  /// **'Estimated duration'**
  String get jobEstimatedDuration;

  /// No description provided for @jobIndefinite.
  ///
  /// In en, this message translates to:
  /// **'Indefinite duration'**
  String get jobIndefinite;

  /// No description provided for @jobWeeks.
  ///
  /// In en, this message translates to:
  /// **'weeks'**
  String get jobWeeks;

  /// No description provided for @jobMonths.
  ///
  /// In en, this message translates to:
  /// **'months'**
  String get jobMonths;

  /// No description provided for @jobCancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get jobCancel;

  /// No description provided for @jobContinue.
  ///
  /// In en, this message translates to:
  /// **'Continue'**
  String get jobContinue;

  /// No description provided for @jobSave.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get jobSave;

  /// No description provided for @jobPublish.
  ///
  /// In en, this message translates to:
  /// **'Publish'**
  String get jobPublish;

  /// No description provided for @jobMyJobs.
  ///
  /// In en, this message translates to:
  /// **'My Jobs'**
  String get jobMyJobs;

  /// No description provided for @jobNoJobs.
  ///
  /// In en, this message translates to:
  /// **'No jobs yet'**
  String get jobNoJobs;

  /// No description provided for @jobNoJobsDesc.
  ///
  /// In en, this message translates to:
  /// **'Create your first job posting to start finding talent.'**
  String get jobNoJobsDesc;

  /// No description provided for @jobStatusOpen.
  ///
  /// In en, this message translates to:
  /// **'Open'**
  String get jobStatusOpen;

  /// No description provided for @jobStatusClosed.
  ///
  /// In en, this message translates to:
  /// **'Closed'**
  String get jobStatusClosed;

  /// No description provided for @jobClose.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get jobClose;

  /// No description provided for @jobReopen.
  ///
  /// In en, this message translates to:
  /// **'Reopen'**
  String get jobReopen;

  /// No description provided for @jobDelete.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get jobDelete;

  /// No description provided for @jobDeleteConfirm.
  ///
  /// In en, this message translates to:
  /// **'Are you sure you want to delete this job? This action cannot be undone.'**
  String get jobDeleteConfirm;

  /// No description provided for @jobDeleteSuccess.
  ///
  /// In en, this message translates to:
  /// **'Job deleted successfully'**
  String get jobDeleteSuccess;

  /// No description provided for @jobReopenSuccess.
  ///
  /// In en, this message translates to:
  /// **'Job reopened successfully'**
  String get jobReopenSuccess;

  /// No description provided for @jobOfferDetails.
  ///
  /// In en, this message translates to:
  /// **'Offer details'**
  String get jobOfferDetails;

  /// No description provided for @jobCandidates.
  ///
  /// In en, this message translates to:
  /// **'Candidates'**
  String get jobCandidates;

  /// No description provided for @jobNoCandidates.
  ///
  /// In en, this message translates to:
  /// **'No candidates yet'**
  String get jobNoCandidates;

  /// No description provided for @jobNoCandidatesDesc.
  ///
  /// In en, this message translates to:
  /// **'Applications will appear here when candidates apply.'**
  String get jobNoCandidatesDesc;

  /// No description provided for @jobEditJob.
  ///
  /// In en, this message translates to:
  /// **'Edit job'**
  String get jobEditJob;

  /// No description provided for @jobPostedOn.
  ///
  /// In en, this message translates to:
  /// **'Posted on'**
  String get jobPostedOn;

  /// No description provided for @jobDescriptionTypeText.
  ///
  /// In en, this message translates to:
  /// **'Text'**
  String get jobDescriptionTypeText;

  /// No description provided for @jobDescriptionTypeVideo.
  ///
  /// In en, this message translates to:
  /// **'Video'**
  String get jobDescriptionTypeVideo;

  /// No description provided for @jobDescriptionTypeBoth.
  ///
  /// In en, this message translates to:
  /// **'Both'**
  String get jobDescriptionTypeBoth;

  /// No description provided for @jobDescriptionType.
  ///
  /// In en, this message translates to:
  /// **'Description type'**
  String get jobDescriptionType;

  /// No description provided for @jobAddVideo.
  ///
  /// In en, this message translates to:
  /// **'Add a video'**
  String get jobAddVideo;

  /// No description provided for @jobVideoUploading.
  ///
  /// In en, this message translates to:
  /// **'Uploading video...'**
  String get jobVideoUploading;

  /// No description provided for @jobVideoUploaded.
  ///
  /// In en, this message translates to:
  /// **'Video uploaded'**
  String get jobVideoUploaded;

  /// No description provided for @jobUpdateSuccess.
  ///
  /// In en, this message translates to:
  /// **'Job updated successfully'**
  String get jobUpdateSuccess;

  /// No description provided for @proposalPropose.
  ///
  /// In en, this message translates to:
  /// **'Send a proposal'**
  String get proposalPropose;

  /// No description provided for @proposalCreate.
  ///
  /// In en, this message translates to:
  /// **'Create a proposal'**
  String get proposalCreate;

  /// No description provided for @proposalTitle.
  ///
  /// In en, this message translates to:
  /// **'Mission title'**
  String get proposalTitle;

  /// No description provided for @proposalTitleHint.
  ///
  /// In en, this message translates to:
  /// **'e.g. Corporate website redesign'**
  String get proposalTitleHint;

  /// No description provided for @proposalDescription.
  ///
  /// In en, this message translates to:
  /// **'Description'**
  String get proposalDescription;

  /// No description provided for @proposalDescriptionHint.
  ///
  /// In en, this message translates to:
  /// **'Detail deliverables and scope of work'**
  String get proposalDescriptionHint;

  /// No description provided for @proposalAmount.
  ///
  /// In en, this message translates to:
  /// **'Amount (€)'**
  String get proposalAmount;

  /// No description provided for @proposalAmountHint.
  ///
  /// In en, this message translates to:
  /// **'1500'**
  String get proposalAmountHint;

  /// No description provided for @proposalDeadline.
  ///
  /// In en, this message translates to:
  /// **'Deadline'**
  String get proposalDeadline;

  /// No description provided for @proposalRecipient.
  ///
  /// In en, this message translates to:
  /// **'Recipient'**
  String get proposalRecipient;

  /// No description provided for @proposalFrom.
  ///
  /// In en, this message translates to:
  /// **'Proposal from'**
  String get proposalFrom;

  /// No description provided for @proposalTotalAmount.
  ///
  /// In en, this message translates to:
  /// **'Total amount'**
  String get proposalTotalAmount;

  /// No description provided for @proposalPending.
  ///
  /// In en, this message translates to:
  /// **'Pending'**
  String get proposalPending;

  /// No description provided for @proposalAccepted.
  ///
  /// In en, this message translates to:
  /// **'Accepted'**
  String get proposalAccepted;

  /// No description provided for @proposalDeclined.
  ///
  /// In en, this message translates to:
  /// **'Declined'**
  String get proposalDeclined;

  /// No description provided for @proposalAccept.
  ///
  /// In en, this message translates to:
  /// **'Accept'**
  String get proposalAccept;

  /// No description provided for @proposalDecline.
  ///
  /// In en, this message translates to:
  /// **'Decline'**
  String get proposalDecline;

  /// No description provided for @proposalSend.
  ///
  /// In en, this message translates to:
  /// **'Send proposal'**
  String get proposalSend;

  /// No description provided for @proposalModify.
  ///
  /// In en, this message translates to:
  /// **'Counter-offer'**
  String get proposalModify;

  /// No description provided for @proposalWithdrawn.
  ///
  /// In en, this message translates to:
  /// **'Withdrawn'**
  String get proposalWithdrawn;

  /// No description provided for @proposalAcceptedMessage.
  ///
  /// In en, this message translates to:
  /// **'Proposal accepted'**
  String get proposalAcceptedMessage;

  /// No description provided for @proposalDeclinedMessage.
  ///
  /// In en, this message translates to:
  /// **'Proposal declined'**
  String get proposalDeclinedMessage;

  /// No description provided for @proposalPaidMessage.
  ///
  /// In en, this message translates to:
  /// **'Payment confirmed, mission in progress'**
  String get proposalPaidMessage;

  /// No description provided for @proposalPaymentRequestedMessage.
  ///
  /// In en, this message translates to:
  /// **'Payment requested'**
  String get proposalPaymentRequestedMessage;

  /// No description provided for @proposalCompletionRequestedMessage.
  ///
  /// In en, this message translates to:
  /// **'Completion requested'**
  String get proposalCompletionRequestedMessage;

  /// No description provided for @proposalCompletedMessage.
  ///
  /// In en, this message translates to:
  /// **'Mission completed'**
  String get proposalCompletedMessage;

  /// No description provided for @proposalCompletionRejectedMessage.
  ///
  /// In en, this message translates to:
  /// **'Completion rejected'**
  String get proposalCompletionRejectedMessage;

  /// No description provided for @evaluationRequestMessage.
  ///
  /// In en, this message translates to:
  /// **'Mission completed! Leave a review'**
  String get evaluationRequestMessage;

  /// No description provided for @leaveReview.
  ///
  /// In en, this message translates to:
  /// **'Leave a review'**
  String get leaveReview;

  /// No description provided for @proposalNewMessage.
  ///
  /// In en, this message translates to:
  /// **'New proposal'**
  String get proposalNewMessage;

  /// No description provided for @proposalModifiedMessage.
  ///
  /// In en, this message translates to:
  /// **'Proposal modified'**
  String get proposalModifiedMessage;

  /// No description provided for @payNow.
  ///
  /// In en, this message translates to:
  /// **'Pay now'**
  String get payNow;

  /// No description provided for @confirmPayment.
  ///
  /// In en, this message translates to:
  /// **'Confirm payment'**
  String get confirmPayment;

  /// No description provided for @paymentSimulation.
  ///
  /// In en, this message translates to:
  /// **'Payment'**
  String get paymentSimulation;

  /// No description provided for @paymentSuccess.
  ///
  /// In en, this message translates to:
  /// **'Payment confirmed!'**
  String get paymentSuccess;

  /// No description provided for @paymentSuccessDesc.
  ///
  /// In en, this message translates to:
  /// **'The mission is now active. Redirecting to projects...'**
  String get paymentSuccessDesc;

  /// No description provided for @noActiveProjects.
  ///
  /// In en, this message translates to:
  /// **'No active projects'**
  String get noActiveProjects;

  /// No description provided for @noActiveProjectsDesc.
  ///
  /// In en, this message translates to:
  /// **'Accepted proposals will appear here once paid.'**
  String get noActiveProjectsDesc;

  /// No description provided for @projectStatusActive.
  ///
  /// In en, this message translates to:
  /// **'Active'**
  String get projectStatusActive;

  /// No description provided for @projectStatusCompleted.
  ///
  /// In en, this message translates to:
  /// **'Completed'**
  String get projectStatusCompleted;

  /// No description provided for @startProject.
  ///
  /// In en, this message translates to:
  /// **'Start a project'**
  String get startProject;

  /// No description provided for @callCalling.
  ///
  /// In en, this message translates to:
  /// **'Calling...'**
  String get callCalling;

  /// No description provided for @callIncomingCall.
  ///
  /// In en, this message translates to:
  /// **'Incoming call'**
  String get callIncomingCall;

  /// No description provided for @callAudioCall.
  ///
  /// In en, this message translates to:
  /// **'Audio call'**
  String get callAudioCall;

  /// No description provided for @callAccept.
  ///
  /// In en, this message translates to:
  /// **'Accept'**
  String get callAccept;

  /// No description provided for @callDecline.
  ///
  /// In en, this message translates to:
  /// **'Decline'**
  String get callDecline;

  /// No description provided for @callHangup.
  ///
  /// In en, this message translates to:
  /// **'Hang up'**
  String get callHangup;

  /// No description provided for @callMute.
  ///
  /// In en, this message translates to:
  /// **'Mute'**
  String get callMute;

  /// No description provided for @callUnmute.
  ///
  /// In en, this message translates to:
  /// **'Unmute'**
  String get callUnmute;

  /// No description provided for @callEnded.
  ///
  /// In en, this message translates to:
  /// **'Call ended'**
  String get callEnded;

  /// No description provided for @callMissed.
  ///
  /// In en, this message translates to:
  /// **'Missed call'**
  String get callMissed;

  /// No description provided for @callStartCall.
  ///
  /// In en, this message translates to:
  /// **'Start audio call'**
  String get callStartCall;

  /// No description provided for @callRecipientOffline.
  ///
  /// In en, this message translates to:
  /// **'Recipient is offline'**
  String get callRecipientOffline;

  /// No description provided for @callUserBusy.
  ///
  /// In en, this message translates to:
  /// **'User is already in a call'**
  String get callUserBusy;

  /// No description provided for @callFailed.
  ///
  /// In en, this message translates to:
  /// **'Call could not be started'**
  String get callFailed;

  /// No description provided for @callUnknownCaller.
  ///
  /// In en, this message translates to:
  /// **'Unknown caller'**
  String get callUnknownCaller;

  /// No description provided for @callVideoCall.
  ///
  /// In en, this message translates to:
  /// **'Video call'**
  String get callVideoCall;

  /// No description provided for @callStartVideoCall.
  ///
  /// In en, this message translates to:
  /// **'Start video call'**
  String get callStartVideoCall;

  /// No description provided for @callCamera.
  ///
  /// In en, this message translates to:
  /// **'Camera'**
  String get callCamera;

  /// No description provided for @callCameraOff.
  ///
  /// In en, this message translates to:
  /// **'Camera off'**
  String get callCameraOff;

  /// No description provided for @callCameraOn.
  ///
  /// In en, this message translates to:
  /// **'Camera on'**
  String get callCameraOn;

  /// No description provided for @callNoVideo.
  ///
  /// In en, this message translates to:
  /// **'Camera is off'**
  String get callNoVideo;

  /// No description provided for @callIncomingVideoCall.
  ///
  /// In en, this message translates to:
  /// **'Incoming video call'**
  String get callIncomingVideoCall;

  /// No description provided for @callTapToReturn.
  ///
  /// In en, this message translates to:
  /// **'Tap to return to call'**
  String get callTapToReturn;

  /// No description provided for @callMinimize.
  ///
  /// In en, this message translates to:
  /// **'Minimize'**
  String get callMinimize;

  /// No description provided for @drawerDashboard.
  ///
  /// In en, this message translates to:
  /// **'Dashboard'**
  String get drawerDashboard;

  /// No description provided for @drawerMessages.
  ///
  /// In en, this message translates to:
  /// **'Messages'**
  String get drawerMessages;

  /// No description provided for @drawerProjects.
  ///
  /// In en, this message translates to:
  /// **'Projects'**
  String get drawerProjects;

  /// No description provided for @drawerJobs.
  ///
  /// In en, this message translates to:
  /// **'Job postings'**
  String get drawerJobs;

  /// No description provided for @drawerProfile.
  ///
  /// In en, this message translates to:
  /// **'My profile'**
  String get drawerProfile;

  /// No description provided for @drawerFindFreelancers.
  ///
  /// In en, this message translates to:
  /// **'Find freelancers'**
  String get drawerFindFreelancers;

  /// No description provided for @drawerFindAgencies.
  ///
  /// In en, this message translates to:
  /// **'Find agencies'**
  String get drawerFindAgencies;

  /// No description provided for @drawerFindReferrers.
  ///
  /// In en, this message translates to:
  /// **'Find referrers'**
  String get drawerFindReferrers;

  /// No description provided for @drawerLogout.
  ///
  /// In en, this message translates to:
  /// **'Log out'**
  String get drawerLogout;

  /// No description provided for @drawerLogoutConfirm.
  ///
  /// In en, this message translates to:
  /// **'Are you sure you want to log out?'**
  String get drawerLogoutConfirm;

  /// No description provided for @drawerSwitchToReferrer.
  ///
  /// In en, this message translates to:
  /// **'Business Referrer'**
  String get drawerSwitchToReferrer;

  /// No description provided for @drawerSwitchToFreelance.
  ///
  /// In en, this message translates to:
  /// **'Freelance Dashboard'**
  String get drawerSwitchToFreelance;

  /// No description provided for @drawerPaymentInfo.
  ///
  /// In en, this message translates to:
  /// **'Payment Info'**
  String get drawerPaymentInfo;

  /// No description provided for @drawerNotifications.
  ///
  /// In en, this message translates to:
  /// **'Notifications'**
  String get drawerNotifications;

  /// No description provided for @notifications.
  ///
  /// In en, this message translates to:
  /// **'Notifications'**
  String get notifications;

  /// No description provided for @noNotifications.
  ///
  /// In en, this message translates to:
  /// **'No notifications yet'**
  String get noNotifications;

  /// No description provided for @noNotificationsDesc.
  ///
  /// In en, this message translates to:
  /// **'You\'ll be notified when something happens'**
  String get noNotificationsDesc;

  /// No description provided for @markAllRead.
  ///
  /// In en, this message translates to:
  /// **'Mark all read'**
  String get markAllRead;

  /// No description provided for @proposalViewDetails.
  ///
  /// In en, this message translates to:
  /// **'View details'**
  String get proposalViewDetails;

  /// No description provided for @paymentInfoTitle.
  ///
  /// In en, this message translates to:
  /// **'Payment Information'**
  String get paymentInfoTitle;

  /// No description provided for @paymentInfoSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Complete your information to receive payments for your projects.'**
  String get paymentInfoSubtitle;

  /// No description provided for @paymentInfoIsBusiness.
  ///
  /// In en, this message translates to:
  /// **'I operate as a registered business'**
  String get paymentInfoIsBusiness;

  /// No description provided for @paymentInfoIsBusinessDesc.
  ///
  /// In en, this message translates to:
  /// **'Enable if your activity is operated through a registered company. Leave disabled if you operate under a sole proprietorship (freelance, independent).'**
  String get paymentInfoIsBusinessDesc;

  /// No description provided for @paymentInfoPersonalInfo.
  ///
  /// In en, this message translates to:
  /// **'Personal Information'**
  String get paymentInfoPersonalInfo;

  /// No description provided for @paymentInfoLegalRep.
  ///
  /// In en, this message translates to:
  /// **'Legal Representative'**
  String get paymentInfoLegalRep;

  /// No description provided for @paymentInfoBusinessInfo.
  ///
  /// In en, this message translates to:
  /// **'Business Information'**
  String get paymentInfoBusinessInfo;

  /// No description provided for @paymentInfoBankAccount.
  ///
  /// In en, this message translates to:
  /// **'Bank Account'**
  String get paymentInfoBankAccount;

  /// No description provided for @paymentInfoFirstName.
  ///
  /// In en, this message translates to:
  /// **'First name'**
  String get paymentInfoFirstName;

  /// No description provided for @paymentInfoLastName.
  ///
  /// In en, this message translates to:
  /// **'Last name'**
  String get paymentInfoLastName;

  /// No description provided for @paymentInfoDob.
  ///
  /// In en, this message translates to:
  /// **'Date of birth'**
  String get paymentInfoDob;

  /// No description provided for @paymentInfoNationality.
  ///
  /// In en, this message translates to:
  /// **'Nationality'**
  String get paymentInfoNationality;

  /// No description provided for @paymentInfoAddress.
  ///
  /// In en, this message translates to:
  /// **'Address'**
  String get paymentInfoAddress;

  /// No description provided for @paymentInfoCity.
  ///
  /// In en, this message translates to:
  /// **'City'**
  String get paymentInfoCity;

  /// No description provided for @paymentInfoPostalCode.
  ///
  /// In en, this message translates to:
  /// **'Postal code'**
  String get paymentInfoPostalCode;

  /// No description provided for @paymentInfoYourRole.
  ///
  /// In en, this message translates to:
  /// **'Your role in the company'**
  String get paymentInfoYourRole;

  /// No description provided for @paymentInfoBusinessName.
  ///
  /// In en, this message translates to:
  /// **'Business name'**
  String get paymentInfoBusinessName;

  /// No description provided for @paymentInfoBusinessAddress.
  ///
  /// In en, this message translates to:
  /// **'Business address'**
  String get paymentInfoBusinessAddress;

  /// No description provided for @paymentInfoBusinessCity.
  ///
  /// In en, this message translates to:
  /// **'Business city'**
  String get paymentInfoBusinessCity;

  /// No description provided for @paymentInfoBusinessPostalCode.
  ///
  /// In en, this message translates to:
  /// **'Business postal code'**
  String get paymentInfoBusinessPostalCode;

  /// No description provided for @paymentInfoBusinessCountry.
  ///
  /// In en, this message translates to:
  /// **'Country of registration'**
  String get paymentInfoBusinessCountry;

  /// No description provided for @paymentInfoTaxId.
  ///
  /// In en, this message translates to:
  /// **'Tax ID'**
  String get paymentInfoTaxId;

  /// No description provided for @paymentInfoTaxIdHint.
  ///
  /// In en, this message translates to:
  /// **'SIRET, EIN, VAT ID...'**
  String get paymentInfoTaxIdHint;

  /// No description provided for @paymentInfoVatNumber.
  ///
  /// In en, this message translates to:
  /// **'VAT number (optional)'**
  String get paymentInfoVatNumber;

  /// No description provided for @paymentInfoVatNumberHint.
  ///
  /// In en, this message translates to:
  /// **'EU VAT number (optional)'**
  String get paymentInfoVatNumberHint;

  /// No description provided for @paymentInfoIban.
  ///
  /// In en, this message translates to:
  /// **'IBAN'**
  String get paymentInfoIban;

  /// No description provided for @paymentInfoIbanHint.
  ///
  /// In en, this message translates to:
  /// **'FR76 1234 5678 9012 3456 78'**
  String get paymentInfoIbanHint;

  /// No description provided for @paymentInfoBic.
  ///
  /// In en, this message translates to:
  /// **'BIC / SWIFT (optional)'**
  String get paymentInfoBic;

  /// No description provided for @paymentInfoBicHint.
  ///
  /// In en, this message translates to:
  /// **'BNPAFRPP'**
  String get paymentInfoBicHint;

  /// No description provided for @paymentInfoIbanHelp.
  ///
  /// In en, this message translates to:
  /// **'If your bank hasn\'t provided an IBAN, you can generate one at'**
  String get paymentInfoIbanHelp;

  /// No description provided for @paymentInfoNoIban.
  ///
  /// In en, this message translates to:
  /// **'I don\'t have an IBAN'**
  String get paymentInfoNoIban;

  /// No description provided for @paymentInfoUseIban.
  ///
  /// In en, this message translates to:
  /// **'I have an IBAN'**
  String get paymentInfoUseIban;

  /// No description provided for @paymentInfoAccountNumber.
  ///
  /// In en, this message translates to:
  /// **'Account number'**
  String get paymentInfoAccountNumber;

  /// No description provided for @paymentInfoRoutingNumber.
  ///
  /// In en, this message translates to:
  /// **'Routing number'**
  String get paymentInfoRoutingNumber;

  /// No description provided for @paymentInfoAccountHolder.
  ///
  /// In en, this message translates to:
  /// **'Account holder name'**
  String get paymentInfoAccountHolder;

  /// No description provided for @paymentInfoBankCountry.
  ///
  /// In en, this message translates to:
  /// **'Bank country'**
  String get paymentInfoBankCountry;

  /// No description provided for @paymentInfoSave.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get paymentInfoSave;

  /// No description provided for @paymentInfoSaved.
  ///
  /// In en, this message translates to:
  /// **'Payment information saved'**
  String get paymentInfoSaved;

  /// No description provided for @paymentInfoIncomplete.
  ///
  /// In en, this message translates to:
  /// **'Complete your payment information to receive payments'**
  String get paymentInfoIncomplete;

  /// No description provided for @paymentInfoRoleOwner.
  ///
  /// In en, this message translates to:
  /// **'Owner'**
  String get paymentInfoRoleOwner;

  /// No description provided for @paymentInfoRoleCeo.
  ///
  /// In en, this message translates to:
  /// **'CEO'**
  String get paymentInfoRoleCeo;

  /// No description provided for @paymentInfoRoleDirector.
  ///
  /// In en, this message translates to:
  /// **'Director'**
  String get paymentInfoRoleDirector;

  /// No description provided for @paymentInfoRolePartner.
  ///
  /// In en, this message translates to:
  /// **'Partner'**
  String get paymentInfoRolePartner;

  /// No description provided for @paymentInfoRoleOther.
  ///
  /// In en, this message translates to:
  /// **'Other'**
  String get paymentInfoRoleOther;

  /// No description provided for @paymentInfoPhone.
  ///
  /// In en, this message translates to:
  /// **'Phone number'**
  String get paymentInfoPhone;

  /// No description provided for @paymentInfoActivitySector.
  ///
  /// In en, this message translates to:
  /// **'Activity sector'**
  String get paymentInfoActivitySector;

  /// No description provided for @paymentInfoBusinessPersons.
  ///
  /// In en, this message translates to:
  /// **'Business representatives'**
  String get paymentInfoBusinessPersons;

  /// No description provided for @paymentInfoSelfRepresentative.
  ///
  /// In en, this message translates to:
  /// **'I am the legal representative'**
  String get paymentInfoSelfRepresentative;

  /// No description provided for @paymentInfoSelfDirector.
  ///
  /// In en, this message translates to:
  /// **'The legal representative is the sole director'**
  String get paymentInfoSelfDirector;

  /// No description provided for @paymentInfoNoMajorOwners.
  ///
  /// In en, this message translates to:
  /// **'No shareholder holds more than 25%'**
  String get paymentInfoNoMajorOwners;

  /// No description provided for @paymentInfoSelfExecutive.
  ///
  /// In en, this message translates to:
  /// **'The legal representative is the sole executive'**
  String get paymentInfoSelfExecutive;

  /// No description provided for @paymentInfoAddPerson.
  ///
  /// In en, this message translates to:
  /// **'Add a person'**
  String get paymentInfoAddPerson;

  /// No description provided for @paymentInfoPerson.
  ///
  /// In en, this message translates to:
  /// **'Person'**
  String get paymentInfoPerson;

  /// No description provided for @identityDocTitle.
  ///
  /// In en, this message translates to:
  /// **'Identity verification'**
  String get identityDocTitle;

  /// No description provided for @identityDocSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Upload a government-issued identity document for verification.'**
  String get identityDocSubtitle;

  /// No description provided for @identityDocType.
  ///
  /// In en, this message translates to:
  /// **'Document type'**
  String get identityDocType;

  /// No description provided for @identityDocPending.
  ///
  /// In en, this message translates to:
  /// **'Pending'**
  String get identityDocPending;

  /// No description provided for @identityDocVerified.
  ///
  /// In en, this message translates to:
  /// **'Verified'**
  String get identityDocVerified;

  /// No description provided for @identityDocRejected.
  ///
  /// In en, this message translates to:
  /// **'Rejected'**
  String get identityDocRejected;

  /// No description provided for @identityDocUploaded.
  ///
  /// In en, this message translates to:
  /// **'Document uploaded successfully'**
  String get identityDocUploaded;

  /// No description provided for @identityDocUpload.
  ///
  /// In en, this message translates to:
  /// **'Upload identity document'**
  String get identityDocUpload;

  /// No description provided for @identityDocUploadDesc.
  ///
  /// In en, this message translates to:
  /// **'Upload a clear photo of your document'**
  String get identityDocUploadDesc;

  /// No description provided for @identityDocPassport.
  ///
  /// In en, this message translates to:
  /// **'Passport'**
  String get identityDocPassport;

  /// No description provided for @identityDocIdCard.
  ///
  /// In en, this message translates to:
  /// **'ID Card'**
  String get identityDocIdCard;

  /// No description provided for @identityDocDrivingLicense.
  ///
  /// In en, this message translates to:
  /// **'Driving License'**
  String get identityDocDrivingLicense;

  /// No description provided for @identityDocSinglePage.
  ///
  /// In en, this message translates to:
  /// **'Single page upload'**
  String get identityDocSinglePage;

  /// No description provided for @identityDocFrontAndBack.
  ///
  /// In en, this message translates to:
  /// **'Front and back required'**
  String get identityDocFrontAndBack;

  /// No description provided for @identityDocFrontSide.
  ///
  /// In en, this message translates to:
  /// **'Front side'**
  String get identityDocFrontSide;

  /// No description provided for @identityDocBackSide.
  ///
  /// In en, this message translates to:
  /// **'Back side'**
  String get identityDocBackSide;

  /// No description provided for @identityDocReplace.
  ///
  /// In en, this message translates to:
  /// **'Replace'**
  String get identityDocReplace;

  /// No description provided for @identityDocSelectType.
  ///
  /// In en, this message translates to:
  /// **'Select document type'**
  String get identityDocSelectType;

  /// No description provided for @identityDocPendingBanner.
  ///
  /// In en, this message translates to:
  /// **'Your document is being reviewed'**
  String get identityDocPendingBanner;

  /// No description provided for @identityDocVerifiedBanner.
  ///
  /// In en, this message translates to:
  /// **'Your identity has been verified'**
  String get identityDocVerifiedBanner;

  /// No description provided for @identityDocRejectedBanner.
  ///
  /// In en, this message translates to:
  /// **'Your document was rejected'**
  String get identityDocRejectedBanner;

  /// No description provided for @identityDocPassportDesc.
  ///
  /// In en, this message translates to:
  /// **'Valid passport, national ID card, or driver\'s license'**
  String get identityDocPassportDesc;

  /// No description provided for @identityDocProofOfAddressDesc.
  ///
  /// In en, this message translates to:
  /// **'Utility bill (less than 3 months old), bank statement, or certificate of residence'**
  String get identityDocProofOfAddressDesc;

  /// No description provided for @identityDocBusinessRegDesc.
  ///
  /// In en, this message translates to:
  /// **'Certificate of incorporation, articles of organization, or official equivalent for your country'**
  String get identityDocBusinessRegDesc;

  /// No description provided for @identityDocProofOfLivenessDesc.
  ///
  /// In en, this message translates to:
  /// **'Live photo of your face to confirm your identity'**
  String get identityDocProofOfLivenessDesc;

  /// No description provided for @identityDocProofOfRegistrationDesc.
  ///
  /// In en, this message translates to:
  /// **'Certificate of registration, incorporation document, or official proof from your country\'s business registry'**
  String get identityDocProofOfRegistrationDesc;

  /// No description provided for @paymentInfoAddRepresentative.
  ///
  /// In en, this message translates to:
  /// **'Add a representative'**
  String get paymentInfoAddRepresentative;

  /// No description provided for @paymentInfoAddDirector.
  ///
  /// In en, this message translates to:
  /// **'Add a director'**
  String get paymentInfoAddDirector;

  /// No description provided for @paymentInfoAddOwner.
  ///
  /// In en, this message translates to:
  /// **'Add a shareholder'**
  String get paymentInfoAddOwner;

  /// No description provided for @paymentInfoAddExecutive.
  ///
  /// In en, this message translates to:
  /// **'Add an executive'**
  String get paymentInfoAddExecutive;

  /// No description provided for @paymentInfoRepresentative.
  ///
  /// In en, this message translates to:
  /// **'Representative'**
  String get paymentInfoRepresentative;

  /// No description provided for @paymentInfoDirectorLabel.
  ///
  /// In en, this message translates to:
  /// **'Director'**
  String get paymentInfoDirectorLabel;

  /// No description provided for @paymentInfoOwnerLabel.
  ///
  /// In en, this message translates to:
  /// **'Shareholder'**
  String get paymentInfoOwnerLabel;

  /// No description provided for @paymentInfoExecutiveLabel.
  ///
  /// In en, this message translates to:
  /// **'Executive'**
  String get paymentInfoExecutiveLabel;

  /// No description provided for @paymentInfoPersonTitle.
  ///
  /// In en, this message translates to:
  /// **'Title'**
  String get paymentInfoPersonTitle;

  /// No description provided for @paymentInfoDateOfBirth.
  ///
  /// In en, this message translates to:
  /// **'Date of birth'**
  String get paymentInfoDateOfBirth;

  /// No description provided for @stripeRequirementsTitle.
  ///
  /// In en, this message translates to:
  /// **'Additional information required'**
  String get stripeRequirementsTitle;

  /// No description provided for @stripeRequirementsDesc.
  ///
  /// In en, this message translates to:
  /// **'Please provide the following information to keep your account active.'**
  String get stripeRequirementsDesc;

  /// No description provided for @stripeCompleteOnStripe.
  ///
  /// In en, this message translates to:
  /// **'Complete on Stripe'**
  String get stripeCompleteOnStripe;

  /// No description provided for @walletTitle.
  ///
  /// In en, this message translates to:
  /// **'Wallet'**
  String get walletTitle;

  /// No description provided for @walletStripeAccount.
  ///
  /// In en, this message translates to:
  /// **'Stripe account'**
  String get walletStripeAccount;

  /// No description provided for @walletCharges.
  ///
  /// In en, this message translates to:
  /// **'Charges'**
  String get walletCharges;

  /// No description provided for @walletPayouts.
  ///
  /// In en, this message translates to:
  /// **'Payouts'**
  String get walletPayouts;

  /// No description provided for @walletEscrow.
  ///
  /// In en, this message translates to:
  /// **'Escrow'**
  String get walletEscrow;

  /// No description provided for @walletAvailable.
  ///
  /// In en, this message translates to:
  /// **'Available'**
  String get walletAvailable;

  /// No description provided for @walletTransferred.
  ///
  /// In en, this message translates to:
  /// **'Transferred'**
  String get walletTransferred;

  /// No description provided for @walletRequestPayout.
  ///
  /// In en, this message translates to:
  /// **'Withdraw'**
  String get walletRequestPayout;

  /// No description provided for @walletPayoutRequested.
  ///
  /// In en, this message translates to:
  /// **'Payout requested successfully'**
  String get walletPayoutRequested;

  /// No description provided for @walletTransactionHistory.
  ///
  /// In en, this message translates to:
  /// **'Transaction history'**
  String get walletTransactionHistory;

  /// No description provided for @walletNoTransactions.
  ///
  /// In en, this message translates to:
  /// **'No transactions yet'**
  String get walletNoTransactions;

  /// No description provided for @drawerWallet.
  ///
  /// In en, this message translates to:
  /// **'Wallet'**
  String get drawerWallet;

  /// No description provided for @reportMessage.
  ///
  /// In en, this message translates to:
  /// **'Report this message'**
  String get reportMessage;

  /// No description provided for @reportUser.
  ///
  /// In en, this message translates to:
  /// **'Report this user'**
  String get reportUser;

  /// No description provided for @report.
  ///
  /// In en, this message translates to:
  /// **'Report'**
  String get report;

  /// No description provided for @selectReason.
  ///
  /// In en, this message translates to:
  /// **'What\'s the issue?'**
  String get selectReason;

  /// No description provided for @reportDescription.
  ///
  /// In en, this message translates to:
  /// **'Additional details'**
  String get reportDescription;

  /// No description provided for @reportDescriptionHint.
  ///
  /// In en, this message translates to:
  /// **'Describe the issue in detail...'**
  String get reportDescriptionHint;

  /// No description provided for @submitReport.
  ///
  /// In en, this message translates to:
  /// **'Submit report'**
  String get submitReport;

  /// No description provided for @reportSubmitting.
  ///
  /// In en, this message translates to:
  /// **'Submitting...'**
  String get reportSubmitting;

  /// No description provided for @reportSent.
  ///
  /// In en, this message translates to:
  /// **'Report submitted. Our team will review it.'**
  String get reportSent;

  /// No description provided for @reportError.
  ///
  /// In en, this message translates to:
  /// **'Failed to submit report.'**
  String get reportError;

  /// No description provided for @reasonHarassment.
  ///
  /// In en, this message translates to:
  /// **'Harassment or bullying'**
  String get reasonHarassment;

  /// No description provided for @reasonFraud.
  ///
  /// In en, this message translates to:
  /// **'Fraud or scam'**
  String get reasonFraud;

  /// No description provided for @reasonOffPlatformPayment.
  ///
  /// In en, this message translates to:
  /// **'Payment outside platform'**
  String get reasonOffPlatformPayment;

  /// No description provided for @reasonSpam.
  ///
  /// In en, this message translates to:
  /// **'Spam'**
  String get reasonSpam;

  /// No description provided for @reasonInappropriateContent.
  ///
  /// In en, this message translates to:
  /// **'Inappropriate content'**
  String get reasonInappropriateContent;

  /// No description provided for @reasonFakeProfile.
  ///
  /// In en, this message translates to:
  /// **'Fake or misleading profile'**
  String get reasonFakeProfile;

  /// No description provided for @reasonUnprofessionalBehavior.
  ///
  /// In en, this message translates to:
  /// **'Unprofessional behavior'**
  String get reasonUnprofessionalBehavior;

  /// No description provided for @reasonOther.
  ///
  /// In en, this message translates to:
  /// **'Other'**
  String get reasonOther;

  /// No description provided for @loadMore.
  ///
  /// In en, this message translates to:
  /// **'Load more'**
  String get loadMore;

  /// No description provided for @candidateDetail.
  ///
  /// In en, this message translates to:
  /// **'Application'**
  String get candidateDetail;

  /// No description provided for @applicationMessage.
  ///
  /// In en, this message translates to:
  /// **'Application message'**
  String get applicationMessage;

  /// No description provided for @applicationVideo.
  ///
  /// In en, this message translates to:
  /// **'Presentation video'**
  String get applicationVideo;

  /// No description provided for @opportunities.
  ///
  /// In en, this message translates to:
  /// **'Opportunities'**
  String get opportunities;

  /// No description provided for @noOpportunities.
  ///
  /// In en, this message translates to:
  /// **'No opportunities at the moment'**
  String get noOpportunities;

  /// No description provided for @jobNotFound.
  ///
  /// In en, this message translates to:
  /// **'Job not found'**
  String get jobNotFound;

  /// No description provided for @budgetTypeOneShot.
  ///
  /// In en, this message translates to:
  /// **'One-time project'**
  String get budgetTypeOneShot;

  /// No description provided for @budgetTypeLongTerm.
  ///
  /// In en, this message translates to:
  /// **'Long-term collaboration'**
  String get budgetTypeLongTerm;

  /// No description provided for @myApplications.
  ///
  /// In en, this message translates to:
  /// **'My applications'**
  String get myApplications;

  /// No description provided for @noApplications.
  ///
  /// In en, this message translates to:
  /// **'You haven\'t applied to any job yet'**
  String get noApplications;

  /// No description provided for @withdrawApplicationTitle.
  ///
  /// In en, this message translates to:
  /// **'Withdraw application?'**
  String get withdrawApplicationTitle;

  /// No description provided for @withdrawAction.
  ///
  /// In en, this message translates to:
  /// **'Withdraw'**
  String get withdrawAction;

  /// No description provided for @applications.
  ///
  /// In en, this message translates to:
  /// **'Applications'**
  String get applications;

  /// No description provided for @noApplicationsYet.
  ///
  /// In en, this message translates to:
  /// **'No applications yet'**
  String get noApplicationsYet;

  /// No description provided for @applyAction.
  ///
  /// In en, this message translates to:
  /// **'Apply'**
  String get applyAction;

  /// No description provided for @alreadyApplied.
  ///
  /// In en, this message translates to:
  /// **'Already applied'**
  String get alreadyApplied;

  /// No description provided for @applyTitle.
  ///
  /// In en, this message translates to:
  /// **'Apply'**
  String get applyTitle;

  /// No description provided for @applyMessageLabel.
  ///
  /// In en, this message translates to:
  /// **'Your message (optional)'**
  String get applyMessageLabel;

  /// No description provided for @applyMessageHint.
  ///
  /// In en, this message translates to:
  /// **'Why are you the right candidate?'**
  String get applyMessageHint;

  /// No description provided for @applyAddVideo.
  ///
  /// In en, this message translates to:
  /// **'Add a video'**
  String get applyAddVideo;

  /// No description provided for @applyUploading.
  ///
  /// In en, this message translates to:
  /// **'Uploading...'**
  String get applyUploading;

  /// No description provided for @applyRemoveVideo.
  ///
  /// In en, this message translates to:
  /// **'Remove video'**
  String get applyRemoveVideo;

  /// No description provided for @applySubmit.
  ///
  /// In en, this message translates to:
  /// **'Send my application'**
  String get applySubmit;

  /// No description provided for @applicationSent.
  ///
  /// In en, this message translates to:
  /// **'Application sent!'**
  String get applicationSent;

  /// No description provided for @applicationSendError.
  ///
  /// In en, this message translates to:
  /// **'Failed to send application'**
  String get applicationSendError;

  /// No description provided for @videoUploadFailed.
  ///
  /// In en, this message translates to:
  /// **'Video upload failed. Please try again.'**
  String get videoUploadFailed;

  /// No description provided for @jobTotalApplicants.
  ///
  /// In en, this message translates to:
  /// **'{count} applicants'**
  String jobTotalApplicants(int count);

  /// No description provided for @jobNewApplicants.
  ///
  /// In en, this message translates to:
  /// **'{count} new'**
  String jobNewApplicants(int count);

  /// No description provided for @candidateOf.
  ///
  /// In en, this message translates to:
  /// **'{current} of {total}'**
  String candidateOf(int current, int total);

  /// No description provided for @uploadProgress.
  ///
  /// In en, this message translates to:
  /// **'{percent}%'**
  String uploadProgress(int percent);
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
    'that was used.',
  );
}
