// Direction 2 v2 — Atelier Soleil raffinée. Améliorations : système photos réalistes, dashboard éditorial, cards humaines avec citations, icônes plus rondes, polish FR (dates en français), micro-typographie.
const { Icons: I_S } = window;

const S = {
  bg: '#fffbf5',
  surface: '#ffffff',
  border: '#f0e6d8',
  borderStrong: '#e0d3bc',
  text: '#2a1f15',
  textMute: '#7a6850',
  textSubtle: '#a89679',
  accent: '#e85d4a',
  accentSoft: '#fde9e3',
  accentDeep: '#c43a26',
  pink: '#f08aa8',
  pinkSoft: '#fde6ed',
  green: '#5a9670',
  greenSoft: '#e8f2eb',
  amber: '#d4924a',
  serif: 'Fraunces, Georgia, serif',
  sans: '"Inter Tight", system-ui, sans-serif',
  mono: '"Geist Mono", monospace',
};

// ─── Système photos : portraits stylisés via SVG (pas d'initiales) ──
// Chaque "photo" = un fond chaud + une silhouette portrait abstraite. Plus humain qu'une initiale, plus pro qu'un emoji, et cohérent avec la palette.
function Portrait({ id = 0, size = 48, rounded = '50%' }) {
  // 6 variations déterministes par id
  const palettes = [
    { bg: '#fde9e3', skin: '#e8a890', hair: '#3d2618', shirt: '#c43a26' },     // 0 — corail
    { bg: '#e8f2eb', skin: '#d4a584', hair: '#5a3a1f', shirt: '#5a9670' },     // 1 — vert olive
    { bg: '#fde6ed', skin: '#d49a82', hair: '#1a1a1a', shirt: '#c84d72' },     // 2 — rose
    { bg: '#fbf0dc', skin: '#c4926e', hair: '#8b4a1f', shirt: '#d4924a' },     // 3 — ambre
    { bg: '#e8e4f4', skin: '#d8a890', hair: '#2a1f3a', shirt: '#6b5b9a' },     // 4 — lilas
    { bg: '#dfecef', skin: '#c89478', hair: '#3d2818', shirt: '#3a6b7a' },     // 5 — bleu
  ];
  const p = palettes[id % 6];
  return (
    <div style={{ width: size, height: size, borderRadius: rounded, background: p.bg, position: 'relative', overflow: 'hidden', flexShrink: 0 }}>
      <svg viewBox="0 0 60 60" width={size} height={size} style={{ display: 'block' }}>
        {/* Cou */}
        <rect x="24" y="38" width="12" height="10" fill={p.skin} />
        {/* Épaules / haut */}
        <path d={`M8 60 Q8 46 30 44 Q52 46 52 60 Z`} fill={p.shirt} />
        {/* Tête */}
        <ellipse cx="30" cy="28" rx="11" ry="13" fill={p.skin} />
        {/* Cheveux */}
        <path d={`M19 24 Q19 13 30 13 Q41 13 41 24 Q41 21 36 19 Q30 17 24 19 Q19 21 19 28 Z`} fill={p.hair} />
      </svg>
    </div>
  );
}

// ─── Icônes plus rondes / chaleureuses (override stroke 2.2 + scale) ──
const SI = ({ name, size = 18 }) => {
  const C = I_S[name];
  return C ? <C size={size} stroke={2} /> : null;
};

// ─── Dates en français ─────────────────────────────────────────────
const FR = {
  today: "Aujourd'hui",
  thisWeek: 'Cette semaine',
  morning: (d) => `${d} matin`,
};

function SSidebar({ active, role = 'enterprise' }) {
  const items = role === 'enterprise' ? [
    { id: 'home', icon: 'Home', label: 'Accueil' },
    { id: 'msg', icon: 'Chat', label: 'Messages', badge: 3 },
    { id: 'proj', icon: 'Folder', label: 'Projets' },
    { id: 'jobs', icon: 'Briefcase', label: 'Annonces' },
    { id: 'team', icon: 'Users', label: 'Équipe' },
  ] : [
    { id: 'home', icon: 'Home', label: 'Accueil' },
    { id: 'msg', icon: 'Chat', label: 'Messages', badge: 2 },
    { id: 'proj', icon: 'Folder', label: 'Mes missions' },
    { id: 'opp', icon: 'Inbox', label: 'Opportunités' },
    { id: 'profile', icon: 'User', label: 'Mon profil' },
  ];
  const find = [
    { id: 'find-f', icon: 'Search', label: 'Freelances', match: 'find' },
    { id: 'find-a', icon: 'Layers', label: 'Agences' },
    { id: 'find-r', icon: 'Sparkle', label: 'Apporteurs' },
  ];
  return (
    <aside style={{ width: 256, height: '100%', background: '#fff', borderRight: `1px solid ${S.border}`, padding: '20px 0', display: 'flex', flexDirection: 'column', fontFamily: S.sans, flexShrink: 0 }}>
      <div style={{ padding: '0 20px 18px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 9 }}>
          <div style={{ width: 30, height: 30, borderRadius: 999, background: S.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: S.serif, fontSize: 18, fontWeight: 600, fontStyle: 'italic' }}>a</div>
          <div style={{ fontFamily: S.serif, fontSize: 22, fontWeight: 500, color: S.text, letterSpacing: '-0.02em' }}>atelier</div>
        </div>
      </div>

      <div style={{ padding: '0 12px 16px' }}>
        <div style={{ background: S.bg, borderRadius: 14, padding: '10px 12px', display: 'flex', alignItems: 'center', gap: 10 }}>
          <Portrait id={2} size={36} />
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>Nova Studio</div>
            <div style={{ fontSize: 11, color: S.textMute }}>{role === 'enterprise' ? 'Entreprise' : 'Prestataire'}</div>
          </div>
        </div>
      </div>

      <nav style={{ padding: '0 8px', flex: 1, display: 'flex', flexDirection: 'column', gap: 1 }}>
        {items.map((it) => {
          const isActive = active === it.id || active === it.match;
          return (
            <div key={it.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 12px', borderRadius: 10, fontSize: 14, fontWeight: 500, color: isActive ? S.accent : S.text, background: isActive ? S.accentSoft : 'transparent', cursor: 'pointer' }}>
              <SI name={it.icon} size={18} />
              <span style={{ flex: 1 }}>{it.label}</span>
              {it.badge && <span style={{ background: S.accent, color: '#fff', fontSize: 10, fontWeight: 700, padding: '1px 7px', borderRadius: 999 }}>{it.badge}</span>}
            </div>
          );
        })}
        <div style={{ height: 1, background: S.border, margin: '12px 12px' }} />
        <div style={{ fontSize: 11, color: S.textSubtle, padding: '4px 12px', fontWeight: 600, letterSpacing: '0.04em' }}>Découvrir</div>
        {find.map((it) => {
          const isActive = active === it.id || active === it.match;
          return (
            <div key={it.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 12px', borderRadius: 10, fontSize: 14, fontWeight: 500, color: isActive ? S.accent : S.text, background: isActive ? S.accentSoft : 'transparent', cursor: 'pointer' }}>
              <SI name={it.icon} size={18} /> <span>{it.label}</span>
            </div>
          );
        })}
      </nav>

      <div style={{ padding: '12px 20px', borderTop: `1px solid ${S.border}` }}>
        <div style={{ background: 'linear-gradient(135deg, #fde9e3, #fde6ed)', padding: 14, borderRadius: 14, position: 'relative', overflow: 'hidden' }}>
          <div style={{ fontSize: 12, fontWeight: 700, color: S.accentDeep, marginBottom: 4, fontFamily: S.serif, fontStyle: 'italic' }}>Atelier Premium</div>
          <div style={{ fontSize: 11, color: S.textMute, marginBottom: 10, lineHeight: 1.4 }}>+50 propositions / mois</div>
          <button style={{ background: S.text, color: '#fff', border: 'none', padding: '6px 12px', fontSize: 11, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Découvrir →</button>
        </div>
      </div>
    </aside>
  );
}

function STopbar() {
  return (
    <div style={{ height: 64, borderBottom: `1px solid ${S.border}`, background: '#fff', display: 'flex', alignItems: 'center', padding: '0 28px', gap: 16, flexShrink: 0 }}>
      <div style={{ flex: 1, maxWidth: 480, display: 'flex', alignItems: 'center', gap: 10, background: S.bg, border: `1px solid ${S.border}`, borderRadius: 999, padding: '10px 18px' }}>
        <SI name="Search" size={15} />
        <input placeholder="Que cherches-tu aujourd'hui ?" style={{ flex: 1, border: 'none', outline: 'none', background: 'transparent', fontSize: 13, color: S.text }} readOnly />
      </div>
      <div style={{ flex: 1 }} />
      <button style={{ background: S.text, border: 'none', padding: '9px 16px', fontSize: 13, fontWeight: 600, color: '#fff', borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
        <SI name="Plus" size={14} /> Publier une annonce
      </button>
      <SI name="Bell" size={20} />
      <Portrait id={2} size={36} />
    </div>
  );
}

// ═══ DASHBOARD v2 — éditorial fort ════════════════════════════════
function SoleilDashboard() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: S.bg, fontFamily: S.sans, color: S.text }}>
      <SSidebar active="home" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <STopbar />
        <div style={{ flex: 1, overflow: 'hidden', padding: '32px 36px' }}>
          {/* Header avec date FR */}
          <div style={{ marginBottom: 22, display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between' }}>
            <div>
              <div style={{ fontSize: 12, color: S.textMute, marginBottom: 4, fontFamily: S.mono, letterSpacing: '0.04em' }}>Mardi matin · 14 mai</div>
              <h1 style={{ fontFamily: S.serif, fontSize: 44, lineHeight: 1.05, margin: 0, fontWeight: 400, letterSpacing: '-0.025em' }}>Bonjour Nova, <span style={{ fontStyle: 'italic', color: S.accent }}>belle journée</span> en perspective.</h1>
              <p style={{ fontSize: 15, color: S.textMute, margin: '8px 0 0', maxWidth: 540 }}>Trois bonnes nouvelles t'attendent : 8 nouvelles candidatures sur ton annonce produit, un paiement à valider, et Élise t'a relancée.</p>
            </div>
          </div>

          {/* Section éditoriale "Cette semaine chez Atelier" */}
          <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 20, padding: 0, marginBottom: 22, overflow: 'hidden', display: 'grid', gridTemplateColumns: '1.2fr 1fr' }}>
            <div style={{ padding: '28px 32px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
              <div>
                <div style={{ fontSize: 11, fontWeight: 700, color: S.accent, marginBottom: 14, letterSpacing: '0.12em', textTransform: 'uppercase', display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ width: 24, height: 1, background: S.accent }} /> Cette semaine chez Atelier
                </div>
                <h2 style={{ fontFamily: S.serif, fontSize: 30, lineHeight: 1.15, margin: 0, fontWeight: 400, letterSpacing: '-0.02em', marginBottom: 12 }}>« J'ai trouvé en 3 jours le freelance qui nous accompagne depuis 18 mois. »</h2>
                <p style={{ fontSize: 13.5, color: S.textMute, margin: 0, lineHeight: 1.6, marginBottom: 18 }}>Comment Pauline, fondatrice de Lemon Aviation, a recruté son équipe design via Atelier — et pourquoi elle ne reviendra pas en arrière.</p>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <Portrait id={4} size={36} />
                <div>
                  <div style={{ fontSize: 12.5, fontWeight: 600 }}>Pauline Roussel</div>
                  <div style={{ fontSize: 11, color: S.textMute }}>Fondatrice · Lemon Aviation</div>
                </div>
                <button style={{ marginLeft: 'auto', background: 'transparent', border: `1px solid ${S.borderStrong}`, padding: '8px 14px', fontSize: 12, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Lire l'histoire →</button>
              </div>
            </div>
            <div style={{ background: 'linear-gradient(135deg, #fde9e3, #fde6ed, #fbf0dc)', position: 'relative', overflow: 'hidden', minHeight: 260 }}>
              <div style={{ position: 'absolute', top: -40, right: -40, width: 220, height: 220, borderRadius: '50%', background: 'radial-gradient(circle, rgba(232,93,74,0.3), transparent 65%)' }} />
              <div style={{ position: 'absolute', bottom: -60, left: 40, width: 180, height: 180, borderRadius: '50%', background: 'radial-gradient(circle, rgba(240,138,168,0.4), transparent 65%)' }} />
              {/* Trio portraits flottants */}
              <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', display: 'flex' }}>
                <div style={{ marginRight: -16, transform: 'rotate(-6deg)', boxShadow: '0 8px 24px rgba(0,0,0,0.12)', borderRadius: '50%' }}><Portrait id={1} size={88} /></div>
                <div style={{ zIndex: 2, transform: 'translateY(-12px)', boxShadow: '0 12px 28px rgba(0,0,0,0.15)', borderRadius: '50%' }}><Portrait id={4} size={108} /></div>
                <div style={{ marginLeft: -16, transform: 'rotate(6deg)', boxShadow: '0 8px 24px rgba(0,0,0,0.12)', borderRadius: '50%' }}><Portrait id={3} size={88} /></div>
              </div>
            </div>
          </div>

          {/* Two column rows */}
          <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr', gap: 18 }}>
            {/* Missions */}
            <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 16, padding: 24 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <div>
                  <h3 style={{ fontFamily: S.serif, fontSize: 22, margin: 0, fontWeight: 500, letterSpacing: '-0.01em' }}>Tes missions du moment</h3>
                  <p style={{ fontSize: 12, color: S.textMute, margin: '2px 0 0' }}>3 actives · prochaine livraison vendredi</p>
                </div>
                <a style={{ fontSize: 12, color: S.accent, fontWeight: 600, cursor: 'pointer' }}>Tout voir →</a>
              </div>
              {[
                { name: 'Refonte site corporate', client: 'Lemon Aviation', amount: '12 400 €', progress: 72, dl: 'livraison vendredi', pid: 1 },
                { name: 'Brand identity Q2', client: 'Cobalt Studio', amount: '8 200 €', progress: 45, dl: 'dans 1 mois', pid: 0 },
                { name: 'Audit SEO technique', client: 'Maison Vega', amount: '3 600 €', progress: 90, dl: 'dans 1 semaine', pid: 5 },
              ].map((m, i) => (
                <div key={i} style={{ padding: '16px 0', borderTop: i > 0 ? `1px solid ${S.border}` : 'none', display: 'grid', gridTemplateColumns: '44px 1fr auto', gap: 14, alignItems: 'center' }}>
                  <Portrait id={m.pid} size={44} rounded={12} />
                  <div>
                    <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 3 }}>{m.name}</div>
                    <div style={{ fontSize: 12, color: S.textMute, marginBottom: 8 }}>avec {m.client} · {m.dl}</div>
                    <div style={{ width: 220, height: 5, background: S.border, borderRadius: 3, overflow: 'hidden' }}>
                      <div style={{ width: m.progress + '%', height: '100%', background: S.accent }} />
                    </div>
                  </div>
                  <div style={{ textAlign: 'right' }}>
                    <div style={{ fontFamily: S.serif, fontSize: 18, fontWeight: 500 }}>{m.amount}</div>
                    <div style={{ fontSize: 11, color: S.textMute, fontFamily: S.mono }}>{m.progress}%</div>
                  </div>
                </div>
              ))}
            </div>

            {/* Activité + stats compact */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 16, padding: 20 }}>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, marginBottom: 4 }}>
                  <span style={{ fontFamily: S.serif, fontSize: 30, fontWeight: 500, lineHeight: 1, letterSpacing: '-0.02em' }}>47 320 €</span>
                  <span style={{ fontSize: 12, color: S.green, fontWeight: 600, background: S.greenSoft, padding: '2px 8px', borderRadius: 999 }}>+12%</span>
                </div>
                <div style={{ fontSize: 12, color: S.textMute }}>Volume engagé ce mois — 3 paiements à valider</div>
              </div>

              <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 16, padding: 20, flex: 1 }}>
                <h3 style={{ fontFamily: S.serif, fontSize: 18, margin: 0, marginBottom: 12, fontWeight: 500 }}>Conversations en cours</h3>
                {[
                  { name: 'Élise Marchand', last: 'Tu as vu le brief mis à jour ?', time: 'il y a 14 min', pid: 1, unread: true },
                  { name: 'Julien Petit', last: 'Voici la v2 des wireframes', time: 'il y a 1 h', pid: 0, unread: true },
                  { name: 'Théo Renaud', last: 'Audit terminé', time: 'ce matin', pid: 5, unread: false },
                ].map((m, i) => (
                  <div key={i} style={{ padding: '10px 0', borderTop: i > 0 ? `1px solid ${S.border}` : 'none', display: 'flex', gap: 10, alignItems: 'center' }}>
                    <Portrait id={m.pid} size={34} />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span style={{ fontSize: 12.5, fontWeight: m.unread ? 700 : 500 }}>{m.name}</span>
                        <span style={{ fontSize: 10, color: S.textMute, fontStyle: 'italic' }}>{m.time}</span>
                      </div>
                      <div style={{ fontSize: 11.5, color: S.textMute, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{m.last}</div>
                    </div>
                    {m.unread && <span style={{ width: 7, height: 7, borderRadius: '50%', background: S.accent }} />}
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

// ═══ FIND v2 — cards humaines avec citation ═══════════════════════
function SoleilFind() {
  const freelancers = [
    { name: 'Élise Marchand', title: 'UX Designer · Brand', loc: 'Paris', tjm: '650 €', exp: '8 ans', rating: 4.9, reviews: 47, avail: 'Disponible', tags: ['Figma', 'Design system'], pid: 1, verified: true, quote: "J'aime poser un cadre méthodo dès la première semaine.", project: 'Qonto · Memo Bank · Spendesk' },
    { name: 'Julien Petit', title: 'Brand & DA', loc: 'Lyon', tjm: '720 €', exp: '12 ans', rating: 5.0, reviews: 31, avail: 'Sous 2 sem.', tags: ['Branding', 'Editorial'], pid: 0, verified: true, quote: 'Une marque, c\'est avant tout un point de vue.', project: 'Le Slip Français · Veja · Cdiscount' },
    { name: 'Théo Renaud', title: 'Dev Full-Stack', loc: 'Remote', tjm: '580 €', exp: '6 ans', rating: 4.8, reviews: 62, avail: 'Disponible', tags: ['Next.js', 'AWS'], pid: 5, verified: true, quote: 'Je code peu, mais je code bien — et je documente tout.', project: 'Doctolib · Aircall · Pennylane' },
    { name: 'Camille Dubois', title: 'Product Designer', loc: 'Bordeaux', tjm: '600 €', exp: '7 ans', rating: 4.9, reviews: 38, avail: 'Disponible', tags: ['Mobile', 'iOS'], pid: 3, verified: false, quote: 'Le mobile, c\'est 80% de l\'usage. Pourquoi le designer en dernier ?', project: 'Lydia · Yuka · Heetch' },
    { name: 'Mehdi Bensalem', title: 'Data Scientist', loc: 'Marseille', tjm: '750 €', exp: '9 ans', rating: 4.7, reviews: 24, avail: 'Sous 1 mois', tags: ['Python', 'ML'], pid: 4, verified: true, quote: 'La donnée est facile. La rendre actionnable, c\'est mon métier.', project: 'BlaBlaCar · Veepee · Ornikar' },
    { name: 'Léa Fontaine', title: 'Motion Designer', loc: 'Nantes', tjm: '520 €', exp: '5 ans', rating: 4.9, reviews: 19, avail: 'Disponible', tags: ['AE', '3D'], pid: 2, verified: true, quote: 'Le mouvement raconte ce que les mots ne peuvent pas.', project: 'Arte · Canal+ · Le Monde' },
  ];

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: S.bg, fontFamily: S.sans, color: S.text }}>
      <SSidebar active="find" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <STopbar />
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          {/* Header éditorial */}
          <div style={{ padding: '32px 36px 24px' }}>
            <div style={{ fontSize: 11, color: S.accent, marginBottom: 8, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase', display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ width: 24, height: 1, background: S.accent }} /> 1 247 freelances vérifiés
            </div>
            <h1 style={{ fontFamily: S.serif, fontSize: 42, margin: 0, fontWeight: 400, marginBottom: 6, letterSpacing: '-0.025em' }}>Rencontre celles et ceux <span style={{ fontStyle: 'italic', color: S.accent }}>qui feront</span> ton prochain projet.</h1>
            <p style={{ fontSize: 15, color: S.textMute, margin: 0, maxWidth: 580 }}>Chaque profil est vérifié à la main par notre équipe. Pas de faux avis, pas de profils fantômes.</p>
          </div>

          {/* Filter chips bar */}
          <div style={{ padding: '0 36px 16px', display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap', borderBottom: `1px solid ${S.border}` }}>
            <button style={{ background: '#fff', border: `1px solid ${S.borderStrong}`, padding: '8px 14px', fontSize: 13, fontWeight: 500, borderRadius: 999, display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
              <SI name="Sliders" size={14} /> Tous les filtres
            </button>
            <span style={{ width: 1, height: 20, background: S.border, margin: '0 4px' }} />
            {[
              { label: 'Disponible', val: 'maintenant', active: true },
              { label: 'TJM', val: 'moins de 700 €', active: true },
              { label: 'Lieu', val: 'France', active: false },
              { label: 'Expertise', val: 'Design, Dev', active: true },
              { label: 'Mode', val: 'Hybride', active: false },
            ].map((c, i) => (
              <button key={i} style={{ background: c.active ? S.accentSoft : '#fff', border: `1px solid ${c.active ? S.accent + '60' : S.border}`, padding: '8px 14px', fontSize: 13, fontWeight: 500, borderRadius: 999, display: 'flex', alignItems: 'center', gap: 6, color: c.active ? S.accentDeep : S.text, cursor: 'pointer' }}>
                <span style={{ color: c.active ? S.accent : S.textMute }}>{c.label}</span>
                <span style={{ fontWeight: 600 }}>{c.val}</span>
                {c.active && <span style={{ marginLeft: 2, color: S.textMute, cursor: 'pointer' }}>×</span>}
              </button>
            ))}
            <div style={{ flex: 1 }} />
            <span style={{ fontSize: 12, color: S.textMute }}>Tri</span>
            <button style={{ background: 'none', border: 'none', fontSize: 13, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 4, cursor: 'pointer', color: S.text }}>Pertinence <SI name="ChevronDown" size={13} /></button>
          </div>

          {/* Grid */}
          <div style={{ flex: 1, overflow: 'auto', padding: '24px 36px' }}>
            <div style={{ marginBottom: 16, fontSize: 13, color: S.textMute }}><strong style={{ color: S.text }}>132 freelances</strong> correspondent à tes critères</div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 18 }}>
              {freelancers.map((f, i) => (
                <div key={i} style={{ background: '#fff', borderRadius: 18, overflow: 'hidden', border: `1px solid ${S.border}`, cursor: 'pointer', display: 'flex', flexDirection: 'column' }}>
                  {/* Photo zone — taille réduite, citation prend la place */}
                  <div style={{ padding: '20px 20px 0', display: 'flex', gap: 14, alignItems: 'flex-start' }}>
                    <div style={{ position: 'relative', flexShrink: 0 }}>
                      <Portrait id={f.pid} size={64} rounded={14} />
                      {f.verified && <div style={{ position: 'absolute', bottom: -3, right: -3, width: 20, height: 20, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 1px 4px rgba(0,0,0,0.12)' }}><SI name="Verified" size={14} /></div>}
                    </div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: 16, fontWeight: 600, fontFamily: S.serif, letterSpacing: '-0.01em', marginBottom: 2 }}>{f.name}</div>
                      <div style={{ fontSize: 12.5, color: S.textMute, marginBottom: 6 }}>{f.title}</div>
                      <div style={{ display: 'flex', gap: 10, fontSize: 11, color: S.textMute, alignItems: 'center' }}>
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 3 }}><SI name="MapPin" size={11} /> {f.loc}</span>
                        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 3 }}><SI name="Star" size={11} /> <strong style={{ color: S.text }}>{f.rating}</strong> ({f.reviews})</span>
                      </div>
                    </div>
                  </div>

                  {/* Citation — l'élément humain qui change tout */}
                  <div style={{ padding: '18px 20px 14px', flex: 1 }}>
                    <div style={{ fontFamily: S.serif, fontStyle: 'italic', fontSize: 14.5, lineHeight: 1.45, color: S.text, marginBottom: 12, letterSpacing: '-0.005em', textWrap: 'pretty' }}>« {f.quote} »</div>
                    <div style={{ fontSize: 11, color: S.textSubtle, marginBottom: 10, fontFamily: S.mono, letterSpacing: '0.02em' }}>A travaillé avec</div>
                    <div style={{ fontSize: 12.5, color: S.textMute, marginBottom: 14 }}>{f.project}</div>
                    <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap' }}>
                      {f.tags.map((t, ti) => (
                        <span key={ti} style={{ fontSize: 11, padding: '3px 9px', background: S.bg, borderRadius: 999, color: S.textMute, fontWeight: 500 }}>{t}</span>
                      ))}
                    </div>
                  </div>

                  {/* Footer */}
                  <div style={{ padding: '12px 20px', borderTop: `1px solid ${S.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: S.bg }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      {f.avail === 'Disponible' && <span style={{ fontSize: 11, color: S.green, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 5 }}><span style={{ width: 6, height: 6, borderRadius: '50%', background: S.green }} /> Disponible</span>}
                      {f.avail !== 'Disponible' && <span style={{ fontSize: 11, color: S.textMute, fontStyle: 'italic' }}>{f.avail}</span>}
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <span style={{ fontFamily: S.serif, fontSize: 17, fontWeight: 600 }}>{f.tjm}</span>
                      <span style={{ fontSize: 11, color: S.textMute }}>/jour</span>
                      <button style={{ background: S.text, color: '#fff', border: 'none', padding: '7px 14px', fontSize: 12, fontWeight: 600, borderRadius: 999, cursor: 'pointer', marginLeft: 4 }}>Contacter</button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ PROFILE v2 — éditorial pleine page ═══════════════════════════
function SoleilProfile() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: S.bg, fontFamily: S.sans, color: S.text }}>
      <SSidebar active="profile" role="freelancer" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <STopbar />
        <div style={{ flex: 1, overflow: 'auto' }}>
          {/* Cover — décoratif, ne déborde plus sur le contenu */}
          <div style={{ height: 160, background: 'linear-gradient(135deg, #fde9e3, #fde6ed, #fbf0dc)', position: 'relative', overflow: 'hidden' }}>
            <div style={{ position: 'absolute', top: -80, right: -40, width: 260, height: 260, borderRadius: '50%', background: 'radial-gradient(circle, rgba(232,93,74,0.28), transparent 70%)' }} />
            <div style={{ position: 'absolute', top: -100, left: 180, width: 200, height: 200, borderRadius: '50%', background: 'radial-gradient(circle, rgba(240,138,168,0.35), transparent 70%)' }} />
          </div>

          <div style={{ padding: '0 48px', marginTop: -28 }}>
            <div style={{ background: '#fff', borderRadius: 20, border: `1px solid ${S.border}`, padding: 28, display: 'flex', gap: 24, alignItems: 'flex-start', boxShadow: '0 4px 24px rgba(42,31,21,0.04)', position: 'relative', zIndex: 2 }}>
              <div style={{ position: 'relative' }}>
                <div style={{ padding: 4, background: '#fff', borderRadius: 24, boxShadow: '0 2px 12px rgba(42,31,21,0.06)' }}>
                  <Portrait id={1} size={130} rounded={20} />
                </div>
                <div style={{ position: 'absolute', bottom: 0, right: 0, width: 28, height: 28, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 2px 6px rgba(0,0,0,0.15)' }}><SI name="Verified" size={18} /></div>
              </div>
              <div style={{ flex: 1, paddingTop: 8 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4 }}>
                  <h1 style={{ fontFamily: S.serif, fontSize: 38, margin: 0, fontWeight: 500, letterSpacing: '-0.025em' }}>Élise Marchand</h1>
                  <span style={{ marginLeft: 'auto', fontSize: 11, padding: '4px 10px', background: S.greenSoft, color: S.green, borderRadius: 999, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 5 }}><span style={{ width: 6, height: 6, borderRadius: '50%', background: S.green }} /> Disponible dès lundi</span>
                </div>
                <div style={{ fontSize: 17, color: S.textMute, marginBottom: 14, fontFamily: S.serif, fontWeight: 400, fontStyle: 'italic' }}>UX Designer & Brand pour startups B2B</div>
                <div style={{ display: 'flex', gap: 20, fontSize: 13, color: S.textMute, alignItems: 'center', marginBottom: 18, flexWrap: 'wrap' }}>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}><SI name="MapPin" size={13} /> Paris, télétravail possible</span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}><SI name="Globe" size={13} /> Français · Anglais</span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}><SI name="Star" size={13} /> <strong style={{ color: S.text }}>4,9</strong> sur 47 avis</span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}><SI name="Clock" size={13} /> Répond en deux heures</span>
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <button style={{ background: S.text, color: '#fff', border: 'none', padding: '11px 22px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}><SI name="Send" size={14} /> Envoyer un message</button>
                  <button style={{ background: '#fff', border: `1px solid ${S.borderStrong}`, padding: '11px 18px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Inviter sur un projet</button>
                  <button style={{ background: '#fff', border: `1px solid ${S.border}`, padding: 11, borderRadius: 999, cursor: 'pointer' }}><SI name="Bookmark" size={14} /></button>
                </div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{ fontSize: 11, color: S.textMute, marginBottom: 4, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600 }}>À partir de</div>
                <div style={{ fontFamily: S.serif, fontSize: 38, fontWeight: 500, lineHeight: 1, letterSpacing: '-0.025em' }}>650 €<span style={{ fontSize: 16, color: S.textMute, fontWeight: 400 }}>/jour</span></div>
              </div>
            </div>

            {/* Tabs */}
            <div style={{ display: 'flex', gap: 4, marginTop: 24, marginBottom: 24, padding: 4, background: '#fff', borderRadius: 999, width: 'fit-content', border: `1px solid ${S.border}` }}>
              {['Son histoire', 'Ses réalisations', 'Avis · 47', 'Tarification'].map((t, i) => (
                <div key={i} style={{ padding: '8px 18px', borderRadius: 999, fontSize: 13, fontWeight: 600, background: i === 0 ? S.text : 'transparent', color: i === 0 ? '#fff' : S.textMute, cursor: 'pointer' }}>{t}</div>
              ))}
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1.7fr 1fr', gap: 24, paddingBottom: 48 }}>
              <div>
                {/* Citation pleine page */}
                <div style={{ padding: '32px 0 24px', borderTop: `1px solid ${S.border}`, marginBottom: 8 }}>
                  <div style={{ fontFamily: S.serif, fontSize: 32, lineHeight: 1.25, fontWeight: 400, fontStyle: 'italic', letterSpacing: '-0.015em', textWrap: 'pretty', color: S.text, marginBottom: 16 }}>« J'aime poser un cadre méthodo dès la première semaine. C'est ce qui fait qu'à la fin du mandat, l'équipe est autonome — et non dépendante de moi. »</div>
                </div>

                <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 16, padding: 28, marginBottom: 20 }}>
                  <h2 style={{ fontFamily: S.serif, fontSize: 24, margin: 0, marginBottom: 14, fontWeight: 500 }}>Mon parcours</h2>
                  <p style={{ fontSize: 15, lineHeight: 1.7, margin: 0, marginBottom: 12, textWrap: 'pretty' }}>
                    J'accompagne les startups B2B dans la conception de produits SaaS clairs et au goût du jour. Huit ans entre Paris et Berlin, avec un faible pour les <strong style={{ color: S.accent }}>fintech, healthtech et marketplaces</strong>.
                  </p>
                  <p style={{ fontSize: 15, lineHeight: 1.7, margin: 0, color: S.textMute, textWrap: 'pretty' }}>
                    Discovery, design system, design ops — j'aime accompagner les équipes produit dans la durée, généralement 3 à 6 mois, en mode 3-4 jours par semaine.
                  </p>
                </div>

                <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 16, padding: 28, marginBottom: 20 }}>
                  <h2 style={{ fontFamily: S.serif, fontSize: 24, margin: 0, marginBottom: 18, fontWeight: 500 }}>Sélection de réalisations</h2>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 12 }}>
                    {[
                      { title: 'Refonte des cartes Qonto', client: 'Qonto · Fintech · 2024', g1: '#3a4ee0', g2: '#7c8df0' },
                      { title: 'Design System v2', client: 'Memo Bank · 2023', g1: '#0e8a5f', g2: '#5fb88a' },
                      { title: 'Onboarding mobile', client: 'Lydia · 2023', g1: '#e8447b', g2: '#f47ea4' },
                      { title: 'Doctolib Pro', client: 'Doctolib · 2022', g1: '#b8721d', g2: '#d9a05c' },
                    ].map((p, i) => (
                      <div key={i} style={{ borderRadius: 12, overflow: 'hidden', border: `1px solid ${S.border}` }}>
                        <div style={{ height: 140, background: `linear-gradient(135deg, ${p.g1}, ${p.g2})` }} />
                        <div style={{ padding: 14, background: '#fff' }}>
                          <div style={{ fontSize: 14, fontWeight: 600, fontFamily: S.serif }}>{p.title}</div>
                          <div style={{ fontSize: 11.5, color: S.textMute, marginTop: 2 }}>{p.client}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 16, padding: 28 }}>
                  <h2 style={{ fontFamily: S.serif, fontSize: 24, margin: 0, marginBottom: 18, fontWeight: 500 }}>Ce qu'on dit d'elle</h2>
                  <div style={{ borderLeft: `3px solid ${S.accent}`, paddingLeft: 20 }}>
                    <div style={{ fontFamily: S.serif, fontSize: 22, lineHeight: 1.4, marginBottom: 14, fontWeight: 400, fontStyle: 'italic', textWrap: 'pretty' }}>« Élise a posé un cadre méthodo dès la première semaine. On est passés d'un design system fragmenté à une vraie cohésion produit. »</div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                      <Portrait id={4} size={36} />
                      <div>
                        <div style={{ fontSize: 13, fontWeight: 600 }}>Sophie Aubry</div>
                        <div style={{ fontSize: 11, color: S.textMute }}>CPO chez Qonto · 4 mois de mission</div>
                      </div>
                      <div style={{ marginLeft: 'auto', display: 'flex', gap: 1 }}>
                        {[1,2,3,4,5].map(s => <SI key={s} name="Star" size={13} />)}
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 16, padding: 22 }}>
                  <h3 style={{ fontFamily: S.serif, fontSize: 18, margin: 0, marginBottom: 14, fontWeight: 600 }}>Ses outils</h3>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                    {['Figma', 'Design System', 'UX Research', 'Brand', 'Webflow', 'Framer', 'Notion'].map((s, i) => (
                      <span key={i} style={{ fontSize: 12, padding: '5px 11px', background: S.bg, borderRadius: 999, fontWeight: 500 }}>{s}</span>
                    ))}
                  </div>
                </div>

                <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 16, padding: 22 }}>
                  <h3 style={{ fontFamily: S.serif, fontSize: 18, margin: 0, marginBottom: 14, fontWeight: 600 }}>Vérifié par Atelier</h3>
                  {[['Identité KYC', true], ['Email professionnel', true], ['SIRET', true], ['Top 5%', true]].map(([l, ok], i) => (
                    <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 0', fontSize: 13 }}>
                      <SI name="CheckCircle" size={15} />
                      {l}
                    </div>
                  ))}
                </div>

                <div style={{ background: '#fff', border: `1px solid ${S.border}`, borderRadius: 16, padding: 22 }}>
                  <h3 style={{ fontFamily: S.serif, fontSize: 18, margin: 0, marginBottom: 14, fontWeight: 600 }}>En quelques chiffres</h3>
                  {[['Missions livrées', '47'], ['Volume facturé', '312 k€'], ['Taux de réembauche', '68 %'], ['Membre depuis', '2021']].map(([l, v], i) => (
                    <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', fontSize: 13, borderBottom: i < 3 ? `1px solid ${S.border}` : 'none' }}>
                      <span style={{ color: S.textMute }}>{l}</span>
                      <span style={{ fontWeight: 600, fontFamily: S.serif }}>{v}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ MESSAGES v2 — dates FR conversationnelles ════════════════════
function SoleilMessages() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: S.bg, fontFamily: S.sans, color: S.text }}>
      <SSidebar active="msg" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <STopbar />
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex' }}>
          <div style={{ width: 320, borderRight: `1px solid ${S.border}`, background: '#fff', display: 'flex', flexDirection: 'column', flexShrink: 0 }}>
            <div style={{ padding: '24px 22px 14px' }}>
              <h2 style={{ fontFamily: S.serif, fontSize: 28, margin: 0, fontWeight: 500, marginBottom: 12, letterSpacing: '-0.02em' }}>Tes conversations</h2>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, background: S.bg, border: `1px solid ${S.border}`, borderRadius: 999, padding: '7px 14px' }}>
                <SI name="Search" size={13} />
                <input placeholder="Rechercher..." style={{ flex: 1, border: 'none', outline: 'none', background: 'transparent', fontSize: 12 }} />
              </div>
            </div>
            <div style={{ flex: 1, overflow: 'auto' }}>
              {[
                { name: 'Élise Marchand', last: 'Tu as vu le brief mis à jour ?', time: 'à l\'instant', unread: true, pid: 1, active: true, tag: 'Refonte produit' },
                { name: 'Julien Petit', last: 'Voici la v2 des wireframes', time: 'il y a 1 h', unread: true, pid: 0, tag: 'Brand Q2' },
                { name: 'Théo Renaud', last: 'Audit terminé', time: 'ce matin', unread: false, pid: 5, tag: 'Audit SEO' },
                { name: 'Camille Dubois', last: 'Disponible la semaine prochaine ?', time: 'hier soir', unread: false, pid: 3, tag: 'Discovery' },
                { name: 'Mehdi Bensalem', last: 'Merci pour la confirmation', time: 'mardi dernier', unread: false, pid: 4, tag: 'Data' },
              ].map((c, i) => (
                <div key={i} style={{ padding: '14px 22px', borderTop: i > 0 ? `1px solid ${S.border}` : 'none', display: 'flex', gap: 12, cursor: 'pointer', background: c.active ? S.accentSoft : 'transparent' }}>
                  <Portrait id={c.pid} size={40} />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                      <span style={{ fontSize: 13, fontWeight: c.unread ? 700 : 600 }}>{c.name}</span>
                      <span style={{ fontSize: 10, color: S.textMute, fontStyle: 'italic' }}>{c.time}</span>
                    </div>
                    <div style={{ fontSize: 12, color: S.textMute, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', marginBottom: 3 }}>{c.last}</div>
                    <span style={{ fontSize: 10, padding: '2px 8px', background: '#fff', border: `1px solid ${S.border}`, borderRadius: 999, color: S.textMute, fontWeight: 600 }}>{c.tag}</span>
                  </div>
                  {c.unread && <span style={{ width: 8, height: 8, borderRadius: '50%', background: S.accent, marginTop: 7, flexShrink: 0 }} />}
                </div>
              ))}
            </div>
          </div>

          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, background: S.bg }}>
            <div style={{ padding: '14px 24px', borderBottom: `1px solid ${S.border}`, display: 'flex', alignItems: 'center', gap: 14, background: '#fff' }}>
              <Portrait id={1} size={42} />
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={{ fontSize: 15, fontWeight: 600, fontFamily: S.serif }}>Élise Marchand</span>
                  <SI name="Verified" size={13} />
                </div>
                <div style={{ fontSize: 12, color: S.green, display: 'flex', alignItems: 'center', gap: 5 }}><span style={{ width: 6, height: 6, borderRadius: '50%', background: S.green }} /> En ligne</div>
              </div>
              <button style={{ background: 'none', border: `1px solid ${S.border}`, padding: 9, borderRadius: '50%', cursor: 'pointer' }}><SI name="Phone" size={15} /></button>
              <button style={{ background: 'none', border: `1px solid ${S.border}`, padding: 9, borderRadius: '50%', cursor: 'pointer' }}><SI name="Video" size={15} /></button>
              <button style={{ background: S.text, color: '#fff', border: 'none', padding: '9px 16px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}><SI name="Briefcase" size={13} /> Démarrer un projet</button>
            </div>

            <div style={{ flex: 1, overflow: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 12 }}>
              <div style={{ textAlign: 'center', fontSize: 11, color: S.textMute, fontStyle: 'italic', fontFamily: S.serif }}>— Mardi matin —</div>

              <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end', maxWidth: '70%' }}>
                <Portrait id={1} size={28} />
                <div style={{ background: '#fff', border: `1px solid ${S.border}`, padding: '12px 16px', borderRadius: '4px 18px 18px 18px', fontSize: 14, lineHeight: 1.5 }}>Salut Nova ! Tu as eu le temps de regarder le brief mis à jour ?</div>
              </div>

              <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                <div style={{ background: S.accent, color: '#fff', padding: '12px 16px', borderRadius: '18px 4px 18px 18px', fontSize: 14, maxWidth: '70%', lineHeight: 1.5 }}>Oui, j'aime beaucoup la nouvelle approche modulaire. Je te prépare une proposition.</div>
              </div>

              <div style={{ background: '#fff', border: `1px solid ${S.accent}40`, borderRadius: 16, padding: 0, alignSelf: 'flex-start', maxWidth: 460, marginTop: 8, overflow: 'hidden', boxShadow: '0 2px 12px rgba(232,93,74,0.08)' }}>
                <div style={{ padding: '12px 18px', background: 'linear-gradient(90deg, #fde9e3, #fde6ed)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <SI name="Briefcase" size={14} />
                    <span style={{ fontSize: 12, fontWeight: 700, color: S.accentDeep, letterSpacing: '0.05em', textTransform: 'uppercase' }}>Nouvelle proposition</span>
                  </div>
                </div>
                <div style={{ padding: 20 }}>
                  <div style={{ fontFamily: S.serif, fontSize: 24, lineHeight: 1.2, marginBottom: 6, fontWeight: 500, letterSpacing: '-0.02em' }}>Refonte produit Nova v2</div>
                  <div style={{ fontSize: 13, color: S.textMute, marginBottom: 16, lineHeight: 1.5 }}>UX onboarding mobile + design system. Trois mois, trois jours par semaine.</div>
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8, marginBottom: 16 }}>
                    {[['Montant', '23 400 €'], ['Durée', '3 mois'], ['Démarrage', 'lundi']].map(([l, v], i) => (
                      <div key={i} style={{ background: S.bg, padding: '10px 12px', borderRadius: 10 }}>
                        <div style={{ fontSize: 10, color: S.textMute, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 3 }}>{l}</div>
                        <div style={{ fontFamily: S.serif, fontSize: 16, fontWeight: 600 }}>{v}</div>
                      </div>
                    ))}
                  </div>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <button style={{ flex: 1, background: S.accent, color: '#fff', border: 'none', padding: '10px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Accepter</button>
                    <button style={{ background: '#fff', color: S.text, border: `1px solid ${S.borderStrong}`, padding: '10px 16px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Négocier</button>
                  </div>
                </div>
              </div>

              <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end', maxWidth: '70%', marginTop: 8 }}>
                <Portrait id={1} size={28} />
                <div style={{ background: '#fff', border: `1px solid ${S.border}`, padding: '12px 16px', borderRadius: '4px 18px 18px 18px', fontSize: 14, lineHeight: 1.5 }}>J'attends ton retour ! Si OK je bloque mon planning dès lundi.</div>
              </div>
            </div>

            <div style={{ borderTop: `1px solid ${S.border}`, padding: '16px 24px', background: '#fff' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, background: S.bg, borderRadius: 999, padding: '10px 16px' }}>
                <SI name="Smiley" size={18} />
                <SI name="Paperclip" size={17} />
                <input placeholder="Écris un message..." style={{ flex: 1, border: 'none', outline: 'none', background: 'transparent', fontSize: 14, fontFamily: S.sans }} />
                <SI name="Mic" size={17} />
                <button style={{ background: S.accent, color: '#fff', border: 'none', width: 34, height: 34, borderRadius: '50%', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SI name="Send" size={14} /></button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

window.SoleilDashboard = SoleilDashboard;
window.SoleilFind = SoleilFind;
window.SoleilProfile = SoleilProfile;
window.SoleilMessages = SoleilMessages;
window.S = S;
window.SI = SI;
window.SSidebar = SSidebar;
window.STopbar = STopbar;
window.Portrait = Portrait;
