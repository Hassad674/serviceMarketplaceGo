// App mobile native — Flutter Material 3 unifié iOS/Android.
// 3 écrans Phase 1 : Recherche freelances, Profil prestataire, Messagerie.
// Adaptation de Soleil v2 aux conventions natives :
//  - Tab bar bas avec FAB central (Material 3)
//  - Headers compacts, large title style iOS
//  - Sheets bas pour les filtres (vs sidebar web)
//  - Pull-to-refresh indicator (statique, illustratif)
//  - Status bar Android (heure + batterie + signal en blanc/noir selon contexte)
//  - Hauteur 844 (iPhone 14/15 standard) — Flutter respecte les safe areas

const SA = window.S;
const SAI = window.SI;
const SAPortrait = window.Portrait;

// ─── Frame app native ─────────────────────────────────────────
function AppFrame({ children, dark = false, statusBarTone = 'dark', bg }) {
  // statusBarTone: 'dark' = icônes noires (fond clair) ; 'light' = icônes blanches (fond foncé)
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', justifyContent: 'center', background: '#d8cfbe', padding: 0, fontFamily: SA.sans }}>
      <div style={{ width: 390, height: 844, background: bg || SA.bg, display: 'flex', flexDirection: 'column', borderRadius: 0, boxShadow: '0 0 0 1px rgba(42,31,21,0.08)', overflow: 'hidden', position: 'relative' }}>
        {/* Status bar (système) */}
        <AppStatusBar tone={statusBarTone} />
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          {children}
        </div>
      </div>
    </div>
  );
}

function AppStatusBar({ tone = 'dark' }) {
  const c = tone === 'light' ? '#fff' : '#1a0f08';
  return (
    <div style={{ flexShrink: 0, height: 44, padding: '12px 24px 0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontFamily: '-apple-system, system-ui', fontSize: 14, fontWeight: 600, color: c, background: 'transparent', position: 'relative', zIndex: 5 }}>
      <span>9:41</span>
      <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
        {/* Signal */}
        <svg width="16" height="10" viewBox="0 0 16 10"><rect x="0" y="6" width="3" height="4" rx="0.5" fill={c}/><rect x="4" y="4" width="3" height="6" rx="0.5" fill={c}/><rect x="8" y="2" width="3" height="8" rx="0.5" fill={c}/><rect x="12" y="0" width="3" height="10" rx="0.5" fill={c}/></svg>
        {/* Wifi */}
        <svg width="14" height="10" viewBox="0 0 14 10"><path d="M7 3 a4 4 0 0 1 4 1.5l1-1A6 6 0 0 0 7 1 a6 6 0 0 0 -5 2.5l1 1A4 4 0 0 1 7 3z" fill={c}/><circle cx="7" cy="8" r="1.2" fill={c}/></svg>
        {/* Batterie */}
        <svg width="24" height="11" viewBox="0 0 24 11"><rect x="0.5" y="0.5" width="20" height="10" rx="2" fill="none" stroke={c} strokeOpacity="0.4"/><rect x="2" y="2" width="14" height="7" rx="1" fill={c}/><rect x="21" y="3.5" width="2" height="4" rx="0.5" fill={c} fillOpacity="0.5"/></svg>
      </div>
    </div>
  );
}

// ─── Tab bar bas avec FAB central (Material 3 + iOS unifié) ─────
function AppTabBar({ active = 'home' }) {
  const tabs = [
    { id: 'home', icon: 'Home', label: 'Accueil' },
    { id: 'search', icon: 'Search', label: 'Découvrir' },
    { id: 'msg', icon: 'Chat', label: 'Messages', badge: 3 },
    { id: 'me', icon: 'User', label: 'Profil' },
  ];
  return (
    <div style={{ flexShrink: 0, position: 'relative', background: '#fff', borderTop: `1px solid ${SA.border}`, padding: '8px 0 22px', display: 'flex', justifyContent: 'space-around', alignItems: 'flex-start' }}>
      {/* FAB central — déborde en haut */}
      <div style={{ position: 'absolute', top: -22, left: '50%', transform: 'translateX(-50%)', width: 52, height: 52, borderRadius: 18, background: SA.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 6px 16px rgba(232,93,74,0.35), 0 2px 4px rgba(232,93,74,0.2)', zIndex: 5 }}>
        <SAI name="Plus" size={22} />
      </div>

      {tabs.slice(0, 2).map(t => <TabItem key={t.id} t={t} active={active === t.id} />)}
      <div style={{ width: 52, flexShrink: 0 }} />
      {tabs.slice(2).map(t => <TabItem key={t.id} t={t} active={active === t.id} />)}
    </div>
  );
}

function TabItem({ t, active }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3, color: active ? SA.accent : SA.textMute, padding: '4px 12px', position: 'relative' }}>
      <div style={{ position: 'relative' }}>
        <SAI name={t.icon} size={22} />
        {t.badge ? <span style={{ position: 'absolute', top: -4, right: -8, background: SA.accent, color: '#fff', fontSize: 9.5, fontWeight: 700, padding: '1px 5px', borderRadius: 999, lineHeight: 1.2 }}>{t.badge}</span> : null}
      </div>
      <div style={{ fontSize: 10.5, fontWeight: active ? 600 : 500 }}>{t.label}</div>
    </div>
  );
}

// ─── Écran 1 : Recherche freelances ──────────────────────────────
function AppRecherche() {
  const freelances = [
    { id: 0, name: 'Camille Dubois', title: 'Product Designer', loc: 'Bordeaux', tjm: '600 €', rating: 4.9, reviews: 87, tags: ['Figma', 'Design system'], availability: 'Dispo 28 mai' },
    { id: 2, name: 'Yacine Benali', title: 'Développeur full-stack', loc: 'Lyon', tjm: '720 €', rating: 5.0, reviews: 124, tags: ['React', 'Node.js'], availability: 'Dispo immédiate', verified: true },
    { id: 1, name: 'Marion Lefèvre', title: 'Brand designer', loc: 'Paris', tjm: '550 €', rating: 4.8, reviews: 63, tags: ['Identité', 'Direction artistique'], availability: 'Dispo dans 2 sem.' },
    { id: 3, name: 'Théo Martinet', title: 'Motion designer', loc: 'Nantes', tjm: '480 €', rating: 4.7, reviews: 41, tags: ['After Effects', 'Cinema 4D'], availability: 'Complet' },
    { id: 4, name: 'Sofia Lambert', title: 'Designer UX/UI', loc: 'Toulouse', tjm: '520 €', rating: 4.9, reviews: 92, tags: ['Recherche', 'Prototypage'], availability: 'Dispo 1 juin' },
  ];
  return (
    <AppFrame>
      {/* Header */}
      <div style={{ flexShrink: 0, padding: '6px 20px 18px', background: SA.bg }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
          <div>
            <div style={{ fontFamily: SA.serif, fontSize: 28, fontWeight: 600, letterSpacing: '-0.02em', color: SA.text }}>Découvrir</div>
            <div style={{ fontSize: 12.5, color: SA.textMute, fontFamily: SA.serif, fontStyle: 'italic', marginTop: 2 }}>328 prestataires correspondants</div>
          </div>
          <div style={{ width: 38, height: 38, borderRadius: '50%', background: '#fff', border: `1px solid ${SA.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', position: 'relative' }}>
            <SAI name="Bell" size={18} />
            <span style={{ position: 'absolute', top: 8, right: 9, width: 8, height: 8, borderRadius: '50%', background: SA.accent, border: '2px solid #fff' }} />
          </div>
        </div>

        {/* Barre de recherche */}
        <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 14, padding: '11px 14px', display: 'flex', alignItems: 'center', gap: 10, boxShadow: '0 1px 2px rgba(42,31,21,0.03)' }}>
          <SAI name="Search" size={17} />
          <input placeholder="Métier, compétence, ville…" style={{ flex: 1, border: 'none', outline: 'none', fontSize: 14, fontFamily: SA.sans, background: 'transparent' }} />
          <div style={{ width: 1, height: 18, background: SA.border }} />
          <SAI name="Sliders" size={17} />
        </div>

        {/* Chips filtres */}
        <div style={{ display: 'flex', gap: 7, marginTop: 12, overflowX: 'auto', paddingBottom: 2 }}>
          {[
            { label: 'Design', active: true },
            { label: 'Développement' },
            { label: 'Marketing' },
            { label: 'Stratégie' },
            { label: 'Vidéo' },
          ].map(c => (
            <span key={c.label} style={{ padding: '6px 12px', borderRadius: 999, fontSize: 12, fontWeight: 600, whiteSpace: 'nowrap', background: c.active ? SA.text : '#fff', color: c.active ? '#fff' : SA.text, border: c.active ? 'none' : `1px solid ${SA.border}` }}>{c.label}</span>
          ))}
        </div>
      </div>

      {/* Liste — pull to refresh indicator */}
      <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px' }}>
        <div style={{ textAlign: 'center', padding: '4px 0 12px', fontSize: 11, color: SA.textSubtle, fontFamily: SA.serif, fontStyle: 'italic' }}>↓ Tirer pour rafraîchir</div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {freelances.map(f => (
            <div key={f.id} style={{ background: '#fff', borderRadius: 16, padding: 14, border: `1px solid ${SA.border}`, display: 'flex', gap: 12 }}>
              <div style={{ position: 'relative', flexShrink: 0 }}>
                <SAPortrait id={f.id} size={56} rounded={14} />
                {f.verified ? <div style={{ position: 'absolute', bottom: -2, right: -2, width: 18, height: 18, borderRadius: '50%', background: SA.green, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', border: '2px solid #fff' }}>
                  <svg width="9" height="9" viewBox="0 0 12 12"><path d="M2 6l3 3 5-6" stroke="#fff" strokeWidth="2.2" fill="none" strokeLinecap="round" strokeLinejoin="round"/></svg>
                </div> : null}
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 6 }}>
                  <div style={{ minWidth: 0, flex: 1 }}>
                    <div style={{ fontSize: 14.5, fontWeight: 600, color: SA.text, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{f.name}</div>
                    <div style={{ fontSize: 12, color: SA.textMute, marginTop: 1 }}>{f.title}</div>
                  </div>
                  <div style={{ textAlign: 'right', flexShrink: 0 }}>
                    <div style={{ fontFamily: SA.mono, fontSize: 13, fontWeight: 600, color: SA.text }}>{f.tjm}</div>
                    <div style={{ fontSize: 9.5, color: SA.textSubtle, letterSpacing: '0.04em' }}>/JOUR</div>
                  </div>
                </div>

                <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginTop: 6, fontSize: 11.5, color: SA.textMute }}>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 3 }}>
                    <svg width="11" height="11" viewBox="0 0 12 12" fill={SA.accent}><path d="M6 1l1.5 3.4L11 5l-2.5 2.4.6 3.4L6 9.2l-3.1 1.6.6-3.4L1 5l3.5-.6z"/></svg>
                    <span style={{ fontWeight: 600, color: SA.text }}>{f.rating}</span>
                    <span>({f.reviews})</span>
                  </span>
                  <span style={{ width: 2, height: 2, borderRadius: '50%', background: SA.textSubtle }} />
                  <span>{f.loc}</span>
                </div>

                <div style={{ display: 'flex', gap: 5, marginTop: 8, flexWrap: 'wrap' }}>
                  {f.tags.map(t => (
                    <span key={t} style={{ padding: '3px 8px', background: SA.bg, borderRadius: 6, fontSize: 10.5, color: SA.textMute, fontWeight: 500 }}>{t}</span>
                  ))}
                </div>

                <div style={{ marginTop: 9, paddingTop: 9, borderTop: `1px dashed ${SA.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <span style={{ fontSize: 11.5, color: f.availability === 'Complet' ? SA.textSubtle : SA.green, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 5 }}>
                    <span style={{ width: 6, height: 6, borderRadius: '50%', background: f.availability === 'Complet' ? SA.textSubtle : SA.green }} />
                    {f.availability}
                  </span>
                  <span style={{ fontSize: 12, fontWeight: 600, color: SA.accent }}>Voir le profil →</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      <AppTabBar active="search" />
    </AppFrame>
  );
}

// ─── Écran 2 : Profil prestataire ────────────────────────────────
function AppProfil() {
  return (
    <AppFrame statusBarTone="light" bg="#fff">
      {/* Hero photo (cover) */}
      <div style={{ flexShrink: 0, position: 'relative', height: 220, background: `linear-gradient(140deg, #c43a26, #e85d4a 50%, #d4924a)`, marginTop: -44 }}>
        {/* Overlay status bar zone */}
        <div style={{ position: 'absolute', top: 0, left: 0, right: 0, height: 44 }} />
        {/* Boutons header */}
        <div style={{ position: 'absolute', top: 52, left: 16, right: 16, display: 'flex', justifyContent: 'space-between' }}>
          <button style={{ width: 36, height: 36, borderRadius: '50%', background: 'rgba(255,255,255,0.25)', backdropFilter: 'blur(10px)', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff' }}>
            <SAI name="ArrowLeft" size={18} />
          </button>
          <div style={{ display: 'flex', gap: 8 }}>
            <button style={{ width: 36, height: 36, borderRadius: '50%', background: 'rgba(255,255,255,0.25)', backdropFilter: 'blur(10px)', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff' }}>
              <SAI name="Share" size={16} />
            </button>
            <button style={{ width: 36, height: 36, borderRadius: '50%', background: 'rgba(255,255,255,0.25)', backdropFilter: 'blur(10px)', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff' }}>
              <SAI name="Save" size={16} />
            </button>
          </div>
        </div>
        {/* Citation flottante */}
        <div style={{ position: 'absolute', bottom: 14, left: 20, right: 20, fontFamily: SA.serif, fontStyle: 'italic', fontSize: 14, color: 'rgba(255,255,255,0.92)', lineHeight: 1.4 }}>
          « Le design qui dure, c'est celui qui sait se faire oublier. »
        </div>
      </div>

      {/* Carte profil — sur fond blanc, pas sur le cover */}
      <div style={{ flex: 1, overflow: 'auto', background: '#fff' }}>
        <div style={{ padding: '20px 20px 0', display: 'flex', gap: 14, alignItems: 'flex-start' }}>
          <div style={{ marginTop: -50, position: 'relative' }}>
            <div style={{ width: 86, height: 86, borderRadius: 22, background: '#fff', padding: 4, boxShadow: '0 6px 16px rgba(42,31,21,0.12)' }}>
              <SAPortrait id={0} size={78} rounded={18} />
            </div>
            <div style={{ position: 'absolute', bottom: -2, right: -2, width: 22, height: 22, borderRadius: '50%', background: SA.green, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', border: '3px solid #fff' }}>
              <svg width="11" height="11" viewBox="0 0 12 12"><path d="M2 6l3 3 5-6" stroke="#fff" strokeWidth="2.2" fill="none" strokeLinecap="round" strokeLinejoin="round"/></svg>
            </div>
          </div>
          <div style={{ flex: 1, paddingTop: 6 }}>
            <div style={{ fontFamily: SA.serif, fontSize: 20, fontWeight: 600, letterSpacing: '-0.01em', color: SA.text, lineHeight: 1.15 }}>Camille Dubois</div>
            <div style={{ fontSize: 13, color: SA.textMute, marginTop: 2 }}>Product Designer · Bordeaux</div>
          </div>
        </div>

        {/* Stats row */}
        <div style={{ padding: '16px 20px 0', display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8 }}>
          {[
            { v: '4.9', l: '87 avis', icon: '★' },
            { v: '7 ans', l: 'expérience' },
            { v: '94 %', l: 'recommandent' },
          ].map((s, i) => (
            <div key={i} style={{ background: SA.bg, borderRadius: 12, padding: '10px 8px', textAlign: 'center' }}>
              <div style={{ fontFamily: SA.serif, fontSize: 16, fontWeight: 600, color: SA.text }}>
                {s.icon ? <span style={{ color: SA.accent, marginRight: 2 }}>{s.icon}</span> : null}{s.v}
              </div>
              <div style={{ fontSize: 10.5, color: SA.textMute, marginTop: 1 }}>{s.l}</div>
            </div>
          ))}
        </div>

        {/* TJM + dispo bandeau */}
        <div style={{ margin: '14px 20px 0', background: SA.accentSoft, border: `1px solid ${SA.accent}33`, borderRadius: 14, padding: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <div style={{ fontSize: 11, color: SA.accentDeep, letterSpacing: '0.04em', fontWeight: 700, textTransform: 'uppercase' }}>Tarif journalier</div>
            <div style={{ fontFamily: SA.serif, fontSize: 22, fontWeight: 600, color: SA.text, marginTop: 2 }}>600 € <span style={{ fontSize: 12, color: SA.textMute, fontWeight: 400 }}>/jour</span></div>
          </div>
          <div style={{ textAlign: 'right' }}>
            <div style={{ fontSize: 11, color: SA.green, letterSpacing: '0.04em', fontWeight: 700, textTransform: 'uppercase' }}>Disponible</div>
            <div style={{ fontSize: 12.5, color: SA.text, marginTop: 2, fontWeight: 600 }}>à partir du 28 mai</div>
          </div>
        </div>

        {/* Tabs */}
        <div style={{ padding: '20px 20px 0', display: 'flex', gap: 18, borderBottom: `1px solid ${SA.border}`, marginTop: 4 }}>
          {['À propos', 'Travaux', 'Avis'].map((t, i) => (
            <div key={t} style={{ paddingBottom: 10, fontSize: 13, fontWeight: 600, color: i === 0 ? SA.text : SA.textMute, borderBottom: i === 0 ? `2px solid ${SA.accent}` : '2px solid transparent', marginBottom: -1 }}>{t}</div>
          ))}
        </div>

        {/* Contenu À propos */}
        <div style={{ padding: '16px 20px 20px' }}>
          <p style={{ fontSize: 13.5, lineHeight: 1.55, color: SA.text, margin: 0 }}>
            Designer produit indépendante depuis 7 ans, j'accompagne les startups B2B SaaS dans la conception d'interfaces utiles, durables, et profondément humaines.
          </p>

          <div style={{ marginTop: 18, fontSize: 11, color: SA.textSubtle, letterSpacing: '0.06em', fontWeight: 600, textTransform: 'uppercase', marginBottom: 10 }}>Compétences</div>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {['Figma', 'Design system', 'Recherche utilisateur', 'Prototypage', 'B2B SaaS', 'Accessibilité'].map(t => (
              <span key={t} style={{ padding: '6px 11px', background: SA.bg, borderRadius: 999, fontSize: 12, color: SA.text, fontWeight: 500 }}>{t}</span>
            ))}
          </div>

          <div style={{ marginTop: 22, fontSize: 11, color: SA.textSubtle, letterSpacing: '0.06em', fontWeight: 600, textTransform: 'uppercase', marginBottom: 10 }}>Ils l'ont recommandée</div>
          <div style={{ background: SA.bg, borderRadius: 14, padding: 14 }}>
            <p style={{ fontFamily: SA.serif, fontStyle: 'italic', fontSize: 14, lineHeight: 1.5, margin: 0, color: SA.text }}>« Camille a transformé notre app. Le design system qu'elle a posé tient depuis 2 ans sans douleur. »</p>
            <div style={{ display: 'flex', alignItems: 'center', gap: 9, marginTop: 12 }}>
              <SAPortrait id={2} size={32} />
              <div>
                <div style={{ fontSize: 12, fontWeight: 600, color: SA.text }}>Léa Bertrand</div>
                <div style={{ fontSize: 11, color: SA.textMute }}>Head of Product, Helio</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* CTA bas — sticky */}
      <div style={{ flexShrink: 0, padding: '12px 20px 28px', background: '#fff', borderTop: `1px solid ${SA.border}`, display: 'flex', gap: 10 }}>
        <button style={{ width: 50, height: 50, borderRadius: 14, background: SA.bg, border: `1px solid ${SA.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: SA.text }}>
          <SAI name="Chat" size={20} />
        </button>
        <button style={{ flex: 1, height: 50, borderRadius: 14, background: SA.accent, color: '#fff', border: 'none', fontSize: 14.5, fontWeight: 600, fontFamily: SA.sans, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}>
          Faire une demande
          <span style={{ fontSize: 16 }}>→</span>
        </button>
      </div>
    </AppFrame>
  );
}

// ─── Écran 3 : Messagerie (chat ouvert) ──────────────────────────
function AppMessagerie() {
  const messages = [
    { from: 'them', text: 'Bonjour Camille, j\'ai bien reçu votre proposition. Le scope colle bien à notre besoin.', time: '10:14' },
    { from: 'them', text: 'Une question : est-ce que vous incluez les ateliers avec nos équipes produit, ou c\'est en option ?', time: '10:14' },
    { from: 'me', text: 'Bonjour Léa, merci pour votre retour. Les 2 ateliers (kickoff + restitution) sont inclus dans la proposition.', time: '10:32' },
    { from: 'me', text: 'Si vous souhaitez des ateliers thématiques en plus (recherche utilisateur, design system), je peux préparer un avenant.', time: '10:32' },
    { from: 'them', text: 'Parfait, c\'est clair. On valide de notre côté, je vous envoie le bon de commande dans la journée.', time: '11:08' },
    { from: 'system', text: 'Léa a partagé un fichier · brief-helio-v2.pdf · 1,4 Mo', time: '11:09' },
    { from: 'them', text: 'Voilà le brief mis à jour avec les contraintes RGPD. Hâte de démarrer !', time: '11:10' },
  ];
  return (
    <AppFrame>
      {/* Header chat */}
      <div style={{ flexShrink: 0, padding: '6px 14px 12px', background: '#fff', borderBottom: `1px solid ${SA.border}`, display: 'flex', alignItems: 'center', gap: 10 }}>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: 'transparent', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SAI name="ArrowLeft" size={20} />
        </button>
        <div style={{ position: 'relative' }}>
          <SAPortrait id={2} size={38} />
          <div style={{ position: 'absolute', bottom: 0, right: 0, width: 11, height: 11, borderRadius: '50%', background: SA.green, border: '2px solid #fff' }} />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: SA.text }}>Léa Bertrand</div>
          <div style={{ fontSize: 11, color: SA.green, fontWeight: 500 }}>En ligne · répond rapidement</div>
        </div>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SA.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SAI name="Phone" size={17} />
        </button>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SA.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SAI name="MoreH" size={17} />
        </button>
      </div>

      {/* Bandeau projet contexte */}
      <div style={{ flexShrink: 0, padding: '10px 14px', background: SA.accentSoft, borderBottom: `1px solid ${SA.accent}22`, display: 'flex', alignItems: 'center', gap: 10 }}>
        <SAI name="Folder" size={15} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: SA.accentDeep, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>Refonte app Helio · Mission en cours</div>
          <div style={{ fontSize: 10.5, color: SA.accentDeep, opacity: 0.7, marginTop: 1 }}>Jalon 2 sur 4 · livraison prévue 12 juin</div>
        </div>
        <SAI name="ChevronRight" size={14} />
      </div>

      {/* Messages */}
      <div style={{ flex: 1, overflow: 'auto', padding: '14px 14px 8px', display: 'flex', flexDirection: 'column', gap: 6 }}>
        <div style={{ textAlign: 'center', fontSize: 11, color: SA.textSubtle, fontFamily: SA.serif, fontStyle: 'italic', padding: '6px 0 12px' }}>Aujourd'hui · 22 mai</div>

        {messages.map((m, i) => {
          if (m.from === 'system') {
            return (
              <div key={i} style={{ alignSelf: 'center', background: SA.bg, border: `1px dashed ${SA.border}`, borderRadius: 10, padding: '8px 12px', fontSize: 11.5, color: SA.textMute, display: 'flex', alignItems: 'center', gap: 8, maxWidth: '85%' }}>
                <div style={{ width: 24, height: 28, background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 4, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  <span style={{ fontSize: 8, fontWeight: 700, color: SA.accent }}>PDF</span>
                </div>
                <div>{m.text}</div>
              </div>
            );
          }
          const me = m.from === 'me';
          return (
            <div key={i} style={{ alignSelf: me ? 'flex-end' : 'flex-start', maxWidth: '78%', display: 'flex', flexDirection: 'column', alignItems: me ? 'flex-end' : 'flex-start', gap: 2 }}>
              <div style={{ background: me ? SA.text : '#fff', color: me ? '#fff' : SA.text, padding: '9px 13px', borderRadius: 16, borderTopRightRadius: me ? 4 : 16, borderTopLeftRadius: me ? 16 : 4, fontSize: 13.5, lineHeight: 1.4, border: me ? 'none' : `1px solid ${SA.border}` }}>
                {m.text}
              </div>
              <div style={{ fontSize: 10, color: SA.textSubtle, padding: '0 6px' }}>{m.time} {me ? '· lu' : ''}</div>
            </div>
          );
        })}

        {/* Indicateur typing */}
        <div style={{ alignSelf: 'flex-start', background: '#fff', border: `1px solid ${SA.border}`, padding: '12px 14px', borderRadius: 16, borderTopLeftRadius: 4, display: 'flex', gap: 4, marginTop: 4 }}>
          {[0, 1, 2].map(i => <span key={i} style={{ width: 6, height: 6, borderRadius: '50%', background: SA.textSubtle }} />)}
        </div>
      </div>

      {/* Input */}
      <div style={{ flexShrink: 0, padding: '10px 14px 24px', background: '#fff', borderTop: `1px solid ${SA.border}`, display: 'flex', alignItems: 'center', gap: 8 }}>
        <button style={{ width: 38, height: 38, borderRadius: '50%', background: SA.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SAI name="Plus" size={18} />
        </button>
        <div style={{ flex: 1, background: SA.bg, borderRadius: 22, padding: '9px 14px', display: 'flex', alignItems: 'center', gap: 8 }}>
          <input placeholder="Message…" style={{ flex: 1, border: 'none', background: 'transparent', outline: 'none', fontSize: 14, fontFamily: SA.sans }} />
          <SAI name="Smile" size={17} />
        </div>
        <button style={{ width: 40, height: 40, borderRadius: '50%', background: SA.accent, color: '#fff', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SAI name="Send" size={17} />
        </button>
      </div>
    </AppFrame>
  );
}

Object.assign(window, {
  AppRecherche, AppProfil, AppMessagerie, AppFrame, AppTabBar,
});
