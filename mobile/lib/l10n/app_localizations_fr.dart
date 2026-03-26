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
}
