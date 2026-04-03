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
  String get jobReopen => 'Reopen';

  @override
  String get jobDelete => 'Delete';

  @override
  String get jobDeleteConfirm =>
      'Are you sure you want to delete this job? This action cannot be undone.';

  @override
  String get jobDeleteSuccess => 'Job deleted successfully';

  @override
  String get jobReopenSuccess => 'Job reopened successfully';

  @override
  String get jobOfferDetails => 'Offer details';

  @override
  String get jobCandidates => 'Candidates';

  @override
  String get jobNoCandidates => 'No candidates yet';

  @override
  String get jobNoCandidatesDesc =>
      'Applications will appear here when candidates apply.';

  @override
  String get jobEditJob => 'Edit job';

  @override
  String get jobPostedOn => 'Posted on';

  @override
  String get jobDescriptionTypeText => 'Text';

  @override
  String get jobDescriptionTypeVideo => 'Video';

  @override
  String get jobDescriptionTypeBoth => 'Both';

  @override
  String get jobDescriptionType => 'Description type';

  @override
  String get jobAddVideo => 'Add a video';

  @override
  String get jobVideoUploading => 'Uploading video...';

  @override
  String get jobVideoUploaded => 'Video uploaded';

  @override
  String get jobUpdateSuccess => 'Job updated successfully';

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
  String get evaluationRequestMessage => 'Mission completed! Leave a review';

  @override
  String get leaveReview => 'Leave a review';

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

  @override
  String get callVideoCall => 'Video call';

  @override
  String get callStartVideoCall => 'Start video call';

  @override
  String get callCamera => 'Camera';

  @override
  String get callCameraOff => 'Camera off';

  @override
  String get callCameraOn => 'Camera on';

  @override
  String get callNoVideo => 'Camera is off';

  @override
  String get callIncomingVideoCall => 'Incoming video call';

  @override
  String get callTapToReturn => 'Tap to return to call';

  @override
  String get callMinimize => 'Minimize';

  @override
  String get drawerDashboard => 'Dashboard';

  @override
  String get drawerMessages => 'Messages';

  @override
  String get drawerProjects => 'Projects';

  @override
  String get drawerJobs => 'Job postings';

  @override
  String get drawerProfile => 'My profile';

  @override
  String get drawerFindFreelancers => 'Find freelancers';

  @override
  String get drawerFindAgencies => 'Find agencies';

  @override
  String get drawerFindReferrers => 'Find referrers';

  @override
  String get drawerLogout => 'Log out';

  @override
  String get drawerLogoutConfirm => 'Are you sure you want to log out?';

  @override
  String get drawerSwitchToReferrer => 'Business Referrer';

  @override
  String get drawerSwitchToFreelance => 'Freelance Dashboard';

  @override
  String get drawerPaymentInfo => 'Payment Info';

  @override
  String get drawerNotifications => 'Notifications';

  @override
  String get notifications => 'Notifications';

  @override
  String get noNotifications => 'No notifications yet';

  @override
  String get noNotificationsDesc =>
      'You\'ll be notified when something happens';

  @override
  String get markAllRead => 'Mark all read';

  @override
  String get proposalViewDetails => 'View details';

  @override
  String get paymentInfoTitle => 'Payment Information';

  @override
  String get paymentInfoSubtitle =>
      'Complete your information to receive payments for your projects.';

  @override
  String get paymentInfoIsBusiness => 'I operate as a registered business';

  @override
  String get paymentInfoIsBusinessDesc =>
      'Enable if your activity is operated through a registered company. Leave disabled if you operate under a sole proprietorship (freelance, independent).';

  @override
  String get paymentInfoPersonalInfo => 'Personal Information';

  @override
  String get paymentInfoLegalRep => 'Legal Representative';

  @override
  String get paymentInfoBusinessInfo => 'Business Information';

  @override
  String get paymentInfoBankAccount => 'Bank Account';

  @override
  String get paymentInfoFirstName => 'First name';

  @override
  String get paymentInfoLastName => 'Last name';

  @override
  String get paymentInfoDob => 'Date of birth';

  @override
  String get paymentInfoNationality => 'Nationality';

  @override
  String get paymentInfoAddress => 'Address';

  @override
  String get paymentInfoCity => 'City';

  @override
  String get paymentInfoPostalCode => 'Postal code';

  @override
  String get paymentInfoYourRole => 'Your role in the company';

  @override
  String get paymentInfoBusinessName => 'Business name';

  @override
  String get paymentInfoBusinessAddress => 'Business address';

  @override
  String get paymentInfoBusinessCity => 'Business city';

  @override
  String get paymentInfoBusinessPostalCode => 'Business postal code';

  @override
  String get paymentInfoBusinessCountry => 'Country of registration';

  @override
  String get paymentInfoTaxId => 'Tax ID';

  @override
  String get paymentInfoTaxIdHint => 'SIRET, EIN, VAT ID...';

  @override
  String get paymentInfoVatNumber => 'VAT number (optional)';

  @override
  String get paymentInfoVatNumberHint => 'EU VAT number (optional)';

  @override
  String get paymentInfoIban => 'IBAN';

  @override
  String get paymentInfoIbanHint => 'FR76 1234 5678 9012 3456 78';

  @override
  String get paymentInfoBic => 'BIC / SWIFT (optional)';

  @override
  String get paymentInfoBicHint => 'BNPAFRPP';

  @override
  String get paymentInfoIbanHelp =>
      'If your bank hasn\'t provided an IBAN, you can generate one at';

  @override
  String get paymentInfoNoIban => 'I don\'t have an IBAN';

  @override
  String get paymentInfoUseIban => 'I have an IBAN';

  @override
  String get paymentInfoAccountNumber => 'Account number';

  @override
  String get paymentInfoRoutingNumber => 'Routing number';

  @override
  String get paymentInfoAccountHolder => 'Account holder name';

  @override
  String get paymentInfoBankCountry => 'Bank country';

  @override
  String get paymentInfoSave => 'Save';

  @override
  String get paymentInfoSaved => 'Payment information saved';

  @override
  String get paymentInfoIncomplete =>
      'Complete your payment information to receive payments';

  @override
  String get paymentInfoRoleOwner => 'Owner';

  @override
  String get paymentInfoRoleCeo => 'CEO';

  @override
  String get paymentInfoRoleDirector => 'Director';

  @override
  String get paymentInfoRolePartner => 'Partner';

  @override
  String get paymentInfoRoleOther => 'Other';

  @override
  String get paymentInfoPhone => 'Phone number';

  @override
  String get paymentInfoActivitySector => 'Activity sector';

  @override
  String get paymentInfoBusinessPersons => 'Business representatives';

  @override
  String get paymentInfoSelfRepresentative => 'I am the legal representative';

  @override
  String get paymentInfoSelfDirector =>
      'The legal representative is the sole director';

  @override
  String get paymentInfoNoMajorOwners => 'No shareholder holds more than 25%';

  @override
  String get paymentInfoSelfExecutive =>
      'The legal representative is the sole executive';

  @override
  String get paymentInfoAddPerson => 'Add a person';

  @override
  String get paymentInfoPerson => 'Person';

  @override
  String get identityDocTitle => 'Identity verification';

  @override
  String get identityDocSubtitle =>
      'Upload a government-issued identity document for verification.';

  @override
  String get identityDocType => 'Document type';

  @override
  String get identityDocPending => 'Pending';

  @override
  String get identityDocVerified => 'Verified';

  @override
  String get identityDocRejected => 'Rejected';

  @override
  String get identityDocUploaded => 'Document uploaded successfully';

  @override
  String get identityDocUpload => 'Upload identity document';

  @override
  String get identityDocUploadDesc => 'Upload a clear photo of your document';

  @override
  String get identityDocPassport => 'Passport';

  @override
  String get identityDocIdCard => 'ID Card';

  @override
  String get identityDocDrivingLicense => 'Driving License';

  @override
  String get identityDocSinglePage => 'Single page upload';

  @override
  String get identityDocFrontAndBack => 'Front and back required';

  @override
  String get identityDocFrontSide => 'Front side';

  @override
  String get identityDocBackSide => 'Back side';

  @override
  String get identityDocReplace => 'Replace';

  @override
  String get identityDocSelectType => 'Select document type';

  @override
  String get identityDocPendingBanner => 'Your document is being reviewed';

  @override
  String get identityDocVerifiedBanner => 'Your identity has been verified';

  @override
  String get identityDocRejectedBanner => 'Your document was rejected';

  @override
  String get paymentInfoAddRepresentative => 'Add a representative';

  @override
  String get paymentInfoAddDirector => 'Add a director';

  @override
  String get paymentInfoAddOwner => 'Add a shareholder';

  @override
  String get paymentInfoAddExecutive => 'Add an executive';

  @override
  String get paymentInfoRepresentative => 'Representative';

  @override
  String get paymentInfoDirectorLabel => 'Director';

  @override
  String get paymentInfoOwnerLabel => 'Shareholder';

  @override
  String get paymentInfoExecutiveLabel => 'Executive';

  @override
  String get paymentInfoPersonTitle => 'Title';

  @override
  String get paymentInfoDateOfBirth => 'Date of birth';

  @override
  String get stripeRequirementsTitle => 'Additional information required';

  @override
  String get stripeRequirementsDesc => 'Please provide the following information to keep your account active.';

  @override
  String get stripeCompleteOnStripe => 'Complete on Stripe';

  @override
  String get walletTitle => 'Wallet';

  @override
  String get walletStripeAccount => 'Stripe account';

  @override
  String get walletCharges => 'Charges';

  @override
  String get walletPayouts => 'Payouts';

  @override
  String get walletEscrow => 'Escrow';

  @override
  String get walletAvailable => 'Available';

  @override
  String get walletTransferred => 'Transferred';

  @override
  String get walletRequestPayout => 'Withdraw';

  @override
  String get walletPayoutRequested => 'Payout requested successfully';

  @override
  String get walletTransactionHistory => 'Transaction history';

  @override
  String get walletNoTransactions => 'No transactions yet';

  @override
  String get drawerWallet => 'Wallet';

  @override
  String get reportMessage => 'Report this message';

  @override
  String get reportUser => 'Report this user';

  @override
  String get report => 'Report';

  @override
  String get selectReason => 'What\'s the issue?';

  @override
  String get reportDescription => 'Additional details';

  @override
  String get reportDescriptionHint => 'Describe the issue in detail...';

  @override
  String get submitReport => 'Submit report';

  @override
  String get reportSubmitting => 'Submitting...';

  @override
  String get reportSent => 'Report submitted. Our team will review it.';

  @override
  String get reportError => 'Failed to submit report.';

  @override
  String get reasonHarassment => 'Harassment or bullying';

  @override
  String get reasonFraud => 'Fraud or scam';

  @override
  String get reasonOffPlatformPayment => 'Payment outside platform';

  @override
  String get reasonSpam => 'Spam';

  @override
  String get reasonInappropriateContent => 'Inappropriate content';

  @override
  String get reasonFakeProfile => 'Fake or misleading profile';

  @override
  String get reasonUnprofessionalBehavior => 'Unprofessional behavior';

  @override
  String get reasonOther => 'Other';

  @override
  String get loadMore => 'Load more';

  @override
  String get candidateDetail => 'Application';

  @override
  String get applicationMessage => 'Application message';

  @override
  String get applicationVideo => 'Presentation video';

  @override
  String get opportunities => 'Opportunities';

  @override
  String get noOpportunities => 'No opportunities at the moment';

  @override
  String get jobNotFound => 'Job not found';

  @override
  String get budgetTypeOneShot => 'One-time project';

  @override
  String get budgetTypeLongTerm => 'Long-term collaboration';

  @override
  String get myApplications => 'My applications';

  @override
  String get noApplications => 'You haven\'t applied to any job yet';

  @override
  String get withdrawApplicationTitle => 'Withdraw application?';

  @override
  String get withdrawAction => 'Withdraw';

  @override
  String get applications => 'Applications';

  @override
  String get noApplicationsYet => 'No applications yet';

  @override
  String get applyAction => 'Apply';

  @override
  String get alreadyApplied => 'Already applied';

  @override
  String get applyTitle => 'Apply';

  @override
  String get applyMessageLabel => 'Your message (optional)';

  @override
  String get applyMessageHint => 'Why are you the right candidate?';

  @override
  String get applyAddVideo => 'Add a video';

  @override
  String get applyUploading => 'Uploading...';

  @override
  String get applyRemoveVideo => 'Remove video';

  @override
  String get applySubmit => 'Send my application';

  @override
  String get applicationSent => 'Application sent!';

  @override
  String get applicationSendError => 'Failed to send application';

  @override
  String get videoUploadFailed => 'Video upload failed. Please try again.';

  @override
  String jobTotalApplicants(int count) {
    return '$count applicants';
  }

  @override
  String jobNewApplicants(int count) {
    return '$count new';
  }

  @override
  String candidateOf(int current, int total) {
    return '$current of $total';
  }

  @override
  String uploadProgress(int percent) {
    return '$percent%';
  }
}
