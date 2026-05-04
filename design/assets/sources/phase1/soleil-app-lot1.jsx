// App Lot 1 — Activité : Dashboard freelance, Dashboard entreprise, Mes candidatures, Détail mission
// Reprend le langage Soleil : éditorial sur les "premiers" écrans (dashboards) + dense sur les listes.

const SL1 = window.S;
const SL1I = window.SI;
const SL1Portrait = window.Portrait;
const _AppFrame_L1 = window.AppFrame;
const _AppTabBar_L1 = window.AppTabBar;

// ─── Dashboard FREELANCE ──────────────────────────────────────
function AppDashboardFreelance() {
  return (
    <_AppFrame_L1>
      {/* Header */}
      <div style={{ flexShrink: 0, padding: '6px 20px 16px', background: SL1.bg }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 11 }}>
            <SL1Portrait id={0} size={42} rounded={12} />
            <div>
              <div style={{ fontSize: 11, color: SL1.textMute, fontFamily: SL1.serif, fontStyle: 'italic' }}>Bonjour Camille,</div>
              <div style={{ fontSize: 14, fontWeight: 600, color: SL1.text }}>jeudi 22 mai</div>
            </div>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <button style={{ width: 38, height: 38, borderRadius: '50%', background: '#fff', border: `1px solid ${SL1.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', position: 'relative' }}>
              <SL1I name="Bell" size={17} />
              <span style={{ position: 'absolute', top: 8, right: 9, width: 8, height: 8, borderRadius: '50%', background: SL1.accent, border: '2px solid #fff' }} />
            </button>
          </div>
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px', display: 'flex', flexDirection: 'column', gap: 14 }}>
        {/* Hero — Mission active */}
        <div style={{ background: '#fff', border: `1px solid ${SL1.border}`, borderRadius: 18, padding: 16, position: 'relative', overflow: 'hidden' }}>
          <div style={{ position: 'absolute', top: -30, right: -30, width: 140, height: 140, borderRadius: '50%', background: `radial-gradient(circle, ${SL1.accentSoft}, transparent 70%)` }} />
          <div style={{ position: 'relative' }}>
            <div style={{ fontSize: 11, color: SL1.accentDeep, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase' }}>Mission en cours</div>
            <div style={{ fontFamily: SL1.serif, fontSize: 21, fontWeight: 600, letterSpacing: '-0.01em', color: SL1.text, marginTop: 4, lineHeight: 1.15 }}>Refonte app Helio</div>
            <div style={{ fontSize: 12.5, color: SL1.textMute, marginTop: 3 }}>pour Léa Bertrand · Helio</div>

            {/* Progression */}
            <div style={{ marginTop: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, color: SL1.textMute, marginBottom: 6 }}>
                <span>Jalon 2 sur 4</span>
                <span style={{ fontFamily: SL1.mono, color: SL1.text, fontWeight: 600 }}>50 %</span>
              </div>
              <div style={{ height: 6, background: SL1.bg, borderRadius: 3, overflow: 'hidden' }}>
                <div style={{ width: '50%', height: '100%', background: SL1.accent, borderRadius: 3 }} />
              </div>
            </div>

            <div style={{ display: 'flex', gap: 8, marginTop: 14, paddingTop: 14, borderTop: `1px solid ${SL1.border}` }}>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 10, color: SL1.textSubtle, fontWeight: 600, letterSpacing: '0.04em' }}>PROCHAIN JALON</div>
                <div style={{ fontSize: 13, color: SL1.text, marginTop: 2, fontWeight: 600 }}>12 juin</div>
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 10, color: SL1.textSubtle, fontWeight: 600, letterSpacing: '0.04em' }}>EN ATTENTE</div>
                <div style={{ fontSize: 13, color: SL1.text, marginTop: 2, fontFamily: SL1.mono, fontWeight: 600 }}>2 400 €</div>
              </div>
            </div>
          </div>
        </div>

        {/* Stats triplet */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8 }}>
          {[
            { l: 'Revenus mois', v: '4 200 €', sub: '+18 %', good: true },
            { l: 'Candidatures', v: '7', sub: '3 vues' },
            { l: 'Note moyenne', v: '4.9', sub: '87 avis', star: true },
          ].map((s, i) => (
            <div key={i} style={{ background: '#fff', border: `1px solid ${SL1.border}`, borderRadius: 14, padding: '12px 10px' }}>
              <div style={{ fontSize: 10, color: SL1.textSubtle, letterSpacing: '0.04em', fontWeight: 600 }}>{s.l.toUpperCase()}</div>
              <div style={{ fontFamily: SL1.serif, fontSize: 18, fontWeight: 600, color: SL1.text, marginTop: 4, display: 'flex', alignItems: 'baseline', gap: 3 }}>
                {s.star ? <span style={{ color: SL1.accent, fontSize: 14 }}>★</span> : null}{s.v}
              </div>
              <div style={{ fontSize: 10.5, color: s.good ? SL1.green : SL1.textMute, marginTop: 2, fontWeight: 500 }}>{s.sub}</div>
            </div>
          ))}
        </div>

        {/* Section : Aujourd'hui */}
        <div>
          <div style={{ fontFamily: SL1.serif, fontSize: 17, fontWeight: 600, letterSpacing: '-0.01em', color: SL1.text, padding: '6px 0 10px' }}>Aujourd'hui</div>
          <div style={{ background: '#fff', border: `1px solid ${SL1.border}`, borderRadius: 14, overflow: 'hidden' }}>
            {[
              { time: '10:00', icon: 'Video', label: 'Atelier kickoff Helio', sub: 'Avec Léa Bertrand · 1h', accent: true },
              { time: '14:30', icon: 'Doc', label: 'Livrer maquettes onboarding', sub: 'Mission Helio · jalon 2' },
              { time: '17:00', icon: 'Chat', label: 'Répondre à 3 messages', sub: '2 prospects, 1 client' },
            ].map((item, i, a) => (
              <div key={i} style={{ display: 'flex', gap: 12, padding: '12px 14px', borderBottom: i < a.length - 1 ? `1px solid ${SL1.border}` : 'none', alignItems: 'center' }}>
                <div style={{ fontFamily: SL1.mono, fontSize: 11, color: SL1.textMute, width: 38, fontWeight: 600 }}>{item.time}</div>
                <div style={{ width: 34, height: 34, borderRadius: 10, background: item.accent ? SL1.accentSoft : SL1.bg, color: item.accent ? SL1.accent : SL1.text, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  <SL1I name={item.icon} size={15} />
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: SL1.text }}>{item.label}</div>
                  <div style={{ fontSize: 11, color: SL1.textMute, marginTop: 1 }}>{item.sub}</div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Opportunités suggérées */}
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', padding: '6px 0 10px' }}>
            <div style={{ fontFamily: SL1.serif, fontSize: 17, fontWeight: 600, letterSpacing: '-0.01em', color: SL1.text }}>Pour vous</div>
            <span style={{ fontSize: 12, color: SL1.accent, fontWeight: 600 }}>Tout voir →</span>
          </div>
          <div style={{ display: 'flex', gap: 10, overflowX: 'auto', paddingBottom: 4, marginRight: -20 }}>
            {[
              { title: 'Refonte design system B2B SaaS', co: 'Trellis', tjm: '650 €', dur: '3 mois', match: 96 },
              { title: 'Brand designer freelance', co: 'Atelier Mure', tjm: '550 €', dur: '6 sem', match: 88 },
            ].map((op, i) => (
              <div key={i} style={{ flexShrink: 0, width: 260, background: '#fff', border: `1px solid ${SL1.border}`, borderRadius: 14, padding: 14 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <span style={{ background: SL1.greenSoft, color: SL1.green, padding: '3px 8px', borderRadius: 999, fontSize: 10.5, fontWeight: 700 }}>Match {op.match} %</span>
                  <SL1I name="Bookmark" size={15} />
                </div>
                <div style={{ fontFamily: SL1.serif, fontSize: 14.5, fontWeight: 600, color: SL1.text, marginTop: 10, lineHeight: 1.25 }}>{op.title}</div>
                <div style={{ fontSize: 11.5, color: SL1.textMute, marginTop: 3 }}>{op.co}</div>
                <div style={{ display: 'flex', gap: 8, marginTop: 12, fontSize: 11, color: SL1.text }}>
                  <span style={{ fontFamily: SL1.mono, fontWeight: 600 }}>{op.tjm}/j</span>
                  <span style={{ color: SL1.textSubtle }}>·</span>
                  <span>{op.dur}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      <_AppTabBar_L1 active="home" />
    </_AppFrame_L1>
  );
}

// ─── Dashboard ENTREPRISE ──────────────────────────────────────
function AppDashboardEntreprise() {
  return (
    <_AppFrame_L1>
      {/* Header */}
      <div style={{ flexShrink: 0, padding: '6px 20px 16px', background: SL1.bg }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontSize: 11, color: SL1.textMute, fontFamily: SL1.serif, fontStyle: 'italic' }}>Bonjour Léa,</div>
            <div style={{ fontFamily: SL1.serif, fontSize: 24, fontWeight: 600, letterSpacing: '-0.02em', color: SL1.text, marginTop: 2 }}>Helio · 3 projets actifs</div>
          </div>
          <div style={{ width: 38, height: 38, borderRadius: '50%', background: '#fff', border: `1px solid ${SL1.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', position: 'relative' }}>
            <SL1I name="Bell" size={17} />
            <span style={{ position: 'absolute', top: 8, right: 9, width: 8, height: 8, borderRadius: '50%', background: SL1.accent, border: '2px solid #fff' }} />
          </div>
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px', display: 'flex', flexDirection: 'column', gap: 14 }}>
        {/* Hero — KPIs paiement */}
        <div style={{ background: SL1.text, color: '#fff', borderRadius: 18, padding: 18, position: 'relative', overflow: 'hidden' }}>
          <div style={{ position: 'absolute', top: -40, right: -40, width: 160, height: 160, borderRadius: '50%', background: `radial-gradient(circle, rgba(232,93,74,0.4), transparent 70%)` }} />
          <div style={{ position: 'relative' }}>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.6)', letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase' }}>Engagement ce mois</div>
            <div style={{ fontFamily: SL1.serif, fontSize: 32, fontWeight: 600, letterSpacing: '-0.02em', marginTop: 4 }}>18 400 €</div>
            <div style={{ display: 'flex', gap: 16, marginTop: 14, fontSize: 12 }}>
              <div>
                <div style={{ color: 'rgba(255,255,255,0.55)', fontSize: 10.5 }}>VERSÉS</div>
                <div style={{ fontFamily: SL1.mono, fontWeight: 600, marginTop: 2 }}>9 600 €</div>
              </div>
              <div style={{ width: 1, background: 'rgba(255,255,255,0.15)' }} />
              <div>
                <div style={{ color: 'rgba(255,255,255,0.55)', fontSize: 10.5 }}>SÉQUESTRE</div>
                <div style={{ fontFamily: SL1.mono, fontWeight: 600, marginTop: 2 }}>6 000 €</div>
              </div>
              <div style={{ width: 1, background: 'rgba(255,255,255,0.15)' }} />
              <div>
                <div style={{ color: 'rgba(255,255,255,0.55)', fontSize: 10.5 }}>À PRÉVOIR</div>
                <div style={{ fontFamily: SL1.mono, fontWeight: 600, marginTop: 2 }}>2 800 €</div>
              </div>
            </div>
          </div>
        </div>

        {/* Stats */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
          {[
            { l: 'Annonces ouvertes', v: '4', sub: '12 nouvelles candid.' },
            { l: 'Freelances actifs', v: '6', sub: 'sur 3 projets' },
          ].map((s, i) => (
            <div key={i} style={{ background: '#fff', border: `1px solid ${SL1.border}`, borderRadius: 14, padding: 12 }}>
              <div style={{ fontSize: 10.5, color: SL1.textSubtle, letterSpacing: '0.04em', fontWeight: 600 }}>{s.l.toUpperCase()}</div>
              <div style={{ fontFamily: SL1.serif, fontSize: 22, fontWeight: 600, color: SL1.text, marginTop: 4 }}>{s.v}</div>
              <div style={{ fontSize: 11, color: SL1.textMute, marginTop: 2 }}>{s.sub}</div>
            </div>
          ))}
        </div>

        {/* Mes projets */}
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', padding: '6px 0 10px' }}>
            <div style={{ fontFamily: SL1.serif, fontSize: 17, fontWeight: 600, color: SL1.text }}>Mes projets</div>
            <span style={{ fontSize: 12, color: SL1.accent, fontWeight: 600 }}>Tout voir →</span>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {[
              { name: 'Refonte app Helio', f: 'Camille Dubois', fId: 0, jalon: 'Jalon 2/4', pct: 50, accent: true },
              { name: 'Identité visuelle', f: 'Marion Lefèvre', fId: 1, jalon: 'Jalon 1/3', pct: 33 },
              { name: 'Setup analytics', f: 'Yacine Benali', fId: 2, jalon: 'Jalon 3/3', pct: 90 },
            ].map((p, i) => (
              <div key={i} style={{ background: '#fff', border: `1px solid ${SL1.border}`, borderRadius: 14, padding: 14 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <div>
                    <div style={{ fontSize: 13.5, fontWeight: 600, color: SL1.text }}>{p.name}</div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 6 }}>
                      <SL1Portrait id={p.fId} size={20} />
                      <span style={{ fontSize: 11.5, color: SL1.textMute }}>avec {p.f}</span>
                    </div>
                  </div>
                  <span style={{ fontSize: 11, color: p.accent ? SL1.accentDeep : SL1.textMute, fontWeight: 600 }}>{p.jalon}</span>
                </div>
                <div style={{ marginTop: 12, height: 4, background: SL1.bg, borderRadius: 2, overflow: 'hidden' }}>
                  <div style={{ width: `${p.pct}%`, height: '100%', background: SL1.accent, borderRadius: 2 }} />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Candidatures à examiner */}
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', padding: '6px 0 10px' }}>
            <div style={{ fontFamily: SL1.serif, fontSize: 17, fontWeight: 600, color: SL1.text }}>À examiner</div>
            <span style={{ fontSize: 11, color: SL1.accent, background: SL1.accentSoft, padding: '3px 8px', borderRadius: 999, fontWeight: 700 }}>12 nouvelles</span>
          </div>
          <div style={{ background: '#fff', border: `1px solid ${SL1.border}`, borderRadius: 14, overflow: 'hidden' }}>
            {[
              { name: 'Sofia Lambert', tag: 'Designer UX/UI', new: true, id: 4 },
              { name: 'Théo Martinet', tag: 'Motion designer', new: true, id: 3 },
              { name: 'Marc Olivier', tag: 'Développeur back', id: 5 },
            ].map((c, i, a) => (
              <div key={i} style={{ display: 'flex', gap: 11, padding: '11px 14px', borderBottom: i < a.length - 1 ? `1px solid ${SL1.border}` : 'none', alignItems: 'center' }}>
                <SL1Portrait id={c.id} size={36} />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: SL1.text, display: 'flex', alignItems: 'center', gap: 6 }}>
                    {c.name}
                    {c.new ? <span style={{ width: 6, height: 6, borderRadius: '50%', background: SL1.accent }} /> : null}
                  </div>
                  <div style={{ fontSize: 11.5, color: SL1.textMute }}>{c.tag}</div>
                </div>
                <SL1I name="ChevronRight" size={16} />
              </div>
            ))}
          </div>
        </div>
      </div>

      <_AppTabBar_L1 active="home" />
    </_AppFrame_L1>
  );
}

// ─── Mes candidatures (FREELANCE) ──────────────────────────────
function AppCandidatures() {
  const apps = [
    { title: 'Refonte design system B2B SaaS', co: 'Trellis', status: 'Vue', statusColor: 'amber', date: 'Il y a 2 j', tjm: '650 €' },
    { title: 'Brand designer freelance', co: 'Atelier Mure', status: 'En discussion', statusColor: 'green', date: 'Il y a 4 j', tjm: '550 €', unread: 2 },
    { title: 'UX writer pour app fintech', co: 'Pact', status: 'Refusée', statusColor: 'mute', date: 'Il y a 1 sem', tjm: '480 €' },
    { title: 'Product designer senior', co: 'Helio', status: 'Acceptée', statusColor: 'green-strong', date: 'Il y a 2 sem', tjm: '600 €' },
    { title: 'Direction artistique', co: 'Verso', status: 'En attente', statusColor: 'mute', date: 'Il y a 3 sem', tjm: '500 €' },
  ];
  const colorMap = {
    'amber': { bg: '#fbf0dc', text: SL1.amber },
    'green': { bg: SL1.greenSoft, text: SL1.green },
    'green-strong': { bg: SL1.green, text: '#fff' },
    'mute': { bg: SL1.bg, text: SL1.textMute },
  };
  return (
    <_AppFrame_L1>
      <div style={{ flexShrink: 0, padding: '6px 20px 14px', background: SL1.bg }}>
        <div style={{ fontFamily: SL1.serif, fontSize: 26, fontWeight: 600, letterSpacing: '-0.02em', color: SL1.text }}>Mes candidatures</div>
        <div style={{ fontSize: 12.5, color: SL1.textMute, fontFamily: SL1.serif, fontStyle: 'italic', marginTop: 2 }}>5 actives · 2 nouvelles réponses</div>
      </div>

      {/* Tabs */}
      <div style={{ flexShrink: 0, padding: '8px 20px 0', background: SL1.bg, display: 'flex', gap: 6, overflowX: 'auto' }}>
        {[
          { l: 'Toutes', n: 12, active: true },
          { l: 'En cours', n: 5 },
          { l: 'Acceptées', n: 4 },
          { l: 'Refusées', n: 3 },
        ].map(t => (
          <span key={t.l} style={{ padding: '6px 12px', borderRadius: 999, fontSize: 12, fontWeight: 600, whiteSpace: 'nowrap', background: t.active ? SL1.text : '#fff', color: t.active ? '#fff' : SL1.textMute, border: t.active ? 'none' : `1px solid ${SL1.border}` }}>{t.l} <span style={{ opacity: 0.6, marginLeft: 2 }}>{t.n}</span></span>
        ))}
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '14px 20px 20px', display: 'flex', flexDirection: 'column', gap: 10 }}>
        {apps.map((a, i) => {
          const c = colorMap[a.statusColor];
          return (
            <div key={i} style={{ background: '#fff', border: `1px solid ${SL1.border}`, borderRadius: 14, padding: 14, position: 'relative' }}>
              {a.unread ? <div style={{ position: 'absolute', top: 12, right: 12, background: SL1.accent, color: '#fff', fontSize: 10, fontWeight: 700, padding: '2px 7px', borderRadius: 999 }}>{a.unread} non lus</div> : null}
              <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12 }}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontFamily: SL1.serif, fontSize: 15, fontWeight: 600, color: SL1.text, lineHeight: 1.25 }}>{a.title}</div>
                  <div style={{ fontSize: 11.5, color: SL1.textMute, marginTop: 3 }}>{a.co} · {a.date}</div>
                </div>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 12, paddingTop: 11, borderTop: `1px dashed ${SL1.border}` }}>
                <span style={{ background: c.bg, color: c.text, padding: '4px 10px', borderRadius: 999, fontSize: 11, fontWeight: 600 }}>{a.status}</span>
                <span style={{ fontFamily: SL1.mono, fontSize: 12.5, color: SL1.text, fontWeight: 600 }}>{a.tjm}/j</span>
              </div>
            </div>
          );
        })}
      </div>

      <_AppTabBar_L1 active="home" />
    </_AppFrame_L1>
  );
}

// ─── Détail mission (FREELANCE — livrer un jalon) ─────────────
function AppMissionDetail() {
  return (
    <_AppFrame_L1 bg="#fff">
      {/* Header */}
      <div style={{ flexShrink: 0, padding: '6px 14px 12px', background: '#fff', borderBottom: `1px solid ${SL1.border}`, display: 'flex', alignItems: 'center', gap: 10 }}>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SL1.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL1I name="ArrowLeft" size={18} />
        </button>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 11, color: SL1.textMute }}>Mission · Helio</div>
          <div style={{ fontSize: 14, fontWeight: 600, color: SL1.text, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>Refonte app Helio</div>
        </div>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SL1.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL1I name="MoreH" size={17} />
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        {/* Hero : titre + montant */}
        <div style={{ padding: '20px 20px 16px', background: SL1.bg }}>
          <div style={{ fontSize: 11, color: SL1.accentDeep, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase' }}>Mission active</div>
          <div style={{ fontFamily: SL1.serif, fontSize: 24, fontWeight: 600, letterSpacing: '-0.015em', color: SL1.text, marginTop: 4, lineHeight: 1.15 }}>Refonte de l'application mobile Helio</div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 9, marginTop: 12 }}>
            <SL1Portrait id={2} size={32} />
            <div>
              <div style={{ fontSize: 12.5, fontWeight: 600, color: SL1.text }}>Léa Bertrand</div>
              <div style={{ fontSize: 11, color: SL1.textMute }}>Head of Product, Helio</div>
            </div>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 6, marginTop: 16, paddingTop: 14, borderTop: `1px solid ${SL1.border}` }}>
            <div>
              <div style={{ fontSize: 9.5, color: SL1.textSubtle, letterSpacing: '0.06em', fontWeight: 600 }}>MONTANT</div>
              <div style={{ fontFamily: SL1.mono, fontSize: 13, color: SL1.text, marginTop: 2, fontWeight: 600 }}>9 600 €</div>
            </div>
            <div>
              <div style={{ fontSize: 9.5, color: SL1.textSubtle, letterSpacing: '0.06em', fontWeight: 600 }}>DURÉE</div>
              <div style={{ fontSize: 13, color: SL1.text, marginTop: 2, fontWeight: 600 }}>3 mois</div>
            </div>
            <div>
              <div style={{ fontSize: 9.5, color: SL1.textSubtle, letterSpacing: '0.06em', fontWeight: 600 }}>FIN</div>
              <div style={{ fontSize: 13, color: SL1.text, marginTop: 2, fontWeight: 600 }}>15 août</div>
            </div>
          </div>
        </div>

        {/* Stepper */}
        <div style={{ padding: '20px 20px 0' }}>
          <div style={{ fontSize: 11, color: SL1.textSubtle, letterSpacing: '0.06em', fontWeight: 600, textTransform: 'uppercase', marginBottom: 14 }}>Jalons</div>
          {[
            { label: 'Discovery + Audit', amount: '2 400 €', date: '15 mai', state: 'done' },
            { label: 'Maquettes onboarding', amount: '2 400 €', date: '12 juin', state: 'current' },
            { label: 'Maquettes parcours principal', amount: '2 400 €', date: '15 juil', state: 'pending' },
            { label: 'Design system + handoff', amount: '2 400 €', date: '15 août', state: 'pending' },
          ].map((j, i, a) => (
            <div key={i} style={{ display: 'flex', gap: 12, paddingBottom: 14 }}>
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 0 }}>
                <div style={{ width: 26, height: 26, borderRadius: '50%', background: j.state === 'done' ? SL1.green : j.state === 'current' ? SL1.accent : '#fff', border: j.state === 'pending' ? `2px solid ${SL1.border}` : 'none', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, fontWeight: 700, flexShrink: 0 }}>
                  {j.state === 'done' ? <svg width="11" height="11" viewBox="0 0 12 12"><path d="M2 6l3 3 5-6" stroke="#fff" strokeWidth="2.2" fill="none" strokeLinecap="round" strokeLinejoin="round"/></svg> : i + 1}
                </div>
                {i < a.length - 1 ? <div style={{ width: 2, flex: 1, background: j.state === 'done' ? SL1.green : SL1.border, marginTop: 2 }} /> : null}
              </div>
              <div style={{ flex: 1, paddingBottom: i < a.length - 1 ? 8 : 0 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 8 }}>
                  <div style={{ fontSize: 13.5, fontWeight: 600, color: j.state === 'pending' ? SL1.textMute : SL1.text }}>{j.label}</div>
                  <div style={{ fontFamily: SL1.mono, fontSize: 12, color: j.state === 'pending' ? SL1.textSubtle : SL1.text, fontWeight: 600 }}>{j.amount}</div>
                </div>
                <div style={{ fontSize: 11, color: SL1.textMute, marginTop: 3 }}>
                  {j.state === 'done' ? `Validé le ${j.date}` : j.state === 'current' ? `Échéance ${j.date}` : `Prévu ${j.date}`}
                </div>
                {j.state === 'current' ? (
                  <div style={{ background: SL1.accentSoft, border: `1px solid ${SL1.accent}33`, borderRadius: 12, padding: 12, marginTop: 10 }}>
                    <div style={{ fontSize: 11.5, color: SL1.accentDeep, fontWeight: 600 }}>📎 Vous avez 1 fichier en brouillon</div>
                    <div style={{ fontSize: 11, color: SL1.text, marginTop: 6 }}>maquettes-onboarding-v3.fig</div>
                    <button style={{ marginTop: 10, width: '100%', padding: '9px 12px', background: SL1.accent, color: '#fff', border: 'none', borderRadius: 10, fontSize: 12.5, fontWeight: 600, fontFamily: SL1.sans }}>Livrer ce jalon →</button>
                  </div>
                ) : null}
              </div>
            </div>
          ))}
        </div>

        {/* Brief */}
        <div style={{ padding: '8px 20px 20px' }}>
          <div style={{ fontSize: 11, color: SL1.textSubtle, letterSpacing: '0.06em', fontWeight: 600, textTransform: 'uppercase', marginBottom: 10 }}>Brief</div>
          <p style={{ fontSize: 13, lineHeight: 1.55, color: SL1.text, margin: 0 }}>
            Refondre l'app mobile Helio (iOS + Android) pour améliorer l'onboarding, simplifier le parcours principal et poser un design system Flutter Material 3 robuste sur 6 écrans clés.
          </p>
        </div>
      </div>
    </_AppFrame_L1>
  );
}

Object.assign(window, {
  AppDashboardFreelance, AppDashboardEntreprise, AppCandidatures, AppMissionDetail,
});
