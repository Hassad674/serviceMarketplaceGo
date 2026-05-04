// Lot E — Onboarding, auth & compte
// 5 écrans : Login · Signup (rôle) · Signup formulaire freelance · Signup formulaire entreprise · Stripe Connect wrapper · Compte
// Note : Pas de SSidebar/STopbar pour les écrans d'auth/onboarding — c'est plein écran.

const SE = window.S;
const SEI = window.SI;
const SESidebar = window.SSidebar;
const SETopbar = window.STopbar;
const SEPortrait = window.Portrait;

// ─── Mark : logo "Atelier" texte ────────────────────────────────
function AtelierMark({ size = 28, color }) {
  return (
    <div style={{ display: 'inline-flex', alignItems: 'center', gap: 9 }}>
      <div style={{ width: size, height: size, borderRadius: 8, background: SE.accent, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontFamily: SE.serif, fontSize: size * 0.55, fontWeight: 600, fontStyle: 'italic' }}>a</div>
      <span style={{ fontFamily: SE.serif, fontSize: size * 0.65, fontWeight: 500, letterSpacing: '-0.015em', color: color || SE.text }}>Atelier</span>
    </div>
  );
}

// Visuel décoratif : trois portraits flottants sur fond chaud — réutilisé dans plusieurs écrans
function PortraitTrio({ scale = 1 }) {
  return (
    <div style={{ position: 'relative', width: 280 * scale, height: 200 * scale }}>
      <div style={{ position: 'absolute', top: 30 * scale, left: 0, transform: 'rotate(-7deg)', boxShadow: '0 12px 28px rgba(0,0,0,0.12)', borderRadius: '50%' }}>
        <SEPortrait id={1} size={92 * scale} />
      </div>
      <div style={{ position: 'absolute', top: 0, left: '50%', transform: 'translateX(-50%)', boxShadow: '0 18px 36px rgba(0,0,0,0.15)', borderRadius: '50%', zIndex: 2 }}>
        <SEPortrait id={4} size={120 * scale} />
      </div>
      <div style={{ position: 'absolute', top: 30 * scale, right: 0, transform: 'rotate(7deg)', boxShadow: '0 12px 28px rgba(0,0,0,0.12)', borderRadius: '50%' }}>
        <SEPortrait id={3} size={92 * scale} />
      </div>
    </div>
  );
}

// ═══ E1 — Login ═════════════════════════════════════════════════
function SoleilLogin() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'grid', gridTemplateColumns: '1fr 1.2fr', background: SE.bg, fontFamily: SE.sans, color: SE.text }}>
      {/* Left — form */}
      <div style={{ padding: '40px 64px', display: 'flex', flexDirection: 'column', position: 'relative' }}>
        <AtelierMark size={32} />

        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', maxWidth: 400, margin: '0 auto', width: '100%' }}>
          <div style={{ fontSize: 11, color: SE.accent, marginBottom: 12, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase' }}>Bon retour</div>
          <h1 style={{ fontFamily: SE.serif, fontSize: 44, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.05 }}>Reprends là <span style={{ fontStyle: 'italic', color: SE.accent }}>où tu en étais.</span></h1>
          <p style={{ fontSize: 14.5, color: SE.textMute, margin: '12px 0 32px', lineHeight: 1.55 }}>Connecte-toi pour suivre tes missions, tes candidatures, et reprendre tes conversations en cours.</p>

          {/* Email */}
          <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Adresse e-mail</label>
          <input type="email" defaultValue="elise.morel@hey.com" style={{ width: '100%', border: `1.5px solid ${SE.accent}`, borderRadius: 12, padding: '13px 16px', fontSize: 14.5, fontFamily: SE.sans, outline: 'none', marginBottom: 16, background: '#fff' }} />

          {/* Password */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 6 }}>
            <label style={{ fontSize: 13, fontWeight: 600 }}>Mot de passe</label>
            <a style={{ fontSize: 12, color: SE.accent, fontWeight: 600, cursor: 'pointer', fontStyle: 'italic', fontFamily: SE.serif }}>oublié ?</a>
          </div>
          <div style={{ position: 'relative', marginBottom: 22 }}>
            <input type="password" defaultValue="••••••••••" style={{ width: '100%', border: `1px solid ${SE.borderStrong}`, borderRadius: 12, padding: '13px 44px 13px 16px', fontSize: 14.5, fontFamily: SE.sans, outline: 'none', background: '#fff' }} />
            <span style={{ position: 'absolute', right: 14, top: '50%', transform: 'translateY(-50%)', cursor: 'pointer', color: SE.textMute }}><SEI name="Eye" size={16} /></span>
          </div>

          <button style={{ width: '100%', background: SE.accent, color: '#fff', border: 'none', padding: '14px', fontSize: 14.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', boxShadow: '0 4px 14px rgba(232,93,74,0.3)', marginBottom: 18 }}>Se connecter</button>

          {/* Or */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, margin: '8px 0 18px' }}>
            <span style={{ flex: 1, height: 1, background: SE.border }} />
            <span style={{ fontSize: 11, color: SE.textMute, fontStyle: 'italic', fontFamily: SE.serif }}>ou</span>
            <span style={{ flex: 1, height: 1, background: SE.border }} />
          </div>

          <div style={{ display: 'flex', gap: 8 }}>
            <button style={{ flex: 1, background: '#fff', border: `1px solid ${SE.borderStrong}`, padding: '11px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}>
              <svg width="16" height="16" viewBox="0 0 24 24"><path fill="#4285F4" d="M21.6 12.2c0-.7-.1-1.4-.2-2H12v3.8h5.4c-.2 1.3-.9 2.3-2 3v2.5h3.2c1.9-1.7 3-4.3 3-7.3z" /><path fill="#34A853" d="M12 22c2.7 0 5-.9 6.6-2.5l-3.2-2.5c-.9.6-2 1-3.4 1-2.6 0-4.8-1.7-5.6-4.1H3v2.5C4.7 19.7 8.1 22 12 22z" /><path fill="#FBBC04" d="M6.4 13.9c-.2-.6-.3-1.2-.3-1.9s.1-1.3.3-1.9V7.6H3C2.4 9 2 10.5 2 12s.4 3 1 4.4l3.4-2.5z" /><path fill="#EA4335" d="M12 5.9c1.5 0 2.8.5 3.8 1.5l2.8-2.8C16.9 3 14.7 2 12 2 8.1 2 4.7 4.3 3 7.6l3.4 2.5c.8-2.4 3-4.2 5.6-4.2z" /></svg>
              Google
            </button>
            <button style={{ flex: 1, background: '#fff', border: `1px solid ${SE.borderStrong}`, padding: '11px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill={SE.text}><path d="M17.05 20.28c-.98.95-2.05.8-3.08.35-1.09-.46-2.09-.48-3.24 0-1.44.62-2.2.44-3.06-.35C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.24 2.31-.93 3.57-.84 1.51.12 2.65.72 3.4 1.8-3.12 1.87-2.38 5.98.48 7.13-.57 1.5-1.31 2.99-2.54 4.09zM12 7.25c-.15-2.23 1.66-4.07 3.74-4.25.29 2.58-2.34 4.5-3.74 4.25z" /></svg>
              Apple
            </button>
          </div>

          <div style={{ marginTop: 28, fontSize: 13, color: SE.textMute, textAlign: 'center' }}>
            Pas encore de compte ? <a style={{ color: SE.accent, fontWeight: 600, cursor: 'pointer' }}>Créer un compte →</a>
          </div>
        </div>

        <div style={{ fontSize: 11, color: SE.textSubtle, textAlign: 'center' }}>
          © Atelier · <a style={{ color: 'inherit' }}>Conditions</a> · <a style={{ color: 'inherit' }}>Confidentialité</a>
        </div>
      </div>

      {/* Right — editorial visual */}
      <div style={{ background: 'linear-gradient(135deg, #fde9e3 0%, #fde6ed 50%, #fbf0dc 100%)', position: 'relative', overflow: 'hidden', display: 'flex', flexDirection: 'column', justifyContent: 'space-between', padding: 56 }}>
        <div style={{ position: 'absolute', top: -80, right: -80, width: 320, height: 320, borderRadius: '50%', background: 'radial-gradient(circle, rgba(232,93,74,0.25), transparent 65%)' }} />
        <div style={{ position: 'absolute', bottom: 80, left: -100, width: 260, height: 260, borderRadius: '50%', background: 'radial-gradient(circle, rgba(240,138,168,0.35), transparent 65%)' }} />

        {/* Manifeste éditorial */}
        <div style={{ position: 'relative', zIndex: 1, maxWidth: 460 }}>
          <div style={{ fontSize: 11, color: SE.accentDeep, marginBottom: 14, fontWeight: 700, letterSpacing: '0.14em', textTransform: 'uppercase' }}>↳ Atelier · le mot juste</div>
          <h2 style={{ fontFamily: SE.serif, fontSize: 38, fontWeight: 400, letterSpacing: '-0.02em', lineHeight: 1.1, margin: 0, color: SE.text, textWrap: 'pretty' }}>Une marketplace qui respecte <span style={{ fontStyle: 'italic', color: SE.accent }}>ton temps</span> et tes <span style={{ fontStyle: 'italic', color: SE.accent }}>tarifs.</span></h2>
        </div>

        {/* Trio portraits centered */}
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', position: 'relative', zIndex: 1, margin: '24px 0' }}>
          <PortraitTrio scale={1.4} />
        </div>

        {/* Trois piliers en bas */}
        <div style={{ position: 'relative', zIndex: 1, display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 18 }}>
          {[
            { icon: 'Shield', l: 'Paiement sécurisé', d: 'Séquestre Stripe sur chaque jalon' },
            { icon: 'Sparkle', l: 'Profils vérifiés', d: 'KYC, références, entretien humain' },
            { icon: 'Heart', l: 'Sans commission cachée', d: '5 % côté entreprise, c\'est tout' },
          ].map((p, i) => (
            <div key={i}>
              <div style={{ width: 32, height: 32, borderRadius: 10, background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SE.accentDeep, marginBottom: 10, boxShadow: '0 2px 8px rgba(42,31,21,0.06)' }}>
                <SEI name={p.icon} size={15} />
              </div>
              <div style={{ fontSize: 13.5, fontWeight: 600, marginBottom: 2, color: SE.text }}>{p.l}</div>
              <div style={{ fontSize: 12, color: SE.textMute, lineHeight: 1.45, textWrap: 'pretty' }}>{p.d}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ═══ E2 — Sign up · Choix de rôle ════════════════════════════════
function SoleilSignupRole() {
  return (
    <div style={{ width: '100%', height: '100%', background: SE.bg, fontFamily: SE.sans, color: SE.text, overflow: 'auto' }}>
      {/* Top bar */}
      <div style={{ padding: '28px 56px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <AtelierMark size={28} />
        <div style={{ fontSize: 13, color: SE.textMute }}>Déjà un compte ? <a style={{ color: SE.accent, fontWeight: 600, cursor: 'pointer' }}>Se connecter</a></div>
      </div>

      <div style={{ maxWidth: 920, margin: '0 auto', padding: '32px 32px 64px', textAlign: 'center' }}>
        <div style={{ fontSize: 11, color: SE.accent, marginBottom: 12, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase' }}>↳ Étape 1 sur 3</div>
        <h1 style={{ fontFamily: SE.serif, fontSize: 52, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.05 }}>Tu viens chercher <span style={{ fontStyle: 'italic', color: SE.accent }}>quoi sur Atelier ?</span></h1>
        <p style={{ fontSize: 16, color: SE.textMute, margin: '14px auto 0', maxWidth: 540, lineHeight: 1.55 }}>Pas d'inquiétude, tu peux changer plus tard. On adapte juste ce qu'on te montre.</p>

        {/* Three big cards */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 14, marginTop: 44, textAlign: 'left' }}>
          {/* Freelance — selected */}
          <div style={{ background: '#fff', border: `2px solid ${SE.accent}`, borderRadius: 24, padding: 28, position: 'relative', cursor: 'pointer', boxShadow: '0 8px 28px rgba(232,93,74,0.12)' }}>
            <div style={{ position: 'absolute', top: 18, right: 18, width: 26, height: 26, borderRadius: '50%', background: SE.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <SEI name="Check" size={14} />
            </div>
            <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 16 }}>
              <SEPortrait id={1} size={76} rounded={18} />
            </div>
            <div style={{ fontSize: 11, color: SE.accentDeep, fontWeight: 700, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 6, textAlign: 'center' }}>Je suis…</div>
            <h2 style={{ fontFamily: SE.serif, fontSize: 24, margin: 0, fontWeight: 500, letterSpacing: '-0.02em', textAlign: 'center', marginBottom: 8 }}>Freelance</h2>
            <p style={{ fontSize: 13, color: SE.textMute, margin: '0 0 18px', textAlign: 'center', lineHeight: 1.5, textWrap: 'pretty' }}>Designer, dev, brand strategist, copywriter… Tu cherches des missions qui te ressemblent.</p>
            <div style={{ borderTop: `1px solid ${SE.border}`, paddingTop: 14, display: 'flex', flexDirection: 'column', gap: 8 }}>
              {[
                'Profil éditorial, pas un CV',
                'Missions présélectionnées',
                'Paiements sécurisés',
              ].map((b, i) => (
                <div key={i} style={{ display: 'flex', gap: 10, fontSize: 12.5, color: SE.text, alignItems: 'center' }}>
                  <span style={{ width: 16, height: 16, borderRadius: '50%', background: SE.accentSoft, color: SE.accent, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                    <SEI name="Check" size={10} />
                  </span>
                  {b}
                </div>
              ))}
            </div>
          </div>

          {/* Entreprise */}
          <div style={{ background: '#fff', border: `1px solid ${SE.border}`, borderRadius: 24, padding: 28, cursor: 'pointer' }}>
            <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 16 }}>
              <div style={{ width: 76, height: 76, borderRadius: 18, background: 'linear-gradient(135deg, #fbf0dc, #fde6ed)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SE.text }}>
                <SEI name="Building" size={36} />
              </div>
            </div>
            <div style={{ fontSize: 11, color: SE.textMute, fontWeight: 700, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 6, textAlign: 'center' }}>Je représente…</div>
            <h2 style={{ fontFamily: SE.serif, fontSize: 24, margin: 0, fontWeight: 500, letterSpacing: '-0.02em', textAlign: 'center', marginBottom: 8 }}>Une entreprise</h2>
            <p style={{ fontSize: 13, color: SE.textMute, margin: '0 0 18px', textAlign: 'center', lineHeight: 1.5, textWrap: 'pretty' }}>Startup, studio, agence… Tu cherches des prestataires de confiance pour des missions précises.</p>
            <div style={{ borderTop: `1px solid ${SE.border}`, paddingTop: 14, display: 'flex', flexDirection: 'column', gap: 8 }}>
              {[
                'Annonces ciblées, pas de spam',
                'Profils vérifiés',
                'Séquestre sécurisé',
              ].map((b, i) => (
                <div key={i} style={{ display: 'flex', gap: 10, fontSize: 12.5, color: SE.text, alignItems: 'center' }}>
                  <span style={{ width: 16, height: 16, borderRadius: '50%', background: SE.bg, color: SE.textMute, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                    <SEI name="Check" size={10} />
                  </span>
                  {b}
                </div>
              ))}
            </div>
          </div>

          {/* Apporteur d'affaires */}
          <div style={{ background: '#fff', border: `1px solid ${SE.border}`, borderRadius: 24, padding: 28, cursor: 'pointer', position: 'relative' }}>
            <div style={{ position: 'absolute', top: 14, right: 14, fontSize: 10, padding: '3px 9px', background: SE.text, color: '#fff', borderRadius: 999, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase' }}>Nouveau</div>
            <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 16 }}>
              <div style={{ width: 76, height: 76, borderRadius: 18, background: 'linear-gradient(135deg, #fde9e3, #fbf0dc)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SE.accentDeep }}>
                <SEI name="Sparkle" size={36} />
              </div>
            </div>
            <div style={{ fontSize: 11, color: SE.textMute, fontWeight: 700, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 6, textAlign: 'center' }}>Je suis…</div>
            <h2 style={{ fontFamily: SE.serif, fontSize: 24, margin: 0, fontWeight: 500, letterSpacing: '-0.02em', textAlign: 'center', marginBottom: 8 }}>Apporteur d'affaires</h2>
            <p style={{ fontSize: 13, color: SE.textMute, margin: '0 0 18px', textAlign: 'center', lineHeight: 1.5, textWrap: 'pretty' }}>Tu connais des entreprises ou des freelances. Tu les recommandes, on te rémunère sur la mission.</p>
            <div style={{ borderTop: `1px solid ${SE.border}`, paddingTop: 14, display: 'flex', flexDirection: 'column', gap: 8 }}>
              {[
                'Lien de parrainage perso',
                'Commission sur chaque mission',
                'Suivi des conversions en direct',
              ].map((b, i) => (
                <div key={i} style={{ display: 'flex', gap: 10, fontSize: 12.5, color: SE.text, alignItems: 'center' }}>
                  <span style={{ width: 16, height: 16, borderRadius: '50%', background: SE.bg, color: SE.textMute, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                    <SEI name="Check" size={10} />
                  </span>
                  {b}
                </div>
              ))}
            </div>
          </div>
        </div>

        <div style={{ marginTop: 36, display: 'flex', justifyContent: 'center', gap: 8 }}>
          <button style={{ background: SE.accent, color: '#fff', border: 'none', padding: '14px 32px', fontSize: 14.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', boxShadow: '0 4px 14px rgba(232,93,74,0.3)', display: 'flex', alignItems: 'center', gap: 8 }}>
            Continuer en tant que freelance <SEI name="ArrowRight" size={14} />
          </button>
        </div>
      </div>
    </div>
  );
}

// ═══ E3 — Sign up · Formulaire freelance ════════════════════════
function SoleilSignupFreelance() {
  return (
    <div style={{ width: '100%', height: '100%', background: SE.bg, fontFamily: SE.sans, color: SE.text, overflow: 'auto' }}>
      <div style={{ padding: '28px 56px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <AtelierMark size={28} />
        <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
          {/* Stepper compact */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            {[1, 2, 3].map((n, i) => (
              <React.Fragment key={i}>
                {i > 0 && <span style={{ width: 18, height: 1, background: i <= 1 ? SE.accent : SE.border }} />}
                <div style={{ width: 22, height: 22, borderRadius: '50%', background: i === 0 ? SE.green : i === 1 ? SE.accent : '#fff', border: i === 2 ? `1.5px solid ${SE.border}` : 'none', color: i === 2 ? SE.textMute : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, fontWeight: 600 }}>
                  {i === 0 ? <SEI name="Check" size={11} /> : n}
                </div>
              </React.Fragment>
            ))}
          </div>
          <div style={{ fontSize: 13, color: SE.textMute }}>Étape 2 sur 3</div>
        </div>
      </div>

      <div style={{ maxWidth: 720, margin: '0 auto', padding: '24px 32px 64px' }}>
        <div style={{ fontSize: 11, color: SE.accent, marginBottom: 10, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase' }}>Bienvenue parmi nous</div>
        <h1 style={{ fontFamily: SE.serif, fontSize: 42, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.05 }}>Parle-nous <span style={{ fontStyle: 'italic', color: SE.accent }}>un peu de toi.</span></h1>
        <p style={{ fontSize: 15, color: SE.textMute, margin: '10px 0 32px', maxWidth: 560, lineHeight: 1.55 }}>On garde ça court. Le reste — projets, références, témoignages — viendra ensuite, à ton rythme.</p>

        <div style={{ background: '#fff', border: `1px solid ${SE.border}`, borderRadius: 20, padding: 36 }}>
          {/* Photo */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 18, marginBottom: 28 }}>
            <div style={{ position: 'relative' }}>
              <SEPortrait id={1} size={80} rounded={16} />
              <div style={{ position: 'absolute', bottom: -3, right: -3, width: 26, height: 26, borderRadius: '50%', background: SE.text, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}><SEI name="Edit" size={12} /></div>
            </div>
            <div>
              <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 3 }}>Ta photo</div>
              <div style={{ fontSize: 12.5, color: SE.textMute, fontStyle: 'italic', fontFamily: SE.serif }}>Carrée, 400×400 minimum, un visage souriant.</div>
            </div>
          </div>

          {/* Identité */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14, marginBottom: 18 }}>
            <div>
              <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Prénom</label>
              <input defaultValue="Élise" style={{ width: '100%', border: `1px solid ${SE.borderStrong}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, fontFamily: SE.sans, outline: 'none', background: '#fff' }} />
            </div>
            <div>
              <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Nom</label>
              <input defaultValue="Morel" style={{ width: '100%', border: `1px solid ${SE.borderStrong}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, fontFamily: SE.sans, outline: 'none', background: '#fff' }} />
            </div>
          </div>

          <div style={{ marginBottom: 18 }}>
            <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Ton métier <span style={{ fontWeight: 400, color: SE.textMute, fontSize: 12 }}>· tel que tu te présenterais en soirée</span></label>
            <input defaultValue="UX/UI Designer · spécialisée SaaS B2B" style={{ width: '100%', border: `1.5px solid ${SE.accent}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, fontFamily: SE.sans, outline: 'none', background: '#fff' }} />
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14, marginBottom: 18 }}>
            <div>
              <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Ville</label>
              <div style={{ position: 'relative' }}>
                <input defaultValue="Paris, 11ᵉ" style={{ width: '100%', border: `1px solid ${SE.borderStrong}`, borderRadius: 10, padding: '11px 14px 11px 38px', fontSize: 14, fontFamily: SE.sans, outline: 'none', background: '#fff' }} />
                <span style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: SE.textMute }}><SEI name="MapPin" size={15} /></span>
              </div>
            </div>
            <div>
              <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>TJM indicatif</label>
              <div style={{ position: 'relative' }}>
                <input defaultValue="650" style={{ width: '100%', border: `1px solid ${SE.borderStrong}`, borderRadius: 10, padding: '11px 38px 11px 14px', fontSize: 14, fontFamily: SE.sans, outline: 'none', background: '#fff' }} />
                <span style={{ position: 'absolute', right: 14, top: '50%', transform: 'translateY(-50%)', color: SE.textMute, fontSize: 13, fontFamily: SE.serif }}>€/j</span>
              </div>
            </div>
          </div>

          {/* Compétences chips */}
          <div style={{ marginBottom: 18 }}>
            <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 8, display: 'block' }}>Compétences <span style={{ fontWeight: 400, color: SE.textMute, fontSize: 12 }}>· choisis 3 à 5 max</span></label>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
              {[
                { l: 'UX Design', selected: true },
                { l: 'UI Design', selected: true },
                { l: 'Design System', selected: true },
                { l: 'Mobile' },
                { l: 'Brand identity' },
                { l: 'UX Research' },
                { l: 'Webflow' },
                { l: 'Figma', selected: true },
                { l: 'Prototypage' },
                { l: 'Design éditorial' },
              ].map((s, i) => (
                <button key={i} style={{ background: s.selected ? SE.text : '#fff', color: s.selected ? '#fff' : SE.text, border: s.selected ? 'none' : `1px solid ${SE.border}`, padding: '7px 13px', fontSize: 13, fontWeight: 500, borderRadius: 999, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                  {s.selected && <SEI name="Check" size={11} />}
                  {s.l}
                </button>
              ))}
            </div>
          </div>

          {/* Bio */}
          <div style={{ marginBottom: 18 }}>
            <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 4, display: 'block' }}>Présente-toi en 2 lignes</label>
            <div style={{ fontSize: 12, color: SE.textMute, marginBottom: 8, fontStyle: 'italic', fontFamily: SE.serif }}>« Tu fais quoi, et tu cherches quoi. »</div>
            <textarea defaultValue="Designer produit avec 7 ans d'expérience en SaaS B2B. J'aime les phases de discovery et les design systems durables. Je travaille en hybride sur Paris, à 80 % du temps." style={{ width: '100%', minHeight: 90, border: `1px solid ${SE.borderStrong}`, borderRadius: 10, padding: '12px 14px', fontSize: 14, fontFamily: SE.sans, outline: 'none', background: '#fff', lineHeight: 1.55, resize: 'vertical' }} />
          </div>

          {/* Disponibilité */}
          <div>
            <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 8, display: 'block' }}>Disponibilité actuelle</label>
            <div style={{ display: 'flex', gap: 8 }}>
              {[
                { l: 'Disponible maintenant', icon: 'CheckCircle', selected: true, color: SE.green },
                { l: 'Dispo dans < 1 mois', icon: 'Clock' },
                { l: 'Pas dispo', icon: 'Pin' },
              ].map((d, i) => (
                <button key={i} style={{ flex: 1, background: d.selected ? SE.greenSoft : '#fff', border: `1.5px solid ${d.selected ? SE.green : SE.border}`, borderRadius: 12, padding: '12px 14px', fontSize: 13, fontWeight: 600, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, color: d.selected ? SE.green : SE.text }}>
                  <SEI name={d.icon} size={14} />
                  {d.l}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 24 }}>
          <button style={{ background: 'none', border: 'none', fontSize: 14, color: SE.textMute, fontWeight: 500, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
            <SEI name="ArrowLeft" size={14} /> Retour
          </button>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <span style={{ fontSize: 12, color: SE.textMute, fontStyle: 'italic', fontFamily: SE.serif }}>Tu pourras compléter plus tard</span>
            <button style={{ background: SE.accent, color: '#fff', border: 'none', padding: '12px 24px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', boxShadow: '0 2px 8px rgba(232,93,74,0.25)', display: 'flex', alignItems: 'center', gap: 6 }}>
              Configurer mes paiements <SEI name="ArrowRight" size={14} />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ E4 — Sign up · Formulaire entreprise ═══════════════════════
function SoleilSignupCompany() {
  return (
    <div style={{ width: '100%', height: '100%', background: SE.bg, fontFamily: SE.sans, color: SE.text, overflow: 'auto' }}>
      <div style={{ padding: '28px 56px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <AtelierMark size={28} />
        <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            {[1, 2, 3].map((n, i) => (
              <React.Fragment key={i}>
                {i > 0 && <span style={{ width: 18, height: 1, background: i <= 1 ? SE.accent : SE.border }} />}
                <div style={{ width: 22, height: 22, borderRadius: '50%', background: i === 0 ? SE.green : i === 1 ? SE.accent : '#fff', border: i === 2 ? `1.5px solid ${SE.border}` : 'none', color: i === 2 ? SE.textMute : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, fontWeight: 600 }}>
                  {i === 0 ? <SEI name="Check" size={11} /> : n}
                </div>
              </React.Fragment>
            ))}
          </div>
          <div style={{ fontSize: 13, color: SE.textMute }}>Étape 2 sur 3</div>
        </div>
      </div>

      <div style={{ maxWidth: 720, margin: '0 auto', padding: '24px 32px 64px' }}>
        <div style={{ fontSize: 11, color: SE.accent, marginBottom: 10, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase' }}>Ton entreprise</div>
        <h1 style={{ fontFamily: SE.serif, fontSize: 42, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.05 }}>Présente <span style={{ fontStyle: 'italic', color: SE.accent }}>ton équipe.</span></h1>
        <p style={{ fontSize: 15, color: SE.textMute, margin: '10px 0 32px', maxWidth: 560, lineHeight: 1.55 }}>Les freelances aiment savoir avec qui ils vont travailler. Quelques infos suffisent pour démarrer.</p>

        <div style={{ background: '#fff', border: `1px solid ${SE.border}`, borderRadius: 20, padding: 36 }}>
          {/* Logo */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 18, marginBottom: 28 }}>
            <div style={{ position: 'relative', width: 80, height: 80, borderRadius: 16, background: 'linear-gradient(135deg, #fbf0dc, #fde6ed)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SE.text, fontFamily: SE.serif, fontSize: 32, fontWeight: 600 }}>
              N
              <div style={{ position: 'absolute', bottom: -3, right: -3, width: 26, height: 26, borderRadius: '50%', background: SE.text, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}><SEI name="Edit" size={12} /></div>
            </div>
            <div>
              <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 3 }}>Logo de l'entreprise</div>
              <div style={{ fontSize: 12.5, color: SE.textMute, fontStyle: 'italic', fontFamily: SE.serif }}>Format carré, fond uni de préférence.</div>
            </div>
          </div>

          {/* Identité */}
          <div style={{ marginBottom: 18 }}>
            <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Nom de l'entreprise</label>
            <input defaultValue="Nova Studio" style={{ width: '100%', border: `1.5px solid ${SE.accent}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, fontFamily: SE.sans, outline: 'none', background: '#fff' }} />
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14, marginBottom: 18 }}>
            <div>
              <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Site web</label>
              <input defaultValue="nova-studio.fr" style={{ width: '100%', border: `1px solid ${SE.borderStrong}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, fontFamily: SE.sans, outline: 'none', background: '#fff' }} />
            </div>
            <div>
              <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Numéro SIRET</label>
              <input defaultValue="852 369 147 00012" style={{ width: '100%', border: `1px solid ${SE.borderStrong}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, fontFamily: SE.mono, outline: 'none', background: '#fff', letterSpacing: '0.04em' }} />
            </div>
          </div>

          {/* Secteur + taille */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14, marginBottom: 18 }}>
            <div>
              <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Secteur</label>
              <div style={{ position: 'relative' }}>
                <select style={{ width: '100%', appearance: 'none', border: `1px solid ${SE.borderStrong}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, background: '#fff', fontFamily: SE.sans, color: SE.text, cursor: 'pointer' }}>
                  <option>SaaS B2B</option>
                </select>
                <span style={{ position: 'absolute', right: 14, top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none' }}><SEI name="ChevronDown" size={14} /></span>
              </div>
            </div>
            <div>
              <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Taille de l'équipe</label>
              <div style={{ display: 'flex', gap: 6 }}>
                {['1–5', '6–20', '21–50', '50+'].map((c, i) => (
                  <button key={i} style={{ flex: 1, padding: '11px 6px', background: i === 1 ? SE.text : '#fff', color: i === 1 ? '#fff' : SE.text, border: i === 1 ? 'none' : `1px solid ${SE.borderStrong}`, borderRadius: 10, fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>{c}</button>
                ))}
              </div>
            </div>
          </div>

          {/* Pitch */}
          <div style={{ marginBottom: 18 }}>
            <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 4, display: 'block' }}>En quelques lignes, c'est quoi Nova Studio ?</label>
            <div style={{ fontSize: 12, color: SE.textMute, marginBottom: 8, fontStyle: 'italic', fontFamily: SE.serif }}>« Le truc que vous faites, et pour qui. »</div>
            <textarea defaultValue="Nova Studio aide les studios créatifs à gérer leurs projets de bout en bout. Série A en mars 2026, équipe produit de 12 personnes basée à Paris." style={{ width: '100%', minHeight: 90, border: `1px solid ${SE.borderStrong}`, borderRadius: 10, padding: '12px 14px', fontSize: 14, fontFamily: SE.sans, outline: 'none', background: '#fff', lineHeight: 1.55, resize: 'vertical' }} />
          </div>

          {/* Tu cherches plutôt */}
          <div>
            <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 8, display: 'block' }}>Tu cherches plutôt…</label>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
              {[
                { l: 'Designers', selected: true },
                { l: 'Dev front-end', selected: true },
                { l: 'Dev back-end' },
                { l: 'Product Managers' },
                { l: 'Brand strategists', selected: true },
                { l: 'Copywriters' },
                { l: 'Motion designers' },
                { l: 'Data / IA' },
              ].map((s, i) => (
                <button key={i} style={{ background: s.selected ? SE.accentSoft : '#fff', color: s.selected ? SE.accentDeep : SE.text, border: `1px solid ${s.selected ? SE.accent : SE.border}`, padding: '7px 13px', fontSize: 13, fontWeight: 500, borderRadius: 999, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                  {s.selected && <SEI name="Check" size={11} />}
                  {s.l}
                </button>
              ))}
            </div>
          </div>
        </div>

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 24 }}>
          <button style={{ background: 'none', border: 'none', fontSize: 14, color: SE.textMute, fontWeight: 500, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
            <SEI name="ArrowLeft" size={14} /> Retour
          </button>
          <button style={{ background: SE.accent, color: '#fff', border: 'none', padding: '12px 24px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', boxShadow: '0 2px 8px rgba(232,93,74,0.25)', display: 'flex', alignItems: 'center', gap: 6 }}>
            Publier ma première annonce <SEI name="ArrowRight" size={14} />
          </button>
        </div>
      </div>
    </div>
  );
}

// ═══ E5 — Stripe Connect wrapper ════════════════════════════════
// Affiche 2 états côte-à-côte dans une seule artboard : initial (onboarding paiements) + urgent (action requise)
function SoleilStripeConnect() {
  return (
    <div style={{ width: '100%', height: '100%', background: SE.bg, fontFamily: SE.sans, color: SE.text, display: 'grid', gridTemplateRows: '1fr 1fr' }}>
      {/* — État 1 : initial onboarding (post-signup freelance) — */}
      <div style={{ padding: '40px 56px', borderBottom: `2px dashed ${SE.borderStrong}`, position: 'relative', overflow: 'hidden' }}>
        <span style={{ position: 'absolute', top: 14, left: 14, fontSize: 10, fontFamily: SE.mono, color: SE.textSubtle, letterSpacing: '0.08em', textTransform: 'uppercase' }}>État 1 · Onboarding initial</span>

        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', maxWidth: 1100, margin: '0 auto', height: '100%' }}>
          {/* Left : copy */}
          <div style={{ flex: '0 0 460px' }}>
            <div style={{ fontSize: 11, color: SE.accent, marginBottom: 12, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase' }}>↳ Étape 3 sur 3 · presque fini</div>
            <h1 style={{ fontFamily: SE.serif, fontSize: 38, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.05 }}>Configure tes <span style={{ fontStyle: 'italic', color: SE.accent }}>paiements.</span></h1>
            <p style={{ fontSize: 14.5, color: SE.textMute, margin: '12px 0 22px', lineHeight: 1.55, textWrap: 'pretty' }}>On utilise Stripe pour sécuriser tes paiements et te verser ton argent rapidement. 4 minutes, c'est fait pour la vie.</p>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginBottom: 22 }}>
              {[
                { l: 'Pièce d\'identité', d: 'Photo recto-verso d\'une carte d\'identité ou passeport' },
                { l: 'Adresse personnelle', d: 'Pour valider ton identité auprès de Stripe' },
                { l: 'IBAN', d: 'Vers lequel on te verse les paiements de tes missions' },
              ].map((s, i) => (
                <div key={i} style={{ display: 'flex', gap: 12, padding: 12, background: '#fff', border: `1px solid ${SE.border}`, borderRadius: 10 }}>
                  <div style={{ width: 28, height: 28, borderRadius: '50%', background: SE.bg, color: SE.textMute, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, fontFamily: SE.serif, fontSize: 13, fontWeight: 600 }}>{i + 1}</div>
                  <div>
                    <div style={{ fontSize: 13.5, fontWeight: 600, marginBottom: 1 }}>{s.l}</div>
                    <div style={{ fontSize: 12, color: SE.textMute, lineHeight: 1.45 }}>{s.d}</div>
                  </div>
                </div>
              ))}
            </div>

            <div style={{ display: 'flex', gap: 10 }}>
              <button style={{ background: SE.accent, color: '#fff', border: 'none', padding: '12px 22px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', boxShadow: '0 2px 8px rgba(232,93,74,0.25)', display: 'flex', alignItems: 'center', gap: 6 }}>
                Continuer avec Stripe <SEI name="ArrowRight" size={14} />
              </button>
              <button style={{ background: 'none', border: 'none', fontSize: 13, color: SE.textMute, fontWeight: 500, cursor: 'pointer', fontStyle: 'italic', fontFamily: SE.serif }}>Plus tard</button>
            </div>
          </div>

          {/* Right : Stripe modal preview */}
          <div style={{ width: 380, background: '#fff', borderRadius: 16, padding: 24, boxShadow: '0 12px 36px rgba(42,31,21,0.12)', border: `1px solid ${SE.border}`, position: 'relative' }}>
            <div style={{ position: 'absolute', top: -10, right: -10, fontSize: 10, padding: '4px 10px', background: SE.text, color: '#fff', borderRadius: 999, fontWeight: 600, letterSpacing: '0.06em', textTransform: 'uppercase' }}>Aperçu</div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 18 }}>
              <span style={{ fontFamily: '-apple-system, BlinkMacSystemFont, sans-serif', fontSize: 16, fontWeight: 700, color: '#635BFF', letterSpacing: '-0.01em' }}>stripe</span>
              <span style={{ fontSize: 11, color: SE.textMute }}>Powered by Stripe Connect</span>
            </div>
            <div style={{ fontSize: 12, color: SE.textMute, marginBottom: 4, fontFamily: SE.mono }}>Compte de paiement</div>
            <div style={{ fontSize: 18, fontWeight: 600, marginBottom: 18 }}>Bienvenue sur Atelier</div>
            <div style={{ fontSize: 12.5, color: SE.text, lineHeight: 1.5, marginBottom: 16 }}>Stripe va te demander quelques infos pour vérifier ton identité et configurer ton compte de paiement.</div>
            <div style={{ borderTop: `1px solid ${SE.border}`, paddingTop: 14, display: 'flex', flexDirection: 'column', gap: 8, fontSize: 12 }}>
              {['Identité', 'Adresse', 'Coordonnées bancaires (IBAN)', 'Vérification finale'].map((s, i) => (
                <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 9, color: SE.textMute }}>
                  <div style={{ width: 18, height: 18, borderRadius: '50%', border: `1.5px solid ${SE.border}` }} />
                  {s}
                </div>
              ))}
            </div>
            <button style={{ width: '100%', background: '#635BFF', color: '#fff', border: 'none', padding: '11px', fontSize: 13, fontWeight: 600, borderRadius: 8, cursor: 'pointer', marginTop: 18 }}>Démarrer la vérification</button>
            <div style={{ marginTop: 12, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6, fontSize: 11, color: SE.textMute }}>
              <SEI name="Shield" size={11} /> Tes données sont chiffrées et hébergées par Stripe
            </div>
          </div>
        </div>
      </div>

      {/* — État 2 : urgent / action requise — */}
      <div style={{ padding: '40px 56px', position: 'relative', overflow: 'hidden', background: 'linear-gradient(to bottom, #fffbf5, #fff5ec)' }}>
        <span style={{ position: 'absolute', top: 14, left: 14, fontSize: 10, fontFamily: SE.mono, color: SE.textSubtle, letterSpacing: '0.08em', textTransform: 'uppercase' }}>État 2 · Action requise (urgent)</span>

        <div style={{ maxWidth: 880, margin: '0 auto', height: '100%', display: 'flex', alignItems: 'center' }}>
          <div style={{ background: '#fff', borderRadius: 20, border: `2px solid ${SE.accent}`, padding: 36, width: '100%', boxShadow: '0 12px 36px rgba(232,93,74,0.18)', position: 'relative' }}>
            {/* Stripe corner */}
            <div style={{ position: 'absolute', top: 18, right: 20, fontSize: 11, color: SE.textMute, display: 'flex', alignItems: 'center', gap: 6 }}>
              via <span style={{ fontFamily: '-apple-system, BlinkMacSystemFont, sans-serif', fontSize: 13, fontWeight: 700, color: '#635BFF' }}>stripe</span>
            </div>

            <div style={{ display: 'flex', gap: 22, alignItems: 'flex-start' }}>
              <div style={{ flexShrink: 0, width: 56, height: 56, borderRadius: '50%', background: SE.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 4px 12px rgba(232,93,74,0.3)' }}>
                <SEI name="Shield" size={26} />
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 11, color: SE.accentDeep, marginBottom: 8, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase' }}>↳ Action requise sous 7 jours</div>
                <h2 style={{ fontFamily: SE.serif, fontSize: 30, margin: 0, fontWeight: 400, letterSpacing: '-0.02em', lineHeight: 1.15 }}>Stripe a besoin d'une <span style={{ fontStyle: 'italic', color: SE.accent }}>information complémentaire.</span></h2>
                <p style={{ fontSize: 14, color: SE.textMute, margin: '10px 0 18px', lineHeight: 1.55, maxWidth: 600, textWrap: 'pretty' }}>Pour rester en conformité avec la réglementation européenne, ton compte de paiement nécessite une vérification additionnelle. Sans ça, on ne pourra plus te verser tes prochains paiements.</p>

                {/* What's needed */}
                <div style={{ background: SE.bg, borderRadius: 12, padding: 16, marginBottom: 18, display: 'flex', flexDirection: 'column', gap: 10 }}>
                  <div style={{ fontSize: 12, fontWeight: 600, color: SE.text, marginBottom: 4 }}>Ce qu'il manque :</div>
                  {[
                    { l: 'Justificatif de domicile de moins de 3 mois', urgent: true },
                    { l: 'Confirmation de ton numéro de TVA intracommunautaire' },
                  ].map((it, i) => (
                    <div key={i} style={{ display: 'flex', gap: 10, alignItems: 'center', fontSize: 13 }}>
                      <span style={{ width: 16, height: 16, borderRadius: '50%', border: `1.5px solid ${it.urgent ? SE.accent : SE.borderStrong}`, background: it.urgent ? SE.accentSoft : 'transparent', flexShrink: 0 }} />
                      <span style={{ color: SE.text }}>{it.l}</span>
                      {it.urgent && <span style={{ marginLeft: 'auto', fontSize: 10, padding: '2px 8px', background: SE.accentSoft, color: SE.accentDeep, borderRadius: 999, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase' }}>Bloquant</span>}
                    </div>
                  ))}
                </div>

                {/* Impact */}
                <div style={{ display: 'flex', gap: 18, fontSize: 12.5, color: SE.textMute, marginBottom: 22, flexWrap: 'wrap' }}>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                    <SEI name="Clock" size={13} /> Plus que <strong style={{ color: SE.accentDeep }}>4 jours</strong> pour régulariser
                  </span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                    <SEI name="Euro" size={13} /> <strong>3 675 €</strong> en attente de versement
                  </span>
                </div>

                <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
                  <button style={{ background: SE.accent, color: '#fff', border: 'none', padding: '13px 24px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', boxShadow: '0 4px 14px rgba(232,93,74,0.3)', display: 'flex', alignItems: 'center', gap: 6 }}>
                    Régulariser maintenant <SEI name="ArrowRight" size={14} />
                  </button>
                  <button style={{ background: '#fff', border: `1px solid ${SE.borderStrong}`, padding: '12px 18px', fontSize: 13.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Voir les détails</button>
                  <span style={{ fontSize: 12, color: SE.textMute, fontStyle: 'italic', fontFamily: SE.serif, marginLeft: 8 }}>~ 3 minutes</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ E6 — Compte (1 écran) ═══════════════════════════════════════
function SoleilAccount() {
  const tabs = [
    { k: 'notif', l: 'Notifications', icon: 'Bell', active: true },
    { k: 'email', l: 'Email', icon: 'Mail' },
    { k: 'pwd', l: 'Mot de passe', icon: 'Shield' },
    { k: 'data', l: 'Données et suppression', icon: 'Sliders' },
  ];

  const sections = [
    {
      title: 'Propositions & Projets',
      rows: [
        { l: 'Nouvelle proposition reçue', p: true, e: true },
        { l: 'Proposition acceptée', p: true, e: true },
        { l: 'Proposition refusée', p: true, e: true },
        { l: 'Proposition modifiée', p: true, e: false },
        { l: 'Paiement reçu', p: true, e: true },
        { l: 'Achèvement demandé', p: true, e: true },
        { l: 'Mission terminée', p: true, e: true },
      ],
    },
    {
      title: 'Avis',
      rows: [
        { l: 'Nouvel avis reçu', p: true, e: false },
        { l: 'Réponse à ton avis', p: true, e: true },
      ],
    },
    {
      title: 'Messages',
      rows: [
        { l: 'Nouveau message', p: true, e: true },
        { l: 'Mention dans une conversation', p: true, e: true },
        { l: 'Conversation archivée', p: false, e: false },
      ],
    },
    {
      title: 'Opportunités & Candidatures',
      rows: [
        { l: 'Nouvelle opportunité qui matche ton profil', p: true, e: true },
        { l: 'Réponse à une candidature', p: true, e: true },
        { l: 'Annonce favorite mise à jour', p: false, e: true },
      ],
    },
  ];

  const Toggle = ({ on }) => (
    <div style={{ width: 36, height: 20, borderRadius: 999, background: on ? SE.accent : SE.border, position: 'relative', cursor: 'pointer', transition: 'background 0.15s' }}>
      <div style={{ width: 16, height: 16, borderRadius: '50%', background: '#fff', position: 'absolute', top: 2, left: on ? 18 : 2, transition: 'left 0.15s', boxShadow: '0 1px 3px rgba(0,0,0,0.18)' }} />
    </div>
  );

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SE.bg, fontFamily: SE.sans, color: SE.text }}>
      <SESidebar active="settings" role="freelancer" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SETopbar />
        <div style={{ flex: 1, overflow: 'auto', padding: '32px 40px' }}>
          <div style={{ maxWidth: 1100, margin: '0 auto' }}>
            <h1 style={{ fontFamily: SE.serif, fontSize: 32, margin: '0 0 28px', fontWeight: 500, letterSpacing: '-0.02em' }}>Paramètres du compte</h1>

            <div style={{ display: 'grid', gridTemplateColumns: '240px 1fr', gap: 28, alignItems: 'flex-start' }}>
              {/* Sidebar tabs */}
              <div style={{ background: '#fff', border: `1px solid ${SE.border}`, borderRadius: 14, padding: 8, display: 'flex', flexDirection: 'column', gap: 2 }}>
                {tabs.map((t, i) => (
                  <button key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', background: t.active ? SE.accentSoft : 'transparent', color: t.active ? SE.accentDeep : SE.text, border: 'none', borderRadius: 10, fontSize: 13.5, fontWeight: t.active ? 600 : 500, cursor: 'pointer', textAlign: 'left', fontFamily: SE.sans }}>
                    <SEI name={t.icon} size={15} />
                    {t.l}
                  </button>
                ))}
              </div>

              {/* Content */}
              <div>
                <div style={{ marginBottom: 22 }}>
                  <h2 style={{ fontFamily: SE.serif, fontSize: 22, margin: '0 0 6px', fontWeight: 600, letterSpacing: '-0.01em' }}>Préférences de notification</h2>
                  <p style={{ fontSize: 13.5, color: SE.textMute, margin: 0 }}>Choisis comment tu souhaites être notifié·e pour chaque type d'événement.</p>
                </div>

                {/* Toggle global */}
                <div style={{ background: '#fff', border: `1px solid ${SE.border}`, borderRadius: 14, padding: '16px 18px', marginBottom: 18, display: 'flex', alignItems: 'center', gap: 14 }}>
                  <div style={{ width: 36, height: 36, borderRadius: 10, background: SE.accentSoft, color: SE.accent, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                    <SEI name="Mail" size={16} />
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 2 }}>Notifications email activées</div>
                    <div style={{ fontSize: 12, color: SE.textMute }}>Active ou désactive tous les emails d'un coup (pratique en dispo, limite Resond 100/jour).</div>
                  </div>
                  <Toggle on={true} />
                </div>

                {/* Sections */}
                {sections.map((s, si) => (
                  <div key={si} style={{ background: '#fff', border: `1px solid ${SE.border}`, borderRadius: 14, marginBottom: 14, overflow: 'hidden' }}>
                    <div style={{ padding: '14px 20px', borderBottom: `1px solid ${SE.border}`, fontSize: 14, fontWeight: 700 }}>{s.title}</div>
                    {/* Header row */}
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 70px 70px', alignItems: 'center', padding: '8px 20px', fontSize: 11, color: SE.textMute, fontWeight: 600, letterSpacing: '0.06em', textTransform: 'uppercase', borderBottom: `1px solid ${SE.border}`, background: SE.bg }}>
                      <span>Type</span>
                      <span style={{ textAlign: 'center' }}>Push</span>
                      <span style={{ textAlign: 'center' }}>Email</span>
                    </div>
                    {s.rows.map((r, ri) => (
                      <div key={ri} style={{ display: 'grid', gridTemplateColumns: '1fr 70px 70px', alignItems: 'center', padding: '12px 20px', borderTop: ri > 0 ? `1px solid ${SE.border}` : 'none', fontSize: 13.5 }}>
                        <span>{r.l}</span>
                        <span style={{ display: 'flex', justifyContent: 'center' }}><Toggle on={r.p} /></span>
                        <span style={{ display: 'flex', justifyContent: 'center' }}><Toggle on={r.e} /></span>
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

window.SoleilLogin = SoleilLogin;
window.SoleilSignupRole = SoleilSignupRole;
window.SoleilSignupFreelance = SoleilSignupFreelance;
window.SoleilSignupCompany = SoleilSignupCompany;
window.SoleilStripeConnect = SoleilStripeConnect;
window.SoleilAccount = SoleilAccount;
