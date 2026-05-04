// Direction B — "Atelier Studio"
// Onyx BG, volt accent (acid lime), Inter Tight + Geist Mono.
// Sensation : tool tech sobre, dense, productif, type Linear/Vercel/Read.cv dark.

const eB = {
  bg: '#0E0E0E',
  card: '#161616',
  cardSoft: '#1A1A1A',
  border: '#262624',
  borderSoft: '#1F1F1D',
  fg: '#F2F2F0',
  mute: '#A8A8A4',
  muteDim: '#7A7A78',
  volt: '#D4FF3A',
  voltSoft: 'rgba(212, 255, 58, 0.12)',
  ember: '#FF6B47',
  emberSoft: 'rgba(255, 107, 71, 0.12)',
  green: '#5DD68A',
  greenSoft: 'rgba(93, 214, 138, 0.12)',
  sans: "'Inter Tight', sans-serif",
  mono: "'Geist Mono', monospace",
};

function SidebarStudio({ active = 'dashboard' }) {
  const items = [
    { id: 'dashboard', label: 'Dashboard', n: '1', k: 'D' },
    { id: 'messages', label: 'Messages', n: '2', k: 'M', badge: 3 },
    { id: 'projects', label: 'Projects', n: '3', k: 'P' },
    { id: 'opportunities', label: 'Opportunities', n: '4', k: 'O' },
    { id: 'applications', label: 'Applications', n: '5', k: 'A' },
    { id: 'profile', label: 'Provider profile', n: '6', k: 'V' },
    { id: 'payment', label: 'Payment info', n: '7', k: 'I' },
    { id: 'wallet', label: 'Wallet', n: '8', k: 'W' },
    { id: 'invoices', label: 'Invoices', n: '9', k: 'F' },
    { id: 'account', label: 'Account', n: '0', k: 'C' },
  ];
  return (
    <aside style={{
      width: 232, background: eB.bg, borderRight: `1px solid ${eB.border}`,
      padding: '20px 0', display: 'flex', flexDirection: 'column',
      fontFamily: eB.sans, flexShrink: 0,
    }}>
      <div style={{ padding: '0 18px', marginBottom: 20, display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ display: 'inline-block', width: 10, height: 10, background: eB.volt }} />
        <span style={{ fontSize: 17, fontWeight: 600, letterSpacing: '-0.02em' }}>atelier</span>
        <span style={{ marginLeft: 'auto', fontFamily: eB.mono, fontSize: 9, color: eB.muteDim, letterSpacing: '0.1em' }}>v2.4</span>
      </div>

      <div style={{ padding: '0 12px', marginBottom: 14 }}>
        <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, padding: '10px 12px', display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{ width: 28, height: 28, borderRadius: 4, background: eB.volt, color: eB.bg, fontSize: 12, fontWeight: 700, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>L</div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 12.5, fontWeight: 600 }}>Léa Marchand</div>
            <div style={{ fontFamily: eB.mono, fontSize: 9, color: eB.muteDim, letterSpacing: '0.08em', marginTop: 1 }}>PROVIDER · TIER 2</div>
          </div>
          <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim }}>⌄</span>
        </div>
        <button style={{ marginTop: 8, width: '100%', background: eB.volt, color: eB.bg, border: 'none', borderRadius: 5, padding: '9px 12px', fontFamily: eB.sans, fontSize: 12.5, fontWeight: 600, display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer', letterSpacing: '-0.01em' }}>
          <span>Refer a deal</span>
          <span style={{ fontFamily: eB.mono, fontSize: 10 }}>⏎ R</span>
        </button>
      </div>

      <div style={{ padding: '0 12px', marginBottom: 8 }}>
        <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 5, padding: '6px 10px', display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim }}>⌕</span>
          <span style={{ fontSize: 12, color: eB.muteDim, flex: 1 }}>Search…</span>
          <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, padding: '1px 5px', border: `1px solid ${eB.border}`, borderRadius: 3 }}>⌘K</span>
        </div>
      </div>

      <nav style={{ flex: 1, padding: '8px 8px' }}>
        {items.map(it => {
          const isActive = it.id === active;
          return (
            <div key={it.id} style={{
              padding: '7px 12px', display: 'flex', alignItems: 'center', gap: 10, borderRadius: 5, cursor: 'pointer',
              background: isActive ? eB.card : 'transparent',
              borderLeft: isActive ? `2px solid ${eB.volt}` : '2px solid transparent',
              color: isActive ? eB.fg : eB.mute,
              fontWeight: isActive ? 500 : 400,
              fontSize: 12.5, marginBottom: 1,
            }}>
              <span style={{ flex: 1 }}>{it.label}</span>
              {it.badge ? (
                <span style={{ fontFamily: eB.mono, fontSize: 9, background: eB.ember, color: eB.bg, borderRadius: 999, padding: '1px 5px', minWidth: 14, textAlign: 'center' }}>{it.badge}</span>
              ) : (
                <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, padding: '1px 5px', border: `1px solid ${eB.border}`, borderRadius: 3 }}>G {it.k}</span>
              )}
            </div>
          );
        })}
      </nav>

      <div style={{ padding: '12px 18px 0', borderTop: `1px solid ${eB.border}` }}>
        <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, display: 'flex', justifyContent: 'space-between' }}>
          <span>● online</span><span>fr · €</span>
        </div>
      </div>
    </aside>
  );
}

function HeaderStudio({ crumbs = ['Workspace', 'Dashboard'] }) {
  return (
    <header style={{ borderBottom: `1px solid ${eB.border}`, padding: '12px 28px', display: 'flex', alignItems: 'center', gap: 16, background: eB.bg }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, fontFamily: eB.mono, fontSize: 11, color: eB.muteDim, letterSpacing: '0.05em' }}>
        {crumbs.map((c, i, a) => (
          <React.Fragment key={i}>
            <span style={{ color: i === a.length - 1 ? eB.fg : eB.muteDim }}>{c}</span>
            {i < a.length - 1 && <span>/</span>}
          </React.Fragment>
        ))}
      </div>
      <div style={{ flex: 1 }} />
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.05em' }}>10 credits left</div>
        <button style={{ background: 'transparent', color: eB.fg, border: `1px solid ${eB.border}`, padding: '5px 10px', borderRadius: 4, fontSize: 11, fontFamily: eB.mono, letterSpacing: '0.05em' }}>UPGRADE</button>
        <span style={{ position: 'relative' }}>
          <span style={{ fontSize: 14, color: eB.mute }}>◔</span>
          <span style={{ position: 'absolute', top: -3, right: -5, background: eB.ember, color: eB.bg, fontFamily: eB.mono, fontSize: 8, borderRadius: 999, padding: '1px 4px' }}>10</span>
        </span>
      </div>
    </header>
  );
}

function ShellStudio({ active, crumbs, children }) {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: eB.bg, fontFamily: eB.sans, color: eB.fg, overflow: 'hidden' }}>
      <SidebarStudio active={active} />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <HeaderStudio crumbs={crumbs} />
        <div style={{ flex: 1, overflow: 'hidden' }}>{children}</div>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────
// DASHBOARD
// ─────────────────────────────────────────────
function DashboardStudio() {
  return (
    <ShellStudio active="dashboard" crumbs={['Workspace', 'Dashboard']}>
      <div style={{ padding: '28px 32px', height: '100%', overflow: 'hidden' }}>
        {/* Hero band */}
        <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 20, marginBottom: 24 }}>
          <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 8, padding: 24, position: 'relative', overflow: 'hidden' }}>
            <div style={{ position: 'absolute', top: 0, right: 0, width: 240, height: '100%', background: `linear-gradient(135deg, transparent 0%, ${eB.voltSoft} 100%)` }} />
            <div style={{ position: 'relative' }}>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 10 }}>FRI 01.05.26 · 16:38 CET</div>
              <div style={{ fontSize: 36, fontWeight: 600, letterSpacing: '-0.03em', lineHeight: 1.05 }}>
                Welcome back, Léa.
              </div>
              <div style={{ fontSize: 14, color: eB.mute, marginTop: 6, lineHeight: 1.5 }}>
                3 unread threads · 2 proposals waiting on you · 1 milestone due tomorrow.
              </div>
              <div style={{ display: 'flex', gap: 8, marginTop: 18 }}>
                <button style={{ background: eB.volt, color: eB.bg, border: 'none', padding: '8px 14px', borderRadius: 5, fontSize: 12.5, fontWeight: 600 }}>Review proposals →</button>
                <button style={{ background: 'transparent', color: eB.fg, border: `1px solid ${eB.border}`, padding: '7px 13px', borderRadius: 5, fontSize: 12.5, fontWeight: 500 }}>Open inbox</button>
              </div>
            </div>
          </div>

          <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 8, padding: 20 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
              <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase' }}>This week</span>
              <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.green }}>↗ +18%</span>
            </div>
            {/* sparkline */}
            <svg width="100%" height="44" viewBox="0 0 200 44" style={{ marginBottom: 12 }}>
              <polyline points="0,30 20,28 40,32 60,22 80,26 100,18 120,20 140,12 160,16 180,8 200,10" fill="none" stroke={eB.volt} strokeWidth="1.5" />
              <polyline points="0,30 20,28 40,32 60,22 80,26 100,18 120,20 140,12 160,16 180,8 200,10 200,44 0,44" fill={eB.voltSoft} stroke="none" />
            </svg>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.05em' }}>
              <span>MON</span><span>TUE</span><span>WED</span><span>THU</span><span>FRI</span>
            </div>
          </div>
        </div>

        {/* Metric grid */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 0, border: `1px solid ${eB.border}`, borderRadius: 8, marginBottom: 24, overflow: 'hidden' }}>
          {[
            { l: 'Revenue MTD', v: '€8,240', d: '+24% MoM', c: eB.green },
            { l: 'Active jobs', v: '3 / 5', d: '2 slots open', c: eB.fg },
            { l: 'Win rate', v: '82%', d: '18 / 22 apps', c: eB.fg },
            { l: 'Rating', v: '4.9', d: '17 reviews', c: eB.volt },
          ].map((s, i) => (
            <div key={i} style={{ padding: '18px 20px', background: eB.card, borderRight: i < 3 ? `1px solid ${eB.border}` : 'none' }}>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 8 }}>{s.l}</div>
              <div style={{ fontFamily: eB.mono, fontSize: 28, fontWeight: 500, letterSpacing: '-0.02em', color: s.c, lineHeight: 1 }}>{s.v}</div>
              <div style={{ fontFamily: eB.mono, fontSize: 11, color: eB.mute, marginTop: 8 }}>{s.d}</div>
            </div>
          ))}
        </div>

        {/* Two cols */}
        <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 20 }}>
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
              <div style={{ fontSize: 13, fontWeight: 600, letterSpacing: '-0.01em' }}>Active missions</div>
              <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim }}>3 active</span>
            </div>
            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 8, overflow: 'hidden' }}>
              {[
                { id: 'MS-2891', c: 'Maison Fauve', t: 'Brand identity refresh', m: '€7,364', p: 60, s: 'on track' },
                { id: 'MS-2877', c: 'Coddo Studio', t: 'Art direction · web', m: '€3,213', p: 30, s: 'kickoff' },
                { id: 'MS-2810', c: 'Coop. Numa', t: 'Design system audit', m: '€4,280', p: 85, s: 'review' },
              ].map((m, i, a) => (
                <div key={i} style={{ padding: '14px 18px', borderTop: i ? `1px solid ${eB.borderSoft}` : 'none' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 6 }}>
                    <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
                      <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.05em' }}>{m.id}</span>
                      <span style={{ fontSize: 13.5, fontWeight: 600, letterSpacing: '-0.01em' }}>{m.t}</span>
                    </div>
                    <span style={{ fontFamily: eB.mono, fontSize: 12, color: eB.fg }}>{m.m}</span>
                  </div>
                  <div style={{ fontSize: 12, color: eB.mute, marginBottom: 10 }}>{m.c}</div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <div style={{ flex: 1, height: 3, background: eB.borderSoft, borderRadius: 2, position: 'relative' }}>
                      <div style={{ position: 'absolute', inset: 0, width: m.p + '%', background: eB.volt, borderRadius: 2 }} />
                    </div>
                    <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.mute, minWidth: 32, textAlign: 'right' }}>{m.p}%</span>
                    <span style={{ fontFamily: eB.mono, fontSize: 9, padding: '2px 7px', borderRadius: 3, background: eB.borderSoft, color: eB.mute, letterSpacing: '0.08em', textTransform: 'uppercase' }}>{m.s}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
              <div style={{ fontSize: 13, fontWeight: 600, letterSpacing: '-0.01em' }}>Matched for you</div>
              <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.volt }}>see all →</span>
            </div>
            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 8 }}>
              {[
                { co: 'Studio Bonsoir', t: 'Art direction', b: '€8 — 12k', m: 94 },
                { co: 'Coop Numa', t: 'Design system rebuild', b: '€15 — 20k', m: 88 },
                { co: 'Atelier Belleville', t: 'Visual identity', b: '€6 — 10k', m: 76 },
              ].map((o, i) => (
                <div key={i} style={{ padding: '14px 18px', borderTop: i ? `1px solid ${eB.borderSoft}` : 'none' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
                    <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.08em', textTransform: 'uppercase' }}>{o.co}</span>
                    <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.volt }}>● {o.m}% match</span>
                  </div>
                  <div style={{ fontSize: 14, fontWeight: 600, letterSpacing: '-0.01em', marginTop: 4 }}>{o.t}</div>
                  <div style={{ fontFamily: eB.mono, fontSize: 11, color: eB.mute, marginTop: 4 }}>{o.b}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </ShellStudio>
  );
}

// ─────────────────────────────────────────────
// FIND FREELANCERS
// ─────────────────────────────────────────────
function FindFreelancersStudio() {
  const people = [
    { i: 'CL', n: 'Camille Lefèvre', t: 'Brand systems & art direction', loc: 'Paris', tjm: 720, rate: 4.9, n2: 28, av: 'now', tags: ['brand', 'art-direction', 'print'] },
    { i: 'YO', n: 'Yann Orhant', t: 'Senior product designer · ex-Doctolib', loc: 'Nantes', tjm: 850, rate: 4.8, n2: 41, av: 'now', tags: ['product', 'ux-research'] },
    { i: 'AB', n: 'Aïssa Benali', t: 'Brand identity & motion', loc: 'Lyon', tjm: 600, rate: 5.0, n2: 14, av: 'soon', tags: ['brand', 'motion'] },
    { i: 'NM', n: 'Noor Maalouf', t: 'Brand strategist · luxury & craft', loc: 'Paris', tjm: 950, rate: 4.9, n2: 22, av: 'now', tags: ['strategy', 'brand'] },
    { i: 'EP', n: 'Élisa Park', t: 'Editorial design & typography', loc: 'Brussels', tjm: 680, rate: 4.7, n2: 19, av: 'soon', tags: ['editorial', 'type'] },
    { i: 'MR', n: 'Mathieu Roussel', t: 'Webflow & technical direction', loc: 'Toulouse', tjm: 720, rate: 4.9, n2: 33, av: 'now', tags: ['webflow', 'dev'] },
  ];
  return (
    <ShellStudio active="find" crumbs={['Workspace', 'Discover', 'Freelancers']}>
      <div style={{ padding: '28px 32px', height: '100%', overflow: 'hidden' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: 24 }}>
          <div>
            <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 8 }}>1,240 verified providers</div>
            <div style={{ fontSize: 32, fontWeight: 600, letterSpacing: '-0.03em', lineHeight: 1 }}>Find freelancers</div>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 5, padding: '8px 12px', fontSize: 12, color: eB.mute, display: 'flex', alignItems: 'center', gap: 8, minWidth: 280 }}>
              <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim }}>⌕</span>
              <span style={{ flex: 1 }}>name, skill, expertise…</span>
              <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, padding: '1px 5px', border: `1px solid ${eB.border}`, borderRadius: 3 }}>/</span>
            </div>
            <button style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 5, padding: '8px 12px', color: eB.fg, fontSize: 12 }}>Sort: Relevance ⌄</button>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '240px 1fr', gap: 24 }}>
          {/* filter sidebar */}
          <aside style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 8, padding: 16, alignSelf: 'start' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
              <span style={{ fontSize: 12, fontWeight: 600 }}>Filters</span>
              <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.volt }}>4 active · clear</span>
            </div>

            {[
              { l: 'Availability', items: ['Now', 'Soon', 'All'], a: 0 },
              { l: 'Work mode', items: ['Remote', 'On site', 'Hybrid'], a: 0 },
            ].map((g, i) => (
              <div key={i} style={{ marginBottom: 16 }}>
                <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 8 }}>{g.l}</div>
                <div style={{ display: 'flex', gap: 4, background: eB.bg, border: `1px solid ${eB.border}`, borderRadius: 5, padding: 2 }}>
                  {g.items.map((it, j) => (
                    <span key={j} style={{
                      fontSize: 11.5, padding: '5px 0', borderRadius: 4, flex: 1, textAlign: 'center',
                      background: j === g.a ? eB.volt : 'transparent',
                      color: j === g.a ? eB.bg : eB.mute, fontWeight: j === g.a ? 600 : 400,
                    }}>{it}</span>
                  ))}
                </div>
              </div>
            ))}

            <div style={{ marginBottom: 16 }}>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 8 }}>Daily rate · €</div>
              <div style={{ display: 'flex', gap: 6 }}>
                <div style={{ flex: 1, background: eB.bg, border: `1px solid ${eB.border}`, padding: '5px 8px', fontFamily: eB.mono, fontSize: 11, borderRadius: 3 }}>400</div>
                <div style={{ flex: 1, background: eB.bg, border: `1px solid ${eB.border}`, padding: '5px 8px', fontFamily: eB.mono, fontSize: 11, borderRadius: 3 }}>1200</div>
              </div>
              <div style={{ height: 2, background: eB.border, marginTop: 12, position: 'relative' }}>
                <div style={{ position: 'absolute', left: '20%', right: '15%', top: 0, bottom: 0, background: eB.volt }} />
                <div style={{ position: 'absolute', left: '20%', top: -3, width: 8, height: 8, background: eB.volt, transform: 'translateX(-50%)' }} />
                <div style={{ position: 'absolute', left: '85%', top: -3, width: 8, height: 8, background: eB.volt, transform: 'translateX(-50%)' }} />
              </div>
            </div>

            <div style={{ marginBottom: 16 }}>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 8 }}>Languages</div>
              <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                {['FR', 'EN', 'ES', 'DE', 'IT', 'PT'].map((l, i) => (
                  <span key={l} style={{
                    fontFamily: eB.mono, fontSize: 11, padding: '4px 8px', borderRadius: 3,
                    border: `1px solid ${i < 2 ? eB.volt : eB.border}`,
                    background: i < 2 ? eB.voltSoft : 'transparent',
                    color: i < 2 ? eB.volt : eB.mute,
                  }}>{l}</span>
                ))}
              </div>
            </div>

            <div>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 8 }}>Expertise</div>
              {['Development', 'Data, AI & ML', 'Design & UI/UX', '3D & Animation', 'Video & Motion', 'Photo'].map((e, i) => (
                <div key={e} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', fontSize: 12, color: eB.mute }}>
                  <span style={{ width: 12, height: 12, border: `1px solid ${eB.border}`, borderRadius: 2, background: i === 2 ? eB.volt : 'transparent', position: 'relative' }}>
                    {i === 2 && <span style={{ color: eB.bg, position: 'absolute', top: -3, left: 1, fontSize: 10, fontWeight: 700 }}>✓</span>}
                  </span>
                  <span>{e}</span>
                </div>
              ))}
            </div>
          </aside>

          {/* grid */}
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
              <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim }}>showing 6 of 1,240</span>
              <div style={{ display: 'flex', gap: 4, background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 4, padding: 2 }}>
                <span style={{ fontFamily: eB.mono, fontSize: 10, padding: '4px 10px', background: eB.borderSoft, borderRadius: 3, color: eB.fg }}>grid</span>
                <span style={{ fontFamily: eB.mono, fontSize: 10, padding: '4px 10px', color: eB.muteDim }}>list</span>
              </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12 }}>
              {people.map((p, i) => (
                <div key={i} style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 8, padding: 16, position: 'relative' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12 }}>
                    <div style={{
                      width: 44, height: 44, borderRadius: 5,
                      background: i % 3 === 0 ? eB.volt : i % 3 === 1 ? eB.ember : eB.card,
                      border: i % 3 === 2 ? `1px solid ${eB.border}` : 'none',
                      color: i % 3 === 2 ? eB.fg : eB.bg,
                      fontSize: 14, fontWeight: 700,
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                    }}>{p.i}</div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 5, fontFamily: eB.mono, fontSize: 9, color: p.av === 'now' ? eB.green : eB.mute, letterSpacing: '0.08em', textTransform: 'uppercase' }}>
                      <span style={{ width: 5, height: 5, borderRadius: '50%', background: p.av === 'now' ? eB.green : eB.ember }} />
                      {p.av === 'now' ? 'Available' : 'Soon'}
                    </div>
                  </div>
                  <div style={{ fontSize: 14.5, fontWeight: 600, letterSpacing: '-0.01em', marginBottom: 4 }}>{p.n}</div>
                  <div style={{ fontSize: 12, color: eB.mute, lineHeight: 1.4, marginBottom: 12, minHeight: 32 }}>{p.t}</div>
                  <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginBottom: 12 }}>
                    {p.tags.map(t => (
                      <span key={t} style={{ fontFamily: eB.mono, fontSize: 10, padding: '2px 7px', background: eB.borderSoft, borderRadius: 3, color: eB.mute }}>{t}</span>
                    ))}
                  </div>
                  <div style={{ borderTop: `1px solid ${eB.borderSoft}`, paddingTop: 10, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div style={{ fontFamily: eB.mono, fontSize: 12, color: eB.fg }}>€{p.tjm}<span style={{ color: eB.muteDim, fontSize: 10 }}>/d</span></div>
                    <div style={{ fontFamily: eB.mono, fontSize: 11, color: eB.mute }}>★ {p.rate} <span style={{ color: eB.muteDim }}>· {p.n2}</span></div>
                    <button style={{ background: eB.borderSoft, border: 'none', color: eB.fg, fontFamily: eB.mono, fontSize: 11, padding: '4px 10px', borderRadius: 4 }}>view →</button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </ShellStudio>
  );
}

// ─────────────────────────────────────────────
// PROFILE
// ─────────────────────────────────────────────
function ProfileStudio() {
  return (
    <ShellStudio active="profile" crumbs={['Workspace', 'Provider profile']}>
      <div style={{ padding: '28px 32px', height: '100%', overflowY: 'auto' }}>
        <div style={{ display: 'grid', gridTemplateColumns: '88px 1fr auto', gap: 20, paddingBottom: 24, borderBottom: `1px solid ${eB.border}`, marginBottom: 28, alignItems: 'center' }}>
          <div style={{ width: 88, height: 88, borderRadius: 8, background: eB.volt, color: eB.bg, fontSize: 36, fontWeight: 700, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>L</div>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
              <span style={{ fontFamily: eB.mono, fontSize: 9, padding: '3px 7px', background: eB.greenSoft, color: eB.green, borderRadius: 3, letterSpacing: '0.08em', textTransform: 'uppercase' }}>● Available</span>
              <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.08em', textTransform: 'uppercase' }}>tier 2 · paris · since 2023</span>
            </div>
            <div style={{ fontSize: 36, fontWeight: 600, letterSpacing: '-0.03em', lineHeight: 1 }}>Léa Marchand</div>
            <div style={{ fontSize: 14, color: eB.mute, marginTop: 6 }}>Art direction & brand identity · 4.9 ★ from 28 deliveries</div>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8, alignItems: 'flex-end' }}>
            <button style={{ background: eB.volt, color: eB.bg, border: 'none', padding: '8px 14px', borderRadius: 5, fontSize: 12.5, fontWeight: 600 }}>Public preview</button>
            <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim }}>profile · 92% complete</div>
            <div style={{ width: 140, height: 2, background: eB.border }}>
              <div style={{ width: '92%', height: '100%', background: eB.volt }} />
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div style={{ display: 'flex', gap: 0, borderBottom: `1px solid ${eB.border}`, marginBottom: 24 }}>
          {['Profile', 'Portfolio', 'Reviews · 28', 'Activity', 'Settings'].map((t, i) => (
            <div key={t} style={{
              padding: '10px 16px', fontSize: 13, color: i === 0 ? eB.fg : eB.mute, fontWeight: i === 0 ? 600 : 400,
              borderBottom: i === 0 ? `2px solid ${eB.volt}` : '2px solid transparent',
              marginBottom: -1, cursor: 'pointer',
            }}>{t}</div>
          ))}
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 320px', gap: 28 }}>
          <div>
            <SectionB num="01" title="About">
              <div style={{ fontSize: 14.5, lineHeight: 1.6, color: eB.fg, maxWidth: 620 }}>
                Independent art director, 8 years. I work with houses in craft, editorial and care on visual identity — from positioning to full systems: print, digital, signage. Previously at Studio Bonsoir & Pentagram Berlin.
              </div>
            </SectionB>

            <SectionB num="02" title="Expertise" subtitle="3 / 5 selected">
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {[
                  ['design/ui-ux', true],
                  ['art-direction', true],
                  ['brand-identity', true],
                  ['data-ai-ml', false],
                  ['3d-animation', false],
                  ['video-motion', false],
                  ['photo-av', false],
                  ['marketing-growth', false],
                  ['copywriting', false],
                  ['biz-dev', false],
                  ['consulting', false],
                  ['ux-research', false],
                ].map(([t, active]) => (
                  <span key={t} style={{
                    fontFamily: eB.mono, fontSize: 11, padding: '5px 10px', borderRadius: 4,
                    border: `1px solid ${active ? eB.volt : eB.border}`,
                    background: active ? eB.voltSoft : 'transparent',
                    color: active ? eB.volt : eB.mute,
                  }}>{t}</span>
                ))}
              </div>
            </SectionB>

            <SectionB num="03" title="Pricing">
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
                {[
                  { l: 'standard daily', v: '720', sub: '7h' },
                  { l: 'art direction', v: '950', sub: 'strategy' },
                  { l: 'workshop', v: '1,800', sub: 'half-day' },
                ].map(p => (
                  <div key={p.l} style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, padding: 14 }}>
                    <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.08em', textTransform: 'uppercase' }}>{p.l}</div>
                    <div style={{ fontFamily: eB.mono, fontSize: 26, fontWeight: 500, marginTop: 6, letterSpacing: '-0.01em' }}>€{p.v}</div>
                    <div style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim, marginTop: 4 }}>{p.sub}</div>
                  </div>
                ))}
              </div>
            </SectionB>

            <SectionB num="04" title="History" subtitle="28 missions · ★ 4.9">
              <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, overflow: 'hidden' }}>
                <div style={{ display: 'grid', gridTemplateColumns: '70px 1fr 90px 90px', padding: '8px 14px', borderBottom: `1px solid ${eB.border}`, background: eB.borderSoft }}>
                  {['ID', 'PROJECT', 'AMOUNT', 'STATUS'].map(h => (
                    <span key={h} style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em' }}>{h}</span>
                  ))}
                </div>
                {[
                  { id: 'MS-2877', c: 'Maison Fauve', t: 'Brand identity refresh', m: '€7,364', d: 'apr 15', r: 5 },
                  { id: 'MS-2810', c: 'Coddo Studio', t: 'Art direction · web', m: '€3,213', d: 'mar 1', r: 5 },
                  { id: 'MS-2742', c: 'Coop. Numa', t: 'Design system', m: '€4,280', d: 'feb 12', r: 4 },
                ].map((p, i) => (
                  <div key={p.id} style={{ display: 'grid', gridTemplateColumns: '70px 1fr 90px 90px', padding: '12px 14px', borderTop: `1px solid ${eB.borderSoft}`, alignItems: 'center' }}>
                    <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim }}>{p.id}</span>
                    <div>
                      <div style={{ fontSize: 13, fontWeight: 500 }}>{p.t}</div>
                      <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, marginTop: 2 }}>{p.c} · {p.d} · {'★'.repeat(p.r)}</div>
                    </div>
                    <span style={{ fontFamily: eB.mono, fontSize: 12 }}>{p.m}</span>
                    <span style={{ fontFamily: eB.mono, fontSize: 9, padding: '2px 6px', background: eB.greenSoft, color: eB.green, borderRadius: 3, letterSpacing: '0.08em', textTransform: 'uppercase', justifySelf: 'start' }}>delivered</span>
                  </div>
                ))}
              </div>
            </SectionB>
          </div>

          {/* sidecards */}
          <div>
            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, padding: 16, marginBottom: 12 }}>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 12 }}>Location · 06</div>
              <div style={{ fontSize: 14, marginBottom: 8 }}>Paris · France</div>
              <div style={{ display: 'flex', gap: 4 }}>
                {['Remote', 'On site', 'Hybrid'].map((m, i) => (
                  <span key={m} style={{ fontFamily: eB.mono, fontSize: 10, padding: '4px 8px', borderRadius: 3, background: i === 2 ? eB.volt : eB.borderSoft, color: i === 2 ? eB.bg : eB.mute, fontWeight: i === 2 ? 600 : 400 }}>{m}</span>
                ))}
              </div>
            </div>

            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, padding: 16, marginBottom: 12 }}>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 12 }}>Languages · 07</div>
              <div style={{ marginBottom: 10 }}>
                <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, marginBottom: 6 }}>professional</div>
                <div style={{ display: 'flex', gap: 5 }}>
                  {['FR', 'EN', 'ES'].map(l => (
                    <span key={l} style={{ fontFamily: eB.mono, fontSize: 11, padding: '4px 8px', background: eB.voltSoft, color: eB.volt, borderRadius: 3 }}>{l}</span>
                  ))}
                </div>
              </div>
              <div>
                <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, marginBottom: 6 }}>conversational</div>
                <div style={{ display: 'flex', gap: 5 }}>
                  {['DE', 'IT'].map(l => (
                    <span key={l} style={{ fontFamily: eB.mono, fontSize: 11, padding: '4px 8px', background: eB.borderSoft, color: eB.mute, borderRadius: 3 }}>{l}</span>
                  ))}
                </div>
              </div>
            </div>

            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, padding: 16 }}>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 12 }}>Skills · 08</div>
              <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                {['typography', 'figma', 'webflow', 'after-effects', 'cinema-4d', 'photoshop', 'indesign', 'illustrator'].map(s => (
                  <span key={s} style={{ fontFamily: eB.mono, fontSize: 10, padding: '3px 7px', background: eB.borderSoft, color: eB.mute, borderRadius: 3 }}>{s}</span>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </ShellStudio>
  );
}

function SectionB({ num, title, subtitle, children }) {
  return (
    <div style={{ marginBottom: 32 }}>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 12, marginBottom: 12 }}>
        <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em' }}>{num}</span>
        <span style={{ fontSize: 16, fontWeight: 600, letterSpacing: '-0.01em' }}>{title}</span>
        {subtitle && <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim, marginLeft: 'auto' }}>{subtitle}</span>}
        <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.volt, marginLeft: subtitle ? 16 : 'auto', cursor: 'pointer' }}>edit</span>
      </div>
      {children}
    </div>
  );
}

// ─────────────────────────────────────────────
// PROJECT DETAIL
// ─────────────────────────────────────────────
function ProjectDetailStudio() {
  return (
    <ShellStudio active="projects" crumbs={['Workspace', 'Projects', 'MS-2891']}>
      <div style={{ padding: '28px 32px', height: '100%', overflow: 'hidden' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 20 }}>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
              <span style={{ fontFamily: eB.mono, fontSize: 10, padding: '3px 8px', background: eB.voltSoft, color: eB.volt, borderRadius: 3, letterSpacing: '0.08em', textTransform: 'uppercase' }}>● Active</span>
              <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim, letterSpacing: '0.05em' }}>MS-2891 · created apr 20.26 · maison fauve</span>
            </div>
            <div style={{ fontSize: 36, fontWeight: 600, letterSpacing: '-0.03em', lineHeight: 1 }}>Brand identity refresh</div>
            <div style={{ fontSize: 14, color: eB.mute, marginTop: 8, maxWidth: 640 }}>Full identity rebuild for the artisan candle house — wordmark, system, packaging, signage.</div>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <button style={{ background: eB.card, border: `1px solid ${eB.border}`, color: eB.fg, padding: '8px 14px', borderRadius: 5, fontSize: 12.5 }}>Open thread</button>
            <button style={{ background: eB.volt, color: eB.bg, border: 'none', padding: '8px 14px', borderRadius: 5, fontSize: 12.5, fontWeight: 600 }}>Mark complete</button>
          </div>
        </div>

        {/* Stepper */}
        <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 8, padding: '20px 24px', marginBottom: 20 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 0 }}>
            {[
              ['Created', true, '04.20'],
              ['Accepted', true, '04.22'],
              ['Paid', true, '05.01'],
              ['Active', true, '05.01'],
              ['Delivered', false, 'in progress'],
              ['Closed', false, '—'],
            ].map(([l, done, d], i, arr) => (
              <React.Fragment key={l}>
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', minWidth: 60 }}>
                  <div style={{
                    width: 22, height: 22, borderRadius: 999,
                    background: done ? eB.volt : eB.bg,
                    border: `1px solid ${done ? eB.volt : eB.border}`,
                    color: done ? eB.bg : eB.muteDim,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontFamily: eB.mono, fontSize: 11, fontWeight: 700,
                  }}>{done ? '✓' : i + 1}</div>
                  <div style={{ fontSize: 12, fontWeight: done ? 600 : 400, color: done ? eB.fg : eB.muteDim, marginTop: 8 }}>{l}</div>
                  <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, marginTop: 2 }}>{d}</div>
                </div>
                {i < arr.length - 1 && <div style={{ flex: 1, height: 1, background: i < 3 ? eB.volt : eB.border, marginBottom: 32 }} />}
              </React.Fragment>
            ))}
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 20 }}>
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
              <span style={{ fontSize: 13, fontWeight: 600 }}>Milestones</span>
              <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim }}>3/5 delivered · €3,900 released</span>
            </div>
            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, overflow: 'hidden' }}>
              {[
                { n: 'M1', t: 'Research & moodboard', s: 'delivered', m: '€900', d: 'apr 24', col: 'green' },
                { n: 'M2', t: 'Wordmark & system', s: 'delivered', m: '€1,200', d: 'may 1', col: 'green' },
                { n: 'M3', t: 'Web art direction', s: 'in progress', m: '€1,800', d: 'may 15', col: 'volt' },
                { n: 'M4', t: 'Packaging & print', s: 'pending', m: '€2,400', d: 'jun 1', col: 'mute' },
                { n: 'M5', t: 'Brand guidelines', s: 'pending', m: '€1,064', d: 'jun 12', col: 'mute' },
              ].map((j, i) => (
                <div key={j.n} style={{ display: 'grid', gridTemplateColumns: '40px 1fr 100px 90px 70px', gap: 10, padding: '12px 16px', borderTop: i ? `1px solid ${eB.borderSoft}` : 'none', alignItems: 'center' }}>
                  <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim }}>{j.n}</span>
                  <div style={{ fontSize: 13.5, fontWeight: 500, letterSpacing: '-0.01em' }}>{j.t}</div>
                  <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim }}>{j.d}</span>
                  <span style={{ fontFamily: eB.mono, fontSize: 12, textAlign: 'right' }}>{j.m}</span>
                  <span style={{
                    fontFamily: eB.mono, fontSize: 9, padding: '3px 7px', borderRadius: 3, letterSpacing: '0.08em', textTransform: 'uppercase', textAlign: 'center',
                    background: j.col === 'green' ? eB.greenSoft : j.col === 'volt' ? eB.voltSoft : eB.borderSoft,
                    color: j.col === 'green' ? eB.green : j.col === 'volt' ? eB.volt : eB.mute,
                  }}>{j.s}</span>
                </div>
              ))}
            </div>

            <div style={{ marginTop: 20, fontSize: 13, fontWeight: 600, marginBottom: 10 }}>Activity</div>
            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, padding: 14 }}>
              {[
                ['M2 marked delivered', 'Léa Marchand', '2h ago'],
                ['€1,200 released to wallet', 'system', '2h ago'],
                ['M3 kicked off', 'Maison Fauve', 'yesterday'],
              ].map(([t, w, d], i) => (
                <div key={i} style={{ display: 'flex', alignItems: 'baseline', gap: 12, padding: '6px 0', borderTop: i ? `1px solid ${eB.borderSoft}` : 'none' }}>
                  <span style={{ width: 6, height: 6, borderRadius: '50%', background: eB.volt, flexShrink: 0, marginTop: 5 }} />
                  <span style={{ fontSize: 12.5, flex: 1 }}>{t}</span>
                  <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim }}>{w} · {d}</span>
                </div>
              ))}
            </div>
          </div>

          <div>
            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, padding: 18, marginBottom: 12 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase' }}>Total contract</span>
                <span style={{ fontFamily: eB.mono, fontSize: 9, padding: '3px 7px', background: eB.greenSoft, color: eB.green, borderRadius: 3, letterSpacing: '0.08em', textTransform: 'uppercase' }}>● escrowed</span>
              </div>
              <div style={{ fontFamily: eB.mono, fontSize: 38, fontWeight: 500, letterSpacing: '-0.02em', lineHeight: 1 }}>€7,364</div>

              <div style={{ marginTop: 16, paddingTop: 14, borderTop: `1px solid ${eB.borderSoft}` }}>
                <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 10 }}>Platform fees</div>
                {[['< €200', '€9.00'], ['€200 – €1,000', '€15.00'], ['> €1,000', '€25.00', true]].map(([l, v, hl]) => (
                  <div key={l} style={{ display: 'flex', justifyContent: 'space-between', padding: '5px 0', fontFamily: eB.mono, fontSize: 11.5, color: hl ? eB.volt : eB.mute, paddingLeft: hl ? 8 : 0, borderLeft: hl ? `2px solid ${eB.volt}` : 'none' }}>
                    <span>{l}</span><span>{v}</span>
                  </div>
                ))}
                <div style={{ marginTop: 10, paddingTop: 10, borderTop: `1px solid ${eB.borderSoft}`, display: 'flex', justifyContent: 'space-between' }}>
                  <span style={{ fontSize: 12.5, fontWeight: 600 }}>You receive</span>
                  <span style={{ fontFamily: eB.mono, fontSize: 13, color: eB.volt }}>€7,339.00</span>
                </div>
              </div>
            </div>

            <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, padding: 16 }}>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 12 }}>Participants</div>
              {[
                { i: 'M', n: 'Maison Fauve', r: 'client', c: eB.ember },
                { i: 'L', n: 'Léa Marchand', r: 'provider', c: eB.volt },
              ].map(p => (
                <div key={p.n} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '7px 0' }}>
                  <div style={{ width: 28, height: 28, borderRadius: 4, background: p.c, color: eB.bg, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 12 }}>{p.i}</div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 13, fontWeight: 500 }}>{p.n}</div>
                    <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.08em' }}>{p.r}</div>
                  </div>
                  <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.volt }}>msg →</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </ShellStudio>
  );
}

// ─────────────────────────────────────────────
// WALLET
// ─────────────────────────────────────────────
function WalletStudio() {
  return (
    <ShellStudio active="wallet" crumbs={['Workspace', 'Wallet']}>
      <div style={{ padding: '28px 32px', height: '100%', overflow: 'hidden' }}>
        {/* Big balance card */}
        <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 10, padding: 28, marginBottom: 20, position: 'relative', overflow: 'hidden' }}>
          {/* Subtle grid pattern */}
          <div style={{ position: 'absolute', inset: 0, backgroundImage: `linear-gradient(${eB.borderSoft} 1px, transparent 1px), linear-gradient(90deg, ${eB.borderSoft} 1px, transparent 1px)`, backgroundSize: '32px 32px', opacity: 0.6 }} />
          <div style={{ position: 'relative', display: 'grid', gridTemplateColumns: '1.5fr 1fr', gap: 32, alignItems: 'flex-end' }}>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
                <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.12em', textTransform: 'uppercase' }}>Total earned · 2026</span>
                <span style={{ fontFamily: eB.mono, fontSize: 9, padding: '3px 7px', background: eB.greenSoft, color: eB.green, borderRadius: 3, letterSpacing: '0.08em', textTransform: 'uppercase' }}>● stripe verified</span>
              </div>
              <div style={{ fontFamily: eB.mono, fontSize: 72, fontWeight: 400, letterSpacing: '-0.04em', lineHeight: 0.95 }}>
                €10,502<span style={{ color: eB.muteDim, fontSize: 28 }}>.00</span>
              </div>
              <div style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim, marginTop: 8 }}>IBAN ••••8420 · payouts enabled</div>
            </div>
            <div style={{ background: eB.bg, border: `1px solid ${eB.border}`, borderRadius: 6, padding: 18 }}>
              <div style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em', textTransform: 'uppercase', marginBottom: 6 }}>Available now</div>
              <div style={{ fontFamily: eB.mono, fontSize: 32, lineHeight: 1, marginBottom: 14 }}>€0.00</div>
              <button style={{ width: '100%', background: eB.borderSoft, color: eB.muteDim, border: `1px solid ${eB.border}`, padding: '10px 16px', borderRadius: 5, fontSize: 12.5, fontWeight: 600 }}>No funds to withdraw</button>
            </div>
          </div>
        </div>

        {/* 3 stat cells */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 0, marginBottom: 24, border: `1px solid ${eB.border}`, borderRadius: 8, overflow: 'hidden' }}>
          {[
            { l: 'In escrow', v: '€0.00', d: 'awaiting completion', c: eB.fg, ic: '◷' },
            { l: 'Available', v: '€0.00', d: 'ready to withdraw', c: eB.fg, ic: '◉' },
            { l: 'Transferred', v: '€10,502.00', d: 'sent to bank', c: eB.green, ic: '↗' },
          ].map((s, i) => (
            <div key={i} style={{ padding: '20px 24px', background: eB.card, borderRight: i < 2 ? `1px solid ${eB.border}` : 'none' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
                <span style={{ fontSize: 13, color: s.c }}>{s.ic}</span>
                <span style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.12em', textTransform: 'uppercase' }}>{s.l}</span>
              </div>
              <div style={{ fontFamily: eB.mono, fontSize: 26, lineHeight: 1, color: s.c }}>{s.v}</div>
              <div style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim, marginTop: 8 }}>{s.d}</div>
            </div>
          ))}
        </div>

        {/* Transactions table */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
          <span style={{ fontSize: 14, fontWeight: 600 }}>Mission history</span>
          <div style={{ display: 'flex', gap: 8 }}>
            <div style={{ display: 'flex', gap: 4, background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 4, padding: 2 }}>
              {['All', 'Escrow', 'Transferred'].map((t, i) => (
                <span key={t} style={{ fontFamily: eB.mono, fontSize: 10, padding: '4px 10px', borderRadius: 3, background: i === 0 ? eB.borderSoft : 'transparent', color: i === 0 ? eB.fg : eB.muteDim, letterSpacing: '0.05em' }}>{t}</span>
              ))}
            </div>
            <button style={{ background: eB.card, border: `1px solid ${eB.border}`, color: eB.fg, padding: '6px 12px', borderRadius: 4, fontFamily: eB.mono, fontSize: 11, letterSpacing: '0.05em' }}>EXPORT CSV</button>
          </div>
        </div>

        <div style={{ background: eB.card, border: `1px solid ${eB.border}`, borderRadius: 6, overflow: 'hidden' }}>
          <div style={{ display: 'grid', gridTemplateColumns: '90px 1.5fr 1fr 110px 110px 90px', padding: '8px 16px', borderBottom: `1px solid ${eB.border}`, background: eB.borderSoft }}>
            {['DATE', 'PROJECT', 'CLIENT', 'FEES', 'NET', 'STATUS'].map(h => (
              <span key={h} style={{ fontFamily: eB.mono, fontSize: 10, color: eB.muteDim, letterSpacing: '0.1em' }}>{h}</span>
            ))}
          </div>
          {[
            { d: '05.01.26', t: 'Brand identity refresh', c: 'Maison Fauve', f: '−€25.00', m: '€7,339.00', s: 'transferred', col: 'green' },
            { d: '04.28.26', t: 'Web art direction', c: 'Coddo Studio', f: '−€25.00', m: '€3,188.00', s: 'transferred', col: 'green' },
            { d: '04.15.26', t: 'Identity workshop', c: 'Atelier Nour', f: '−€15.00', m: '€1,785.00', s: 'in escrow', col: 'volt' },
            { d: '04.02.26', t: 'Design system audit', c: 'Coop. Numa', f: '−€25.00', m: '€4,255.00', s: 'transferred', col: 'green' },
            { d: '03.28.26', t: 'Brand strategy', c: 'Maison Pivoine', f: '−€25.00', m: '€2,375.00', s: 'transferred', col: 'green' },
          ].map((r, i) => (
            <div key={i} style={{ display: 'grid', gridTemplateColumns: '90px 1.5fr 1fr 110px 110px 90px', padding: '11px 16px', borderTop: `1px solid ${eB.borderSoft}`, alignItems: 'center' }}>
              <span style={{ fontFamily: eB.mono, fontSize: 11, color: eB.muteDim }}>{r.d}</span>
              <span style={{ fontSize: 13, fontWeight: 500 }}>{r.t}</span>
              <span style={{ fontFamily: eB.mono, fontSize: 11.5, color: eB.mute }}>{r.c}</span>
              <span style={{ fontFamily: eB.mono, fontSize: 11.5, color: eB.muteDim }}>{r.f}</span>
              <span style={{ fontFamily: eB.mono, fontSize: 12.5, color: eB.fg }}>{r.m}</span>
              <span style={{
                fontFamily: eB.mono, fontSize: 9, padding: '3px 7px', borderRadius: 3, letterSpacing: '0.08em', textTransform: 'uppercase', textAlign: 'center', justifySelf: 'start',
                background: r.col === 'green' ? eB.greenSoft : eB.voltSoft,
                color: r.col === 'green' ? eB.green : eB.volt,
              }}>● {r.s}</span>
            </div>
          ))}
        </div>
      </div>
    </ShellStudio>
  );
}

Object.assign(window, {
  DashboardStudio, FindFreelancersStudio, ProfileStudio, ProjectDetailStudio, WalletStudio,
});
