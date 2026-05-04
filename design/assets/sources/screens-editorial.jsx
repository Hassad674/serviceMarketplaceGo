// Direction A — "Atelier Éditorial"
// Ivoire warm BG, terracotta accent, Instrument Serif display + Geist sans + Geist Mono.
// Sensation : magazine premium, sobre, mature, européen.

const eA = {
  bg: '#F5F1EA',
  card: '#FAF7F2',
  cardEdge: '#E5DDC9',
  ink: '#1A1612',
  tabac: '#594D3D',
  mute: '#857A6A',
  rust: '#C2410C',
  rustSoft: '#F1E0D2',
  sapin: '#3F6B4F',
  sapinSoft: '#DCE5DD',
  serif: "'Instrument Serif', serif",
  sans: "'Geist', sans-serif",
  mono: "'Geist Mono', monospace",
};

function SidebarEditorial({ active = 'dashboard', role = 'PRESTATAIRE', name = 'Léa Marchand' }) {
  const items = [
    { id: 'dashboard', label: 'Tableau de bord', n: '01' },
    { id: 'messages', label: 'Messages', n: '02', badge: 3 },
    { id: 'projects', label: 'Projets', n: '03' },
    { id: 'opportunities', label: 'Opportunités', n: '04' },
    { id: 'applications', label: 'Mes candidatures', n: '05' },
    { id: 'profile', label: 'Profil prestataire', n: '06' },
    { id: 'payment', label: 'Infos paiement', n: '07' },
    { id: 'wallet', label: 'Portefeuille', n: '08' },
    { id: 'invoices', label: 'Factures', n: '09' },
    { id: 'account', label: 'Compte', n: '10' },
  ];
  return (
    <aside style={{
      width: 248,
      background: eA.bg,
      borderRight: `1px solid ${eA.cardEdge}`,
      padding: '24px 0 24px 0',
      display: 'flex',
      flexDirection: 'column',
      fontFamily: eA.sans,
      flexShrink: 0,
    }}>
      <div style={{ padding: '0 24px', marginBottom: 28, display: 'flex', alignItems: 'baseline', gap: 6 }}>
        <span style={{ fontFamily: eA.serif, fontSize: 28, color: eA.ink, lineHeight: 1, letterSpacing: '-0.01em' }}>Atelier</span>
        <span style={{ color: eA.rust, fontFamily: eA.serif, fontSize: 28, fontStyle: 'italic', lineHeight: 1 }}>.</span>
      </div>

      <div style={{ padding: '0 16px 0 16px', marginBottom: 20 }}>
        <div style={{
          background: eA.card,
          border: `1px solid ${eA.cardEdge}`,
          borderRadius: 8,
          padding: '12px 14px',
          display: 'flex',
          alignItems: 'center',
          gap: 10,
        }}>
          <div style={{
            width: 34, height: 34, borderRadius: '50%',
            background: eA.ink, color: eA.bg,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontFamily: eA.serif, fontSize: 16,
          }}>{name[0]}</div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 500, color: eA.ink, lineHeight: 1.2, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{name}</div>
            <div style={{ fontFamily: eA.mono, fontSize: 9, color: eA.mute, letterSpacing: '0.1em', marginTop: 2 }}>{role}</div>
          </div>
        </div>

        <button style={{
          marginTop: 12, width: '100%',
          background: eA.ink, color: eA.bg,
          border: 'none', borderRadius: 6,
          padding: '10px 14px',
          fontFamily: eA.sans, fontSize: 12.5, fontWeight: 500,
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          cursor: 'pointer',
        }}>
          <span>Apporter une affaire</span>
          <span style={{ fontFamily: eA.mono, fontSize: 11, opacity: 0.6 }}>+</span>
        </button>
      </div>

      <nav style={{ flex: 1, padding: '0 8px' }}>
        {items.map(it => {
          const isActive = it.id === active;
          return (
            <div key={it.id} style={{
              padding: '8px 14px',
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              borderRadius: 6,
              cursor: 'pointer',
              background: isActive ? eA.rustSoft : 'transparent',
              color: isActive ? eA.rust : eA.tabac,
              fontWeight: isActive ? 500 : 400,
              fontSize: 13,
              marginBottom: 1,
              position: 'relative',
            }}>
              <span style={{ fontFamily: eA.mono, fontSize: 9.5, color: isActive ? eA.rust : eA.mute, letterSpacing: '0.05em' }}>{it.n}</span>
              <span style={{ flex: 1 }}>{it.label}</span>
              {it.badge && (
                <span style={{ fontFamily: eA.mono, fontSize: 10, background: eA.ink, color: eA.bg, borderRadius: 999, padding: '1px 6px', minWidth: 16, textAlign: 'center' }}>{it.badge}</span>
              )}
            </div>
          );
        })}
      </nav>

      <div style={{ padding: '16px 24px 0', borderTop: `1px solid ${eA.cardEdge}`, marginTop: 16 }}>
        <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.08em' }}>v 2.4 · Paris</div>
      </div>
    </aside>
  );
}

function HeaderEditorial({ q = 'Recherche…' }) {
  return (
    <header style={{
      borderBottom: `1px solid ${eA.cardEdge}`,
      padding: '14px 32px',
      display: 'flex',
      alignItems: 'center',
      gap: 16,
      background: eA.bg,
    }}>
      <div style={{ flex: 1, maxWidth: 480, position: 'relative' }}>
        <div style={{ position: 'absolute', left: 14, top: '50%', transform: 'translateY(-50%)', fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.08em' }}>⌕</div>
        <input value={q} readOnly style={{
          width: '100%',
          background: eA.card,
          border: `1px solid ${eA.cardEdge}`,
          borderRadius: 6,
          padding: '8px 14px 8px 34px',
          fontFamily: eA.sans, fontSize: 13,
          color: eA.tabac,
          outline: 'none',
        }} />
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.08em', textTransform: 'uppercase' }}>FR · €</div>
        <button style={{ background: eA.rust, color: eA.bg, border: 'none', padding: '7px 14px', borderRadius: 4, fontFamily: eA.sans, fontSize: 12, fontWeight: 500, display: 'flex', alignItems: 'center', gap: 6 }}>
          <span>✦</span> Premium
        </button>
        <div style={{ width: 1, height: 22, background: eA.cardEdge }} />
        <div style={{ position: 'relative' }}>
          <span style={{ fontSize: 16 }}>◔</span>
          <span style={{ position: 'absolute', top: -4, right: -6, background: eA.rust, color: eA.bg, fontFamily: eA.mono, fontSize: 9, borderRadius: 999, padding: '1px 5px' }}>10</span>
        </div>
        <div style={{ width: 30, height: 30, borderRadius: '50%', background: eA.ink, color: eA.bg, display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: eA.serif, fontSize: 13 }}>L</div>
      </div>
    </header>
  );
}

function ShellEditorial({ active, children }) {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: eA.bg, fontFamily: eA.sans, color: eA.ink, overflow: 'hidden' }}>
      <SidebarEditorial active={active} />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <HeaderEditorial />
        <div style={{ flex: 1, overflow: 'hidden' }}>{children}</div>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────
// DASHBOARD
// ─────────────────────────────────────────────
function DashboardEditorial() {
  return (
    <ShellEditorial active="dashboard">
      <div style={{ padding: '40px 56px', height: '100%', overflow: 'hidden' }}>
        {/* Hero — editorial */}
        <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 40, alignItems: 'end', marginBottom: 40, paddingBottom: 32, borderBottom: `1px solid ${eA.cardEdge}` }}>
          <div>
            <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.12em', textTransform: 'uppercase', marginBottom: 14 }}>Vendredi 1 mai · 16:38</div>
            <div style={{ fontFamily: eA.serif, fontSize: 56, lineHeight: 1, letterSpacing: '-0.01em' }}>
              Bonjour, <span style={{ fontStyle: 'italic', color: eA.rust }}>Léa</span>.
            </div>
            <div style={{ fontFamily: eA.serif, fontSize: 22, fontStyle: 'italic', color: eA.tabac, marginTop: 12 }}>
              3 conversations à reprendre, 2 propositions à valider.
            </div>
          </div>
          <div style={{ background: eA.card, border: `1px solid ${eA.cardEdge}`, borderRadius: 8, padding: 20 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
              <div style={{ fontFamily: eA.mono, fontSize: 10, letterSpacing: '0.12em', textTransform: 'uppercase', color: eA.mute }}>À faire aujourd'hui</div>
              <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.rust }}>5 éléments</div>
            </div>
            {[
              { t: 'Valider la proposition de Hassan', m: '5 363 €', urgent: true },
              { t: 'Livrer le jalon · Refonte brand Coddo', m: '1 200 €' },
              { t: 'Répondre à 3 messages', m: '' },
            ].map((it, i) => (
              <div key={i} style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', padding: '8px 0', borderTop: i ? `1px solid ${eA.cardEdge}` : 'none' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span style={{ width: 6, height: 6, borderRadius: '50%', background: it.urgent ? eA.rust : eA.tabac }} />
                  <span style={{ fontSize: 13 }}>{it.t}</span>
                </div>
                <span style={{ fontFamily: eA.mono, fontSize: 12, color: eA.tabac }}>{it.m}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Stats */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 20, marginBottom: 40 }}>
          {[
            { k: 'Revenus du mois', v: '8 240', u: '€', d: '↗ +24% vs avril' },
            { k: 'Missions actives', v: '3', u: '', d: 'sur 5 max' },
            { k: 'Taux d\'acceptation', v: '82', u: '%', d: 'des 22 candidatures' },
            { k: 'Note moyenne', v: '4.9', u: '/5', d: 'sur 17 avis' },
          ].map((s, i) => (
            <div key={i} style={{ background: eA.card, border: `1px solid ${eA.cardEdge}`, borderRadius: 8, padding: 20 }}>
              <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 14 }}>{s.k}</div>
              <div style={{ display: 'flex', alignItems: 'baseline', gap: 4 }}>
                <span style={{ fontFamily: eA.serif, fontSize: 44, lineHeight: 1, letterSpacing: '-0.02em' }}>{s.v}</span>
                <span style={{ fontFamily: eA.serif, fontSize: 22, color: eA.tabac }}>{s.u}</span>
              </div>
              <div style={{ fontSize: 11.5, color: eA.tabac, marginTop: 10 }}>{s.d}</div>
            </div>
          ))}
        </div>

        {/* Two columns: pipeline + opportunities */}
        <div style={{ display: 'grid', gridTemplateColumns: '1.3fr 1fr', gap: 24 }}>
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 16 }}>
              <div style={{ fontFamily: eA.serif, fontSize: 24 }}>Missions en cours</div>
              <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute }}>03 — voir tout</div>
            </div>
            {[
              { c: 'Coddo Studio', t: 'Refonte de l\'identité de marque', s: 'Active', m: '7 364', p: 60 },
              { c: 'Maison Fauve', t: 'Direction artistique du site', s: 'En séquestre', m: '3 213', p: 30 },
            ].map((m, i) => (
              <div key={i} style={{ background: eA.card, border: `1px solid ${eA.cardEdge}`, borderRadius: 8, padding: '18px 22px', marginBottom: 12 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 6 }}>
                  <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase' }}>{m.c}</div>
                  <div style={{ fontFamily: eA.mono, fontSize: 13, color: eA.ink }}>{m.m} €</div>
                </div>
                <div style={{ fontFamily: eA.serif, fontSize: 22, marginBottom: 12 }}>{m.t}</div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                  <div style={{ flex: 1, height: 2, background: eA.cardEdge, position: 'relative' }}>
                    <div style={{ position: 'absolute', inset: 0, width: m.p + '%', background: eA.rust }} />
                  </div>
                  <span style={{ fontFamily: eA.mono, fontSize: 11, color: eA.tabac }}>{m.p}%</span>
                  <span style={{ fontSize: 11, padding: '2px 8px', background: m.s === 'Active' ? eA.sapinSoft : eA.rustSoft, color: m.s === 'Active' ? eA.sapin : eA.rust, borderRadius: 999 }}>{m.s}</span>
                </div>
              </div>
            ))}
          </div>

          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 16 }}>
              <div style={{ fontFamily: eA.serif, fontSize: 24 }}>Pour vous</div>
              <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute }}>04</div>
            </div>
            {[
              { co: 'Studio Bonsoir', t: 'Direction artistique', b: '8 — 12k €' },
              { co: 'Coopérative Numa', t: 'Refonte du système de design', b: '15 — 20k €' },
              { co: 'Atelier Belleville', t: 'Identité visuelle complète', b: '6 — 10k €' },
            ].map((o, i) => (
              <div key={i} style={{ borderBottom: `1px solid ${eA.cardEdge}`, padding: '14px 0' }}>
                <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase' }}>{o.co}</div>
                <div style={{ fontFamily: eA.serif, fontSize: 19, marginTop: 4 }}>{o.t}</div>
                <div style={{ fontFamily: eA.mono, fontSize: 12, color: eA.rust, marginTop: 6 }}>{o.b}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </ShellEditorial>
  );
}

// ─────────────────────────────────────────────
// FIND FREELANCERS
// ─────────────────────────────────────────────
function FindFreelancersEditorial() {
  const people = [
    { i: 'CL', n: 'Camille Lefèvre', t: 'Direction artistique & systèmes de marque', loc: 'Paris', tjm: 720, rate: 4.9, n2: 28, av: 'now', tags: ['Brand', 'Direction artistique', 'Print'] },
    { i: 'YO', n: 'Yann Orhant', t: 'Designer produit senior · ex-Doctolib', loc: 'Nantes', tjm: 850, rate: 4.8, n2: 41, av: 'now', tags: ['Design produit', 'UX research'] },
    { i: 'AB', n: 'Aïssa Benali', t: 'Identité de marque & motion', loc: 'Lyon', tjm: 600, rate: 5.0, n2: 14, av: 'soon', tags: ['Brand', 'Motion'] },
    { i: 'NM', n: 'Noor Maalouf', t: 'Stratège marque · luxe & artisanat', loc: 'Paris', tjm: 950, rate: 4.9, n2: 22, av: 'now', tags: ['Stratégie', 'Brand'] },
    { i: 'EP', n: 'Élisa Park', t: 'Design éditorial & typographie', loc: 'Bruxelles', tjm: 680, rate: 4.7, n2: 19, av: 'soon', tags: ['Éditorial', 'Type'] },
    { i: 'MR', n: 'Mathieu Roussel', t: 'Webflow & direction technique', loc: 'Toulouse', tjm: 720, rate: 4.9, n2: 33, av: 'now', tags: ['Webflow', 'Dev'] },
  ];
  return (
    <ShellEditorial active="find">
      <div style={{ padding: '40px 56px', height: '100%', overflow: 'hidden' }}>
        {/* Editorial header */}
        <div style={{ marginBottom: 32, paddingBottom: 24, borderBottom: `1px solid ${eA.cardEdge}` }}>
          <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.12em', textTransform: 'uppercase', marginBottom: 12 }}>Annuaire · 1 240 prestataires vérifiés</div>
          <div style={{ fontFamily: eA.serif, fontSize: 64, lineHeight: 0.95, letterSpacing: '-0.02em' }}>
            Trouvez quelqu'un de <span style={{ fontStyle: 'italic', color: eA.rust }}>juste</span>.
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '260px 1fr', gap: 40 }}>
          {/* Filters */}
          <aside>
            <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.12em', textTransform: 'uppercase', marginBottom: 12 }}>Filtres · 1 240 résultats</div>

            {[
              { l: 'Disponibilité', items: ['Maintenant', 'Sous 2 semaines', 'Tous'], active: 0 },
              { l: 'Mode de travail', items: ['À distance', 'Sur site', 'Hybride'], active: 2 },
            ].map((g, i) => (
              <div key={i} style={{ marginBottom: 22 }}>
                <div style={{ fontSize: 12, fontWeight: 500, marginBottom: 10 }}>{g.l}</div>
                <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                  {g.items.map((it, j) => (
                    <span key={j} style={{
                      fontSize: 11.5,
                      padding: '5px 11px',
                      borderRadius: 999,
                      border: `1px solid ${j === g.active ? eA.ink : eA.cardEdge}`,
                      background: j === g.active ? eA.ink : 'transparent',
                      color: j === g.active ? eA.bg : eA.tabac,
                    }}>{it}</span>
                  ))}
                </div>
              </div>
            ))}

            <div style={{ marginBottom: 22 }}>
              <div style={{ fontSize: 12, fontWeight: 500, marginBottom: 10 }}>TJM (€)</div>
              <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                <div style={{ flex: 1, background: eA.card, border: `1px solid ${eA.cardEdge}`, padding: '6px 10px', fontFamily: eA.mono, fontSize: 12, borderRadius: 4 }}>400</div>
                <span style={{ color: eA.mute }}>—</span>
                <div style={{ flex: 1, background: eA.card, border: `1px solid ${eA.cardEdge}`, padding: '6px 10px', fontFamily: eA.mono, fontSize: 12, borderRadius: 4 }}>1 200</div>
              </div>
              <div style={{ height: 2, background: eA.cardEdge, marginTop: 14, position: 'relative' }}>
                <div style={{ position: 'absolute', left: '20%', right: '15%', top: 0, bottom: 0, background: eA.rust }} />
                <div style={{ position: 'absolute', left: '20%', top: -4, width: 10, height: 10, borderRadius: '50%', background: eA.ink, transform: 'translateX(-50%)' }} />
                <div style={{ position: 'absolute', left: '85%', top: -4, width: 10, height: 10, borderRadius: '50%', background: eA.ink, transform: 'translateX(-50%)' }} />
              </div>
            </div>

            <div style={{ marginBottom: 22 }}>
              <div style={{ fontSize: 12, fontWeight: 500, marginBottom: 10 }}>Expertise</div>
              {['Design & UI/UX', 'Direction artistique', 'Développement', 'Marketing & growth', 'Rédaction', 'Data & IA'].map((e, i) => (
                <div key={e} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 0', fontSize: 12.5, color: eA.tabac }}>
                  <span style={{ width: 14, height: 14, border: `1px solid ${eA.tabac}`, borderRadius: 3, background: i < 2 ? eA.ink : 'transparent', position: 'relative' }}>
                    {i < 2 && <span style={{ color: eA.bg, position: 'absolute', top: -2, left: 1, fontSize: 11 }}>✓</span>}
                  </span>
                  <span>{e}</span>
                </div>
              ))}
            </div>
          </aside>

          {/* Grid */}
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 20 }}>
              <div style={{ fontSize: 13, color: eA.tabac }}>Affichage · <span style={{ color: eA.ink, fontWeight: 500 }}>6 sur 1 240</span></div>
              <div style={{ display: 'flex', gap: 14, fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.05em' }}>
                <span style={{ color: eA.ink }}>Pertinence</span>
                <span>TJM ↑</span>
                <span>Note</span>
                <span>Récents</span>
              </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 14 }}>
              {people.map((p, i) => (
                <div key={i} style={{ background: eA.card, border: `1px solid ${eA.cardEdge}`, borderRadius: 8, padding: 20, position: 'relative' }}>
                  {/* Index */}
                  <div style={{ position: 'absolute', top: 14, right: 16, fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em' }}>0{i+1}</div>
                  <div style={{
                    width: 56, height: 56, borderRadius: 6,
                    background: i % 2 ? eA.ink : eA.rust,
                    color: eA.bg,
                    fontFamily: eA.serif, fontSize: 24,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    marginBottom: 14,
                  }}>{p.i}</div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontFamily: eA.mono, fontSize: 10, color: p.av === 'now' ? eA.sapin : eA.tabac, letterSpacing: '0.08em', textTransform: 'uppercase', marginBottom: 6 }}>
                    <span style={{ width: 6, height: 6, borderRadius: '50%', background: p.av === 'now' ? eA.sapin : eA.rust }} />
                    {p.av === 'now' ? 'Disponible' : 'Sous 2 semaines'}
                  </div>
                  <div style={{ fontFamily: eA.serif, fontSize: 22, lineHeight: 1.1, marginBottom: 4 }}>{p.n}</div>
                  <div style={{ fontSize: 12.5, color: eA.tabac, lineHeight: 1.45, marginBottom: 14, minHeight: 36 }}>{p.t}</div>
                  <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap', marginBottom: 14 }}>
                    {p.tags.slice(0, 2).map(t => (
                      <span key={t} style={{ fontSize: 11, padding: '3px 8px', border: `1px solid ${eA.cardEdge}`, borderRadius: 999, color: eA.tabac }}>{t}</span>
                    ))}
                  </div>
                  <div style={{ borderTop: `1px solid ${eA.cardEdge}`, paddingTop: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                      <div style={{ fontFamily: eA.mono, fontSize: 9, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase' }}>TJM</div>
                      <div style={{ fontFamily: eA.mono, fontSize: 14, color: eA.ink, marginTop: 2 }}>{p.tjm} €</div>
                    </div>
                    <div style={{ textAlign: 'right' }}>
                      <div style={{ fontFamily: eA.mono, fontSize: 9, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase' }}>Note</div>
                      <div style={{ fontFamily: eA.mono, fontSize: 14, color: eA.ink, marginTop: 2 }}>{p.rate} <span style={{ color: eA.mute, fontSize: 11 }}>· {p.n2}</span></div>
                    </div>
                    <div style={{ marginLeft: 'auto', fontSize: 18, color: eA.ink }}>→</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </ShellEditorial>
  );
}

// ─────────────────────────────────────────────
// PROFILE
// ─────────────────────────────────────────────
function ProfileEditorial() {
  return (
    <ShellEditorial active="profile">
      <div style={{ padding: '40px 56px', height: '100%', overflowY: 'auto' }}>
        {/* Identity hero */}
        <div style={{ display: 'grid', gridTemplateColumns: '120px 1fr auto', gap: 32, paddingBottom: 32, borderBottom: `1px solid ${eA.cardEdge}`, marginBottom: 32, alignItems: 'start' }}>
          <div style={{ width: 120, height: 120, borderRadius: 8, background: eA.ink, color: eA.bg, fontFamily: eA.serif, fontSize: 56, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>L</div>
          <div>
            <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.12em', textTransform: 'uppercase', marginBottom: 10 }}>Profil prestataire · Paris</div>
            <div style={{ fontFamily: eA.serif, fontSize: 56, lineHeight: 1, letterSpacing: '-0.02em' }}>Léa Marchand</div>
            <div style={{ fontFamily: eA.serif, fontSize: 24, fontStyle: 'italic', color: eA.tabac, marginTop: 8 }}>
              Direction artistique & identité de marque
            </div>
            <div style={{ display: 'flex', gap: 18, marginTop: 18, alignItems: 'center' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontFamily: eA.mono, fontSize: 11, color: eA.sapin, letterSpacing: '0.08em', textTransform: 'uppercase' }}>
                <span style={{ width: 6, height: 6, borderRadius: '50%', background: eA.sapin }} />
                Disponible maintenant
              </div>
              <div style={{ width: 1, height: 14, background: eA.cardEdge }} />
              <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.tabac, letterSpacing: '0.08em' }}>4.9 ★ · 28 missions livrées</div>
              <div style={{ width: 1, height: 14, background: eA.cardEdge }} />
              <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.tabac, letterSpacing: '0.08em' }}>Membre depuis 2023</div>
            </div>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8, alignItems: 'flex-end' }}>
            <button style={{ background: eA.ink, color: eA.bg, border: 'none', padding: '10px 18px', borderRadius: 6, fontSize: 13, fontWeight: 500 }}>Aperçu public</button>
            <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em' }}>Profil rempli à 92%</div>
            <div style={{ width: 140, height: 2, background: eA.cardEdge, position: 'relative' }}>
              <div style={{ position: 'absolute', inset: 0, width: '92%', background: eA.rust }} />
            </div>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '180px 1fr', gap: 40 }}>
          {/* TOC */}
          <aside style={{ position: 'sticky', top: 0 }}>
            <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.12em', textTransform: 'uppercase', marginBottom: 14 }}>Sommaire</div>
            {[
              ['01', 'À propos', false],
              ['02', 'Vidéo de présentation', false],
              ['03', 'Domaines d\'expertise', true],
              ['04', 'Tarifs', false],
              ['05', 'Historique', false],
              ['06', 'Localisation', false],
              ['07', 'Langues', false],
              ['08', 'Compétences', false],
              ['09', 'Réseaux sociaux', false],
            ].map(([n, t, active]) => (
              <div key={n} style={{ display: 'flex', gap: 10, padding: '5px 0', fontSize: 12.5, color: active ? eA.rust : eA.tabac, fontWeight: active ? 500 : 400, borderLeft: active ? `1px solid ${eA.rust}` : '1px solid transparent', paddingLeft: 10, marginLeft: -10 }}>
                <span style={{ fontFamily: eA.mono, fontSize: 10, color: active ? eA.rust : eA.mute }}>{n}</span>
                <span>{t}</span>
              </div>
            ))}
          </aside>

          {/* Main column */}
          <div>
            {/* À propos */}
            <Section eA={eA} num="01" title="À propos" editable>
              <div style={{ fontFamily: eA.serif, fontSize: 22, fontStyle: 'italic', lineHeight: 1.4, color: eA.ink, maxWidth: 620 }}>
                "Je conçois des identités qui durent — pour des marques qui veulent dire quelque chose, sans crier."
              </div>
              <div style={{ marginTop: 18, fontSize: 14, lineHeight: 1.65, color: eA.tabac, maxWidth: 620 }}>
                Direction artistique indépendante depuis 8 ans. J'accompagne des entreprises de l'artisanat, de l'édition et du soin sur leur identité visuelle, du positionnement à l'application complète : print, digital, signalétique. Anciennement chez Studio Bonsoir et Pentagram Berlin.
              </div>
            </Section>

            {/* Domaines */}
            <Section eA={eA} num="03" title="Domaines d'expertise" subtitle="3 sur 5 sélectionnés" editable>
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {[
                  ['Design & UI/UX', true],
                  ['Direction artistique', true],
                  ['Brand & Identité', true],
                  ['Data, IA & ML', false],
                  ['Design 3D & Animation', false],
                  ['Vidéo & Motion', false],
                  ['Photo & Audiovisuel', false],
                  ['Marketing & Growth', false],
                  ['Rédaction & Traduction', false],
                  ['Business Development', false],
                  ['Consulting & Stratégie', false],
                  ['Product & UX Research', false],
                ].map(([t, active]) => (
                  <span key={t} style={{
                    fontSize: 12,
                    padding: '6px 12px',
                    borderRadius: 999,
                    border: `1px solid ${active ? eA.ink : eA.cardEdge}`,
                    background: active ? eA.ink : 'transparent',
                    color: active ? eA.bg : eA.tabac,
                  }}>{t}</span>
                ))}
              </div>
            </Section>

            {/* Tarifs */}
            <Section eA={eA} num="04" title="Tarifs" subtitle="Comment vous facturez vos prestations" editable>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12 }}>
                {[
                  { l: 'TJM standard', v: '720', sub: '7h de travail' },
                  { l: 'Direction artistique', v: '950', sub: 'jour stratégie' },
                  { l: 'Workshop', v: '1 800', sub: 'demi-journée' },
                ].map(p => (
                  <div key={p.l} style={{ background: eA.card, border: `1px solid ${eA.cardEdge}`, borderRadius: 8, padding: 18 }}>
                    <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase' }}>{p.l}</div>
                    <div style={{ fontFamily: eA.serif, fontSize: 38, marginTop: 8 }}>{p.v} <span style={{ fontSize: 18, color: eA.tabac }}>€</span></div>
                    <div style={{ fontSize: 11.5, color: eA.tabac, marginTop: 4 }}>{p.sub}</div>
                  </div>
                ))}
              </div>
            </Section>

            {/* Historique */}
            <Section eA={eA} num="05" title="Historique" subtitle="28 missions terminées · note moyenne 4.9/5">
              {[
                { c: 'Maison Fauve', t: 'Refonte de l\'identité de marque', m: '7 364 €', d: '15 avril 2026', r: 5 },
                { c: 'Coddo Studio', t: 'Direction artistique site marchand', m: '3 213 €', d: '1 mars 2026', r: 5 },
                { c: 'Coopérative Numa', t: 'Système de design · documentation', m: '4 280 €', d: '12 février 2026', r: 4 },
              ].map((p, i) => (
                <div key={i} style={{ display: 'grid', gridTemplateColumns: '64px 1fr auto', gap: 18, alignItems: 'baseline', padding: '18px 0', borderTop: `1px solid ${eA.cardEdge}` }}>
                  <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.1em' }}>0{i+1} · 26</div>
                  <div>
                    <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 4 }}>{p.c}</div>
                    <div style={{ fontFamily: eA.serif, fontSize: 22 }}>{p.t}</div>
                    <div style={{ fontSize: 11.5, color: eA.tabac, marginTop: 6 }}>{p.d} · {'★'.repeat(p.r)}</div>
                  </div>
                  <div style={{ fontFamily: eA.mono, fontSize: 16, color: eA.ink }}>{p.m}</div>
                </div>
              ))}
            </Section>

            <Section eA={eA} num="06" title="Localisation" editable>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16 }}>
                <FieldA label="Ville" value="Paris" />
                <FieldA label="Pays" value="France" />
                <FieldA label="Mode" tags={['À distance', 'Sur site', 'Hybride']} active={2} />
              </div>
            </Section>
          </div>
        </div>
      </div>
    </ShellEditorial>
  );
}

function Section({ num, title, subtitle, children, editable, eA }) {
  return (
    <div style={{ marginBottom: 48 }}>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 14, marginBottom: 16 }}>
        <span style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.1em' }}>{num}</span>
        <div style={{ fontFamily: eA.serif, fontSize: 32, lineHeight: 1, letterSpacing: '-0.01em' }}>{title}</div>
        {subtitle && <div style={{ fontSize: 12, color: eA.tabac, marginLeft: 'auto' }}>{subtitle}</div>}
        {editable && <div style={{ fontSize: 12, color: eA.rust, marginLeft: subtitle ? 16 : 'auto', cursor: 'pointer' }}>Modifier ↗</div>}
      </div>
      <div>{children}</div>
    </div>
  );
}
function FieldA({ label, value, tags, active }) {
  return (
    <div>
      <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 8 }}>{label}</div>
      {value && (
        <div style={{ background: eA.card, border: `1px solid ${eA.cardEdge}`, padding: '8px 12px', fontSize: 13, borderRadius: 4 }}>{value}</div>
      )}
      {tags && (
        <div style={{ display: 'flex', gap: 6 }}>
          {tags.map((t, i) => (
            <span key={t} style={{
              fontSize: 12, padding: '6px 12px', borderRadius: 999,
              border: `1px solid ${i === active ? eA.ink : eA.cardEdge}`,
              background: i === active ? eA.ink : 'transparent',
              color: i === active ? eA.bg : eA.tabac,
            }}>{t}</span>
          ))}
        </div>
      )}
    </div>
  );
}

// ─────────────────────────────────────────────
// PROJECT DETAIL
// ─────────────────────────────────────────────
function ProjectDetailEditorial() {
  return (
    <ShellEditorial active="projects">
      <div style={{ padding: '40px 56px', height: '100%', overflow: 'hidden' }}>
        <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.1em', marginBottom: 14 }}>← Retour aux projets</div>

        <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', marginBottom: 8 }}>
          <div>
            <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.12em', textTransform: 'uppercase', marginBottom: 8 }}>Projet · MS-2891 · Maison Fauve</div>
            <div style={{ fontFamily: eA.serif, fontSize: 52, lineHeight: 1, letterSpacing: '-0.02em' }}>
              Refonte de l'identité <span style={{ fontStyle: 'italic' }}>Maison Fauve</span>
            </div>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <button style={{ background: 'transparent', border: `1px solid ${eA.ink}`, color: eA.ink, padding: '10px 16px', borderRadius: 6, fontSize: 13 }}>Conversation</button>
            <button style={{ background: eA.ink, color: eA.bg, border: 'none', padding: '10px 16px', borderRadius: 6, fontSize: 13 }}>Marquer terminé</button>
          </div>
        </div>

        {/* Stepper */}
        <div style={{ marginTop: 36, marginBottom: 32, paddingBottom: 32, borderBottom: `1px solid ${eA.cardEdge}` }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 0 }}>
            {[
              ['Crée', true, '20.04'],
              ['Acceptée', true, '22.04'],
              ['Payée', true, '01.05'],
              ['Active', true, '01.05'],
              ['Livrée', false, '— en cours'],
              ['Terminée', false, '—'],
            ].map(([l, done, d], i, arr) => (
              <React.Fragment key={l}>
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', minWidth: 0 }}>
                  <div style={{
                    width: 28, height: 28, borderRadius: '50%',
                    background: done ? eA.ink : eA.bg,
                    border: `1px solid ${done ? eA.ink : eA.cardEdge}`,
                    color: done ? eA.bg : eA.mute,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontFamily: eA.mono, fontSize: 12,
                  }}>{done ? '✓' : i + 1}</div>
                  <div style={{ fontSize: 12, fontWeight: done ? 500 : 400, color: done ? eA.ink : eA.mute, marginTop: 8 }}>{l}</div>
                  <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.05em', marginTop: 2 }}>{d}</div>
                </div>
                {i < arr.length - 1 && <div style={{ flex: 1, height: 1, background: i < 3 ? eA.ink : eA.cardEdge, margin: '0 4px', marginBottom: 32 }} />}
              </React.Fragment>
            ))}
          </div>
        </div>

        {/* Two col body */}
        <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 32 }}>
          <div>
            <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 8 }}>Description</div>
            <div style={{ fontFamily: eA.serif, fontSize: 22, lineHeight: 1.4, color: eA.ink, marginBottom: 22 }}>
              Refonte complète de l'identité visuelle pour la maison de bougies artisanales — du wordmark aux applications print et packaging.
            </div>

            <div style={{ marginBottom: 18, fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase' }}>Jalons · 3 sur 5 livrés</div>
            {[
              { n: '01', t: 'Recherche & moodboard', s: 'Livré', m: '900 €', d: '24 avril' },
              { n: '02', t: 'Wordmark & système', s: 'Livré', m: '1 200 €', d: '1 mai' },
              { n: '03', t: 'Direction artistique site', s: 'En cours', m: '1 800 €', d: '15 mai' },
              { n: '04', t: 'Packaging & print', s: 'À venir', m: '2 400 €', d: '1 juin' },
              { n: '05', t: 'Charte de marque', s: 'À venir', m: '1 064 €', d: '12 juin' },
            ].map(j => (
              <div key={j.n} style={{ display: 'grid', gridTemplateColumns: '40px 1fr 100px 100px', gap: 14, padding: '14px 0', borderTop: `1px solid ${eA.cardEdge}`, alignItems: 'baseline' }}>
                <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.1em' }}>{j.n}</div>
                <div>
                  <div style={{ fontFamily: eA.serif, fontSize: 18 }}>{j.t}</div>
                  <div style={{ fontSize: 11.5, color: eA.tabac, marginTop: 2 }}>échéance {j.d}</div>
                </div>
                <div style={{ fontSize: 11, padding: '3px 9px', borderRadius: 999, textAlign: 'center',
                  background: j.s === 'Livré' ? eA.sapinSoft : j.s === 'En cours' ? eA.rustSoft : 'transparent',
                  color: j.s === 'Livré' ? eA.sapin : j.s === 'En cours' ? eA.rust : eA.mute,
                  border: j.s === 'À venir' ? `1px solid ${eA.cardEdge}` : 'none',
                  justifySelf: 'start',
                }}>{j.s}</div>
                <div style={{ fontFamily: eA.mono, fontSize: 14, color: eA.ink, textAlign: 'right' }}>{j.m}</div>
              </div>
            ))}
          </div>

          {/* Right rail */}
          <div>
            <div style={{ background: eA.card, border: `1px solid ${eA.cardEdge}`, borderRadius: 8, padding: 20, marginBottom: 16 }}>
              <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 4 }}>Paiement unique</div>
              <div style={{ fontFamily: eA.serif, fontSize: 56, lineHeight: 1, letterSpacing: '-0.02em', marginTop: 6 }}>7 364<span style={{ fontSize: 24, color: eA.tabac }}> €</span></div>
              <div style={{ display: 'inline-block', marginTop: 12, padding: '4px 10px', background: eA.sapinSoft, color: eA.sapin, fontFamily: eA.mono, fontSize: 11, letterSpacing: '0.05em', borderRadius: 4 }}>● Payé · en séquestre</div>

              <div style={{ borderTop: `1px solid ${eA.cardEdge}`, marginTop: 20, paddingTop: 16 }}>
                <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 10 }}>Frais plateforme</div>
                {[['Moins de 200 €', '9,00 €'], ['200 — 1 000 €', '15,00 €'], ['Plus de 1 000 €', '25,00 €', true]].map(([l, v, hl]) => (
                  <div key={l} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', borderLeft: hl ? `2px solid ${eA.rust}` : 'none', paddingLeft: hl ? 10 : 0 }}>
                    <span style={{ fontSize: 12.5, color: hl ? eA.rust : eA.tabac }}>{l}</span>
                    <span style={{ fontFamily: eA.mono, fontSize: 12.5, color: hl ? eA.rust : eA.tabac }}>{v}</span>
                  </div>
                ))}
                <div style={{ borderTop: `1px solid ${eA.cardEdge}`, marginTop: 12, paddingTop: 12, display: 'flex', justifyContent: 'space-between' }}>
                  <span style={{ fontSize: 13, fontWeight: 500 }}>Tu encaisses</span>
                  <span style={{ fontFamily: eA.mono, fontSize: 14, color: eA.ink }}>7 339,00 €</span>
                </div>
              </div>
            </div>

            <div style={{ background: eA.card, border: `1px solid ${eA.cardEdge}`, borderRadius: 8, padding: 20 }}>
              <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 14 }}>Participants</div>
              {[
                { i: 'M', n: 'Maison Fauve', r: 'Client', c: eA.ink },
                { i: 'L', n: 'Léa Marchand', r: 'Prestataire', c: eA.rust },
              ].map(p => (
                <div key={p.n} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '8px 0' }}>
                  <div style={{ width: 32, height: 32, borderRadius: '50%', background: p.c, color: eA.bg, display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: eA.serif, fontSize: 14 }}>{p.i}</div>
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 500 }}>{p.n}</div>
                    <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.08em', textTransform: 'uppercase' }}>{p.r}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </ShellEditorial>
  );
}

// ─────────────────────────────────────────────
// WALLET
// ─────────────────────────────────────────────
function WalletEditorial() {
  return (
    <ShellEditorial active="wallet">
      <div style={{ padding: '40px 56px', height: '100%', overflow: 'hidden' }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr', gap: 32, marginBottom: 32, paddingBottom: 32, borderBottom: `1px solid ${eA.cardEdge}` }}>
          <div>
            <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.12em', textTransform: 'uppercase', marginBottom: 14 }}>Portefeuille · revenus 2026</div>
            <div style={{ fontFamily: eA.serif, fontSize: 96, lineHeight: 0.95, letterSpacing: '-0.03em' }}>
              10 502<span style={{ fontSize: 36, color: eA.tabac, fontStyle: 'italic' }}> €</span>
            </div>
            <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginTop: 12 }}>
              <span style={{ fontFamily: eA.mono, fontSize: 11, padding: '4px 10px', background: eA.sapinSoft, color: eA.sapin, borderRadius: 4, letterSpacing: '0.05em' }}>● Stripe actif</span>
              <span style={{ fontSize: 12, color: eA.tabac }}>Virements activés · IBAN ••••8420</span>
            </div>
          </div>
          <div>
            <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 8 }}>Solde disponible</div>
            <div style={{ fontFamily: eA.serif, fontSize: 56, lineHeight: 1 }}>0<span style={{ fontSize: 24, color: eA.tabac }}> €</span></div>
            <div style={{ fontSize: 12, color: eA.tabac, marginTop: 6, marginBottom: 16 }}>Aucun fonds disponible pour retrait pour l'instant.</div>
            <button style={{ width: '100%', background: eA.ink, color: eA.bg, padding: '12px 20px', border: 'none', borderRadius: 6, fontSize: 13, fontWeight: 500 }}>Retirer des fonds →</button>
          </div>
        </div>

        {/* 3 stat columns */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 0, marginBottom: 36, borderTop: `1px solid ${eA.cardEdge}`, borderBottom: `1px solid ${eA.cardEdge}` }}>
          {[
            { l: 'En séquestre', v: '0,00', d: 'aucune mission en attente', c: eA.tabac },
            { l: 'Disponible', v: '0,00', d: 'prêt à être retiré', c: eA.tabac },
            { l: 'Transféré', v: '10 502,00', d: 'déjà versé sur ton compte', c: eA.sapin },
          ].map((s, i) => (
            <div key={i} style={{ padding: '24px 0', paddingLeft: i ? 24 : 0, borderLeft: i ? `1px solid ${eA.cardEdge}` : 'none' }}>
              <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.12em', textTransform: 'uppercase' }}>{s.l}</div>
              <div style={{ fontFamily: eA.serif, fontSize: 44, lineHeight: 1, marginTop: 10, color: s.c }}>{s.v}<span style={{ fontSize: 18, color: eA.tabac }}> €</span></div>
              <div style={{ fontSize: 11.5, color: eA.tabac, marginTop: 8 }}>{s.d}</div>
            </div>
          ))}
        </div>

        {/* History */}
        <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', marginBottom: 20 }}>
          <div style={{ fontFamily: eA.serif, fontSize: 28, letterSpacing: '-0.01em' }}>Historique des missions</div>
          <div style={{ display: 'flex', gap: 14, fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.05em' }}>
            <span style={{ color: eA.ink }}>Tous</span>
            <span>En séquestre</span>
            <span>Transférés</span>
            <span>Exporter ↗</span>
          </div>
        </div>

        {[
          { d: '01.05.26', c: 'Maison Fauve', t: 'Refonte identité', m: '7 364,00', f: '−25,00 € frais', s: 'Transféré', sc: 'sapin' },
          { d: '28.04.26', c: 'Coddo Studio', t: 'Direction artistique site', m: '3 188,00', f: '−25,00 € frais', s: 'Transféré', sc: 'sapin' },
          { d: '15.04.26', c: 'Atelier Nour', t: 'Workshop identité', m: '1 800,00', f: '−15,00 € frais', s: 'En séquestre', sc: 'rust' },
          { d: '02.04.26', c: 'Coopérative Numa', t: 'Système de design', m: '4 280,00', f: '−25,00 € frais', s: 'Transféré', sc: 'sapin' },
        ].map((r, i) => (
          <div key={i} style={{ display: 'grid', gridTemplateColumns: '90px 1.5fr 1fr 130px 110px', gap: 16, padding: '14px 0', borderTop: `1px solid ${eA.cardEdge}`, alignItems: 'baseline' }}>
            <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute, letterSpacing: '0.05em' }}>{r.d}</div>
            <div>
              <div style={{ fontFamily: eA.mono, fontSize: 10, color: eA.mute, letterSpacing: '0.1em', textTransform: 'uppercase' }}>{r.c}</div>
              <div style={{ fontFamily: eA.serif, fontSize: 18, marginTop: 2 }}>{r.t}</div>
            </div>
            <div style={{ fontFamily: eA.mono, fontSize: 11, color: eA.mute }}>{r.f}</div>
            <div style={{ fontFamily: eA.mono, fontSize: 16, color: eA.ink, textAlign: 'right' }}>{r.m} €</div>
            <div style={{ fontSize: 11, padding: '3px 10px', borderRadius: 999, textAlign: 'center',
              background: r.sc === 'sapin' ? eA.sapinSoft : eA.rustSoft,
              color: r.sc === 'sapin' ? eA.sapin : eA.rust,
              justifySelf: 'end',
            }}>● {r.s}</div>
          </div>
        ))}
      </div>
    </ShellEditorial>
  );
}

Object.assign(window, {
  DashboardEditorial, FindFreelancersEditorial, ProfileEditorial, ProjectDetailEditorial, WalletEditorial,
});
