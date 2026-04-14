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
  String get reviewTitleClientToProvider => 'Laisser un avis';

  @override
  String get reviewTitleProviderToClient => 'Évaluer le client';

  @override
  String get reviewSubtitleProviderToClient =>
      'Comment s\'est passée votre expérience avec ce client ?';

  @override
  String get reviewErrorWindowClosed =>
      'La fenêtre d\'évaluation est fermée (14 jours après la fin de la mission).';

  @override
  String get reviewErrorNotParticipant =>
      'Seuls les participants de cette mission peuvent laisser un avis.';

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
  String get drawerTeam => 'Équipe';

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
  String get reasonFraudOrScam => 'Fraude ou arnaque';

  @override
  String get reasonMisleadingDescription => 'Description trompeuse';

  @override
  String get reportJob => 'Signaler cette offre';

  @override
  String get reportApplication => 'Signaler cette candidature';

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

  @override
  String creditsRemaining(int count) {
    return '$count crédits restants';
  }

  @override
  String get noCreditsLeft => 'Vous n\'avez plus de crédits de candidature';

  @override
  String get creditsHowItWorks => 'Comment fonctionnent les crédits ?';

  @override
  String get creditsExplanation1 => 'Chaque candidature coûte 1 crédit';

  @override
  String get creditsExplanation2 =>
      'Chaque lundi, votre solde est remis à 10 crédits s\'il est inférieur à 10';

  @override
  String get creditsExplanation3 =>
      'Chaque mission signée vous rapporte 5 crédits bonus';

  @override
  String get creditsExplanation4 =>
      'Votre solde peut aller jusqu\'à 50 crédits maximum';

  @override
  String get noCreditsCannotApply =>
      'Vous avez besoin de crédits pour postuler à cette opportunité';

  @override
  String get paymentInfoSetup => 'Configurer les paiements';

  @override
  String get paymentInfoComplete => 'Compléter la vérification';

  @override
  String get paymentInfoEdit => 'Modifier les infos de paiement';

  @override
  String get paymentInfoActive => 'Compte entièrement actif';

  @override
  String get paymentInfoActiveDesc =>
      'Vous pouvez recevoir des paiements et transférer des fonds.';

  @override
  String get paymentInfoPending => 'Vérification en cours';

  @override
  String paymentInfoPendingDesc(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count informations à compléter',
      one: '$count information à compléter',
    );
    return '$_temp0';
  }

  @override
  String get paymentInfoNotConfigured => 'Non configuré';

  @override
  String get paymentInfoNotConfiguredDesc =>
      'Configurez votre compte de paiement pour recevoir des fonds.';

  @override
  String get paymentInfoCharges => 'Paiements';

  @override
  String get paymentInfoPayouts => 'Virements';

  @override
  String get kycBannerPendingTitle => 'Configurez vos paiements';

  @override
  String kycBannerPendingBody(int days) {
    return 'Vous avez des fonds en attente. Finalisez dans les $days jours pour éviter des restrictions.';
  }

  @override
  String get kycBannerRestrictedTitle => 'Compte restreint';

  @override
  String get kycBannerRestrictedBody =>
      'Vous ne pouvez plus postuler ni créer de propositions tant que votre compte de paiement n\'est pas configuré.';

  @override
  String get kycBannerCta => 'Configurer';

  @override
  String get disputeOpenTitle => 'Litige en cours';

  @override
  String get disputeNegotiationTitle => 'Negociation en cours';

  @override
  String get disputeEscalatedTitle => 'En mediation';

  @override
  String get disputeResolvedTitle => 'Litige resolu';

  @override
  String get disputeCounterPropose => 'Faire une proposition';

  @override
  String get disputeCancel => 'Annuler le litige';

  @override
  String get disputeOpenBtn => 'Signaler un probleme';

  @override
  String get disputeStatusOpen => 'Litige en cours';

  @override
  String get disputeStatusNegotiation => 'Negociation en cours';

  @override
  String get disputeStatusEscalated => 'En mediation';

  @override
  String get disputeStatusResolved => 'Litige resolu';

  @override
  String get disputeStatusCancelled => 'Litige annule';

  @override
  String get disputeReasonWorkNotConforming => 'Travail non conforme au scope';

  @override
  String get disputeReasonNonDelivery => 'Non-livraison';

  @override
  String get disputeReasonInsufficientQuality => 'Qualite insuffisante';

  @override
  String get disputeReasonClientGhosting => 'Client injoignable';

  @override
  String get disputeReasonScopeCreep => 'Demande hors du scope initial';

  @override
  String get disputeReasonRefusalToValidate =>
      'Refus de valider sans justification';

  @override
  String get disputeReasonHarassment => 'Harcelement';

  @override
  String get disputeReasonOther => 'Autre';

  @override
  String get disputeReasonLabel => 'Raison du signalement';

  @override
  String get disputeReasonPlaceholder => 'Selectionnez une raison';

  @override
  String get disputeAmountLabel => 'Que demandez-vous ?';

  @override
  String disputeTotalRefund(String amount) {
    return 'Remboursement total ($amount)';
  }

  @override
  String disputeTotalRelease(String amount) {
    return 'Liberation totale des fonds ($amount)';
  }

  @override
  String get disputePartialAmount => 'Montant partiel';

  @override
  String get disputeMessageToPartyLabel => 'Message a l\'autre partie';

  @override
  String get disputeMessageToPartyHint =>
      'Ce message sera visible dans la conversation. Expliquez clairement votre demande.';

  @override
  String get disputeMessageToPartyPlaceholder =>
      'Expliquez ce que vous attendez et pourquoi...';

  @override
  String get disputeDescriptionLabel =>
      'Description detaillee pour la mediation (optionnel)';

  @override
  String get disputeDescriptionHint =>
      'Ce texte ne sera lu que par l\'equipe de mediation si le litige est escalade.';

  @override
  String get disputeDescriptionPlaceholder =>
      'Contexte supplementaire, chronologie des evenements, description des preuves...';

  @override
  String get disputeFormWarning =>
      'L\'ouverture d\'un litige gele les fonds jusqu\'a resolution. L\'autre partie sera notifiee.';

  @override
  String get disputeSubmit => 'Soumettre le litige';

  @override
  String get disputeAccept => 'Accepter';

  @override
  String get disputeReject => 'Refuser';

  @override
  String get disputeCounterSubmit => 'Envoyer la proposition';

  @override
  String get disputeAddFiles => 'Ajouter des fichiers';

  @override
  String get disputeCancelBtn => 'Annuler';

  @override
  String get disputeViewDetails => 'Voir les details';

  @override
  String get disputeReportProblem => 'Signaler un probleme';

  @override
  String get disputeCounterSplitLabel => 'Repartition proposee';

  @override
  String get disputeCounterMessageLabel => 'Message (optionnel)';

  @override
  String get disputeCounterMessagePlaceholder =>
      'Expliquez votre proposition...';

  @override
  String get disputeRequestedAmount => 'demande';

  @override
  String disputeDaysLeft(int days) {
    return '$days jours avant escalade';
  }

  @override
  String get disputeEscalationSoon => 'Escalade imminente';

  @override
  String get disputeLastProposal => 'Derniere proposition';

  @override
  String disputeSplit(String client, String provider) {
    return '$client au client, $provider au prestataire';
  }

  @override
  String get disputeResolution => 'Resolution';

  @override
  String get disputeInProgress => 'Un litige est en cours sur cette mission';

  @override
  String get disputeClient => 'Client';

  @override
  String get disputeProvider => 'Prestataire';

  @override
  String get disputeOpenedLabel => 'Litige ouvert';

  @override
  String get disputeCounterProposalLabel => 'Proposition';

  @override
  String get disputeCounterAcceptedLabel => 'Proposition acceptee';

  @override
  String get disputeCounterRejectedLabel => 'Proposition refusee';

  @override
  String get disputeEscalatedLabel => 'Escalade en mediation';

  @override
  String get disputeResolvedLabel => 'Litige resolu';

  @override
  String get disputeCancelledLabel => 'Litige annule';

  @override
  String get disputeAutoResolvedLabel => 'Litige resolu automatiquement';

  @override
  String get disputeCancellationRequestedLabel => 'Demande d\'annulation';

  @override
  String get disputeCancellationRefusedLabel => 'Annulation refusee';

  @override
  String get disputeYourLastProposalRefused =>
      'Votre dernière proposition a été refusée';

  @override
  String get disputeEscalatedNegotiationStillOpen =>
      'Le litige est maintenant en médiation. Tant que l\'admin n\'a pas rendu sa décision, vous pouvez encore vous mettre d\'accord à l\'amiable.';

  @override
  String get disputeCancellationRequestPending =>
      'Demande d\'annulation en attente';

  @override
  String get disputeCancellationRequestWaiting =>
      'En attente de l\'accord de l\'autre partie pour annuler le litige.';

  @override
  String get disputeCancellationRequestConsent =>
      'L\'autre partie demande l\'annulation de ce litige. Votre accord est requis.';

  @override
  String get disputeCancellationRequestSent =>
      'Demande d\'annulation envoyee. En attente de la reponse de l\'autre partie.';

  @override
  String get disputeAcceptCancellation => 'Accepter l\'annulation';

  @override
  String get disputeRefuseCancellation => 'Refuser';

  @override
  String get disputeDecisionTitle => 'Décision de médiation';

  @override
  String disputeDecisionYourShare(int percent, String amount) {
    return 'Vous recevez $percent% — $amount';
  }

  @override
  String get disputeDecisionMessage => 'Message de l\'admin';

  @override
  String disputeDecisionRenderedOn(String date) {
    return 'Rendue le $date';
  }

  @override
  String get disputeCancelledTitle => 'Litige annulé';

  @override
  String get disputeCancelledSubtitle =>
      'Le litige a été annulé par accord mutuel.';

  @override
  String get projectStatusDisputed => 'En litige';

  @override
  String get permissionDenied =>
      'Vous n\'avez pas la permission d\'effectuer cette action';

  @override
  String get permissionDeniedSend =>
      'Vous n\'avez pas la permission d\'envoyer des messages';

  @override
  String get permissionDeniedWithdraw =>
      'Vous n\'avez pas la permission de demander un retrait';

  @override
  String get permissionDeniedEdit =>
      'Vous n\'avez pas la permission de modifier cette ressource';

  @override
  String get teamScreenTitle => 'Équipe';

  @override
  String get teamMembersSection => 'Membres';

  @override
  String get teamNoMembers => 'Aucun membre';

  @override
  String get teamNoOrganization => 'Aucune organisation';

  @override
  String get teamNoOrganizationDescription =>
      'Vous n\'êtes rattaché à aucune organisation pour le moment.';

  @override
  String get teamLoadError => 'Impossible de charger l\'équipe';

  @override
  String get teamRetry => 'Réessayer';

  @override
  String get teamInviteButton => 'Inviter';

  @override
  String get teamInviteDialogTitle => 'Inviter un nouveau membre';

  @override
  String get teamInviteDialogDescription =>
      'Envoyez un lien d\'invitation sécurisé. Le nouveau membre définira son mot de passe à sa première connexion.';

  @override
  String get teamInviteEmailLabel => 'Email';

  @override
  String get teamInviteEmailHint => 'collegue@example.com';

  @override
  String get teamInviteFirstNameLabel => 'Prénom';

  @override
  String get teamInviteLastNameLabel => 'Nom';

  @override
  String get teamInviteTitleLabel => 'Titre (optionnel)';

  @override
  String get teamInviteTitleHint => 'ex. Chef de projet';

  @override
  String get teamInviteRoleLabel => 'Rôle';

  @override
  String get teamInviteRoleHelp =>
      'Vous pourrez changer le rôle plus tard depuis la liste des membres.';

  @override
  String get teamInviteRoleAdmin => 'Admin';

  @override
  String get teamInviteRoleMember => 'Membre';

  @override
  String get teamInviteRoleViewer => 'Observateur';

  @override
  String get teamInviteSendButton => 'Envoyer l\'invitation';

  @override
  String get teamInviteCancelButton => 'Annuler';

  @override
  String teamInviteSuccess(String email) {
    return 'Invitation envoyée à $email';
  }

  @override
  String get teamInviteEmailRequired => 'L\'email est requis';

  @override
  String get teamInviteEmailInvalid => 'Veuillez saisir un email valide';

  @override
  String get teamInviteFirstNameRequired => 'Le prénom est requis';

  @override
  String get teamInviteLastNameRequired => 'Le nom est requis';

  @override
  String get teamInviteFailed =>
      'Impossible d\'envoyer l\'invitation. Veuillez réessayer.';

  @override
  String get teamRolePermissionsTitle => 'Rôles et permissions';

  @override
  String get teamRolePermissionsSubtitle =>
      'Ce que chaque rôle peut faire dans cette organisation.';

  @override
  String get teamRolePermissionsReadOnlyTitle => 'Vue en lecture seule';

  @override
  String get teamRolePermissionsReadOnlyDescription =>
      'Seul le propriétaire peut modifier les permissions. Les autres membres voient la matrice à titre informatif.';

  @override
  String get teamRolePermissionsLoadError =>
      'Impossible de charger les permissions';

  @override
  String get teamRolePermissionsModifiedBadge => 'Modifié';

  @override
  String teamRolePermissionsPending(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count modifications en attente',
      one: '1 modification en attente',
    );
    return '$_temp0';
  }

  @override
  String get teamRolePermissionsDiscard => 'Annuler';

  @override
  String get teamRolePermissionsSave => 'Enregistrer';

  @override
  String get teamRolePermissionsConfirmTitle => 'Confirmer les modifications';

  @override
  String teamRolePermissionsConfirmDescription(int count, String role) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'Cette action va mettre à jour $count permissions pour le rôle $role. Les membres concernés seront déconnectés et devront se reconnecter.',
      one:
          'Cette action va mettre à jour 1 permission pour le rôle $role. Les membres concernés seront déconnectés et devront se reconnecter.',
    );
    return '$_temp0';
  }

  @override
  String get teamRolePermissionsConfirmButton => 'Enregistrer';

  @override
  String get teamRolePermissionsCancelButton => 'Annuler';

  @override
  String teamRolePermissionsSaveSuccess(int affected) {
    return 'Permissions mises à jour. $affected session(s) invalidée(s).';
  }

  @override
  String get teamRolePermissionsSaveFailed =>
      'Impossible d\'enregistrer les permissions. Veuillez réessayer.';

  @override
  String get teamRolePermissionsOwnerExclusiveTitle =>
      'Permissions exclusives au propriétaire';

  @override
  String get teamRolePermissionsOwnerExclusiveDescription =>
      'Ces permissions ne peuvent pas être personnalisées et sont réservées au propriétaire de l\'organisation.';

  @override
  String get teamRolePermissionsStateGrantedOverride => 'Accordée';

  @override
  String get teamRolePermissionsStateRevokedOverride => 'Révoquée';

  @override
  String get teamRolePermissionsStateLocked => 'Verrouillée';

  @override
  String get teamRolePermissionRoleAdmin => 'Admin';

  @override
  String get teamRolePermissionRoleMember => 'Membre';

  @override
  String get teamRolePermissionRoleViewer => 'Observateur';

  @override
  String get teamRolePermissionRoleOwner => 'Propriétaire';

  @override
  String get teamRolePermissionGroupTeam => 'Équipe';

  @override
  String get teamRolePermissionGroupOrgProfile => 'Profil public';

  @override
  String get teamRolePermissionGroupJobs => 'Missions';

  @override
  String get teamRolePermissionGroupProposals => 'Propositions';

  @override
  String get teamRolePermissionGroupMessaging => 'Messagerie';

  @override
  String get teamRolePermissionGroupReviews => 'Avis';

  @override
  String get teamRolePermissionGroupWallet => 'Portefeuille';

  @override
  String get teamRolePermissionGroupBilling => 'Facturation';

  @override
  String get teamRolePermissionGroupKyc => 'KYC';

  @override
  String get teamRolePermissionGroupDanger => 'Zone sensible';

  @override
  String get teamMemberActions => 'Actions';

  @override
  String get teamMemberEdit => 'Modifier';

  @override
  String get teamMemberRemove => 'Retirer';

  @override
  String get teamMemberFallbackName => 'Membre';

  @override
  String get teamMemberCannotEditSelf =>
      'Vous ne pouvez pas modifier votre propre adhésion.';

  @override
  String get teamMemberCannotRemoveSelf =>
      'Utilisez plutôt « Quitter l\'organisation ».';

  @override
  String teamEditMemberDialogTitle(String name) {
    return 'Modifier $name';
  }

  @override
  String get teamEditMemberRoleLabel => 'Rôle';

  @override
  String get teamEditMemberTitleLabel => 'Titre';

  @override
  String get teamEditMemberTitleHint => 'ex. Chef de projet';

  @override
  String get teamEditMemberSave => 'Enregistrer';

  @override
  String get teamEditMemberSuccess => 'Membre mis à jour';

  @override
  String get teamEditMemberFailed =>
      'Impossible de mettre à jour le membre. Veuillez réessayer.';

  @override
  String get teamEditMemberNoChanges => 'Aucune modification à enregistrer.';

  @override
  String get teamRemoveMemberDialogTitle => 'Retirer le membre';

  @override
  String teamRemoveMemberConfirm(String name) {
    return 'Voulez-vous retirer $name de l\'organisation ? L\'accès sera révoqué immédiatement.';
  }

  @override
  String get teamRemoveMemberConfirmButton => 'Retirer';

  @override
  String teamRemoveMemberSuccess(String name) {
    return '$name a été retiré';
  }

  @override
  String get teamRemoveMemberFailed =>
      'Impossible de retirer le membre. Veuillez réessayer.';

  @override
  String get teamInvitationsSection => 'Invitations en attente';

  @override
  String teamInvitationsCountLabel(int count) {
    return 'Invitations en attente ($count)';
  }

  @override
  String get teamInvitationsEmpty => 'Aucune invitation en attente.';

  @override
  String get teamInvitationsLoadFailed =>
      'Impossible de charger les invitations.';

  @override
  String teamInvitationSentAgo(int days) {
    return 'Envoyée il y a $days jour(s)';
  }

  @override
  String get teamInvitationSentToday => 'Envoyée aujourd\'hui';

  @override
  String teamInvitationExpiresIn(int days) {
    return 'Expire dans $days jour(s)';
  }

  @override
  String get teamInvitationExpired => 'Expirée';

  @override
  String get teamInvitationCancelTooltip => 'Annuler l\'invitation';

  @override
  String get teamInvitationResendTooltip => 'Renvoyer l\'invitation';

  @override
  String get teamInvitationCancelDialogTitle => 'Annuler l\'invitation';

  @override
  String teamInvitationCancelDialogBody(String email) {
    return 'Annuler l\'invitation envoyée à $email ? Cette personne ne pourra plus rejoindre l\'organisation avec ce lien.';
  }

  @override
  String get teamInvitationCancelConfirm => 'Annuler l\'invitation';

  @override
  String get teamInvitationCancelKeep => 'Conserver';

  @override
  String get teamInvitationCancelSuccess => 'Invitation annulée';

  @override
  String get teamInvitationCancelFailed =>
      'Impossible d\'annuler l\'invitation. Veuillez réessayer.';

  @override
  String get teamInvitationResendSuccess => 'Invitation renvoyée';

  @override
  String get teamInvitationResendFailed =>
      'Impossible de renvoyer l\'invitation. Veuillez réessayer.';

  @override
  String get teamLeaveAction => 'Quitter l\'organisation';

  @override
  String get teamLeaveDialogTitle => 'Quitter l\'organisation';

  @override
  String get teamLeaveDialogBody =>
      'Vous perdrez l\'accès aux données et aux conversations de cette organisation. Cette action est irréversible.';

  @override
  String get teamLeaveConfirmHint => 'Tapez QUITTER pour confirmer';

  @override
  String get teamLeaveConfirmKeyword => 'QUITTER';

  @override
  String get teamLeaveConfirmButton => 'Quitter l\'organisation';

  @override
  String get teamLeaveSuccess => 'Vous avez quitté l\'organisation';

  @override
  String get teamLeaveFailed =>
      'Impossible de quitter l\'organisation. Veuillez réessayer.';

  @override
  String get teamTransferAction => 'Transférer la propriété';

  @override
  String get teamTransferDialogTitle => 'Transférer la propriété';

  @override
  String get teamTransferDialogBody =>
      'Choisissez un Admin qui deviendra le nouveau propriétaire de cette organisation. Vous deviendrez Admin une fois qu\'il aura accepté. Cette action est irréversible.';

  @override
  String get teamTransferTargetLabel => 'Nouveau propriétaire';

  @override
  String get teamTransferNoEligible =>
      'Aucun Admin disponible. Promouvez d\'abord un membre au rôle d\'Admin.';

  @override
  String get teamTransferConfirmButton => 'Envoyer la demande';

  @override
  String get teamTransferSuccess => 'Demande de transfert envoyée';

  @override
  String get teamTransferFailed =>
      'Impossible d\'initier le transfert. Veuillez réessayer.';

  @override
  String get teamPendingTransferTargetTitle =>
      'Une propriété vous est proposée';

  @override
  String get teamPendingTransferTargetBody =>
      'Acceptez pour devenir le nouveau propriétaire de cette organisation. Refusez pour conserver votre rôle actuel.';

  @override
  String get teamPendingTransferInitiatorTitle =>
      'Transfert de propriété en attente';

  @override
  String get teamPendingTransferInitiatorBody =>
      'En attente d\'acceptation par l\'Admin destinataire.';

  @override
  String get teamPendingTransferReadOnlyTitle =>
      'Transfert de propriété en cours';

  @override
  String get teamPendingTransferReadOnlyBody =>
      'Un transfert de propriété est en cours pour cette organisation.';

  @override
  String teamPendingTransferExpiresOn(String date) {
    return 'Expire le $date';
  }

  @override
  String get teamPendingTransferAccept => 'Accepter';

  @override
  String get teamPendingTransferDecline => 'Refuser';

  @override
  String get teamPendingTransferCancel => 'Annuler le transfert';

  @override
  String get teamPendingTransferAcceptSuccess =>
      'Vous êtes désormais propriétaire de cette organisation';

  @override
  String get teamPendingTransferAcceptFailed =>
      'Impossible d\'accepter le transfert. Veuillez réessayer.';

  @override
  String get teamPendingTransferDeclineDialogTitle => 'Refuser le transfert';

  @override
  String get teamPendingTransferDeclineDialogBody =>
      'Refuser le transfert de propriété ? Le propriétaire actuel conservera son rôle.';

  @override
  String get teamPendingTransferDeclineSuccess => 'Transfert refusé';

  @override
  String get teamPendingTransferDeclineFailed =>
      'Impossible de refuser le transfert. Veuillez réessayer.';

  @override
  String get teamPendingTransferCancelDialogTitle => 'Annuler le transfert';

  @override
  String get teamPendingTransferCancelDialogBody =>
      'Annuler le transfert de propriété en attente ? Vous resterez propriétaire.';

  @override
  String get teamPendingTransferCancelSuccess => 'Transfert annulé';

  @override
  String get teamPendingTransferCancelFailed =>
      'Impossible d\'annuler le transfert. Veuillez réessayer.';

  @override
  String get teamRoleOwner => 'Propriétaire';

  @override
  String get teamRoleAdmin => 'Admin';

  @override
  String get teamRoleMember => 'Membre';

  @override
  String get teamRoleViewer => 'Observateur';

  @override
  String get expertiseSectionTitle => 'Domaines d\'expertise';

  @override
  String expertiseSectionSubtitle(int max) {
    return 'Choisissez jusqu\'à $max domaines qui mettent en valeur votre savoir-faire';
  }

  @override
  String get expertiseAddDomains => 'Ajouter des domaines';

  @override
  String get expertiseSave => 'Enregistrer';

  @override
  String get expertiseSaving => 'Enregistrement...';

  @override
  String expertiseMaxReached(int max) {
    return 'Vous avez atteint le maximum de $max domaines';
  }

  @override
  String expertiseCounter(int count, int max) {
    return '$count/$max sélectionnés';
  }

  @override
  String get expertiseEmptyPrivate =>
      'Aucune expertise sélectionnée pour le moment.';

  @override
  String get expertiseErrorGeneric =>
      'Impossible d\'enregistrer vos expertises. Veuillez réessayer.';

  @override
  String get expertiseDomainDevelopment => 'Développement';

  @override
  String get expertiseDomainDataAiMl => 'Data, IA & Machine Learning';

  @override
  String get expertiseDomainDesignUiUx => 'Design & UI/UX';

  @override
  String get expertiseDomainDesign3dAnimation => 'Design 3D & Animation';

  @override
  String get expertiseDomainVideoMotion => 'Vidéo & Motion';

  @override
  String get expertiseDomainPhotoAudiovisual => 'Photo & Audiovisuel';

  @override
  String get expertiseDomainMarketingGrowth => 'Marketing & Growth';

  @override
  String get expertiseDomainWritingTranslation => 'Rédaction & Traduction';

  @override
  String get expertiseDomainBusinessDevSales => 'Business Development & Ventes';

  @override
  String get expertiseDomainConsultingStrategy => 'Consulting & Stratégie';

  @override
  String get expertiseDomainProductUxResearch => 'Product & UX Research';

  @override
  String get expertiseDomainOpsAdminSupport => 'Ops, Admin & Support';

  @override
  String get expertiseDomainLegal => 'Legal & Droit';

  @override
  String get expertiseDomainFinanceAccounting => 'Finance & Comptabilité';

  @override
  String get expertiseDomainHrRecruitment => 'RH & Recrutement';

  @override
  String get skillsDisplaySectionTitle => 'Compétences';

  @override
  String skillsDisplayMoreSuffix(int count) {
    return '+$count';
  }

  @override
  String get skillsSectionTitle => 'Compétences';

  @override
  String skillsSectionSubtitle(int max) {
    return 'Jusqu\'à $max compétences';
  }

  @override
  String get skillsEmpty => 'Aucune compétence ajoutée';

  @override
  String get skillsEditButton => 'Modifier mes compétences';

  @override
  String get skillsModalTitle => 'Mes compétences';

  @override
  String get skillsSearchPlaceholder => 'Chercher une compétence...';

  @override
  String skillsCounter(int count, int max) {
    return '$count / $max';
  }

  @override
  String get skillsBrowseHeading => 'Parcourir par domaine';

  @override
  String get skillsSave => 'Enregistrer';

  @override
  String get skillsSaving => 'Enregistrement...';

  @override
  String get skillsCancel => 'Annuler';

  @override
  String skillsCreateNew(String query) {
    return 'Créer « $query »';
  }

  @override
  String skillsUsageCount(int count) {
    return '$count pros';
  }

  @override
  String skillsErrorTooMany(int max) {
    return 'Tu as dépassé la limite de $max compétences';
  }

  @override
  String get skillsErrorDisabled => 'Indisponible pour ce type de compte';

  @override
  String get skillsErrorGeneric => 'Une erreur est survenue';

  @override
  String get tier1AvailabilitySectionTitle => 'Disponibilité';

  @override
  String get tier1AvailabilityStatusAvailableNow => 'Disponible maintenant';

  @override
  String get tier1AvailabilityStatusAvailableSoon => 'Disponible bientôt';

  @override
  String get tier1AvailabilityStatusNotAvailable => 'Indisponible';

  @override
  String get tier1AvailabilityReferrerTitle =>
      'Disponibilité en tant qu\'apporteur d\'affaires';

  @override
  String get tier1AvailabilityDirectLabel => 'Prestation';

  @override
  String get tier1AvailabilityReferrerLabel => 'Apport d\'affaires';

  @override
  String get tier1AvailabilityEditButton => 'Mettre à jour ma disponibilité';

  @override
  String get tier1LocationSectionTitle => 'Localisation';

  @override
  String get tier1LocationCityLabel => 'Ville';

  @override
  String get tier1LocationCityPlaceholder => 'Paris';

  @override
  String get tier1LocationCountryLabel => 'Pays';

  @override
  String get tier1LocationCountryPlaceholder => 'Sélectionner un pays';

  @override
  String get tier1LocationWorkModeLabel => 'Mode de travail';

  @override
  String get tier1LocationWorkModeRemote => 'À distance';

  @override
  String get tier1LocationWorkModeOnSite => 'Sur site';

  @override
  String get tier1LocationWorkModeHybrid => 'Hybride';

  @override
  String get tier1LocationTravelRadiusLabel => 'Rayon de déplacement (km)';

  @override
  String get tier1LocationTravelRadiusPlaceholder => 'ex. 50';

  @override
  String get tier1LocationEmpty =>
      'Ajoute ta ville pour être plus facile à trouver';

  @override
  String get tier1LocationEditButton => 'Mettre à jour ma localisation';

  @override
  String get tier1LanguagesSectionTitle => 'Langues';

  @override
  String get tier1LanguagesProfessionalLabel => 'Langues professionnelles';

  @override
  String get tier1LanguagesConversationalLabel => 'Langues conversationnelles';

  @override
  String get tier1LanguagesSearchPlaceholder => 'Rechercher une langue...';

  @override
  String get tier1LanguagesEmpty =>
      'Déclare les langues dans lesquelles tu peux travailler';

  @override
  String get tier1LanguagesEditButton => 'Mettre à jour mes langues';

  @override
  String tier1LanguagesCountLabel(int count) {
    return '$count sélectionnée(s)';
  }

  @override
  String get tier1PricingSectionTitle => 'Tarifs';

  @override
  String get tier1PricingDirectSectionTitle => 'Tarifs';

  @override
  String get tier1PricingReferralSectionTitle => 'Tarifs d\'apport d\'affaires';

  @override
  String get tier1PricingEmpty => 'Aucun tarif déclaré';

  @override
  String get tier1PricingEditButton => 'Modifier mes tarifs';

  @override
  String get tier1PricingModalTitle => 'Mes tarifs';

  @override
  String get tier1PricingDirectModalTitle => 'Modifier mes tarifs';

  @override
  String get tier1PricingReferralModalTitle => 'Modifier mes tarifs d\'apport';

  @override
  String get tier1PricingKindDirect => 'Prestation directe';

  @override
  String get tier1PricingKindReferral => 'Apport d\'affaires';

  @override
  String get tier1PricingNegotiableLabel => 'Est-ce négociable ?';

  @override
  String get tier1PricingNegotiableYes => 'Oui';

  @override
  String get tier1PricingNegotiableNo => 'Non';

  @override
  String get tier1PricingNegotiableBadge => 'négociable';

  @override
  String get tier1PricingTypeDaily => 'TJM (taux journalier)';

  @override
  String get tier1PricingTypeHourly => 'Taux horaire';

  @override
  String get tier1PricingTypeProjectFrom => 'À partir de (par projet)';

  @override
  String get tier1PricingTypeProjectRange => 'Fourchette par projet';

  @override
  String get tier1PricingTypeCommissionPct => 'Commission en pourcentage';

  @override
  String get tier1PricingTypeCommissionFlat => 'Commission forfaitaire';

  @override
  String get tier1PricingMinLabel => 'Montant min';

  @override
  String get tier1PricingMaxLabel => 'Montant max';

  @override
  String get tier1PricingCurrencyLabel => 'Devise';

  @override
  String get tier1PricingNoteLabel => 'Note';

  @override
  String get tier1PricingNotePlaceholder => 'Négociable selon scope...';

  @override
  String get tier1PricingPreviewHeading => 'Aperçu sur ta card';

  @override
  String get tier1PricingEmptyPreview => '–';

  @override
  String get tier1PricingDeleteKind => 'Supprimer cette ligne';

  @override
  String get tier1PricingEnableReferralRow =>
      'Ajouter une ligne apport d\'affaires';

  @override
  String get tier1Save => 'Enregistrer';

  @override
  String get tier1Saving => 'Enregistrement...';

  @override
  String get tier1Cancel => 'Annuler';

  @override
  String get tier1Delete => 'Supprimer';

  @override
  String get tier1Close => 'Fermer';

  @override
  String get tier1ErrorGeneric => 'Une erreur est survenue';

  @override
  String get tier1ErrorPricingInvalidAmount => 'Saisis un montant valide';

  @override
  String get tier1ErrorLocationRequireCity => 'La ville est obligatoire';
}
