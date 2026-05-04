// Direction 1 — Atelier Maison. Stripe-inspired. Indigo accent, structured grid, Inter Tight + Instrument Serif.
const { Icons } = window;

const MAISON = {
  bg: '#fafaf9',
  surface: '#ffffff',
  border: '#eceae3',
  borderStrong: '#d8d4c8',
  text: '#0a0a0a',
  textMute: '#6b6558',
  textSubtle: '#9a9385',
  accent: '#3a4ee0',
  accentSoft: '#eef0fe',
  accentDeep: '#1f2caa',
  pink: '#e8447b',
  pinkSoft: '#fde9f0',
  green: '#0e8a5f',
  greenSoft: '#e6f5ee',
  amber: '#b8721d',
  amberSoft: '#fbf0dc',
  serif: 'Instrument Serif, Georgia, serif',
  sans: '"Inter Tight", system-ui, sans-serif',
  mono: '"Geist Mono", ui-monospace, monospace',
};

// ─── Shared sidebar ────────────────────────────────────────────────
function MaisonSidebar({ active = 'home', role = 'enterprise' }) {
  const items = role === 'enterprise' ? [
    { id: 'home', icon: 'Home', label: 'Tableau de bord' },
    { id: 'msg', icon: 'Chat', label: 'Messages', badge: 3 },
    { id: 'proj', icon: 'Folder', label: 'Projets' },
    { id: 'jobs', icon: 'Briefcase', label: 'Jobs' },
    { id: 'team', icon: 'Users', label: 'Équipe' },
    { id: 'profile', icon: 'Building', label: 'Profil entreprise' },
  ] : [
    { id: 'home', icon: 'Home', label: 'Tableau de bord' },
    { id: 'msg', icon: 'Chat', label: 'Messages', badge: 2 },
    { id: 'proj', icon: 'Folder', label: 'Projets' },
    { id: 'opp', icon: 'Inbox', label: 'Opportunités' },
    { id: 'apply', icon: 'Doc', label: 'Mes candidatures' },
    { id: 'profile', icon: 'User', label: 'Profil prestataire' },
  ];
  const find = [
    { id: 'find-f', icon: 'Search', label: 'Trouver des freelances', match: 'find' },
    { id: 'find-a', icon: 'Layers', label: 'Trouver des agences' },
    { id: 'find-r', icon: 'Sparkle', label: 'Trouver des apporteurs' },
  ];
  const bottom = [
    { id: 'wallet', icon: 'Wallet', label: 'Portefeuille' },
    { id: 'invoice', icon: 'Doc', label: 'Factures' },
    { id: 'account', icon: 'Cog', label: 'Compte' },
  ];

  return (
    <aside style={{ width: 248, height: '100%', background: '#f6f3ec', borderRight: `1px solid ${MAISON.border}`, padding: '24px 0', display: 'flex', flexDirection: 'column', fontFamily: MAISON.sans, flexShrink: 0 }}>
      {/* Brand */}
      <div style={{ padding: '0 24px 28px', borderBottom: `1px solid ${MAISON.border}`, marginBottom: 20 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{ width: 28, height: 28, background: MAISON.text, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: MAISON.serif, fontStyle: 'italic', fontSize: 18, borderRadius: 2 }}>A</div>
          <div style={{ fontFamily: MAISON.serif, fontSize: 22, letterSpacing: '-0.02em' }}>Atelier</div>
        </div>
      </div>

      {/* Profile chip */}
      <div style={{ padding: '0 16px 20px' }}>
        <div style={{ background: '#fff', border: `1px solid ${MAISON.border}`, borderRadius: 4, padding: '12px 14px', display: 'flex', alignItems: 'center', gap: 12 }}>
          <div style={{ width: 36, height: 36, borderRadius: '50%', background: 'linear-gradient(135deg,#3a4ee0,#7c8df0)', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 13 }}>NV</div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 500, color: MAISON.text, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>Nova Studio</div>
            <div style={{ fontSize: 10, letterSpacing: '0.1em', textTransform: 'uppercase', color: MAISON.accent, fontWeight: 600 }}>{role === 'enterprise' ? 'Entreprise' : 'Prestataire'}</div>
          </div>
        </div>
      </div>

      <nav style={{ padding: '0 12px', flex: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
        {items.map((it) => {
          const I = Icons[it.icon];
          const isActive = active === it.id || active === it.match;
          return (
            <div key={it.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '9px 12px', borderRadius: 4, fontSize: 14, fontWeight: 500, color: isActive ? MAISON.text : MAISON.textMute, background: isActive ? '#fff' : 'transparent', border: isActive ? `1px solid ${MAISON.border}` : '1px solid transparent', cursor: 'pointer' }}>
              <I size={17} stroke={isActive ? 1.8 : 1.5} />
              <span style={{ flex: 1 }}>{it.label}</span>
              {it.badge && <span style={{ background: MAISON.accent, color: '#fff', fontSize: 10, fontWeight: 600, padding: '1px 6px', borderRadius: 8 }}>{it.badge}</span>}
            </div>
          );
        })}

        <div style={{ height: 1, background: MAISON.border, margin: '12px 12px' }} />
        <div style={{ fontSize: 10, letterSpacing: '0.15em', textTransform: 'uppercase', color: MAISON.textSubtle, padding: '6px 12px', fontWeight: 600 }}>Découvrir</div>
        {find.map((it) => {
          const I = Icons[it.icon];
          const isActive = active === it.id || active === it.match;
          return (
            <div key={it.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '9px 12px', borderRadius: 4, fontSize: 14, fontWeight: 500, color: isActive ? MAISON.text : MAISON.textMute, background: isActive ? '#fff' : 'transparent', border: isActive ? `1px solid ${MAISON.border}` : '1px solid transparent', cursor: 'pointer' }}>
              <I size={17} stroke={isActive ? 1.8 : 1.5} />
              <span>{it.label}</span>
            </div>
          );
        })}
      </nav>

      <div style={{ padding: '0 12px', display: 'flex', flexDirection: 'column', gap: 2, borderTop: `1px solid ${MAISON.border}`, paddingTop: 12 }}>
        {bottom.map((it) => {
          const I = Icons[it.icon];
          return (
            <div key={it.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '9px 12px', borderRadius: 4, fontSize: 13, fontWeight: 500, color: MAISON.textMute, cursor: 'pointer' }}>
              <I size={16} stroke={1.5} />
              <span>{it.label}</span>
            </div>
          );
        })}
      </div>
    </aside>
  );
}

// ─── Shared topbar ─────────────────────────────────────────────────
function MaisonTopbar({ search = 'Rechercher dans Atelier...' }) {
  return (
    <div style={{ height: 60, borderBottom: `1px solid ${MAISON.border}`, background: MAISON.bg, display: 'flex', alignItems: 'center', padding: '0 32px', gap: 20, flexShrink: 0 }}>
      <div style={{ flex: 1, maxWidth: 480, display: 'flex', alignItems: 'center', gap: 10, background: '#fff', border: `1px solid ${MAISON.border}`, borderRadius: 4, padding: '8px 14px' }}>
        <Icons.Search size={15} style={{ color: MAISON.textMute }} />
        <input placeholder={search} style={{ flex: 1, border: 'none', outline: 'none', background: 'transparent', fontSize: 13, color: MAISON.text, fontFamily: MAISON.sans }} readOnly />
        <span style={{ fontFamily: MAISON.mono, fontSize: 10, color: MAISON.textSubtle, padding: '2px 5px', border: `1px solid ${MAISON.border}`, borderRadius: 3 }}>⌘K</span>
      </div>
      <div style={{ flex: 1 }} />
      <button style={{ background: '#fff', border: `1px solid ${MAISON.border}`, padding: '7px 12px', fontSize: 12, fontWeight: 600, color: MAISON.text, borderRadius: 4, fontFamily: MAISON.sans, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
        <Icons.Sparkle size={13} />
        Inviter
      </button>
      <div style={{ position: 'relative', cursor: 'pointer' }}>
        <Icons.Bell size={18} style={{ color: MAISON.textMute }} />
        <span style={{ position: 'absolute', top: -2, right: -2, width: 7, height: 7, borderRadius: '50%', background: MAISON.pink, border: '1.5px solid ' + MAISON.bg }} />
      </div>
      <div style={{ width: 32, height: 32, borderRadius: '50%', background: 'linear-gradient(135deg,#3a4ee0,#7c8df0)', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 12 }}>NV</div>
    </div>
  );
}

// ─── Mobile preview frame ──────────────────────────────────────────
function MobileFrame({ children, label }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10 }}>
      <div style={{ width: 320, height: 660, background: '#1a1a1a', borderRadius: 32, padding: 6, boxShadow: '0 12px 32px rgba(0,0,0,0.18), 0 2px 6px rgba(0,0,0,0.08)' }}>
        <div style={{ width: '100%', height: '100%', borderRadius: 26, overflow: 'hidden', position: 'relative', background: MAISON.bg }}>
          {/* notch */}
          <div style={{ position: 'absolute', top: 8, left: '50%', transform: 'translateX(-50%)', width: 90, height: 22, background: '#1a1a1a', borderRadius: 12, zIndex: 10 }} />
          {/* status bar */}
          <div style={{ position: 'absolute', top: 0, left: 0, right: 0, height: 36, padding: '0 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontSize: 11, fontWeight: 600, fontFamily: MAISON.sans, zIndex: 5, color: MAISON.text }}>
            <span style={{ fontFeatureSettings: '"tnum"' }}>9:41</span>
            <span style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
              <span style={{ display: 'inline-block', width: 16, height: 10, border: `1px solid ${MAISON.text}`, borderRadius: 2, position: 'relative' }}>
                <span style={{ display: 'block', position: 'absolute', inset: 1, background: MAISON.text, width: '70%', borderRadius: 1 }} />
              </span>
            </span>
          </div>
          <div style={{ paddingTop: 36, height: '100%', overflow: 'hidden' }}>{children}</div>
        </div>
      </div>
      <div style={{ fontFamily: MAISON.mono, fontSize: 10, color: MAISON.textSubtle, letterSpacing: '0.1em', textTransform: 'uppercase' }}>{label}</div>
    </div>
  );
}

// ═══ DASHBOARD ═════════════════════════════════════════════════════
function MaisonDashboard() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: MAISON.bg, fontFamily: MAISON.sans, color: MAISON.text }}>
      <MaisonSidebar active="home" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <MaisonTopbar />
        <div style={{ flex: 1, overflow: 'hidden', padding: '32px 40px' }}>
          {/* Header */}
          <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', marginBottom: 28, paddingBottom: 24, borderBottom: `1px solid ${MAISON.border}` }}>
            <div>
              <div style={{ fontFamily: MAISON.mono, fontSize: 11, color: MAISON.textMute, letterSpacing: '0.15em', textTransform: 'uppercase', marginBottom: 8 }}>Jeudi 1ᵉʳ mai · 09:41</div>
              <h1 style={{ fontFamily: MAISON.serif, fontSize: 44, lineHeight: 1.05, margin: 0, fontWeight: 400, letterSpacing: '-0.02em' }}>
                Bonjour, <em style={{ color: MAISON.accent }}>Nova</em>.
              </h1>
            </div>
            <div style={{ display: 'flex', gap: 10 }}>
              <button style={{ background: '#fff', border: `1px solid ${MAISON.border}`, padding: '10px 16px', fontSize: 13, fontWeight: 500, color: MAISON.text, borderRadius: 4, cursor: 'pointer', fontFamily: MAISON.sans }}>Voir le rapport</button>
              <button style={{ background: MAISON.text, border: 'none', padding: '10px 16px', fontSize: 13, fontWeight: 500, color: '#fff', borderRadius: 4, cursor: 'pointer', fontFamily: MAISON.sans, display: 'flex', alignItems: 'center', gap: 6 }}>
                <Icons.Plus size={14} />
                Nouveau projet
              </button>
            </div>
          </div>

          {/* Stat row */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 32 }}>
            {[
              { label: 'Volume engagé', val: '47 320', unit: '€', delta: '+12,4%', deltaColor: MAISON.green, sub: 'ce mois' },
              { label: 'Missions actives', val: '12', unit: '', delta: '+3', deltaColor: MAISON.green, sub: 'vs avril' },
              { label: 'Candidatures reçues', val: '34', unit: '', delta: '8 nouvelles', deltaColor: MAISON.accent, sub: 'à examiner' },
              { label: 'Apports en cours', val: '5', unit: '', delta: '2 en attente', deltaColor: MAISON.amber, sub: 'de réponse' },
            ].map((s, i) => (
              <div key={i} style={{ background: '#fff', border: `1px solid ${MAISON.border}`, borderRadius: 4, padding: 20 }}>
                <div style={{ fontSize: 11, letterSpacing: '0.12em', textTransform: 'uppercase', color: MAISON.textMute, fontWeight: 600, marginBottom: 12 }}>{s.label}</div>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 4, marginBottom: 8 }}>
                  <span style={{ fontFamily: MAISON.serif, fontSize: 36, fontWeight: 400, letterSpacing: '-0.02em', lineHeight: 1 }}>{s.val}</span>
                  {s.unit && <span style={{ fontFamily: MAISON.serif, fontSize: 22, color: MAISON.textMute }}>{s.unit}</span>}
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12 }}>
                  <span style={{ color: s.deltaColor, fontWeight: 600 }}>{s.delta}</span>
                  <span style={{ color: MAISON.textMute }}>{s.sub}</span>
                </div>
              </div>
            ))}
          </div>

          {/* Content grid */}
          <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr', gap: 16 }}>
            {/* Active missions */}
            <div style={{ background: '#fff', border: `1px solid ${MAISON.border}`, borderRadius: 4 }}>
              <div style={{ padding: '16px 20px', borderBottom: `1px solid ${MAISON.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 10 }}>
                  <h3 style={{ fontFamily: MAISON.serif, fontSize: 22, margin: 0, fontWeight: 400 }}>Missions en cours</h3>
                  <span style={{ fontFamily: MAISON.mono, fontSize: 11, color: MAISON.textMute }}>12 actives</span>
                </div>
                <button style={{ background: 'none', border: 'none', fontSize: 12, color: MAISON.accent, fontWeight: 600, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4 }}>Tout voir <Icons.ArrowRight size={12} /></button>
              </div>
              <div>
                {[
                  { name: 'Refonte site corporate', client: 'Lemon Aviation', amount: '12 400 €', progress: 72, dl: '15 mai', tag: 'Web', tagColor: MAISON.accent, p: 'EM', pBg: '#0e8a5f' },
                  { name: 'Brand identity Q2', client: 'Cobalt Studio', amount: '8 200 €', progress: 45, dl: '28 mai', tag: 'Brand', tagColor: MAISON.pink, p: 'JP', pBg: '#e85d4a' },
                  { name: 'Audit SEO technique', client: 'Maison Vega', amount: '3 600 €', progress: 90, dl: '08 mai', tag: 'SEO', tagColor: MAISON.amber, p: 'TR', pBg: '#3a4ee0' },
                ].map((m, i) => (
                  <div key={i} style={{ padding: '18px 20px', borderBottom: i < 2 ? `1px solid ${MAISON.border}` : 'none', display: 'grid', gridTemplateColumns: '32px 1fr auto auto', gap: 14, alignItems: 'center' }}>
                    <div style={{ width: 32, height: 32, borderRadius: '50%', background: m.pBg, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 11 }}>{m.p}</div>
                    <div style={{ minWidth: 0 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4 }}>
                        <span style={{ fontSize: 14, fontWeight: 500 }}>{m.name}</span>
                        <span style={{ fontSize: 10, padding: '1px 6px', background: m.tagColor + '15', color: m.tagColor, fontWeight: 600, borderRadius: 2, letterSpacing: '0.05em', textTransform: 'uppercase' }}>{m.tag}</span>
                      </div>
                      <div style={{ fontSize: 12, color: MAISON.textMute }}>{m.client} · échéance {m.dl}</div>
                    </div>
                    <div style={{ textAlign: 'right' }}>
                      <div style={{ fontFamily: MAISON.serif, fontSize: 18, lineHeight: 1 }}>{m.amount}</div>
                      <div style={{ fontSize: 11, color: MAISON.textMute, fontFamily: MAISON.mono }}>{m.progress}% complété</div>
                    </div>
                    <div style={{ width: 80, height: 4, background: MAISON.border, borderRadius: 2, overflow: 'hidden' }}>
                      <div style={{ width: m.progress + '%', height: '100%', background: MAISON.text }} />
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Side column */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {/* Inbox preview */}
              <div style={{ background: '#fff', border: `1px solid ${MAISON.border}`, borderRadius: 4 }}>
                <div style={{ padding: '14px 18px', borderBottom: `1px solid ${MAISON.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <h3 style={{ fontFamily: MAISON.serif, fontSize: 18, margin: 0, fontWeight: 400 }}>Messages</h3>
                  <span style={{ fontSize: 10, padding: '2px 7px', background: MAISON.accent, color: '#fff', borderRadius: 8, fontWeight: 600 }}>3</span>
                </div>
                {[
                  { name: 'Élise Marchand', last: 'Tu as vu le brief mis à jour ?', time: '14 min', unread: true, p: 'EM', pBg: '#0e8a5f' },
                  { name: 'Julien Petit', last: 'Voici la v2 des wireframes —', time: '1 h', unread: true, p: 'JP', pBg: '#e85d4a' },
                  { name: 'Théo Renaud', last: 'Audit terminé, rapport ci-joint', time: '3 h', unread: false, p: 'TR', pBg: '#3a4ee0' },
                ].map((m, i) => (
                  <div key={i} style={{ padding: '12px 18px', borderBottom: i < 2 ? `1px solid ${MAISON.border}` : 'none', display: 'flex', gap: 10, alignItems: 'center' }}>
                    <div style={{ width: 28, height: 28, borderRadius: '50%', background: m.pBg, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 10 }}>{m.p}</div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 2 }}>
                        <span style={{ fontSize: 13, fontWeight: m.unread ? 600 : 500 }}>{m.name}</span>
                        <span style={{ fontSize: 10, color: MAISON.textMute, fontFamily: MAISON.mono }}>{m.time}</span>
                      </div>
                      <div style={{ fontSize: 12, color: MAISON.textMute, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{m.last}</div>
                    </div>
                    {m.unread && <span style={{ width: 6, height: 6, borderRadius: '50%', background: MAISON.accent }} />}
                  </div>
                ))}
              </div>

              {/* CTA card */}
              <div style={{ background: MAISON.text, color: '#fff', padding: 24, borderRadius: 4, position: 'relative', overflow: 'hidden' }}>
                <div style={{ position: 'absolute', top: -20, right: -20, width: 120, height: 120, borderRadius: '50%', background: 'radial-gradient(circle, rgba(58,78,224,0.4), transparent 70%)' }} />
                <div style={{ fontFamily: MAISON.mono, fontSize: 10, letterSpacing: '0.15em', color: '#7c8df0', marginBottom: 12, textTransform: 'uppercase' }}>Suggestion</div>
                <div style={{ fontFamily: MAISON.serif, fontSize: 22, lineHeight: 1.2, marginBottom: 12, fontStyle: 'italic' }}>"4 prestataires correspondent à ton dernier brief."</div>
                <button style={{ background: '#fff', color: MAISON.text, border: 'none', padding: '8px 14px', fontSize: 12, fontWeight: 600, borderRadius: 3, cursor: 'pointer', fontFamily: MAISON.sans, display: 'inline-flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
                  Voir les profils <Icons.ArrowRight size={12} />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

window.MaisonDashboard = MaisonDashboard;
window.MaisonSidebar = MaisonSidebar;
window.MaisonTopbar = MaisonTopbar;
window.MAISON_TOKENS = MAISON;
