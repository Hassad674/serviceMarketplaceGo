// Lot D mobile — Profil prestataire (public + privé)
const SDM = window.S;
const SDMI = window.SI;
const SDMPortrait = window.Portrait;
const { MobileFrame, MobileHeader, MobileBottomNav, MobileSegmented } = window;

function MProfileSection({ title, subtitle, isPrivate, children }) {
  return (
    <div style={{ background: '#fff', border: `1px solid ${SDM.border}`, borderRadius: 14, padding: 16, marginBottom: 10 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 10, gap: 10 }}>
        <div>
          <h2 style={{ fontFamily: SDM.serif, fontSize: 15, margin: 0, fontWeight: 700, letterSpacing: '-0.005em' }}>{title}</h2>
          {subtitle ? <div style={{ fontSize: 11, color: SDM.textMute, marginTop: 2, fontStyle: 'italic', fontFamily: SDM.serif }}>{subtitle}</div> : null}
        </div>
        {isPrivate ? <button style={{ background: 'transparent', border: `1px solid ${SDM.border}`, width: 28, height: 28, borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SDM.textMute, flexShrink: 0 }}><SDMI name="Edit" size={12} /></button> : null}
      </div>
      {children}
    </div>
  );
}

function SoleilProfileMobile({ isPrivate = false }) {
  return (
    <MobileFrame url={isPrivate ? "atelier.fr/profil" : "atelier.fr/elise-marchand"}>
      <MobileHeader title={isPrivate ? "Mon profil" : "Profil"} back={!isPrivate} action={isPrivate ? <button style={{ background: SDM.bg, border: 'none', padding: '7px 12px', fontSize: 11.5, fontWeight: 600, borderRadius: 999, display: 'flex', alignItems: 'center', gap: 5 }}><SDMI name="Eye" size={11} /> Aperçu</button> : <button style={{ width: 36, height: 36, borderRadius: '50%', background: SDM.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SDMI name="Bookmark" size={14} /></button>} />

      <div style={{ flex: 1, overflow: 'auto', padding: '14px 14px 100px' }}>
        {/* Header card */}
        <div style={{ background: '#fff', border: `1px solid ${SDM.border}`, borderRadius: 14, padding: 18, marginBottom: 10, textAlign: 'center', position: 'relative' }}>
          {isPrivate ? <button style={{ position: 'absolute', top: 12, right: 12, background: 'transparent', border: `1px solid ${SDM.border}`, width: 28, height: 28, borderRadius: '50%', color: SDM.textMute }}><SDMI name="Edit" size={12} /></button> : null}
          <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 12, position: 'relative' }}>
            <div style={{ position: 'relative' }}>
              <SDMPortrait id={1} size={88} rounded={16} />
              {!isPrivate ? <div style={{ position: 'absolute', bottom: -2, right: -2, width: 22, height: 22, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 2px 6px rgba(0,0,0,0.15)' }}><SDMI name="Verified" size={13} /></div> : null}
            </div>
          </div>
          <h1 style={{ fontFamily: SDM.serif, fontSize: 22, margin: 0, fontWeight: 500, letterSpacing: '-0.02em' }}>Élise Marchand</h1>
          <div style={{ fontSize: 13, color: SDM.textMute, fontFamily: SDM.serif, fontStyle: 'italic', marginTop: 3, marginBottom: 10 }}>UX Designer & Brand pour startups B2B</div>
          <div style={{ display: 'inline-flex', alignItems: 'center', gap: 5, padding: '5px 12px', background: SDM.greenSoft, color: SDM.green, borderRadius: 999, fontSize: 11.5, fontWeight: 600, marginBottom: 14 }}><span style={{ width: 6, height: 6, borderRadius: '50%', background: SDM.green }} /> Disponible dès lundi</div>
          <div style={{ display: 'flex', justifyContent: 'center', gap: 14, fontSize: 11.5, color: SDM.textMute, flexWrap: 'wrap', marginBottom: 12 }}>
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><SDMI name="MapPin" size={12} /> Paris</span>
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><SDMI name="Star" size={12} /> <strong style={{ color: SDM.text }}>4,9</strong> · 47</span>
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><SDMI name="Clock" size={12} /> Répond en 2 h</span>
          </div>
          <div style={{ paddingTop: 12, borderTop: `1px solid ${SDM.border}` }}>
            <div style={{ fontSize: 10.5, color: SDM.textMute, letterSpacing: '0.06em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 2 }}>À partir de</div>
            <div style={{ fontFamily: SDM.serif, fontSize: 26, fontWeight: 500, letterSpacing: '-0.02em' }}>650 €<span style={{ fontSize: 13, color: SDM.textMute, fontWeight: 400 }}>/jour</span></div>
          </div>
        </div>

        {/* Complétion (privé seulement) */}
        {isPrivate ? (
          <div style={{ background: SDM.accentSoft, border: `1px solid ${SDM.accent}`, borderRadius: 12, padding: 14, marginBottom: 10 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6, fontSize: 12.5 }}>
              <span style={{ fontWeight: 600 }}>Profil complété à 78 %</span>
              <span style={{ color: SDM.accentDeep, fontWeight: 700 }}>4 sections</span>
            </div>
            <div style={{ height: 5, background: 'rgba(196,58,38,0.15)', borderRadius: 999, overflow: 'hidden' }}>
              <div style={{ width: '78%', height: '100%', background: SDM.accent, borderRadius: 999 }} />
            </div>
          </div>
        ) : null}

        {/* À propos */}
        <MProfileSection title="À propos" isPrivate={isPrivate}>
          <p style={{ fontSize: 13, lineHeight: 1.65, margin: 0, marginBottom: 8 }}>J'accompagne les startups B2B dans la conception de produits SaaS clairs et au goût du jour. Huit ans entre Paris et Berlin, avec un faible pour les <strong style={{ color: SDM.accent }}>fintech, healthtech et marketplaces</strong>.</p>
          <p style={{ fontSize: 13, lineHeight: 1.65, margin: 0, color: SDM.textMute }}>Discovery, design system, design ops — j'aime accompagner les équipes produit dans la durée, généralement 3-4 jours par semaine.</p>
        </MProfileSection>

        {/* Vidéo */}
        <MProfileSection title="Vidéo de présentation" isPrivate={isPrivate}>
          <div style={{ position: 'relative', borderRadius: 10, overflow: 'hidden', height: 180, background: 'linear-gradient(135deg, #2a1f15 0%, #4a3520 100%)' }}>
            <div style={{ position: 'absolute', inset: 0, background: 'radial-gradient(circle at 30% 40%, rgba(232,93,74,0.25), transparent 60%)' }} />
            <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <div style={{ width: 52, height: 52, borderRadius: '50%', background: 'rgba(255,255,255,0.95)', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 6px 20px rgba(0,0,0,0.3)' }}>
                <div style={{ width: 0, height: 0, borderLeft: `14px solid ${SDM.text}`, borderTop: '9px solid transparent', borderBottom: '9px solid transparent', marginLeft: 4 }} />
              </div>
            </div>
            <div style={{ position: 'absolute', bottom: 10, left: 12, color: '#fff', fontSize: 10.5, opacity: 0.85, fontFamily: SDM.serif, fontStyle: 'italic' }}>1 min 12</div>
          </div>
        </MProfileSection>

        {/* Disponibilité */}
        <MProfileSection title="Disponibilité" isPrivate={isPrivate}>
          <div style={{ fontSize: 12.5, color: SDM.textMute, lineHeight: 1.6 }}><strong style={{ color: SDM.text }}>3-4 jours</strong> par semaine · <strong style={{ color: SDM.text }}>À distance</strong> ou <strong style={{ color: SDM.text }}>hybride Paris</strong></div>
        </MProfileSection>

        {/* Domaines */}
        <MProfileSection title="Domaines d'expertise" isPrivate={isPrivate}>
          <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap' }}>
            {['Design & UI/UX', 'Product & UX Research', 'Consulting & Stratégie'].map((t, i) => (
              <span key={i} style={{ fontSize: 11.5, padding: '5px 11px', background: SDM.accentSoft, color: SDM.accentDeep, border: `1px solid ${SDM.accent}`, borderRadius: 999, fontWeight: 600 }}>{t}</span>
            ))}
          </div>
        </MProfileSection>

        {/* Compétences */}
        <MProfileSection title="Compétences & outils" isPrivate={isPrivate}>
          <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap' }}>
            {['Figma', 'Design System', 'UX Research', 'Brand', 'Webflow', 'Framer', 'Notion', 'Discovery', 'Prototypage'].map((s, i) => (
              <span key={i} style={{ fontSize: 11, padding: '4px 10px', background: SDM.bg, borderRadius: 999, fontWeight: 500 }}>{s}</span>
            ))}
          </div>
        </MProfileSection>

        {/* Tarifs */}
        <MProfileSection title="Tarifs" isPrivate={isPrivate}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 10 }}>
            <div style={{ background: SDM.bg, borderRadius: 10, padding: 12 }}>
              <div style={{ fontSize: 10, color: SDM.textMute, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 3 }}>TJM</div>
              <div style={{ fontFamily: SDM.serif, fontSize: 18, fontWeight: 500 }}>650 €</div>
            </div>
            <div style={{ background: SDM.bg, borderRadius: 10, padding: 12 }}>
              <div style={{ fontSize: 10, color: SDM.textMute, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 3 }}>1/2 journée</div>
              <div style={{ fontFamily: SDM.serif, fontSize: 18, fontWeight: 500 }}>380 €</div>
            </div>
          </div>
          <div style={{ fontSize: 11.5, color: SDM.textMute, marginBottom: 6, fontWeight: 600 }}>Forfaits</div>
          <div style={{ fontSize: 12, padding: '8px 0', borderTop: `1px solid ${SDM.border}`, display: 'flex', justifyContent: 'space-between' }}>
            <span>Audit UX express</span><strong>1 800 €</strong>
          </div>
          <div style={{ fontSize: 12, padding: '8px 0', borderTop: `1px solid ${SDM.border}`, display: 'flex', justifyContent: 'space-between' }}>
            <span>Refonte design system</span><span style={{ color: SDM.textMute, fontStyle: 'italic' }}>sur devis</span>
          </div>
        </MProfileSection>

        {/* Réalisations */}
        <MProfileSection title="Réalisations" isPrivate={isPrivate}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 6 }}>
            {[['#fde9e3', '#c43a26'], ['#e8f2eb', '#5a9670'], ['#fde6ed', '#c43a26'], ['#fbf0dc', '#d4924a']].map(([bg, color], i) => (
              <div key={i} style={{ aspectRatio: '1', background: bg, borderRadius: 10, padding: 10, display: 'flex', alignItems: 'flex-end' }}>
                <div style={{ fontSize: 10.5, color, fontWeight: 600 }}>Projet {i + 1}</div>
              </div>
            ))}
          </div>
        </MProfileSection>

        {/* Avis */}
        <MProfileSection title="Avis · 4,9 sur 47" isPrivate={isPrivate}>
          {[
            { name: 'Marc Lefèvre', co: 'Nova', text: 'Élise a su poser un cadre méthodo en deux semaines. On a refait notre onboarding avec une vraie clarté.' },
            { name: 'Sophie Aubry', co: 'Memo Bank', text: 'Une vraie partenaire produit. À recommander les yeux fermés.' },
          ].map((r, i) => (
            <div key={i} style={{ paddingTop: i === 0 ? 0 : 12, paddingBottom: i === 1 ? 0 : 12, borderBottom: i === 0 ? `1px solid ${SDM.border}` : 'none' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                <SDMPortrait id={i + 2} size={28} />
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 12.5, fontWeight: 600 }}>{r.name}</div>
                  <div style={{ fontSize: 10.5, color: SDM.textMute }}>{r.co}</div>
                </div>
                <div style={{ fontSize: 11, color: SDM.amber, fontWeight: 700 }}>★ 5,0</div>
              </div>
              <p style={{ fontSize: 12, lineHeight: 1.55, margin: 0, color: SDM.text, fontFamily: SDM.serif, fontStyle: 'italic' }}>« {r.text} »</p>
            </div>
          ))}
        </MProfileSection>
      </div>

      {/* Sticky CTA — public uniquement */}
      {!isPrivate ? (
        <div style={{ position: 'absolute', left: 0, right: 0, bottom: 0, padding: '12px 14px', background: '#fff', borderTop: `1px solid ${SDM.border}`, display: 'flex', gap: 8 }}>
          <button style={{ width: 48, background: '#fff', border: `1px solid ${SDM.borderStrong}`, borderRadius: 999, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SDMI name="Bookmark" size={16} /></button>
          <button style={{ flex: 1, background: SDM.text, color: '#fff', border: 'none', padding: '12px', fontSize: 13, fontWeight: 600, borderRadius: 999, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}><SDMI name="Send" size={14} /> Envoyer un message</button>
        </div>
      ) : null}
    </MobileFrame>
  );
}

window.SoleilProfileMobile = SoleilProfileMobile;
