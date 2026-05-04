// Lot A mobile additions — Création/édition de Job + Wallet refined + Project detail enriched

const SAM2 = window.S;
const SAM2I = window.SI;
const _MF = window.MobileFrame;
const _MH = window.MobileHeader;
const _MBN = window.MobileBottomNav;

// ─── Création de Job — mobile ────────────────────────────────────
function SoleilJobCreateMobile() {
  return (
    <_MF url="atelier.fr/annonce/nouvelle">
      <_MH title="Nouvelle annonce" back />

      <div style={{ flex: 1, overflow: 'auto', background: SAM2.bg }}>
        {/* Hero éditorial */}
        <div style={{ padding: '18px 16px 22px', background: '#fff', borderBottom: `1px solid ${SAM2.border}` }}>
          <h1 style={{ fontFamily: SAM2.serif, fontSize: 26, fontWeight: 400, letterSpacing: '-0.02em', lineHeight: 1.1, margin: 0 }}>
            Publier <span style={{ fontStyle: 'italic', color: SAM2.accent }}>une annonce.</span>
          </h1>
          <p style={{ fontSize: 13, color: SAM2.textMute, margin: '6px 0 0', lineHeight: 1.5 }}>Décris la mission. Plus c'est précis, plus les candidatures sont pertinentes.</p>
        </div>

        {/* Stepper */}
        <div style={{ padding: '12px 16px', background: '#fff', borderBottom: `1px solid ${SAM2.border}`, display: 'flex', gap: 6, alignItems: 'center' }}>
          {['Brief', 'Budget', 'Vidéo', 'Publier'].map((s, i) => (
            <React.Fragment key={s}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                <div style={{ width: 18, height: 18, borderRadius: '50%', background: i === 0 ? SAM2.accent : SAM2.bg, color: i === 0 ? '#fff' : SAM2.textMute, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, fontWeight: 700 }}>{i + 1}</div>
                <span style={{ fontSize: 11, fontWeight: 600, color: i === 0 ? SAM2.text : SAM2.textMute }}>{s}</span>
              </div>
              {i < 3 ? <div style={{ flex: 1, height: 1, background: SAM2.border }} /> : null}
            </React.Fragment>
          ))}
        </div>

        {/* Section : titre */}
        <div style={{ padding: '16px', background: '#fff', borderBottom: `1px solid ${SAM2.border}`, marginTop: 8 }}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', color: SAM2.textMute, marginBottom: 8 }}>Titre de l'annonce</label>
          <input defaultValue="Refonte de l'app produit Nova" style={{ width: '100%', border: `1.5px solid ${SAM2.accent}`, borderRadius: 12, padding: '11px 14px', fontSize: 14, outline: 'none', background: '#fff', fontFamily: SAM2.sans }} />
        </div>

        {/* Section : compétences */}
        <div style={{ padding: '16px', background: '#fff', borderBottom: `1px solid ${SAM2.border}`, marginTop: 8 }}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', color: SAM2.textMute, marginBottom: 6 }}>Compétences attendues</label>
          <div style={{ fontSize: 11.5, color: SAM2.textMute, marginBottom: 10 }}>Tape pour rechercher, ou pioche dans les suggestions.</div>
          <div style={{ border: `1.5px solid ${SAM2.accent}`, borderRadius: 12, padding: '8px 10px', display: 'flex', flexWrap: 'wrap', gap: 5, minHeight: 44, alignItems: 'center' }}>
            {['UX Design', 'Design System', 'Figma', 'SaaS B2B'].map((t, i) => (
              <span key={i} style={{ fontSize: 11.5, padding: '4px 9px', background: SAM2.accentSoft, color: SAM2.accentDeep, borderRadius: 999, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                {t}<span style={{ opacity: 0.6 }}>×</span>
              </span>
            ))}
            <input placeholder="Ajouter…" style={{ border: 'none', outline: 'none', flex: 1, minWidth: 80, fontSize: 13, padding: 2 }} />
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 5, marginTop: 9 }}>
            {['Mobile design', 'UX Research', 'Brand'].map((s, i) => (
              <button key={i} style={{ fontSize: 11, padding: '4px 9px', background: SAM2.bg, border: `1px dashed ${SAM2.borderStrong}`, borderRadius: 999, color: SAM2.textMute, display: 'inline-flex', alignItems: 'center', gap: 3 }}>
                <SAM2I name="Plus" size={10} /> {s}
              </button>
            ))}
          </div>
        </div>

        {/* Section : type de mission */}
        <div style={{ padding: '16px', background: '#fff', borderBottom: `1px solid ${SAM2.border}`, marginTop: 8 }}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', color: SAM2.textMute, marginBottom: 12 }}>Type de mission</label>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {[
              { l: 'Projet ponctuel', d: 'Livraison définie, < 2 mois', icon: 'Pulse' },
              { l: 'Mission longue', d: 'Collaboration > 2 mois', icon: 'Pin', active: true },
              { l: 'Régie temps plein', d: 'Présence quotidienne', icon: 'Briefcase' },
            ].map((o, i) => (
              <div key={i} style={{ background: o.active ? SAM2.accentSoft : '#fff', border: `1.5px solid ${o.active ? SAM2.accent : SAM2.border}`, borderRadius: 12, padding: '12px 14px', display: 'flex', alignItems: 'center', gap: 12 }}>
                <SAM2I name={o.icon} size={18} />
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 13.5, fontWeight: 600, color: o.active ? SAM2.accentDeep : SAM2.text }}>{o.l}</div>
                  <div style={{ fontSize: 11, color: SAM2.textMute, marginTop: 1 }}>{o.d}</div>
                </div>
                <div style={{ width: 18, height: 18, borderRadius: '50%', border: `1.5px solid ${o.active ? SAM2.accent : SAM2.borderStrong}`, background: o.active ? SAM2.accent : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  {o.active ? <div style={{ width: 6, height: 6, borderRadius: '50%', background: '#fff' }} /> : null}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Section : durée + budget */}
        <div style={{ padding: '16px', background: '#fff', borderBottom: `1px solid ${SAM2.border}`, marginTop: 8 }}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', color: SAM2.textMute, marginBottom: 8 }}>Durée estimée</label>
          <div style={{ position: 'relative', marginBottom: 14 }}>
            <select style={{ width: '100%', appearance: 'none', border: `1px solid ${SAM2.borderStrong}`, borderRadius: 12, padding: '11px 14px', fontSize: 13.5, background: '#fff', fontFamily: SAM2.sans, color: SAM2.text }}>
              <option>3 à 6 mois</option>
            </select>
            <span style={{ position: 'absolute', right: 12, top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none' }}><SAM2I name="ChevronDown" size={14} /></span>
          </div>

          <label style={{ display: 'block', fontSize: 12, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', color: SAM2.textMute, marginBottom: 8 }}>Fourchette de budget</label>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <div style={{ flex: 1, position: 'relative' }}>
              <input defaultValue="8 000" style={{ width: '100%', border: `1px solid ${SAM2.borderStrong}`, borderRadius: 12, padding: '11px 28px 11px 12px', fontSize: 13.5, outline: 'none', fontFamily: SAM2.sans }} />
              <span style={{ position: 'absolute', right: 10, top: '50%', transform: 'translateY(-50%)', color: SAM2.textMute, fontSize: 12, fontFamily: SAM2.serif }}>€</span>
            </div>
            <span style={{ color: SAM2.textMute, fontSize: 12 }}>→</span>
            <div style={{ flex: 1, position: 'relative' }}>
              <input defaultValue="12 000" style={{ width: '100%', border: `1px solid ${SAM2.borderStrong}`, borderRadius: 12, padding: '11px 28px 11px 12px', fontSize: 13.5, outline: 'none', fontFamily: SAM2.sans }} />
              <span style={{ position: 'absolute', right: 10, top: '50%', transform: 'translateY(-50%)', color: SAM2.textMute, fontSize: 12, fontFamily: SAM2.serif }}>€</span>
            </div>
          </div>
        </div>

        {/* Section : mode de travail */}
        <div style={{ padding: '16px', background: '#fff', borderBottom: `1px solid ${SAM2.border}`, marginTop: 8 }}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', color: SAM2.textMute, marginBottom: 10 }}>Mode de travail</label>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 6 }}>
            {[
              { l: 'Sur site', icon: 'Building' },
              { l: 'Hybride', icon: 'Layers', active: true },
              { l: '100 % remote', icon: 'Globe' },
            ].map((o, i) => (
              <div key={i} style={{ background: o.active ? SAM2.accentSoft : '#fff', border: `1.5px solid ${o.active ? SAM2.accent : SAM2.border}`, borderRadius: 12, padding: '12px 8px', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6 }}>
                <SAM2I name={o.icon} size={18} />
                <div style={{ fontSize: 12, fontWeight: 600, color: o.active ? SAM2.accentDeep : SAM2.text, textAlign: 'center' }}>{o.l}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Section : vidéo */}
        <div style={{ padding: '16px', background: '#fff', borderBottom: `1px solid ${SAM2.border}`, marginTop: 8 }}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', color: SAM2.textMute, marginBottom: 6 }}>Vidéo · optionnel</label>
          <div style={{ fontSize: 11.5, color: SAM2.textMute, marginBottom: 10, lineHeight: 1.5 }}>30-90 sec. Les annonces avec vidéo reçoivent <strong style={{ color: SAM2.accent }}>3× plus</strong> de candidatures.</div>
          <div style={{ border: `1.5px dashed ${SAM2.borderStrong}`, borderRadius: 12, background: '#fff', padding: 14, display: 'flex', alignItems: 'center', gap: 12 }}>
            <div style={{ width: 56, height: 44, borderRadius: 8, background: 'linear-gradient(135deg, #fbf0dc, #fde6ed)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
              <div style={{ width: 26, height: 26, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <div style={{ width: 0, height: 0, borderTop: '5px solid transparent', borderBottom: '5px solid transparent', borderLeft: `8px solid ${SAM2.accent}`, marginLeft: 2 }} />
              </div>
            </div>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12.5, fontWeight: 600 }}>Ajouter une vidéo</div>
              <div style={{ fontSize: 11, color: SAM2.textMute, fontStyle: 'italic', fontFamily: SAM2.serif, marginTop: 1 }}>Tournée au tél, naturel, c'est parfait.</div>
            </div>
            <SAM2I name="Plus" size={14} />
          </div>
        </div>

        {/* Section : description */}
        <div style={{ padding: '16px', background: '#fff', borderBottom: `1px solid ${SAM2.border}`, marginTop: 8 }}>
          <label style={{ display: 'block', fontSize: 12, fontWeight: 700, letterSpacing: '0.04em', textTransform: 'uppercase', color: SAM2.textMute, marginBottom: 6 }}>Détaille la mission</label>
          <div style={{ fontSize: 11.5, color: SAM2.textMute, marginBottom: 10 }}>Le contexte, les enjeux, l'équipe en place.</div>
          <div style={{ border: `1px solid ${SAM2.borderStrong}`, borderRadius: 12, overflow: 'hidden' }}>
            <div style={{ display: 'flex', gap: 2, padding: '6px 10px', borderBottom: `1px solid ${SAM2.border}`, background: SAM2.bg }}>
              {['B', 'I', 'U', '"', '⏎'].map((b, i) => (
                <span key={i} style={{ width: 26, height: 26, borderRadius: 6, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12, fontWeight: i === 0 ? 700 : 500, fontStyle: i === 1 ? 'italic' : 'normal', color: SAM2.textMute }}>{b}</span>
              ))}
            </div>
            <textarea defaultValue="Nous sommes Nova Studio, une SaaS B2B qui aide les studios créatifs à gérer leurs projets. Refonte complète de l'app produit, design system inclus." style={{ width: '100%', minHeight: 100, border: 'none', outline: 'none', padding: '10px 12px', fontSize: 13, fontFamily: SAM2.sans, lineHeight: 1.55, resize: 'none', color: SAM2.text }} />
          </div>
          <div style={{ fontSize: 11, color: SAM2.textMute, marginTop: 5, textAlign: 'right', fontStyle: 'italic', fontFamily: SAM2.serif }}>~ 280 / 2000</div>
        </div>

        {/* Bottom action bar */}
        <div style={{ padding: '14px 16px 80px', display: 'flex', gap: 8 }}>
          <button style={{ flex: 1, background: '#fff', border: `1px solid ${SAM2.borderStrong}`, padding: '12px', fontSize: 12.5, fontWeight: 600, borderRadius: 999 }}>Brouillon</button>
          <button style={{ flex: 1.4, background: SAM2.accent, color: '#fff', border: 'none', padding: '12px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6, boxShadow: '0 2px 8px rgba(232,93,74,0.25)' }}>
            Publier <SAM2I name="ArrowRight" size={13} />
          </button>
        </div>
      </div>

      <_MBN active="jobs" role="enterprise" />
    </_MF>
  );
}

window.SoleilJobCreateMobile = SoleilJobCreateMobile;
