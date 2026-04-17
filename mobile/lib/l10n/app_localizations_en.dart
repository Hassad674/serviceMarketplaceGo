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
  String searchTotalEarnedLine(String amount) {
    return '$amount earned';
  }

  @override
  String searchCompletedProjects(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '# projects',
      one: '# project',
    );
    return '$_temp0';
  }

  @override
  String get searchNegotiableBadge => 'Negotiable';

  @override
  String get searchLoadMore => 'Load more';

  @override
  String get searchEmptyTitle => 'No results';

  @override
  String get searchEmptyDescription => 'Try broadening your filters.';

  @override
  String get searchEmptyCta => 'Reset filters';

  @override
  String get searchFiltersRadius => 'Radius (km)';

  @override
  String get searchFiltersSkillsHint => 'Type a skill and press Enter';

  @override
  String get searchDidYouMean => 'Did you mean';

  @override
  String searchResultsCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count results',
      one: '1 result',
      zero: 'No results',
    );
    return '$_temp0';
  }

  @override
  String get searchFiltersTitle => 'Filters';

  @override
  String get searchFiltersAvailability => 'Availability';

  @override
  String get searchFiltersAvailableNow => 'Now';

  @override
  String get searchFiltersAvailableSoon => 'Soon';

  @override
  String get searchFiltersAll => 'All';

  @override
  String get searchFiltersPrice => 'Daily rate';

  @override
  String get searchFiltersPriceMin => 'Min';

  @override
  String get searchFiltersPriceMax => 'Max';

  @override
  String get searchFiltersLocation => 'Location';

  @override
  String get searchFiltersLocationCity => 'City';

  @override
  String get searchFiltersLocationCountry => 'Country';

  @override
  String get searchFiltersLanguages => 'Languages';

  @override
  String get searchFiltersExpertise => 'Expertise';

  @override
  String get searchFiltersSkills => 'Skills';

  @override
  String get searchFiltersRating => 'Minimum rating';

  @override
  String get searchFiltersWorkMode => 'Work mode';

  @override
  String get searchFiltersRemote => 'Remote';

  @override
  String get searchFiltersOnSite => 'On site';

  @override
  String get searchFiltersHybrid => 'Hybrid';

  @override
  String get searchFiltersApply => 'Apply';

  @override
  String get searchFiltersReset => 'Reset';

  @override
  String get searchFiltersOpen => 'Filters';

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
  String get reviewTitleClientToProvider => 'Leave a review';

  @override
  String get reviewTitleProviderToClient => 'Review the client';

  @override
  String get reviewSubtitleProviderToClient =>
      'How was your experience with this client?';

  @override
  String get reviewErrorWindowClosed =>
      'The review window has closed (14 days after mission completion).';

  @override
  String get reviewErrorNotParticipant =>
      'Only the participants of this mission can leave a review.';

  @override
  String get proposalNewMessage => 'New proposal';

  @override
  String get proposalModifiedMessage => 'Proposal modified';

  @override
  String get milestoneActionFailed =>
      'Could not update milestone. Please try again.';

  @override
  String milestoneSequenceLabel(int sequence) {
    return 'Milestone $sequence';
  }

  @override
  String get milestoneFundTitle => 'Fund this milestone';

  @override
  String get milestoneFundDescription =>
      'Release the escrow amount for this milestone so the provider can start working on it.';

  @override
  String get milestoneFundConfirm => 'Fund milestone';

  @override
  String get milestoneSubmitTitle => 'Submit for approval';

  @override
  String get milestoneSubmitDescription =>
      'Mark this milestone as delivered. The client will be notified and asked to approve.';

  @override
  String get milestoneSubmitConfirm => 'Submit milestone';

  @override
  String get milestoneApproveTitle => 'Approve milestone';

  @override
  String get milestoneApproveDescription =>
      'Release the escrow to the provider and move to the next milestone (if any).';

  @override
  String get milestoneApproveConfirm => 'Approve and pay';

  @override
  String get milestoneRejectTitle => 'Request revisions';

  @override
  String get milestoneRejectDescription =>
      'Send the milestone back to the provider for revisions. The escrow stays in hold.';

  @override
  String get milestoneRejectConfirm => 'Request revisions';

  @override
  String get submitWork => 'Submit work';

  @override
  String get approveWork => 'Approve work';

  @override
  String get requestRevisions => 'Request revisions';

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
  String get drawerTeam => 'Team';

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
  String get identityDocPassportDesc =>
      'Valid passport, national ID card, or driver\'s license';

  @override
  String get identityDocProofOfAddressDesc =>
      'Utility bill (less than 3 months old), bank statement, or certificate of residence';

  @override
  String get identityDocBusinessRegDesc =>
      'Certificate of incorporation, articles of organization, or official equivalent for your country';

  @override
  String get identityDocProofOfLivenessDesc =>
      'Live photo of your face to confirm your identity';

  @override
  String get identityDocProofOfRegistrationDesc =>
      'Certificate of registration, incorporation document, or official proof from your country\'s business registry';

  @override
  String get stripeRequirementsTitle => 'Additional information required';

  @override
  String get stripeRequirementsDesc =>
      'Please provide the following information to keep your account active.';

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
  String get reasonFraudOrScam => 'Fraud or scam';

  @override
  String get reasonMisleadingDescription => 'Misleading description';

  @override
  String get reportJob => 'Report this job';

  @override
  String get reportApplication => 'Report this application';

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
  String get applicantTypeMismatch =>
      'Your account type cannot apply to this job';

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

  @override
  String creditsRemaining(int count) {
    return '$count credits remaining';
  }

  @override
  String get noCreditsLeft => 'You have no application credits left';

  @override
  String get creditsHowItWorks => 'How do credits work?';

  @override
  String get creditsExplanation1 => 'Each application costs 1 credit';

  @override
  String get creditsExplanation2 =>
      'Every Monday, your balance is reset to 10 credits if it\'s below 10';

  @override
  String get creditsExplanation3 =>
      'Each signed mission earns you 5 bonus credits';

  @override
  String get creditsExplanation4 =>
      'Your balance can go up to 50 credits maximum';

  @override
  String get noCreditsCannotApply =>
      'You need credits to apply to this opportunity';

  @override
  String get paymentInfoSetup => 'Set up payments';

  @override
  String get paymentInfoComplete => 'Complete verification';

  @override
  String get paymentInfoEdit => 'Edit payment info';

  @override
  String get paymentInfoActive => 'Account fully active';

  @override
  String get paymentInfoActiveDesc =>
      'You can receive payments and transfer funds.';

  @override
  String get paymentInfoPending => 'Verification in progress';

  @override
  String paymentInfoPendingDesc(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count items to complete',
      one: '$count item to complete',
    );
    return '$_temp0';
  }

  @override
  String get paymentInfoNotConfigured => 'Not configured';

  @override
  String get paymentInfoNotConfiguredDesc =>
      'Set up your payment account to start receiving funds.';

  @override
  String get paymentInfoCharges => 'Payments';

  @override
  String get paymentInfoPayouts => 'Transfers';

  @override
  String get kycBannerPendingTitle => 'Set up your payment info';

  @override
  String kycBannerPendingBody(int days) {
    return 'You have funds pending. Complete setup within $days days to avoid restrictions.';
  }

  @override
  String get kycBannerRestrictedTitle => 'Account restricted';

  @override
  String get kycBannerRestrictedBody =>
      'You cannot apply to jobs or create proposals until you complete your payment setup.';

  @override
  String get kycBannerCta => 'Set up now';

  @override
  String get disputeOpenTitle => 'Dispute in progress';

  @override
  String get disputeNegotiationTitle => 'Negotiation in progress';

  @override
  String get disputeEscalatedTitle => 'Under mediation';

  @override
  String get disputeResolvedTitle => 'Dispute resolved';

  @override
  String get disputeCounterPropose => 'Make a proposal';

  @override
  String get disputeCancel => 'Cancel dispute';

  @override
  String get disputeOpenBtn => 'Report a problem';

  @override
  String get disputeStatusOpen => 'Dispute in progress';

  @override
  String get disputeStatusNegotiation => 'Negotiation in progress';

  @override
  String get disputeStatusEscalated => 'Under mediation';

  @override
  String get disputeStatusResolved => 'Dispute resolved';

  @override
  String get disputeStatusCancelled => 'Dispute cancelled';

  @override
  String get disputeReasonWorkNotConforming => 'Work does not conform to scope';

  @override
  String get disputeReasonNonDelivery => 'Non-delivery';

  @override
  String get disputeReasonInsufficientQuality => 'Insufficient quality';

  @override
  String get disputeReasonClientGhosting => 'Client unresponsive';

  @override
  String get disputeReasonScopeCreep => 'Scope creep';

  @override
  String get disputeReasonRefusalToValidate =>
      'Refusal to validate without reason';

  @override
  String get disputeReasonHarassment => 'Harassment';

  @override
  String get disputeReasonOther => 'Other';

  @override
  String get disputeReasonLabel => 'Reason';

  @override
  String get disputeReasonPlaceholder => 'Select a reason';

  @override
  String get disputeAmountLabel => 'What are you requesting?';

  @override
  String disputeTotalRefund(String amount) {
    return 'Full refund ($amount)';
  }

  @override
  String disputeTotalRelease(String amount) {
    return 'Full fund release ($amount)';
  }

  @override
  String get disputePartialAmount => 'Partial amount';

  @override
  String get disputeMessageToPartyLabel => 'Message to the other party';

  @override
  String get disputeMessageToPartyHint =>
      'This message will be visible in the conversation. Explain your request clearly.';

  @override
  String get disputeMessageToPartyPlaceholder =>
      'Explain what you expect and why...';

  @override
  String get disputeDescriptionLabel =>
      'Detailed description for mediation (optional)';

  @override
  String get disputeDescriptionHint =>
      'This will only be read by the mediation team if the dispute is escalated.';

  @override
  String get disputeDescriptionPlaceholder =>
      'Additional context, timeline of events, evidence descriptions...';

  @override
  String get disputeFormWarning =>
      'Opening a dispute freezes the funds until resolution. The other party will be notified.';

  @override
  String get disputeSubmit => 'Submit dispute';

  @override
  String get disputeAccept => 'Accept';

  @override
  String get disputeReject => 'Reject';

  @override
  String get disputeCounterSubmit => 'Send proposal';

  @override
  String get disputeAddFiles => 'Add files';

  @override
  String get disputeCancelBtn => 'Cancel';

  @override
  String get disputeViewDetails => 'View details';

  @override
  String get disputeReportProblem => 'Report a problem';

  @override
  String get disputeCounterSplitLabel => 'Proposed split';

  @override
  String get disputeCounterMessageLabel => 'Message (optional)';

  @override
  String get disputeCounterMessagePlaceholder => 'Explain your proposal...';

  @override
  String get disputeRequestedAmount => 'requested';

  @override
  String disputeDaysLeft(int days) {
    return '$days days left before escalation';
  }

  @override
  String get disputeEscalationSoon => 'Escalation imminent';

  @override
  String get disputeLastProposal => 'Last proposal';

  @override
  String disputeSplit(String client, String provider) {
    return '$client to client, $provider to provider';
  }

  @override
  String get disputeResolution => 'Resolution';

  @override
  String get disputeInProgress => 'A dispute is in progress on this mission';

  @override
  String get disputeClient => 'Client';

  @override
  String get disputeProvider => 'Provider';

  @override
  String get disputeOpenedLabel => 'Dispute opened';

  @override
  String get disputeCounterProposalLabel => 'Proposal';

  @override
  String get disputeCounterAcceptedLabel => 'Proposal accepted';

  @override
  String get disputeCounterRejectedLabel => 'Proposal rejected';

  @override
  String get disputeEscalatedLabel => 'Escalated to mediation';

  @override
  String get disputeResolvedLabel => 'Dispute resolved';

  @override
  String get disputeCancelledLabel => 'Dispute cancelled';

  @override
  String get disputeAutoResolvedLabel => 'Dispute auto-resolved';

  @override
  String get disputeCancellationRequestedLabel => 'Cancellation request';

  @override
  String get disputeCancellationRefusedLabel => 'Cancellation refused';

  @override
  String get disputeYourLastProposalRefused => 'Your last proposal was refused';

  @override
  String get disputeEscalatedNegotiationStillOpen =>
      'The dispute is now under mediation. Until the admin renders a final decision, you can still reach an amicable agreement together.';

  @override
  String get disputeCancellationRequestPending =>
      'Cancellation request pending';

  @override
  String get disputeCancellationRequestWaiting =>
      'Waiting for the other party to accept or refuse your cancellation request.';

  @override
  String get disputeCancellationRequestConsent =>
      'The other party is asking to cancel this dispute. Your consent is required.';

  @override
  String get disputeCancellationRequestSent =>
      'Cancellation request sent. Waiting for the other party\'s response.';

  @override
  String get disputeAcceptCancellation => 'Accept cancellation';

  @override
  String get disputeRefuseCancellation => 'Refuse';

  @override
  String get disputeDecisionTitle => 'Mediation decision';

  @override
  String disputeDecisionYourShare(int percent, String amount) {
    return 'You receive $percent% — $amount';
  }

  @override
  String get disputeDecisionMessage => 'Message from the admin';

  @override
  String disputeDecisionRenderedOn(String date) {
    return 'Rendered on $date';
  }

  @override
  String get disputeCancelledTitle => 'Dispute cancelled';

  @override
  String get disputeCancelledSubtitle =>
      'The dispute was cancelled by mutual agreement.';

  @override
  String get projectStatusDisputed => 'Disputed';

  @override
  String get permissionDenied =>
      'You do not have permission to perform this action';

  @override
  String get permissionDeniedSend =>
      'You do not have permission to send messages';

  @override
  String get permissionDeniedWithdraw =>
      'You do not have permission to request payouts';

  @override
  String get permissionDeniedEdit =>
      'You do not have permission to edit this resource';

  @override
  String get teamScreenTitle => 'Team';

  @override
  String get teamMembersSection => 'Members';

  @override
  String get teamNoMembers => 'No members';

  @override
  String get teamNoOrganization => 'No organization';

  @override
  String get teamNoOrganizationDescription =>
      'You are not attached to any organization yet.';

  @override
  String get teamLoadError => 'Could not load team';

  @override
  String get teamRetry => 'Retry';

  @override
  String get teamInviteButton => 'Invite';

  @override
  String get teamInviteDialogTitle => 'Invite a new member';

  @override
  String get teamInviteDialogDescription =>
      'Send a secure invitation link to a new teammate. They will set their own password on first sign-in.';

  @override
  String get teamInviteEmailLabel => 'Email';

  @override
  String get teamInviteEmailHint => 'teammate@example.com';

  @override
  String get teamInviteFirstNameLabel => 'First name';

  @override
  String get teamInviteLastNameLabel => 'Last name';

  @override
  String get teamInviteTitleLabel => 'Title (optional)';

  @override
  String get teamInviteTitleHint => 'e.g. Project Manager';

  @override
  String get teamInviteRoleLabel => 'Role';

  @override
  String get teamInviteRoleHelp =>
      'You can change the role later from the members list.';

  @override
  String get teamInviteRoleAdmin => 'Admin';

  @override
  String get teamInviteRoleMember => 'Member';

  @override
  String get teamInviteRoleViewer => 'Viewer';

  @override
  String get teamInviteSendButton => 'Send invitation';

  @override
  String get teamInviteCancelButton => 'Cancel';

  @override
  String teamInviteSuccess(String email) {
    return 'Invitation sent to $email';
  }

  @override
  String get teamInviteEmailRequired => 'Email is required';

  @override
  String get teamInviteEmailInvalid => 'Please enter a valid email address';

  @override
  String get teamInviteFirstNameRequired => 'First name is required';

  @override
  String get teamInviteLastNameRequired => 'Last name is required';

  @override
  String get teamInviteFailed => 'Could not send invitation. Please try again.';

  @override
  String get teamRolePermissionsTitle => 'Roles & permissions';

  @override
  String get teamRolePermissionsSubtitle =>
      'What each role can do in this organization.';

  @override
  String get teamRolePermissionsReadOnlyTitle => 'Read-only view';

  @override
  String get teamRolePermissionsReadOnlyDescription =>
      'Only the Owner can modify role permissions. Other members see the matrix for reference.';

  @override
  String get teamRolePermissionsLoadError => 'Could not load role permissions';

  @override
  String get teamRolePermissionsModifiedBadge => 'Modified';

  @override
  String teamRolePermissionsPending(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count changes pending',
      one: '1 change pending',
    );
    return '$_temp0';
  }

  @override
  String get teamRolePermissionsDiscard => 'Discard';

  @override
  String get teamRolePermissionsSave => 'Save';

  @override
  String get teamRolePermissionsConfirmTitle => 'Confirm role changes';

  @override
  String teamRolePermissionsConfirmDescription(int count, String role) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'This will update $count permissions for the $role role. Affected members will be signed out and must sign in again.',
      one:
          'This will update 1 permission for the $role role. Affected members will be signed out and must sign in again.',
    );
    return '$_temp0';
  }

  @override
  String get teamRolePermissionsConfirmButton => 'Save changes';

  @override
  String get teamRolePermissionsCancelButton => 'Cancel';

  @override
  String teamRolePermissionsSaveSuccess(int affected) {
    return 'Permissions updated. $affected session(s) invalidated.';
  }

  @override
  String get teamRolePermissionsSaveFailed =>
      'Could not save permissions. Please try again.';

  @override
  String get teamRolePermissionsOwnerExclusiveTitle =>
      'Owner-exclusive permissions';

  @override
  String get teamRolePermissionsOwnerExclusiveDescription =>
      'These permissions cannot be customized and are reserved for the organization Owner.';

  @override
  String get teamRolePermissionsStateGrantedOverride => 'Granted';

  @override
  String get teamRolePermissionsStateRevokedOverride => 'Revoked';

  @override
  String get teamRolePermissionsStateLocked => 'Locked';

  @override
  String get teamRolePermissionRoleAdmin => 'Admin';

  @override
  String get teamRolePermissionRoleMember => 'Member';

  @override
  String get teamRolePermissionRoleViewer => 'Viewer';

  @override
  String get teamRolePermissionRoleOwner => 'Owner';

  @override
  String get teamRolePermissionGroupTeam => 'Team';

  @override
  String get teamRolePermissionGroupOrgProfile => 'Public profile';

  @override
  String get teamRolePermissionGroupJobs => 'Jobs';

  @override
  String get teamRolePermissionGroupProposals => 'Proposals';

  @override
  String get teamRolePermissionGroupMessaging => 'Messaging';

  @override
  String get teamRolePermissionGroupReviews => 'Reviews';

  @override
  String get teamRolePermissionGroupWallet => 'Wallet';

  @override
  String get teamRolePermissionGroupBilling => 'Billing';

  @override
  String get teamRolePermissionGroupKyc => 'KYC';

  @override
  String get teamRolePermissionGroupDanger => 'Danger zone';

  @override
  String get teamMemberActions => 'Actions';

  @override
  String get teamMemberEdit => 'Edit';

  @override
  String get teamMemberRemove => 'Remove';

  @override
  String get teamMemberFallbackName => 'Member';

  @override
  String get teamMemberCannotEditSelf => 'You cannot edit your own membership.';

  @override
  String get teamMemberCannotRemoveSelf => 'Use Leave organization instead.';

  @override
  String teamEditMemberDialogTitle(String name) {
    return 'Edit $name';
  }

  @override
  String get teamEditMemberRoleLabel => 'Role';

  @override
  String get teamEditMemberTitleLabel => 'Title';

  @override
  String get teamEditMemberTitleHint => 'e.g. Project Manager';

  @override
  String get teamEditMemberSave => 'Save changes';

  @override
  String get teamEditMemberSuccess => 'Member updated';

  @override
  String get teamEditMemberFailed =>
      'Could not update member. Please try again.';

  @override
  String get teamEditMemberNoChanges => 'No changes to save.';

  @override
  String get teamRemoveMemberDialogTitle => 'Remove member';

  @override
  String teamRemoveMemberConfirm(String name) {
    return 'Are you sure you want to remove $name from the organization? They will lose access immediately.';
  }

  @override
  String get teamRemoveMemberConfirmButton => 'Remove';

  @override
  String teamRemoveMemberSuccess(String name) {
    return '$name has been removed';
  }

  @override
  String get teamRemoveMemberFailed =>
      'Could not remove member. Please try again.';

  @override
  String get teamInvitationsSection => 'Pending invitations';

  @override
  String teamInvitationsCountLabel(int count) {
    return 'Pending invitations ($count)';
  }

  @override
  String get teamInvitationsEmpty => 'No pending invitations.';

  @override
  String get teamInvitationsLoadFailed => 'Could not load invitations.';

  @override
  String teamInvitationSentAgo(int days) {
    return 'Sent $days day(s) ago';
  }

  @override
  String get teamInvitationSentToday => 'Sent today';

  @override
  String teamInvitationExpiresIn(int days) {
    return 'Expires in $days day(s)';
  }

  @override
  String get teamInvitationExpired => 'Expired';

  @override
  String get teamInvitationCancelTooltip => 'Cancel invitation';

  @override
  String get teamInvitationResendTooltip => 'Resend invitation';

  @override
  String get teamInvitationCancelDialogTitle => 'Cancel invitation';

  @override
  String teamInvitationCancelDialogBody(String email) {
    return 'Cancel the invitation sent to $email? They will no longer be able to join with this link.';
  }

  @override
  String get teamInvitationCancelConfirm => 'Cancel invitation';

  @override
  String get teamInvitationCancelKeep => 'Keep';

  @override
  String get teamInvitationCancelSuccess => 'Invitation cancelled';

  @override
  String get teamInvitationCancelFailed =>
      'Could not cancel invitation. Please try again.';

  @override
  String get teamInvitationResendSuccess => 'Invitation resent';

  @override
  String get teamInvitationResendFailed =>
      'Could not resend invitation. Please try again.';

  @override
  String get teamLeaveAction => 'Leave organization';

  @override
  String get teamLeaveDialogTitle => 'Leave organization';

  @override
  String get teamLeaveDialogBody =>
      'You will lose access to this organization\'s data and conversations. This cannot be undone.';

  @override
  String get teamLeaveConfirmHint => 'Type LEAVE to confirm';

  @override
  String get teamLeaveConfirmKeyword => 'LEAVE';

  @override
  String get teamLeaveConfirmButton => 'Leave organization';

  @override
  String get teamLeaveSuccess => 'You have left the organization';

  @override
  String get teamLeaveFailed =>
      'Could not leave the organization. Please try again.';

  @override
  String get teamTransferAction => 'Transfer ownership';

  @override
  String get teamTransferDialogTitle => 'Transfer ownership';

  @override
  String get teamTransferDialogBody =>
      'Choose an Admin who will become the new Owner of this organization. You will become an Admin once they accept. This cannot be undone.';

  @override
  String get teamTransferTargetLabel => 'New owner';

  @override
  String get teamTransferNoEligible =>
      'There are no Admins available. Promote a member to Admin first.';

  @override
  String get teamTransferConfirmButton => 'Send transfer request';

  @override
  String get teamTransferSuccess => 'Transfer request sent';

  @override
  String get teamTransferFailed =>
      'Could not initiate transfer. Please try again.';

  @override
  String get teamPendingTransferTargetTitle =>
      'You have been offered ownership';

  @override
  String get teamPendingTransferTargetBody =>
      'Accept to become the new Owner of this organization. Decline to keep your current role.';

  @override
  String get teamPendingTransferInitiatorTitle => 'Ownership transfer pending';

  @override
  String get teamPendingTransferInitiatorBody =>
      'Waiting for the target Admin to accept ownership of this organization.';

  @override
  String get teamPendingTransferReadOnlyTitle =>
      'Ownership transfer in progress';

  @override
  String get teamPendingTransferReadOnlyBody =>
      'An ownership transfer is currently pending for this organization.';

  @override
  String teamPendingTransferExpiresOn(String date) {
    return 'Expires on $date';
  }

  @override
  String get teamPendingTransferAccept => 'Accept';

  @override
  String get teamPendingTransferDecline => 'Decline';

  @override
  String get teamPendingTransferCancel => 'Cancel transfer';

  @override
  String get teamPendingTransferAcceptSuccess =>
      'You are now the Owner of this organization';

  @override
  String get teamPendingTransferAcceptFailed =>
      'Could not accept transfer. Please try again.';

  @override
  String get teamPendingTransferDeclineDialogTitle => 'Decline transfer';

  @override
  String get teamPendingTransferDeclineDialogBody =>
      'Decline the ownership transfer? The current Owner will keep their role.';

  @override
  String get teamPendingTransferDeclineSuccess => 'Transfer declined';

  @override
  String get teamPendingTransferDeclineFailed =>
      'Could not decline transfer. Please try again.';

  @override
  String get teamPendingTransferCancelDialogTitle => 'Cancel transfer';

  @override
  String get teamPendingTransferCancelDialogBody =>
      'Cancel the pending ownership transfer? You will remain the Owner.';

  @override
  String get teamPendingTransferCancelSuccess => 'Transfer cancelled';

  @override
  String get teamPendingTransferCancelFailed =>
      'Could not cancel transfer. Please try again.';

  @override
  String get teamRoleOwner => 'Owner';

  @override
  String get teamRoleAdmin => 'Admin';

  @override
  String get teamRoleMember => 'Member';

  @override
  String get teamRoleViewer => 'Viewer';

  @override
  String get expertiseSectionTitle => 'Areas of expertise';

  @override
  String expertiseSectionSubtitle(int max) {
    return 'Pick up to $max domains that showcase what you do best';
  }

  @override
  String get expertiseAddDomains => 'Add domains';

  @override
  String get expertiseSave => 'Save';

  @override
  String get expertiseSaving => 'Saving...';

  @override
  String expertiseMaxReached(int max) {
    return 'You\'ve reached the maximum of $max domains';
  }

  @override
  String expertiseCounter(int count, int max) {
    return '$count/$max selected';
  }

  @override
  String get expertiseEmptyPrivate => 'No expertise selected yet.';

  @override
  String get expertiseErrorGeneric =>
      'Could not save your expertise. Please try again.';

  @override
  String get expertiseDomainDevelopment => 'Development';

  @override
  String get expertiseDomainDataAiMl => 'Data, AI & Machine Learning';

  @override
  String get expertiseDomainDesignUiUx => 'Design & UI/UX';

  @override
  String get expertiseDomainDesign3dAnimation => '3D Design & Animation';

  @override
  String get expertiseDomainVideoMotion => 'Video & Motion';

  @override
  String get expertiseDomainPhotoAudiovisual => 'Photo & Audiovisual';

  @override
  String get expertiseDomainMarketingGrowth => 'Marketing & Growth';

  @override
  String get expertiseDomainWritingTranslation => 'Writing & Translation';

  @override
  String get expertiseDomainBusinessDevSales => 'Business Development & Sales';

  @override
  String get expertiseDomainConsultingStrategy => 'Consulting & Strategy';

  @override
  String get expertiseDomainProductUxResearch => 'Product & UX Research';

  @override
  String get expertiseDomainOpsAdminSupport => 'Ops, Admin & Support';

  @override
  String get expertiseDomainLegal => 'Legal';

  @override
  String get expertiseDomainFinanceAccounting => 'Finance & Accounting';

  @override
  String get expertiseDomainHrRecruitment => 'HR & Recruitment';

  @override
  String get skillsDisplaySectionTitle => 'Skills';

  @override
  String skillsDisplayMoreSuffix(int count) {
    return '+$count';
  }

  @override
  String get skillsSectionTitle => 'Skills';

  @override
  String skillsSectionSubtitle(int max) {
    return 'Up to $max skills';
  }

  @override
  String get skillsEmpty => 'No skills added yet';

  @override
  String get skillsEditButton => 'Edit my skills';

  @override
  String get skillsModalTitle => 'My skills';

  @override
  String get skillsSearchPlaceholder => 'Search for a skill...';

  @override
  String skillsCounter(int count, int max) {
    return '$count / $max';
  }

  @override
  String get skillsBrowseHeading => 'Browse by domain';

  @override
  String get skillsSave => 'Save';

  @override
  String get skillsSaving => 'Saving...';

  @override
  String get skillsCancel => 'Cancel';

  @override
  String skillsCreateNew(String query) {
    return 'Create \"$query\"';
  }

  @override
  String skillsUsageCount(int count) {
    return '$count pros';
  }

  @override
  String skillsErrorTooMany(int max) {
    return 'You\'ve reached the limit of $max skills';
  }

  @override
  String get skillsErrorDisabled => 'Unavailable for this account type';

  @override
  String get skillsErrorGeneric => 'Something went wrong';

  @override
  String get tier1AvailabilitySectionTitle => 'Availability';

  @override
  String get tier1AvailabilityStatusAvailableNow => 'Available now';

  @override
  String get tier1AvailabilityStatusAvailableSoon => 'Available soon';

  @override
  String get tier1AvailabilityStatusNotAvailable => 'Unavailable';

  @override
  String get tier1AvailabilityReferrerTitle =>
      'Availability as a business referrer';

  @override
  String get tier1AvailabilityDirectLabel => 'Services';

  @override
  String get tier1AvailabilityReferrerLabel => 'Referrer';

  @override
  String get tier1AvailabilityEditButton => 'Update availability';

  @override
  String get tier1LocationSectionTitle => 'Location';

  @override
  String get tier1LocationCityLabel => 'City';

  @override
  String get tier1LocationCityPlaceholder => 'Paris';

  @override
  String get tier1LocationCountryLabel => 'Country';

  @override
  String get tier1LocationCountryPlaceholder => 'Select a country';

  @override
  String get tier1LocationWorkModeLabel => 'Work mode';

  @override
  String get tier1LocationWorkModeRemote => 'Remote';

  @override
  String get tier1LocationWorkModeOnSite => 'On-site';

  @override
  String get tier1LocationWorkModeHybrid => 'Hybrid';

  @override
  String get tier1LocationTravelRadiusLabel => 'Travel radius (km)';

  @override
  String tier1LocationTravelRadiusShort(int km) {
    return 'Up to $km km';
  }

  @override
  String get tier1LocationTravelRadiusPlaceholder => 'e.g. 50';

  @override
  String get tier1LocationEmpty => 'Add your city to help clients find you';

  @override
  String get tier1LocationEditButton => 'Update location';

  @override
  String get tier1LocationCityAutocompletePlaceholder => 'Search for a city…';

  @override
  String get tier1LocationCityAutocompleteHint =>
      'Type at least 2 characters to search';

  @override
  String get tier1LocationCityAutocompleteEmpty => 'No city found';

  @override
  String get tier1LanguagesSectionTitle => 'Languages';

  @override
  String get tier1LanguagesProfessionalLabel => 'Professional';

  @override
  String get tier1LanguagesConversationalLabel => 'Conversational';

  @override
  String get tier1LanguagesSearchPlaceholder => 'Search a language...';

  @override
  String get tier1LanguagesEmpty => 'Declare the languages you work in';

  @override
  String get tier1LanguagesEditButton => 'Update languages';

  @override
  String tier1LanguagesCountLabel(int count) {
    return '$count selected';
  }

  @override
  String get tier1LanguagesNoResults => 'No language found';

  @override
  String get tier1LanguagesClearAll => 'Clear all';

  @override
  String get tier1LanguagesProfessionalHelp =>
      'I can deliver work in these languages.';

  @override
  String get tier1LanguagesConversationalHelp =>
      'I can chat but not deliver in these languages.';

  @override
  String get tier1PricingSectionTitle => 'Pricing';

  @override
  String get tier1PricingDirectSectionTitle => 'Pricing';

  @override
  String get tier1PricingReferralSectionTitle => 'Referral pricing';

  @override
  String get tier1PricingEmpty => 'No pricing declared yet';

  @override
  String get tier1PricingEditButton => 'Update my pricing';

  @override
  String get tier1PricingModalTitle => 'My pricing';

  @override
  String get tier1PricingDirectModalTitle => 'Edit pricing';

  @override
  String get tier1PricingReferralModalTitle => 'Edit referral pricing';

  @override
  String get tier1PricingKindDirect => 'Direct service';

  @override
  String get tier1PricingKindReferral => 'Business referrer';

  @override
  String get tier1PricingNegotiableLabel => 'Is it negotiable?';

  @override
  String get tier1PricingNegotiableYes => 'Yes';

  @override
  String get tier1PricingNegotiableNo => 'No';

  @override
  String get tier1PricingNegotiableBadge => 'negotiable';

  @override
  String get tier1PricingTypeDaily => 'Daily rate';

  @override
  String get tier1PricingTypeHourly => 'Hourly rate';

  @override
  String get tier1PricingTypeProjectFrom => 'From (per project)';

  @override
  String get tier1PricingTypeProjectRange => 'Range (per project)';

  @override
  String get tier1PricingTypeCommissionPct => 'Commission percentage';

  @override
  String get tier1PricingTypeCommissionFlat => 'Flat commission';

  @override
  String get tier1PricingMinLabel => 'Min amount';

  @override
  String get tier1PricingMaxLabel => 'Max amount';

  @override
  String get tier1PricingCurrencyLabel => 'Currency';

  @override
  String get tier1PricingNoteLabel => 'Note';

  @override
  String get tier1PricingNotePlaceholder => 'Negotiable depending on scope...';

  @override
  String get tier1PricingPreviewHeading => 'Card preview';

  @override
  String get tier1PricingEmptyPreview => '–';

  @override
  String get tier1PricingDeleteKind => 'Remove this row';

  @override
  String get tier1PricingEnableReferralRow => 'Add a business-referrer row';

  @override
  String get tier1Save => 'Save';

  @override
  String get tier1Saving => 'Saving...';

  @override
  String get tier1Cancel => 'Cancel';

  @override
  String get tier1Delete => 'Delete';

  @override
  String get tier1Close => 'Close';

  @override
  String get tier1ErrorGeneric => 'Something went wrong';

  @override
  String get tier1ErrorPricingInvalidAmount => 'Enter a valid amount';

  @override
  String get tier1ErrorLocationRequireCity => 'City is required';

  @override
  String get projectHistory => 'Project history';

  @override
  String get referrerProjectHistoryEmpty => 'No deals referred yet';

  @override
  String get socialLinks => 'Social networks';

  @override
  String get editSocialLinks => 'Edit social links';

  @override
  String get noSocialLinks => 'No social links added yet';

  @override
  String get socialLinksSaved => 'Social links saved';

  @override
  String get socialLinksSaveError => 'Failed to save social links';

  @override
  String get socialLinkEnterUrl => 'Enter URL';

  @override
  String get socialLinkLinkedin => 'LinkedIn';

  @override
  String get socialLinkInstagram => 'Instagram';

  @override
  String get socialLinkYoutube => 'YouTube';

  @override
  String get socialLinkTwitter => 'Twitter';

  @override
  String get socialLinkGithub => 'GitHub';

  @override
  String get socialLinkWebsite => 'Website';
}
