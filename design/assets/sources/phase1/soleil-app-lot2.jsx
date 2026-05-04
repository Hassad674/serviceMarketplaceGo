// App Lot 2 — Annonces (entreprise) : Mes annonces, Détail annonce + candidatures, Création annonce

const SL2 = window.S;
const SL2I = window.SI;
const SL2Portrait = window.Portrait;
const _AppFrame_L2 = window.AppFrame;
const _AppTabBar_L2 = window.AppTabBar;

// ─── Mes annonces (liste) ──────────────────────────────────────
function AppAnnonces() {
  const jobs = [
    { title: 'Product designer senior', status: 'Ouverte', sc: 'green', cands: 12, new: 3, budget: '600 €', dur: '3 mois', date: 'Postée il y a 4 j', boost: true },
    { title: 'Brand designer pour rebranding', status: 'Ouverte', sc: 'green', cands: 8, new: 1, budget: '550 €', dur: '6 sem', date: 'Postée il y a 1 sem' },
    { title: 'Développeur back-end Node', status: 'En pause', sc: 'amber', cands: 24, new: 0, budget: '700 €', dur: '4 mois', date: 'Postée il y a 2 sem' },
    { title: 'Motion designer (After Effects)', status: 'Brouillon', sc: 'mute', cands: 0, new: 0, budget: '480 €', dur: '3 sem', date: 'Modifiée il y a 1 j' },
    { title: 'UX writer fintech', status: 'Fermée', sc: 'mute', cands: 17, new: 0, budget: '500 €', dur: '2 mois', date: 'Fermée le 15 mai' },
  ];
  const colors = {
    'green': { bg: SL2.greenSoft, text: SL2.green },
    'amber': { bg: '#fbf0dc', text: SL2.amber },
    'mute': { bg: SL2.bg, text: SL2.textMute },
  };
  return (
    <_AppFrame_L2>
      {/* Header */}
      <div style={{ flexShrink: 0, padding: '6px 20px 14px', background: SL2.bg, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <div style={{ fontFamily: SL2.serif, fontSize: 26, fontWeight: 600, letterSpacing: '-0.02em', color: SL2.text }}>Annonces</div>
          <div style={{ fontSize: 12.5, color: SL2.textMute, fontFamily: SL2.serif, fontStyle: 'italic', marginTop: 2 }}>2 ouvertes · 4 candidatures nouvelles</div>
        </div>
        <button style={{ width: 38, height: 38, borderRadius: '50%', background: SL2.text, color: '#fff', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL2I name="Plus" size={18} />
        </button>
      </div>

      {/* Tabs */}
      <div style={{ flexShrink: 0, padding: '0 20px 12px', background: SL2.bg, display: 'flex', gap: 6, overflowX: 'auto' }}>
        {[
          { l: 'Toutes', n: 5, active: true },
          { l: 'Ouvertes', n: 2 },
          { l: 'En pause', n: 1 },
          { l: 'Brouillons', n: 1 },
          { l: 'Fermées', n: 1 },
        ].map(t => (
          <span key={t.l} style={{ padding: '6px 12px', borderRadius: 999, fontSize: 12, fontWeight: 600, whiteSpace: 'nowrap', background: t.active ? SL2.text : '#fff', color: t.active ? '#fff' : SL2.textMute, border: t.active ? 'none' : `1px solid ${SL2.border}` }}>{t.l} <span style={{ opacity: 0.6 }}>{t.n}</span></span>
        ))}
      </div>

      {/* Liste */}
      <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px', display: 'flex', flexDirection: 'column', gap: 10 }}>
        {jobs.map((j, i) => {
          const c = colors[j.sc];
          return (
            <div key={i} style={{ background: '#fff', border: `1px solid ${SL2.border}`, borderRadius: 14, padding: 14 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 10 }}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                    <span style={{ background: c.bg, color: c.text, padding: '3px 8px', borderRadius: 999, fontSize: 10.5, fontWeight: 700 }}>{j.status}</span>
                    {j.boost ? <span style={{ background: SL2.accent, color: '#fff', padding: '3px 8px', borderRadius: 999, fontSize: 10.5, fontWeight: 700, display: 'flex', alignItems: 'center', gap: 3 }}>★ Boost</span> : null}
                  </div>
                  <div style={{ fontFamily: SL2.serif, fontSize: 15.5, fontWeight: 600, color: SL2.text, lineHeight: 1.25 }}>{j.title}</div>
                  <div style={{ fontSize: 11, color: SL2.textSubtle, marginTop: 4, fontStyle: 'italic', fontFamily: SL2.serif }}>{j.date}</div>
                </div>
                <SL2I name="MoreH" size={18} />
              </div>

              <div style={{ display: 'flex', gap: 12, marginTop: 12, paddingTop: 12, borderTop: `1px dashed ${SL2.border}`, fontSize: 11.5, color: SL2.textMute }}>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><SL2I name="Euro" size={12} /><span style={{ fontFamily: SL2.mono, color: SL2.text, fontWeight: 600 }}>{j.budget}/j</span></span>
                <span style={{ width: 2, height: 2, borderRadius: '50%', background: SL2.textSubtle, alignSelf: 'center' }} />
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><SL2I name="Clock" size={12} />{j.dur}</span>
              </div>

              {j.cands > 0 ? (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 10, padding: 10, background: SL2.bg, borderRadius: 10 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <div style={{ display: 'flex' }}>
                      {[0, 1, 2].slice(0, Math.min(3, j.cands)).map(id => (
                        <div key={id} style={{ marginLeft: id === 0 ? 0 : -8, border: '2px solid #fff', borderRadius: '50%' }}>
                          <SL2Portrait id={id} size={22} />
                        </div>
                      ))}
                    </div>
                    <span style={{ fontSize: 12, color: SL2.text, fontWeight: 600 }}>{j.cands} candidatures</span>
                    {j.new > 0 ? <span style={{ background: SL2.accent, color: '#fff', fontSize: 10, fontWeight: 700, padding: '1px 6px', borderRadius: 999 }}>{j.new} nouv.</span> : null}
                  </div>
                  <span style={{ fontSize: 12, color: SL2.accent, fontWeight: 600 }}>Voir →</span>
                </div>
              ) : null}
            </div>
          );
        })}
      </div>

      <_AppTabBar_L2 active="home" />
    </_AppFrame_L2>
  );
}

// ─── Détail annonce + candidatures reçues ─────────────────────
function AppAnnonceDetail() {
  return (
    <_AppFrame_L2 bg="#fff">
      {/* Header */}
      <div style={{ flexShrink: 0, padding: '6px 14px 12px', background: '#fff', borderBottom: `1px solid ${SL2.border}`, display: 'flex', alignItems: 'center', gap: 10 }}>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SL2.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL2I name="ArrowLeft" size={18} />
        </button>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 11, color: SL2.textMute }}>Annonce · ouverte</div>
          <div style={{ fontSize: 14, fontWeight: 600, color: SL2.text, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>Product designer senior</div>
        </div>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SL2.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL2I name="Edit" size={16} />
        </button>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SL2.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL2I name="MoreH" size={17} />
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        {/* Stats */}
        <div style={{ padding: '16px 20px 14px', background: SL2.bg, display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8 }}>
          {[
            { l: 'CANDID.', v: '12', sub: '3 nouvelles', accent: true },
            { l: 'VUES', v: '184', sub: '+24 cette sem.' },
            { l: 'TAUX RÉP.', v: '92 %', sub: 'sous 12 h' },
          ].map((s, i) => (
            <div key={i} style={{ background: '#fff', border: `1px solid ${SL2.border}`, borderRadius: 12, padding: '10px 9px' }}>
              <div style={{ fontSize: 9.5, color: SL2.textSubtle, letterSpacing: '0.06em', fontWeight: 600 }}>{s.l}</div>
              <div style={{ fontFamily: SL2.serif, fontSize: 19, fontWeight: 600, color: s.accent ? SL2.accent : SL2.text, marginTop: 3 }}>{s.v}</div>
              <div style={{ fontSize: 10, color: SL2.textMute, marginTop: 1 }}>{s.sub}</div>
            </div>
          ))}
        </div>

        {/* Tabs */}
        <div style={{ padding: '0 20px', display: 'flex', gap: 18, borderBottom: `1px solid ${SL2.border}`, background: '#fff' }}>
          {[
            { l: 'Candidatures', n: 12, active: true },
            { l: 'Description' },
            { l: 'Stats' },
          ].map((t, i) => (
            <div key={t.l} style={{ padding: '10px 0', fontSize: 13, fontWeight: 600, color: t.active ? SL2.text : SL2.textMute, borderBottom: t.active ? `2px solid ${SL2.accent}` : '2px solid transparent', marginBottom: -1 }}>
              {t.l}{t.n ? <span style={{ marginLeft: 5, fontSize: 11, color: SL2.textSubtle, fontWeight: 500 }}>{t.n}</span> : null}
            </div>
          ))}
        </div>

        {/* Filtres rapides */}
        <div style={{ padding: '12px 20px 6px', background: '#fff', display: 'flex', gap: 6, overflowX: 'auto' }}>
          {['Toutes', 'Nouvelles', 'À examiner', 'Présélection', 'Refusées'].map((f, i) => (
            <span key={f} style={{ padding: '5px 10px', borderRadius: 999, fontSize: 11, fontWeight: 600, whiteSpace: 'nowrap', background: i === 0 ? SL2.text : SL2.bg, color: i === 0 ? '#fff' : SL2.textMute }}>{f}</span>
          ))}
        </div>

        {/* Liste candidatures */}
        <div style={{ padding: '8px 20px 20px', display: 'flex', flexDirection: 'column', gap: 10 }}>
          {[
            { id: 4, name: 'Sofia Lambert', title: 'UX/UI · Toulouse', tjm: '520 €', match: 96, new: true, msg: 'J\'aime particulièrement votre approche produit. Je serais ravie de…' },
            { id: 3, name: 'Théo Martinet', title: 'Motion + Product · Nantes', tjm: '480 €', match: 88, new: true, msg: 'Bonjour, je vois que vous cherchez un profil senior. J\'ai 8 ans…' },
            { id: 0, name: 'Camille Dubois', title: 'Product Designer · Bordeaux', tjm: '600 €', match: 94, msg: 'Bonjour Léa, votre brief résonne fort avec mes 7 ans en B2B SaaS…', short: true },
            { id: 5, name: 'Marc Olivier', title: 'Designer senior · Paris', tjm: '650 €', match: 76 },
          ].map((c, i) => (
            <div key={i} style={{ background: '#fff', border: `1px solid ${c.short ? SL2.green : SL2.border}`, borderRadius: 14, padding: 12, position: 'relative' }}>
              {c.short ? <div style={{ position: 'absolute', top: -8, right: 14, background: SL2.green, color: '#fff', fontSize: 10, fontWeight: 700, padding: '2px 8px', borderRadius: 999 }}>★ Préselection</div> : null}
              <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
                <div style={{ position: 'relative', flexShrink: 0 }}>
                  <SL2Portrait id={c.id} size={44} />
                  {c.new ? <div style={{ position: 'absolute', top: -2, right: -2, width: 12, height: 12, borderRadius: '50%', background: SL2.accent, border: '2px solid #fff' }} /> : null}
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <div style={{ minWidth: 0, flex: 1 }}>
                      <div style={{ fontSize: 13.5, fontWeight: 600, color: SL2.text }}>{c.name}</div>
                      <div style={{ fontSize: 11.5, color: SL2.textMute, marginTop: 1 }}>{c.title}</div>
                    </div>
                    <div style={{ textAlign: 'right', flexShrink: 0 }}>
                      <span style={{ background: c.match >= 90 ? SL2.greenSoft : SL2.bg, color: c.match >= 90 ? SL2.green : SL2.textMute, padding: '2px 7px', borderRadius: 999, fontSize: 10.5, fontWeight: 700 }}>{c.match} %</span>
                      <div style={{ fontFamily: SL2.mono, fontSize: 11.5, color: SL2.text, marginTop: 4, fontWeight: 600 }}>{c.tjm}/j</div>
                    </div>
                  </div>
                  {c.msg ? <p style={{ fontSize: 12, color: SL2.textMute, lineHeight: 1.45, margin: '8px 0 0', fontFamily: SL2.serif, fontStyle: 'italic' }}>« {c.msg} »</p> : null}
                </div>
              </div>

              <div style={{ display: 'flex', gap: 6, marginTop: 12, paddingTop: 11, borderTop: `1px dashed ${SL2.border}` }}>
                <button style={{ flex: 1, padding: '7px 10px', background: SL2.bg, color: SL2.text, border: 'none', borderRadius: 9, fontSize: 12, fontWeight: 600, fontFamily: SL2.sans }}>Profil</button>
                <button style={{ flex: 1, padding: '7px 10px', background: SL2.bg, color: SL2.text, border: 'none', borderRadius: 9, fontSize: 12, fontWeight: 600, fontFamily: SL2.sans }}>Message</button>
                <button style={{ flex: 1, padding: '7px 10px', background: SL2.text, color: '#fff', border: 'none', borderRadius: 9, fontSize: 12, fontWeight: 600, fontFamily: SL2.sans }}>Recruter</button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </_AppFrame_L2>
  );
}

// ─── Création annonce (formulaire en plusieurs étapes) ────────
function AppAnnonceCreation() {
  return (
    <_AppFrame_L2 bg="#fff">
      <div style={{ flexShrink: 0, padding: '6px 14px 10px', background: '#fff', borderBottom: `1px solid ${SL2.border}`, display: 'flex', alignItems: 'center', gap: 10 }}>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SL2.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL2I name="X" size={17} />
        </button>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: SL2.text }}>Nouvelle annonce</div>
          <div style={{ fontSize: 11, color: SL2.textMute, marginTop: 1 }}>Étape 2 sur 4 · Détails</div>
        </div>
        <button style={{ padding: '6px 12px', background: 'transparent', border: 'none', color: SL2.textMute, fontSize: 12.5, fontWeight: 600 }}>Brouillon</button>
      </div>

      {/* Stepper progression */}
      <div style={{ flexShrink: 0, padding: '12px 20px 0', background: '#fff' }}>
        <div style={{ display: 'flex', gap: 4 }}>
          {[1, 2, 3, 4].map(i => (
            <div key={i} style={{ flex: 1, height: 3, borderRadius: 2, background: i <= 2 ? SL2.accent : SL2.border }} />
          ))}
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '20px 20px 20px' }}>
        <div style={{ fontFamily: SL2.serif, fontSize: 22, fontWeight: 600, letterSpacing: '-0.015em', color: SL2.text, lineHeight: 1.2 }}>Décrivez la mission</div>
        <div style={{ fontFamily: SL2.serif, fontSize: 13, fontStyle: 'italic', color: SL2.textMute, marginTop: 4 }}>Soyez clair sur le besoin et les compétences attendues — les bons profils répondent en 24 h.</div>

        {/* Champ titre */}
        <div style={{ marginTop: 20 }}>
          <label style={{ fontSize: 11.5, fontWeight: 600, color: SL2.text, display: 'block', marginBottom: 7 }}>Titre de l'annonce</label>
          <div style={{ background: '#fff', border: `1.5px solid ${SL2.accent}`, borderRadius: 12, padding: '12px 14px' }}>
            <div style={{ fontSize: 14, color: SL2.text, fontWeight: 500 }}>Product designer senior</div>
          </div>
          <div style={{ fontSize: 10.5, color: SL2.textSubtle, marginTop: 5, fontStyle: 'italic', fontFamily: SL2.serif }}>23 / 80 caractères</div>
        </div>

        {/* Champ description */}
        <div style={{ marginTop: 18 }}>
          <label style={{ fontSize: 11.5, fontWeight: 600, color: SL2.text, display: 'block', marginBottom: 7 }}>Description du besoin</label>
          <div style={{ background: '#fff', border: `1px solid ${SL2.border}`, borderRadius: 12, padding: '12px 14px', minHeight: 100 }}>
            <div style={{ fontSize: 13, color: SL2.text, lineHeight: 1.5 }}>
              Nous cherchons un product designer senior pour accompagner la refonte de notre app mobile sur 3 mois. Stack Flutter, design system à poser…
            </div>
          </div>
        </div>

        {/* Compétences */}
        <div style={{ marginTop: 18 }}>
          <label style={{ fontSize: 11.5, fontWeight: 600, color: SL2.text, display: 'block', marginBottom: 7 }}>Compétences requises</label>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
            {['Figma', 'Design system', 'Mobile B2C'].map(t => (
              <span key={t} style={{ padding: '6px 11px', background: SL2.accentSoft, color: SL2.accentDeep, borderRadius: 999, fontSize: 12, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 5 }}>
                {t}
                <SL2I name="X" size={11} />
              </span>
            ))}
            <span style={{ padding: '6px 11px', background: SL2.bg, color: SL2.textMute, border: `1px dashed ${SL2.borderStrong}`, borderRadius: 999, fontSize: 12, fontWeight: 500, display: 'flex', alignItems: 'center', gap: 4 }}>
              <SL2I name="Plus" size={11} />Ajouter
            </span>
          </div>
        </div>

        {/* Budget + durée */}
        <div style={{ marginTop: 18, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
          <div>
            <label style={{ fontSize: 11.5, fontWeight: 600, color: SL2.text, display: 'block', marginBottom: 7 }}>TJM cible</label>
            <div style={{ background: '#fff', border: `1px solid ${SL2.border}`, borderRadius: 12, padding: '12px 14px', display: 'flex', alignItems: 'baseline', gap: 4 }}>
              <span style={{ fontFamily: SL2.mono, fontSize: 15, fontWeight: 600, color: SL2.text }}>600</span>
              <span style={{ fontSize: 12, color: SL2.textMute }}>€</span>
            </div>
          </div>
          <div>
            <label style={{ fontSize: 11.5, fontWeight: 600, color: SL2.text, display: 'block', marginBottom: 7 }}>Durée</label>
            <div style={{ background: '#fff', border: `1px solid ${SL2.border}`, borderRadius: 12, padding: '12px 14px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span style={{ fontSize: 13, fontWeight: 600, color: SL2.text }}>3 mois</span>
              <SL2I name="ChevronDown" size={14} />
            </div>
          </div>
        </div>

        {/* Toggle remote */}
        <div style={{ marginTop: 14, background: SL2.bg, borderRadius: 12, padding: '12px 14px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontSize: 13, fontWeight: 600, color: SL2.text }}>Mission 100 % à distance</div>
            <div style={{ fontSize: 11, color: SL2.textMute, marginTop: 1 }}>Sinon, nous vous demanderons la ville</div>
          </div>
          <div style={{ width: 38, height: 22, borderRadius: 999, background: SL2.accent, position: 'relative', flexShrink: 0 }}>
            <div style={{ position: 'absolute', top: 2, right: 2, width: 18, height: 18, borderRadius: '50%', background: '#fff', boxShadow: '0 1px 2px rgba(0,0,0,0.2)' }} />
          </div>
        </div>
      </div>

      {/* CTA bas */}
      <div style={{ flexShrink: 0, padding: '12px 20px 28px', background: '#fff', borderTop: `1px solid ${SL2.border}`, display: 'flex', gap: 10 }}>
        <button style={{ width: 50, height: 50, borderRadius: 14, background: SL2.bg, border: `1px solid ${SL2.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: SL2.text }}>
          <SL2I name="ArrowLeft" size={18} />
        </button>
        <button style={{ flex: 1, height: 50, borderRadius: 14, background: SL2.accent, color: '#fff', border: 'none', fontSize: 14.5, fontWeight: 600, fontFamily: SL2.sans }}>
          Continuer →
        </button>
      </div>
    </_AppFrame_L2>
  );
}

Object.assign(window, { AppAnnonces, AppAnnonceDetail, AppAnnonceCreation });
