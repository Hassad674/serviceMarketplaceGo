// Lot C mobile — Dashboard, Opportunités, Détail oppo, Candidatures, Mission
const SCM = window.S;
const SCMI = window.SI;
const SCMPortrait = window.Portrait;
const { MobileFrame, MobileHeader, MobileBottomNav, MobileSegmented, MobileListItem, MobileFab } = window;

// ─── CM1 — Dashboard freelance mobile ─────────────────────────
function SoleilFreelancerDashboardMobile() {
  return (
    <MobileFrame url="atelier.fr">
      <div style={{ padding: '18px 18px 14px', background: '#fff', borderBottom: `1px solid ${SCM.border}`, display: 'flex', alignItems: 'center', gap: 12 }}>
        <SCMPortrait id={1} size={40} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 11, color: SCM.textMute, fontFamily: SCM.mono }}>Mardi · 14 mai</div>
          <div style={{ fontSize: 16, fontWeight: 600, fontFamily: SCM.serif }}>Bonjour Élise</div>
        </div>
        <button style={{ width: 38, height: 38, borderRadius: '50%', background: SCM.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', position: 'relative' }}>
          <SCMI name="Bell" size={17} />
          <span style={{ position: 'absolute', top: 6, right: 8, width: 7, height: 7, borderRadius: '50%', background: SCM.accent }} />
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '16px 16px 90px' }}>
        <h1 style={{ fontFamily: SCM.serif, fontSize: 24, lineHeight: 1.15, margin: 0, fontWeight: 400, letterSpacing: '-0.02em', marginBottom: 16 }}>Belle <span style={{ fontStyle: 'italic', color: SCM.accent }}>semaine</span> qui s'annonce.</h1>

        {/* Stats inline */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 20 }}>
          {[
            { l: 'Revenus du mois', v: '4 920 €', accent: true },
            { l: 'Missions actives', v: '3' },
            { l: 'Candidatures', v: '7', sub: '2 en cours' },
            { l: 'Note', v: '4,9', sub: '38 avis' },
          ].map((s, i) => (
            <div key={i} style={{ background: s.accent ? SCM.text : '#fff', color: s.accent ? '#fff' : SCM.text, border: s.accent ? 'none' : `1px solid ${SCM.border}`, borderRadius: 12, padding: 14 }}>
              <div style={{ fontSize: 10.5, color: s.accent ? 'rgba(255,255,255,0.6)' : SCM.textMute, marginBottom: 4, fontWeight: 600, letterSpacing: '0.04em', textTransform: 'uppercase' }}>{s.l}</div>
              <div style={{ fontFamily: SCM.serif, fontSize: 22, fontWeight: 500, letterSpacing: '-0.02em' }}>{s.v}</div>
              {s.sub ? <div style={{ fontSize: 10.5, color: s.accent ? 'rgba(255,255,255,0.6)' : SCM.textSubtle, marginTop: 2 }}>{s.sub}</div> : null}
            </div>
          ))}
        </div>

        {/* À faire aujourd'hui */}
        <div style={{ fontSize: 12, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase', color: SCM.textMute, marginBottom: 10 }}>À faire aujourd'hui</div>
        <div style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, marginBottom: 22, overflow: 'hidden' }}>
          <div style={{ padding: 14, borderLeft: `3px solid ${SCM.accent}`, borderBottom: `1px solid ${SCM.border}` }}>
            <div style={{ fontSize: 13.5, fontWeight: 600, marginBottom: 3 }}>Livrer le jalon « Wireframes » · Nova</div>
            <div style={{ fontSize: 11.5, color: SCM.textMute }}>Aujourd'hui · 1 800 € en séquestre</div>
          </div>
          <div style={{ padding: 14, borderBottom: `1px solid ${SCM.border}` }}>
            <div style={{ fontSize: 13.5, fontWeight: 600, marginBottom: 3 }}>Répondre à Sophie · Qonto</div>
            <div style={{ fontSize: 11.5, color: SCM.textMute }}>Relance d'hier 16 h</div>
          </div>
          <div style={{ padding: 14 }}>
            <div style={{ fontSize: 13.5, fontWeight: 600, marginBottom: 3 }}>Facturer Doctolib · 2 400 €</div>
            <div style={{ fontSize: 11.5, color: SCM.textMute }}>Mission terminée mardi</div>
          </div>
        </div>

        {/* Opportunités */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 10 }}>
          <div style={{ fontSize: 12, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase', color: SCM.textMute }}>Pour toi · 3 nouvelles</div>
          <a style={{ fontSize: 12, color: SCM.accent, fontWeight: 600 }}>Voir tout</a>
        </div>
        {[
          { co: 'Memo Bank', title: 'Refonte du parcours d\'inscription', tjm: '700 €/j', match: 94 },
          { co: 'Lydia', title: 'Audit UX onboarding mobile', tjm: '650 €/j', match: 88 },
        ].map((o, i) => (
          <div key={i} style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, padding: 14, marginBottom: 8 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
              <div style={{ fontSize: 11, color: SCM.textMute, fontWeight: 600 }}>{o.co}</div>
              <div style={{ fontSize: 11, color: SCM.green, fontWeight: 700, padding: '2px 7px', background: SCM.greenSoft, borderRadius: 999 }}>{o.match}% match</div>
            </div>
            <div style={{ fontSize: 14, fontWeight: 600, fontFamily: SCM.serif, marginBottom: 8 }}>{o.title}</div>
            <div style={{ fontSize: 12, color: SCM.textMute }}>{o.tjm} · 3-4 mois</div>
          </div>
        ))}
      </div>
      <MobileBottomNav active="home" role="freelancer" />
    </MobileFrame>
  );
}

// ─── CM2 — Opportunités ──────────────────────────────────────
function SoleilOpportunitiesMobile() {
  const [tab, setTab] = React.useState(0);
  return (
    <MobileFrame url="atelier.fr/opportunites">
      <MobileHeader title="Opportunités" action={<button style={{ width: 36, height: 36, borderRadius: '50%', background: SCM.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SCMI name="Filter" size={15} /></button>} />
      <div style={{ padding: '12px 14px', flexShrink: 0, background: '#fff', borderBottom: `1px solid ${SCM.border}` }}>
        <MobileSegmented items={['Pour toi · 12', 'Sauvées · 4', 'Récentes']} active={tab} />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '12px 14px 90px' }}>
        {[
          { co: 'Memo Bank', title: 'Refonte parcours inscription', tjm: '700 €/j', loc: 'Paris · Hybride', dur: '3-4 mois', match: 94, posted: 'il y a 2 h' },
          { co: 'Lydia', title: 'Audit UX onboarding mobile', tjm: '650 €/j', loc: 'Remote', dur: '6 semaines', match: 88, posted: 'hier' },
          { co: 'Qonto', title: 'Design system v2 · scaling', tjm: '750 €/j', loc: 'Paris', dur: '4-6 mois', match: 86, posted: 'il y a 3 j' },
          { co: 'Doctolib', title: 'Refonte espace pro', tjm: '600 €/j', loc: 'Remote', dur: '3 mois', match: 79, posted: 'il y a 5 j' },
        ].map((o, i) => (
          <div key={i} style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 14, padding: 16, marginBottom: 10 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
                <div style={{ width: 24, height: 24, borderRadius: 6, background: SCM.bg, fontSize: 11, fontWeight: 700, fontFamily: SCM.serif, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{o.co[0]}</div>
                <div style={{ fontSize: 12, fontWeight: 600 }}>{o.co}</div>
              </div>
              <div style={{ fontSize: 10.5, color: SCM.green, fontWeight: 700, padding: '2px 7px', background: SCM.greenSoft, borderRadius: 999 }}>{o.match}%</div>
            </div>
            <div style={{ fontSize: 15, fontWeight: 600, fontFamily: SCM.serif, marginBottom: 8, lineHeight: 1.3 }}>{o.title}</div>
            <div style={{ display: 'flex', gap: 12, fontSize: 11.5, color: SCM.textMute, flexWrap: 'wrap', marginBottom: 10 }}>
              <span><strong style={{ color: SCM.text }}>{o.tjm}</strong></span>
              <span>· {o.loc}</span>
              <span>· {o.dur}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingTop: 10, borderTop: `1px solid ${SCM.border}` }}>
              <div style={{ fontSize: 11, color: SCM.textSubtle, fontStyle: 'italic', fontFamily: SCM.serif }}>{o.posted}</div>
              <div style={{ display: 'flex', gap: 6 }}>
                <button style={{ width: 32, height: 32, borderRadius: '50%', background: SCM.bg, border: 'none' }}><SCMI name="Bookmark" size={13} /></button>
                <button style={{ background: SCM.text, color: '#fff', border: 'none', padding: '7px 14px', fontSize: 12, fontWeight: 600, borderRadius: 999 }}>Voir →</button>
              </div>
            </div>
          </div>
        ))}
      </div>
      <MobileBottomNav active="opp" role="freelancer" />
    </MobileFrame>
  );
}

// ─── CM3 — Détail opportunité ──────────────────────────────────
function SoleilOpportunityDetailMobile() {
  return (
    <MobileFrame url="atelier.fr/oppo/memo-bank">
      <MobileHeader title="Memo Bank" subtitle="Fintech B2B · Paris" back action={<button style={{ width: 36, height: 36, borderRadius: '50%', background: SCM.bg, border: 'none' }}><SCMI name="Bookmark" size={14} /></button>} />
      <div style={{ flex: 1, overflow: 'auto', padding: '16px 16px 100px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 14 }}>
          <h1 style={{ fontFamily: SCM.serif, fontSize: 22, margin: 0, fontWeight: 500, lineHeight: 1.2, letterSpacing: '-0.015em', flex: 1 }}>Refonte du parcours d'inscription</h1>
          <div style={{ fontSize: 11, color: SCM.green, fontWeight: 700, padding: '4px 10px', background: SCM.greenSoft, borderRadius: 999, marginLeft: 10 }}>94%</div>
        </div>

        <div style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, padding: 14, marginBottom: 14 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <div><div style={{ fontSize: 10.5, color: SCM.textMute, marginBottom: 3, fontWeight: 600, letterSpacing: '0.05em', textTransform: 'uppercase' }}>TJM</div><div style={{ fontFamily: SCM.serif, fontSize: 18, fontWeight: 500 }}>700 €/j</div></div>
            <div><div style={{ fontSize: 10.5, color: SCM.textMute, marginBottom: 3, fontWeight: 600, letterSpacing: '0.05em', textTransform: 'uppercase' }}>Durée</div><div style={{ fontFamily: SCM.serif, fontSize: 18, fontWeight: 500 }}>3-4 mois</div></div>
            <div><div style={{ fontSize: 10.5, color: SCM.textMute, marginBottom: 3, fontWeight: 600, letterSpacing: '0.05em', textTransform: 'uppercase' }}>Lieu</div><div style={{ fontFamily: SCM.serif, fontSize: 14, fontWeight: 500 }}>Paris · Hybride</div></div>
            <div><div style={{ fontSize: 10.5, color: SCM.textMute, marginBottom: 3, fontWeight: 600, letterSpacing: '0.05em', textTransform: 'uppercase' }}>Démarrage</div><div style={{ fontFamily: SCM.serif, fontSize: 14, fontWeight: 500 }}>Juin 2026</div></div>
          </div>
        </div>

        <div style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, padding: 16, marginBottom: 14 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 8 }}>La mission</div>
          <p style={{ fontSize: 13, lineHeight: 1.6, margin: 0, color: SCM.text }}>On cherche un·e UX qui sait poser un cadre méthodo solide. L'enjeu : faire passer notre taux de conversion d'inscription de 38% à 55% sur 4 mois, en repartant des entretiens utilisateurs.</p>
        </div>

        <div style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, padding: 16, marginBottom: 14 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 10 }}>Compétences attendues</div>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {['UX Research', 'Discovery', 'Figma', 'Design System', 'Mobile first'].map(t => (
              <span key={t} style={{ fontSize: 11.5, padding: '5px 10px', background: SCM.accentSoft, color: SCM.accentDeep, borderRadius: 999, fontWeight: 600 }}>{t}</span>
            ))}
          </div>
        </div>

        <div style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, padding: 14 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 10 }}>L'équipe</div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <SCMPortrait id={4} size={36} />
            <div>
              <div style={{ fontSize: 13, fontWeight: 600 }}>Sophie Aubry</div>
              <div style={{ fontSize: 11, color: SCM.textMute }}>Head of Product, ton interlocutrice</div>
            </div>
          </div>
        </div>
      </div>
      <div style={{ position: 'absolute', left: 0, right: 0, bottom: 0, padding: '12px 14px', background: '#fff', borderTop: `1px solid ${SCM.border}`, display: 'flex', gap: 8 }}>
        <button style={{ flex: 1, background: '#fff', border: `1px solid ${SCM.borderStrong}`, padding: '12px', fontSize: 13, fontWeight: 600, borderRadius: 999 }}>Poser une question</button>
        <button style={{ flex: 2, background: SCM.text, color: '#fff', border: 'none', padding: '12px', fontSize: 13, fontWeight: 600, borderRadius: 999 }}>Postuler →</button>
      </div>
    </MobileFrame>
  );
}

// ─── CM4 — Mes candidatures ───────────────────────────────────
function SoleilMyApplicationsMobile() {
  const [tab, setTab] = React.useState(0);
  const items = [
    { co: 'Qonto', title: 'Refonte cartes pro', status: 'Entretien programmé', kind: 'interview', date: 'Jeudi 11 h' },
    { co: 'Lydia', title: 'Audit onboarding mobile', status: 'En cours d\'examen', kind: 'sent', date: '2 jours' },
    { co: 'Memo Bank', title: 'Parcours inscription', status: 'Acceptée', kind: 'won', date: 'Démarre lundi' },
    { co: 'BlaBlaCar', title: 'Design system mobile', status: 'Non retenue', kind: 'lost', date: 'Il y a 1 sem' },
  ];
  const colors = { interview: SCM.accent, sent: SCM.amber, won: SCM.green, lost: SCM.textSubtle };
  const bgs = { interview: SCM.accentSoft, sent: SCM.amberSoft || '#fbf0dc', won: SCM.greenSoft, lost: SCM.bg };
  return (
    <MobileFrame url="atelier.fr/candidatures">
      <MobileHeader title="Mes candidatures" />
      <div style={{ padding: '12px 14px', flexShrink: 0, background: '#fff', borderBottom: `1px solid ${SCM.border}` }}>
        <MobileSegmented items={['Toutes · 7', 'En cours · 4', 'Terminées · 3']} active={tab} />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '12px 14px 90px' }}>
        {items.map((a, i) => (
          <div key={i} style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, padding: 14, marginBottom: 10 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
              <div style={{ fontSize: 12, color: SCM.textMute, fontWeight: 600 }}>{a.co}</div>
              <span style={{ fontSize: 10.5, padding: '3px 8px', background: bgs[a.kind], color: colors[a.kind], borderRadius: 999, fontWeight: 700 }}>{a.status}</span>
            </div>
            <div style={{ fontSize: 14.5, fontWeight: 600, fontFamily: SCM.serif, marginBottom: 6 }}>{a.title}</div>
            <div style={{ fontSize: 11.5, color: SCM.textMute, display: 'flex', alignItems: 'center', gap: 5 }}><SCMI name="Clock" size={11} /> {a.date}</div>
          </div>
        ))}
      </div>
      <MobileBottomNav active="opp" role="freelancer" />
    </MobileFrame>
  );
}

// ─── CM5 — Détail mission (livrer un jalon) ────────────────────
function SoleilFreelancerProjectMobile() {
  return (
    <MobileFrame url="atelier.fr/mission/nova">
      <MobileHeader title="Nova · Refonte SaaS" subtitle="Mission en cours · semaine 3 sur 12" back />
      <div style={{ flex: 1, overflow: 'auto', padding: '14px 14px 24px' }}>
        {/* Stepper jalons */}
        <div style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, padding: 14, marginBottom: 14 }}>
          <div style={{ fontSize: 12, fontWeight: 700, marginBottom: 12 }}>Jalons</div>
          {[
            { l: 'Discovery & entretiens', s: 'done', amount: '1 200 €' },
            { l: 'Wireframes', s: 'current', amount: '1 800 €', due: 'À livrer aujourd\'hui' },
            { l: 'UI haute fidélité', s: 'todo', amount: '2 400 €' },
            { l: 'Handoff dev', s: 'todo', amount: '1 200 €' },
          ].map((j, i, arr) => (
            <div key={i} style={{ display: 'flex', gap: 12, position: 'relative', paddingBottom: i === arr.length - 1 ? 0 : 14 }}>
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', flexShrink: 0 }}>
                <div style={{ width: 24, height: 24, borderRadius: '50%', background: j.s === 'done' ? SCM.green : j.s === 'current' ? SCM.accent : SCM.bg, color: j.s === 'todo' ? SCM.textMute : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, fontWeight: 700 }}>
                  {j.s === 'done' ? '✓' : i + 1}
                </div>
                {i < arr.length - 1 ? <div style={{ width: 2, flex: 1, background: j.s === 'done' ? SCM.green : SCM.border, marginTop: 2 }} /> : null}
              </div>
              <div style={{ flex: 1, paddingBottom: i < arr.length - 1 ? 4 : 0 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 8 }}>
                  <div style={{ fontSize: 13.5, fontWeight: j.s === 'current' ? 700 : 600, color: j.s === 'todo' ? SCM.textMute : SCM.text }}>{j.l}</div>
                  <div style={{ fontSize: 12, fontWeight: 700, color: j.s === 'todo' ? SCM.textSubtle : SCM.text, fontFamily: SCM.serif }}>{j.amount}</div>
                </div>
                {j.due ? <div style={{ fontSize: 11, color: SCM.accent, marginTop: 2, fontWeight: 600 }}>{j.due}</div> : null}
              </div>
            </div>
          ))}
        </div>

        {/* Action livrer */}
        <div style={{ background: SCM.accentSoft, border: `1.5px solid ${SCM.accent}`, borderRadius: 12, padding: 16, marginBottom: 14 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 4, color: SCM.accentDeep }}>Livrer « Wireframes »</div>
          <div style={{ fontSize: 12, color: SCM.text, marginBottom: 12, lineHeight: 1.55 }}>1 800 € sont en séquestre. Une fois validés par Nova, ils sont versés sous 48 h.</div>
          <button style={{ width: '100%', background: SCM.text, color: '#fff', border: 'none', padding: '12px', fontSize: 13, fontWeight: 600, borderRadius: 999, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}><SCMI name="Send" size={14} /> Soumettre la livraison</button>
        </div>

        {/* Équipe / contact */}
        <div style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, padding: 14, marginBottom: 14 }}>
          <div style={{ fontSize: 12, fontWeight: 700, marginBottom: 10 }}>Ton interlocuteur</div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <SCMPortrait id={2} size={36} />
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 13, fontWeight: 600 }}>Marc Lefèvre</div>
              <div style={{ fontSize: 11, color: SCM.textMute }}>Head of Product, Nova</div>
            </div>
            <button style={{ background: SCM.bg, border: 'none', padding: '8px 14px', fontSize: 12, fontWeight: 600, borderRadius: 999 }}>Message</button>
          </div>
        </div>

        {/* Récap financier */}
        <div style={{ background: '#fff', border: `1px solid ${SCM.border}`, borderRadius: 12, padding: 14 }}>
          <div style={{ fontSize: 12, fontWeight: 700, marginBottom: 10 }}>Récap financier</div>
          {[['Total mission', '6 600 €'], ['Déjà versé', '1 200 €'], ['En séquestre', '1 800 €'], ['À venir', '3 600 €']].map(([l, v], i) => (
            <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', fontSize: 12.5, borderBottom: i < 3 ? `1px solid ${SCM.border}` : 'none' }}>
              <span style={{ color: SCM.textMute }}>{l}</span>
              <span style={{ fontWeight: 600, fontFamily: SCM.serif }}>{v}</span>
            </div>
          ))}
        </div>
      </div>
    </MobileFrame>
  );
}

window.SoleilFreelancerDashboardMobile = SoleilFreelancerDashboardMobile;
window.SoleilOpportunitiesMobile = SoleilOpportunitiesMobile;
window.SoleilOpportunityDetailMobile = SoleilOpportunityDetailMobile;
window.SoleilMyApplicationsMobile = SoleilMyApplicationsMobile;
window.SoleilFreelancerProjectMobile = SoleilFreelancerProjectMobile;
