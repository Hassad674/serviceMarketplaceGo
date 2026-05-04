// Lot D — Profil prestataire (page unique, sections empilées, public/privé)
// Logique: même page pour les deux. En privé chaque section a un crayon → édition inline. En public, lecture seule + bouton "Envoyer un message" sticky.

const SD = window.S;
const SDI = window.SI;
const SDSidebar = window.SSidebar;
const SDTopbar = window.STopbar;
const SDPortrait = window.Portrait;

// ─── Section wrapper avec crayon optionnel (mode privé) ────────
function ProfileSection({ title, subtitle, isPrivate, editing, onEdit, onCancel, onSave, children }) {
  return (
    <div style={{ background: '#fff', border: `1px solid ${SD.border}`, borderRadius: 16, padding: 24, marginBottom: 14 }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: 14, gap: 16 }}>
        <div>
          <h2 style={{ fontFamily: SD.serif, fontSize: 20, margin: 0, fontWeight: 600, letterSpacing: '-0.005em' }}>{title}</h2>
          {subtitle ? <div style={{ fontSize: 12.5, color: SD.textMute, marginTop: 3, fontStyle: 'italic', fontFamily: SD.serif }}>{subtitle}</div> : null}
        </div>
        {isPrivate ? (
          editing ? (
            <div style={{ display: 'flex', gap: 6 }}>
              <button onClick={onCancel} style={{ background: '#fff', border: `1px solid ${SD.border}`, padding: '7px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', color: SD.textMute }}>Annuler</button>
              <button onClick={onSave} style={{ background: SD.text, color: '#fff', border: 'none', padding: '7px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Enregistrer</button>
            </div>
          ) : (
            <button onClick={onEdit} style={{ background: 'transparent', border: `1px solid ${SD.border}`, width: 32, height: 32, borderRadius: '50%', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SD.textMute, flexShrink: 0 }}><SDI name="Edit" size={14} /></button>
          )
        ) : null}
      </div>
      {children}
    </div>
  );
}

// ─── Header (photo, nom, titre, dispo, tarif, action) ──────────
function HeaderBlock({ isPrivate, editingSection, setEditingSection }) {
  const editing = editingSection === 'header';
  return (
    <div style={{ background: '#fff', border: `1px solid ${SD.border}`, borderRadius: 16, padding: 28, marginBottom: 14, display: 'flex', gap: 22, alignItems: 'flex-start' }}>
      <div style={{ position: 'relative' }}>
        <div style={{ padding: 4, background: '#fff', borderRadius: 22, boxShadow: '0 2px 12px rgba(42,31,21,0.06)' }}>
          <SDPortrait id={1} size={120} rounded={18} />
        </div>
        {isPrivate ? <div style={{ position: 'absolute', bottom: 4, right: 4, width: 28, height: 28, borderRadius: '50%', background: SD.text, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}><SDI name="Edit" size={12} /></div> : <div style={{ position: 'absolute', bottom: 0, right: 0, width: 26, height: 26, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 2px 6px rgba(0,0,0,0.15)' }}><SDI name="Verified" size={16} /></div>}
      </div>
      <div style={{ flex: 1, paddingTop: 4 }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, marginBottom: 4, flexWrap: 'wrap' }}>
          {editing ? (
            <input defaultValue="Élise Marchand" style={{ fontFamily: SD.serif, fontSize: 32, fontWeight: 500, letterSpacing: '-0.025em', border: `1.5px solid ${SD.accent}`, borderRadius: 8, padding: '4px 10px', outline: 'none' }} />
          ) : (
            <h1 style={{ fontFamily: SD.serif, fontSize: 32, margin: 0, fontWeight: 500, letterSpacing: '-0.025em' }}>Élise Marchand</h1>
          )}
          <span style={{ fontSize: 11, padding: '4px 10px', background: SD.greenSoft, color: SD.green, borderRadius: 999, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 5 }}><span style={{ width: 6, height: 6, borderRadius: '50%', background: SD.green }} /> Disponible dès lundi</span>
        </div>
        {editing ? (
          <input defaultValue="UX Designer & Brand pour startups B2B" style={{ width: '60%', fontFamily: SD.serif, fontSize: 16, fontStyle: 'italic', color: SD.textMute, border: `1px solid ${SD.borderStrong}`, borderRadius: 8, padding: '6px 10px', outline: 'none', marginBottom: 12 }} />
        ) : (
          <div style={{ fontSize: 16, color: SD.textMute, marginBottom: 12, fontFamily: SD.serif, fontStyle: 'italic' }}>UX Designer & Brand pour startups B2B</div>
        )}
        <div style={{ display: 'flex', gap: 18, fontSize: 12.5, color: SD.textMute, alignItems: 'center', flexWrap: 'wrap' }}>
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}><SDI name="MapPin" size={13} /> Paris</span>
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}><SDI name="Globe" size={13} /> Français · Anglais</span>
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}><SDI name="Star" size={13} /> <strong style={{ color: SD.text }}>4,9</strong> · 47 avis</span>
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}><SDI name="Clock" size={13} /> Répond en 2 h</span>
        </div>
      </div>
      <div style={{ textAlign: 'right', display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 12 }}>
        <div>
          <div style={{ fontSize: 10.5, color: SD.textMute, marginBottom: 2, letterSpacing: '0.06em', textTransform: 'uppercase', fontWeight: 600 }}>À partir de</div>
          <div style={{ fontFamily: SD.serif, fontSize: 32, fontWeight: 500, lineHeight: 1, letterSpacing: '-0.025em' }}>650 €<span style={{ fontSize: 14, color: SD.textMute, fontWeight: 400 }}>/jour</span></div>
        </div>
        {isPrivate ? (
          editing ? (
            <div style={{ display: 'flex', gap: 6 }}>
              <button onClick={() => setEditingSection(null)} style={{ background: '#fff', border: `1px solid ${SD.border}`, padding: '7px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', color: SD.textMute }}>Annuler</button>
              <button onClick={() => setEditingSection(null)} style={{ background: SD.text, color: '#fff', border: 'none', padding: '7px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Enregistrer</button>
            </div>
          ) : (
            <button onClick={() => setEditingSection('header')} style={{ background: 'transparent', border: `1px solid ${SD.border}`, width: 32, height: 32, borderRadius: '50%', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SD.textMute }}><SDI name="Edit" size={14} /></button>
          )
        ) : null}
      </div>
    </div>
  );
}

// ─── Profil unifié ─────────────────────────────────────────────
function SoleilProfile({ isPrivate = false }) {
  const [editingSection, setEditingSection] = React.useState(null);
  const e = (key) => editingSection === key;
  const open = (key) => setEditingSection(key);
  const close = () => setEditingSection(null);

  const editProps = (key) => ({
    isPrivate,
    editing: e(key),
    onEdit: () => open(key),
    onCancel: close,
    onSave: close,
  });

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SD.bg, fontFamily: SD.sans, color: SD.text }}>
      <SDSidebar active={isPrivate ? 'profile' : 'find'} role={isPrivate ? 'freelancer' : 'enterprise'} />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SDTopbar />
        <div style={{ flex: 1, overflow: 'auto', padding: '28px 40px', position: 'relative' }}>
          <div style={{ maxWidth: 880, margin: '0 auto' }}>

            {/* Fil d'ariane */}
            {!isPrivate ? (
              <div style={{ fontSize: 12, color: SD.textMute, marginBottom: 14, display: 'flex', alignItems: 'center', gap: 6 }}>
                <SDI name="ArrowLeft" size={12} /> Retour aux freelances
              </div>
            ) : (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
                <div style={{ fontSize: 13, color: SD.textMute, fontFamily: SD.serif, fontStyle: 'italic' }}>Mode édition · les modifications sont enregistrées section par section.</div>
                <button style={{ background: '#fff', border: `1px solid ${SD.borderStrong}`, padding: '7px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}><SDI name="Eye" size={13} /> Aperçu public</button>
              </div>
            )}

            {/* 0 — Header */}
            <HeaderBlock isPrivate={isPrivate} editingSection={editingSection} setEditingSection={setEditingSection} />

            {/* Complétion (privé seulement) */}
            {isPrivate ? (
              <div style={{ background: SD.accentSoft, border: `1px solid ${SD.accent}`, borderRadius: 14, padding: '14px 18px', marginBottom: 14, display: 'flex', alignItems: 'center', gap: 16 }}>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6, fontSize: 13 }}>
                    <span style={{ fontWeight: 600 }}>Profil complété à 78 %</span>
                    <span style={{ color: SD.accentDeep, fontWeight: 700 }}>4 sections à finir</span>
                  </div>
                  <div style={{ height: 6, background: 'rgba(196,58,38,0.15)', borderRadius: 999, overflow: 'hidden' }}>
                    <div style={{ width: '78%', height: '100%', background: SD.accent, borderRadius: 999 }} />
                  </div>
                </div>
                <div style={{ fontSize: 12, color: SD.textMute, fontStyle: 'italic', fontFamily: SD.serif, maxWidth: 240, textAlign: 'right' }}>Compétences, langues, tarifs et avis pour atteindre 100 %.</div>
              </div>
            ) : null}

            {/* 1 — À propos */}
            <ProfileSection title="À propos" {...editProps('about')}>
              {e('about') ? (
                <textarea defaultValue="J'accompagne les startups B2B dans la conception de produits SaaS clairs et au goût du jour. Huit ans entre Paris et Berlin, avec un faible pour les fintech, healthtech et marketplaces.

Discovery, design system, design ops — j'aime accompagner les équipes produit dans la durée, généralement 3 à 6 mois, en mode 3-4 jours par semaine." style={{ width: '100%', minHeight: 140, border: `1.5px solid ${SD.accent}`, borderRadius: 10, padding: '12px 14px', fontSize: 14.5, fontFamily: SD.sans, outline: 'none', lineHeight: 1.65, resize: 'vertical' }} />
              ) : (
                <div>
                  <p style={{ fontSize: 14.5, lineHeight: 1.7, margin: 0, marginBottom: 10, textWrap: 'pretty' }}>J'accompagne les startups B2B dans la conception de produits SaaS clairs et au goût du jour. Huit ans entre Paris et Berlin, avec un faible pour les <strong style={{ color: SD.accent }}>fintech, healthtech et marketplaces</strong>.</p>
                  <p style={{ fontSize: 14.5, lineHeight: 1.7, margin: 0, color: SD.textMute, textWrap: 'pretty' }}>Discovery, design system, design ops — j'aime accompagner les équipes produit dans la durée, généralement 3 à 6 mois, en mode 3-4 jours par semaine.</p>
                </div>
              )}
            </ProfileSection>

            {/* 2 — Vidéo de présentation */}
            <ProfileSection title="Vidéo de présentation" subtitle={isPrivate ? "Une minute pour te présenter. Ça double tes chances d'être contactée." : null} {...editProps('video')}>
              <div style={{ position: 'relative', borderRadius: 12, overflow: 'hidden', height: 280, background: 'linear-gradient(135deg, #2a1f15 0%, #4a3520 100%)' }}>
                <div style={{ position: 'absolute', inset: 0, background: 'radial-gradient(circle at 30% 40%, rgba(232,93,74,0.25), transparent 60%)' }} />
                <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <div style={{ width: 72, height: 72, borderRadius: '50%', background: 'rgba(255,255,255,0.95)', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', boxShadow: '0 8px 32px rgba(0,0,0,0.3)' }}>
                    <div style={{ width: 0, height: 0, borderLeft: `20px solid ${SD.text}`, borderTop: '12px solid transparent', borderBottom: '12px solid transparent', marginLeft: 6 }} />
                  </div>
                </div>
                <div style={{ position: 'absolute', bottom: 16, left: 18, color: '#fff', fontSize: 12, opacity: 0.85, fontFamily: SD.serif, fontStyle: 'italic' }}>1 min 12 · « Bonjour, je suis Élise… »</div>
                <div style={{ position: 'absolute', bottom: 16, right: 18, color: '#fff', fontSize: 11, opacity: 0.7, display: 'flex', alignItems: 'center', gap: 5 }}><SDI name="Eye" size={11} /> 234 vues</div>
              </div>
            </ProfileSection>

            {/* 3 — Disponibilité */}
            <ProfileSection title="Disponibilité" {...editProps('availability')}>
              {e('availability') ? (
                <div>
                  <div style={{ display: 'flex', gap: 6, marginBottom: 14 }}>
                    {[
                      { l: 'Disponible maintenant', sel: true, color: SD.green, bg: SD.greenSoft },
                      { l: 'Disponible bientôt', color: SD.amber, bg: '#fbf0dc' },
                      { l: 'Indisponible', color: SD.textMute, bg: SD.bg },
                    ].map((d, i) => (
                      <button key={i} style={{ flex: 1, padding: '10px 12px', background: d.sel ? d.bg : '#fff', border: `1.5px solid ${d.sel ? d.color : SD.border}`, color: d.sel ? d.color : SD.text, borderRadius: 12, fontSize: 13, fontWeight: 600, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}>{d.sel ? <SDI name="CheckCircle" size={13} /> : null}{d.l}</button>
                    ))}
                  </div>
                  <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8 }}>Mode de travail</div>
                  <div style={{ display: 'flex', gap: 6 }}>
                    {[{ l: 'Sur site' }, { l: 'À distance', sel: true }, { l: 'Hybride', sel: true }].map((m, i) => (
                      <button key={i} style={{ padding: '7px 14px', background: m.sel ? SD.text : '#fff', color: m.sel ? '#fff' : SD.text, border: m.sel ? 'none' : `1px solid ${SD.border}`, borderRadius: 999, fontSize: 12, fontWeight: 600, cursor: 'pointer' }}>{m.l}</button>
                    ))}
                  </div>
                </div>
              ) : (
                <div style={{ display: 'flex', gap: 24, alignItems: 'center', flexWrap: 'wrap' }}>
                  <div style={{ display: 'inline-flex', alignItems: 'center', gap: 7, padding: '7px 14px', background: SD.greenSoft, color: SD.green, borderRadius: 999, fontSize: 13, fontWeight: 600 }}><span style={{ width: 7, height: 7, borderRadius: '50%', background: SD.green }} /> Disponible dès lundi 5 mai</div>
                  <div style={{ fontSize: 13.5, color: SD.textMute }}><strong style={{ color: SD.text }}>3-4 jours</strong> par semaine · <strong style={{ color: SD.text }}>À distance</strong> ou <strong style={{ color: SD.text }}>hybride Paris</strong></div>
                </div>
              )}
            </ProfileSection>

            {/* 4 — Domaines d'expertise */}
            <ProfileSection title="Domaines d'expertise" subtitle={isPrivate ? "Choisis jusqu'à 5 domaines qui mettent en valeur ce que tu fais le mieux." : null} {...editProps('expertise')}>
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {(e('expertise') ? [
                  { l: 'Design & UI/UX', sel: true },
                  { l: 'Product & UX Research', sel: true },
                  { l: 'Consulting & Stratégie', sel: true },
                  { l: 'Développement' },
                  { l: 'Data, IA & Machine Learning' },
                  { l: 'Design 3D & Animation' },
                  { l: 'Vidéo & Motion' },
                  { l: 'Photo & Audiovisuel' },
                  { l: 'Marketing & Growth' },
                  { l: 'Rédaction & Traduction' },
                  { l: 'Business Development & Ventes' },
                ] : [
                  { l: 'Design & UI/UX', sel: true },
                  { l: 'Product & UX Research', sel: true },
                  { l: 'Consulting & Stratégie', sel: true },
                ]).map((t, i) => (
                  <span key={i} style={{ fontSize: 13, padding: '7px 14px', background: t.sel ? SD.accentSoft : '#fff', color: t.sel ? SD.accentDeep : SD.text, border: t.sel ? `1.5px solid ${SD.accent}` : `1px solid ${SD.border}`, borderRadius: 999, fontWeight: t.sel ? 600 : 500, cursor: e('expertise') ? 'pointer' : 'default' }}>{t.l}</span>
                ))}
              </div>
              {e('expertise') ? <div style={{ marginTop: 10, fontSize: 11.5, color: SD.textMute }}>3/5 sélectionnés</div> : null}
            </ProfileSection>

            {/* 5 — Compétences & outils */}
            <ProfileSection title="Compétences & outils" {...editProps('skills')}>
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {['Figma', 'Design System', 'UX Research', 'Brand', 'Webflow', 'Framer', 'Notion', 'Discovery', 'Wireframing', 'Prototypage', 'Atomic Design'].map((s, i) => (
                  <span key={i} style={{ fontSize: 12.5, padding: '6px 12px', background: SD.bg, borderRadius: 999, fontWeight: 500, display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                    {s}
                    {e('skills') ? <span style={{ cursor: 'pointer', opacity: 0.5, fontSize: 14 }}>×</span> : null}
                  </span>
                ))}
                {e('skills') ? <input placeholder="Ajouter…" style={{ border: `1px dashed ${SD.borderStrong}`, borderRadius: 999, padding: '6px 12px', fontSize: 12.5, outline: 'none', minWidth: 120 }} /> : null}
              </div>
            </ProfileSection>

            {/* 6 — Tarifs */}
            <ProfileSection title="Tarifs" subtitle={isPrivate ? "Comment tu factures tes prestations." : null} {...editProps('rates')}>
              {e('rates') ? (
                <div>
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                    <div>
                      <label style={{ fontSize: 12, fontWeight: 600, display: 'block', marginBottom: 6 }}>TJM (jour)</label>
                      <div style={{ position: 'relative' }}>
                        <input defaultValue="650" style={{ width: '100%', border: `1.5px solid ${SD.accent}`, borderRadius: 10, padding: '10px 36px 10px 12px', fontSize: 14, fontFamily: SD.sans, outline: 'none' }} />
                        <span style={{ position: 'absolute', right: 14, top: '50%', transform: 'translateY(-50%)', color: SD.textMute, fontSize: 13, fontFamily: SD.serif }}>€/j</span>
                      </div>
                    </div>
                    <div>
                      <label style={{ fontSize: 12, fontWeight: 600, display: 'block', marginBottom: 6 }}>Demi-journée</label>
                      <div style={{ position: 'relative' }}>
                        <input defaultValue="380" style={{ width: '100%', border: `1px solid ${SD.borderStrong}`, borderRadius: 10, padding: '10px 36px 10px 12px', fontSize: 14, fontFamily: SD.sans, outline: 'none' }} />
                        <span style={{ position: 'absolute', right: 14, top: '50%', transform: 'translateY(-50%)', color: SD.textMute, fontSize: 13, fontFamily: SD.serif }}>€</span>
                      </div>
                    </div>
                  </div>
                  <div style={{ marginTop: 14 }}>
                    <label style={{ fontSize: 12, fontWeight: 600, display: 'block', marginBottom: 6 }}>Forfaits proposés</label>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                      <input defaultValue="Audit UX express — 1 800 €" style={{ border: `1px solid ${SD.borderStrong}`, borderRadius: 10, padding: '10px 12px', fontSize: 13.5, fontFamily: SD.sans, outline: 'none' }} />
                      <input defaultValue="Refonte design system — sur devis" style={{ border: `1px solid ${SD.borderStrong}`, borderRadius: 10, padding: '10px 12px', fontSize: 13.5, fontFamily: SD.sans, outline: 'none' }} />
                      <button style={{ alignSelf: 'flex-start', background: 'transparent', border: `1px dashed ${SD.borderStrong}`, padding: '7px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', color: SD.textMute, display: 'flex', alignItems: 'center', gap: 5 }}><SDI name="Plus" size={12} /> Ajouter un forfait</button>
                    </div>
                  </div>
                </div>
              ) : (
                <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
                  <div style={{ flex: '1 1 180px', padding: 16, background: SD.bg, borderRadius: 12 }}>
                    <div style={{ fontSize: 11, color: SD.textMute, marginBottom: 4, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600 }}>TJM</div>
                    <div style={{ fontFamily: SD.serif, fontSize: 26, fontWeight: 500, letterSpacing: '-0.02em' }}>650 €<span style={{ fontSize: 13, color: SD.textMute, fontWeight: 400 }}>/jour</span></div>
                  </div>
                  <div style={{ flex: '1 1 180px', padding: 16, background: SD.bg, borderRadius: 12 }}>
                    <div style={{ fontSize: 11, color: SD.textMute, marginBottom: 4, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600 }}>Demi-journée</div>
                    <div style={{ fontFamily: SD.serif, fontSize: 26, fontWeight: 500, letterSpacing: '-0.02em' }}>380 €</div>
                  </div>
                  <div style={{ flex: '1 1 180px', padding: 16, background: SD.bg, borderRadius: 12 }}>
                    <div style={{ fontSize: 11, color: SD.textMute, marginBottom: 4, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600 }}>Audit UX express</div>
                    <div style={{ fontFamily: SD.serif, fontSize: 26, fontWeight: 500, letterSpacing: '-0.02em' }}>1 800 €</div>
                  </div>
                </div>
              )}
            </ProfileSection>

            {/* 7 — Réalisations */}
            <ProfileSection title="Réalisations" {...editProps('portfolio')}>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 12 }}>
                {[
                  { title: 'Refonte des cartes Qonto', client: 'Qonto · Fintech · 2024', g1: '#3a4ee0', g2: '#7c8df0' },
                  { title: 'Design System v2', client: 'Memo Bank · 2023', g1: '#0e8a5f', g2: '#5fb88a' },
                  { title: 'Onboarding mobile', client: 'Lydia · 2023', g1: '#e8447b', g2: '#f47ea4' },
                  { title: 'Doctolib Pro', client: 'Doctolib · 2022', g1: '#b8721d', g2: '#d9a05c' },
                ].map((p, i) => (
                  <div key={i} style={{ borderRadius: 12, overflow: 'hidden', border: `1px solid ${SD.border}`, position: 'relative' }}>
                    <div style={{ height: 140, background: `linear-gradient(135deg, ${p.g1}, ${p.g2})` }} />
                    {e('portfolio') ? <button style={{ position: 'absolute', top: 8, right: 8, width: 26, height: 26, borderRadius: 8, background: 'rgba(255,255,255,0.95)', border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SDI name="Edit" size={11} /></button> : null}
                    <div style={{ padding: 12, background: '#fff' }}>
                      <div style={{ fontSize: 13.5, fontWeight: 600, fontFamily: SD.serif }}>{p.title}</div>
                      <div style={{ fontSize: 11.5, color: SD.textMute, marginTop: 2 }}>{p.client}</div>
                    </div>
                  </div>
                ))}
                {e('portfolio') ? (
                  <div style={{ borderRadius: 12, border: `1.5px dashed ${SD.borderStrong}`, padding: '20px 12px', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 8, cursor: 'pointer', minHeight: 168 }}>
                    <div style={{ width: 36, height: 36, borderRadius: '50%', background: SD.bg, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SDI name="Plus" size={16} /></div>
                    <div style={{ fontSize: 12.5, fontWeight: 600 }}>Ajouter une réalisation</div>
                  </div>
                ) : null}
              </div>
            </ProfileSection>

            {/* 8 — Historique des projets & avis */}
            <ProfileSection title="Historique des projets" subtitle="3 missions terminées sur Atelier" isPrivate={false}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                {/* Projet avec avis */}
                <div style={{ border: `1px solid ${SD.border}`, borderRadius: 14, padding: 18, background: SD.bg }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 12 }}>
                    <div style={{ fontSize: 12, fontWeight: 700, color: SD.green, padding: '4px 10px', background: SD.greenSoft, borderRadius: 999 }}>4 200 €</div>
                    <div style={{ fontSize: 12, color: SD.textMute, display: 'flex', alignItems: 'center', gap: 5 }}><SDI name="Clock" size={11} /> Terminé le 18 avril 2026</div>
                  </div>
                  <div style={{ fontSize: 16, fontWeight: 600, fontFamily: SD.serif, marginBottom: 12 }}>Refonte du parcours d'inscription Qonto Pro</div>
                  <div style={{ background: '#fff', borderRadius: 10, padding: 14, border: `1px solid ${SD.border}` }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                      <div style={{ display: 'flex', gap: 1 }}>
                        {[1,2,3,4,5].map(s => <SDI key={s} name="Star" size={14} />)}
                      </div>
                      <div style={{ fontSize: 11, color: SD.textMute }}>18/04/2026</div>
                    </div>
                    <div style={{ display: 'flex', gap: 14, fontSize: 11.5, color: SD.textMute, marginBottom: 10 }}>
                      <span>Respect des délais <strong style={{ color: SD.text }}>5/5</strong></span>
                      <span>Communication <strong style={{ color: SD.text }}>5/5</strong></span>
                      <span>Qualité du livrable <strong style={{ color: SD.text }}>5/5</strong></span>
                    </div>
                    <div style={{ fontSize: 13.5, lineHeight: 1.6, fontFamily: SD.serif, fontStyle: 'italic', color: SD.text, textWrap: 'pretty' }}>« Élise a posé un cadre méthodo dès la première semaine. On est passés d'un design system fragmenté à une vraie cohésion produit. »</div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginTop: 12, paddingTop: 10, borderTop: `1px solid ${SD.border}` }}>
                      <SDPortrait id={4} size={28} />
                      <div style={{ fontSize: 12 }}><strong>Sophie Aubry</strong> · CPO chez Qonto</div>
                    </div>
                  </div>
                </div>

                {/* Projet avec avis vidéo */}
                <div style={{ border: `1px solid ${SD.border}`, borderRadius: 14, padding: 18, background: SD.bg }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 12 }}>
                    <div style={{ fontSize: 12, fontWeight: 700, color: SD.green, padding: '4px 10px', background: SD.greenSoft, borderRadius: 999 }}>7 350 €</div>
                    <div style={{ fontSize: 12, color: SD.textMute, display: 'flex', alignItems: 'center', gap: 5 }}><SDI name="Clock" size={11} /> Terminé le 02 mars 2026</div>
                  </div>
                  <div style={{ fontSize: 16, fontWeight: 600, fontFamily: SD.serif, marginBottom: 12 }}>Onboarding mobile Lydia · 6 semaines</div>
                  <div style={{ background: '#fff', borderRadius: 10, padding: 14, border: `1px solid ${SD.border}` }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
                      <div style={{ display: 'flex', gap: 1 }}>
                        {[1,2,3,4,5].map(s => <SDI key={s} name="Star" size={14} />)}
                      </div>
                      <div style={{ fontSize: 11, color: SD.textMute, display: 'flex', alignItems: 'center', gap: 4 }}><SDI name="Video" size={11} /> Avis vidéo · 02/03/2026</div>
                    </div>
                    <div style={{ position: 'relative', borderRadius: 10, overflow: 'hidden', height: 160, background: 'linear-gradient(135deg, #5a3a1f 0%, #8b4a1f 100%)', marginBottom: 12 }}>
                      <div style={{ position: 'absolute', inset: 0, background: 'radial-gradient(circle at 70% 50%, rgba(232,93,74,0.3), transparent 60%)' }} />
                      <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <div style={{ width: 56, height: 56, borderRadius: '50%', background: 'rgba(255,255,255,0.95)', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}>
                          <div style={{ width: 0, height: 0, borderLeft: `15px solid ${SD.text}`, borderTop: '10px solid transparent', borderBottom: '10px solid transparent', marginLeft: 5 }} />
                        </div>
                      </div>
                      <div style={{ position: 'absolute', bottom: 12, left: 14, color: '#fff', fontSize: 11, opacity: 0.85, fontFamily: SD.serif, fontStyle: 'italic' }}>0 min 48 · « On a senti la différence dès la première semaine… »</div>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10, paddingTop: 4 }}>
                      <SDPortrait id={2} size={28} />
                      <div style={{ fontSize: 12 }}><strong>Marc Lefèvre</strong> · Head of Product, Lydia</div>
                    </div>
                  </div>
                </div>

                {/* Projet en attente d'avis */}
                <div style={{ border: `1px solid ${SD.border}`, borderRadius: 14, padding: 18, background: SD.bg }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 12 }}>
                    <div style={{ fontSize: 12, fontWeight: 700, color: SD.green, padding: '4px 10px', background: SD.greenSoft, borderRadius: 999 }}>2 800 €</div>
                    <div style={{ fontSize: 12, color: SD.textMute, display: 'flex', alignItems: 'center', gap: 5 }}><SDI name="Clock" size={11} /> Terminé le 28 avril 2026</div>
                  </div>
                  <div style={{ fontSize: 16, fontWeight: 600, fontFamily: SD.serif, marginBottom: 10 }}>Audit UX Memo Bank · semaine</div>
                  <div style={{ fontSize: 12.5, color: SD.textMute, fontStyle: 'italic', fontFamily: SD.serif, padding: '10px 14px', background: '#fff', borderRadius: 10, border: `1px dashed ${SD.borderStrong}`, textAlign: 'center' }}>En attente d'avis · le client a 14 jours pour le déposer</div>
                </div>
              </div>
            </ProfileSection>

            {/* 9 — Localisation */}
            <ProfileSection title="Localisation" subtitle={isPrivate ? "D'où tu travailles, et comment." : null} {...editProps('location')}>
              {e('location') ? (
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                  <div>
                    <label style={{ fontSize: 12, fontWeight: 600, display: 'block', marginBottom: 6 }}>Ville</label>
                    <input defaultValue="Paris, 11ᵉ" style={{ width: '100%', border: `1.5px solid ${SD.accent}`, borderRadius: 10, padding: '10px 12px', fontSize: 13.5, outline: 'none' }} />
                  </div>
                  <div>
                    <label style={{ fontSize: 12, fontWeight: 600, display: 'block', marginBottom: 6 }}>Pays</label>
                    <select style={{ width: '100%', border: `1px solid ${SD.borderStrong}`, borderRadius: 10, padding: '10px 12px', fontSize: 13.5, outline: 'none', background: '#fff' }}><option>France</option></select>
                  </div>
                </div>
              ) : (
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 14 }}>
                  <SDI name="MapPin" size={15} />
                  <span><strong>Paris, 11ᵉ</strong> · France</span>
                </div>
              )}
            </ProfileSection>

            {/* 10 — Langues */}
            <ProfileSection title="Langues" subtitle={isPrivate ? "Les langues dans lesquelles tu peux travailler." : null} {...editProps('languages')}>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
                <div>
                  <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8 }}>Professionnelles</div>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                    {['Français', 'Anglais'].map((l, i) => (
                      <span key={i} style={{ fontSize: 12.5, padding: '6px 12px', background: SD.accentSoft, color: SD.accentDeep, borderRadius: 999, fontWeight: 600 }}>{l}{e('languages') ? <span style={{ marginLeft: 6, opacity: 0.6, cursor: 'pointer' }}>×</span> : null}</span>
                    ))}
                  </div>
                </div>
                <div>
                  <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8 }}>Conversationnelles</div>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                    <span style={{ fontSize: 12.5, padding: '6px 12px', background: SD.bg, borderRadius: 999, fontWeight: 500 }}>Espagnol</span>
                  </div>
                </div>
              </div>
            </ProfileSection>

            {/* 11 — Réseaux sociaux */}
            <ProfileSection title="Réseaux sociaux" {...editProps('social')}>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {[
                  { l: 'LinkedIn', v: 'elise-marchand' },
                  { l: 'Dribbble', v: 'elisem' },
                  { l: 'Behance', v: 'elise.marchand' },
                  { l: 'Site perso', v: 'elise.studio' },
                ].map((s, i) => (
                  <a key={i} style={{ fontSize: 12.5, padding: '8px 14px', background: '#fff', border: `1px solid ${SD.border}`, borderRadius: 999, fontWeight: 500, display: 'inline-flex', alignItems: 'center', gap: 6, color: SD.text, textDecoration: 'none', cursor: 'pointer' }}>
                    <SDI name="Globe" size={12} />
                    <span>{s.l}</span>
                    <span style={{ color: SD.textMute }}>· {s.v}</span>
                  </a>
                ))}
              </div>
            </ProfileSection>

            {/* 12 — Vérifié par Atelier */}
            {!isPrivate ? (
              <ProfileSection title="Vérifié par Atelier" isPrivate={false}>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12 }}>
                  {[
                    { l: 'Identité KYC', icon: 'Verified' },
                    { l: 'Email pro', icon: 'CheckCircle' },
                    { l: 'SIRET valide', icon: 'CheckCircle' },
                    { l: 'Top 5%', icon: 'Star' },
                  ].map((v, i) => (
                    <div key={i} style={{ padding: 14, background: SD.bg, borderRadius: 12, display: 'flex', alignItems: 'center', gap: 10 }}>
                      <div style={{ width: 32, height: 32, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SD.green }}><SDI name={v.icon} size={16} /></div>
                      <div style={{ fontSize: 12.5, fontWeight: 600 }}>{v.l}</div>
                    </div>
                  ))}
                </div>
              </ProfileSection>
            ) : null}

            {/* Suppression / paramètres avancés (privé) */}
            {isPrivate ? (
              <div style={{ marginTop: 24, padding: '16px 22px', background: 'transparent', textAlign: 'center', fontSize: 12, color: SD.textSubtle }}>
                Besoin d'aide ? <a style={{ color: SD.text, textDecoration: 'underline', cursor: 'pointer' }}>Contacte-nous</a> · <a style={{ color: SD.text, textDecoration: 'underline', cursor: 'pointer' }}>Désactiver mon profil</a>
              </div>
            ) : null}

          </div>

          {/* Bouton sticky "Envoyer un message" — vue publique */}
          {!isPrivate ? (
            <div style={{ position: 'sticky', bottom: 24, marginTop: 24, marginRight: 'auto', marginLeft: 'auto', maxWidth: 880, display: 'flex', justifyContent: 'flex-end', pointerEvents: 'none' }}>
              <button style={{ background: SD.text, color: '#fff', border: 'none', padding: '14px 24px', fontSize: 14, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, boxShadow: '0 8px 32px rgba(42,31,21,0.25)', pointerEvents: 'auto' }}><SDI name="Send" size={15} /> Envoyer un message à Élise</button>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

window.SoleilProfile = SoleilProfile;
