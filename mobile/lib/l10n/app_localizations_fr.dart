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
  String get callUnknownCaller => 'Appelant inconnu';
}
