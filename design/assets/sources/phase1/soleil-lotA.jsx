// Lot A — Cycle transactionnel entreprise (Soleil v2)
// 5 écrans : Jobs · Détail Job (2 onglets) · Création Job · Détail Projet
// Réutilise SSidebar, STopbar, Portrait, S, SI depuis soleil.jsx (chargé avant)

const SA = window.S || {
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

const SAI = window.SI || (() => null);
// On utilise SSidebar et STopbar via window
const SASidebar = (props) => window.SSidebar ? window.SSidebar(props) : null;
const SATopbar = () => window.STopbar ? window.STopbar() : null;
const SAPortrait = (props) => window.Portrait ? window.Portrait(props) : null;

// ─── Petits helpers visuels ────────────────────────────────────────
function PageHeader({ kicker, title, italicPart, sub, rightSlot, breadcrumb }) {
  return (
    <div style={{ padding: '28px 36px 22px', borderBottom: `1px solid ${SA.border}`, background: '#fff' }}>
      {breadcrumb && <div style={{ fontSize: 12, color: SA.textMute, marginBottom: 10, display: 'flex', alignItems: 'center', gap: 6 }}>{breadcrumb}</div>}
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', gap: 24 }}>
        <div style={{ flex: 1 }}>
          {kicker && (
            <div style={{ fontSize: 11, color: SA.accent, marginBottom: 8, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase', display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ width: 24, height: 1, background: SA.accent }} /> {kicker}
            </div>
          )}
          <h1 style={{ fontFamily: SA.serif, fontSize: 38, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1 }}>
            {title}
            {italicPart && <span style={{ fontStyle: 'italic', color: SA.accent }}> {italicPart}</span>}
          </h1>
          {sub && <p style={{ fontSize: 14, color: SA.textMute, margin: '6px 0 0', maxWidth: 580 }}>{sub}</p>}
        </div>
        {rightSlot}
      </div>
    </div>
  );
}

function StatusPill({ label, kind = 'open' }) {
  const styles = {
    open:    { bg: SA.greenSoft, color: SA.green, dot: SA.green },
    closed:  { bg: SA.bg, color: SA.textMute, dot: SA.textSubtle },
    draft:   { bg: SA.amberSoft, color: SA.amber, dot: SA.amber },
    paused:  { bg: SA.amberSoft, color: SA.amber, dot: SA.amber },
    urgent:  { bg: SA.accentSoft, color: SA.accentDeep, dot: SA.accent },
  };
  const s = styles[kind];
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 11, padding: '4px 10px', background: s.bg, color: s.color, borderRadius: 999, fontWeight: 600 }}>
      <span style={{ width: 6, height: 6, borderRadius: '50%', background: s.dot }} /> {label}
    </span>
  );
}

// ═══ A1 — Jobs (liste annonces) ═══════════════════════════════════
function SoleilJobsList() {
  const jobs = [
    { id: 1, title: 'Refonte de l\'app produit Nova', desc: 'Nous cherchons un·e UX/UI Designer senior pour piloter la refonte complète de notre app SaaS. Mission longue durée, 4j/sem, en hybride Paris.', tags: ['UX Design', 'Mobile', '4j / semaine'], budget: '8 000 — 12 000 €', deadline: 'Démarrage le 27 mai', applicants: 12, new: 4, status: 'open', kind: 'Mission longue', posted: 'il y a 3 jours', shortlist: 3 },
    { id: 2, title: 'Identité visuelle & charte graphique', desc: 'Création complète de la nouvelle identité de marque pour notre lancement Q3. Logo, charte, déclinaisons digitales.', tags: ['Branding', 'Logo', 'Charte'], budget: '6 000 — 9 000 €', deadline: 'Réception avant le 15 juin', applicants: 28, new: 12, status: 'open', kind: 'Projet ponctuel', posted: 'il y a 1 semaine', shortlist: 5 },
    { id: 3, title: 'Audit SEO technique', desc: 'Audit complet de notre stack Next.js. On cherche quelqu\'un qui sait lire un Lighthouse et écrire un rapport actionnable.', tags: ['SEO', 'Next.js', 'Audit'], budget: '3 000 — 4 500 €', deadline: 'Mission de 2 semaines', applicants: 8, new: 0, status: 'paused', kind: 'Projet ponctuel', posted: 'il y a 2 semaines', shortlist: 2 },
    { id: 4, title: 'Développement back-end API paiement', desc: 'Refonte de notre brique paiement Stripe Connect. Architecture à reprendre, dette technique, mais base saine.', tags: ['Node.js', 'Stripe', 'API'], budget: '15 000 — 22 000 €', deadline: 'Mission longue · démarrage juin', applicants: 6, new: 2, status: 'draft', kind: 'Mission longue', posted: 'brouillon', shortlist: 0 },
  ];

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SA.bg, fontFamily: SA.sans, color: SA.text }}>
      <SASidebar active="jobs" role="enterprise" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SATopbar />
        <PageHeader
          kicker={`${jobs.length} annonces · 54 candidats au total`}
          title="Tes annonces,"
          italicPart="ton vivier."
          sub="Suis tes recrutements actifs, gère les candidatures et publie de nouvelles missions."
          rightSlot={
            <div style={{ display: 'flex', gap: 8 }}>
              <button style={{ background: '#fff', border: `1px solid ${SA.borderStrong}`, padding: '11px 16px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
                <SAI name="Sliders" size={14} /> Filtrer
              </button>
              <button style={{ background: SA.accent, color: '#fff', border: 'none', padding: '11px 22px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 7, boxShadow: '0 2px 8px rgba(232,93,74,0.25)' }}>
                <SAI name="Plus" size={15} /> Publier une annonce
              </button>
            </div>
          }
        />

        <div style={{ flex: 1, overflow: 'auto', padding: '24px 36px' }}>
          {/* Tabs */}
          <div style={{ display: 'flex', gap: 24, borderBottom: `1px solid ${SA.border}`, marginBottom: 22 }}>
            {[
              { l: 'Toutes', n: 4, active: true },
              { l: 'Ouvertes', n: 2 },
              { l: 'En pause', n: 1 },
              { l: 'Brouillons', n: 1 },
              { l: 'Archivées', n: 8 },
            ].map((t, i) => (
              <div key={i} style={{ padding: '10px 2px', borderBottom: t.active ? `2px solid ${SA.accent}` : '2px solid transparent', fontSize: 14, fontWeight: 600, color: t.active ? SA.text : SA.textMute, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 7 }}>
                {t.l} <span style={{ fontSize: 11, padding: '1px 7px', background: t.active ? SA.accentSoft : SA.bg, color: t.active ? SA.accentDeep : SA.textMute, borderRadius: 999, fontWeight: 700 }}>{t.n}</span>
              </div>
            ))}
          </div>

          {/* Job cards */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {jobs.map((j) => (
              <div key={j.id} style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 18, padding: 24, cursor: 'pointer', display: 'grid', gridTemplateColumns: '1fr 280px', gap: 28 }}>
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
                    <span style={{ fontSize: 11, color: SA.textMute, fontFamily: SA.mono, letterSpacing: '0.05em' }}>{j.kind} · {j.posted}</span>
                    <StatusPill label={j.status === 'open' ? 'Ouverte' : j.status === 'paused' ? 'En pause' : 'Brouillon'} kind={j.status} />
                  </div>
                  <h3 style={{ fontFamily: SA.serif, fontSize: 24, margin: 0, fontWeight: 500, marginBottom: 8, letterSpacing: '-0.015em' }}>{j.title}</h3>
                  <p style={{ fontSize: 14, color: SA.textMute, margin: 0, marginBottom: 14, lineHeight: 1.55, maxWidth: 640, textWrap: 'pretty' }}>{j.desc}</p>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 16 }}>
                    {j.tags.map((t, ti) => (
                      <span key={ti} style={{ fontSize: 11.5, padding: '4px 10px', background: SA.bg, borderRadius: 999, fontWeight: 500, color: SA.textMute }}>{t}</span>
                    ))}
                  </div>
                  <div style={{ display: 'flex', gap: 22, fontSize: 12.5, color: SA.textMute, alignItems: 'center' }}>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SAI name="Euro" size={13} /> <strong style={{ color: SA.text, fontFamily: SA.serif, fontSize: 14 }}>{j.budget}</strong></span>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SAI name="Clock" size={13} /> {j.deadline}</span>
                  </div>
                </div>

                {/* Right rail — applicants */}
                <div style={{ borderLeft: `1px solid ${SA.border}`, paddingLeft: 24, display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
                  <div>
                    <div style={{ fontSize: 11, color: SA.textMute, marginBottom: 6, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600 }}>Candidatures</div>
                    <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 14 }}>
                      <span style={{ fontFamily: SA.serif, fontSize: 36, fontWeight: 500, lineHeight: 1, letterSpacing: '-0.025em' }}>{j.applicants}</span>
                      {j.new > 0 && <span style={{ fontSize: 11, color: SA.accent, fontWeight: 700, background: SA.accentSoft, padding: '2px 8px', borderRadius: 999 }}>+{j.new} nouvelles</span>}
                    </div>
                    {j.applicants > 0 && (
                      <>
                        <div style={{ display: 'flex', marginBottom: 10 }}>
                          {[0, 2, 3, 4, 1].slice(0, Math.min(5, j.applicants)).map((p, i) => (
                            <div key={i} style={{ marginLeft: i === 0 ? 0 : -8, border: '2px solid #fff', borderRadius: '50%' }}>
                              <SAPortrait id={p} size={30} />
                            </div>
                          ))}
                          {j.applicants > 5 && (
                            <div style={{ marginLeft: -8, width: 30, height: 30, borderRadius: '50%', background: SA.bg, border: '2px solid #fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, fontWeight: 700, color: SA.textMute }}>+{j.applicants - 5}</div>
                          )}
                        </div>
                        {j.shortlist > 0 && <div style={{ fontSize: 11.5, color: SA.textMute, marginBottom: 10 }}>Tu as présélectionné <strong style={{ color: SA.text }}>{j.shortlist}</strong> candidat·es</div>}
                      </>
                    )}
                  </div>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <button style={{ flex: 1, background: SA.text, color: '#fff', border: 'none', padding: '9px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Voir les candidats</button>
                    <button style={{ background: '#fff', border: `1px solid ${SA.border}`, width: 36, height: 36, borderRadius: '50%', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SAI name="More" size={15} /></button>
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

// ═══ A2 — Détail Job · onglet Description ════════════════════════
function SoleilJobDetailDesc() {
  return <SoleilJobDetail tab="desc" />;
}
function SoleilJobDetailCands() {
  return <SoleilJobDetail tab="cands" />;
}

function SoleilJobDetail({ tab }) {
  const candidates = [
    { name: 'Élise Marchand', title: 'UX Designer · Brand', tjm: '650 €', match: 96, applied: 'il y a 2 h', pid: 1, verified: true, status: 'shortlist', note: "Méthodo solide, exactement le profil qu'on cherche.", quote: "J'aime poser un cadre méthodo dès la première semaine." },
    { name: 'Julien Petit', title: 'Brand & DA senior', tjm: '720 €', match: 88, applied: 'hier', pid: 0, verified: true, status: 'shortlist', note: 'Portfolio très cohérent. À contacter.', quote: "Une marque, c'est avant tout un point de vue." },
    { name: 'Camille Dubois', title: 'Product Designer', tjm: '600 €', match: 84, applied: 'il y a 3 jours', pid: 3, verified: false, status: 'new', quote: "Le mobile, c'est 80% de l'usage." },
    { name: 'Léa Fontaine', title: 'Motion & Direction artistique', tjm: '520 €', match: 78, applied: 'il y a 4 jours', pid: 2, verified: true, status: 'pending', quote: 'Le mouvement raconte ce que les mots ne peuvent pas.' },
    { name: 'Mehdi Bensalem', title: 'Product Designer', tjm: '750 €', match: 72, applied: 'il y a 1 semaine', pid: 4, verified: true, status: 'pending' },
    { name: 'Théo Renaud', title: 'UI Designer', tjm: '580 €', match: 65, applied: 'il y a 1 semaine', pid: 5, verified: true, status: 'declined', note: 'Pas le bon niveau de séniorité.' },
  ];

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SA.bg, fontFamily: SA.sans, color: SA.text }}>
      <SASidebar active="jobs" role="enterprise" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SATopbar />

        {/* Header */}
        <div style={{ padding: '24px 36px 0', background: '#fff', borderBottom: `1px solid ${SA.border}` }}>
          <div style={{ fontSize: 12, color: SA.textMute, marginBottom: 14, display: 'flex', alignItems: 'center', gap: 6 }}>
            <SAI name="ArrowLeft" size={13} /> <span style={{ cursor: 'pointer' }}>Toutes mes annonces</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 24, marginBottom: 18 }}>
            <div style={{ flex: 1 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
                <span style={{ fontSize: 11, color: SA.textMute, fontFamily: SA.mono, letterSpacing: '0.05em' }}>Mission longue · Publiée il y a 3 jours</span>
                <StatusPill label="Ouverte" kind="open" />
              </div>
              <h1 style={{ fontFamily: SA.serif, fontSize: 36, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1 }}>Refonte de l'app produit Nova</h1>
              <div style={{ display: 'flex', gap: 22, fontSize: 13, color: SA.textMute, marginTop: 10, alignItems: 'center', flexWrap: 'wrap' }}>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SAI name="Euro" size={13} /> <strong style={{ color: SA.text, fontFamily: SA.serif, fontSize: 15 }}>8 000 — 12 000 €</strong></span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SAI name="Clock" size={13} /> Démarrage le 27 mai</span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SAI name="MapPin" size={13} /> Hybride · Paris</span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><SAI name="Eye" size={13} /> 247 vues</span>
              </div>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <button style={{ background: '#fff', border: `1px solid ${SA.borderStrong}`, padding: '10px 16px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}><SAI name="Edit" size={13} /> Modifier</button>
              <button style={{ background: SA.text, color: '#fff', border: 'none', padding: '10px 18px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Mettre en avant</button>
              <button style={{ background: '#fff', border: `1px solid ${SA.border}`, width: 38, height: 38, borderRadius: '50%', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SAI name="More" size={15} /></button>
            </div>
          </div>

          {/* Tabs */}
          <div style={{ display: 'flex', gap: 28 }}>
            <div style={{ padding: '12px 2px', borderBottom: tab === 'desc' ? `2px solid ${SA.accent}` : '2px solid transparent', fontSize: 14, fontWeight: 600, color: tab === 'desc' ? SA.text : SA.textMute, cursor: 'pointer' }}>Description</div>
            <div style={{ padding: '12px 2px', borderBottom: tab === 'cands' ? `2px solid ${SA.accent}` : '2px solid transparent', fontSize: 14, fontWeight: 600, color: tab === 'cands' ? SA.text : SA.textMute, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 7 }}>
              Candidatures <span style={{ fontSize: 11, padding: '1px 7px', background: SA.accentSoft, color: SA.accentDeep, borderRadius: 999, fontWeight: 700 }}>12</span>
            </div>
          </div>
        </div>

        <div style={{ flex: 1, overflow: 'auto', padding: '28px 36px' }}>
          {tab === 'desc' && (
            <div style={{ display: 'grid', gridTemplateColumns: '1.7fr 1fr', gap: 24 }}>
              <div>
                <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 16, padding: 32, marginBottom: 20 }}>
                  <h2 style={{ fontFamily: SA.serif, fontSize: 22, margin: 0, marginBottom: 14, fontWeight: 500, letterSpacing: '-0.01em' }}>Le contexte</h2>
                  <p style={{ fontSize: 15, lineHeight: 1.7, margin: 0, marginBottom: 14, textWrap: 'pretty' }}>
                    Nous sommes <strong style={{ color: SA.accent }}>Nova Studio</strong>, une SaaS B2B qui aide les studios créatifs à gérer leurs projets. Nous avons levé en Série A en mars et entamons une refonte produit ambitieuse pour les 12 prochains mois.
                  </p>
                  <p style={{ fontSize: 15, lineHeight: 1.7, margin: 0, color: SA.textMute, textWrap: 'pretty' }}>
                    L'app actuelle a 3 ans, elle a beaucoup grandi et c'est devenu touffu. On veut tout reprendre en partant des usages réels, en allégeant l'interface, et en posant un design system solide pour la suite.
                  </p>
                </div>

                {/* Espace vidéo */}
                <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 16, padding: 32 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
                    <SAI name="Play" size={18} />
                    <h2 style={{ fontFamily: SA.serif, fontSize: 22, margin: 0, fontWeight: 500, letterSpacing: '-0.01em' }}>Mot du recruteur</h2>
                  </div>
                  <div style={{ position: 'relative', borderRadius: 14, overflow: 'hidden', background: 'linear-gradient(135deg, #fbf0dc 0%, #fde6ed 50%, #fde9e3 100%)', aspectRatio: '16 / 9', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}>
                    <div style={{ position: 'absolute', inset: 0, background: 'radial-gradient(circle at 30% 50%, rgba(232,93,74,0.18), transparent 60%)' }} />
                    <SAPortrait id={4} size={120} rounded={20} />
                    <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                      <div style={{ width: 64, height: 64, borderRadius: '50%', background: 'rgba(255,255,255,0.95)', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 8px 24px rgba(0,0,0,0.18)' }}>
                        <div style={{ width: 0, height: 0, borderTop: '12px solid transparent', borderBottom: '12px solid transparent', borderLeft: `18px solid ${SA.accent}`, marginLeft: 4 }} />
                      </div>
                    </div>
                    <div style={{ position: 'absolute', bottom: 14, left: 16, right: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', color: SA.text }}>
                      <div>
                        <div style={{ fontSize: 12, fontFamily: SA.mono, color: SA.accentDeep, fontWeight: 600, marginBottom: 3 }}>SOPHIE A. · CPO</div>
                        <div style={{ fontFamily: SA.serif, fontSize: 15, fontStyle: 'italic', fontWeight: 500 }}>« Je te raconte le projet en 90 secondes. »</div>
                      </div>
                      <div style={{ fontSize: 12, fontFamily: SA.mono, color: SA.text, opacity: 0.7, background: 'rgba(255,255,255,0.85)', padding: '3px 8px', borderRadius: 6 }}>1:24</div>
                    </div>
                  </div>
                </div>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 16, padding: 22 }}>
                  <h3 style={{ fontFamily: SA.serif, fontSize: 18, margin: 0, marginBottom: 14, fontWeight: 600 }}>Compétences attendues</h3>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                    {['Figma', 'Design System', 'UX Research', 'SaaS B2B', 'Mobile', 'Notion', 'Webflow'].map((s, i) => (
                      <span key={i} style={{ fontSize: 12, padding: '5px 11px', background: SA.bg, borderRadius: 999, fontWeight: 500 }}>{s}</span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          )}

          {tab === 'cands' && (
            <div>
              {/* Filters bar */}
              <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 22, flexWrap: 'wrap' }}>
                {[
                  { l: 'Toutes', n: 12, active: true },
                  { l: 'Nouvelles', n: 4 },
                  { l: 'Présélectionnées', n: 3 },
                  { l: 'En attente', n: 2 },
                  { l: 'Refusées', n: 3 },
                ].map((c, i) => (
                  <button key={i} style={{ background: c.active ? SA.text : '#fff', color: c.active ? '#fff' : SA.text, border: c.active ? 'none' : `1px solid ${SA.border}`, padding: '8px 14px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
                    {c.l} <span style={{ fontSize: 11, opacity: 0.7 }}>{c.n}</span>
                  </button>
                ))}
                <div style={{ flex: 1 }} />
                <span style={{ fontSize: 12, color: SA.textMute }}>Tri</span>
                <button style={{ background: 'none', border: 'none', fontSize: 13, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 4, cursor: 'pointer' }}>Plus récente <SAI name="ChevronDown" size={13} /></button>
              </div>

              {/* Candidates list */}
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {candidates.map((c, i) => (
                  <div key={i} style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 16, padding: 22, display: 'grid', gridTemplateColumns: '64px 1fr auto', gap: 20, alignItems: 'center', cursor: 'pointer', opacity: c.status === 'declined' ? 0.6 : 1 }}>
                    <div style={{ position: 'relative' }}>
                      <SAPortrait id={c.pid} size={56} rounded={14} />
                      {c.verified && <div style={{ position: 'absolute', bottom: -3, right: -3, width: 22, height: 22, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 1px 4px rgba(0,0,0,0.12)' }}><SAI name="Verified" size={15} /></div>}
                    </div>
                    <div style={{ minWidth: 0 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 4 }}>
                        <span style={{ fontFamily: SA.serif, fontSize: 18, fontWeight: 600, letterSpacing: '-0.01em' }}>{c.name}</span>
                        {c.status === 'shortlist' && <span style={{ fontSize: 10.5, padding: '2px 8px', background: SA.accentSoft, color: SA.accentDeep, borderRadius: 999, fontWeight: 700, letterSpacing: '0.05em', textTransform: 'uppercase', display: 'inline-flex', alignItems: 'center', gap: 4 }}><SAI name="Star" size={10} /> Présélectionné·e</span>}
                        {c.status === 'new' && <span style={{ fontSize: 10.5, padding: '2px 8px', background: SA.greenSoft, color: SA.green, borderRadius: 999, fontWeight: 700, letterSpacing: '0.05em', textTransform: 'uppercase' }}>Nouveau</span>}
                        {c.status === 'declined' && <span style={{ fontSize: 10.5, padding: '2px 8px', background: SA.bg, color: SA.textMute, borderRadius: 999, fontWeight: 600 }}>Refusé·e</span>}
                      </div>
                      <div style={{ fontSize: 13, color: SA.textMute, marginBottom: 6 }}>{c.title} · TJM {c.tjm} · {c.applied}</div>
                      {c.quote && <div style={{ fontFamily: SA.serif, fontSize: 13.5, fontStyle: 'italic', color: SA.text, lineHeight: 1.4, marginBottom: c.note ? 8 : 0 }}>« {c.quote} »</div>}
                      {c.note && <div style={{ fontSize: 12, color: SA.amber, padding: '6px 10px', background: SA.amberSoft, borderRadius: 8, display: 'inline-flex', alignItems: 'center', gap: 6, marginTop: 4 }}><SAI name="Pin" size={11} /> Ta note : {c.note}</div>}
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 6, alignItems: 'flex-end' }}>
                      <button style={{ background: SA.text, color: '#fff', border: 'none', padding: '8px 16px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', whiteSpace: 'nowrap' }}>Voir le profil</button>
                      <button style={{ background: '#fff', color: SA.accent, border: `1px solid ${SA.accent}40`, padding: '8px 16px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5, whiteSpace: 'nowrap' }}><SAI name="Send" size={11} /> Message</button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ═══ A3 — Création / édition d'un Job ════════════════════════════
function SoleilJobCreate() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SA.bg, fontFamily: SA.sans, color: SA.text }}>
      <SASidebar active="jobs" role="enterprise" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SATopbar />

        <div style={{ flex: 1, overflow: 'auto' }}>
          <div style={{ maxWidth: 1040, margin: '0 auto', padding: '40px 36px 64px' }}>
            <div style={{ fontSize: 12, color: SA.textMute, marginBottom: 14, display: 'flex', alignItems: 'center', gap: 6 }}>
              <SAI name="ArrowLeft" size={13} /> <span style={{ cursor: 'pointer' }}>Toutes mes annonces</span>
            </div>
            <div style={{ marginBottom: 32 }}>
              <h1 style={{ fontFamily: SA.serif, fontSize: 42, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1 }}>Publier <span style={{ fontStyle: 'italic', color: SA.accent }}>une nouvelle annonce.</span></h1>
              <p style={{ fontSize: 15, color: SA.textMute, margin: '10px 0 0', maxWidth: 620 }}>Décris la mission, le budget et le profil que tu cherches. Plus c'est précis, plus les candidatures sont pertinentes.</p>
            </div>

            {/* Form card */}
            <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 20, padding: 40 }}>
              {/* Titre */}
              <div style={{ marginBottom: 28 }}>
                <label style={{ display: 'block', fontSize: 14, fontWeight: 600, marginBottom: 8 }}>Titre de l'annonce</label>
                <input defaultValue="Refonte de l'app produit Nova" placeholder="ex. Refonte de notre app mobile..." style={{ width: '100%', border: `1.5px solid ${SA.accent}`, borderRadius: 14, padding: '12px 16px', fontSize: 15, fontFamily: SA.sans, outline: 'none', background: '#fff' }} />
              </div>
              {/* Compétences */}
              <div style={{ marginBottom: 28 }}>
                <label style={{ display: 'block', fontSize: 14, fontWeight: 600, marginBottom: 4 }}>Compétences attendues</label>
                <div style={{ fontSize: 12.5, color: SA.textMute, marginBottom: 12 }}>Tape pour rechercher, ou choisis dans les suggestions ci-dessous.</div>
                <div style={{ border: `1.5px solid ${SA.accent}`, borderRadius: 14, padding: '10px 14px', background: '#fff', display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 6, minHeight: 50 }}>
                  {['UX Design', 'Design System', 'Figma', 'SaaS B2B'].map((t, i) => (
                    <span key={i} style={{ fontSize: 12.5, padding: '5px 10px', background: SA.accentSoft, color: SA.accentDeep, borderRadius: 999, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                      {t} <span style={{ cursor: 'pointer', opacity: 0.6 }}>×</span>
                    </span>
                  ))}
                  <input placeholder="Ajouter une compétence..." style={{ border: 'none', outline: 'none', flex: 1, minWidth: 160, fontSize: 14, padding: 4, fontFamily: SA.sans }} />
                </div>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 10 }}>
                  <span style={{ fontSize: 11, color: SA.textSubtle, marginRight: 4, padding: '5px 0' }}>Suggestions :</span>
                  {['Mobile design', 'UX Research', 'Notion', 'Webflow', 'Brand'].map((s, i) => (
                    <button key={i} style={{ fontSize: 12, padding: '5px 10px', background: SA.bg, border: `1px dashed ${SA.borderStrong}`, borderRadius: 999, fontWeight: 500, color: SA.textMute, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                      <SAI name="Plus" size={11} /> {s}
                    </button>
                  ))}
                </div>
              </div>

              {/* Type de mission */}
              <div style={{ marginBottom: 28 }}>
                <label style={{ display: 'block', fontSize: 14, fontWeight: 600, marginBottom: 12 }}>Type de mission</label>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10 }}>
                  {[
                    { l: 'Projet ponctuel', d: 'Livraison définie, durée < 2 mois', icon: 'Pulse' },
                    { l: 'Mission longue', d: 'Collaboration > 2 mois', icon: 'Pin', active: true },
                    { l: 'Régie temps plein', d: 'Présence quotidienne', icon: 'Briefcase' },
                  ].map((o, i) => (
                    <button key={i} style={{ background: o.active ? SA.accentSoft : '#fff', border: `1.5px solid ${o.active ? SA.accent : SA.border}`, borderRadius: 14, padding: '16px 14px', textAlign: 'left', cursor: 'pointer', display: 'flex', flexDirection: 'column', gap: 8 }}>
                      <SAI name={o.icon} size={20} />
                      <div style={{ fontSize: 14, fontWeight: 600, color: o.active ? SA.accentDeep : SA.text }}>{o.l}</div>
                      <div style={{ fontSize: 11.5, color: SA.textMute, lineHeight: 1.4 }}>{o.d}</div>
                    </button>
                  ))}
                </div>
              </div>

              {/* Durée + budget */}
              <div style={{ marginBottom: 28, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
                <div>
                  <label style={{ display: 'block', fontSize: 14, fontWeight: 600, marginBottom: 8 }}>Durée estimée</label>
                  <div style={{ position: 'relative' }}>
                    <select style={{ width: '100%', appearance: 'none', border: `1px solid ${SA.borderStrong}`, borderRadius: 14, padding: '12px 16px', fontSize: 14, background: '#fff', fontFamily: SA.sans, color: SA.text, cursor: 'pointer' }}>
                      <option>3 à 6 mois</option>
                    </select>
                    <span style={{ position: 'absolute', right: 14, top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none' }}><SAI name="ChevronDown" size={15} /></span>
                  </div>
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: 14, fontWeight: 600, marginBottom: 8 }}>Fourchette de budget</label>
                  <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                    <div style={{ flex: 1, position: 'relative' }}>
                      <input defaultValue="8 000" style={{ width: '100%', border: `1px solid ${SA.borderStrong}`, borderRadius: 14, padding: '12px 36px 12px 16px', fontSize: 14, fontFamily: SA.sans, outline: 'none', background: '#fff' }} />
                      <span style={{ position: 'absolute', right: 14, top: '50%', transform: 'translateY(-50%)', color: SA.textMute, fontSize: 13, fontFamily: SA.serif }}>€</span>
                    </div>
                    <span style={{ color: SA.textMute, fontSize: 13 }}>→</span>
                    <div style={{ flex: 1, position: 'relative' }}>
                      <input defaultValue="12 000" style={{ width: '100%', border: `1px solid ${SA.borderStrong}`, borderRadius: 14, padding: '12px 36px 12px 16px', fontSize: 14, fontFamily: SA.sans, outline: 'none', background: '#fff' }} />
                      <span style={{ position: 'absolute', right: 14, top: '50%', transform: 'translateY(-50%)', color: SA.textMute, fontSize: 13, fontFamily: SA.serif }}>€</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Mode de travail */}
              <div style={{ marginBottom: 28 }}>
                <label style={{ display: 'block', fontSize: 14, fontWeight: 600, marginBottom: 12 }}>Mode de travail</label>
                <div style={{ display: 'flex', gap: 10 }}>
                  {[
                    { l: 'Sur site', sub: 'Paris', icon: 'Building' },
                    { l: 'Hybride', sub: '2-3 j/sem', icon: 'Layers', active: true },
                    { l: '100 % remote', sub: 'Aucun déplacement', icon: 'Globe' },
                  ].map((o, i) => (
                    <button key={i} style={{ flex: 1, background: o.active ? SA.accentSoft : '#fff', border: `1.5px solid ${o.active ? SA.accent : SA.border}`, borderRadius: 14, padding: '14px 16px', textAlign: 'left', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 12 }}>
                      <SAI name={o.icon} size={20} />
                      <div>
                        <div style={{ fontSize: 14, fontWeight: 600, color: o.active ? SA.accentDeep : SA.text }}>{o.l}</div>
                        <div style={{ fontSize: 11.5, color: SA.textMute }}>{o.sub}</div>
                      </div>
                    </button>
                  ))}
                </div>
              </div>

              {/* Vidéo de présentation */}
              <div style={{ marginBottom: 28 }}>
                <label style={{ display: 'block', fontSize: 14, fontWeight: 600, marginBottom: 4 }}>Vidéo de présentation <span style={{ fontWeight: 400, color: SA.textMute, fontSize: 12 }}>· optionnel, mais ça change tout</span></label>
                <div style={{ fontSize: 12.5, color: SA.textMute, marginBottom: 12 }}>30 à 90 secondes pour parler du contexte avec ta voix. Les annonces avec vidéo reçoivent <strong style={{ color: SA.accent }}>3× plus de candidatures</strong>.</div>
                <div style={{ border: `1.5px dashed ${SA.borderStrong}`, borderRadius: 14, background: '#fff', padding: 24, display: 'flex', alignItems: 'center', gap: 18 }}>
                  <div style={{ flexShrink: 0, width: 88, height: 64, borderRadius: 10, background: 'linear-gradient(135deg, #fbf0dc 0%, #fde6ed 100%)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <div style={{ width: 36, height: 36, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 2px 8px rgba(0,0,0,0.08)' }}>
                      <div style={{ width: 0, height: 0, borderTop: '7px solid transparent', borderBottom: '7px solid transparent', borderLeft: `11px solid ${SA.accent}`, marginLeft: 3 }} />
                    </div>
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 13.5, fontWeight: 600, marginBottom: 3 }}>Ajoute une vidéo (.mp4, .mov · 200 Mo max)</div>
                    <div style={{ fontSize: 12, color: SA.textMute, fontStyle: 'italic', fontFamily: SA.serif }}>Tournée au téléphone, naturel, c'est parfait. On s'occupe du reste.</div>
                  </div>
                  <button style={{ background: '#fff', border: `1px solid ${SA.borderStrong}`, padding: '9px 16px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
                    <SAI name="Plus" size={13} /> Choisir un fichier
                  </button>
                </div>
              </div>

              {/* Description longue */}
              <div>
                <label style={{ display: 'block', fontSize: 14, fontWeight: 600, marginBottom: 4 }}>Détaille la mission</label>
                <div style={{ fontSize: 12.5, color: SA.textMute, marginBottom: 10 }}>Le contexte, les enjeux, l'équipe en place. Plus c'est concret, mieux c'est.</div>
                <div style={{ border: `1px solid ${SA.borderStrong}`, borderRadius: 14, background: '#fff', overflow: 'hidden' }}>
                  <div style={{ display: 'flex', gap: 4, padding: '8px 12px', borderBottom: `1px solid ${SA.border}`, background: SA.bg }}>
                    {['B', 'I', 'U', '·', '"', '⏎'].map((b, i) => (
                      <button key={i} style={{ width: 28, height: 28, border: 'none', background: 'transparent', borderRadius: 6, cursor: 'pointer', fontSize: 13, fontWeight: i === 0 ? 700 : 500, fontStyle: i === 1 ? 'italic' : 'normal', color: SA.textMute }}>{b}</button>
                    ))}
                  </div>
                  <textarea defaultValue="Nous sommes Nova Studio, une SaaS B2B qui aide les studios créatifs à gérer leurs projets..." style={{ width: '100%', minHeight: 140, border: 'none', outline: 'none', padding: '14px 16px', fontSize: 14, fontFamily: SA.sans, lineHeight: 1.6, resize: 'vertical', color: SA.text }} />
                </div>
                <div style={{ fontSize: 11.5, color: SA.textMute, marginTop: 6, textAlign: 'right', fontStyle: 'italic', fontFamily: SA.serif }}>~ 280 / 2000 caractères</div>
              </div>
            </div>

            {/* Footer actions */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 24 }}>
              <button style={{ background: 'none', border: 'none', fontSize: 14, color: SA.textMute, fontWeight: 500, cursor: 'pointer' }}>Annuler</button>
              <div style={{ display: 'flex', gap: 8 }}>
                <button style={{ background: '#fff', border: `1px solid ${SA.borderStrong}`, padding: '12px 20px', fontSize: 13.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Enregistrer en brouillon</button>
                <button style={{ background: SA.accent, color: '#fff', border: 'none', padding: '12px 28px', fontSize: 13.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6, boxShadow: '0 2px 8px rgba(232,93,74,0.25)' }}>
                  Publier l'annonce <SAI name="ArrowRight" size={14} />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ A4 — Détail Projet (stepper + frais plateforme) ════════════
function SoleilProjectDetail() {
  const stages = [
    { l: 'Créée', date: '12 mai', done: true },
    { l: 'Acceptée', date: '13 mai', done: true },
    { l: 'Payée', date: '14 mai', done: true },
    { l: 'Active', date: 'depuis le 15 mai', current: true },
    { l: 'Terminée', date: 'fin prévue 30 juin', done: false },
  ];

  const milestones = [
    { l: 'Discovery & audit', amount: 2400, status: 'paid', date: '20 mai' },
    { l: 'Wireframes v1', amount: 3200, status: 'paid', date: '5 juin' },
    { l: 'Design system', amount: 4200, status: 'in_review', date: 'livré le 18 juin' },
    { l: 'Maquettes finales', amount: 2600, status: 'pending', date: 'attendu 30 juin' },
  ];

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SA.bg, fontFamily: SA.sans, color: SA.text }}>
      <SASidebar active="proj" role="enterprise" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SATopbar />

        <div style={{ flex: 1, overflow: 'auto' }}>
          <div style={{ padding: '24px 36px 0', background: '#fff', borderBottom: `1px solid ${SA.border}` }}>
            <div style={{ fontSize: 12, color: SA.textMute, marginBottom: 14, display: 'flex', alignItems: 'center', gap: 6 }}>
              <SAI name="ArrowLeft" size={13} /> <span style={{ cursor: 'pointer' }}>Tous mes projets</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 24, marginBottom: 22 }}>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 11, color: SA.textMute, marginBottom: 8, fontFamily: SA.mono, letterSpacing: '0.05em' }}>PROJET-2026-014 · Mission longue</div>
                <h1 style={{ fontFamily: SA.serif, fontSize: 34, margin: 0, fontWeight: 400, letterSpacing: '-0.025em', lineHeight: 1.1, marginBottom: 12 }}>Refonte de l'app produit Nova v2</h1>
                <div style={{ display: 'flex', alignItems: 'center', gap: 14, fontSize: 13, color: SA.textMute }}>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
                    <SAPortrait id={1} size={28} />
                    <span>avec <strong style={{ color: SA.text }}>Élise Marchand</strong></span>
                  </span>
                  <span style={{ width: 4, height: 4, borderRadius: '50%', background: SA.borderStrong }} />
                  <span>3 mois · 4 jours/semaine</span>
                  <span style={{ width: 4, height: 4, borderRadius: '50%', background: SA.borderStrong }} />
                  <span>démarré le 15 mai</span>
                </div>
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                <button style={{ background: '#fff', border: `1px solid ${SA.borderStrong}`, padding: '10px 16px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}><SAI name="Chat" size={13} /> Conversation</button>
                <button style={{ background: SA.text, color: '#fff', border: 'none', padding: '10px 18px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Marquer comme livré</button>
              </div>
            </div>

            {/* Stepper */}
            <div style={{ paddingBottom: 24 }}>
              <div style={{ display: 'flex', alignItems: 'flex-start', position: 'relative' }}>
                {stages.map((s, i) => (
                  <div key={i} style={{ flex: 1, position: 'relative', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                    {/* Connector */}
                    {i < stages.length - 1 && (
                      <div style={{ position: 'absolute', top: 14, left: '50%', right: '-50%', height: 2, background: stages[i + 1].done || stages[i + 1].current ? SA.green : SA.border, zIndex: 0 }} />
                    )}
                    {/* Dot */}
                    <div style={{ width: 30, height: 30, borderRadius: '50%', background: s.current ? SA.accent : s.done ? SA.green : '#fff', border: s.current ? `3px solid ${SA.accentSoft}` : s.done ? 'none' : `2px solid ${SA.border}`, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1, position: 'relative', boxShadow: s.current ? '0 0 0 6px rgba(232,93,74,0.15)' : 'none' }}>
                      {s.done && <SAI name="Check" size={15} />}
                      {s.current && <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#fff' }} />}
                    </div>
                    <div style={{ marginTop: 10, textAlign: 'center' }}>
                      <div style={{ fontSize: 13, fontWeight: 600, color: s.current ? SA.accentDeep : s.done ? SA.text : SA.textMute, fontFamily: SA.serif, letterSpacing: '-0.01em' }}>{s.l}</div>
                      <div style={{ fontSize: 11, color: SA.textMute, marginTop: 2, fontStyle: s.current ? 'italic' : 'normal' }}>{s.date}</div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div style={{ padding: '28px 36px', display: 'grid', gridTemplateColumns: '1.6fr 1fr', gap: 24 }}>
            {/* Left col */}
            <div>
              {/* Milestones */}
              <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 16, padding: 28, marginBottom: 20 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: 18 }}>
                  <div>
                    <h2 style={{ fontFamily: SA.serif, fontSize: 22, margin: 0, fontWeight: 500, letterSpacing: '-0.01em' }}>Jalons & livrables</h2>
                    <div style={{ fontSize: 12.5, color: SA.textMute, marginTop: 3 }}>2 livrés sur 4 · prochaine échéance le 30 juin</div>
                  </div>
                  <button style={{ background: 'none', border: 'none', fontSize: 13, color: SA.accent, fontWeight: 600, cursor: 'pointer' }}>+ Ajouter un jalon</button>
                </div>

                {milestones.map((m, i) => {
                  const cfg = {
                    paid:      { dot: SA.green,  bg: SA.greenSoft, label: 'Payé',          icon: 'CheckCircle' },
                    in_review: { dot: SA.amber,  bg: SA.amberSoft, label: 'En validation', icon: 'Clock' },
                    pending:   { dot: SA.textSubtle, bg: SA.bg,    label: 'À venir',       icon: 'Clock' },
                  }[m.status];
                  return (
                    <div key={i} style={{ padding: '14px 0', borderTop: i > 0 ? `1px solid ${SA.border}` : 'none', display: 'grid', gridTemplateColumns: '34px 1fr auto auto', gap: 16, alignItems: 'center' }}>
                      <div style={{ width: 30, height: 30, borderRadius: '50%', background: cfg.bg, color: cfg.dot, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <SAI name={cfg.icon} size={16} />
                      </div>
                      <div>
                        <div style={{ fontSize: 14.5, fontWeight: 600, fontFamily: SA.serif, marginBottom: 2 }}>{m.l}</div>
                        <div style={{ fontSize: 11.5, color: SA.textMute }}>{m.date}</div>
                      </div>
                      <span style={{ fontSize: 11, padding: '3px 10px', background: cfg.bg, color: cfg.dot, borderRadius: 999, fontWeight: 600 }}>{cfg.label}</span>
                      <div style={{ fontFamily: SA.serif, fontSize: 17, fontWeight: 600, minWidth: 90, textAlign: 'right' }}>{m.amount.toLocaleString('fr-FR')} €</div>
                    </div>
                  );
                })}
              </div>

              {/* Frais plateforme */}
              <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 16, padding: 28 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 16 }}>
                  <div style={{ width: 30, height: 30, borderRadius: '50%', background: SA.accentSoft, color: SA.accent, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <SAI name="Sparkle" size={15} />
                  </div>
                  <div>
                    <h3 style={{ fontFamily: SA.serif, fontSize: 18, margin: 0, fontWeight: 600 }}>Frais plateforme estimés</h3>
                    <div style={{ fontSize: 12, color: SA.textMute }}>Calculés selon ta grille tarifaire actuelle.</div>
                  </div>
                </div>

                <div style={{ background: SA.bg, borderRadius: 12, overflow: 'hidden' }}>
                  {[
                    ['Moins de 200 €', '9,00 €', false],
                    ['200 € — 1 000 €', '15,00 €', false],
                    ['Plus de 1 000 €', '25,00 €', true],
                  ].map(([l, v, active], i) => (
                    <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '12px 16px', borderTop: i > 0 ? `1px solid ${SA.border}` : 'none', background: active ? '#fff' : 'transparent', borderLeft: active ? `3px solid ${SA.accent}` : '3px solid transparent' }}>
                      <span style={{ fontSize: 13.5, color: active ? SA.text : SA.textMute, fontWeight: active ? 600 : 500 }}>{l}</span>
                      <span style={{ fontFamily: SA.serif, fontSize: 14.5, fontWeight: active ? 700 : 500, color: active ? SA.accent : SA.text }}>{v}</span>
                    </div>
                  ))}
                </div>

                <div style={{ marginTop: 16, padding: '14px 16px', background: SA.accentSoft, borderRadius: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <div style={{ fontSize: 12, color: SA.accentDeep, fontWeight: 600, marginBottom: 2 }}>Tu encaisseras</div>
                    <div style={{ fontFamily: SA.serif, fontSize: 24, fontWeight: 600, letterSpacing: '-0.02em' }}>12 375,00 €</div>
                    <div style={{ fontSize: 11.5, color: SA.textMute, marginTop: 2 }}>sur 12 400 € — frais 25,00 €</div>
                  </div>
                  <button style={{ background: '#fff', border: `1px solid ${SA.accent}`, color: SA.accentDeep, padding: '8px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}>
                    <SAI name="Sparkle" size={11} /> Passer Premium · 0 € de frais
                  </button>
                </div>
              </div>
            </div>

            {/* Right col */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {/* Paiement */}
              <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 16, padding: 22 }}>
                <div style={{ fontSize: 11, color: SA.textMute, marginBottom: 8, letterSpacing: '0.06em', textTransform: 'uppercase', fontWeight: 600 }}>Montant total</div>
                <div style={{ fontFamily: SA.serif, fontSize: 36, fontWeight: 500, lineHeight: 1, letterSpacing: '-0.025em', marginBottom: 4 }}>12 400 €</div>
                <div style={{ fontSize: 12.5, color: SA.textMute, marginBottom: 16 }}>Paiement par jalon · sous séquestre Stripe</div>

                {/* Progress */}
                <div style={{ background: SA.bg, borderRadius: 10, padding: 14, marginBottom: 12 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11.5, marginBottom: 6 }}>
                    <span style={{ color: SA.green, fontWeight: 600 }}>5 600 € versés</span>
                    <span style={{ color: SA.textMute }}>6 800 € restants</span>
                  </div>
                  <div style={{ height: 6, background: SA.border, borderRadius: 3, overflow: 'hidden', display: 'flex' }}>
                    <div style={{ width: '45%', height: '100%', background: SA.green }} />
                    <div style={{ width: '34%', height: '100%', background: SA.amber }} />
                  </div>
                  <div style={{ display: 'flex', gap: 14, marginTop: 8, fontSize: 10.5, color: SA.textMute }}>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><span style={{ width: 7, height: 7, borderRadius: 2, background: SA.green }} /> Versé</span>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><span style={{ width: 7, height: 7, borderRadius: 2, background: SA.amber }} /> En séquestre</span>
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><span style={{ width: 7, height: 7, borderRadius: 2, background: SA.border }} /> À venir</span>
                  </div>
                </div>

                <button style={{ width: '100%', background: SA.text, color: '#fff', border: 'none', padding: '11px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Voir le détail des paiements</button>
              </div>

              {/* Participants */}
              <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 16, padding: 22 }}>
                <h3 style={{ fontFamily: SA.serif, fontSize: 17, margin: 0, marginBottom: 14, fontWeight: 600 }}>Participants</h3>
                {[
                  { name: 'Nova Studio', sub: 'Client · Toi', pid: 2, role: 'enterprise' },
                  { name: 'Élise Marchand', sub: 'Prestataire', pid: 1, role: 'freelance' },
                ].map((p, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 0', borderTop: i > 0 ? `1px solid ${SA.border}` : 'none' }}>
                    <SAPortrait id={p.pid} size={36} />
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: 13, fontWeight: 600 }}>{p.name}</div>
                      <div style={{ fontSize: 11, color: SA.textMute }}>{p.sub}</div>
                    </div>
                    <span style={{ fontSize: 10, padding: '3px 8px', background: p.role === 'enterprise' ? SA.bg : SA.accentSoft, color: p.role === 'enterprise' ? SA.textMute : SA.accentDeep, borderRadius: 999, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase' }}>{p.role === 'enterprise' ? 'Client' : 'Freelance'}</span>
                  </div>
                ))}
              </div>

              {/* Documents */}
              <div style={{ background: '#fff', border: `1px solid ${SA.border}`, borderRadius: 16, padding: 22 }}>
                <h3 style={{ fontFamily: SA.serif, fontSize: 17, margin: 0, marginBottom: 14, fontWeight: 600 }}>Documents</h3>
                {[
                  { name: 'Brief de mission v2', size: '480 ko', date: '14 mai' },
                  { name: 'Contrat signé', size: '1,2 Mo', date: '14 mai' },
                  { name: 'Wireframes v1', size: '8,4 Mo', date: '5 juin' },
                ].map((d, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 0', borderTop: i > 0 ? `1px solid ${SA.border}` : 'none' }}>
                    <div style={{ width: 30, height: 30, borderRadius: 8, background: SA.bg, color: SA.textMute, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SAI name="Doc" size={14} /></div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: 12.5, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{d.name}</div>
                      <div style={{ fontSize: 10.5, color: SA.textMute }}>{d.size} · {d.date}</div>
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

window.SoleilJobsList = SoleilJobsList;
window.SoleilJobDetailDesc = SoleilJobDetailDesc;
window.SoleilJobDetailCands = SoleilJobDetailCands;
window.SoleilJobCreate = SoleilJobCreate;
window.SoleilProjectDetail = SoleilProjectDetail;
