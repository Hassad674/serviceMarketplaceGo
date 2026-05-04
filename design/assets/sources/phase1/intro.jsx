function Intro() {
  return (
    <div style={{ width: '100%', height: '100%', background: '#faf7f1', padding: '80px 96px', fontFamily: 'Inter Tight, sans-serif', overflow: 'hidden', position: 'relative' }}>
      {/* Decorative number */}
      <div style={{ position: 'absolute', top: 40, right: 64, fontFamily: 'Instrument Serif, serif', fontSize: 220, lineHeight: 1, color: 'rgba(186, 99, 64, 0.08)', fontStyle: 'italic' }}>00</div>

      <div style={{ fontSize: 11, letterSpacing: '0.2em', textTransform: 'uppercase', color: '#7a6f60', marginBottom: 24 }}>Brief — phase 1</div>
      <h1 style={{ fontFamily: 'Instrument Serif, serif', fontSize: 72, lineHeight: 1.05, margin: 0, marginBottom: 32, fontWeight: 400, maxWidth: 900, letterSpacing: '-0.02em' }}>
        Trois directions pour <em style={{ color: '#ba6340' }}>Atelier</em>.<br/>
        Une seule sera déclinée.
      </h1>
      <p style={{ fontSize: 18, lineHeight: 1.6, color: '#3d3528', maxWidth: 720, margin: 0, marginBottom: 48 }}>
        Marketplace freelance / agences / apporteurs / entreprises. Le backend est solide, l'identité visuelle ne suit pas. On explore ici 3 territoires de marque sur 4 écrans clés. Tu choisis, on décline le reste.
      </p>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 32, marginBottom: 56 }}>
        {[
          { num: '01', name: 'Maison', tag: 'Stripe-like, structuré', body: 'Sérieux pro, accent indigo profond, grille rigoureuse. La direction qui inspire confiance et fait penser fintech B2B mature.', accent: '#3a4ee0', bg: '#f3f3ff' },
          { num: '02', name: 'Soleil', tag: 'Airbnb-like, humain', body: 'Photo-driven, palette corail/sable, typo généreuse. La plus chaleureuse — pour faire venir aussi les freelances créatifs et les TPE.', accent: '#e85d4a', bg: '#fff4ef' },
          { num: '03', name: 'Place', tag: 'Upwork-like, marketplace', body: 'Dense, riche, orienté volume. Cards efficaces, badges, données partout. Pour assumer le côté marketplace pro.', accent: '#0e8a5f', bg: '#eef9f4' },
        ].map((d) => (
          <div key={d.num} style={{ background: d.bg, padding: 28, border: '1px solid rgba(0,0,0,0.06)', borderRadius: 4 }}>
            <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', marginBottom: 16 }}>
              <span style={{ fontFamily: 'Geist Mono, monospace', fontSize: 12, color: '#7a6f60', letterSpacing: '0.1em' }}>{d.num}</span>
              <span style={{ width: 8, height: 8, borderRadius: '50%', background: d.accent }}></span>
            </div>
            <div style={{ fontFamily: 'Instrument Serif, serif', fontSize: 36, lineHeight: 1, marginBottom: 8, fontStyle: 'italic' }}>{d.name}</div>
            <div style={{ fontSize: 12, color: d.accent, fontWeight: 500, marginBottom: 14, letterSpacing: '0.05em', textTransform: 'uppercase' }}>{d.tag}</div>
            <div style={{ fontSize: 14, lineHeight: 1.55, color: '#3d3528' }}>{d.body}</div>
          </div>
        ))}
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 64, paddingTop: 32, borderTop: '1px solid rgba(0,0,0,0.08)' }}>
        <div>
          <div style={{ fontSize: 11, letterSpacing: '0.2em', textTransform: 'uppercase', color: '#7a6f60', marginBottom: 14 }}>Ce qu'on explore</div>
          <ul style={{ fontSize: 15, lineHeight: 1.8, margin: 0, paddingLeft: 18, color: '#1a1a1a' }}>
            <li>Dashboard <span style={{ color: '#7a6f60' }}>— l'écran qui donne le ton</span></li>
            <li>Find Freelancers <span style={{ color: '#7a6f60' }}>— la page la plus fréquentée</span></li>
            <li>Profil prestataire <span style={{ color: '#7a6f60' }}>— vitrine d'un freelance</span></li>
            <li>Messages + proposal <span style={{ color: '#7a6f60' }}>— interaction métier centrale</span></li>
          </ul>
        </div>
        <div>
          <div style={{ fontSize: 11, letterSpacing: '0.2em', textTransform: 'uppercase', color: '#7a6f60', marginBottom: 14 }}>Phase 2 (après ton choix)</div>
          <ul style={{ fontSize: 15, lineHeight: 1.8, margin: 0, paddingLeft: 18, color: '#1a1a1a' }}>
            <li>Toutes les pages restantes <span style={{ color: '#7a6f60' }}>— jobs, projets, wallet, account…</span></li>
            <li>Mobile + desktop responsive</li>
            <li>Landing publique + auth</li>
            <li>Tweaks (couleur, typo, densité, radius, sidebar)</li>
            <li>Guide d'implémentation Next.js</li>
          </ul>
        </div>
      </div>

      <div style={{ position: 'absolute', bottom: 40, right: 64, fontFamily: 'Geist Mono, monospace', fontSize: 11, color: '#7a6f60', letterSpacing: '0.1em' }}>
        ATELIER · MARKETPLACE · 2026
      </div>
    </div>
  );
}

window.Intro = Intro;
