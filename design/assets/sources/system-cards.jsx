// System overview cards — typography, color, components for each direction

function SystemCardEditorial() {
  return (
    <div style={{
      width: '100%', height: '100%',
      background: '#F5F1EA',
      color: '#1A1612',
      padding: '48px',
      fontFamily: "'Geist', sans-serif",
      display: 'grid',
      gridTemplateColumns: '1fr 1fr',
      gap: '40px',
      overflow: 'hidden',
    }}>
      {/* LEFT — Identity */}
      <div>
        <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, letterSpacing: '0.12em', textTransform: 'uppercase', color: '#857A6A', marginBottom: 24 }}>
          A · Direction Éditoriale
        </div>
        <div style={{ fontFamily: "'Instrument Serif', serif", fontSize: 84, lineHeight: 0.95, fontWeight: 400, letterSpacing: '-0.02em' }}>
          Atelier<span style={{ color: '#C2410C', fontStyle: 'italic' }}>.</span>
        </div>
        <div style={{ fontFamily: "'Instrument Serif', serif", fontStyle: 'italic', fontSize: 22, color: '#594D3D', marginTop: 12, marginBottom: 32 }}>
          Le travail, à sa juste valeur.
        </div>

        <div style={{ borderTop: '1px solid #1A1612', paddingTop: 20, marginTop: 24 }}>
          <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 10, letterSpacing: '0.12em', textTransform: 'uppercase', color: '#857A6A', marginBottom: 14 }}>Typographie</div>
          <div style={{ fontFamily: "'Instrument Serif', serif", fontSize: 40, lineHeight: 1, marginBottom: 4 }}>Instrument Serif</div>
          <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, color: '#857A6A', marginBottom: 18 }}>display · titres · accents éditoriaux</div>
          <div style={{ fontFamily: "'Geist', sans-serif", fontSize: 26, fontWeight: 500, lineHeight: 1, marginBottom: 4 }}>Geist Sans</div>
          <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, color: '#857A6A', marginBottom: 18 }}>UI · texte · labels</div>
          <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 18, fontWeight: 500 }}>Geist Mono</div>
          <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, color: '#857A6A' }}>chiffres · TJM · IDs · timestamps</div>
        </div>
      </div>

      {/* RIGHT — Color & Components */}
      <div>
        <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 10, letterSpacing: '0.12em', textTransform: 'uppercase', color: '#857A6A', marginBottom: 14 }}>Palette</div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: 6, marginBottom: 28 }}>
          {[
            { c: '#F5F1EA', n: 'Ivoire', t: 'BG' },
            { c: '#EAE3D5', n: 'Sable', t: 'card' },
            { c: '#594D3D', n: 'Tabac', t: 'mute' },
            { c: '#1A1612', n: 'Encre', t: 'fg' },
            { c: '#C2410C', n: 'Rouille', t: 'accent' },
            { c: '#3F6B4F', n: 'Sapin', t: 'success' },
          ].map(s => (
            <div key={s.c}>
              <div style={{ height: 56, background: s.c, border: s.c === '#F5F1EA' ? '1px solid #1A1612' : 'none' }} />
              <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 9, color: '#1A1612', marginTop: 6, lineHeight: 1.3 }}>{s.n}</div>
              <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 9, color: '#857A6A' }}>{s.t}</div>
            </div>
          ))}
        </div>

        <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 10, letterSpacing: '0.12em', textTransform: 'uppercase', color: '#857A6A', marginBottom: 14 }}>Composants</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {/* Button */}
          <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
            <button style={{ background: '#1A1612', color: '#F5F1EA', border: 'none', padding: '12px 20px', borderRadius: 6, fontFamily: "'Geist', sans-serif", fontSize: 13, fontWeight: 500, letterSpacing: '0.01em' }}>Publier la mission</button>
            <button style={{ background: 'transparent', color: '#1A1612', border: '1px solid #1A1612', padding: '11px 19px', borderRadius: 6, fontFamily: "'Geist', sans-serif", fontSize: 13, fontWeight: 500 }}>Annuler</button>
            <button style={{ background: '#C2410C', color: '#F5F1EA', border: 'none', padding: '12px 20px', borderRadius: 6, fontFamily: "'Geist', sans-serif", fontSize: 13, fontWeight: 500 }}>+ Apporteur</button>
          </div>
          {/* Card sample */}
          <div style={{ background: '#FAF7F2', border: '1px solid #E5DDC9', borderRadius: 8, padding: '18px 20px' }}>
            <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between' }}>
              <div style={{ fontFamily: "'Instrument Serif', serif", fontSize: 22, fontWeight: 400 }}>Refonte d'identité</div>
              <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 16, color: '#1A1612' }}>3 213 €</div>
            </div>
            <div style={{ display: 'flex', gap: 8, marginTop: 10, alignItems: 'center' }}>
              <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#3F6B4F' }} />
              <span style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, color: '#594D3D', textTransform: 'uppercase', letterSpacing: '0.08em' }}>Mission active · échéance 15 juin</span>
            </div>
          </div>
          {/* Tags */}
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {['Design & UI/UX', 'Brand', 'Direction artistique', 'Webflow'].map(t => (
              <span key={t} style={{ fontFamily: "'Geist', sans-serif", fontSize: 12, padding: '5px 11px', border: '1px solid #1A1612', borderRadius: 999, color: '#1A1612' }}>{t}</span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function SystemCardStudio() {
  return (
    <div style={{
      width: '100%', height: '100%',
      background: '#0E0E0E',
      color: '#F2F2F0',
      padding: '48px',
      fontFamily: "'Inter Tight', sans-serif",
      display: 'grid',
      gridTemplateColumns: '1fr 1fr',
      gap: '40px',
      overflow: 'hidden',
    }}>
      {/* LEFT */}
      <div>
        <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, letterSpacing: '0.12em', textTransform: 'uppercase', color: '#7A7A78', marginBottom: 24 }}>
          B · Direction Studio
        </div>
        <div style={{ fontFamily: "'Inter Tight', sans-serif", fontSize: 84, lineHeight: 0.92, fontWeight: 600, letterSpacing: '-0.04em', display: 'flex', alignItems: 'center', gap: 14 }}>
          <span style={{ display: 'inline-block', width: 14, height: 14, background: '#D4FF3A', borderRadius: 0 }} />
          atelier
        </div>
        <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 13, color: '#A8A8A4', marginTop: 14, marginBottom: 32, letterSpacing: '0.02em' }}>
          /ɑtəlje/ — workspace for serious work
        </div>

        <div style={{ borderTop: '1px solid #2A2A28', paddingTop: 20, marginTop: 24 }}>
          <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 10, letterSpacing: '0.12em', textTransform: 'uppercase', color: '#7A7A78', marginBottom: 14 }}>Typography</div>
          <div style={{ fontFamily: "'Inter Tight', sans-serif", fontSize: 40, lineHeight: 1, fontWeight: 600, letterSpacing: '-0.03em', marginBottom: 4 }}>Inter Tight</div>
          <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, color: '#7A7A78', marginBottom: 18 }}>display · UI · everywhere</div>
          <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 26, fontWeight: 500, lineHeight: 1, marginBottom: 4 }}>Geist Mono</div>
          <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, color: '#7A7A78' }}>amounts · IDs · metadata · code</div>
        </div>
      </div>

      {/* RIGHT */}
      <div>
        <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 10, letterSpacing: '0.12em', textTransform: 'uppercase', color: '#7A7A78', marginBottom: 14 }}>Palette</div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: 6, marginBottom: 28 }}>
          {[
            { c: '#0E0E0E', n: 'Onyx', t: 'BG' },
            { c: '#1A1A1A', n: 'Soot', t: 'card' },
            { c: '#2A2A28', n: 'Iron', t: 'border' },
            { c: '#A8A8A4', n: 'Bone', t: 'mute' },
            { c: '#D4FF3A', n: 'Volt', t: 'accent' },
            { c: '#FF6B47', n: 'Ember', t: 'alert' },
          ].map(s => (
            <div key={s.c}>
              <div style={{ height: 56, background: s.c, border: s.c === '#0E0E0E' ? '1px solid #2A2A28' : 'none' }} />
              <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 9, color: '#F2F2F0', marginTop: 6, lineHeight: 1.3 }}>{s.n}</div>
              <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 9, color: '#7A7A78' }}>{s.t}</div>
            </div>
          ))}
        </div>

        <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 10, letterSpacing: '0.12em', textTransform: 'uppercase', color: '#7A7A78', marginBottom: 14 }}>Components</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
            <button style={{ background: '#D4FF3A', color: '#0E0E0E', border: 'none', padding: '12px 20px', borderRadius: 4, fontFamily: "'Inter Tight', sans-serif", fontSize: 13, fontWeight: 600, letterSpacing: '-0.01em' }}>Publish job →</button>
            <button style={{ background: 'transparent', color: '#F2F2F0', border: '1px solid #2A2A28', padding: '11px 19px', borderRadius: 4, fontFamily: "'Inter Tight', sans-serif", fontSize: 13, fontWeight: 500 }}>Cancel</button>
            <button style={{ background: '#1A1A1A', color: '#F2F2F0', border: '1px solid #2A2A28', padding: '11px 19px', borderRadius: 4, fontFamily: "'Inter Tight', sans-serif", fontSize: 13, fontWeight: 500 }}>+ Refer</button>
          </div>
          <div style={{ background: '#1A1A1A', border: '1px solid #2A2A28', borderRadius: 6, padding: '18px 20px' }}>
            <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between' }}>
              <div style={{ fontFamily: "'Inter Tight', sans-serif", fontSize: 18, fontWeight: 600, letterSpacing: '-0.02em' }}>Brand identity refresh</div>
              <div style={{ fontFamily: "'Geist Mono', monospace", fontSize: 16, color: '#D4FF3A' }}>€3,213</div>
            </div>
            <div style={{ display: 'flex', gap: 10, marginTop: 10, alignItems: 'center' }}>
              <span style={{ fontFamily: "'Geist Mono', monospace", fontSize: 10, padding: '3px 8px', background: '#0E0E0E', border: '1px solid #2A2A28', color: '#A8A8A4', borderRadius: 3, textTransform: 'uppercase', letterSpacing: '0.08em' }}>active</span>
              <span style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, color: '#7A7A78' }}>due 06.15.26 · MS-2891</span>
            </div>
          </div>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {['design/ui', 'brand', 'art-direction', 'webflow'].map(t => (
              <span key={t} style={{ fontFamily: "'Geist Mono', monospace", fontSize: 11, padding: '4px 9px', background: '#1A1A1A', border: '1px solid #2A2A28', borderRadius: 3, color: '#A8A8A4' }}>{t}</span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { SystemCardEditorial, SystemCardStudio });
