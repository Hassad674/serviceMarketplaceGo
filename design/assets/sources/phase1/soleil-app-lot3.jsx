// App Lot 3 — Argent : Portefeuille, Factures, Paiement détail
const SL3 = window.S;
const SL3I = window.SI;
const _AppFrame_L3 = window.AppFrame;
const _AppTabBar_L3 = window.AppTabBar;
const SL3Portrait = window.Portrait;

// ─── Portefeuille ──────────────────────────────────────────────
function AppWallet() {
  return (
    <_AppFrame_L3>
      <div style={{ flexShrink: 0, padding: '6px 20px 14px', background: SL3.bg, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <div style={{ fontFamily: SL3.serif, fontSize: 26, fontWeight: 600, letterSpacing: '-0.02em', color: SL3.text }}>Portefeuille</div>
          <div style={{ fontSize: 12.5, color: SL3.textMute, fontFamily: SL3.serif, fontStyle: 'italic', marginTop: 2 }}>Vos revenus et virements</div>
        </div>
        <button style={{ width: 38, height: 38, borderRadius: '50%', background: '#fff', border: `1px solid ${SL3.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL3I name="Settings" size={17} />
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px', display: 'flex', flexDirection: 'column', gap: 14 }}>
        {/* Hero balance */}
        <div style={{ background: SL3.text, color: '#fff', borderRadius: 18, padding: 20, position: 'relative', overflow: 'hidden' }}>
          <div style={{ position: 'absolute', top: -50, right: -50, width: 180, height: 180, borderRadius: '50%', background: `radial-gradient(circle, rgba(232,93,74,0.35), transparent 70%)` }} />
          <div style={{ position: 'relative' }}>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.55)', letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase' }}>Solde disponible</div>
            <div style={{ fontFamily: SL3.serif, fontSize: 38, fontWeight: 600, letterSpacing: '-0.025em', marginTop: 4, lineHeight: 1 }}>3 240,00 €</div>
            <div style={{ display: 'flex', gap: 16, marginTop: 18, fontSize: 12 }}>
              <div>
                <div style={{ color: 'rgba(255,255,255,0.55)', fontSize: 10.5 }}>SÉQUESTRE</div>
                <div style={{ fontFamily: SL3.mono, fontWeight: 600, marginTop: 2 }}>4 800 €</div>
              </div>
              <div style={{ width: 1, background: 'rgba(255,255,255,0.15)' }} />
              <div>
                <div style={{ color: 'rgba(255,255,255,0.55)', fontSize: 10.5 }}>MOIS EN COURS</div>
                <div style={{ fontFamily: SL3.mono, fontWeight: 600, marginTop: 2 }}>4 200 €</div>
              </div>
            </div>
            <button style={{ marginTop: 18, width: '100%', padding: '11px', background: SL3.accent, color: '#fff', border: 'none', borderRadius: 12, fontSize: 13.5, fontWeight: 600, fontFamily: SL3.sans }}>
              Virer sur mon compte
            </button>
          </div>
        </div>

        {/* Stripe Connect status */}
        <div style={{ background: '#fff', border: `1px solid ${SL3.border}`, borderRadius: 14, padding: 14, display: 'flex', alignItems: 'center', gap: 11 }}>
          <div style={{ width: 40, height: 40, borderRadius: 12, background: SL3.greenSoft, display: 'flex', alignItems: 'center', justifyContent: 'center', color: SL3.green, flexShrink: 0 }}>
            <SL3I name="Shield" size={18} />
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: SL3.text }}>Stripe Connect actif</div>
            <div style={{ fontSize: 11, color: SL3.textMute, marginTop: 1 }}>IBAN ····4729 · Sociéte vérifiée</div>
          </div>
          <SL3I name="ChevronRight" size={16} />
        </div>

        {/* Transactions */}
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', padding: '6px 0 10px' }}>
            <div style={{ fontFamily: SL3.serif, fontSize: 17, fontWeight: 600, color: SL3.text }}>Mouvements</div>
            <span style={{ fontSize: 12, color: SL3.accent, fontWeight: 600 }}>Tout voir →</span>
          </div>
          <div style={{ background: '#fff', border: `1px solid ${SL3.border}`, borderRadius: 14, overflow: 'hidden' }}>
            {[
              { type: 'in', label: 'Jalon 1 · Refonte Helio', sub: 'reçu de Léa Bertrand', amount: '+ 2 400 €', date: '15 mai' },
              { type: 'out', label: 'Virement bancaire', sub: 'IBAN ····4729', amount: '− 1 800 €', date: '12 mai' },
              { type: 'in', label: 'Jalon 1 · Identité Verso', sub: 'reçu de Marie Lambert', amount: '+ 1 600 €', date: '8 mai' },
              { type: 'fee', label: 'Frais plateforme', sub: 'Mai 2024 · 5%', amount: '− 200 €', date: '5 mai' },
              { type: 'in', label: 'Jalon final · Pact', sub: 'reçu de Pact SAS', amount: '+ 3 200 €', date: '2 mai' },
            ].map((t, i, a) => {
              const colors = {
                'in': { bg: SL3.greenSoft, text: SL3.green, icon: 'ArrowDown' },
                'out': { bg: SL3.bg, text: SL3.text, icon: 'ArrowUp' },
                'fee': { bg: SL3.bg, text: SL3.textMute, icon: 'Pulse' },
              };
              const c = colors[t.type];
              return (
                <div key={i} style={{ display: 'flex', gap: 11, padding: '12px 14px', borderBottom: i < a.length - 1 ? `1px solid ${SL3.border}` : 'none', alignItems: 'center' }}>
                  <div style={{ width: 34, height: 34, borderRadius: 10, background: c.bg, color: c.text, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, fontFamily: SL3.serif, fontSize: 14, fontWeight: 700 }}>
                    {t.type === 'in' ? '↓' : t.type === 'out' ? '↑' : '·'}
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 13, fontWeight: 600, color: SL3.text }}>{t.label}</div>
                    <div style={{ fontSize: 11, color: SL3.textMute, marginTop: 1 }}>{t.sub} · {t.date}</div>
                  </div>
                  <div style={{ fontFamily: SL3.mono, fontSize: 13, color: t.type === 'in' ? SL3.green : SL3.text, fontWeight: 700 }}>{t.amount}</div>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <_AppTabBar_L3 active="wallet" />
    </_AppFrame_L3>
  );
}

// ─── Factures ──────────────────────────────────────────────────
function AppFactures() {
  return (
    <_AppFrame_L3>
      <div style={{ flexShrink: 0, padding: '6px 20px 14px', background: SL3.bg }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontFamily: SL3.serif, fontSize: 26, fontWeight: 600, letterSpacing: '-0.02em', color: SL3.text }}>Factures</div>
            <div style={{ fontSize: 12.5, color: SL3.textMute, fontFamily: SL3.serif, fontStyle: 'italic', marginTop: 2 }}>Émises automatiquement à chaque paiement</div>
          </div>
          <button style={{ width: 38, height: 38, borderRadius: '50%', background: '#fff', border: `1px solid ${SL3.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <SL3I name="Filter" size={16} />
          </button>
        </div>
      </div>

      {/* Sub tabs */}
      <div style={{ flexShrink: 0, padding: '0 20px 12px', background: SL3.bg, display: 'flex', gap: 6 }}>
        {[
          { l: 'Émises', n: 12, active: true },
          { l: 'Reçues', n: 4 },
          { l: 'Brouillons', n: 1 },
        ].map(t => (
          <span key={t.l} style={{ padding: '6px 12px', borderRadius: 999, fontSize: 12, fontWeight: 600, background: t.active ? SL3.text : '#fff', color: t.active ? '#fff' : SL3.textMute, border: t.active ? 'none' : `1px solid ${SL3.border}` }}>{t.l} <span style={{ opacity: 0.6 }}>{t.n}</span></span>
        ))}
      </div>

      {/* Mois en cours hero */}
      <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px' }}>
        <div style={{ background: '#fff', border: `1px solid ${SL3.border}`, borderRadius: 14, padding: 14, display: 'flex', alignItems: 'center', gap: 14 }}>
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 10.5, color: SL3.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase' }}>Mai 2024</div>
            <div style={{ fontFamily: SL3.serif, fontSize: 22, fontWeight: 600, color: SL3.text, marginTop: 2 }}>4 200 €</div>
            <div style={{ fontSize: 11, color: SL3.textMute, marginTop: 1 }}>3 factures émises</div>
          </div>
          <div style={{ width: 60, height: 60, borderRadius: 14, background: SL3.accentSoft, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <span style={{ fontFamily: SL3.serif, fontSize: 22, fontWeight: 600, color: SL3.accent }}>€</span>
          </div>
        </div>

        {/* Section ce mois */}
        <div style={{ fontSize: 11, color: SL3.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', margin: '20px 4px 10px' }}>Ce mois</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {[
            { num: 'F-2024-0042', client: 'Helio', proj: 'Refonte app · Jalon 1', amount: '2 400 €', date: '15 mai', paid: true },
            { num: 'F-2024-0041', client: 'Verso', proj: 'Identité visuelle · J1', amount: '1 600 €', date: '8 mai', paid: true },
            { num: 'F-2024-0040', client: 'Pact SAS', proj: 'UX writing · final', amount: '3 200 €', date: '2 mai', paid: true },
          ].map((f, i) => (
            <div key={i} style={{ background: '#fff', border: `1px solid ${SL3.border}`, borderRadius: 12, padding: 12, display: 'flex', alignItems: 'center', gap: 11 }}>
              <div style={{ width: 38, height: 46, borderRadius: 8, background: SL3.bg, border: `1px solid ${SL3.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <SL3I name="File" size={17} />
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                  <span style={{ fontFamily: SL3.mono, fontSize: 11, color: SL3.textMute, fontWeight: 600 }}>{f.num}</span>
                  {f.paid ? <span style={{ background: SL3.greenSoft, color: SL3.green, fontSize: 9, fontWeight: 700, padding: '1px 6px', borderRadius: 999 }}>PAYÉE</span> : null}
                </div>
                <div style={{ fontSize: 13, fontWeight: 600, color: SL3.text, marginTop: 2 }}>{f.client}</div>
                <div style={{ fontSize: 11, color: SL3.textMute, marginTop: 1 }}>{f.proj} · {f.date}</div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{ fontFamily: SL3.mono, fontSize: 13, color: SL3.text, fontWeight: 700 }}>{f.amount}</div>
                <div style={{ fontSize: 10, color: SL3.textMute, marginTop: 4 }}>↓ PDF</div>
              </div>
            </div>
          ))}
        </div>

        {/* Section avril */}
        <div style={{ fontSize: 11, color: SL3.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', margin: '20px 4px 10px' }}>Avril 2024</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {[
            { num: 'F-2024-0039', client: 'Trellis', proj: 'Audit produit', amount: '1 800 €', date: '28 avr', paid: true },
            { num: 'F-2024-0038', client: 'Atelier Mure', proj: 'Brand guidelines', amount: '2 200 €', date: '12 avr', paid: true },
          ].map((f, i) => (
            <div key={i} style={{ background: '#fff', border: `1px solid ${SL3.border}`, borderRadius: 12, padding: 12, display: 'flex', alignItems: 'center', gap: 11 }}>
              <div style={{ width: 38, height: 46, borderRadius: 8, background: SL3.bg, border: `1px solid ${SL3.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <SL3I name="File" size={17} />
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                  <span style={{ fontFamily: SL3.mono, fontSize: 11, color: SL3.textMute, fontWeight: 600 }}>{f.num}</span>
                  <span style={{ background: SL3.greenSoft, color: SL3.green, fontSize: 9, fontWeight: 700, padding: '1px 6px', borderRadius: 999 }}>PAYÉE</span>
                </div>
                <div style={{ fontSize: 13, fontWeight: 600, color: SL3.text, marginTop: 2 }}>{f.client}</div>
                <div style={{ fontSize: 11, color: SL3.textMute, marginTop: 1 }}>{f.proj} · {f.date}</div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{ fontFamily: SL3.mono, fontSize: 13, color: SL3.text, fontWeight: 700 }}>{f.amount}</div>
                <div style={{ fontSize: 10, color: SL3.textMute, marginTop: 4 }}>↓ PDF</div>
              </div>
            </div>
          ))}
        </div>
      </div>

      <_AppTabBar_L3 active="wallet" />
    </_AppFrame_L3>
  );
}

// ─── Détail paiement (transaction sur jalon) ────────────────
function AppPaiementDetail() {
  return (
    <_AppFrame_L3 bg="#fff">
      <div style={{ flexShrink: 0, padding: '6px 14px 12px', background: '#fff', borderBottom: `1px solid ${SL3.border}`, display: 'flex', alignItems: 'center', gap: 10 }}>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SL3.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL3I name="ArrowLeft" size={18} />
        </button>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 11, color: SL3.textMute }}>Mouvement · entrant</div>
          <div style={{ fontSize: 14, fontWeight: 600, color: SL3.text }}>Jalon 1 · Refonte Helio</div>
        </div>
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SL3.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL3I name="Share" size={16} />
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        {/* Hero montant */}
        <div style={{ padding: '32px 20px 24px', textAlign: 'center', background: SL3.bg }}>
          <div style={{ width: 52, height: 52, borderRadius: 16, background: SL3.greenSoft, color: SL3.green, display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 14px', fontFamily: SL3.serif, fontSize: 26, fontWeight: 700 }}>↓</div>
          <div style={{ fontSize: 11, color: SL3.textSubtle, letterSpacing: '0.08em', fontWeight: 700, textTransform: 'uppercase' }}>Reçu</div>
          <div style={{ fontFamily: SL3.serif, fontSize: 42, fontWeight: 600, color: SL3.text, letterSpacing: '-0.025em', marginTop: 4, lineHeight: 1 }}>+ 2 400,00 €</div>
          <div style={{ fontSize: 13, color: SL3.textMute, marginTop: 6, fontFamily: SL3.serif, fontStyle: 'italic' }}>15 mai 2024 · 14:32</div>
        </div>

        {/* Récap */}
        <div style={{ padding: '20px 20px 0' }}>
          <div style={{ fontSize: 11, color: SL3.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', marginBottom: 12 }}>Détails</div>

          <div style={{ background: '#fff', border: `1px solid ${SL3.border}`, borderRadius: 14, padding: 4 }}>
            {[
              { l: 'De', v: 'Léa Bertrand · Helio', leftIcon: 2 },
              { l: 'Mission', v: 'Refonte app Helio' },
              { l: 'Jalon', v: '1 sur 4 — Discovery + Audit' },
              { l: 'Méthode', v: 'Stripe Connect' },
              { l: 'Statut', v: 'Confirmé', good: true },
            ].map((row, i, a) => (
              <div key={i} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 12px', borderBottom: i < a.length - 1 ? `1px solid ${SL3.border}` : 'none', gap: 12 }}>
                <span style={{ fontSize: 12, color: SL3.textMute, flexShrink: 0 }}>{row.l}</span>
                <div style={{ display: 'flex', alignItems: 'center', gap: 7, minWidth: 0 }}>
                  {row.leftIcon !== undefined ? <SL3Portrait id={row.leftIcon} size={20} /> : null}
                  <span style={{ fontSize: 13, fontWeight: 600, color: row.good ? SL3.green : SL3.text, textAlign: 'right' }}>{row.v}</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Décomposition montant */}
        <div style={{ padding: '20px 20px 0' }}>
          <div style={{ fontSize: 11, color: SL3.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', marginBottom: 12 }}>Décomposition</div>
          <div style={{ background: '#fff', border: `1px solid ${SL3.border}`, borderRadius: 14, padding: 14 }}>
            {[
              { l: 'Montant brut', v: '2 530,00 €' },
              { l: 'Frais plateforme (5%)', v: '− 126,50 €', mute: true },
              { l: 'Frais Stripe', v: '− 3,50 €', mute: true },
            ].map((row, i) => (
              <div key={i} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0', fontSize: 13 }}>
                <span style={{ color: SL3.textMute }}>{row.l}</span>
                <span style={{ fontFamily: SL3.mono, color: row.mute ? SL3.textMute : SL3.text, fontWeight: 600 }}>{row.v}</span>
              </div>
            ))}
            <div style={{ borderTop: `1px solid ${SL3.border}`, marginTop: 8, paddingTop: 10, display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: SL3.text }}>Net reçu</span>
              <span style={{ fontFamily: SL3.mono, fontSize: 14, color: SL3.green, fontWeight: 700 }}>2 400,00 €</span>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div style={{ padding: '24px 20px 28px', display: 'flex', gap: 8 }}>
          <button style={{ flex: 1, padding: '12px', background: SL3.bg, color: SL3.text, border: `1px solid ${SL3.border}`, borderRadius: 12, fontSize: 13, fontWeight: 600, fontFamily: SL3.sans }}>↓ Facture PDF</button>
          <button style={{ flex: 1, padding: '12px', background: SL3.bg, color: SL3.text, border: `1px solid ${SL3.border}`, borderRadius: 12, fontSize: 13, fontWeight: 600, fontFamily: SL3.sans }}>Partager</button>
        </div>
      </div>
    </_AppFrame_L3>
  );
}

Object.assign(window, { AppWallet, AppFactures, AppPaiementDetail });
