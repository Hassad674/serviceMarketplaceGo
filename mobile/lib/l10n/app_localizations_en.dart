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

  @override
  String get search => 'Search';

  @override
  String get findFreelancers => 'Find Freelancers';

  @override
  String get findAgencies => 'Find Agencies';

  @override
  String get findReferrers => 'Find Referrers';

  @override
  String get noProfilesFound => 'No profiles found';

  @override
  String get searchTryAgain => 'Try again later or adjust your search.';

  @override
  String get couldNotLoadProfiles =>
      'Could not load profiles. Check your connection.';

  @override
  String get couldNotLoadProfile => 'Could not load profile';

  @override
  String get checkConnectionRetry => 'Check your connection and try again.';

  @override
  String get somethingWentWrong => 'Something went wrong';

  @override
  String get retry => 'Retry';

  @override
  String get tapToPlay => 'Tap to play';

  @override
  String get replaceVideo => 'Replace video';

  @override
  String get removeVideo => 'Remove video';

  @override
  String get removeVideoConfirmTitle => 'Remove video';

  @override
  String get removeVideoConfirmMessage =>
      'Are you sure you want to remove your presentation video?';

  @override
  String get remove => 'Remove';

  @override
  String get darkMode => 'Dark Mode';

  @override
  String get aboutPlaceholder =>
      'Tell others about yourself and your expertise';

  @override
  String get aboutEditHint => 'Tell others about yourself...';

  @override
  String get aboutUpdated => 'About updated';

  @override
  String get titlePlaceholder => 'Add your professional title';

  @override
  String get videoRemoved => 'Video removed';

  @override
  String get couldNotOpenVideo => 'Could not open video';

  @override
  String get messagingSearchHint => 'Search conversations...';

  @override
  String get messagingNoMessages => 'No messages in this conversation';

  @override
  String get messagingNoConversations => 'No conversations yet';

  @override
  String get messagingWriteMessage => 'Write your message...';

  @override
  String get messagingOnline => 'Online';

  @override
  String get messagingOffline => 'Offline';

  @override
  String get messagingAllRoles => 'All';

  @override
  String get messagingAgency => 'Agency';

  @override
  String get messagingFreelancer => 'Freelance/Referrer';

  @override
  String get messagingEnterprise => 'Enterprise';

  @override
  String get messagingConversationNotFound => 'Conversation not found';

  @override
  String get messagingSendMessage => 'Send a message';

  @override
  String messagingTyping(String name) {
    return '$name is typing...';
  }

  @override
  String get messagingTypingShort => 'typing...';

  @override
  String get messagingEdited => 'edited';

  @override
  String get messagingDeleted => 'This message was deleted';

  @override
  String get messagingDelivered => 'Delivered';

  @override
  String get messagingRead => 'Read';

  @override
  String get messagingSent => 'Sent';

  @override
  String get messagingSending => 'Sending...';

  @override
  String get messagingReconnecting => 'Reconnecting...';

  @override
  String get messagingEditMessage => 'Edit message';

  @override
  String get messagingDeleteMessage => 'Delete message';

  @override
  String get messagingDeleteConfirm =>
      'Are you sure you want to delete this message?';

  @override
  String get messagingFileUpload => 'Send a file';

  @override
  String get messagingStartConversation =>
      'No messages yet. Start the conversation!';

  @override
  String get messagingLoadMore => 'Load more';

  @override
  String get messagingVoiceMessage => 'Voice message';

  @override
  String get messagingRecording => 'Recording...';

  @override
  String get messagingCancelRecording => 'Cancel';

  @override
  String get messagingMicrophonePermission => 'Microphone access required';

  @override
  String get messagingReply => 'Reply';

  @override
  String messagingReplyingTo(String name) {
    return 'Replying to $name';
  }

  @override
  String get projects => 'Projects';

  @override
  String get createProject => 'Create Project';

  @override
  String get noProjects => 'No projects yet';

  @override
  String get noProjectsDesc => 'Create your first project to get started.';

  @override
  String get paymentType => 'Payment type';

  @override
  String get invoiceBilling => 'Invoice billing';

  @override
  String get invoiceBillingDesc =>
      'Classic invoicing with flexible billing cycles.';

  @override
  String get escrowPayments => 'Escrow payments';

  @override
  String get escrowPaymentsDesc =>
      'Funds held securely until milestones are approved.';

  @override
  String get projectStructure => 'Structure';

  @override
  String get milestone => 'Milestone';

  @override
  String get oneTime => 'One-time';

  @override
  String get billingDetails => 'Billing details';

  @override
  String get fixed => 'Fixed';

  @override
  String get hourly => 'Hourly';

  @override
  String get rate => 'Rate';

  @override
  String get frequency => 'Frequency';

  @override
  String get weekly => 'Weekly';

  @override
  String get biWeekly => 'Bi-weekly';

  @override
  String get monthly => 'Monthly';

  @override
  String get projectDetails => 'Details';

  @override
  String get projectTitle => 'Project title';

  @override
  String get projectDescription => 'Description';

  @override
  String get requiredSkills => 'Required skills';

  @override
  String get addSkillHint => 'Type a skill and press add';

  @override
  String get timeline => 'Timeline';

  @override
  String get startDate => 'Start date';

  @override
  String get deadline => 'Deadline';

  @override
  String get ongoing => 'Ongoing';

  @override
  String get whoCanApply => 'Who can apply';

  @override
  String get freelancersAndAgencies => 'Freelancers & Agencies';

  @override
  String get freelancersOnly => 'Freelancers only';

  @override
  String get agenciesOnly => 'Agencies only';

  @override
  String get negotiable => 'Budget is negotiable';

  @override
  String get milestoneTitle => 'Title';

  @override
  String get milestoneDescription => 'Deliverables';

  @override
  String get milestoneAmount => 'Amount';

  @override
  String get totalAmount => 'Total amount';

  @override
  String get addMilestone => 'Add milestone';

  @override
  String get publishProject => 'Publish project';

  @override
  String get projectPublished => 'Project published successfully';

  @override
  String get jobCreateJob => 'Create job';

  @override
  String get jobDetails => 'Job details';

  @override
  String get jobBudgetAndDuration => 'Budget and duration';

  @override
  String get jobTitle => 'Job title';

  @override
  String get jobTitleHint => 'Add a descriptive title';

  @override
  String get jobDescription => 'Job description';

  @override
  String get jobSkills => 'Skills';

  @override
  String get jobSkillsHint => 'ex. UX Design, Web Development';

  @override
  String get jobTools => 'Tools';

  @override
  String get jobToolsHint => 'ex. Figma, Canva, Webflow';

  @override
  String get jobContractorCount => 'How many contractors?';

  @override
  String get jobApplicantType => 'Who can apply?';

  @override
  String get jobApplicantAll => 'All';

  @override
  String get jobApplicantFreelancers => 'Freelancers';

  @override
  String get jobApplicantAgencies => 'Agencies';

  @override
  String get jobBudgetType => 'Project type';

  @override
  String get jobOngoing => 'Ongoing';

  @override
  String get jobOneTime => 'One-time';

  @override
  String get jobPaymentFrequency => 'Payment frequency';

  @override
  String get jobHourly => 'Hourly';

  @override
  String get jobWeekly => 'Weekly';

  @override
  String get jobMonthly => 'Monthly';

  @override
  String get jobMinRate => 'Min. rate';

  @override
  String get jobMaxRate => 'Max. rate';

  @override
  String get jobMinBudget => 'Min. budget';

  @override
  String get jobMaxBudget => 'Max. budget';

  @override
  String get jobMaxHours => 'Max. hours/week';

  @override
  String get jobEstimatedDuration => 'Estimated duration';

  @override
  String get jobIndefinite => 'Indefinite duration';

  @override
  String get jobWeeks => 'weeks';

  @override
  String get jobMonths => 'months';

  @override
  String get jobCancel => 'Cancel';

  @override
  String get jobContinue => 'Continue';

  @override
  String get jobSave => 'Save';

  @override
  String get jobPublish => 'Publish';

  @override
  String get jobMyJobs => 'My Jobs';

  @override
  String get jobNoJobs => 'No jobs yet';

  @override
  String get jobNoJobsDesc =>
      'Create your first job posting to start finding talent.';

  @override
  String get jobStatusOpen => 'Open';

  @override
  String get jobStatusClosed => 'Closed';

  @override
  String get jobClose => 'Close';

  @override
  String get proposalPropose => 'Send a proposal';

  @override
  String get proposalCreate => 'Create a proposal';

  @override
  String get proposalTitle => 'Mission title';

  @override
  String get proposalTitleHint => 'e.g. Corporate website redesign';

  @override
  String get proposalDescription => 'Description';

  @override
  String get proposalDescriptionHint => 'Detail deliverables and scope of work';

  @override
  String get proposalAmount => 'Amount (€)';

  @override
  String get proposalAmountHint => '1500';

  @override
  String get proposalDeadline => 'Deadline';

  @override
  String get proposalRecipient => 'Recipient';

  @override
  String get proposalFrom => 'Proposal from';

  @override
  String get proposalTotalAmount => 'Total amount';

  @override
  String get proposalPending => 'Pending';

  @override
  String get proposalAccepted => 'Accepted';

  @override
  String get proposalDeclined => 'Declined';

  @override
  String get proposalAccept => 'Accept';

  @override
  String get proposalDecline => 'Decline';

  @override
  String get proposalSend => 'Send proposal';

  @override
  String get proposalModify => 'Counter-offer';

  @override
  String get proposalWithdrawn => 'Withdrawn';

  @override
  String get proposalAcceptedMessage => 'Proposal accepted';

  @override
  String get proposalDeclinedMessage => 'Proposal declined';

  @override
  String get proposalPaidMessage => 'Payment confirmed, mission in progress';

  @override
  String get proposalPaymentRequestedMessage => 'Payment requested';

  @override
  String get proposalCompletionRequestedMessage => 'Completion requested';

  @override
  String get proposalCompletedMessage => 'Mission completed';

  @override
  String get proposalCompletionRejectedMessage => 'Completion rejected';

  @override
  String get evaluationRequestMessage => 'Please leave a review';

  @override
  String get proposalNewMessage => 'New proposal';

  @override
  String get proposalModifiedMessage => 'Proposal modified';

  @override
  String get payNow => 'Pay now';

  @override
  String get confirmPayment => 'Confirm payment';

  @override
  String get paymentSimulation => 'Payment';

  @override
  String get paymentSuccess => 'Payment confirmed!';

  @override
  String get paymentSuccessDesc =>
      'The mission is now active. Redirecting to projects...';

  @override
  String get noActiveProjects => 'No active projects';

  @override
  String get noActiveProjectsDesc =>
      'Accepted proposals will appear here once paid.';

  @override
  String get projectStatusActive => 'Active';

  @override
  String get projectStatusCompleted => 'Completed';

  @override
  String get startProject => 'Start a project';

  @override
  String get callCalling => 'Calling...';

  @override
  String get callIncomingCall => 'Incoming call';

  @override
  String get callAudioCall => 'Audio call';

  @override
  String get callAccept => 'Accept';

  @override
  String get callDecline => 'Decline';

  @override
  String get callHangup => 'Hang up';

  @override
  String get callMute => 'Mute';

  @override
  String get callUnmute => 'Unmute';

  @override
  String get callEnded => 'Call ended';

  @override
  String get callMissed => 'Missed call';

  @override
  String get callStartCall => 'Start audio call';

  @override
  String get callRecipientOffline => 'Recipient is offline';

  @override
  String get callUserBusy => 'User is already in a call';

  @override
  String get callFailed => 'Call could not be started';

  @override
  String get callUnknownCaller => 'Unknown caller';
}
