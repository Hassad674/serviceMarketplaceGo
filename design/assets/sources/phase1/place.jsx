// Direction 3 — Atelier Place. Upwork-like dense marketplace with REFINED pink (deeper magenta accent), data-rich, badges, lots of info density. Reclaims the user's pink in a pro way.
const { Icons: I_P } = window;

const P = {
  bg: '#fafafb',
  surface: '#ffffff',
  surfaceAlt: '#f6f5f8',
  border: '#e8e6ee',
  borderStrong: '#d4d1dc',
  text: '#161420',
  textMute: '#5e5870',
  textSubtle: '#8c87a0',
  // Refined pink — not pastel, not bubblegum. Deep magenta-rose, like Hashicorp + a touch of Gumroad pink
  accent: '#c2185b',
  accentSoft: '#fde6ee',
  accentDeep: '#8e1146',
  accent2: '#5b3aff',  // electric purple as secondary
  accent2Soft: '#ece7ff',
  green: '#0a7c4d',
  greenSoft: '#dff2e8',
  amber: '#b87814',
  red: '#c0392b',
  serif: '"Newsreader", Georgia, serif',
  sans: '"Inter Tight", system-ui, sans-serif',
  mono: '"Geist Mono", monospace',
};

function PSidebar({ active = 'home', role = 'enterprise' }) {
  const items = role === 'enterprise' ? [
    { id: 'home', icon: 'Home', label: 'Tableau de bord' },
    { id: 'msg', icon: 'Chat', label: 'Messages', badge: 3 },
    { id: 'proj', icon: 'Folder', label: 'Projets', count: '12' },
    { id: 'jobs', icon: 'Briefcase', label: 'Annonces', count: '4' },
    { id: 'team', icon: 'Users', label: 'Équipe' },
    { id: 'bill', icon: 'Receipt', label: 'Facturation' },
  ] : [
    { id: 'home', icon: 'Home', label: 'Tableau de bord' },
    { id: 'msg', icon: 'Chat', label: 'Messages', badge: 2 },
    { id: 'proj', icon: 'Folder', label: 'Mes missions', count: '7' },
    { id: 'opp', icon: 'Inbox', label: 'Opportunités', badge: 12 },
    { id: 'profile', icon: 'User', label: 'Mon profil' },
    { id: 'earn', icon: 'Receipt', label: 'Revenus' },
  ];
  const find = [
    { id: 'find-f', icon: 'Search', label: 'Freelances', match: 'find' },
    { id: 'find-a', icon: 'Layers', label: 'Agences' },
    { id: 'find-r', icon: 'Sparkle', label: 'Apporteurs' },
  ];
  return (
    <aside style={{ width: 240, height: '100%', background: '#fff', borderRight: `1px solid ${P.border}`, display: 'flex', flexDirection: 'column', fontFamily: P.sans, flexShrink: 0 }}>
      <div style={{ padding: '18px 20px 14px', borderBottom: `1px solid ${P.border}` }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 9 }}>
          <div style={{ width: 28, height: 28, borderRadius: 7, background: P.text, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: P.serif, fontSize: 17, fontWeight: 600, fontStyle: 'italic' }}>a</div>
          <div style={{ fontSize: 16, fontWeight: 600, color: P.text, letterSpacing: '-0.02em' }}>atelier<span style={{ color: P.accent }}>.</span></div>
        </div>
      </div>

      <div style={{ padding: '12px' }}>
        <div style={{ background: P.surfaceAlt, borderRadius: 10, padding: '10px 12px', display: 'flex', alignItems: 'center', gap: 10, cursor: 'pointer' }}>
          <div style={{ width: 30, height: 30, borderRadius: 8, background: P.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 12, letterSpacing: '0.02em' }}>NV</div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 12.5, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>Nova Studio</div>
            <div style={{ fontSize: 10.5, color: P.textMute, display: 'flex', alignItems: 'center', gap: 4 }}>
              <span style={{ padding: '1px 5px', background: '#fff', border: `1px solid ${P.border}`, borderRadius: 3, fontSize: 9, fontWeight: 600, color: P.accent }}>PRO</span>
              {role === 'enterprise' ? 'Entreprise' : 'Prestataire'}
            </div>
          </div>
          <I_P.ChevronDown size={12} style={{ color: P.textMute }} />
        </div>
      </div>

      <nav style={{ padding: '0 8px', flex: 1, overflow: 'auto' }}>
        <div style={{ fontSize: 10, color: P.textSubtle, padding: '8px 12px 6px', fontWeight: 700, letterSpacing: '0.08em', textTransform: 'uppercase' }}>Espace de travail</div>
        {items.map((it) => {
          const Ic = I_P[it.icon];
          const isActive = active === it.id;
          return (
            <div key={it.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '7px 12px', borderRadius: 7, fontSize: 13, fontWeight: isActive ? 600 : 500, color: isActive ? P.text : P.textMute, background: isActive ? P.surfaceAlt : 'transparent', cursor: 'pointer', position: 'relative' }}>
              {isActive && <div style={{ position: 'absolute', left: -8, top: 8, bottom: 8, width: 2, background: P.accent, borderRadius: 2 }} />}
              <Ic size={15} stroke={1.7} />
              <span style={{ flex: 1 }}>{it.label}</span>
              {it.badge && <span style={{ background: P.accent, color: '#fff', fontSize: 9.5, fontWeight: 700, padding: '1px 5px', borderRadius: 4, minWidth: 16, textAlign: 'center' }}>{it.badge}</span>}
              {it.count && !it.badge && <span style={{ fontSize: 10, color: P.textSubtle, fontWeight: 600 }}>{it.count}</span>}
            </div>
          );
        })}

        <div style={{ fontSize: 10, color: P.textSubtle, padding: '14px 12px 6px', fontWeight: 700, letterSpacing: '0.08em', textTransform: 'uppercase' }}>Marketplace</div>
        {find.map((it) => {
          const Ic = I_P[it.icon];
          const isActive = active === it.id || active === it.match;
          return (
            <div key={it.id} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '7px 12px', borderRadius: 7, fontSize: 13, fontWeight: isActive ? 600 : 500, color: isActive ? P.text : P.textMute, background: isActive ? P.surfaceAlt : 'transparent', cursor: 'pointer', position: 'relative' }}>
              {isActive && <div style={{ position: 'absolute', left: -8, top: 8, bottom: 8, width: 2, background: P.accent, borderRadius: 2 }} />}
              <Ic size={15} stroke={1.7} />
              <span>{it.label}</span>
            </div>
          );
        })}
      </nav>

      <div style={{ padding: 12, borderTop: `1px solid ${P.border}` }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 8px', cursor: 'pointer', borderRadius: 7 }}>
          <I_P.Settings size={15} style={{ color: P.textMute }} stroke={1.7} />
          <span style={{ fontSize: 12.5, color: P.textMute, fontWeight: 500 }}>Paramètres</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 8px', cursor: 'pointer', borderRadius: 7 }}>
          <I_P.Bell size={15} style={{ color: P.textMute }} stroke={1.7} />
          <span style={{ fontSize: 12.5, color: P.textMute, fontWeight: 500 }}>Notifications</span>
        </div>
      </div>
    </aside>
  );
}

function PTopbar({ title, breadcrumb }) {
  return (
    <div style={{ height: 56, borderBottom: `1px solid ${P.border}`, background: '#fff', display: 'flex', alignItems: 'center', padding: '0 24px', gap: 16, flexShrink: 0 }}>
      {breadcrumb && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12.5, color: P.textMute, fontWeight: 500 }}>
          {breadcrumb.map((b, i) => (
            <React.Fragment key={i}>
              {i > 0 && <I_P.ChevronRight size={11} style={{ opacity: 0.5 }} />}
              <span style={{ color: i === breadcrumb.length - 1 ? P.text : P.textMute, fontWeight: i === breadcrumb.length - 1 ? 600 : 500 }}>{b}</span>
            </React.Fragment>
          ))}
        </div>
      )}
      <div style={{ flex: 1 }} />
      <div style={{ display: 'flex', alignItems: 'center', gap: 7, background: P.surfaceAlt, border: `1px solid ${P.border}`, borderRadius: 7, padding: '6px 11px', width: 280 }}>
        <I_P.Search size={13} style={{ color: P.textMute }} />
        <input placeholder="Rechercher..." style={{ flex: 1, border: 'none', outline: 'none', background: 'transparent', fontSize: 12, color: P.text }} readOnly />
        <span style={{ fontSize: 10, color: P.textSubtle, padding: '1px 5px', background: '#fff', border: `1px solid ${P.border}`, borderRadius: 3, fontFamily: P.mono }}>⌘K</span>
      </div>
      <button style={{ background: P.surfaceAlt, border: `1px solid ${P.border}`, padding: '6px 11px', fontSize: 12, fontWeight: 600, color: P.text, borderRadius: 7, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}>
        <I_P.Help size={13} /> Aide
      </button>
      <button style={{ background: P.text, border: 'none', padding: '7px 13px', fontSize: 12, fontWeight: 600, color: '#fff', borderRadius: 7, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}>
        <I_P.Plus size={13} /> Publier une annonce
      </button>
      <div style={{ width: 32, height: 32, borderRadius: 8, background: P.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 11 }}>NV</div>
    </div>
  );
}

function Avatar({ initials, bg = P.accent, size = 36, square = false }) {
  return (
    <div style={{ width: size, height: size, borderRadius: square ? 8 : '50%', background: bg, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: size * 0.36, flexShrink: 0, letterSpacing: '0.02em' }}>{initials}</div>
  );
}

// ═══ DASHBOARD ═════════════════════════════════════════════════════
function PlaceDashboard() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: P.bg, fontFamily: P.sans, color: P.text }}>
      <PSidebar active="home" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <PTopbar breadcrumb={['Nova Studio', 'Tableau de bord']} />
        <div style={{ flex: 1, overflow: 'hidden', padding: '24px 28px' }}>
          {/* Header */}
          <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', marginBottom: 22 }}>
            <div>
              <h1 style={{ fontFamily: P.serif, fontSize: 32, lineHeight: 1.1, margin: 0, fontWeight: 500, letterSpacing: '-0.02em' }}>Bonjour Nova,</h1>
              <p style={{ fontSize: 13.5, color: P.textMute, margin: '4px 0 0' }}>Mardi 14 mai · 3 actions à valider · 8 nouvelles candidatures</p>
            </div>
            <div style={{ display: 'flex', gap: 6 }}>
              {['7j', '30j', '90j', 'Tout'].map((t, i) => (
                <button key={i} style={{ background: i === 1 ? P.text : '#fff', border: `1px solid ${i === 1 ? P.text : P.border}`, color: i === 1 ? '#fff' : P.textMute, padding: '6px 13px', fontSize: 12, fontWeight: 600, borderRadius: 6, cursor: 'pointer' }}>{t}</button>
              ))}
            </div>
          </div>

          {/* Stat cards row */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12, marginBottom: 18 }}>
            {[
              { label: 'Volume engagé', val: '47 320 €', delta: '+12,4%', deltaColor: P.green, sub: 'vs 30 derniers j.', icon: 'Trend' },
              { label: 'Missions actives', val: '12', delta: '+3', deltaColor: P.green, sub: '4 démarrent cette sem.', icon: 'Pulse' },
              { label: 'Candidatures reçues', val: '127', delta: '+24', deltaColor: P.green, sub: 'sur 4 annonces', icon: 'Inbox' },
              { label: 'Taux de complétion', val: '94%', delta: '+1,2%', deltaColor: P.green, sub: '47 missions livrées', icon: 'CheckCircle' },
            ].map((s, i) => {
              const Ic = I_P[s.icon];
              return (
                <div key={i} style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 10, padding: 16 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
                    <div style={{ width: 28, height: 28, borderRadius: 6, background: P.accentSoft, display: 'flex', alignItems: 'center', justifyContent: 'center', color: P.accent }}><Ic size={14} /></div>
                    <span style={{ fontSize: 11, color: s.deltaColor, fontWeight: 600, background: P.greenSoft, padding: '2px 6px', borderRadius: 4 }}>{s.delta}</span>
                  </div>
                  <div style={{ fontSize: 11, color: P.textMute, marginBottom: 4, fontWeight: 500 }}>{s.label}</div>
                  <div style={{ fontFamily: P.serif, fontSize: 26, fontWeight: 500, lineHeight: 1, marginBottom: 4, letterSpacing: '-0.01em' }}>{s.val}</div>
                  <div style={{ fontSize: 11, color: P.textSubtle }}>{s.sub}</div>
                </div>
              );
            })}
          </div>

          {/* Big content row */}
          <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr', gap: 16 }}>
            {/* Active jobs table */}
            <div style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 10, overflow: 'hidden' }}>
              <div style={{ padding: '14px 18px', borderBottom: `1px solid ${P.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div>
                  <div style={{ fontSize: 14, fontWeight: 600 }}>Annonces en cours</div>
                  <div style={{ fontSize: 11.5, color: P.textMute, marginTop: 1 }}>4 actives · 127 candidatures totales</div>
                </div>
                <div style={{ display: 'flex', gap: 6 }}>
                  <button style={{ background: '#fff', border: `1px solid ${P.border}`, padding: '5px 9px', fontSize: 11.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4, color: P.textMute }}><I_P.Filter size={11} /> Filtrer</button>
                  <button style={{ background: P.accent, border: 'none', padding: '5px 11px', fontSize: 11.5, fontWeight: 600, color: '#fff', borderRadius: 6, cursor: 'pointer' }}>+ Nouvelle annonce</button>
                </div>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '2.5fr 1fr 1fr 1fr 80px', padding: '8px 18px', fontSize: 10, color: P.textSubtle, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase', borderBottom: `1px solid ${P.border}`, background: P.surfaceAlt }}>
                <div>Annonce</div>
                <div>Budget</div>
                <div>Candidat.</div>
                <div>Statut</div>
                <div style={{ textAlign: 'right' }}></div>
              </div>
              {[
                { title: 'Refonte site corporate B2B', sub: 'UX · Webflow · 3 mois', budget: '12 400 €', cand: 34, status: 'Active', sColor: P.green, sBg: P.greenSoft },
                { title: 'Brand identity Q2 — Lemon', sub: 'Branding · DA · 2 mois', budget: '8 200 €', cand: 28, status: 'Active', sColor: P.green, sBg: P.greenSoft },
                { title: 'Audit SEO technique', sub: 'SEO · Tech · 1 mois', budget: '3 600 €', cand: 19, status: 'Shortlist', sColor: P.amber, sBg: '#fbf0dc' },
                { title: 'Motion launch teaser', sub: 'Motion · 3D · 6 sem.', budget: '6 800 €', cand: 46, status: 'Draft', sColor: P.textMute, sBg: P.surfaceAlt },
              ].map((r, i) => (
                <div key={i} style={{ display: 'grid', gridTemplateColumns: '2.5fr 1fr 1fr 1fr 80px', padding: '14px 18px', borderBottom: i < 3 ? `1px solid ${P.border}` : 'none', alignItems: 'center', fontSize: 13 }}>
                  <div>
                    <div style={{ fontWeight: 600, marginBottom: 2 }}>{r.title}</div>
                    <div style={{ fontSize: 11, color: P.textMute }}>{r.sub}</div>
                  </div>
                  <div style={{ fontFamily: P.mono, fontSize: 12.5, fontWeight: 600 }}>{r.budget}</div>
                  <div>
                    <div style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 12 }}>
                      <strong style={{ fontFamily: P.mono }}>{r.cand}</strong>
                      <span style={{ display: 'flex' }}>
                        {['#0a7c4d', '#5b3aff', '#c2185b'].map((c, ci) => (
                          <div key={ci} style={{ width: 18, height: 18, borderRadius: '50%', background: c, marginLeft: ci ? -5 : 0, border: '2px solid #fff' }} />
                        ))}
                      </span>
                    </div>
                  </div>
                  <div><span style={{ fontSize: 11, padding: '2px 8px', background: r.sBg, color: r.sColor, borderRadius: 4, fontWeight: 600 }}>{r.status}</span></div>
                  <div style={{ textAlign: 'right' }}>
                    <button style={{ background: 'none', border: `1px solid ${P.border}`, padding: '4px 9px', fontSize: 11, fontWeight: 600, borderRadius: 5, cursor: 'pointer', color: P.text }}>Voir</button>
                  </div>
                </div>
              ))}
            </div>

            {/* Right column */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              {/* Spotlight pick */}
              <div style={{ background: 'linear-gradient(135deg, #fde6ee, #ece7ff)', border: `1px solid ${P.accentSoft}`, borderRadius: 10, padding: 16, position: 'relative', overflow: 'hidden' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 10 }}>
                  <I_P.Sparkle size={13} style={{ color: P.accent }} />
                  <span style={{ fontSize: 10.5, fontWeight: 700, color: P.accent, letterSpacing: '0.08em', textTransform: 'uppercase' }}>Talent du jour</span>
                </div>
                <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
                  <Avatar initials="EM" bg={P.accent} size={48} square />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 1, display: 'flex', alignItems: 'center', gap: 5 }}>Élise Marchand <I_P.Verified size={12} style={{ color: P.accent }} /></div>
                    <div style={{ fontSize: 11.5, color: P.textMute, marginBottom: 6 }}>UX Designer · Brand · 8 ans</div>
                    <div style={{ display: 'flex', gap: 10, fontSize: 11, color: P.textMute, marginBottom: 10 }}>
                      <span><I_P.Star size={10} fill={P.amber} style={{ color: P.amber, verticalAlign: '-1px' }} /> <strong style={{ color: P.text, fontFamily: P.mono }}>4,9</strong></span>
                      <span style={{ fontFamily: P.mono, color: P.text, fontWeight: 600 }}>650 €/j</span>
                      <span style={{ color: P.green, fontWeight: 600 }}>● Dispo</span>
                    </div>
                    <button style={{ background: P.text, color: '#fff', border: 'none', padding: '5px 11px', fontSize: 11, fontWeight: 600, borderRadius: 5, cursor: 'pointer' }}>Inviter sur projet →</button>
                  </div>
                </div>
              </div>

              {/* Activity feed */}
              <div style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 10, padding: '14px 16px', flex: 1 }}>
                <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 12 }}>Activité récente</div>
                {[
                  { who: 'Julien Petit', what: 'a accepté ta proposition', ctx: 'Brand Q2', t: '14 min', c: '#5b3aff', icon: 'CheckCircle', iColor: P.green },
                  { who: 'Élise M.', what: 'a envoyé un message', ctx: 'Refonte produit', t: '1 h', c: '#0a7c4d', icon: 'Chat', iColor: P.accent },
                  { who: 'Théo R.', what: 'a livré le jalon 2/3', ctx: 'Audit SEO', t: '3 h', c: '#3a4ee0', icon: 'Package', iColor: P.amber },
                  { who: 'Camille D.', what: 'a postulé', ctx: 'Annonce Discovery', t: 'Hier', c: '#b8721d', icon: 'Inbox', iColor: P.textMute },
                ].map((a, i) => {
                  const Ic = I_P[a.icon];
                  return (
                    <div key={i} style={{ display: 'flex', gap: 10, padding: '8px 0', borderTop: i > 0 ? `1px solid ${P.border}` : 'none', alignItems: 'flex-start' }}>
                      <div style={{ width: 22, height: 22, borderRadius: 5, background: P.surfaceAlt, color: a.iColor, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, marginTop: 1 }}><Ic size={11} /></div>
                      <div style={{ flex: 1, minWidth: 0, fontSize: 12, lineHeight: 1.4 }}>
                        <div><strong>{a.who}</strong> {a.what} <span style={{ color: P.textMute }}>· {a.ctx}</span></div>
                        <div style={{ fontSize: 10.5, color: P.textSubtle, marginTop: 1 }}>{a.t}</div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ FIND ══════════════════════════════════════════════════════════
function PlaceFind() {
  const freelancers = [
    { name: 'Élise Marchand', title: 'UX Designer · Brand systems', loc: 'Paris', tjm: '650', exp: 8, rating: 4.9, reviews: 47, avail: 'Disponible', tags: ['Figma', 'Design system', 'B2B SaaS'], p: 'EM', pBg: '#0a7c4d', verified: true, top: true, completion: 98 },
    { name: 'Julien Petit', title: 'Brand & Direction Artistique', loc: 'Lyon', tjm: '720', exp: 12, rating: 5.0, reviews: 31, avail: 'Sous 2 sem.', tags: ['Branding', 'Editorial', 'Print'], p: 'JP', pBg: '#5b3aff', verified: true, top: true, completion: 100 },
    { name: 'Théo Renaud', title: 'Dev Full-Stack · Cloud', loc: 'Remote', tjm: '580', exp: 6, rating: 4.8, reviews: 62, avail: 'Disponible', tags: ['Next.js', 'AWS', 'Postgres'], p: 'TR', pBg: '#c2185b', verified: true, top: false, completion: 96 },
    { name: 'Camille Dubois', title: 'Product Designer · Mobile', loc: 'Bordeaux', tjm: '600', exp: 7, rating: 4.9, reviews: 38, avail: 'Disponible', tags: ['Mobile', 'iOS', 'Discovery'], p: 'CD', pBg: '#b87814', verified: false, top: false, completion: 92 },
    { name: 'Mehdi Bensalem', title: 'Data Scientist · ML', loc: 'Marseille', tjm: '750', exp: 9, rating: 4.7, reviews: 24, avail: 'Sous 1 mois', tags: ['Python', 'ML', 'BigQuery'], p: 'MB', pBg: '#1f2caa', verified: true, top: true, completion: 95 },
    { name: 'Léa Fontaine', title: 'Motion Designer · 3D', loc: 'Nantes', tjm: '520', exp: 5, rating: 4.9, reviews: 19, avail: 'Disponible', tags: ['After Effects', 'Cinema 4D'], p: 'LF', pBg: '#c0392b', verified: true, top: false, completion: 100 },
  ];

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: P.bg, fontFamily: P.sans, color: P.text }}>
      <PSidebar active="find" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <PTopbar breadcrumb={['Marketplace', 'Freelances']} />
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex' }}>
          {/* Filters left */}
          <div style={{ width: 240, borderRight: `1px solid ${P.border}`, background: '#fff', padding: 18, overflow: 'auto', flexShrink: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
              <div style={{ fontSize: 13, fontWeight: 700 }}>Filtres</div>
              <button style={{ background: 'none', border: 'none', fontSize: 11, color: P.accent, fontWeight: 600, cursor: 'pointer' }}>Réinitialiser</button>
            </div>

            {[
              { title: 'Disponibilité', items: [['Maintenant', 142, true], ['Sous 2 sem.', 89, false], ['Sous 1 mois', 56, false]] },
              { title: 'Localisation', items: [['Paris', 312, false], ['Lyon', 87, false], ['Remote', 245, true]] },
              { title: 'Expertise', items: [['Design (UI/UX)', 234, true], ['Développement', 312, false], ['Brand · DA', 89, false], ['Marketing', 76, false], ['Data', 54, false]] },
            ].map((f, i) => (
              <div key={i} style={{ marginBottom: 18 }}>
                <div style={{ fontSize: 11, fontWeight: 700, color: P.textSubtle, letterSpacing: '0.06em', textTransform: 'uppercase', marginBottom: 8 }}>{f.title}</div>
                {f.items.map(([l, n, on], ii) => (
                  <label key={ii} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', fontSize: 12.5, color: P.text, cursor: 'pointer' }}>
                    <span style={{ width: 14, height: 14, border: `1.5px solid ${on ? P.accent : P.borderStrong}`, borderRadius: 3, background: on ? P.accent : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>{on && <I_P.Check size={9} style={{ color: '#fff' }} />}</span>
                    <span style={{ flex: 1 }}>{l}</span>
                    <span style={{ fontSize: 10.5, color: P.textSubtle, fontFamily: P.mono }}>{n}</span>
                  </label>
                ))}
              </div>
            ))}

            <div style={{ marginBottom: 18 }}>
              <div style={{ fontSize: 11, fontWeight: 700, color: P.textSubtle, letterSpacing: '0.06em', textTransform: 'uppercase', marginBottom: 8 }}>TJM</div>
              <div style={{ display: 'flex', gap: 6, marginBottom: 8 }}>
                <input value="350" readOnly style={{ flex: 1, padding: '6px 8px', fontSize: 11.5, border: `1px solid ${P.border}`, borderRadius: 5, fontFamily: P.mono }} />
                <span style={{ alignSelf: 'center', color: P.textMute, fontSize: 11 }}>—</span>
                <input value="900" readOnly style={{ flex: 1, padding: '6px 8px', fontSize: 11.5, border: `1px solid ${P.border}`, borderRadius: 5, fontFamily: P.mono }} />
              </div>
              <div style={{ height: 4, background: P.border, borderRadius: 2, position: 'relative' }}>
                <div style={{ position: 'absolute', left: '20%', right: '30%', top: 0, bottom: 0, background: P.accent, borderRadius: 2 }} />
                <div style={{ position: 'absolute', left: '20%', top: -4, width: 12, height: 12, borderRadius: '50%', background: '#fff', border: `2px solid ${P.accent}`, transform: 'translateX(-50%)' }} />
                <div style={{ position: 'absolute', left: '70%', top: -4, width: 12, height: 12, borderRadius: '50%', background: '#fff', border: `2px solid ${P.accent}`, transform: 'translateX(-50%)' }} />
              </div>
            </div>

            <div>
              <div style={{ fontSize: 11, fontWeight: 700, color: P.textSubtle, letterSpacing: '0.06em', textTransform: 'uppercase', marginBottom: 8 }}>Vérifications</div>
              {['Vérifié Atelier', 'Top 5%', 'Identité KYC'].map((l, i) => (
                <label key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', fontSize: 12.5, cursor: 'pointer' }}>
                  <span style={{ width: 14, height: 14, border: `1.5px solid ${i === 0 ? P.accent : P.borderStrong}`, borderRadius: 3, background: i === 0 ? P.accent : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>{i === 0 && <I_P.Check size={9} style={{ color: '#fff' }} />}</span>
                  <span>{l}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Right results */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
            {/* Result header */}
            <div style={{ padding: '20px 28px 14px', borderBottom: `1px solid ${P.border}`, background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <h1 style={{ fontFamily: P.serif, fontSize: 24, margin: 0, fontWeight: 500, letterSpacing: '-0.01em' }}>132 freelances correspondent</h1>
                <p style={{ fontSize: 12, color: P.textMute, margin: '2px 0 0' }}>Tri par pertinence · 6 actifs sur ton brief "Refonte produit B2B"</p>
              </div>
              <div style={{ display: 'flex', gap: 6 }}>
                <button style={{ background: '#fff', border: `1px solid ${P.border}`, padding: '6px 11px', fontSize: 11.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}><I_P.Save size={11} /> Sauver la recherche</button>
                <button style={{ background: '#fff', border: `1px solid ${P.border}`, padding: '6px 11px', fontSize: 11.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}>Trier : Pertinence <I_P.ChevronDown size={10} /></button>
              </div>
            </div>

            {/* Result rows — list view, denser than Soleil's grid */}
            <div style={{ flex: 1, overflow: 'auto', padding: '12px 28px 28px' }}>
              {freelancers.map((f, i) => (
                <div key={i} style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 10, padding: 18, marginBottom: 10, display: 'grid', gridTemplateColumns: '64px 1fr 200px 130px', gap: 18, alignItems: 'center', cursor: 'pointer' }}>
                  <div style={{ position: 'relative' }}>
                    <Avatar initials={f.p} bg={f.pBg} size={56} square />
                    {f.verified && <div style={{ position: 'absolute', bottom: -3, right: -3, width: 18, height: 18, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><I_P.Verified size={14} style={{ color: P.accent }} /></div>}
                  </div>

                  <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 3 }}>
                      <span style={{ fontSize: 15, fontWeight: 700 }}>{f.name}</span>
                      {f.top && <span style={{ fontSize: 9.5, padding: '1px 5px', background: 'linear-gradient(90deg,#c2185b,#5b3aff)', color: '#fff', borderRadius: 3, fontWeight: 700, letterSpacing: '0.04em' }}>TOP 5%</span>}
                      <span style={{ marginLeft: 6, fontSize: 11, padding: '1px 6px', background: f.avail === 'Disponible' ? P.greenSoft : P.surfaceAlt, color: f.avail === 'Disponible' ? P.green : P.textMute, borderRadius: 3, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 4 }}><span style={{ width: 5, height: 5, borderRadius: '50%', background: f.avail === 'Disponible' ? P.green : P.textMute }} /> {f.avail}</span>
                    </div>
                    <div style={{ fontSize: 12.5, color: P.textMute, marginBottom: 6 }}>{f.title}</div>
                    <div style={{ display: 'flex', gap: 12, fontSize: 11, color: P.textMute, marginBottom: 8, alignItems: 'center' }}>
                      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 3 }}><I_P.MapPin size={11} /> {f.loc}</span>
                      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 3 }}><I_P.Briefcase size={11} /> {f.exp} ans XP</span>
                      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 3 }}><I_P.CheckCircle size={11} style={{ color: P.green }} /> {f.completion}% complétés</span>
                    </div>
                    <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap' }}>
                      {f.tags.map((t, ti) => (
                        <span key={ti} style={{ fontSize: 10.5, padding: '2px 8px', background: P.surfaceAlt, border: `1px solid ${P.border}`, borderRadius: 4, color: P.text, fontWeight: 500 }}>{t}</span>
                      ))}
                    </div>
                  </div>

                  {/* Stats column */}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 6, fontSize: 11.5 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}><span style={{ color: P.textMute }}>Note</span><span style={{ fontWeight: 700, fontFamily: P.mono }}><I_P.Star size={10} fill={P.amber} style={{ color: P.amber, verticalAlign: '-1px', marginRight: 3 }} />{f.rating} <span style={{ color: P.textMute, fontWeight: 500 }}>({f.reviews})</span></span></div>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}><span style={{ color: P.textMute }}>Réponse</span><span style={{ fontWeight: 600 }}>~ 2h</span></div>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}><span style={{ color: P.textMute }}>Réembauche</span><span style={{ fontWeight: 600, color: P.green }}>{60 + i * 5}%</span></div>
                  </div>

                  {/* Action column */}
                  <div style={{ textAlign: 'right' }}>
                    <div style={{ marginBottom: 8 }}>
                      <div style={{ fontFamily: P.serif, fontSize: 22, fontWeight: 600, lineHeight: 1, letterSpacing: '-0.01em' }}>{f.tjm} €<span style={{ fontSize: 11, color: P.textMute, fontWeight: 400 }}>/j</span></div>
                    </div>
                    <button style={{ display: 'block', width: '100%', background: P.text, color: '#fff', border: 'none', padding: '7px', fontSize: 11.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer', marginBottom: 5 }}>Contacter</button>
                    <button style={{ display: 'block', width: '100%', background: '#fff', color: P.text, border: `1px solid ${P.border}`, padding: '6px', fontSize: 11.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer' }}>Voir profil</button>
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

// ═══ PROFILE ═══════════════════════════════════════════════════════
function PlaceProfile() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: P.bg, fontFamily: P.sans, color: P.text }}>
      <PSidebar active="profile" role="freelancer" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <PTopbar breadcrumb={['Marketplace', 'Freelances', 'Élise Marchand']} />
        <div style={{ flex: 1, overflow: 'auto', padding: '24px 28px' }}>
          {/* Header card */}
          <div style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 12, padding: 24, marginBottom: 16, display: 'grid', gridTemplateColumns: '88px 1fr 280px', gap: 24, alignItems: 'flex-start' }}>
            <div style={{ position: 'relative' }}>
              <Avatar initials="EM" bg="#0a7c4d" size={88} square />
              <div style={{ position: 'absolute', bottom: -4, right: -4, width: 26, height: 26, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', border: `1px solid ${P.border}` }}><I_P.Verified size={18} style={{ color: P.accent }} /></div>
            </div>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 5 }}>
                <h1 style={{ fontFamily: P.serif, fontSize: 30, margin: 0, fontWeight: 500, letterSpacing: '-0.02em' }}>Élise Marchand</h1>
                <span style={{ fontSize: 10, padding: '2px 7px', background: 'linear-gradient(90deg,#c2185b,#5b3aff)', color: '#fff', borderRadius: 4, fontWeight: 700, letterSpacing: '0.04em' }}>TOP 5%</span>
                <span style={{ fontSize: 11, padding: '2px 8px', background: P.greenSoft, color: P.green, borderRadius: 4, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 4 }}><span style={{ width: 5, height: 5, borderRadius: '50%', background: P.green }} /> Disponible immédiatement</span>
              </div>
              <div style={{ fontSize: 15, color: P.textMute, marginBottom: 14 }}>UX Designer & Brand pour startups B2B · Paris</div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 18, paddingTop: 14, borderTop: `1px solid ${P.border}` }}>
                {[
                  ['Note', '4,9 / 5', '47 avis'],
                  ['Missions', '47', '8 ans XP'],
                  ['Volume', '312 k€', 'cumul.'],
                  ['Réembauche', '68%', 'récurrence'],
                ].map(([l, v, s], i) => (
                  <div key={i}>
                    <div style={{ fontSize: 10, color: P.textSubtle, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase', marginBottom: 3 }}>{l}</div>
                    <div style={{ fontFamily: P.serif, fontSize: 22, fontWeight: 500, lineHeight: 1, letterSpacing: '-0.01em' }}>{v}</div>
                    <div style={{ fontSize: 10.5, color: P.textMute, marginTop: 2 }}>{s}</div>
                  </div>
                ))}
              </div>
            </div>
            <div style={{ background: P.surfaceAlt, borderRadius: 10, padding: 18 }}>
              <div style={{ fontSize: 10.5, color: P.textSubtle, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase', marginBottom: 6 }}>À partir de</div>
              <div style={{ fontFamily: P.serif, fontSize: 36, fontWeight: 600, lineHeight: 1, marginBottom: 4, letterSpacing: '-0.02em' }}>650 €<span style={{ fontSize: 14, color: P.textMute, fontWeight: 400 }}>/j</span></div>
              <div style={{ fontSize: 11, color: P.textMute, marginBottom: 14 }}>Hors taxes · 3-4 j/sem typique</div>
              <button style={{ display: 'block', width: '100%', background: P.text, color: '#fff', border: 'none', padding: '10px', fontSize: 13, fontWeight: 600, borderRadius: 6, cursor: 'pointer', marginBottom: 6 }}>Envoyer un message</button>
              <button style={{ display: 'block', width: '100%', background: P.accent, color: '#fff', border: 'none', padding: '10px', fontSize: 13, fontWeight: 600, borderRadius: 6, cursor: 'pointer' }}>Inviter sur un projet</button>
              <div style={{ display: 'flex', gap: 6, marginTop: 8 }}>
                <button style={{ flex: 1, background: '#fff', border: `1px solid ${P.border}`, padding: '7px', fontSize: 11.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 5 }}><I_P.Bookmark size={12} /> Sauver</button>
                <button style={{ flex: 1, background: '#fff', border: `1px solid ${P.border}`, padding: '7px', fontSize: 11.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 5 }}><I_P.Share size={12} /> Partager</button>
              </div>
            </div>
          </div>

          {/* Tabs */}
          <div style={{ display: 'flex', gap: 0, marginBottom: 16, borderBottom: `1px solid ${P.border}` }}>
            {['Vue d\'ensemble', 'Réalisations · 24', 'Avis · 47', 'Tarification', 'Disponibilités'].map((t, i) => (
              <div key={i} style={{ padding: '10px 16px', fontSize: 13, fontWeight: i === 0 ? 700 : 500, color: i === 0 ? P.text : P.textMute, cursor: 'pointer', borderBottom: i === 0 ? `2px solid ${P.accent}` : '2px solid transparent', marginBottom: -1 }}>{t}</div>
            ))}
          </div>

          {/* Content row */}
          <div style={{ display: 'grid', gridTemplateColumns: '1.7fr 1fr', gap: 16 }}>
            <div>
              {/* Bio */}
              <div style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 10, padding: 22, marginBottom: 14 }}>
                <h2 style={{ fontFamily: P.serif, fontSize: 20, margin: 0, marginBottom: 12, fontWeight: 500 }}>Présentation</h2>
                <p style={{ fontSize: 14, lineHeight: 1.65, margin: 0, marginBottom: 10, textWrap: 'pretty' }}>J'accompagne les startups B2B dans la conception de produits SaaS clairs et performants. Huit ans d'expérience entre Paris et Berlin, avec une spécialisation <strong>fintech, healthtech et marketplaces</strong>.</p>
                <p style={{ fontSize: 14, lineHeight: 1.65, margin: 0, color: P.textMute, textWrap: 'pretty' }}>Discovery, design system, design ops — j'aime accompagner les équipes produit dans la durée, en mode 3-4 j/sem sur 3 à 6 mois.</p>
              </div>

              {/* Realisations */}
              <div style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 10, padding: 22, marginBottom: 14 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
                  <h2 style={{ fontFamily: P.serif, fontSize: 20, margin: 0, fontWeight: 500 }}>Réalisations sélectionnées</h2>
                  <a style={{ fontSize: 12, color: P.accent, fontWeight: 600, cursor: 'pointer' }}>Voir les 24 →</a>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
                  {[
                    { title: 'Qonto Cards', client: 'Qonto · Fintech', g1: '#3a4ee0', g2: '#7c8df0' },
                    { title: 'Memo Bank DS v2', client: 'Memo Bank', g1: '#0a7c4d', g2: '#5fb88a' },
                    { title: 'Lydia onboarding', client: 'Lydia · Mobile', g1: '#c2185b', g2: '#ee6b9c' },
                    { title: 'Doctolib Pro', client: 'Doctolib', g1: '#b87814', g2: '#d9a05c' },
                    { title: 'Spendesk pricing', client: 'Spendesk', g1: '#5b3aff', g2: '#9b85ff' },
                    { title: 'Pennylane DA', client: 'Pennylane', g1: '#1f2caa', g2: '#5765d0' },
                  ].map((p, i) => (
                    <div key={i} style={{ borderRadius: 8, overflow: 'hidden', border: `1px solid ${P.border}` }}>
                      <div style={{ height: 100, background: `linear-gradient(135deg, ${p.g1}, ${p.g2})` }} />
                      <div style={{ padding: 10 }}>
                        <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 1 }}>{p.title}</div>
                        <div style={{ fontSize: 10.5, color: P.textMute }}>{p.client}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Reviews preview */}
              <div style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 10, padding: 22 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
                  <h2 style={{ fontFamily: P.serif, fontSize: 20, margin: 0, fontWeight: 500 }}>Avis clients</h2>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <I_P.Star size={14} fill={P.amber} style={{ color: P.amber }} />
                    <span style={{ fontSize: 14, fontWeight: 700, fontFamily: P.mono }}>4,9</span>
                    <span style={{ fontSize: 12, color: P.textMute }}>· 47 avis</span>
                  </div>
                </div>
                {[
                  { name: 'Sophie Aubry', role: 'CPO chez Qonto', text: 'Élise a posé un cadre méthodo dès la première semaine. On est passés d\'un design system fragmenté à une vraie cohésion produit.', rating: 5, p: 'SA', pBg: '#1f2caa', when: 'Il y a 2 semaines · Mission de 4 mois' },
                  { name: 'Marc Lévêque', role: 'Founder chez Vega', text: 'Très pro, livre dans les délais, et surtout sait challenger un brief. La meilleure UX qu\'on ait eue.', rating: 5, p: 'ML', pBg: '#c2185b', when: 'Il y a 1 mois · Mission de 6 sem.' },
                ].map((r, i) => (
                  <div key={i} style={{ padding: '14px 0', borderTop: i > 0 ? `1px solid ${P.border}` : 'none' }}>
                    <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
                      <Avatar initials={r.p} bg={r.pBg} size={36} square />
                      <div style={{ flex: 1 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                          <span style={{ fontSize: 13, fontWeight: 700 }}>{r.name}</span>
                          <span style={{ fontSize: 11.5, color: P.textMute }}>· {r.role}</span>
                          <div style={{ marginLeft: 'auto', display: 'flex', gap: 1 }}>
                            {[1,2,3,4,5].map(s => <I_P.Star key={s} size={11} fill={P.amber} style={{ color: P.amber }} />)}
                          </div>
                        </div>
                        <p style={{ fontSize: 13, lineHeight: 1.55, margin: 0, marginBottom: 5, textWrap: 'pretty' }}>"{r.text}"</p>
                        <div style={{ fontSize: 10.5, color: P.textSubtle }}>{r.when}</div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Right rail */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <div style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 10, padding: 18 }}>
                <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 12 }}>Compétences principales</div>
                {[
                  ['Figma · Design System', 98],
                  ['UX Research', 92],
                  ['Brand Identity', 85],
                  ['Webflow', 78],
                  ['Framer', 72],
                ].map(([s, n], i) => (
                  <div key={i} style={{ marginBottom: 10 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 4 }}>
                      <span style={{ fontWeight: 500 }}>{s}</span>
                      <span style={{ fontFamily: P.mono, color: P.textMute, fontWeight: 600 }}>{n}%</span>
                    </div>
                    <div style={{ height: 4, background: P.border, borderRadius: 2 }}>
                      <div style={{ width: n + '%', height: '100%', background: P.accent, borderRadius: 2 }} />
                    </div>
                  </div>
                ))}
              </div>

              <div style={{ background: '#fff', border: `1px solid ${P.border}`, borderRadius: 10, padding: 18 }}>
                <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 12 }}>Vérifications Atelier</div>
                {[
                  ['Identité KYC', 'Vérifié 14/03/24'],
                  ['Email pro', 'elise@studio.fr'],
                  ['SIRET', '892 314 ...'],
                  ['Compte bancaire', 'Validé Stripe'],
                  ['Top 5%', 'Mis à jour mensuel'],
                ].map(([l, sub], i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '7px 0', borderTop: i > 0 ? `1px solid ${P.border}` : 'none' }}>
                    <I_P.CheckCircle size={14} style={{ color: P.green, flexShrink: 0 }} />
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: 12.5, fontWeight: 600 }}>{l}</div>
                      <div style={{ fontSize: 10.5, color: P.textMute }}>{sub}</div>
                    </div>
                  </div>
                ))}
              </div>

              <div style={{ background: P.surfaceAlt, border: `1px solid ${P.border}`, borderRadius: 10, padding: 18 }}>
                <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 8 }}>Modes de collaboration</div>
                {[['3-4 j/sem · 3-6 mois', 'Préféré'], ['Mission ponctuelle', 'Sur demande'], ['Audit / consulting', 'À partir de 1 200 €/j']].map(([l, v], i) => (
                  <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '7px 0', borderTop: i > 0 ? `1px solid ${P.borderStrong}` : 'none', fontSize: 12 }}>
                    <span style={{ fontWeight: 500 }}>{l}</span>
                    <span style={{ color: P.textMute }}>{v}</span>
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

// ═══ MESSAGES ══════════════════════════════════════════════════════
function PlaceMessages() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: P.bg, fontFamily: P.sans, color: P.text }}>
      <PSidebar active="msg" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <PTopbar breadcrumb={['Messages']} />
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex' }}>
          {/* Conv list */}
          <div style={{ width: 300, borderRight: `1px solid ${P.border}`, background: '#fff', display: 'flex', flexDirection: 'column', flexShrink: 0 }}>
            <div style={{ padding: '14px 16px', borderBottom: `1px solid ${P.border}` }}>
              <div style={{ display: 'flex', gap: 4, marginBottom: 10, padding: 3, background: P.surfaceAlt, borderRadius: 6 }}>
                {['Tous', 'Non lus 3', 'Projets', 'Archivés'].map((t, i) => (
                  <div key={i} style={{ padding: '5px 10px', fontSize: 11.5, fontWeight: 600, borderRadius: 4, cursor: 'pointer', flex: i === 0 ? 1 : 'unset', textAlign: 'center', background: i === 0 ? '#fff' : 'transparent', color: i === 0 ? P.text : P.textMute, boxShadow: i === 0 ? '0 1px 2px rgba(0,0,0,0.04)' : 'none' }}>{t}</div>
                ))}
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 7, background: P.surfaceAlt, border: `1px solid ${P.border}`, borderRadius: 6, padding: '6px 11px' }}>
                <I_P.Search size={12} style={{ color: P.textMute }} />
                <input placeholder="Rechercher conversations..." style={{ flex: 1, border: 'none', outline: 'none', background: 'transparent', fontSize: 11.5 }} />
              </div>
            </div>
            <div style={{ flex: 1, overflow: 'auto' }}>
              {[
                { name: 'Élise Marchand', last: 'Tu as vu le brief mis à jour ?', time: '14 min', unread: 2, p: 'EM', pBg: '#0a7c4d', active: true, tag: 'Refonte produit', tagColor: P.accent },
                { name: 'Julien Petit', last: 'Voici la v2 des wireframes', time: '1 h', unread: 1, p: 'JP', pBg: '#5b3aff', tag: 'Brand Q2', tagColor: '#5b3aff' },
                { name: 'Théo Renaud', last: 'Audit terminé, livraison en cours', time: '3 h', unread: 0, p: 'TR', pBg: '#c2185b', tag: 'Audit SEO', tagColor: P.amber },
                { name: 'Camille Dubois', last: 'Disponible la semaine prochaine ?', time: 'Hier', unread: 0, p: 'CD', pBg: '#b87814', tag: 'Discovery', tagColor: P.green },
                { name: 'Mehdi Bensalem', last: 'Merci pour la confirmation 👍', time: 'Mar.', unread: 0, p: 'MB', pBg: '#1f2caa', tag: 'Data', tagColor: P.textMute },
                { name: 'Léa Fontaine', last: 'Pas de souci, à lundi alors', time: '12 mai', unread: 0, p: 'LF', pBg: '#c0392b', tag: 'Motion', tagColor: P.textMute },
              ].map((c, i) => (
                <div key={i} style={{ padding: '12px 16px', borderBottom: `1px solid ${P.border}`, display: 'flex', gap: 10, cursor: 'pointer', background: c.active ? P.surfaceAlt : 'transparent', borderLeft: c.active ? `3px solid ${P.accent}` : '3px solid transparent' }}>
                  <Avatar initials={c.p} bg={c.pBg} size={36} square />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 1 }}>
                      <span style={{ fontSize: 12.5, fontWeight: c.unread ? 700 : 600 }}>{c.name}</span>
                      <span style={{ fontSize: 10, color: P.textMute }}>{c.time}</span>
                    </div>
                    <div style={{ fontSize: 11.5, color: c.unread ? P.text : P.textMute, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', marginBottom: 4 }}>{c.last}</div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <span style={{ fontSize: 9.5, padding: '1px 6px', background: '#fff', border: `1px solid ${P.border}`, borderRadius: 3, color: c.tagColor, fontWeight: 700, letterSpacing: '0.02em' }}>{c.tag}</span>
                      {c.unread > 0 && <span style={{ marginLeft: 'auto', fontSize: 10, padding: '1px 6px', background: P.accent, color: '#fff', borderRadius: 8, fontWeight: 700 }}>{c.unread}</span>}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Thread + side panel */}
          <div style={{ flex: 1, display: 'flex', minWidth: 0 }}>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
              {/* Thread header */}
              <div style={{ padding: '12px 22px', borderBottom: `1px solid ${P.border}`, display: 'flex', alignItems: 'center', gap: 12, background: '#fff' }}>
                <Avatar initials="EM" bg="#0a7c4d" size={38} square />
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{ fontSize: 14, fontWeight: 700 }}>Élise Marchand</span>
                    <I_P.Verified size={13} style={{ color: P.accent }} />
                    <span style={{ fontSize: 9.5, padding: '1px 5px', background: 'linear-gradient(90deg,#c2185b,#5b3aff)', color: '#fff', borderRadius: 3, fontWeight: 700, letterSpacing: '0.04em' }}>TOP 5%</span>
                  </div>
                  <div style={{ fontSize: 11, color: P.textMute, display: 'flex', alignItems: 'center', gap: 5 }}><span style={{ width: 5, height: 5, borderRadius: '50%', background: P.green }} /> En ligne · UX Designer · Paris</div>
                </div>
                <button style={{ background: '#fff', border: `1px solid ${P.border}`, padding: '6px 11px', fontSize: 11.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}><I_P.Phone size={11} /> Appel</button>
                <button style={{ background: P.text, color: '#fff', border: 'none', padding: '7px 13px', fontSize: 11.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}><I_P.Briefcase size={11} /> Démarrer un projet</button>
                <I_P.MoreH size={16} style={{ color: P.textMute, cursor: 'pointer' }} />
              </div>

              {/* Messages */}
              <div style={{ flex: 1, overflow: 'auto', padding: '20px 22px', display: 'flex', flexDirection: 'column', gap: 10 }}>
                <div style={{ textAlign: 'center', fontSize: 10, color: P.textSubtle, fontWeight: 700, letterSpacing: '0.1em', textTransform: 'uppercase', margin: '4px 0 8px' }}>— Aujourd'hui · 14 mai —</div>

                <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end', maxWidth: '70%' }}>
                  <Avatar initials="EM" bg="#0a7c4d" size={26} square />
                  <div>
                    <div style={{ background: '#fff', border: `1px solid ${P.border}`, padding: '10px 14px', borderRadius: '4px 12px 12px 12px', fontSize: 13, lineHeight: 1.5 }}>Salut Nova ! Tu as eu le temps de regarder le brief mis à jour ?</div>
                    <div style={{ fontSize: 9.5, color: P.textSubtle, marginTop: 3, marginLeft: 4 }}>14:32</div>
                  </div>
                </div>

                <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                  <div>
                    <div style={{ background: P.text, color: '#fff', padding: '10px 14px', borderRadius: '12px 4px 12px 12px', fontSize: 13, maxWidth: 360, lineHeight: 1.5 }}>Oui ! Très bien, j'aime bcp la nouvelle approche modulaire 🙌 Je te prépare une proposition détaillée.</div>
                    <div style={{ fontSize: 9.5, color: P.textSubtle, marginTop: 3, textAlign: 'right' }}>14:38 · Lu</div>
                  </div>
                </div>

                {/* Proposal card */}
                <div style={{ background: '#fff', border: `2px solid ${P.accent}`, borderRadius: 10, padding: 0, alignSelf: 'flex-start', maxWidth: 480, marginTop: 8, overflow: 'hidden' }}>
                  <div style={{ padding: '10px 16px', background: P.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
                      <I_P.Briefcase size={13} />
                      <span style={{ fontSize: 11.5, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase' }}>Proposition de mission</span>
                    </div>
                    <span style={{ fontSize: 10, fontFamily: P.mono, opacity: 0.8 }}>#PRP-0148</span>
                  </div>
                  <div style={{ padding: 18 }}>
                    <div style={{ fontFamily: P.serif, fontSize: 20, lineHeight: 1.2, marginBottom: 4, fontWeight: 600, letterSpacing: '-0.01em' }}>Refonte produit Nova v2</div>
                    <div style={{ fontSize: 12, color: P.textMute, marginBottom: 14, lineHeight: 1.5 }}>UX onboarding mobile + design system Figma. 3 mois en 3 j/sem. Démarrage souhaité : 15 mai.</div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8, marginBottom: 14 }}>
                      {[['Montant', '23 400 €'], ['Durée', '3 mois'], ['TJM', '650 €'], ['Jalons', '3']].map(([l, v], i) => (
                        <div key={i} style={{ background: P.surfaceAlt, padding: '8px 10px', borderRadius: 6 }}>
                          <div style={{ fontSize: 9.5, color: P.textSubtle, letterSpacing: '0.06em', textTransform: 'uppercase', fontWeight: 700, marginBottom: 2 }}>{l}</div>
                          <div style={{ fontFamily: P.serif, fontSize: 14, fontWeight: 600 }}>{v}</div>
                        </div>
                      ))}
                    </div>
                    <div style={{ background: P.accentSoft, padding: '8px 12px', borderRadius: 6, fontSize: 11, color: P.accentDeep, fontWeight: 600, marginBottom: 14, display: 'flex', alignItems: 'center', gap: 6 }}>
                      <I_P.Shield size={11} /> Paiement séquestré jusqu'à validation des jalons
                    </div>
                    <div style={{ display: 'flex', gap: 6 }}>
                      <button style={{ flex: 1, background: P.accent, color: '#fff', border: 'none', padding: '9px', fontSize: 12.5, fontWeight: 700, borderRadius: 6, cursor: 'pointer' }}>✓ Accepter & démarrer</button>
                      <button style={{ background: '#fff', color: P.text, border: `1px solid ${P.border}`, padding: '9px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer' }}>Négocier</button>
                      <button style={{ background: '#fff', color: P.textMute, border: `1px solid ${P.border}`, padding: '9px 12px', fontSize: 12.5, fontWeight: 600, borderRadius: 6, cursor: 'pointer' }}>Refuser</button>
                    </div>
                  </div>
                </div>

                <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end', maxWidth: '70%', marginTop: 4 }}>
                  <Avatar initials="EM" bg="#0a7c4d" size={26} square />
                  <div>
                    <div style={{ background: '#fff', border: `1px solid ${P.border}`, padding: '10px 14px', borderRadius: '4px 12px 12px 12px', fontSize: 13, lineHeight: 1.5 }}>J'attends ton retour ! Si OK je bloque mon planning dès lundi 👌</div>
                    <div style={{ fontSize: 9.5, color: P.textSubtle, marginTop: 3, marginLeft: 4 }}>14:51</div>
                  </div>
                </div>
              </div>

              {/* Composer */}
              <div style={{ borderTop: `1px solid ${P.border}`, padding: '12px 22px', background: '#fff' }}>
                <div style={{ border: `1px solid ${P.border}`, borderRadius: 8, background: '#fff' }}>
                  <textarea placeholder="Écrire un message... (⌘ + ↵ pour envoyer)" style={{ width: '100%', border: 'none', outline: 'none', background: 'transparent', fontSize: 13, fontFamily: P.sans, padding: '10px 14px', resize: 'none', minHeight: 48, color: P.text }} />
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', borderTop: `1px solid ${P.border}` }}>
                    <I_P.Paperclip size={15} style={{ color: P.textMute, cursor: 'pointer' }} />
                    <I_P.Smiley size={15} style={{ color: P.textMute, cursor: 'pointer' }} />
                    <I_P.Briefcase size={15} style={{ color: P.textMute, cursor: 'pointer' }} title="Insérer proposition" />
                    <span style={{ flex: 1 }} />
                    <span style={{ fontSize: 10.5, color: P.textSubtle, fontFamily: P.mono }}>⌘ + ↵</span>
                    <button style={{ background: P.accent, color: '#fff', border: 'none', padding: '6px 14px', fontSize: 12, fontWeight: 700, borderRadius: 5, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}>Envoyer <I_P.Send size={11} /></button>
                  </div>
                </div>
              </div>
            </div>

            {/* Right context panel */}
            <div style={{ width: 260, borderLeft: `1px solid ${P.border}`, background: '#fff', padding: 18, flexShrink: 0, overflow: 'auto' }}>
              <div style={{ fontSize: 10.5, color: P.textSubtle, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase', marginBottom: 10 }}>Contexte projet</div>
              <div style={{ background: P.surfaceAlt, borderRadius: 8, padding: 12, marginBottom: 14 }}>
                <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 4 }}>Refonte produit Nova v2</div>
                <div style={{ fontSize: 11, color: P.textMute, marginBottom: 10 }}>Mission active · démarrée le 15 mai</div>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, marginBottom: 4 }}><span style={{ color: P.textMute }}>Avancement</span><span style={{ fontWeight: 700, fontFamily: P.mono }}>72%</span></div>
                <div style={{ height: 4, background: P.border, borderRadius: 2 }}>
                  <div style={{ width: '72%', height: '100%', background: P.accent, borderRadius: 2 }} />
                </div>
              </div>

              <div style={{ fontSize: 10.5, color: P.textSubtle, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase', marginBottom: 8 }}>Jalons</div>
              {[
                ['Discovery & audit', 'Validé · 28 mars', P.green, true],
                ['Wireframes v2', 'En cours · livr. 18 mai', P.accent, false],
                ['Design system', 'À démarrer', P.textSubtle, false],
              ].map(([l, s, c, done], i) => (
                <div key={i} style={{ display: 'flex', gap: 10, padding: '8px 0', borderTop: i > 0 ? `1px solid ${P.border}` : 'none' }}>
                  <div style={{ width: 14, height: 14, borderRadius: '50%', border: `1.5px solid ${c}`, background: done ? c : 'transparent', flexShrink: 0, marginTop: 2, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{done && <I_P.Check size={9} style={{ color: '#fff' }} />}</div>
                  <div>
                    <div style={{ fontSize: 12, fontWeight: 600 }}>{l}</div>
                    <div style={{ fontSize: 10.5, color: P.textMute }}>{s}</div>
                  </div>
                </div>
              ))}

              <div style={{ fontSize: 10.5, color: P.textSubtle, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase', marginTop: 18, marginBottom: 8 }}>Fichiers partagés · 8</div>
              {[['brief-v2.pdf', '2,4 Mo'], ['wireframes.fig', '12 Mo'], ['recherche.pdf', '4,1 Mo']].map(([n, s], i) => (
                <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '7px 0', borderTop: i > 0 ? `1px solid ${P.border}` : 'none' }}>
                  <div style={{ width: 26, height: 26, borderRadius: 5, background: P.accentSoft, color: P.accent, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}><I_P.File size={12} /></div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 11.5, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{n}</div>
                    <div style={{ fontSize: 10, color: P.textMute }}>{s}</div>
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

window.PlaceDashboard = PlaceDashboard;
window.PlaceFind = PlaceFind;
window.PlaceProfile = PlaceProfile;
window.PlaceMessages = PlaceMessages;
