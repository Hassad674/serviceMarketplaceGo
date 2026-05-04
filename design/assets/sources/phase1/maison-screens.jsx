// Direction 1 — Maison: Find / Profile / Messages screens
const { Icons: I_M } = window;
const M = window.MAISON_TOKENS;
const M_Sidebar = window.MaisonSidebar;
const M_Topbar = window.MaisonTopbar;

// ═══ FIND FREELANCERS ══════════════════════════════════════════════
function MaisonFind() {
  const freelancers = [
    { name: 'Élise Marchand', title: 'UX Designer · Brand', loc: 'Paris', tjm: '650 €', exp: '8 ans', rating: 4.9, reviews: 47, avail: 'Disponible', tags: ['Figma', 'Design system', 'B2B'], p: 'EM', pBg: '#0e8a5f', verified: true, photo: true },
    { name: 'Julien Petit', title: 'Brand & Direction artistique', loc: 'Lyon', tjm: '720 €', exp: '12 ans', rating: 5.0, reviews: 31, avail: 'Sous 2 semaines', tags: ['Branding', 'Editorial', 'Art direction'], p: 'JP', pBg: '#e85d4a', verified: true, photo: false },
    { name: 'Théo Renaud', title: 'Développeur Full-Stack', loc: 'Remote', tjm: '580 €', exp: '6 ans', rating: 4.8, reviews: 62, avail: 'Disponible', tags: ['Next.js', 'Postgres', 'AWS'], p: 'TR', pBg: '#3a4ee0', verified: true, photo: true },
    { name: 'Camille Dubois', title: 'Product Designer', loc: 'Bordeaux', tjm: '600 €', exp: '7 ans', rating: 4.9, reviews: 38, avail: 'Disponible', tags: ['Mobile', 'iOS', 'Design ops'], p: 'CD', pBg: '#b8721d', verified: false, photo: false },
    { name: 'Mehdi Bensalem', title: 'Data Scientist', loc: 'Marseille', tjm: '750 €', exp: '9 ans', rating: 4.7, reviews: 24, avail: 'Sous 1 mois', tags: ['Python', 'ML', 'NLP'], p: 'MB', pBg: '#1f2caa', verified: true, photo: true },
    { name: 'Léa Fontaine', title: 'Motion Designer', loc: 'Nantes', tjm: '520 €', exp: '5 ans', rating: 4.9, reviews: 19, avail: 'Disponible', tags: ['After Effects', '3D', 'Branding'], p: 'LF', pBg: '#e8447b', verified: true, photo: false },
  ];

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: M.bg, fontFamily: M.sans, color: M.text }}>
      <M_Sidebar active="find" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <M_Topbar />
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex' }}>
          {/* Filters */}
          <aside style={{ width: 260, borderRight: `1px solid ${M.border}`, padding: '24px 20px', overflow: 'auto', flexShrink: 0 }}>
            <div style={{ fontFamily: M.serif, fontSize: 22, marginBottom: 20, fontWeight: 400 }}>Filtres</div>

            {[
              { title: 'Disponibilité', items: [['Disponible maintenant', true], ['Sous 2 semaines', false], ['Sous 1 mois', false]] },
              { title: 'Mode de travail', items: [['Remote', true], ['Sur site', false], ['Hybride', true]] },
              { title: 'Vérifié', items: [['Identité KYC', true], ['Top rated', false]] },
            ].map((g, gi) => (
              <div key={gi} style={{ marginBottom: 22 }}>
                <div style={{ fontSize: 11, letterSpacing: '0.12em', textTransform: 'uppercase', color: M.textMute, fontWeight: 600, marginBottom: 10 }}>{g.title}</div>
                {g.items.map(([label, checked], i) => (
                  <label key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '5px 0', fontSize: 13, cursor: 'pointer' }}>
                    <span style={{ width: 14, height: 14, border: `1.5px solid ${checked ? M.text : M.borderStrong}`, background: checked ? M.text : '#fff', borderRadius: 2, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                      {checked && <I_M.Check size={10} stroke={2.5} style={{ color: '#fff' }} />}
                    </span>
                    {label}
                  </label>
                ))}
              </div>
            ))}

            <div style={{ marginBottom: 22 }}>
              <div style={{ fontSize: 11, letterSpacing: '0.12em', textTransform: 'uppercase', color: M.textMute, fontWeight: 600, marginBottom: 10 }}>TJM</div>
              <div style={{ position: 'relative', height: 4, background: M.border, borderRadius: 2, marginBottom: 10 }}>
                <div style={{ position: 'absolute', left: '20%', right: '30%', height: '100%', background: M.text }} />
                <div style={{ position: 'absolute', left: '20%', top: -4, width: 12, height: 12, background: '#fff', border: `2px solid ${M.text}`, borderRadius: '50%', transform: 'translateX(-50%)' }} />
                <div style={{ position: 'absolute', left: '70%', top: -4, width: 12, height: 12, background: '#fff', border: `2px solid ${M.text}`, borderRadius: '50%', transform: 'translateX(-50%)' }} />
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, color: M.textMute, fontFamily: M.mono }}>
                <span>400 €</span>
                <span>950 €</span>
              </div>
            </div>

            <div style={{ marginBottom: 22 }}>
              <div style={{ fontSize: 11, letterSpacing: '0.12em', textTransform: 'uppercase', color: M.textMute, fontWeight: 600, marginBottom: 10 }}>Expertise</div>
              {['Développement', 'Design & UI/UX', 'Marketing & Growth', 'Data & IA', 'Photo & Vidéo', 'Conseil & Stratégie'].map((t, i) => (
                <label key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '5px 0', fontSize: 13, cursor: 'pointer' }}>
                  <span style={{ width: 14, height: 14, border: `1.5px solid ${i < 2 ? M.text : M.borderStrong}`, background: i < 2 ? M.text : '#fff', borderRadius: 2, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    {i < 2 && <I_M.Check size={10} stroke={2.5} style={{ color: '#fff' }} />}
                  </span>
                  {t}
                </label>
              ))}
            </div>

            <button style={{ width: '100%', background: M.text, color: '#fff', border: 'none', padding: '10px', fontSize: 13, fontWeight: 600, borderRadius: 4, cursor: 'pointer', fontFamily: M.sans }}>Appliquer</button>
            <button style={{ width: '100%', background: 'transparent', border: 'none', padding: '8px', fontSize: 12, color: M.textMute, cursor: 'pointer', marginTop: 6 }}>Réinitialiser</button>
          </aside>

          {/* Results */}
          <div style={{ flex: 1, overflow: 'auto', padding: '28px 32px' }}>
            <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', marginBottom: 20, paddingBottom: 18, borderBottom: `1px solid ${M.border}` }}>
              <div>
                <h1 style={{ fontFamily: M.serif, fontSize: 36, margin: 0, fontWeight: 400, marginBottom: 6, letterSpacing: '-0.02em' }}>Trouver des <em style={{ color: M.accent }}>freelances</em></h1>
                <div style={{ fontSize: 13, color: M.textMute }}><strong style={{ color: M.text, fontWeight: 600 }}>1 247 prestataires</strong> · 6 résultats correspondent à tes filtres</div>
              </div>
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <span style={{ fontSize: 12, color: M.textMute }}>Tri</span>
                <button style={{ background: '#fff', border: `1px solid ${M.border}`, padding: '7px 12px', fontSize: 12, fontWeight: 500, borderRadius: 4, display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>Pertinence <I_M.ChevronDown size={12} /></button>
              </div>
            </div>

            {/* Active filter chips */}
            <div style={{ display: 'flex', gap: 6, marginBottom: 20, flexWrap: 'wrap' }}>
              {['Disponible maintenant', 'Hybride', 'Identité KYC', 'Développement', 'Design & UI/UX', '400–950 €/j'].map((c, i) => (
                <span key={i} style={{ background: '#fff', border: `1px solid ${M.border}`, padding: '5px 10px', fontSize: 12, fontWeight: 500, borderRadius: 4, display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                  {c} <span style={{ color: M.textMute, cursor: 'pointer' }}>×</span>
                </span>
              ))}
            </div>

            {/* Grid */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 14 }}>
              {freelancers.map((f, i) => (
                <div key={i} style={{ background: '#fff', border: `1px solid ${M.border}`, borderRadius: 4, padding: 20, display: 'flex', gap: 14, position: 'relative' }}>
                  <button style={{ position: 'absolute', top: 14, right: 14, background: 'none', border: 'none', cursor: 'pointer', color: M.textMute }}><I_M.Bookmark size={16} /></button>
                  <div style={{ position: 'relative', flexShrink: 0 }}>
                    {f.photo ? (
                      <div style={{ width: 56, height: 56, borderRadius: '50%', background: `linear-gradient(135deg, ${f.pBg}, ${f.pBg}cc)`, position: 'relative', overflow: 'hidden' }}>
                        <div style={{ position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.15)' }} />
                        <div style={{ position: 'absolute', bottom: 8, left: 0, right: 0, textAlign: 'center', color: '#fff', fontSize: 18, fontWeight: 600, fontFamily: M.serif, fontStyle: 'italic' }}>{f.p}</div>
                      </div>
                    ) : (
                      <div style={{ width: 56, height: 56, borderRadius: '50%', background: f.pBg, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 18 }}>{f.p}</div>
                    )}
                    {f.avail === 'Disponible' && <span style={{ position: 'absolute', bottom: 0, right: 0, width: 14, height: 14, borderRadius: '50%', background: M.green, border: '2px solid #fff' }} />}
                  </div>

                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
                      <span style={{ fontSize: 15, fontWeight: 600 }}>{f.name}</span>
                      {f.verified && <I_M.Verified size={14} style={{ color: M.accent }} />}
                    </div>
                    <div style={{ fontSize: 13, color: M.textMute, marginBottom: 8 }}>{f.title}</div>
                    <div style={{ display: 'flex', gap: 14, fontSize: 12, color: M.textMute, marginBottom: 10, alignItems: 'center' }}>
                      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><I_M.MapPin size={11} /> {f.loc}</span>
                      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><I_M.Star size={11} fill="currentColor" style={{ color: M.amber }} /> <strong style={{ color: M.text, fontWeight: 600 }}>{f.rating}</strong> · {f.reviews}</span>
                      <span>{f.exp}</span>
                    </div>
                    <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginBottom: 12 }}>
                      {f.tags.map((t, ti) => (
                        <span key={ti} style={{ fontSize: 11, padding: '2px 7px', background: M.bg, border: `1px solid ${M.border}`, borderRadius: 2, color: M.textMute }}>{t}</span>
                      ))}
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderTop: `1px solid ${M.border}`, paddingTop: 10, marginTop: 4 }}>
                      <div>
                        <span style={{ fontFamily: M.serif, fontSize: 18, fontWeight: 400 }}>{f.tjm}</span>
                        <span style={{ fontSize: 11, color: M.textMute, marginLeft: 4 }}>/ jour</span>
                      </div>
                      <div style={{ fontSize: 11, color: f.avail === 'Disponible' ? M.green : M.amber, fontWeight: 600 }}>{f.avail}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ PROFILE ═══════════════════════════════════════════════════════
function MaisonProfile() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: M.bg, fontFamily: M.sans, color: M.text }}>
      <M_Sidebar active="profile" role="freelancer" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <M_Topbar />
        <div style={{ flex: 1, overflow: 'auto' }}>
          {/* Hero */}
          <div style={{ padding: '40px 48px 0', borderBottom: `1px solid ${M.border}` }}>
            <div style={{ display: 'flex', gap: 32, alignItems: 'flex-start', marginBottom: 32 }}>
              <div style={{ width: 132, height: 132, borderRadius: 4, background: 'linear-gradient(135deg,#0e8a5f,#1f8a4a)', position: 'relative', overflow: 'hidden', flexShrink: 0 }}>
                <div style={{ position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.1)' }} />
                <div style={{ position: 'absolute', bottom: 18, left: 0, right: 0, textAlign: 'center', color: '#fff', fontSize: 56, fontFamily: M.serif, fontStyle: 'italic', fontWeight: 400 }}>EM</div>
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                  <span style={{ fontFamily: M.mono, fontSize: 11, color: M.textMute, letterSpacing: '0.12em', textTransform: 'uppercase' }}>Prestataire · ID 02384</span>
                  <span style={{ width: 4, height: 4, borderRadius: '50%', background: M.textSubtle }} />
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontSize: 11, color: M.green, fontWeight: 600 }}><span style={{ width: 6, height: 6, borderRadius: '50%', background: M.green }} /> Disponible immédiatement</span>
                </div>
                <h1 style={{ fontFamily: M.serif, fontSize: 56, lineHeight: 1, margin: 0, marginBottom: 12, fontWeight: 400, letterSpacing: '-0.02em' }}>
                  Élise Marchand <I_M.Verified size={28} style={{ color: M.accent, verticalAlign: 'middle', marginLeft: 4 }} />
                </h1>
                <div style={{ fontSize: 18, color: M.textMute, marginBottom: 16, fontFamily: M.serif, fontStyle: 'italic' }}>UX Designer & Brand pour startups B2B</div>
                <div style={{ display: 'flex', gap: 22, fontSize: 13, color: M.textMute, alignItems: 'center', marginBottom: 16 }}>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><I_M.MapPin size={14} /> Paris · télétravail OK</span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><I_M.Globe size={14} /> Français, English</span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><I_M.Clock size={14} /> Répond en ~2h</span>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><I_M.Star size={14} fill="currentColor" style={{ color: M.amber }} /> <strong style={{ color: M.text, fontWeight: 600 }}>4,9</strong> sur 47 avis</span>
                </div>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8, alignItems: 'flex-end' }}>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontFamily: M.serif, fontSize: 36, fontWeight: 400, lineHeight: 1, letterSpacing: '-0.02em' }}>650 €<span style={{ fontSize: 18, color: M.textMute }}>/j</span></div>
                  <div style={{ fontSize: 11, color: M.textMute, marginTop: 4 }}>négociable selon scope</div>
                </div>
                <div style={{ display: 'flex', gap: 6 }}>
                  <button style={{ background: '#fff', border: `1px solid ${M.border}`, padding: '10px 14px', fontSize: 13, fontWeight: 500, borderRadius: 4, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}><I_M.Bookmark size={14} /> Sauvegarder</button>
                  <button style={{ background: M.text, color: '#fff', border: 'none', padding: '10px 16px', fontSize: 13, fontWeight: 600, borderRadius: 4, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}><I_M.Send size={14} /> Contacter</button>
                </div>
              </div>
            </div>

            {/* Tabs */}
            <div style={{ display: 'flex', gap: 28, fontSize: 13, fontWeight: 500 }}>
              {['Aperçu', 'Réalisations', 'Avis (47)', 'Tarification', 'À propos'].map((t, i) => (
                <div key={i} style={{ padding: '12px 0', borderBottom: i === 0 ? `2px solid ${M.text}` : '2px solid transparent', color: i === 0 ? M.text : M.textMute, cursor: 'pointer' }}>{t}</div>
              ))}
            </div>
          </div>

          {/* Content */}
          <div style={{ padding: '32px 48px', display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 32 }}>
            <div>
              {/* About */}
              <section style={{ marginBottom: 36 }}>
                <h2 style={{ fontFamily: M.serif, fontSize: 28, margin: 0, marginBottom: 14, fontWeight: 400, letterSpacing: '-0.01em' }}>À propos</h2>
                <p style={{ fontSize: 15, lineHeight: 1.7, color: '#1a1a1a', margin: 0, marginBottom: 12, textWrap: 'pretty' }}>
                  J'accompagne les startups B2B dans la conception de produits SaaS clairs, utiles, et au goût du jour. Huit ans d'expérience entre Paris et Berlin, avec un faible pour les <em style={{ color: M.accent }}>fintech, healthtech et marketplaces</em>. Mes clients réguliers : Qonto, Lydia, Memo Bank.
                </p>
                <p style={{ fontSize: 15, lineHeight: 1.7, color: '#1a1a1a', margin: 0, textWrap: 'pretty' }}>
                  J'interviens sur les phases de discovery, design system, design ops, et accompagne les équipes produit dans la durée — généralement <strong>3 à 6 mois</strong>, en mode 3-4 jours par semaine.
                </p>
              </section>

              {/* Skills */}
              <section style={{ marginBottom: 36 }}>
                <h2 style={{ fontFamily: M.serif, fontSize: 28, margin: 0, marginBottom: 16, fontWeight: 400 }}>Compétences</h2>
                <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                  {[['Figma', 95], ['Design System', 90], ['UX Research', 85], ['Brand Identity', 80], ['Webflow', 70], ['Framer', 65], ['Notion', 60], ['Protopie', 55]].map(([s, level], i) => (
                    <span key={i} style={{ fontSize: 13, padding: '6px 12px', background: '#fff', border: `1px solid ${M.border}`, borderRadius: 4, fontWeight: 500 }}>
                      {s} <span style={{ fontFamily: M.mono, fontSize: 10, color: M.textMute, marginLeft: 4 }}>{level}</span>
                    </span>
                  ))}
                </div>
              </section>

              {/* Portfolio */}
              <section style={{ marginBottom: 36 }}>
                <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', marginBottom: 16 }}>
                  <h2 style={{ fontFamily: M.serif, fontSize: 28, margin: 0, fontWeight: 400 }}>Sélection de réalisations</h2>
                  <a style={{ fontSize: 12, color: M.accent, fontWeight: 600, cursor: 'pointer' }}>Voir tout (24) →</a>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 14 }}>
                  {[
                    { title: 'Qonto Cards — refonte', client: 'Qonto · 2025', tag: 'Fintech', g1: '#3a4ee0', g2: '#7c8df0' },
                    { title: 'Memo Bank — design system v2', client: 'Memo Bank · 2025', tag: 'Banking', g1: '#0e8a5f', g2: '#5fb88a' },
                    { title: 'Lydia Pro onboarding', client: 'Lydia · 2024', tag: 'Mobile', g1: '#e8447b', g2: '#f47ea4' },
                    { title: 'Doctolib Pro dashboard', client: 'Doctolib · 2024', tag: 'Healthtech', g1: '#b8721d', g2: '#d9a05c' },
                  ].map((p, i) => (
                    <div key={i} style={{ background: '#fff', border: `1px solid ${M.border}`, borderRadius: 4, overflow: 'hidden' }}>
                      <div style={{ height: 140, background: `linear-gradient(135deg, ${p.g1}, ${p.g2})`, position: 'relative' }}>
                        <span style={{ position: 'absolute', top: 12, left: 12, fontSize: 10, padding: '3px 8px', background: 'rgba(255,255,255,0.95)', borderRadius: 2, fontWeight: 600, letterSpacing: '0.05em', textTransform: 'uppercase' }}>{p.tag}</span>
                      </div>
                      <div style={{ padding: 14 }}>
                        <div style={{ fontSize: 14, fontWeight: 500, marginBottom: 4 }}>{p.title}</div>
                        <div style={{ fontSize: 12, color: M.textMute }}>{p.client}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </section>

              {/* Review */}
              <section>
                <h2 style={{ fontFamily: M.serif, fontSize: 28, margin: 0, marginBottom: 16, fontWeight: 400 }}>Ce que disent les clients</h2>
                <div style={{ background: '#fff', border: `1px solid ${M.border}`, borderRadius: 4, padding: 24 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 14 }}>
                    <div style={{ width: 40, height: 40, borderRadius: '50%', background: '#1f2caa', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 13 }}>SA</div>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: 14, fontWeight: 600 }}>Sophie Aubry · CPO chez Qonto</div>
                      <div style={{ fontSize: 12, color: M.textMute }}>Mission de 4 mois · janvier 2025</div>
                    </div>
                    <div style={{ display: 'flex', gap: 1 }}>
                      {[1,2,3,4,5].map(s => <I_M.Star key={s} size={14} fill="currentColor" style={{ color: M.amber }} />)}
                    </div>
                  </div>
                  <div style={{ fontFamily: M.serif, fontSize: 19, lineHeight: 1.5, color: M.text, fontStyle: 'italic', textWrap: 'pretty' }}>
                    "Élise a posé un cadre méthodo dès la première semaine. On est passés d'un design system fragmenté à une vraie cohésion produit. Je referai appel à elle sans hésiter."
                  </div>
                </div>
              </section>
            </div>

            {/* Aside */}
            <div>
              <div style={{ background: '#fff', border: `1px solid ${M.border}`, borderRadius: 4, padding: 20, marginBottom: 16 }}>
                <div style={{ fontSize: 11, letterSpacing: '0.12em', textTransform: 'uppercase', color: M.textMute, fontWeight: 600, marginBottom: 14 }}>Vérifications</div>
                {[
                  ['Identité KYC', true],
                  ['Email professionnel', true],
                  ['Numéro SIRET', true],
                  ['Top 5% catégorie', true],
                  ['Assurance RC Pro', false],
                ].map(([l, ok], i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '7px 0', fontSize: 13, borderBottom: i < 4 ? `1px solid ${M.border}` : 'none' }}>
                    {ok ? <I_M.CheckCircle size={15} style={{ color: M.green }} /> : <span style={{ width: 15, height: 15, borderRadius: '50%', border: `1.5px solid ${M.borderStrong}` }} />}
                    <span style={{ color: ok ? M.text : M.textMute }}>{l}</span>
                  </div>
                ))}
              </div>

              <div style={{ background: '#fff', border: `1px solid ${M.border}`, borderRadius: 4, padding: 20, marginBottom: 16 }}>
                <div style={{ fontSize: 11, letterSpacing: '0.12em', textTransform: 'uppercase', color: M.textMute, fontWeight: 600, marginBottom: 14 }}>Statistiques</div>
                {[
                  ['Missions complétées', '47'],
                  ['Volume facturé', '312 k€'],
                  ['Taux de réembauche', '68%'],
                  ['Membre depuis', 'mars 2021'],
                ].map(([l, v], i) => (
                  <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '7px 0', fontSize: 13, borderBottom: i < 3 ? `1px solid ${M.border}` : 'none' }}>
                    <span style={{ color: M.textMute }}>{l}</span>
                    <span style={{ fontWeight: 600 }}>{v}</span>
                  </div>
                ))}
              </div>

              <div style={{ background: M.accentSoft, border: `1px solid ${M.accent}30`, borderRadius: 4, padding: 20 }}>
                <div style={{ fontFamily: M.serif, fontSize: 18, marginBottom: 6, fontStyle: 'italic' }}>Recommandée par Atelier</div>
                <div style={{ fontSize: 12, color: M.textMute, lineHeight: 1.5, marginBottom: 12 }}>Profil correspondant à 92% à votre dernier brief "Refonte produit B2B".</div>
                <button style={{ background: M.accent, color: '#fff', border: 'none', padding: '8px 14px', fontSize: 12, fontWeight: 600, borderRadius: 3, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 6 }}>Inviter sur un job <I_M.ArrowRight size={12} /></button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ═══ MESSAGES ══════════════════════════════════════════════════════
function MaisonMessages() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: M.bg, fontFamily: M.sans, color: M.text }}>
      <M_Sidebar active="msg" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <M_Topbar search="Rechercher dans les messages..." />
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex' }}>
          {/* Conversation list */}
          <div style={{ width: 320, borderRight: `1px solid ${M.border}`, display: 'flex', flexDirection: 'column', flexShrink: 0 }}>
            <div style={{ padding: '20px 20px 14px', borderBottom: `1px solid ${M.border}` }}>
              <h2 style={{ fontFamily: M.serif, fontSize: 26, margin: 0, fontWeight: 400, marginBottom: 12 }}>Messages</h2>
              <div style={{ display: 'flex', gap: 4, fontSize: 12 }}>
                {[['Tous', true, 12], ['Non lus', false, 3], ['Archivés', false, 0]].map(([l, a, n], i) => (
                  <button key={i} style={{ background: a ? M.text : 'transparent', color: a ? '#fff' : M.textMute, border: a ? 'none' : `1px solid ${M.border}`, padding: '5px 10px', borderRadius: 3, fontWeight: 500, fontSize: 12, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}>
                    {l} {n > 0 && <span style={{ fontSize: 10, opacity: 0.7 }}>{n}</span>}
                  </button>
                ))}
              </div>
            </div>
            <div style={{ flex: 1, overflow: 'auto' }}>
              {[
                { name: 'Élise Marchand', last: 'Tu as vu le brief mis à jour ?', time: '14 min', unread: true, p: 'EM', pBg: '#0e8a5f', tag: 'Mission · Refonte site', active: true },
                { name: 'Julien Petit', last: 'Voici la v2 des wireframes —', time: '1 h', unread: true, p: 'JP', pBg: '#e85d4a', tag: 'Brand identity Q2' },
                { name: 'Théo Renaud', last: 'Audit terminé, rapport ci-joint', time: '3 h', unread: false, p: 'TR', pBg: '#3a4ee0', tag: 'Audit SEO' },
                { name: 'Camille Dubois', last: 'Disponible la semaine prochaine ?', time: 'Hier', unread: false, p: 'CD', pBg: '#b8721d', tag: 'Discovery' },
                { name: 'Mehdi Bensalem', last: 'Merci pour la confirmation', time: 'Mar.', unread: false, p: 'MB', pBg: '#1f2caa', tag: 'Audit data' },
              ].map((c, i) => (
                <div key={i} style={{ padding: '14px 20px', borderBottom: `1px solid ${M.border}`, display: 'flex', gap: 12, cursor: 'pointer', background: c.active ? '#f6f3ec' : 'transparent', borderLeft: c.active ? `2px solid ${M.text}` : '2px solid transparent' }}>
                  <div style={{ width: 36, height: 36, borderRadius: '50%', background: c.pBg, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 12, flexShrink: 0 }}>{c.p}</div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                      <span style={{ fontSize: 13, fontWeight: c.unread ? 600 : 500 }}>{c.name}</span>
                      <span style={{ fontSize: 10, color: M.textMute, fontFamily: M.mono }}>{c.time}</span>
                    </div>
                    <div style={{ fontSize: 12, color: M.textMute, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', marginBottom: 4 }}>{c.last}</div>
                    <div style={{ fontSize: 10, color: M.accent, fontWeight: 600, letterSpacing: '0.05em', textTransform: 'uppercase' }}>{c.tag}</div>
                  </div>
                  {c.unread && <span style={{ width: 7, height: 7, borderRadius: '50%', background: M.accent, marginTop: 6 }} />}
                </div>
              ))}
            </div>
          </div>

          {/* Active conversation */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
            {/* Header */}
            <div style={{ padding: '14px 24px', borderBottom: `1px solid ${M.border}`, display: 'flex', alignItems: 'center', gap: 14, background: '#fff' }}>
              <div style={{ width: 40, height: 40, borderRadius: '50%', background: '#0e8a5f', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 13 }}>EM</div>
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={{ fontSize: 15, fontWeight: 600 }}>Élise Marchand</span>
                  <I_M.Verified size={13} style={{ color: M.accent }} />
                </div>
                <div style={{ fontSize: 12, color: M.green, display: 'flex', alignItems: 'center', gap: 5 }}><span style={{ width: 6, height: 6, borderRadius: '50%', background: M.green }} /> En ligne · UX Designer · Paris</div>
              </div>
              <button style={{ background: 'none', border: `1px solid ${M.border}`, padding: 8, borderRadius: 4, cursor: 'pointer' }}><I_M.Phone size={15} /></button>
              <button style={{ background: 'none', border: `1px solid ${M.border}`, padding: 8, borderRadius: 4, cursor: 'pointer' }}><I_M.Video size={15} /></button>
              <button style={{ background: M.text, color: '#fff', border: 'none', padding: '8px 14px', fontSize: 12, fontWeight: 600, borderRadius: 4, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
                <I_M.Briefcase size={13} /> Démarrer un projet
              </button>
            </div>

            {/* Messages */}
            <div style={{ flex: 1, overflow: 'auto', padding: '20px 24px', display: 'flex', flexDirection: 'column', gap: 12, background: M.bg }}>
              <div style={{ textAlign: 'center', fontSize: 11, color: M.textMute, fontFamily: M.mono, letterSpacing: '0.1em', textTransform: 'uppercase' }}>Aujourd'hui</div>

              {/* Their msg */}
              <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end', maxWidth: '70%' }}>
                <div style={{ width: 26, height: 26, borderRadius: '50%', background: '#0e8a5f', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 10, flexShrink: 0 }}>EM</div>
                <div style={{ background: '#fff', border: `1px solid ${M.border}`, padding: '10px 14px', borderRadius: '4px 12px 12px 12px', fontSize: 14, lineHeight: 1.5 }}>Salut Nova ! Tu as eu le temps de regarder le brief mis à jour pour la v2 ? J'ai ajouté la section onboarding mobile.</div>
              </div>

              {/* My msg */}
              <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                <div style={{ background: M.text, color: '#fff', padding: '10px 14px', borderRadius: '12px 4px 12px 12px', fontSize: 14, maxWidth: '70%', lineHeight: 1.5 }}>Oui ! Très clair, j'aime beaucoup la nouvelle approche modulaire.</div>
              </div>

              {/* Proposal card */}
              <div style={{ background: '#fff', border: `1px solid ${M.accent}40`, borderRadius: 4, padding: 0, alignSelf: 'flex-start', maxWidth: 460, marginTop: 8, overflow: 'hidden' }}>
                <div style={{ padding: '12px 18px', background: M.accentSoft, borderBottom: `1px solid ${M.accent}30`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <I_M.Briefcase size={14} style={{ color: M.accent }} />
                    <span style={{ fontSize: 12, fontWeight: 600, color: M.accent, letterSpacing: '0.05em', textTransform: 'uppercase' }}>Proposition de mission</span>
                  </div>
                  <span style={{ fontSize: 10, fontFamily: M.mono, color: M.textMute }}>#PROP-1247</span>
                </div>
                <div style={{ padding: 18 }}>
                  <div style={{ fontFamily: M.serif, fontSize: 22, lineHeight: 1.2, marginBottom: 6, fontWeight: 400 }}>Refonte produit Nova v2</div>
                  <div style={{ fontSize: 13, color: M.textMute, marginBottom: 14, lineHeight: 1.5 }}>Refonte UX du parcours d'onboarding mobile + révision du design system existant. 3 mois, 3 jours/semaine.</div>
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12, padding: '12px 0', borderTop: `1px solid ${M.border}`, borderBottom: `1px solid ${M.border}`, marginBottom: 14 }}>
                    <div>
                      <div style={{ fontSize: 10, color: M.textMute, letterSpacing: '0.1em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 4 }}>Montant</div>
                      <div style={{ fontFamily: M.serif, fontSize: 22, lineHeight: 1, fontWeight: 400 }}>23 400 €</div>
                    </div>
                    <div>
                      <div style={{ fontSize: 10, color: M.textMute, letterSpacing: '0.1em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 4 }}>Durée</div>
                      <div style={{ fontFamily: M.serif, fontSize: 22, lineHeight: 1, fontWeight: 400 }}>3 mois</div>
                    </div>
                    <div>
                      <div style={{ fontSize: 10, color: M.textMute, letterSpacing: '0.1em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 4 }}>Démarrage</div>
                      <div style={{ fontFamily: M.serif, fontSize: 22, lineHeight: 1, fontWeight: 400 }}>15 mai</div>
                    </div>
                  </div>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <button style={{ flex: 1, background: M.text, color: '#fff', border: 'none', padding: '10px', fontSize: 13, fontWeight: 600, borderRadius: 3, cursor: 'pointer' }}>Accepter</button>
                    <button style={{ background: '#fff', color: M.text, border: `1px solid ${M.border}`, padding: '10px 14px', fontSize: 13, fontWeight: 500, borderRadius: 3, cursor: 'pointer' }}>Négocier</button>
                    <button style={{ background: 'none', border: 'none', color: M.textMute, padding: '10px', fontSize: 13, cursor: 'pointer' }}>Voir détail</button>
                  </div>
                </div>
              </div>

              {/* Their msg */}
              <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end', maxWidth: '70%', marginTop: 8 }}>
                <div style={{ width: 26, height: 26, borderRadius: '50%', background: '#0e8a5f', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 600, fontSize: 10, flexShrink: 0 }}>EM</div>
                <div style={{ background: '#fff', border: `1px solid ${M.border}`, padding: '10px 14px', borderRadius: '4px 12px 12px 12px', fontSize: 14, lineHeight: 1.5 }}>J'attends ton retour. Si OK je bloque mon planning dès lundi 👌</div>
              </div>

              <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end', maxWidth: '70%' }}>
                <div style={{ width: 26, flexShrink: 0 }} />
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: M.textMute, fontSize: 12 }}>
                  <span style={{ display: 'flex', gap: 3 }}>
                    {[0,1,2].map(i => <span key={i} style={{ width: 6, height: 6, borderRadius: '50%', background: M.textMute, opacity: 0.4 + i * 0.2 }} />)}
                  </span>
                  Élise écrit...
                </div>
              </div>
            </div>

            {/* Composer */}
            <div style={{ borderTop: `1px solid ${M.border}`, padding: '14px 24px', background: '#fff' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, border: `1px solid ${M.border}`, borderRadius: 4, padding: '8px 12px' }}>
                <button style={{ background: 'none', border: 'none', color: M.textMute, cursor: 'pointer', padding: 4 }}><I_M.Paperclip size={16} /></button>
                <button style={{ background: 'none', border: 'none', color: M.textMute, cursor: 'pointer', padding: 4 }}><I_M.Briefcase size={16} /></button>
                <input placeholder="Écrire un message..." style={{ flex: 1, border: 'none', outline: 'none', fontSize: 14, padding: '6px 0', fontFamily: M.sans }} />
                <button style={{ background: M.text, color: '#fff', border: 'none', padding: '7px 12px', borderRadius: 3, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, fontWeight: 600 }}><I_M.Send size={13} /> Envoyer</button>
              </div>
              <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                {['Envoyer une proposition', 'Demander un devis', 'Partager des fichiers'].map((t, i) => (
                  <button key={i} style={{ background: 'transparent', border: `1px solid ${M.border}`, padding: '5px 10px', fontSize: 11, fontWeight: 500, borderRadius: 3, color: M.textMute, cursor: 'pointer' }}>{t}</button>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

window.MaisonFind = MaisonFind;
window.MaisonProfile = MaisonProfile;
window.MaisonMessages = MaisonMessages;
