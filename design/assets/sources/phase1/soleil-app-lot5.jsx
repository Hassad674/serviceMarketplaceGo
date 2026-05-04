// App Lot 5 — Compte : Login, Signup, Profil/Paramètres
const SL5 = window.S;
const SL5I = window.SI;
const _AppFrame_L5 = window.AppFrame;
const _AppTabBar_L5 = window.AppTabBar;
const SL5Portrait = window.Portrait;

// ─── Login ─────────────────────────────────────────────────
function AppLogin() {
  return (
    <_AppFrame_L5 bg="#fff">
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', padding: '20px 28px 28px' }}>
        {/* Logo + intro */}
        <div style={{ marginTop: 40 }}>
          <div style={{ width: 48, height: 48, borderRadius: 14, background: SL5.accent, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontFamily: SL5.serif, fontSize: 22, fontWeight: 600 }}>A</div>
          <div style={{ fontFamily: SL5.serif, fontSize: 32, fontWeight: 600, letterSpacing: '-0.025em', color: SL5.text, marginTop: 36, lineHeight: 1.1 }}>Bon retour parmi nous.</div>
          <div style={{ fontFamily: SL5.serif, fontSize: 15, fontStyle: 'italic', color: SL5.textMute, marginTop: 8 }}>Connectez-vous pour retrouver vos missions et conversations.</div>
        </div>

        <div style={{ marginTop: 36, display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div>
            <label style={{ fontSize: 11.5, fontWeight: 600, color: SL5.text, display: 'block', marginBottom: 6 }}>E-mail</label>
            <div style={{ background: '#fff', border: `1.5px solid ${SL5.accent}`, borderRadius: 12, padding: '13px 14px' }}>
              <span style={{ fontSize: 14, color: SL5.text }}>camille.dubois@atelier.fr</span>
            </div>
          </div>
          <div>
            <label style={{ fontSize: 11.5, fontWeight: 600, color: SL5.text, display: 'block', marginBottom: 6 }}>Mot de passe</label>
            <div style={{ background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 12, padding: '13px 14px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span style={{ fontSize: 14, color: SL5.text, letterSpacing: 4 }}>••••••••</span>
              <SL5I name="Eye" size={16} />
            </div>
          </div>
          <div style={{ textAlign: 'right' }}>
            <span style={{ fontSize: 12, color: SL5.accent, fontWeight: 600, fontFamily: SL5.serif, fontStyle: 'italic' }}>Mot de passe oublié ?</span>
          </div>
        </div>

        <button style={{ marginTop: 24, padding: '14px', background: SL5.accent, color: '#fff', border: 'none', borderRadius: 14, fontSize: 14.5, fontWeight: 600, fontFamily: SL5.sans }}>Se connecter</button>

        {/* Séparateur */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, margin: '24px 0' }}>
          <div style={{ flex: 1, height: 1, background: SL5.border }} />
          <span style={{ fontSize: 11, color: SL5.textMute, fontFamily: SL5.serif, fontStyle: 'italic' }}>ou continuer avec</span>
          <div style={{ flex: 1, height: 1, background: SL5.border }} />
        </div>

        {/* SSO */}
        <div style={{ display: 'flex', gap: 8 }}>
          {['Google', 'Apple', 'LinkedIn'].map(p => (
            <button key={p} style={{ flex: 1, padding: '12px', background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 12, fontSize: 12.5, fontWeight: 600, color: SL5.text, fontFamily: SL5.sans }}>{p}</button>
          ))}
        </div>

        {/* Footer */}
        <div style={{ marginTop: 'auto', textAlign: 'center', fontSize: 12.5, color: SL5.textMute }}>
          Pas encore de compte ? <span style={{ color: SL5.accent, fontWeight: 700 }}>Créer un compte</span>
        </div>
      </div>
    </_AppFrame_L5>
  );
}

// ─── Signup choix de rôle ──────────────────────────────────
function AppSignupRole() {
  return (
    <_AppFrame_L5 bg="#fff">
      <div style={{ flexShrink: 0, padding: '6px 14px 8px', display: 'flex', alignItems: 'center', gap: 10 }}>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SL5.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL5I name="ArrowLeft" size={18} />
        </button>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 11, color: SL5.textMute }}>Création de compte</div>
          <div style={{ fontSize: 13.5, fontWeight: 600, color: SL5.text }}>Étape 1 sur 3</div>
        </div>
      </div>

      <div style={{ flex: 1, padding: '20px 28px 28px', display: 'flex', flexDirection: 'column' }}>
        <div style={{ fontFamily: SL5.serif, fontSize: 28, fontWeight: 600, letterSpacing: '-0.025em', color: SL5.text, lineHeight: 1.15 }}>Comment souhaitez-vous utiliser Atelier ?</div>
        <div style={{ fontFamily: SL5.serif, fontSize: 14, fontStyle: 'italic', color: SL5.textMute, marginTop: 8 }}>Vous pourrez ajouter le second rôle plus tard.</div>

        <div style={{ marginTop: 28, display: 'flex', flexDirection: 'column', gap: 12 }}>
          {/* Option freelance — sélectionnée */}
          <div style={{ background: SL5.accentSoft, border: `2px solid ${SL5.accent}`, borderRadius: 18, padding: 18, position: 'relative' }}>
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
              <div style={{ width: 46, height: 46, borderRadius: 14, background: SL5.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <SL5I name="User" size={20} />
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontFamily: SL5.serif, fontSize: 17, fontWeight: 600, color: SL5.text }}>Je suis prestataire</div>
                <div style={{ fontSize: 12.5, color: SL5.textMute, marginTop: 4, lineHeight: 1.45 }}>Trouvez des missions, créez votre profil et soyez payé en toute sécurité.</div>
              </div>
              <div style={{ width: 22, height: 22, borderRadius: '50%', background: SL5.accent, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <svg width="11" height="11" viewBox="0 0 12 12"><path d="M2 6l3 3 5-6" stroke="#fff" strokeWidth="2.2" fill="none" strokeLinecap="round" strokeLinejoin="round"/></svg>
              </div>
            </div>
          </div>

          {/* Option entreprise */}
          <div style={{ background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 18, padding: 18 }}>
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
              <div style={{ width: 46, height: 46, borderRadius: 14, background: SL5.bg, color: SL5.text, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <SL5I name="Building" size={20} />
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontFamily: SL5.serif, fontSize: 17, fontWeight: 600, color: SL5.text }}>Je suis une entreprise</div>
                <div style={{ fontSize: 12.5, color: SL5.textMute, marginTop: 4, lineHeight: 1.45 }}>Publiez vos annonces, recrutez des freelances vérifiés, gérez vos projets.</div>
              </div>
              <div style={{ width: 22, height: 22, borderRadius: '50%', border: `2px solid ${SL5.border}`, flexShrink: 0 }} />
            </div>
          </div>

          {/* Les deux */}
          <div style={{ background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 18, padding: 18 }}>
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
              <div style={{ width: 46, height: 46, borderRadius: 14, background: SL5.bg, color: SL5.text, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <SL5I name="Layers" size={20} />
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontFamily: SL5.serif, fontSize: 17, fontWeight: 600, color: SL5.text }}>Les deux <span style={{ fontSize: 11, color: SL5.accent, background: SL5.accentSoft, padding: '2px 7px', borderRadius: 999, fontWeight: 700, marginLeft: 4 }}>Recommandé</span></div>
                <div style={{ fontSize: 12.5, color: SL5.textMute, marginTop: 4, lineHeight: 1.45 }}>Basculez entre les deux rôles selon vos besoins.</div>
              </div>
              <div style={{ width: 22, height: 22, borderRadius: '50%', border: `2px solid ${SL5.border}`, flexShrink: 0 }} />
            </div>
          </div>
        </div>

        <button style={{ marginTop: 'auto', padding: '14px', background: SL5.accent, color: '#fff', border: 'none', borderRadius: 14, fontSize: 14.5, fontWeight: 600, fontFamily: SL5.sans }}>Continuer →</button>
      </div>
    </_AppFrame_L5>
  );
}

// ─── Profil / paramètres ──────────────────────────────────
function AppCompte() {
  return (
    <_AppFrame_L5>
      <div style={{ flexShrink: 0, padding: '6px 20px 16px', background: SL5.bg }}>
        <div style={{ fontFamily: SL5.serif, fontSize: 26, fontWeight: 600, letterSpacing: '-0.02em', color: SL5.text }}>Mon compte</div>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px', display: 'flex', flexDirection: 'column', gap: 14 }}>
        {/* Profil card */}
        <div style={{ background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 16, padding: 16, display: 'flex', alignItems: 'center', gap: 12 }}>
          <SL5Portrait id={0} size={56} rounded={16} />
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
              <div style={{ fontSize: 15, fontWeight: 600, color: SL5.text }}>Camille Dubois</div>
              <div style={{ width: 16, height: 16, borderRadius: '50%', background: SL5.green, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 9, fontWeight: 700 }}>✓</div>
            </div>
            <div style={{ fontSize: 12, color: SL5.textMute, marginTop: 2 }}>Product designer · Freelance</div>
            <div style={{ fontSize: 11, color: SL5.accent, fontFamily: SL5.serif, fontStyle: 'italic', marginTop: 3 }}>Voir mon profil public →</div>
          </div>
          <SL5I name="ChevronRight" size={16} />
        </div>

        {/* Switch role */}
        <div style={{ background: SL5.text, color: '#fff', borderRadius: 14, padding: '12px 14px', display: 'flex', alignItems: 'center', gap: 11 }}>
          <SL5I name="Refresh" size={17} />
          <div style={{ flex: 1, fontSize: 13, fontWeight: 600 }}>Basculer en mode entreprise</div>
          <SL5I name="ChevronRight" size={15} />
        </div>

        {/* Section Compte */}
        <div>
          <div style={{ fontSize: 11, color: SL5.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', marginBottom: 8, paddingLeft: 4 }}>Compte</div>
          <div style={{ background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 14, overflow: 'hidden' }}>
            {[
              { i: 'User', l: 'Profil public', r: 'Modifier' },
              { i: 'Briefcase', l: 'Compétences & expérience' },
              { i: 'Globe', l: 'Langues parlées', r: 'FR · EN' },
              { i: 'Tag', l: 'TJM & disponibilité', r: '600 €/j' },
            ].map((row, i, a) => (
              <SettingsRow key={i} row={row} last={i === a.length - 1} />
            ))}
          </div>
        </div>

        {/* Section Paiement */}
        <div>
          <div style={{ fontSize: 11, color: SL5.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', marginBottom: 8, paddingLeft: 4 }}>Paiement</div>
          <div style={{ background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 14, overflow: 'hidden' }}>
            {[
              { i: 'Card', l: 'Stripe Connect', r: 'Actif', good: true },
              { i: 'Receipt', l: 'Profil de facturation', r: 'SIRET · TVA' },
              { i: 'File', l: 'Mes factures' },
            ].map((row, i, a) => <SettingsRow key={i} row={row} last={i === a.length - 1} />)}
          </div>
        </div>

        {/* Section Préférences */}
        <div>
          <div style={{ fontSize: 11, color: SL5.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', marginBottom: 8, paddingLeft: 4 }}>Préférences</div>
          <div style={{ background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 14, overflow: 'hidden' }}>
            {[
              { i: 'Bell', l: 'Notifications push', toggle: true, on: true },
              { i: 'Lock', l: 'Authentification à deux facteurs', toggle: true, on: false },
              { i: 'Globe', l: 'Langue de l\'app', r: 'Français' },
            ].map((row, i, a) => <SettingsRow key={i} row={row} last={i === a.length - 1} />)}
          </div>
        </div>

        {/* Section Aide */}
        <div>
          <div style={{ fontSize: 11, color: SL5.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', marginBottom: 8, paddingLeft: 4 }}>Aide & légal</div>
          <div style={{ background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 14, overflow: 'hidden' }}>
            {[
              { i: 'Help', l: 'Centre d\'aide' },
              { i: 'Shield', l: 'Confidentialité' },
              { i: 'Doc', l: 'Conditions générales' },
            ].map((row, i, a) => <SettingsRow key={i} row={row} last={i === a.length - 1} />)}
          </div>
        </div>

        {/* Logout */}
        <button style={{ background: '#fff', border: `1px solid ${SL5.border}`, borderRadius: 14, padding: '14px', fontSize: 13.5, fontWeight: 600, color: SL5.accent, fontFamily: SL5.sans, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}>
          <SL5I name="LogOut" size={15} />
          Se déconnecter
        </button>

        <div style={{ textAlign: 'center', fontSize: 11, color: SL5.textSubtle, fontFamily: SL5.serif, fontStyle: 'italic', padding: '4px 0' }}>Atelier · v1.4.0</div>
      </div>

      <_AppTabBar_L5 active="profile" />
    </_AppFrame_L5>
  );
}

function SettingsRow({ row, last }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '13px 14px', borderBottom: last ? 'none' : `1px solid ${SL5.border}` }}>
      <div style={{ width: 32, height: 32, borderRadius: 10, background: SL5.bg, color: SL5.text, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
        <SL5I name={row.i} size={15} />
      </div>
      <div style={{ flex: 1, fontSize: 13.5, fontWeight: 500, color: SL5.text }}>{row.l}</div>
      {row.toggle ? (
        <div style={{ width: 36, height: 22, borderRadius: 999, background: row.on ? SL5.accent : SL5.border, position: 'relative', flexShrink: 0 }}>
          <div style={{ position: 'absolute', top: 2, [row.on ? 'right' : 'left']: 2, width: 18, height: 18, borderRadius: '50%', background: '#fff', boxShadow: '0 1px 2px rgba(0,0,0,0.15)' }} />
        </div>
      ) : (
        <>
          {row.r ? <span style={{ fontSize: 12, color: row.good ? SL5.green : SL5.textMute, fontWeight: row.good ? 600 : 400 }}>{row.r}</span> : null}
          <SL5I name="ChevronRight" size={14} />
        </>
      )}
    </div>
  );
}

Object.assign(window, { AppLogin, AppSignupRole, AppCompte });
