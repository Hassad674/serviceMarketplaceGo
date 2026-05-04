// Lot A mobile — Annonces, Détail (desc + candidatures), Détail projet
const SAM = window.S;
const SAMI = window.SI;
const SAMPortrait = window.Portrait;
const { MobileFrame, MobileHeader, MobileBottomNav, MobileSegmented } = window;

// ─── AM1 — Mes annonces (entreprise) ──────────────────────────
function SoleilJobsListMobile() {
  const [tab, setTab] = React.useState(0);
  const jobs = [
    { title: 'Refonte de l\'app produit Nova', loc: 'Paris · Hybride', tjm: '700 €/j', cands: 12, status: 'Publiée', kind: 'open', dur: '4 mois' },
    { title: 'UX Research onboarding', loc: 'Remote', tjm: '600 €/j', cands: 7, status: 'Publiée', kind: 'open', dur: '6 sem' },
    { title: 'Design system v2', loc: 'Paris', tjm: '750 €/j', cands: 24, status: 'En sélection', kind: 'pending', dur: '4-6 mois' },
    { title: 'Brand identity Naveo', loc: 'Hybride', tjm: '650 €/j', cands: 0, status: 'Brouillon', kind: 'draft', dur: '3 mois' },
  ];
  const colors = { open: SAM.green, pending: SAM.amber, draft: SAM.textSubtle };
  const bgs = { open: SAM.greenSoft, pending: SAM.amberSoft, draft: SAM.bg };
  return (
    <MobileFrame url="atelier.fr/jobs">
      <MobileHeader title="Mes annonces" action={<button style={{ background: SAM.text, color: '#fff', border: 'none', padding: '7px 14px', fontSize: 11.5, fontWeight: 600, borderRadius: 999, display: 'flex', alignItems: 'center', gap: 5 }}><SAMI name="Plus" size={11} /> Nouvelle</button>} />
      <div style={{ padding: '12px 14px', flexShrink: 0, background: '#fff', borderBottom: `1px solid ${SAM.border}` }}>
        <MobileSegmented items={['Toutes · 4', 'Publiées · 2', 'En sélection · 1', 'Brouillons']} active={tab} />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '12px 14px 90px' }}>
        {jobs.map((j, i) => (
          <div key={i} style={{ background: '#fff', border: `1px solid ${SAM.border}`, borderRadius: 14, padding: 14, marginBottom: 10 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 6, gap: 8 }}>
              <div style={{ fontSize: 14.5, fontWeight: 600, fontFamily: SAM.serif, lineHeight: 1.3, flex: 1 }}>{j.title}</div>
              <span style={{ fontSize: 10.5, padding: '3px 8px', background: bgs[j.kind], color: colors[j.kind], borderRadius: 999, fontWeight: 700, flexShrink: 0 }}>{j.status}</span>
            </div>
            <div style={{ display: 'flex', gap: 10, fontSize: 11.5, color: SAM.textMute, flexWrap: 'wrap', marginBottom: 10 }}>
              <span><strong style={{ color: SAM.text }}>{j.tjm}</strong></span>
              <span>· {j.loc}</span>
              <span>· {j.dur}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingTop: 10, borderTop: `1px solid ${SAM.border}` }}>
              <div style={{ fontSize: 11.5, color: SAM.text, display: 'flex', alignItems: 'center', gap: 5 }}><SAMI name="User" size={12} /> <strong>{j.cands}</strong> candidatures</div>
              <button style={{ background: SAM.bg, border: 'none', padding: '6px 14px', fontSize: 11.5, fontWeight: 600, borderRadius: 999 }}>Gérer →</button>
            </div>
          </div>
        ))}
      </div>
      <MobileBottomNav active="jobs" role="enterprise" />
    </MobileFrame>
  );
}

// ─── AM2 — Détail annonce · Description ───────────────────────
function SoleilJobDetailDescMobile() {
  const [tab, setTab] = React.useState(0);
  return (
    <MobileFrame url="atelier.fr/job/nova">
      <MobileHeader title="Refonte app Nova" subtitle="Publiée · 12 candidatures" back action={<button style={{ width: 36, height: 36, borderRadius: '50%', background: SAM.bg, border: 'none' }}><SAMI name="MoreHorizontal" size={14} /></button>} />
      <div style={{ padding: '12px 14px', flexShrink: 0, background: '#fff', borderBottom: `1px solid ${SAM.border}` }}>
        <MobileSegmented items={['Description', 'Candidatures · 12']} active={0} />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '14px 14px 100px' }}>
        {/* Méta */}
        <div style={{ background: '#fff', border: `1px solid ${SAM.border}`, borderRadius: 12, padding: 14, marginBottom: 12 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <div><div style={{ fontSize: 10.5, color: SAM.textMute, marginBottom: 3, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600 }}>TJM</div><div style={{ fontFamily: SAM.serif, fontSize: 17, fontWeight: 500 }}>700 €/j</div></div>
            <div><div style={{ fontSize: 10.5, color: SAM.textMute, marginBottom: 3, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600 }}>Durée</div><div style={{ fontFamily: SAM.serif, fontSize: 17, fontWeight: 500 }}>3-4 mois</div></div>
            <div><div style={{ fontSize: 10.5, color: SAM.textMute, marginBottom: 3, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600 }}>Lieu</div><div style={{ fontFamily: SAM.serif, fontSize: 14, fontWeight: 500 }}>Paris · Hybride</div></div>
            <div><div style={{ fontSize: 10.5, color: SAM.textMute, marginBottom: 3, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600 }}>Démarrage</div><div style={{ fontFamily: SAM.serif, fontSize: 14, fontWeight: 500 }}>Juin 2026</div></div>
          </div>
        </div>

        {/* Description */}
        <div style={{ background: '#fff', border: `1px solid ${SAM.border}`, borderRadius: 12, padding: 16, marginBottom: 12 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 8 }}>La mission</div>
          <p style={{ fontSize: 13, lineHeight: 1.6, margin: 0, marginBottom: 10 }}>Nova cherche un·e UX/UI Designer senior pour piloter la refonte complète de son app SaaS. Mission longue durée, 4j/sem.</p>
          <div style={{ fontSize: 12.5, fontWeight: 700, marginBottom: 8 }}>Objectifs</div>
          <ul style={{ fontSize: 12.5, lineHeight: 1.7, paddingLeft: 18, margin: 0, color: SAM.text }}>
            <li>Auditer l'existant et cadrer la roadmap design</li>
            <li>Reposer la design system avec les devs</li>
            <li>Livrer wireframes, UI haute fidélité, handoff</li>
          </ul>
        </div>

        {/* Compétences */}
        <div style={{ background: '#fff', border: `1px solid ${SAM.border}`, borderRadius: 12, padding: 16, marginBottom: 12 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 10 }}>Compétences attendues</div>
          <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap' }}>
            {['UX/UI', 'Design System', 'Figma', 'SaaS B2B', 'Discovery'].map(t => (
              <span key={t} style={{ fontSize: 11.5, padding: '5px 10px', background: SAM.accentSoft, color: SAM.accentDeep, borderRadius: 999, fontWeight: 600 }}>{t}</span>
            ))}
          </div>
        </div>

        {/* Stats */}
        <div style={{ background: SAM.bg, borderRadius: 12, padding: 14, fontSize: 12, color: SAM.textMute, lineHeight: 1.6 }}>
          <strong style={{ color: SAM.text }}>234 vues</strong> · <strong style={{ color: SAM.text }}>12 candidatures</strong> · publiée il y a 5 jours
        </div>
      </div>
      <div style={{ position: 'absolute', left: 0, right: 0, bottom: 0, padding: '12px 14px', background: '#fff', borderTop: `1px solid ${SAM.border}`, display: 'flex', gap: 8 }}>
        <button style={{ flex: 1, background: '#fff', border: `1px solid ${SAM.borderStrong}`, padding: '12px', fontSize: 13, fontWeight: 600, borderRadius: 999 }}>Modifier</button>
        <button style={{ flex: 1, background: SAM.text, color: '#fff', border: 'none', padding: '12px', fontSize: 13, fontWeight: 600, borderRadius: 999 }}>Voir candidatures</button>
      </div>
    </MobileFrame>
  );
}

// ─── AM3 — Détail annonce · Candidatures ──────────────────────
function SoleilJobDetailCandsMobile() {
  const cands = [
    { name: 'Élise Marchand', title: 'UX Designer · Brand', tjm: '650 €', match: 96, pid: 1, verified: true, status: 'Sélectionnée', kind: 'shortlist' },
    { name: 'Thomas Reyer', title: 'Senior Product Designer', tjm: '720 €', match: 91, pid: 2, verified: true, status: 'À examiner', kind: 'new' },
    { name: 'Camille Lopez', title: 'UX/UI · 6 ans', tjm: '580 €', match: 88, pid: 3, status: 'À examiner', kind: 'new' },
    { name: 'Hugo Bensoussan', title: 'Designer Product', tjm: '600 €', match: 82, pid: 4, status: 'Non retenue', kind: 'rejected' },
  ];
  const colors = { shortlist: SAM.green, new: SAM.accent, rejected: SAM.textSubtle };
  const bgs = { shortlist: SAM.greenSoft, new: SAM.accentSoft, rejected: SAM.bg };
  return (
    <MobileFrame url="atelier.fr/job/nova/candidatures">
      <MobileHeader title="Refonte app Nova" subtitle="12 candidatures" back />
      <div style={{ padding: '12px 14px', flexShrink: 0, background: '#fff', borderBottom: `1px solid ${SAM.border}` }}>
        <MobileSegmented items={['Description', 'Candidatures · 12']} active={1} />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '12px 14px 90px' }}>
        {cands.map((c, i) => (
          <div key={i} style={{ background: '#fff', border: `1px solid ${SAM.border}`, borderRadius: 14, padding: 14, marginBottom: 10 }}>
            <div style={{ display: 'flex', gap: 12, marginBottom: 10 }}>
              <SAMPortrait id={c.pid} size={44} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 5, marginBottom: 2 }}>
                  <div style={{ fontSize: 13.5, fontWeight: 600 }}>{c.name}</div>
                  {c.verified ? <SAMI name="Verified" size={12} /> : null}
                </div>
                <div style={{ fontSize: 11.5, color: SAM.textMute, fontFamily: SAM.serif, fontStyle: 'italic' }}>{c.title}</div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{ fontFamily: SAM.serif, fontSize: 14, fontWeight: 600 }}>{c.tjm}</div>
                <div style={{ fontSize: 10.5, color: SAM.textMute }}>/ jour</div>
              </div>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingTop: 10, borderTop: `1px solid ${SAM.border}` }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ fontSize: 10.5, color: SAM.green, fontWeight: 700, padding: '2px 7px', background: SAM.greenSoft, borderRadius: 999 }}>{c.match}% match</span>
                <span style={{ fontSize: 10.5, padding: '2px 8px', background: bgs[c.kind], color: colors[c.kind], borderRadius: 999, fontWeight: 700 }}>{c.status}</span>
              </div>
              <button style={{ background: SAM.bg, border: 'none', padding: '6px 12px', fontSize: 11.5, fontWeight: 600, borderRadius: 999 }}>Voir →</button>
            </div>
          </div>
        ))}
      </div>
    </MobileFrame>
  );
}

// ─── AM4 — Détail projet (stepper + frais) ────────────────────
function SoleilProjectDetailMobile() {
  return (
    <MobileFrame url="atelier.fr/projet/nova">
      <MobileHeader title="Nova · Refonte SaaS" subtitle="Avec Élise Marchand" back />
      <div style={{ flex: 1, overflow: 'auto', padding: '14px 14px 24px' }}>
        {/* Statut */}
        <div style={{ background: '#fff', border: `1px solid ${SAM.border}`, borderRadius: 12, padding: 14, marginBottom: 12, display: 'flex', alignItems: 'center', gap: 10 }}>
          <SAMPortrait id={1} size={44} />
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 13.5, fontWeight: 600, marginBottom: 1 }}>Élise Marchand</div>
            <div style={{ fontSize: 11.5, color: SAM.textMute }}>Démarré il y a 3 semaines</div>
          </div>
          <button style={{ background: SAM.bg, border: 'none', padding: '8px 14px', fontSize: 11.5, fontWeight: 600, borderRadius: 999 }}>Message</button>
        </div>

        {/* Stepper jalons */}
        <div style={{ background: '#fff', border: `1px solid ${SAM.border}`, borderRadius: 12, padding: 14, marginBottom: 12 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 12 }}>Jalons</div>
          {[
            { l: 'Discovery & entretiens', s: 'done', amount: '1 200 €' },
            { l: 'Wireframes', s: 'review', amount: '1 800 €', due: 'Livré · à valider' },
            { l: 'UI haute fidélité', s: 'todo', amount: '2 400 €' },
            { l: 'Handoff dev', s: 'todo', amount: '1 200 €' },
          ].map((j, i, arr) => (
            <div key={i} style={{ display: 'flex', gap: 12, paddingBottom: i === arr.length - 1 ? 0 : 14 }}>
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', flexShrink: 0 }}>
                <div style={{ width: 24, height: 24, borderRadius: '50%', background: j.s === 'done' ? SAM.green : j.s === 'review' ? SAM.accent : SAM.bg, color: j.s === 'todo' ? SAM.textMute : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, fontWeight: 700 }}>
                  {j.s === 'done' ? '✓' : i + 1}
                </div>
                {i < arr.length - 1 ? <div style={{ width: 2, flex: 1, background: j.s === 'done' ? SAM.green : SAM.border, marginTop: 2 }} /> : null}
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 8 }}>
                  <div style={{ fontSize: 13.5, fontWeight: j.s === 'review' ? 700 : 600, color: j.s === 'todo' ? SAM.textMute : SAM.text }}>{j.l}</div>
                  <div style={{ fontSize: 12, fontWeight: 700, fontFamily: SAM.serif, color: j.s === 'todo' ? SAM.textSubtle : SAM.text }}>{j.amount}</div>
                </div>
                {j.due ? <div style={{ fontSize: 11, color: SAM.accent, marginTop: 2, fontWeight: 600 }}>{j.due}</div> : null}
              </div>
            </div>
          ))}
        </div>

        {/* Action valider */}
        <div style={{ background: SAM.accentSoft, border: `1.5px solid ${SAM.accent}`, borderRadius: 12, padding: 14, marginBottom: 12 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 4, color: SAM.accentDeep }}>Wireframes livrés</div>
          <div style={{ fontSize: 12, color: SAM.text, marginBottom: 12, lineHeight: 1.55 }}>Élise a livré le jalon ce matin. Examine les fichiers et débloque le paiement.</div>
          <div style={{ display: 'flex', gap: 6 }}>
            <button style={{ flex: 1, background: '#fff', border: `1px solid ${SAM.borderStrong}`, padding: '10px', fontSize: 12, fontWeight: 600, borderRadius: 999 }}>Demander une révision</button>
            <button style={{ flex: 1, background: SAM.text, color: '#fff', border: 'none', padding: '10px', fontSize: 12, fontWeight: 600, borderRadius: 999 }}>Valider · 1 800 €</button>
          </div>
        </div>

        {/* Récap financier */}
        <div style={{ background: '#fff', border: `1px solid ${SAM.border}`, borderRadius: 12, padding: 14 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 10 }}>Récap financier</div>
          {[['Total mission', '6 600 €'], ['Déjà versé', '1 200 €'], ['En séquestre', '1 800 €'], ['Frais Atelier (5 %)', '330 €']].map(([l, v], i, arr) => (
            <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', fontSize: 12.5, borderBottom: i < arr.length - 1 ? `1px solid ${SAM.border}` : 'none' }}>
              <span style={{ color: SAM.textMute }}>{l}</span>
              <span style={{ fontWeight: 600, fontFamily: SAM.serif }}>{v}</span>
            </div>
          ))}
        </div>
      </div>
    </MobileFrame>
  );
}

window.SoleilJobsListMobile = SoleilJobsListMobile;
window.SoleilJobDetailDescMobile = SoleilJobDetailDescMobile;
window.SoleilJobDetailCandsMobile = SoleilJobDetailCandsMobile;
window.SoleilProjectDetailMobile = SoleilProjectDetailMobile;
