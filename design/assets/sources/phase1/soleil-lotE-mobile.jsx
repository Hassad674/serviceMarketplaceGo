// Lot E mobile — Login, Signup choix rôle, Compte
// Largeur cible : 390px. Bottom nav uniquement quand l'utilisateur est connecté.

const SEM = window.S;
const SEMI = window.SI;
const SEMPortrait = window.Portrait;
const { MobileFrame, MobileHeader, MobileBottomNav, MobileSegmented, MobileListItem } = window;

// Logo compact pour mobile
function AtelierMarkM({ size = 24 }) {
  return (
    <div style={{ display: 'inline-flex', alignItems: 'center', gap: 7 }}>
      <div style={{ width: size, height: size, borderRadius: 6, background: SEM.accent, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontFamily: SEM.serif, fontSize: size * 0.55, fontWeight: 600, fontStyle: 'italic' }}>a</div>
      <span style={{ fontFamily: SEM.serif, fontSize: size * 0.7, fontWeight: 500, letterSpacing: '-0.015em' }}>Atelier</span>
    </div>
  );
}

// ═══ EM1 — Login mobile ═════════════════════════════════════════
function SoleilLoginMobile() {
  return (
    <MobileFrame url="atelier.fr/login">
      <div style={{ flex: 1, overflow: 'auto', padding: '32px 24px 24px', display: 'flex', flexDirection: 'column' }}>
        <AtelierMarkM size={26} />
        <div style={{ marginTop: 36, marginBottom: 28 }}>
          <div style={{ fontSize: 10.5, color: SEM.accent, marginBottom: 8, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase' }}>Bon retour</div>
          <h1 style={{ fontFamily: SEM.serif, fontSize: 32, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1 }}>Reprends là <span style={{ fontStyle: 'italic', color: SEM.accent }}>où tu en étais.</span></h1>
          <p style={{ fontSize: 13.5, color: SEM.textMute, margin: '10px 0 0', lineHeight: 1.55 }}>Connecte-toi pour suivre tes missions et tes candidatures.</p>
        </div>

        <label style={{ fontSize: 12, fontWeight: 600, marginBottom: 6, display: 'block' }}>Adresse e-mail</label>
        <input type="email" defaultValue="elise.morel@hey.com" style={{ width: '100%', border: `1.5px solid ${SEM.accent}`, borderRadius: 10, padding: '12px 14px', fontSize: 14, fontFamily: SEM.sans, outline: 'none', marginBottom: 14, background: '#fff' }} />

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 6 }}>
          <label style={{ fontSize: 12, fontWeight: 600 }}>Mot de passe</label>
          <a style={{ fontSize: 11, color: SEM.accent, fontWeight: 600, fontStyle: 'italic', fontFamily: SEM.serif }}>oublié ?</a>
        </div>
        <div style={{ position: 'relative', marginBottom: 18 }}>
          <input type="password" defaultValue="••••••••••" style={{ width: '100%', border: `1px solid ${SEM.borderStrong}`, borderRadius: 10, padding: '12px 40px 12px 14px', fontSize: 14, fontFamily: SEM.sans, outline: 'none', background: '#fff' }} />
          <div style={{ position: 'absolute', right: 12, top: '50%', transform: 'translateY(-50%)', color: SEM.textMute }}><SEMI name="Eye" size={16} /></div>
        </div>

        <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12.5, color: SEM.textMute, marginBottom: 18 }}>
          <input type="checkbox" defaultChecked /> Rester connectée sur ce navigateur
        </label>

        <button style={{ background: SEM.text, color: '#fff', border: 'none', padding: '14px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Se connecter</button>

        <div style={{ display: 'flex', alignItems: 'center', gap: 10, margin: '20px 0', fontSize: 11, color: SEM.textSubtle, fontStyle: 'italic', fontFamily: SEM.serif }}>
          <div style={{ flex: 1, height: 1, background: SEM.border }} /> ou <div style={{ flex: 1, height: 1, background: SEM.border }} />
        </div>

        <button style={{ background: '#fff', border: `1px solid ${SEM.borderStrong}`, padding: '12px', fontSize: 13.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, marginBottom: 10 }}>
          <span style={{ fontSize: 14 }}>G</span> Continuer avec Google
        </button>
        <button style={{ background: '#fff', border: `1px solid ${SEM.borderStrong}`, padding: '12px', fontSize: 13.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}>
          LinkedIn
        </button>

        <div style={{ marginTop: 'auto', paddingTop: 28, textAlign: 'center', fontSize: 12.5, color: SEM.textMute }}>
          Première fois ici ? <a style={{ color: SEM.accent, fontWeight: 600 }}>Créer un compte</a>
        </div>
      </div>
    </MobileFrame>
  );
}

// ═══ EM2 — Signup choix rôle ═════════════════════════════════════
function SoleilSignupMobile() {
  return (
    <MobileFrame url="atelier.fr/signup">
      <div style={{ flex: 1, overflow: 'auto', padding: '24px 20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 28 }}>
          <AtelierMarkM size={24} />
          <div style={{ fontSize: 11.5, color: SEM.textMute }}>Déjà inscrit ? <a style={{ color: SEM.accent, fontWeight: 600 }}>Connexion</a></div>
        </div>
        <div style={{ fontSize: 10.5, color: SEM.accent, marginBottom: 8, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase' }}>Bienvenue chez Atelier</div>
        <h1 style={{ fontFamily: SEM.serif, fontSize: 30, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1, marginBottom: 8 }}>Tu rejoins <span style={{ fontStyle: 'italic', color: SEM.accent }}>en tant que…</span></h1>
        <p style={{ fontSize: 13, color: SEM.textMute, margin: '0 0 22px', lineHeight: 1.55 }}>Tu pourras toujours changer plus tard depuis ton compte.</p>

        {[
          { id: 'fr', icon: '✦', title: 'Freelance', desc: 'Je propose mes compétences sur des missions.', tag: null, sel: true },
          { id: 'co', icon: '◆', title: 'Entreprise', desc: 'Je cherche des freelances pour des projets.', tag: null, sel: false },
          { id: 'ap', icon: '◌', title: 'Apporteur d\'affaires', desc: 'Je recommande des freelances et touche une commission.', tag: 'Nouveau', sel: false },
        ].map(opt => (
          <div key={opt.id} style={{ background: '#fff', border: opt.sel ? `2px solid ${SEM.accent}` : `1px solid ${SEM.border}`, borderRadius: 14, padding: 16, marginBottom: 10, cursor: 'pointer', position: 'relative' }}>
            {opt.tag ? <div style={{ position: 'absolute', top: 12, right: 12, fontSize: 10, padding: '3px 8px', background: SEM.accentSoft, color: SEM.accentDeep, borderRadius: 999, fontWeight: 700 }}>{opt.tag}</div> : null}
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
              <div style={{ width: 40, height: 40, borderRadius: 10, background: opt.sel ? SEM.accentSoft : SEM.bg, color: opt.sel ? SEM.accentDeep : SEM.textMute, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 18, fontFamily: SEM.serif, flexShrink: 0 }}>{opt.icon}</div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 15, fontWeight: 600, fontFamily: SEM.serif, marginBottom: 3 }}>{opt.title}</div>
                <div style={{ fontSize: 12.5, color: SEM.textMute, lineHeight: 1.5 }}>{opt.desc}</div>
              </div>
              <div style={{ width: 18, height: 18, borderRadius: '50%', border: `2px solid ${opt.sel ? SEM.accent : SEM.borderStrong}`, background: opt.sel ? SEM.accent : 'transparent', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, marginTop: 4 }}>
                {opt.sel ? <div style={{ width: 6, height: 6, borderRadius: '50%', background: '#fff' }} /> : null}
              </div>
            </div>
          </div>
        ))}

        <button style={{ background: SEM.text, color: '#fff', border: 'none', padding: '14px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', width: '100%', marginTop: 18 }}>Continuer en tant que freelance →</button>

        <div style={{ marginTop: 18, padding: '12px 14px', background: SEM.greenSoft, borderRadius: 10, fontSize: 12, color: SEM.green, display: 'flex', alignItems: 'center', gap: 8 }}>
          <SEMI name="Shield" size={14} /> Profil vérifié, paiement sécurisé. Pas de commission cachée.
        </div>
      </div>
    </MobileFrame>
  );
}

// ═══ EM3 — Compte ═════════════════════════════════════════════
// Liste de réglages au lieu du tableau Push/Email desktop. Le tableau ne tient pas en 390 ; on bascule en accordéons.
function SoleilCompteMobile() {
  const [tab, setTab] = React.useState(0);
  return (
    <MobileFrame url="atelier.fr/compte">
      <MobileHeader title="Mon compte" subtitle="Réglages, sécurité, données" />
      <div style={{ padding: '12px 14px 0', flexShrink: 0, background: '#fff' }}>
        <MobileSegmented items={['Notifs', 'Email', 'Sécurité', 'Données']} active={tab} />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '12px 0 80px' }}>
        {tab === 0 ? (
          <>
            <div style={{ padding: '14px 16px', background: '#fff', display: 'flex', alignItems: 'center', gap: 12, borderBottom: `1px solid ${SEM.border}` }}>
              <SEMI name="Bell" size={18} />
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 13.5, fontWeight: 600 }}>Notifications activées</div>
                <div style={{ fontSize: 11.5, color: SEM.textMute }}>Désactiver coupe tout d'un coup.</div>
              </div>
              <div style={{ width: 44, height: 26, background: SEM.green, borderRadius: 999, position: 'relative', cursor: 'pointer', flexShrink: 0 }}>
                <div style={{ position: 'absolute', top: 2, left: 20, width: 22, height: 22, background: '#fff', borderRadius: '50%', boxShadow: '0 2px 4px rgba(0,0,0,0.2)' }} />
              </div>
            </div>

            {[
              { title: 'Propositions & Projets', desc: 'Nouvelles invitations, propositions reçues, projets validés' },
              { title: 'Avis', desc: 'Quand tu reçois un nouvel avis sur une mission terminée' },
              { title: 'Messages', desc: 'Nouveaux messages dans tes conversations' },
              { title: 'Opportunités & Candidatures', desc: 'Réponses à tes candidatures, opportunités correspondantes' },
            ].map((s, i) => (
              <div key={i} style={{ background: '#fff', borderBottom: `1px solid ${SEM.border}`, padding: '14px 16px' }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                  <div style={{ fontSize: 14, fontWeight: 600 }}>{s.title}</div>
                  <SEMI name="ChevronDown" size={14} />
                </div>
                <div style={{ fontSize: 11.5, color: SEM.textMute, marginBottom: 12 }}>{s.desc}</div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <div style={{ flex: 1, padding: '8px 12px', background: SEM.bg, borderRadius: 10, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <span style={{ fontSize: 12, fontWeight: 600 }}>Push</span>
                    <div style={{ width: 32, height: 18, background: SEM.green, borderRadius: 999, position: 'relative' }}>
                      <div style={{ position: 'absolute', top: 2, left: 16, width: 14, height: 14, background: '#fff', borderRadius: '50%' }} />
                    </div>
                  </div>
                  <div style={{ flex: 1, padding: '8px 12px', background: SEM.bg, borderRadius: 10, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <span style={{ fontSize: 12, fontWeight: 600 }}>Email</span>
                    <div style={{ width: 32, height: 18, background: i === 1 ? SEM.borderStrong : SEM.green, borderRadius: 999, position: 'relative' }}>
                      <div style={{ position: 'absolute', top: 2, left: i === 1 ? 2 : 16, width: 14, height: 14, background: '#fff', borderRadius: '50%' }} />
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </>
        ) : tab === 1 ? (
          <>
            <div style={{ padding: '14px 16px', background: '#fff', borderBottom: `1px solid ${SEM.border}` }}>
              <div style={{ fontSize: 11, color: SEM.textMute, marginBottom: 4, fontWeight: 600, letterSpacing: '0.05em', textTransform: 'uppercase' }}>Email principal</div>
              <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 4 }}>elise.morel@hey.com</div>
              <div style={{ fontSize: 11, color: SEM.green, fontWeight: 600 }}>✓ Vérifié</div>
            </div>
            <MobileListItem title="Changer d'adresse e-mail" subtitle="Un lien sera envoyé à la nouvelle adresse" trailing={<SEMI name="ChevronRight" size={14} />} />
            <MobileListItem title="Fréquence des digests" subtitle="Quotidien · 9 h" trailing={<SEMI name="ChevronRight" size={14} />} />
          </>
        ) : tab === 2 ? (
          <>
            <MobileListItem leading={<div style={{ width: 36, height: 36, borderRadius: 10, background: SEM.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SEMI name="Lock" size={16} /></div>} title="Mot de passe" subtitle="Modifié il y a 3 mois" trailing={<SEMI name="ChevronRight" size={14} />} />
            <MobileListItem leading={<div style={{ width: 36, height: 36, borderRadius: 10, background: SEM.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SEMI name="Shield" size={16} /></div>} title="Authentification 2 facteurs" subtitle="Désactivée — recommandée" trailing={<span style={{ fontSize: 11, padding: '3px 8px', background: SEM.accentSoft, color: SEM.accentDeep, borderRadius: 999, fontWeight: 700 }}>Activer</span>} />
            <MobileListItem leading={<div style={{ width: 36, height: 36, borderRadius: 10, background: SEM.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SEMI name="Phone" size={16} /></div>} title="Sessions actives" subtitle="2 appareils connectés" trailing={<SEMI name="ChevronRight" size={14} />} />
          </>
        ) : (
          <>
            <MobileListItem title="Télécharger mes données" subtitle="Export ZIP au format JSON · sous 24h" trailing={<SEMI name="ChevronRight" size={14} />} />
            <MobileListItem title="Désactiver mon compte" subtitle="Réversible pendant 30 jours" trailing={<SEMI name="ChevronRight" size={14} />} />
            <div style={{ padding: '20px 16px' }}>
              <button style={{ width: '100%', background: 'transparent', border: `1px solid ${SEM.accentDeep}`, color: SEM.accentDeep, padding: '12px', fontSize: 13.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Supprimer définitivement mon compte</button>
              <div style={{ fontSize: 11, color: SEM.textSubtle, marginTop: 10, textAlign: 'center', fontStyle: 'italic', fontFamily: SEM.serif }}>Cette action est définitive. Tes missions terminées restent accessibles à leurs clients.</div>
            </div>
          </>
        )}
      </div>
      <MobileBottomNav active="profile" role="freelancer" />
    </MobileFrame>
  );
}

window.SoleilLoginMobile = SoleilLoginMobile;
window.SoleilSignupMobile = SoleilSignupMobile;
window.SoleilCompteMobile = SoleilCompteMobile;
