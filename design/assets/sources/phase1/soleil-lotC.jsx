// Lot C — Côté prestataire (freelance)
// 5 écrans : Dashboard prestataire · Opportunités feed · Détail opportunité · Mes candidatures · Détail mission freelance
// Réutilise SSidebar (role="freelancer"), STopbar via window. SC = palette local.

const SC = window.S || {
  bg: '#fffbf5', surface: '#ffffff', border: '#f0e6d8', borderStrong: '#e0d3bc',
  text: '#2a1f15', textMute: '#7a6850', textSubtle: '#a89679',
  accent: '#e85d4a', accentSoft: '#fde9e3', accentDeep: '#c43a26',
  pink: '#f08aa8', pinkSoft: '#fde6ed',
  green: '#5a9670', greenSoft: '#e8f2eb',
  amber: '#d4924a', amberSoft: '#fbf0dc',
  serif: 'Fraunces, Georgia, serif',
  sans: '"Inter Tight", system-ui, sans-serif',
  mono: '"Geist Mono", monospace',
};
const SCI = window.SI || (() => null);
const SCSidebar = (props) => window.SSidebar ? window.SSidebar(props) : null;
const SCTopbar = () => window.STopbar ? window.STopbar() : null;
const SCPortrait = (props) => window.Portrait ? window.Portrait(props) : null;

// ─── pill réutilisée
function CStatusPill({ label, kind = 'open' }) {
  const styles = {
    open:    { bg: SC.greenSoft, color: SC.green, dot: SC.green },
    pending: { bg: SC.amberSoft, color: SC.amber, dot: SC.amber },
    won:     { bg: SC.greenSoft, color: SC.green, dot: SC.green },
    lost:    { bg: SC.bg, color: SC.textMute, dot: SC.textSubtle },
    sent:    { bg: SC.pinkSoft, color: SC.accentDeep, dot: SC.pink },
    interview: { bg: SC.accentSoft, color: SC.accentDeep, dot: SC.accent },
  };
  const s = styles[kind] || styles.open;
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 11, padding: '4px 10px', background: s.bg, color: s.color, borderRadius: 999, fontWeight: 600 }}>
      <span style={{ width: 6, height: 6, borderRadius: '50%', background: s.dot }} /> {label}
    </span>
  );
}

// ═══ C1 — Dashboard prestataire ═══════════════════════════════════
function SoleilFreelancerDashboard() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SC.bg, fontFamily: SC.sans, color: SC.text }}>
      <SCSidebar active="home" role="freelancer" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SCTopbar />
        <div style={{ flex: 1, overflow: 'auto', padding: '32px 40px' }}>
          {/* Header */}
          <div style={{ marginBottom: 24 }}>
            <div style={{ fontSize: 12, color: SC.textMute, marginBottom: 4, fontFamily: SC.mono, letterSpacing: '0.04em' }}>Mardi matin · 14 mai</div>
            <h1 style={{ fontFamily: SC.serif, fontSize: 42, lineHeight: 1.05, margin: 0, fontWeight: 400, letterSpacing: '-0.025em' }}>Bonjour Élise, <span style={{ fontStyle: 'italic', color: SC.accent }}>belle semaine</span> qui s'annonce.</h1>
            <p style={{ fontSize: 15, color: SC.textMute, margin: '8px 0 0', maxWidth: 580 }}>3 opportunités correspondent à ton profil, un jalon à livrer chez Nova, et Sophie t'a relancée.</p>
          </div>

          {/* Quick stats — 4 cards */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 14, marginBottom: 22 }}>
            {[
              { l: 'Missions actives', v: '3', sub: 'sur 5 ce mois', icon: 'Folder' },
              { l: 'Candidatures envoyées', v: '7', sub: '2 en cours', icon: 'Send' },
              { l: 'Revenus du mois', v: '4 920 €', sub: 'sur 6 800 € prévus', icon: 'Euro', accent: true },
              { l: 'Note moyenne', v: '4,9', sub: 'sur 38 avis', icon: 'Star' },
            ].map((s, i) => (
              <div key={i} style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 20 }}>
                <div style={{ width: 32, height: 32, borderRadius: 10, background: s.accent ? SC.accentSoft : SC.bg, color: s.accent ? SC.accent : SC.textMute, display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: 10 }}>
                  <SCI name={s.icon} size={16} />
                </div>
                <div style={{ fontSize: 11, color: SC.textMute, letterSpacing: '0.04em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 4 }}>{s.l}</div>
                <div style={{ fontFamily: SC.serif, fontSize: 30, fontWeight: 500, letterSpacing: '-0.02em', lineHeight: 1, marginBottom: 4 }}>{s.v}</div>
                <div style={{ fontSize: 11.5, color: SC.textMute }}>{s.sub}</div>
              </div>
            ))}
          </div>

          {/* Two cols */}
          <div style={{ display: 'grid', gridTemplateColumns: '1.5fr 1fr', gap: 18 }}>
            {/* Missions actives */}
            <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 26 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: 18 }}>
                <div>
                  <h2 style={{ fontFamily: SC.serif, fontSize: 22, margin: 0, fontWeight: 500, letterSpacing: '-0.01em' }}>Tes missions du moment</h2>
                  <p style={{ fontSize: 12, color: SC.textMute, margin: '3px 0 0' }}>Prochain jalon à livrer chez Nova Studio · jeudi</p>
                </div>
                <a style={{ fontSize: 12, color: SC.accent, fontWeight: 600, cursor: 'pointer' }}>Tout voir →</a>
              </div>
              {[
                { name: "Refonte de l'app produit", client: 'Nova Studio', amount: '12 400 €', progress: 60, dl: 'jalon jeudi · Design system', pid: 2, urgent: true },
                { name: 'Brand identity Q2', client: 'Cobalt Studio', amount: '8 200 €', progress: 35, dl: 'livraison dans 3 sem.', pid: 0 },
                { name: 'Audit UX onboarding', client: 'Maison Vega', amount: '3 600 €', progress: 90, dl: 'finalisation cette semaine', pid: 5 },
              ].map((m, i) => (
                <div key={i} style={{ padding: '16px 0', borderTop: i > 0 ? `1px solid ${SC.border}` : 'none', display: 'grid', gridTemplateColumns: '44px 1fr auto', gap: 14, alignItems: 'center' }}>
                  <SCPortrait id={m.pid} size={44} rounded={12} />
                  <div style={{ minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 3 }}>
                      <span style={{ fontFamily: SC.serif, fontSize: 16, fontWeight: 600, letterSpacing: '-0.01em' }}>{m.name}</span>
                      {m.urgent && <span style={{ fontSize: 10, padding: '2px 7px', background: SC.accentSoft, color: SC.accentDeep, borderRadius: 999, fontWeight: 700 }}>À livrer</span>}
                    </div>
                    <div style={{ fontSize: 12, color: SC.textMute, marginBottom: 8 }}>avec {m.client} · {m.dl}</div>
                    <div style={{ width: 240, height: 5, background: SC.border, borderRadius: 3, overflow: 'hidden' }}>
                      <div style={{ width: m.progress + '%', height: '100%', background: SC.accent }} />
                    </div>
                  </div>
                  <div style={{ textAlign: 'right' }}>
                    <div style={{ fontFamily: SC.serif, fontSize: 18, fontWeight: 600 }}>{m.amount}</div>
                    <div style={{ fontSize: 10.5, color: SC.textMute, marginTop: 2 }}>{m.progress}% avancé</div>
                  </div>
                </div>
              ))}
            </div>

            {/* Right col — opportunities recommended + activity */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 18 }}>
              {/* Reco opportunités */}
              <div style={{ background: 'linear-gradient(135deg, #fde9e3, #fde6ed)', borderRadius: 16, padding: 22, border: `1px solid ${SC.border}` }}>
                <div style={{ fontSize: 11, fontWeight: 700, color: SC.accentDeep, marginBottom: 8, letterSpacing: '0.1em', textTransform: 'uppercase', display: 'flex', alignItems: 'center', gap: 7 }}>
                  <SCI name="Sparkle" size={13} /> Pour toi cette semaine
                </div>
                <h3 style={{ fontFamily: SC.serif, fontSize: 19, margin: '0 0 6px', fontWeight: 500, lineHeight: 1.25 }}>3 missions correspondent à ton profil</h3>
                <p style={{ fontSize: 12.5, color: SC.textMute, margin: '0 0 14px', lineHeight: 1.5 }}>UX Designer · SaaS B2B · Mission longue à Paris</p>
                <button style={{ background: SC.text, color: '#fff', border: 'none', padding: '9px 16px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Voir les opportunités →</button>
              </div>

              {/* Activité récente */}
              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 22 }}>
                <h3 style={{ fontFamily: SC.serif, fontSize: 18, margin: '0 0 14px', fontWeight: 600 }}>Activité récente</h3>
                {[
                  { t: 'Sophie de Nova t\'a envoyé un message', time: 'il y a 2 h', icon: 'Chat', color: SC.accent },
                  { t: 'Maison Vega a validé ton jalon', time: 'hier', icon: 'CheckCircle', color: SC.green },
                  { t: 'Ta candidature a été présélectionnée chez Cobalt', time: 'il y a 2 j', icon: 'Star', color: SC.amber },
                  { t: 'Paiement reçu : 2 400 €', time: 'il y a 3 j', icon: 'Euro', color: SC.green },
                ].map((a, i) => (
                  <div key={i} style={{ display: 'flex', gap: 11, padding: '10px 0', borderTop: i > 0 ? `1px solid ${SC.border}` : 'none' }}>
                    <div style={{ width: 28, height: 28, borderRadius: 8, background: SC.bg, color: a.color, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                      <SCI name={a.icon} size={14} />
                    </div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: 12.5, lineHeight: 1.4 }}>{a.t}</div>
                      <div style={{ fontSize: 10.5, color: SC.textMute, marginTop: 2 }}>{a.time}</div>
                    </div>
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

// ═══ C2 — Opportunités (feed) ═════════════════════════════════════
function SoleilOpportunities() {
  const opps = [
    { id: 1, title: 'Refonte de l\'app produit Nova', co: 'Nova Studio', desc: 'Mission longue, UX/UI senior, design system à co-construire avec une équipe produit de 6 personnes.', tags: ['UX Design', 'Mobile', 'Design System'], budget: '8 000 — 12 000 €', mode: 'Hybride · Paris', posted: 'il y a 3 j', match: 'Ton profil correspond', verified: true, candidates: 12, pid: 2, kind: 'Mission longue' },
    { id: 2, title: 'Identité visuelle pour app Tel-Avivienne', co: 'Mishbatzar', desc: 'Lancement Q3, refonte complète : logo, charte, declinaisons digitales, tone of voice.', tags: ['Branding', 'Logo', 'Charte'], budget: '6 000 — 9 000 €', mode: '100 % remote', posted: 'il y a 5 j', verified: true, candidates: 28, pid: 4, kind: 'Projet ponctuel' },
    { id: 3, title: 'UX Audit — checkout fintech', co: 'Helios Pay', desc: 'Audit complet de notre flow de paiement, recos actionables, pas de production.', tags: ['UX Research', 'Fintech', 'Audit'], budget: '4 000 — 5 500 €', mode: 'Remote', posted: 'il y a 1 sem.', verified: false, candidates: 8, pid: 5, kind: 'Projet ponctuel' },
    { id: 4, title: 'Product Designer — feature pricing', co: 'Klaxoon', desc: 'Nouvelle feature de pricing dynamique. Discovery + design, 2 mois plein temps.', tags: ['Product Design', 'SaaS', 'Pricing'], budget: '15 000 — 18 000 €', mode: 'Hybride · Rennes', posted: 'il y a 1 sem.', verified: true, candidates: 6, pid: 0, kind: 'Mission longue' },
  ];

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SC.bg, fontFamily: SC.sans, color: SC.text }}>
      <SCSidebar active="opp" role="freelancer" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SCTopbar />
        <div style={{ flex: 1, overflow: 'auto' }}>
          {/* Editorial header */}
          <div style={{ padding: '32px 40px 22px' }}>
            <div style={{ fontSize: 11, color: SC.accent, marginBottom: 8, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase', display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ width: 24, height: 1, background: SC.accent }} /> 47 nouvelles missions cette semaine
            </div>
            <h1 style={{ fontFamily: SC.serif, fontSize: 42, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1 }}>Trouve ta <span style={{ fontStyle: 'italic', color: SC.accent }}>prochaine mission.</span></h1>
            <p style={{ fontSize: 15, color: SC.textMute, margin: '8px 0 0', maxWidth: 580 }}>On t'a sélectionné les missions qui correspondent à ton profil et tes envies. Filtre, suis tes favoris, postule en deux clics.</p>
          </div>

          {/* Filters */}
          <div style={{ padding: '0 40px 22px', display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
            <button style={{ background: '#fff', border: `1px solid ${SC.borderStrong}`, padding: '8px 14px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
              <SCI name="Sliders" size={14} /> Tous les filtres
            </button>
            <span style={{ width: 1, height: 20, background: SC.border, margin: '0 4px' }} />
            {[
              { l: 'Pour toi', active: true },
              { l: 'UX / Product' },
              { l: 'Branding' },
              { l: 'Mission longue' },
              { l: 'Remote' },
              { l: '> 5 000 €' },
            ].map((f, i) => (
              <button key={i} style={{ background: f.active ? SC.text : '#fff', color: f.active ? '#fff' : SC.text, border: f.active ? 'none' : `1px solid ${SC.border}`, padding: '7px 13px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>{f.l}</button>
            ))}
            <div style={{ flex: 1 }} />
            <span style={{ fontSize: 12, color: SC.textMute }}>Tri</span>
            <button style={{ background: 'none', border: 'none', fontSize: 13, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 4, cursor: 'pointer' }}>Plus récentes <SCI name="ChevronDown" size={13} /></button>
          </div>

          {/* Feed */}
          <div style={{ padding: '0 40px 40px', display: 'flex', flexDirection: 'column', gap: 14 }}>
            {opps.map((o, i) => (
              <div key={o.id} style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 18, padding: 24, cursor: 'pointer', display: 'grid', gridTemplateColumns: '1fr 240px', gap: 28 }}>
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
                    <SCPortrait id={o.pid} size={36} rounded={10} />
                    <div>
                      <div style={{ fontSize: 13, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6 }}>
                        {o.co}
                        {o.verified && <SCI name="Verified" size={13} />}
                      </div>
                      <div style={{ fontSize: 11, color: SC.textMute, fontFamily: SC.mono }}>{o.kind} · {o.posted}</div>
                    </div>
                    {o.match && <span style={{ marginLeft: 'auto', fontSize: 10.5, padding: '3px 9px', background: SC.greenSoft, color: SC.green, borderRadius: 999, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                      <SCI name="Sparkle" size={10} /> {o.match}
                    </span>}
                  </div>
                  <h3 style={{ fontFamily: SC.serif, fontSize: 22, margin: '0 0 8px', fontWeight: 500, letterSpacing: '-0.015em' }}>{o.title}</h3>
                  <p style={{ fontSize: 14, color: SC.textMute, margin: '0 0 14px', lineHeight: 1.55, maxWidth: 640, textWrap: 'pretty' }}>{o.desc}</p>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 14 }}>
                    {o.tags.map((t, ti) => (
                      <span key={ti} style={{ fontSize: 11.5, padding: '4px 10px', background: SC.bg, borderRadius: 999, fontWeight: 500, color: SC.textMute }}>{t}</span>
                    ))}
                  </div>
                  <div style={{ display: 'flex', gap: 22, fontSize: 12.5, color: SC.textMute, alignItems: 'center' }}>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SCI name="Euro" size={13} /> <strong style={{ color: SC.text, fontFamily: SC.serif, fontSize: 14 }}>{o.budget}</strong></span>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SCI name="MapPin" size={13} /> {o.mode}</span>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SCI name="Users" size={13} /> {o.candidates} candidatures</span>
                  </div>
                </div>
                <div style={{ borderLeft: `1px solid ${SC.border}`, paddingLeft: 24, display: 'flex', flexDirection: 'column', justifyContent: 'space-between', gap: 12 }}>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                    <button style={{ width: '100%', background: SC.accent, color: '#fff', border: 'none', padding: '11px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', boxShadow: '0 2px 8px rgba(232,93,74,0.25)' }}>Postuler</button>
                    <button style={{ width: '100%', background: '#fff', color: SC.text, border: `1px solid ${SC.borderStrong}`, padding: '10px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}>
                      <SCI name="Bookmark" size={13} /> Sauvegarder
                    </button>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 11, color: SC.textMute }}>
                    <SCI name="Clock" size={12} /> Plus que 8 jours pour candidater
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ C3 — Détail opportunité + flow candidature ═════════════════
function SoleilOpportunityDetail() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SC.bg, fontFamily: SC.sans, color: SC.text }}>
      <SCSidebar active="opp" role="freelancer" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SCTopbar />
        <div style={{ flex: 1, overflow: 'auto' }}>
          {/* Header card */}
          <div style={{ padding: '24px 40px 0', background: '#fff', borderBottom: `1px solid ${SC.border}` }}>
            <div style={{ fontSize: 12, color: SC.textMute, marginBottom: 14, display: 'flex', alignItems: 'center', gap: 6 }}>
              <SCI name="ArrowLeft" size={13} /> <span style={{ cursor: 'pointer' }}>Toutes les opportunités</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 24, marginBottom: 22 }}>
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 14 }}>
                  <SCPortrait id={2} size={56} rounded={14} />
                  <div>
                    <div style={{ fontSize: 14, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6 }}>
                      Nova Studio <SCI name="Verified" size={14} />
                    </div>
                    <div style={{ fontSize: 12, color: SC.textMute }}>SaaS B2B · 12 personnes · Paris · sur Atelier depuis 2024</div>
                  </div>
                  <span style={{ marginLeft: 'auto', fontSize: 10.5, padding: '4px 10px', background: SC.greenSoft, color: SC.green, borderRadius: 999, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                    <SCI name="Sparkle" size={11} /> Ton profil correspond
                  </span>
                </div>
                <div style={{ fontSize: 11, color: SC.textMute, marginBottom: 8, fontFamily: SC.mono }}>Mission longue · publiée il y a 3 jours</div>
                <h1 style={{ fontFamily: SC.serif, fontSize: 38, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1 }}>Refonte de l'app produit Nova</h1>
                <div style={{ display: 'flex', gap: 22, fontSize: 13, color: SC.textMute, marginTop: 14, alignItems: 'center', flexWrap: 'wrap' }}>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SCI name="Euro" size={13} /> <strong style={{ color: SC.text, fontFamily: SC.serif, fontSize: 16 }}>8 000 — 12 000 €</strong></span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SCI name="Clock" size={13} /> Démarrage le 27 mai · 3 mois</span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SCI name="MapPin" size={13} /> Hybride · Paris</span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SCI name="Users" size={13} /> 12 candidatures</span>
                </div>
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                <button style={{ background: '#fff', border: `1px solid ${SC.borderStrong}`, padding: '10px 14px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
                  <SCI name="Bookmark" size={13} /> Sauvegarder
                </button>
                <button style={{ background: SC.accent, color: '#fff', border: 'none', padding: '10px 22px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', boxShadow: '0 2px 8px rgba(232,93,74,0.25)' }}>Postuler</button>
              </div>
            </div>
          </div>

          {/* Body */}
          <div style={{ padding: '28px 40px', display: 'grid', gridTemplateColumns: '1.7fr 1fr', gap: 24 }}>
            {/* Left col */}
            <div>
              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 32, marginBottom: 18 }}>
                <h2 style={{ fontFamily: SC.serif, fontSize: 22, margin: '0 0 14px', fontWeight: 500, letterSpacing: '-0.01em' }}>Le contexte</h2>
                <p style={{ fontSize: 15, lineHeight: 1.7, margin: '0 0 14px', textWrap: 'pretty' }}>
                  <strong style={{ color: SC.accent }}>Nova Studio</strong> est une SaaS B2B qui aide les studios créatifs à gérer leurs projets. Ils ont levé en Série A en mars et entament une refonte produit ambitieuse pour les 12 prochains mois.
                </p>
                <p style={{ fontSize: 15, lineHeight: 1.7, margin: 0, color: SC.textMute, textWrap: 'pretty' }}>
                  L'app actuelle a 3 ans et c'est devenu touffu. L'objectif : repartir des usages réels, alléger l'interface, et poser un design system solide.
                </p>
              </div>

              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 32, marginBottom: 18 }}>
                <h2 style={{ fontFamily: SC.serif, fontSize: 22, margin: '0 0 14px', fontWeight: 500 }}>Tes missions</h2>
                <ul style={{ margin: 0, paddingLeft: 0, listStyle: 'none', display: 'flex', flexDirection: 'column', gap: 12 }}>
                  {[
                    'Conduire une phase de discovery (4 sem) — interviews users, audit de l\'existant.',
                    'Co-construire un nouveau design system avec l\'équipe produit.',
                    'Concevoir les flows clés : onboarding, projets, facturation.',
                    'Travailler en duo avec le lead dev (Théo) sur les specs.',
                    'Documenter, présenter en revue produit toutes les 2 semaines.',
                  ].map((it, i) => (
                    <li key={i} style={{ display: 'flex', gap: 12, fontSize: 14.5, lineHeight: 1.6 }}>
                      <span style={{ flexShrink: 0, width: 22, height: 22, borderRadius: '50%', background: SC.accentSoft, color: SC.accent, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12, fontFamily: SC.serif, fontWeight: 600 }}>{i + 1}</span>
                      <span style={{ flex: 1, paddingTop: 1 }}>{it}</span>
                    </li>
                  ))}
                </ul>
              </div>

              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 32 }}>
                <h2 style={{ fontFamily: SC.serif, fontSize: 22, margin: '0 0 14px', fontWeight: 500 }}>Profil recherché</h2>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                  {[
                    ['7 ans+', 'd\'expérience en produit'],
                    ['Design system', 'tu sais en construire un solide'],
                    ['SaaS B2B', 'tu connais les enjeux'],
                    ['Français', 'écrit et oral courant'],
                  ].map(([t, d], i) => (
                    <div key={i} style={{ background: SC.bg, borderRadius: 12, padding: '14px 16px' }}>
                      <div style={{ fontFamily: SC.serif, fontSize: 18, fontWeight: 600, marginBottom: 3 }}>{t}</div>
                      <div style={{ fontSize: 12.5, color: SC.textMute }}>{d}</div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Right col */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {/* Apply CTA card — sticky-ish */}
              <div style={{ background: 'linear-gradient(135deg, #fde9e3, #fde6ed)', borderRadius: 16, padding: 24, border: `1px solid ${SC.border}` }}>
                <div style={{ fontSize: 11, fontWeight: 700, color: SC.accentDeep, marginBottom: 8, letterSpacing: '0.1em', textTransform: 'uppercase' }}>Prête à candidater ?</div>
                <h3 style={{ fontFamily: SC.serif, fontSize: 19, margin: '0 0 8px', fontWeight: 500, lineHeight: 1.3 }}>Réponds en quelques mots, ils te répondent en moyenne sous 24 h.</h3>
                <p style={{ fontSize: 12.5, color: SC.textMute, margin: '0 0 14px', lineHeight: 1.5 }}>Une lettre de motivation, ton TJM, et tu peux y aller.</p>
                <button style={{ width: '100%', background: SC.text, color: '#fff', border: 'none', padding: '12px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 7 }}>
                  <SCI name="Send" size={14} /> Postuler à cette mission
                </button>
              </div>

              {/* Form quick — message + TJM */}
              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 22 }}>
                <h3 style={{ fontFamily: SC.serif, fontSize: 17, margin: '0 0 12px', fontWeight: 600 }}>Ton message</h3>
                <div style={{ border: `1px solid ${SC.borderStrong}`, borderRadius: 12, padding: 12, fontSize: 13, color: SC.text, lineHeight: 1.5, fontFamily: SC.sans, minHeight: 120 }}>
                  Bonjour Sophie,<br /><br />
                  Je suis très intéressée par votre projet de refonte. Mon expérience récente avec Lemon Aviation et Cobalt Studio s'aligne bien avec vos enjeux SaaS B2B...
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginTop: 10, fontSize: 11, color: SC.textMute }}>
                  <span>~ 240 / 1500 caractères</span>
                  <span style={{ fontStyle: 'italic', fontFamily: SC.serif }}>Auto-sauvegardé</span>
                </div>
                <div style={{ marginTop: 16 }}>
                  <label style={{ fontSize: 13, fontWeight: 600, marginBottom: 6, display: 'block' }}>Ton TJM pour cette mission</label>
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                    <div style={{ flex: 1, position: 'relative' }}>
                      <input defaultValue="650" style={{ width: '100%', border: `1px solid ${SC.borderStrong}`, borderRadius: 10, padding: '10px 32px 10px 14px', fontSize: 14, fontFamily: SC.sans, outline: 'none' }} />
                      <span style={{ position: 'absolute', right: 12, top: '50%', transform: 'translateY(-50%)', color: SC.textMute, fontSize: 13, fontFamily: SC.serif }}>€/j</span>
                    </div>
                    <span style={{ fontSize: 11.5, color: SC.textMute }}>Habituel : 600 — 700 €</span>
                  </div>
                </div>
              </div>

              {/* L'équipe */}
              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 22 }}>
                <h3 style={{ fontFamily: SC.serif, fontSize: 17, margin: '0 0 14px', fontWeight: 600 }}>L'équipe</h3>
                {[
                  { name: 'Sophie A.', role: 'CPO · ton interlocutrice', pid: 4 },
                  { name: 'Théo R.', role: 'Lead Dev', pid: 5 },
                  { name: 'Marie L.', role: 'Product Manager', pid: 2 },
                ].map((m, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 0' }}>
                    <SCPortrait id={m.pid} size={36} />
                    <div>
                      <div style={{ fontSize: 13, fontWeight: 600 }}>{m.name}</div>
                      <div style={{ fontSize: 11.5, color: SC.textMute }}>{m.role}</div>
                    </div>
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

// ═══ C4 — Mes candidatures ═══════════════════════════════════════
function SoleilMyApplications() {
  const apps = [
    { co: 'Nova Studio', mission: 'Refonte de l\'app produit', tjm: '650 €/j', sent: 'il y a 2 j', status: 'interview', kind: 'Mission longue', pid: 2, lastEvent: 'Sophie a accepté ton appel demain à 14h' },
    { co: 'Cobalt Studio', mission: 'Brand identity Q2 — déclinaisons', tjm: '720 €/j', sent: 'il y a 1 sem.', status: 'sent', kind: 'Projet ponctuel', pid: 0, lastEvent: 'Vue il y a 3 jours · pas encore de réponse' },
    { co: 'Helios Pay', mission: 'UX Audit — checkout', tjm: '600 €/j', sent: 'il y a 1 sem.', status: 'sent', kind: 'Projet ponctuel', pid: 5, lastEvent: 'Vue · en attente d\'une décision' },
    { co: 'Lemon Aviation', mission: 'Refonte site corporate', tjm: '680 €/j', sent: 'il y a 2 sem.', status: 'won', kind: 'Mission longue', pid: 1, lastEvent: 'Mission acceptée · démarrée le 1er mai' },
    { co: 'Maison Vega', mission: 'Étude UX onboarding', tjm: '580 €/j', sent: 'il y a 2 sem.', status: 'won', kind: 'Projet ponctuel', pid: 3, lastEvent: 'Mission terminée · payée' },
    { co: 'Klaxoon', mission: 'Product Designer pricing', tjm: '700 €/j', sent: 'il y a 3 sem.', status: 'lost', kind: 'Mission longue', pid: 4, lastEvent: 'Ils ont retenu un autre profil' },
  ];

  const labelOf = { sent: 'Envoyée', interview: 'Entretien prévu', won: 'Acceptée', lost: 'Refusée' };

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SC.bg, fontFamily: SC.sans, color: SC.text }}>
      <SCSidebar active="apps" role="freelancer" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SCTopbar />
        <div style={{ flex: 1, overflow: 'auto', padding: '32px 40px' }}>
          <div style={{ marginBottom: 22 }}>
            <h1 style={{ fontFamily: SC.serif, fontSize: 38, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1 }}>Tes candidatures, <span style={{ fontStyle: 'italic', color: SC.accent }}>au calme.</span></h1>
            <p style={{ fontSize: 14, color: SC.textMute, margin: '8px 0 0', maxWidth: 560 }}>Suis l'avancement, relance au bon moment, archive ce qui n'a pas marché.</p>
          </div>

          {/* Stats compact */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12, marginBottom: 22 }}>
            {[
              { l: 'Envoyées', v: '6' },
              { l: 'En cours', v: '3', accent: true },
              { l: 'Acceptées', v: '2', color: SC.green },
              { l: 'Taux de conversion', v: '33 %' },
            ].map((s, i) => (
              <div key={i} style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 14, padding: 18 }}>
                <div style={{ fontSize: 11, color: SC.textMute, letterSpacing: '0.04em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 4 }}>{s.l}</div>
                <div style={{ fontFamily: SC.serif, fontSize: 28, fontWeight: 500, letterSpacing: '-0.02em', color: s.color || (s.accent ? SC.accent : SC.text) }}>{s.v}</div>
              </div>
            ))}
          </div>

          {/* Tabs */}
          <div style={{ display: 'flex', gap: 8, marginBottom: 18, flexWrap: 'wrap' }}>
            {[
              { l: 'Toutes', n: 6, active: true },
              { l: 'En cours', n: 3 },
              { l: 'Entretien', n: 1 },
              { l: 'Acceptées', n: 2 },
              { l: 'Refusées', n: 1 },
            ].map((t, i) => (
              <button key={i} style={{ background: t.active ? SC.text : '#fff', color: t.active ? '#fff' : SC.text, border: t.active ? 'none' : `1px solid ${SC.border}`, padding: '8px 14px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
                {t.l} <span style={{ fontSize: 11, opacity: 0.7 }}>{t.n}</span>
              </button>
            ))}
          </div>

          {/* List */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {apps.map((a, i) => (
              <div key={i} style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 20, display: 'grid', gridTemplateColumns: '50px 1fr 180px 180px', gap: 18, alignItems: 'center', cursor: 'pointer', opacity: a.status === 'lost' ? 0.65 : 1 }}>
                <SCPortrait id={a.pid} size={50} rounded={12} />
                <div style={{ minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 3 }}>
                    <span style={{ fontSize: 15, fontWeight: 600, fontFamily: SC.serif, letterSpacing: '-0.01em' }}>{a.co}</span>
                    <span style={{ fontSize: 11, color: SC.textMute, fontFamily: SC.mono }}>{a.kind}</span>
                  </div>
                  <div style={{ fontSize: 13.5, marginBottom: 4 }}>{a.mission}</div>
                  <div style={{ fontSize: 11.5, color: SC.textMute, display: 'flex', alignItems: 'center', gap: 6 }}>
                    <SCI name={a.status === 'interview' ? 'Phone' : a.status === 'won' ? 'CheckCircle' : a.status === 'lost' ? 'Inbox' : 'Clock'} size={11} />
                    {a.lastEvent}
                  </div>
                </div>
                <div>
                  <CStatusPill label={labelOf[a.status]} kind={a.status} />
                  <div style={{ fontSize: 11, color: SC.textMute, marginTop: 6 }}>Envoyée {a.sent}</div>
                  <div style={{ fontSize: 12, color: SC.text, fontWeight: 600, marginTop: 4 }}>TJM proposé : {a.tjm}</div>
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6, alignItems: 'flex-end' }}>
                  {a.status === 'sent' && <button style={{ background: SC.text, color: '#fff', border: 'none', padding: '8px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', whiteSpace: 'nowrap' }}>Relancer</button>}
                  {a.status === 'interview' && <button style={{ background: SC.accent, color: '#fff', border: 'none', padding: '8px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', whiteSpace: 'nowrap', display: 'flex', alignItems: 'center', gap: 5 }}><SCI name="Phone" size={11} /> Préparer l'appel</button>}
                  {a.status === 'won' && <button style={{ background: SC.green, color: '#fff', border: 'none', padding: '8px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', whiteSpace: 'nowrap' }}>Voir la mission</button>}
                  {a.status === 'lost' && <button style={{ background: '#fff', color: SC.textMute, border: `1px solid ${SC.border}`, padding: '8px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', whiteSpace: 'nowrap' }}>Archiver</button>}
                  <button style={{ background: 'none', color: SC.textMute, border: 'none', fontSize: 12, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4 }}>
                    <SCI name="Eye" size={12} /> Voir l'annonce
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ C5 — Détail mission côté freelance (livrer un jalon) ═══════
function SoleilFreelancerProject() {
  const stages = [
    { l: 'Acceptée', date: '13 mai', done: true },
    { l: 'Payée', date: '14 mai', done: true },
    { l: 'Active', date: 'depuis le 15 mai', current: true },
    { l: 'Livrée', date: 'fin prévue 30 juin', done: false },
    { l: 'Validée', date: 'paiement final', done: false },
  ];
  const milestones = [
    { l: 'Discovery & audit', amount: 2400, status: 'paid', date: 'livré le 20 mai · payé' },
    { l: 'Wireframes v1', amount: 3200, status: 'paid', date: 'livré le 5 juin · payé' },
    { l: 'Design system', amount: 4200, status: 'in_review', date: 'livré le 18 juin · en attente de validation', current: true },
    { l: 'Maquettes finales', amount: 2600, status: 'pending', date: 'à livrer pour le 30 juin' },
  ];

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SC.bg, fontFamily: SC.sans, color: SC.text }}>
      <SCSidebar active="proj" role="freelancer" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SCTopbar />
        <div style={{ flex: 1, overflow: 'auto' }}>
          {/* Header */}
          <div style={{ padding: '24px 40px 0', background: '#fff', borderBottom: `1px solid ${SC.border}` }}>
            <div style={{ fontSize: 12, color: SC.textMute, marginBottom: 14, display: 'flex', alignItems: 'center', gap: 6 }}>
              <SCI name="ArrowLeft" size={13} /> <span style={{ cursor: 'pointer' }}>Toutes mes missions</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 24, marginBottom: 22 }}>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 11, color: SC.textMute, marginBottom: 8, fontFamily: SC.mono, letterSpacing: '0.05em' }}>MISSION-2026-014 · Mission longue</div>
                <h1 style={{ fontFamily: SC.serif, fontSize: 34, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1, marginBottom: 12 }}>Refonte de l'app produit Nova v2</h1>
                <div style={{ display: 'flex', alignItems: 'center', gap: 14, fontSize: 13, color: SC.textMute }}>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
                    <SCPortrait id={2} size={28} />
                    <span>chez <strong style={{ color: SC.text }}>Nova Studio</strong></span>
                  </span>
                  <span style={{ width: 4, height: 4, borderRadius: '50%', background: SC.borderStrong }} />
                  <span>3 mois · 4 jours/sem</span>
                  <span style={{ width: 4, height: 4, borderRadius: '50%', background: SC.borderStrong }} />
                  <span>démarré le 15 mai</span>
                </div>
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                <button style={{ background: '#fff', border: `1px solid ${SC.borderStrong}`, padding: '10px 16px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}><SCI name="Chat" size={13} /> Conversation</button>
                <button style={{ background: SC.accent, color: '#fff', border: 'none', padding: '10px 18px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', boxShadow: '0 2px 8px rgba(232,93,74,0.25)', display: 'flex', alignItems: 'center', gap: 6 }}>
                  <SCI name="Send" size={13} /> Livrer un jalon
                </button>
              </div>
            </div>

            {/* Stepper */}
            <div style={{ paddingBottom: 24 }}>
              <div style={{ display: 'flex', alignItems: 'flex-start', position: 'relative' }}>
                {stages.map((s, i) => (
                  <div key={i} style={{ flex: 1, position: 'relative', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                    {i < stages.length - 1 && (
                      <div style={{ position: 'absolute', top: 14, left: '50%', right: '-50%', height: 2, background: stages[i + 1].done || stages[i + 1].current ? SC.green : SC.border, zIndex: 0 }} />
                    )}
                    <div style={{ width: 30, height: 30, borderRadius: '50%', background: s.current ? SC.accent : s.done ? SC.green : '#fff', border: s.current ? `3px solid ${SC.accentSoft}` : s.done ? 'none' : `2px solid ${SC.border}`, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1, position: 'relative', boxShadow: s.current ? '0 0 0 6px rgba(232,93,74,0.15)' : 'none' }}>
                      {s.done && <SCI name="Check" size={15} />}
                      {s.current && <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#fff' }} />}
                    </div>
                    <div style={{ marginTop: 10, textAlign: 'center' }}>
                      <div style={{ fontSize: 13, fontWeight: 600, color: s.current ? SC.accentDeep : s.done ? SC.text : SC.textMute, fontFamily: SC.serif, letterSpacing: '-0.01em' }}>{s.l}</div>
                      <div style={{ fontSize: 11, color: SC.textMute, marginTop: 2, fontStyle: s.current ? 'italic' : 'normal' }}>{s.date}</div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div style={{ padding: '28px 40px', display: 'grid', gridTemplateColumns: '1.6fr 1fr', gap: 24 }}>
            <div>
              {/* Action urgente */}
              <div style={{ background: 'linear-gradient(135deg, #fde9e3, #fde6ed)', border: `1px solid ${SC.accent}40`, borderRadius: 16, padding: 22, marginBottom: 18, display: 'flex', gap: 16, alignItems: 'center' }}>
                <div style={{ width: 44, height: 44, borderRadius: '50%', background: SC.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  <SCI name="Clock" size={20} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 11, color: SC.accentDeep, fontWeight: 700, letterSpacing: '0.08em', textTransform: 'uppercase', marginBottom: 4 }}>Prochaine action</div>
                  <div style={{ fontFamily: SC.serif, fontSize: 18, fontWeight: 600, marginBottom: 3 }}>Livre le jalon « Maquettes finales » d'ici le 30 juin</div>
                  <div style={{ fontSize: 12.5, color: SC.textMute }}>2 600 € en séquestre · sera versé sous 48 h après validation</div>
                </div>
                <button style={{ background: SC.text, color: '#fff', border: 'none', padding: '10px 18px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', whiteSpace: 'nowrap' }}>Marquer comme livré</button>
              </div>

              {/* Jalons */}
              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 28 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: 18 }}>
                  <div>
                    <h2 style={{ fontFamily: SC.serif, fontSize: 22, margin: 0, fontWeight: 500, letterSpacing: '-0.01em' }}>Jalons & livrables</h2>
                    <div style={{ fontSize: 12.5, color: SC.textMute, marginTop: 3 }}>2 livrés et payés · 1 en attente de validation client · 1 à venir</div>
                  </div>
                </div>

                {milestones.map((m, i) => {
                  const cfg = {
                    paid:      { dot: SC.green,  bg: SC.greenSoft, label: 'Payé',                icon: 'CheckCircle' },
                    in_review: { dot: SC.amber,  bg: SC.amberSoft, label: 'En attente client',   icon: 'Clock' },
                    pending:   { dot: SC.textSubtle, bg: SC.bg,    label: 'À livrer',            icon: 'Pin' },
                  }[m.status];
                  return (
                    <div key={i} style={{ padding: '14px 0', borderTop: i > 0 ? `1px solid ${SC.border}` : 'none', display: 'grid', gridTemplateColumns: '34px 1fr auto auto', gap: 16, alignItems: 'center', background: m.current ? `linear-gradient(90deg, ${SC.amberSoft} 0%, transparent 100%)` : 'transparent', marginLeft: m.current ? -28 : 0, marginRight: m.current ? -28 : 0, paddingLeft: m.current ? 28 : 0, paddingRight: m.current ? 28 : 0, borderTopColor: m.current ? 'transparent' : SC.border }}>
                      <div style={{ width: 30, height: 30, borderRadius: '50%', background: cfg.bg, color: cfg.dot, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <SCI name={cfg.icon} size={16} />
                      </div>
                      <div>
                        <div style={{ fontSize: 14.5, fontWeight: 600, fontFamily: SC.serif, marginBottom: 2 }}>{m.l}</div>
                        <div style={{ fontSize: 11.5, color: SC.textMute }}>{m.date}</div>
                      </div>
                      <span style={{ fontSize: 11, padding: '3px 10px', background: cfg.bg, color: cfg.dot, borderRadius: 999, fontWeight: 600 }}>{cfg.label}</span>
                      <div style={{ fontFamily: SC.serif, fontSize: 17, fontWeight: 600, minWidth: 90, textAlign: 'right' }}>{m.amount.toLocaleString('fr-FR')} €</div>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Right col */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {/* Ce que tu vas toucher */}
              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 22 }}>
                <div style={{ fontSize: 11, color: SC.textMute, marginBottom: 8, letterSpacing: '0.06em', textTransform: 'uppercase', fontWeight: 600 }}>Ce que tu vas toucher</div>
                <div style={{ fontFamily: SC.serif, fontSize: 32, fontWeight: 500, lineHeight: 1, letterSpacing: '-0.025em', marginBottom: 4 }}>10 850 €</div>
                <div style={{ fontSize: 12.5, color: SC.textMute, marginBottom: 16 }}>sur 12 400 € contractés · frais Atelier 12,5 %</div>

                <div style={{ background: SC.bg, borderRadius: 10, padding: 14, marginBottom: 12 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11.5, marginBottom: 6 }}>
                    <span style={{ color: SC.green, fontWeight: 600 }}>4 900 € versés</span>
                    <span style={{ color: SC.amber, fontWeight: 600 }}>3 675 € en séquestre</span>
                  </div>
                  <div style={{ height: 6, background: SC.border, borderRadius: 3, overflow: 'hidden', display: 'flex' }}>
                    <div style={{ width: '45%', height: '100%', background: SC.green }} />
                    <div style={{ width: '34%', height: '100%', background: SC.amber }} />
                  </div>
                  <div style={{ display: 'flex', gap: 14, marginTop: 8, fontSize: 10.5, color: SC.textMute }}>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><span style={{ width: 7, height: 7, borderRadius: 2, background: SC.green }} /> Versé</span>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><span style={{ width: 7, height: 7, borderRadius: 2, background: SC.amber }} /> En séquestre</span>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><span style={{ width: 7, height: 7, borderRadius: 2, background: SC.border }} /> À venir</span>
                  </div>
                </div>

                <button style={{ width: '100%', background: SC.text, color: '#fff', border: 'none', padding: '11px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Voir mon portefeuille</button>
              </div>

              {/* Client */}
              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 22 }}>
                <h3 style={{ fontFamily: SC.serif, fontSize: 17, margin: '0 0 14px', fontWeight: 600 }}>Ton interlocuteur</h3>
                <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 14 }}>
                  <SCPortrait id={4} size={48} />
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 14, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 5 }}>
                      Sophie A. <SCI name="Verified" size={12} />
                    </div>
                    <div style={{ fontSize: 11.5, color: SC.textMute }}>CPO chez Nova Studio</div>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 6 }}>
                  <button style={{ flex: 1, background: '#fff', border: `1px solid ${SC.border}`, padding: '8px', fontSize: 12, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 5 }}>
                    <SCI name="Chat" size={12} /> Message
                  </button>
                  <button style={{ flex: 1, background: '#fff', border: `1px solid ${SC.border}`, padding: '8px', fontSize: 12, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 5 }}>
                    <SCI name="Phone" size={12} /> Appeler
                  </button>
                </div>
              </div>

              {/* Documents */}
              <div style={{ background: '#fff', border: `1px solid ${SC.border}`, borderRadius: 16, padding: 22 }}>
                <h3 style={{ fontFamily: SC.serif, fontSize: 17, margin: '0 0 14px', fontWeight: 600 }}>Documents</h3>
                {[
                  { name: 'Brief de mission v2', size: '480 ko' },
                  { name: 'Contrat signé', size: '1,2 Mo' },
                  { name: 'Wireframes v1 (livré)', size: '8,4 Mo' },
                ].map((d, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 0', borderTop: i > 0 ? `1px solid ${SC.border}` : 'none' }}>
                    <div style={{ width: 30, height: 30, borderRadius: 8, background: SC.bg, color: SC.textMute, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SCI name="Doc" size={14} /></div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: 12.5, fontWeight: 600 }}>{d.name}</div>
                      <div style={{ fontSize: 10.5, color: SC.textMute }}>{d.size}</div>
                    </div>
                  </div>
                ))}
                <button style={{ width: '100%', marginTop: 12, background: SC.bg, border: `1px dashed ${SC.borderStrong}`, padding: '10px', fontSize: 12.5, fontWeight: 600, borderRadius: 10, cursor: 'pointer', color: SC.textMute, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}>
                  <SCI name="Plus" size={13} /> Ajouter un livrable
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

window.SoleilFreelancerDashboard = SoleilFreelancerDashboard;
window.SoleilOpportunities = SoleilOpportunities;
window.SoleilOpportunityDetail = SoleilOpportunityDetail;
window.SoleilMyApplications = SoleilMyApplications;
window.SoleilFreelancerProject = SoleilFreelancerProject;
