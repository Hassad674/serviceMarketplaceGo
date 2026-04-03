// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for French (`fr`).
class AppLocalizationsFr extends AppLocalizations {
  AppLocalizationsFr([String locale = 'fr']) : super(locale);

  @override
  String get appTitle => 'Marketplace Service';

  @override
  String get signIn => 'Se connecter';

  @override
  String get signUp => 'S\'inscrire';

  @override
  String get signOut => 'Se déconnecter';

  @override
  String get email => 'Email';

  @override
  String get emailHint => 'vous@exemple.com';

  @override
  String get password => 'Mot de passe';

  @override
  String get passwordHint => 'Votre mot de passe';

  @override
  String get confirmPassword => 'Confirmer le mot de passe';

  @override
  String get confirmPasswordHint => 'Confirmez votre mot de passe';

  @override
  String get passwordRequirements =>
      'Minimum 8 caractères avec majuscule, minuscule et chiffre';

  @override
  String get forgotPassword => 'Mot de passe oublié ?';

  @override
  String get noAccount => 'Pas encore de compte ?';

  @override
  String get alreadyRegistered => 'Déjà inscrit ?';

  @override
  String get changeProfile => 'Changer de profil';

  @override
  String get signingIn => 'Connexion...';

  @override
  String get signingUp => 'Inscription...';

  @override
  String get agencyName => 'Nom de l\'agence';

  @override
  String get agencyNameHint => 'Nom commercial de votre agence';

  @override
  String get companyName => 'Nom de l\'entreprise';

  @override
  String get companyNameHint => 'Nom de votre entreprise';

  @override
  String get firstName => 'Prénom';

  @override
  String get firstNameHint => 'Jean';

  @override
  String get lastName => 'Nom';

  @override
  String get lastNameHint => 'Dupont';

  @override
  String get createAgencyAccount => 'Créer mon compte agence';

  @override
  String get createFreelanceAccount => 'Créer mon compte freelance';

  @override
  String get createEnterpriseAccount => 'Créer mon compte entreprise';

  @override
  String get roleSelectionTitle => 'Rejoignez la marketplace';

  @override
  String get roleSelectionSubtitle => 'Choisissez votre profil professionnel';

  @override
  String get roleAgency => 'Agence';

  @override
  String get roleAgencyDesc =>
      'Gérez vos missions, votre équipe et votre visibilité.';

  @override
  String get roleFreelance => 'Freelance / Apporteur d\'affaire';

  @override
  String get roleFreelanceDesc =>
      'Gérez vos missions et développez votre activité.';

  @override
  String get roleEnterprise => 'Entreprise';

  @override
  String get roleEnterpriseDesc =>
      'Trouvez les meilleurs prestataires pour vos projets.';

  @override
  String get welcomeBack => 'Bonjour,';

  @override
  String get dashboard => 'Tableau de bord';

  @override
  String get home => 'Accueil';

  @override
  String get messages => 'Messages';

  @override
  String get missions => 'Missions';

  @override
  String get profile => 'Profil';

  @override
  String get myProfile => 'Mon Profil';

  @override
  String get settings => 'Paramètres';

  @override
  String get activeMissions => 'Missions en cours';

  @override
  String get activeContracts => 'Contrats actifs';

  @override
  String get unreadMessages => 'Messages non lus';

  @override
  String get conversations => 'Conversations';

  @override
  String get monthlyRevenue => 'Revenus du mois';

  @override
  String get thisMonth => 'Ce mois-ci';

  @override
  String get activeProjects => 'Projets en cours';

  @override
  String get totalBudget => 'Budget total';

  @override
  String get spentThisMonth => 'Dépensé ce mois';

  @override
  String get referrals => 'Mises en relation';

  @override
  String get pendingResponse => 'En attente de réponse';

  @override
  String get completedMissions => 'Missions terminées';

  @override
  String get totalHistory => 'Total historique';

  @override
  String get commissions => 'Commissions';

  @override
  String get totalEarned => 'Total gagné';

  @override
  String get businessReferrerMode => 'Mode Apporteur d\'affaires';

  @override
  String get freelanceDashboard => 'Dashboard Freelance';

  @override
  String get referrerMode => 'Mode Apporteur';

  @override
  String get presentationVideo => 'Vidéo de présentation';

  @override
  String get noVideo => 'Aucune vidéo de présentation';

  @override
  String get addVideo => 'Ajouter une vidéo';

  @override
  String get videoUpdated => 'Vidéo mise à jour';

  @override
  String get photoUpdated => 'Photo mise à jour';

  @override
  String get addPhoto => 'Ajouter une photo';

  @override
  String get takePhoto => 'Prendre une photo';

  @override
  String get chooseFromGallery => 'Choisir depuis la galerie';

  @override
  String get chooseFile => 'Choisir un fichier';

  @override
  String get upload => 'Envoyer';

  @override
  String get cancel => 'Annuler';

  @override
  String get save => 'Enregistrer';

  @override
  String get fileTooLarge => 'Fichier trop volumineux';

  @override
  String get uploadError => 'Échec de l\'envoi';

  @override
  String maxSize(String size) {
    return 'Taille maximum : $size';
  }

  @override
  String get about => 'À propos';

  @override
  String get professionalTitle => 'Titre professionnel';

  @override
  String get noTitle => 'Aucun titre ajouté';

  @override
  String get unexpectedError => 'Une erreur inattendue est survenue';

  @override
  String get connectionError => 'Erreur de connexion. Vérifiez votre internet.';

  @override
  String get timeoutError => 'La requête a expiré. Réessayez.';

  @override
  String get serverError => 'Erreur serveur. Réessayez plus tard.';

  @override
  String get comingSoon => 'Bientôt disponible';

  @override
  String get fieldRequired => 'Ce champ est requis';

  @override
  String get invalidEmail => 'Adresse email invalide';

  @override
  String get passwordTooShort => 'Minimum 8 caractères';

  @override
  String get passwordNoUppercase => 'Au moins une majuscule';

  @override
  String get passwordNoLowercase => 'Au moins une minuscule';

  @override
  String get passwordNoDigit => 'Au moins un chiffre';

  @override
  String get passwordsDoNotMatch => 'Les mots de passe ne correspondent pas';

  @override
  String get search => 'Rechercher';

  @override
  String get findFreelancers => 'Trouver des Freelances';

  @override
  String get findAgencies => 'Trouver des Agences';

  @override
  String get findReferrers => 'Trouver des Apporteurs';

  @override
  String get noProfilesFound => 'Aucun profil trouvé';

  @override
  String get searchTryAgain =>
      'Réessayez plus tard ou modifiez votre recherche.';

  @override
  String get couldNotLoadProfiles =>
      'Impossible de charger les profils. Vérifiez votre connexion.';

  @override
  String get couldNotLoadProfile => 'Impossible de charger le profil';

  @override
  String get checkConnectionRetry => 'Vérifiez votre connexion et réessayez.';

  @override
  String get somethingWentWrong => 'Une erreur est survenue';

  @override
  String get retry => 'Réessayer';

  @override
  String get tapToPlay => 'Appuyez pour lire';

  @override
  String get replaceVideo => 'Remplacer la vidéo';

  @override
  String get removeVideo => 'Supprimer la vidéo';

  @override
  String get removeVideoConfirmTitle => 'Supprimer la vidéo';

  @override
  String get removeVideoConfirmMessage =>
      'Êtes-vous sûr de vouloir supprimer votre vidéo de présentation ?';

  @override
  String get remove => 'Supprimer';

  @override
  String get darkMode => 'Mode sombre';

  @override
  String get aboutPlaceholder => 'Parlez de vous et de votre expertise';

  @override
  String get aboutEditHint => 'Parlez de vous...';

  @override
  String get aboutUpdated => 'À propos mis à jour';

  @override
  String get titlePlaceholder => 'Ajoutez votre titre professionnel';

  @override
  String get videoRemoved => 'Vidéo supprimée';

  @override
  String get couldNotOpenVideo => 'Impossible d\'ouvrir la vidéo';

  @override
  String get messagingSearchHint => 'Rechercher une conversation...';

  @override
  String get messagingNoMessages => 'Aucun message dans cette conversation';

  @override
  String get messagingNoConversations => 'Aucune conversation';

  @override
  String get messagingWriteMessage => 'Écrivez votre message...';

  @override
  String get messagingOnline => 'En ligne';

  @override
  String get messagingOffline => 'Hors ligne';

  @override
  String get messagingAllRoles => 'Tous';

  @override
  String get messagingAgency => 'Agence';

  @override
  String get messagingFreelancer => 'Freelance/Apporteur';

  @override
  String get messagingEnterprise => 'Entreprise';

  @override
  String get messagingConversationNotFound => 'Conversation introuvable';

  @override
  String get messagingSendMessage => 'Envoyer un message';

  @override
  String messagingTyping(String name) {
    return '$name est en train d\'écrire...';
  }

  @override
  String get messagingTypingShort => 'est en train d\'écrire...';

  @override
  String get messagingEdited => 'modifié';

  @override
  String get messagingDeleted => 'Ce message a été supprimé';

  @override
  String get messagingDelivered => 'Distribué';

  @override
  String get messagingRead => 'Lu';

  @override
  String get messagingSent => 'Envoyé';

  @override
  String get messagingSending => 'Envoi en cours...';

  @override
  String get messagingReconnecting => 'Reconnexion...';

  @override
  String get messagingEditMessage => 'Modifier le message';

  @override
  String get messagingDeleteMessage => 'Supprimer le message';

  @override
  String get messagingDeleteConfirm =>
      'Êtes-vous sûr de vouloir supprimer ce message ?';

  @override
  String get messagingFileUpload => 'Envoyer un fichier';

  @override
  String get messagingStartConversation =>
      'Aucun message. Lancez la conversation !';

  @override
  String get messagingLoadMore => 'Charger plus';

  @override
  String get messagingVoiceMessage => 'Message vocal';

  @override
  String get messagingRecording => 'Enregistrement...';

  @override
  String get messagingCancelRecording => 'Annuler';

  @override
  String get messagingMicrophonePermission => 'Accès au microphone requis';

  @override
  String get messagingReply => 'Répondre';

  @override
  String messagingReplyingTo(String name) {
    return 'Réponse à $name';
  }

  @override
  String get projects => 'Projets';

  @override
  String get createProject => 'Créer un projet';

  @override
  String get noProjects => 'Aucun projet';

  @override
  String get noProjectsDesc => 'Créez votre premier projet pour commencer.';

  @override
  String get paymentType => 'Type de paiement';

  @override
  String get invoiceBilling => 'Facturation';

  @override
  String get invoiceBillingDesc =>
      'Facturation classique avec cycles de paiement flexibles.';

  @override
  String get escrowPayments => 'Paiement sécurisé';

  @override
  String get escrowPaymentsDesc =>
      'Fonds sécurisés jusqu\'à validation des jalons.';

  @override
  String get projectStructure => 'Structure';

  @override
  String get milestone => 'Jalon';

  @override
  String get oneTime => 'Paiement unique';

  @override
  String get billingDetails => 'Détails de facturation';

  @override
  String get fixed => 'Fixe';

  @override
  String get hourly => 'Horaire';

  @override
  String get rate => 'Tarif';

  @override
  String get frequency => 'Fréquence';

  @override
  String get weekly => 'Hebdomadaire';

  @override
  String get biWeekly => 'Bimensuel';

  @override
  String get monthly => 'Mensuel';

  @override
  String get projectDetails => 'Détails';

  @override
  String get projectTitle => 'Titre du projet';

  @override
  String get projectDescription => 'Description';

  @override
  String get requiredSkills => 'Compétences requises';

  @override
  String get addSkillHint => 'Tapez une compétence et appuyez sur ajouter';

  @override
  String get timeline => 'Calendrier';

  @override
  String get startDate => 'Date de début';

  @override
  String get deadline => 'Date limite';

  @override
  String get ongoing => 'En continu';

  @override
  String get whoCanApply => 'Qui peut postuler';

  @override
  String get freelancersAndAgencies => 'Freelances & Agences';

  @override
  String get freelancersOnly => 'Freelances uniquement';

  @override
  String get agenciesOnly => 'Agences uniquement';

  @override
  String get negotiable => 'Le budget est négociable';

  @override
  String get milestoneTitle => 'Titre';

  @override
  String get milestoneDescription => 'Livrables';

  @override
  String get milestoneAmount => 'Montant';

  @override
  String get totalAmount => 'Montant total';

  @override
  String get addMilestone => 'Ajouter un jalon';

  @override
  String get publishProject => 'Publier le projet';

  @override
  String get projectPublished => 'Projet publié avec succès';

  @override
  String get jobCreateJob => 'Créer une offre';

  @override
  String get jobDetails => 'Détails de l\'offre';

  @override
  String get jobBudgetAndDuration => 'Budget et durée';

  @override
  String get jobTitle => 'Titre de l\'offre';

  @override
  String get jobTitleHint => 'Ajoutez un titre descriptif';

  @override
  String get jobDescription => 'Description de l\'offre';

  @override
  String get jobSkills => 'Compétences';

  @override
  String get jobSkillsHint => 'ex. Design UX, Développement Web';

  @override
  String get jobTools => 'Outils';

  @override
  String get jobToolsHint => 'ex. Figma, Canva, Webflow';

  @override
  String get jobContractorCount => 'Combien de prestataires ?';

  @override
  String get jobApplicantType => 'Qui peut postuler ?';

  @override
  String get jobApplicantAll => 'Tous';

  @override
  String get jobApplicantFreelancers => 'Freelances';

  @override
  String get jobApplicantAgencies => 'Agences';

  @override
  String get jobBudgetType => 'Type de projet';

  @override
  String get jobOngoing => 'Long terme';

  @override
  String get jobOneTime => 'Ponctuel';

  @override
  String get jobPaymentFrequency => 'Fréquence de paiement';

  @override
  String get jobHourly => 'Horaire';

  @override
  String get jobWeekly => 'Hebdomadaire';

  @override
  String get jobMonthly => 'Mensuel';

  @override
  String get jobMinRate => 'Tarif min.';

  @override
  String get jobMaxRate => 'Tarif max.';

  @override
  String get jobMinBudget => 'Budget min.';

  @override
  String get jobMaxBudget => 'Budget max.';

  @override
  String get jobMaxHours => 'Heures max./semaine';

  @override
  String get jobEstimatedDuration => 'Durée estimée';

  @override
  String get jobIndefinite => 'Durée indéterminée';

  @override
  String get jobWeeks => 'semaines';

  @override
  String get jobMonths => 'mois';

  @override
  String get jobCancel => 'Annuler';

  @override
  String get jobContinue => 'Continuer';

  @override
  String get jobSave => 'Enregistrer';

  @override
  String get jobPublish => 'Publier';

  @override
  String get jobMyJobs => 'Mes offres';

  @override
  String get jobNoJobs => 'Aucune offre';

  @override
  String get jobNoJobsDesc =>
      'Créez votre première offre d\'emploi pour trouver des talents.';

  @override
  String get jobStatusOpen => 'Ouverte';

  @override
  String get jobStatusClosed => 'Fermée';

  @override
  String get jobClose => 'Fermer';

  @override
  String get jobReopen => 'Rouvrir';

  @override
  String get jobDelete => 'Supprimer';

  @override
  String get jobDeleteConfirm =>
      'Êtes-vous sûr de vouloir supprimer cette offre ? Cette action est irréversible.';

  @override
  String get jobDeleteSuccess => 'Offre supprimée avec succès';

  @override
  String get jobReopenSuccess => 'Offre rouverte avec succès';

  @override
  String get jobOfferDetails => 'Détails de l\'offre';

  @override
  String get jobCandidates => 'Candidatures';

  @override
  String get jobNoCandidates => 'Aucune candidature';

  @override
  String get jobNoCandidatesDesc =>
      'Les candidatures apparaîtront ici lorsque des candidats postuleront.';

  @override
  String get jobEditJob => 'Modifier l\'offre';

  @override
  String get jobPostedOn => 'Publiée le';

  @override
  String get jobDescriptionTypeText => 'Texte';

  @override
  String get jobDescriptionTypeVideo => 'Vidéo';

  @override
  String get jobDescriptionTypeBoth => 'Les deux';

  @override
  String get jobDescriptionType => 'Type de description';

  @override
  String get jobAddVideo => 'Ajouter une vidéo';

  @override
  String get jobVideoUploading => 'Envoi de la vidéo...';

  @override
  String get jobVideoUploaded => 'Vidéo envoyée';

  @override
  String get jobUpdateSuccess => 'Offre mise à jour avec succès';

  @override
  String get proposalPropose => 'Envoyer une proposition';

  @override
  String get proposalCreate => 'Créer une proposition';

  @override
  String get proposalTitle => 'Titre de la mission';

  @override
  String get proposalTitleHint => 'ex. Refonte du site web corporate';

  @override
  String get proposalDescription => 'Description';

  @override
  String get proposalDescriptionHint =>
      'Détaillez les livrables et le scope du travail';

  @override
  String get proposalAmount => 'Montant (€)';

  @override
  String get proposalAmountHint => '1500';

  @override
  String get proposalDeadline => 'Date limite';

  @override
  String get proposalRecipient => 'Destinataire';

  @override
  String get proposalFrom => 'Proposition de';

  @override
  String get proposalTotalAmount => 'Montant total';

  @override
  String get proposalPending => 'En attente';

  @override
  String get proposalAccepted => 'Acceptée';

  @override
  String get proposalDeclined => 'Refusée';

  @override
  String get proposalAccept => 'Accepter';

  @override
  String get proposalDecline => 'Refuser';

  @override
  String get proposalSend => 'Envoyer la proposition';

  @override
  String get proposalModify => 'Contre-proposition';

  @override
  String get proposalWithdrawn => 'Retirée';

  @override
  String get proposalAcceptedMessage => 'Proposition acceptée';

  @override
  String get proposalDeclinedMessage => 'Proposition refusée';

  @override
  String get proposalPaidMessage => 'Paiement confirmé, mission en cours';

  @override
  String get proposalPaymentRequestedMessage => 'Paiement demandé';

  @override
  String get proposalCompletionRequestedMessage => 'Achèvement demandé';

  @override
  String get proposalCompletedMessage => 'Mission terminée';

  @override
  String get proposalCompletionRejectedMessage => 'Achèvement refusé';

  @override
  String get evaluationRequestMessage =>
      'Mission terminée ! Laissez votre avis';

  @override
  String get leaveReview => 'Évaluer';

  @override
  String get proposalNewMessage => 'Nouvelle proposition';

  @override
  String get proposalModifiedMessage => 'Proposition modifiée';

  @override
  String get payNow => 'Payer maintenant';

  @override
  String get confirmPayment => 'Confirmer le paiement';

  @override
  String get paymentSimulation => 'Paiement';

  @override
  String get paymentSuccess => 'Paiement confirmé !';

  @override
  String get paymentSuccessDesc =>
      'La mission est maintenant active. Redirection vers les projets...';

  @override
  String get noActiveProjects => 'Aucun projet actif';

  @override
  String get noActiveProjectsDesc =>
      'Les propositions acceptées apparaîtront ici une fois payées.';

  @override
  String get projectStatusActive => 'Actif';

  @override
  String get projectStatusCompleted => 'Terminé';

  @override
  String get startProject => 'Proposer un projet';

  @override
  String get callCalling => 'Appel en cours...';

  @override
  String get callIncomingCall => 'Appel entrant';

  @override
  String get callAudioCall => 'Appel audio';

  @override
  String get callAccept => 'Accepter';

  @override
  String get callDecline => 'Refuser';

  @override
  String get callHangup => 'Raccrocher';

  @override
  String get callMute => 'Couper le micro';

  @override
  String get callUnmute => 'Activer le micro';

  @override
  String get callEnded => 'Appel terminé';

  @override
  String get callMissed => 'Appel manqué';

  @override
  String get callStartCall => 'Démarrer un appel audio';

  @override
  String get callRecipientOffline => 'Le destinataire est hors ligne';

  @override
  String get callUserBusy => 'L\'utilisateur est déjà en appel';

  @override
  String get callFailed => 'L\'appel n\'a pas pu être lancé';

  @override
  String get callUnknownCaller => 'Appelant inconnu';

  @override
  String get callVideoCall => 'Appel vidéo';

  @override
  String get callStartVideoCall => 'Démarrer un appel vidéo';

  @override
  String get callCamera => 'Caméra';

  @override
  String get callCameraOff => 'Caméra désactivée';

  @override
  String get callCameraOn => 'Caméra activée';

  @override
  String get callNoVideo => 'La caméra est désactivée';

  @override
  String get callIncomingVideoCall => 'Appel vidéo entrant';

  @override
  String get callTapToReturn => 'Appuyez pour revenir à l\'appel';

  @override
  String get callMinimize => 'Réduire';

  @override
  String get drawerDashboard => 'Tableau de bord';

  @override
  String get drawerMessages => 'Messages';

  @override
  String get drawerProjects => 'Projets';

  @override
  String get drawerJobs => 'Offres d\'emploi';

  @override
  String get drawerProfile => 'Mon profil';

  @override
  String get drawerFindFreelancers => 'Trouver des freelances';

  @override
  String get drawerFindAgencies => 'Trouver des agences';

  @override
  String get drawerFindReferrers => 'Trouver des apporteurs';

  @override
  String get drawerLogout => 'Se déconnecter';

  @override
  String get drawerLogoutConfirm => 'Voulez-vous vraiment vous déconnecter ?';

  @override
  String get drawerSwitchToReferrer => 'Apporteur d\'affaires';

  @override
  String get drawerSwitchToFreelance => 'Dashboard Freelance';

  @override
  String get drawerPaymentInfo => 'Infos paiement';

  @override
  String get drawerNotifications => 'Notifications';

  @override
  String get notifications => 'Notifications';

  @override
  String get noNotifications => 'Aucune notification';

  @override
  String get noNotificationsDesc =>
      'Vous serez notifié lorsque quelque chose se passe';

  @override
  String get markAllRead => 'Tout marquer comme lu';

  @override
  String get proposalViewDetails => 'Voir les détails';

  @override
  String get paymentInfoTitle => 'Informations de paiement';

  @override
  String get paymentInfoSubtitle =>
      'Complétez vos informations pour recevoir les paiements de vos missions.';

  @override
  String get paymentInfoIsBusiness =>
      'J\'exerce en tant qu\'entreprise enregistrée';

  @override
  String get paymentInfoIsBusinessDesc =>
      'Activez si votre activité est exercée par une société enregistrée. Laissez désactivé si vous possédez un statut juridique en nom propre (freelance, indépendant).';

  @override
  String get paymentInfoPersonalInfo => 'Informations personnelles';

  @override
  String get paymentInfoLegalRep => 'Représentant légal';

  @override
  String get paymentInfoBusinessInfo => 'Informations de l\'entreprise';

  @override
  String get paymentInfoBankAccount => 'Compte bancaire';

  @override
  String get paymentInfoFirstName => 'Prénom';

  @override
  String get paymentInfoLastName => 'Nom';

  @override
  String get paymentInfoDob => 'Date de naissance';

  @override
  String get paymentInfoNationality => 'Nationalité';

  @override
  String get paymentInfoAddress => 'Adresse';

  @override
  String get paymentInfoCity => 'Ville';

  @override
  String get paymentInfoPostalCode => 'Code postal';

  @override
  String get paymentInfoYourRole => 'Votre rôle dans l\'entreprise';

  @override
  String get paymentInfoBusinessName => 'Raison sociale';

  @override
  String get paymentInfoBusinessAddress => 'Adresse du siège';

  @override
  String get paymentInfoBusinessCity => 'Ville du siège';

  @override
  String get paymentInfoBusinessPostalCode => 'Code postal du siège';

  @override
  String get paymentInfoBusinessCountry => 'Pays d\'enregistrement';

  @override
  String get paymentInfoTaxId => 'Numéro d\'identification fiscale';

  @override
  String get paymentInfoTaxIdHint => 'SIRET, EIN, numéro TVA...';

  @override
  String get paymentInfoVatNumber => 'Numéro de TVA (optionnel)';

  @override
  String get paymentInfoVatNumberHint =>
      'Numéro de TVA intracommunautaire (optionnel)';

  @override
  String get paymentInfoIban => 'IBAN';

  @override
  String get paymentInfoIbanHint => 'FR76 1234 5678 9012 3456 78';

  @override
  String get paymentInfoBic => 'BIC / SWIFT (optionnel)';

  @override
  String get paymentInfoBicHint => 'BNPAFRPP';

  @override
  String get paymentInfoIbanHelp =>
      'Si votre banque ne vous a pas fourni d\'IBAN, vous pouvez en générer un sur';

  @override
  String get paymentInfoNoIban => 'Je n\'ai pas d\'IBAN';

  @override
  String get paymentInfoUseIban => 'J\'ai un IBAN';

  @override
  String get paymentInfoAccountNumber => 'Numéro de compte';

  @override
  String get paymentInfoRoutingNumber => 'Numéro de routage';

  @override
  String get paymentInfoAccountHolder => 'Titulaire du compte';

  @override
  String get paymentInfoBankCountry => 'Pays de la banque';

  @override
  String get paymentInfoSave => 'Enregistrer';

  @override
  String get paymentInfoSaved => 'Informations de paiement enregistrées';

  @override
  String get paymentInfoIncomplete =>
      'Complétez vos informations de paiement pour recevoir vos paiements';

  @override
  String get paymentInfoRoleOwner => 'Propriétaire';

  @override
  String get paymentInfoRoleCeo => 'PDG / Gérant';

  @override
  String get paymentInfoRoleDirector => 'Directeur';

  @override
  String get paymentInfoRolePartner => 'Associé';

  @override
  String get paymentInfoRoleOther => 'Autre';

  @override
  String get paymentInfoPhone => 'Numéro de téléphone';

  @override
  String get paymentInfoActivitySector => 'Secteur d\'activité';

  @override
  String get paymentInfoBusinessPersons => 'Représentants de l\'entreprise';

  @override
  String get paymentInfoSelfRepresentative => 'Je suis le représentant légal';

  @override
  String get paymentInfoSelfDirector =>
      'Le représentant légal est le seul dirigeant';

  @override
  String get paymentInfoNoMajorOwners =>
      'Aucun actionnaire ne détient plus de 25%';

  @override
  String get paymentInfoSelfExecutive =>
      'Le représentant légal est le seul cadre dirigeant';

  @override
  String get paymentInfoAddPerson => 'Ajouter une personne';

  @override
  String get paymentInfoPerson => 'Personne';

  @override
  String get identityDocTitle => 'Vérification d\'identité';

  @override
  String get identityDocSubtitle =>
      'Téléversez un document d\'identité officiel pour la vérification.';

  @override
  String get identityDocType => 'Type de document';

  @override
  String get identityDocPending => 'En attente';

  @override
  String get identityDocVerified => 'Vérifié';

  @override
  String get identityDocRejected => 'Rejeté';

  @override
  String get identityDocUploaded => 'Document téléversé avec succès';

  @override
  String get identityDocUpload => 'Téléverser un document d\'identité';

  @override
  String get identityDocUploadDesc =>
      'Téléversez une photo nette de votre document';

  @override
  String get identityDocPassport => 'Passeport';

  @override
  String get identityDocIdCard => 'Carte d\'identité';

  @override
  String get identityDocDrivingLicense => 'Permis de conduire';

  @override
  String get identityDocSinglePage => 'Page unique';

  @override
  String get identityDocFrontAndBack => 'Recto et verso requis';

  @override
  String get identityDocFrontSide => 'Recto';

  @override
  String get identityDocBackSide => 'Verso';

  @override
  String get identityDocReplace => 'Remplacer';

  @override
  String get identityDocSelectType => 'Choisissez le type de document';

  @override
  String get identityDocPendingBanner =>
      'Votre document est en cours de vérification';

  @override
  String get identityDocVerifiedBanner => 'Votre identité a été vérifiée';

  @override
  String get identityDocRejectedBanner => 'Votre document a été rejeté';

  @override
  String get identityDocPassportDesc =>
      'Passeport, carte d\'identité nationale ou permis de conduire en cours de validité';

  @override
  String get identityDocProofOfAddressDesc =>
      'Facture de moins de 3 mois (électricité, eau, internet), relevé bancaire ou attestation de résidence';

  @override
  String get identityDocBusinessRegDesc =>
      'KBIS, extrait K, certificat d\'incorporation ou équivalent officiel de votre pays';

  @override
  String get identityDocProofOfLivenessDesc =>
      'Photo de votre visage prise en direct pour confirmer votre identité';

  @override
  String get identityDocProofOfRegistrationDesc =>
      'Certificat d\'enregistrement, document d\'incorporation ou preuve officielle du registre des entreprises de votre pays';

  @override
  String get paymentInfoAddRepresentative => 'Ajouter un représentant';

  @override
  String get paymentInfoAddDirector => 'Ajouter un dirigeant';

  @override
  String get paymentInfoAddOwner => 'Ajouter un actionnaire';

  @override
  String get paymentInfoAddExecutive => 'Ajouter un cadre dirigeant';

  @override
  String get paymentInfoRepresentative => 'Représentant';

  @override
  String get paymentInfoDirectorLabel => 'Dirigeant';

  @override
  String get paymentInfoOwnerLabel => 'Actionnaire';

  @override
  String get paymentInfoExecutiveLabel => 'Cadre dirigeant';

  @override
  String get paymentInfoPersonTitle => 'Titre';

  @override
  String get paymentInfoDateOfBirth => 'Date de naissance';

  @override
  String get stripeRequirementsTitle => 'Informations supplémentaires requises';

  @override
  String get stripeRequirementsDesc =>
      'Veuillez fournir les informations suivantes pour maintenir votre compte actif.';

  @override
  String get stripeCompleteOnStripe => 'Compléter sur Stripe';

  @override
  String get walletTitle => 'Portefeuille';

  @override
  String get walletStripeAccount => 'Compte Stripe';

  @override
  String get walletCharges => 'Paiements';

  @override
  String get walletPayouts => 'Virements';

  @override
  String get walletEscrow => 'Séquestre';

  @override
  String get walletAvailable => 'Disponible';

  @override
  String get walletTransferred => 'Transféré';

  @override
  String get walletRequestPayout => 'Retirer';

  @override
  String get walletPayoutRequested => 'Demande de virement effectuée';

  @override
  String get walletTransactionHistory => 'Historique des transactions';

  @override
  String get walletNoTransactions => 'Aucune transaction';

  @override
  String get drawerWallet => 'Portefeuille';

  @override
  String get reportMessage => 'Signaler ce message';

  @override
  String get reportUser => 'Signaler cet utilisateur';

  @override
  String get report => 'Signaler';

  @override
  String get selectReason => 'Quel est le problème ?';

  @override
  String get reportDescription => 'Détails supplémentaires';

  @override
  String get reportDescriptionHint => 'Décrivez le problème en détail...';

  @override
  String get submitReport => 'Envoyer le signalement';

  @override
  String get reportSubmitting => 'Envoi en cours...';

  @override
  String get reportSent => 'Signalement envoyé. Notre équipe va l\'examiner.';

  @override
  String get reportError => 'Échec de l\'envoi du signalement.';

  @override
  String get reasonHarassment => 'Harcèlement ou intimidation';

  @override
  String get reasonFraud => 'Fraude ou arnaque';

  @override
  String get reasonOffPlatformPayment => 'Paiement hors plateforme';

  @override
  String get reasonSpam => 'Spam';

  @override
  String get reasonInappropriateContent => 'Contenu inapproprié';

  @override
  String get reasonFakeProfile => 'Profil faux ou trompeur';

  @override
  String get reasonUnprofessionalBehavior => 'Comportement non professionnel';

  @override
  String get reasonOther => 'Autre';

  @override
  String get loadMore => 'Voir plus';

  @override
  String get candidateDetail => 'Candidature';

  @override
  String get applicationMessage => 'Message de candidature';

  @override
  String get applicationVideo => 'Vidéo de présentation';

  @override
  String get opportunities => 'Opportunités';

  @override
  String get noOpportunities => 'Aucune opportunité pour le moment';

  @override
  String get jobNotFound => 'Offre introuvable';

  @override
  String get budgetTypeOneShot => 'Projet ponctuel';

  @override
  String get budgetTypeLongTerm => 'Collaboration long terme';

  @override
  String get myApplications => 'Mes candidatures';

  @override
  String get noApplications => 'Vous n\'avez postulé à aucune offre';

  @override
  String get withdrawApplicationTitle => 'Retirer la candidature ?';

  @override
  String get withdrawAction => 'Retirer';

  @override
  String get applications => 'Candidatures';

  @override
  String get noApplicationsYet => 'Aucune candidature pour le moment';

  @override
  String get applyAction => 'Postuler';

  @override
  String get alreadyApplied => 'Déjà postulé';

  @override
  String get applicantTypeMismatch =>
      'Votre type de compte ne peut pas postuler à cette offre';

  @override
  String get applyTitle => 'Postuler';

  @override
  String get applyMessageLabel => 'Votre message (optionnel)';

  @override
  String get applyMessageHint => 'Pourquoi êtes-vous le bon candidat ?';

  @override
  String get applyAddVideo => 'Ajouter une vidéo';

  @override
  String get applyUploading => 'Envoi en cours...';

  @override
  String get applyRemoveVideo => 'Supprimer la vidéo';

  @override
  String get applySubmit => 'Envoyer ma candidature';

  @override
  String get applicationSent => 'Candidature envoyée !';

  @override
  String get applicationSendError => 'Erreur lors de l\'envoi';

  @override
  String get videoUploadFailed =>
      'Échec de l\'envoi de la vidéo. Veuillez réessayer.';

  @override
  String jobTotalApplicants(int count) {
    return '$count candidats';
  }

  @override
  String jobNewApplicants(int count) {
    return '$count nouveaux';
  }

  @override
  String candidateOf(int current, int total) {
    return '$current sur $total';
  }

  @override
  String uploadProgress(int percent) {
    return '$percent%';
  }
}
