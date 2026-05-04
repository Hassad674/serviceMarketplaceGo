// Lot B v2 — STRICT alignement sur l'app existante. On garde EXACTEMENT les mêmes éléments
// fonctionnels, on injecte juste l'identité visuelle Soleil v2.
const { Icons: IB } = window;

const SB = {
  bg: '#fffbf5',
  surface: '#ffffff',
  border: '#f0e6d8',
  borderStrong: '#e0d3bc',
  text: '#2a1f15',
  textMute: '#7a6850',
  textSubtle: '#a89679',
  accent: '#e85d4a',
  accentSoft: '#fde9e3',
  accentDeep: '#c43a26',
  pink: '#f08aa8',
  pinkSoft: '#fde6ed',
  pinkDeep: '#c84d72',
  green: '#5a9670',
  greenSoft: '#e8f2eb',
  amber: '#d4924a',
  amberSoft: '#fbf0dc',
  blue: '#3a6b7a',
  blueSoft: '#dfecef',
  serif: 'Fraunces, Georgia, serif',
  sans: '"Inter Tight", system-ui, sans-serif',
  mono: '"Geist Mono", monospace',
};

const BI = ({ name, size = 18 }) => {
  const C = IB[name];
  return C ? <C size={size} stroke={2} /> : null;
};

// ─── Sidebar fidèle aux screenshots ──────────────────────────────
function FSidebar({ active }) {
  const items = [
    { id: 'home', icon: 'Home', label: 'Tableau de bord' },
    { id: 'msg', icon: 'Chat', label: 'Messages' },
    { id: 'proj', icon: 'Folder', label: 'Projets' },
    { id: 'opp', icon: 'Inbox', label: 'Opportunités' },
    { id: 'cands', icon: 'File', label: 'Mes candidatures' },
    { id: 'profile', icon: 'User', label: 'Profil prestataire' },
    { id: 'payout', icon: 'Euro', label: 'Infos paiement' },
    { id: 'wallet', icon: 'Wallet', label: 'Portefeuille' },
    { id: 'invoices', icon: 'Receipt', label: 'Factures' },
    { id: 'account', icon: 'Settings', label: 'Compte' },
  ];
  return (
    <aside style={{ width: 256, height: '100%', background: '#fff', borderRight: `1px solid ${SB.border}`, padding: '20px 0', display: 'flex', flexDirection: 'column', fontFamily: SB.sans, flexShrink: 0 }}>
      <div style={{ padding: '0 20px 18px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 9 }}>
          <div style={{ width: 30, height: 30, borderRadius: 999, background: SB.accent, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: SB.serif, fontSize: 18, fontWeight: 600, fontStyle: 'italic' }}>a</div>
          <div style={{ fontFamily: SB.serif, fontSize: 22, fontWeight: 500, color: SB.text, letterSpacing: '-0.02em' }}>Atelier</div>
        </div>
      </div>

      {/* User chip */}
      <div style={{ padding: '0 12px 14px' }}>
        <div style={{ background: SB.bg, borderRadius: 14, padding: '10px 12px', display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{ width: 36, height: 36, borderRadius: 999, background: SB.pink, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 13, fontFamily: SB.serif, flexShrink: 0 }}>FP</div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>fvfrgver frgrtgze</div>
            <div style={{ fontSize: 10, color: SB.pinkDeep, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase' }}>Prestataire</div>
          </div>
        </div>
      </div>

      {/* Apporteur d'affaire CTA */}
      <div style={{ padding: '0 12px 14px' }}>
        <button style={{ width: '100%', background: SB.pink, color: '#fff', border: 'none', borderRadius: 12, padding: '11px 14px', fontSize: 13, fontWeight: 700, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6, boxShadow: '0 4px 12px rgba(240,138,168,0.35)' }}>
          <BI name="Sparkle" size={14} /> Apporter d'affaire
        </button>
      </div>

      <nav style={{ padding: '0 8px', flex: 1, display: 'flex', flexDirection: 'column', gap: 1, overflow: 'auto' }}>
        {items.map(it => {
          const isActive = active === it.id;
          return (
            <div key={it.id} style={{ display: 'flex', alignItems: 'center', gap: 12, padding: '10px 12px', borderRadius: 10, fontSize: 13.5, fontWeight: 500, color: isActive ? SB.accentDeep : SB.text, background: isActive ? SB.accentSoft : 'transparent', cursor: 'pointer' }}>
              <BI name={it.icon} size={17} />
              <span style={{ flex: 1 }}>{it.label}</span>
            </div>
          );
        })}
      </nav>

      <div style={{ padding: '12px 20px', borderTop: `1px solid ${SB.border}`, display: 'flex', flexDirection: 'column', gap: 10 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, color: SB.textMute, cursor: 'pointer' }}>
          <BI name="ChevronRight" size={13} /> Réduire
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div style={{ width: 26, height: 26, borderRadius: 999, background: SB.bg, border: `1px solid ${SB.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: SB.textMute }}><BI name="User" size={12} /></div>
          <span style={{ fontSize: 12, color: SB.textMute }}>Se déconnecter</span>
        </div>
      </div>
    </aside>
  );
}

function FTopbar() {
  return (
    <div style={{ height: 60, borderBottom: `1px solid ${SB.border}`, background: '#fff', display: 'flex', alignItems: 'center', padding: '0 28px', gap: 16, flexShrink: 0 }}>
      <div style={{ flex: 1, maxWidth: 480 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, background: SB.bg, border: `1px solid ${SB.border}`, borderRadius: 999, padding: '8px 16px' }}>
          <BI name="Search" size={15} />
          <span style={{ fontSize: 13, color: SB.textSubtle }}>Recherche…</span>
        </div>
      </div>
      <div style={{ flex: 1 }} />
      <button style={{ background: SB.accent, color: '#fff', border: 'none', padding: '8px 16px', fontSize: 12.5, fontWeight: 700, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6, boxShadow: '0 4px 12px rgba(232,93,74,0.3)' }}>
        <BI name="Sparkle" size={13} /> Passer Premium
      </button>
      <div style={{ width: 36, height: 36, borderRadius: 999, border: `1px solid ${SB.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: SB.textMute }}><BI name="Smiley" size={16} /></div>
      <div style={{ position: 'relative' }}>
        <div style={{ width: 36, height: 36, borderRadius: 999, border: `1px solid ${SB.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><BI name="Bell" size={16} /></div>
        <span style={{ position: 'absolute', top: -2, right: -2, background: SB.accent, color: '#fff', fontSize: 9, fontWeight: 700, padding: '1px 5px', borderRadius: 999 }}>10</span>
      </div>
      <div style={{ width: 36, height: 36, borderRadius: 999, background: SB.pink, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 12, fontFamily: SB.serif }}>FP</div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════
// 1) PORTEFEUILLE — strict app
// ═══════════════════════════════════════════════════════════════════
function SoleilWallet() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SB.bg, fontFamily: SB.sans, color: SB.text }}>
      <FSidebar active="wallet" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <FTopbar />
        <div style={{ flex: 1, overflow: 'auto', padding: '32px 40px', maxWidth: 1100, width: '100%', margin: '0 auto' }}>

          {/* HERO — Portefeuille principal */}
          <div style={{ background: '#fff', border: `1px solid ${SB.border}`, borderRadius: 18, padding: '26px 28px', marginBottom: 14, position: 'relative', overflow: 'hidden' }}>
            <div style={{ position: 'absolute', top: -60, right: -60, width: 220, height: 220, borderRadius: '50%', background: 'radial-gradient(circle, rgba(232,93,74,0.07), transparent 65%)' }} />
            <div style={{ position: 'relative' }}>
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14, marginBottom: 18 }}>
                <div style={{ width: 44, height: 44, borderRadius: 12, background: SB.accentSoft, color: SB.accent, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <BI name="Wallet" size={20} />
                </div>
                <div style={{ flex: 1 }}>
                  <h2 style={{ fontFamily: SB.serif, fontSize: 26, fontWeight: 500, margin: 0, letterSpacing: '-0.02em' }}>Portefeuille</h2>
                  <p style={{ fontSize: 13, color: SB.textMute, margin: '2px 0 0' }}>Vos revenus issus des missions et commissions d'apport</p>
                </div>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: 24, alignItems: 'center' }}>
                <div>
                  <div style={{ fontSize: 11, color: SB.textSubtle, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase', marginBottom: 6 }}>Revenus totaux</div>
                  <div style={{ fontFamily: SB.serif, fontSize: 56, fontWeight: 400, lineHeight: 1, letterSpacing: '-0.035em', color: SB.text, marginBottom: 14 }}>10 502<span style={{ fontSize: 32 }}>,00 €</span></div>
                  <div style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 12, color: SB.green, fontWeight: 600, background: SB.greenSoft, padding: '5px 12px', borderRadius: 999 }}>
                    <BI name="CheckCircle" size={13} /> Compte Stripe prêt — virements activés
                  </div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <button disabled style={{ background: SB.pink, color: '#fff', border: 'none', padding: '12px 22px', fontSize: 14, fontWeight: 700, borderRadius: 999, cursor: 'not-allowed', opacity: 0.55, display: 'inline-flex', alignItems: 'center', gap: 8, boxShadow: '0 4px 14px rgba(240,138,168,0.3)' }}>
                    <BI name="ArrowRight" size={14} /> Retirer 0,00 €
                  </button>
                  <div style={{ fontSize: 11, color: SB.textSubtle, marginTop: 8 }}>Aucun fonds disponible</div>
                </div>
              </div>

              <div style={{ borderTop: `1px solid ${SB.border}`, marginTop: 22, paddingTop: 14, display: 'flex', gap: 22 }}>
                <a style={{ fontSize: 12.5, color: SB.text, fontWeight: 500, display: 'inline-flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
                  <BI name="Edit" size={13} /> Modifier mes infos de facturation
                </a>
                <a style={{ fontSize: 12.5, color: SB.text, fontWeight: 500, display: 'inline-flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
                  <BI name="Euro" size={13} /> Mes infos de paiement Stripe
                </a>
              </div>
            </div>
          </div>

          {/* Mois en cours */}
          <div style={{ background: '#fff', border: `1px solid ${SB.border}`, borderRadius: 16, padding: '20px 24px', marginBottom: 22, display: 'flex', alignItems: 'center', gap: 16 }}>
            <div style={{ width: 40, height: 40, borderRadius: 11, background: SB.amberSoft, color: SB.amber, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <BI name="Calendar" size={18} />
            </div>
            <div style={{ flex: 1 }}>
              <div style={{ display: 'flex', alignItems: 'baseline', gap: 12, marginBottom: 2 }}>
                <h3 style={{ fontFamily: SB.serif, fontSize: 18, margin: 0, fontWeight: 600, letterSpacing: '-0.01em' }}>Mois en cours</h3>
                <span style={{ fontSize: 11.5, color: SB.textMute, fontFamily: SB.mono }}>Du 1er mai 2026 au 1 juin 2026</span>
              </div>
              <div style={{ fontSize: 13, color: SB.textMute }}>3 jalons livrés — <span style={{ color: SB.text, fontWeight: 600 }}>75,00 €</span> de commission</div>
            </div>
            <a style={{ fontSize: 12.5, color: SB.accent, fontWeight: 700, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4 }}>
              Voir le détail <BI name="ChevronDown" size={12} />
            </a>
          </div>

          {/* Mes missions — 3 cards */}
          <div style={{ marginBottom: 24 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
              <BI name="Folder" size={17} />
              <h3 style={{ fontFamily: SB.serif, fontSize: 22, margin: 0, fontWeight: 500, letterSpacing: '-0.015em' }}>Mes missions</h3>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12 }}>
              {[
                { icon: 'Clock', color: SB.amber, soft: SB.amberSoft, label: 'En séquestre', value: '0,00 €', sub: 'Missions payées, en attente de complétion' },
                { icon: 'Wallet', color: SB.green, soft: SB.greenSoft, label: 'Disponible', value: '0,00 €', sub: 'Prêt à être retiré sur votre compte bancaire' },
                { icon: 'Send', color: SB.blue, soft: SB.blueSoft, label: 'Transféré', value: '10 502,00 €', sub: 'Total déjà versé sur votre compte bancaire' },
              ].map((m, i) => (
                <div key={i} style={{ background: '#fff', border: `1px solid ${SB.border}`, borderRadius: 14, padding: 18 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
                    <div style={{ width: 22, height: 22, borderRadius: 7, background: m.soft, color: m.color, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><BI name={m.icon} size={12} /></div>
                    <span style={{ fontSize: 11.5, color: SB.textMute, fontWeight: 600, letterSpacing: '0.04em', textTransform: 'uppercase' }}>{m.label}</span>
                  </div>
                  <div style={{ fontFamily: SB.serif, fontSize: 26, fontWeight: 500, lineHeight: 1, letterSpacing: '-0.025em', marginBottom: 8, color: SB.text }}>{m.value}</div>
                  <div style={{ fontSize: 11.5, color: SB.textMute, lineHeight: 1.4 }}>{m.sub}</div>
                </div>
              ))}
            </div>
          </div>

          {/* Historique des missions */}
          <div style={{ background: '#fff', border: `1px solid ${SB.border}`, borderRadius: 16, overflow: 'hidden' }}>
            <div style={{ padding: '20px 24px', borderBottom: `1px solid ${SB.border}` }}>
              <h3 style={{ fontFamily: SB.serif, fontSize: 22, margin: 0, fontWeight: 500, letterSpacing: '-0.015em' }}>Historique des missions</h3>
              <p style={{ fontSize: 12.5, color: SB.textMute, margin: '2px 0 0' }}>Toutes vos missions — du séquestre au transfert</p>
            </div>
            {[
              { date: '01/05/2026', status: 'En séquestre', tag: 'mission en cours · accepted', amount: '3 898,00 €', fees: '-25,00 € Frais plateforme', badge: 'En séquestre', color: SB.amber, soft: SB.amberSoft, dot: SB.amber },
              { date: '01/05/2026', status: 'Terminée', amount: '3 188,00 €', fees: '-25,00 € Frais plateforme', badge: 'Transféré', color: SB.green, soft: SB.greenSoft, dot: SB.green },
              { date: '01/05/2026', status: 'Terminée', amount: '4 207,00 €', fees: '-25,00 € Frais plateforme', badge: 'Transféré', color: SB.green, soft: SB.greenSoft, dot: SB.green },
            ].map((m, i, arr) => (
              <div key={i} style={{ padding: '16px 24px', borderBottom: i < arr.length - 1 ? `1px solid ${SB.border}` : 'none', display: 'flex', alignItems: 'center', gap: 16, position: 'relative' }}>
                <div style={{ position: 'absolute', left: 0, top: 0, bottom: 0, width: 3, background: m.dot }} />
                <div style={{ width: 36, height: 36, borderRadius: 10, background: m.soft, color: m.color, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  <BI name={m.badge === 'Transféré' ? 'CheckCircle' : 'Clock'} size={16} />
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 2 }}>Mission du {m.date}</div>
                  <div style={{ fontSize: 12, color: SB.textMute, display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span>{m.status}</span>
                    {m.tag && <><span>—</span><span style={{ fontFamily: SB.mono, fontSize: 11 }}>{m.tag}</span></>}
                  </div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontFamily: SB.serif, fontSize: 18, fontWeight: 600, lineHeight: 1 }}>{m.amount}</div>
                  <div style={{ fontSize: 11, color: SB.textSubtle, marginTop: 4 }}>{m.fees}</div>
                </div>
                <span style={{ fontSize: 11, fontWeight: 700, color: m.color, background: m.soft, padding: '5px 12px', borderRadius: 999, whiteSpace: 'nowrap' }}>{m.badge}</span>
              </div>
            ))}
          </div>

        </div>
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════
// 2) FACTURES — sans les 4 cards stats
// ═══════════════════════════════════════════════════════════════════
function SoleilInvoices() {
  // Avatar pastilles couleur (clients sans portrait)
  const ClientChip = ({ id, name }) => {
    const palettes = [
      { bg: SB.accentSoft, fg: SB.accentDeep },
      { bg: SB.greenSoft, fg: SB.green },
      { bg: SB.pinkSoft, fg: SB.pinkDeep },
      { bg: SB.amberSoft, fg: SB.amber },
      { bg: SB.blueSoft, fg: SB.blue },
    ];
    const p = palettes[id % 5];
    const init = name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase();
    return (
      <div style={{ width: 30, height: 30, borderRadius: 8, background: p.bg, color: p.fg, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, fontWeight: 700, fontFamily: SB.serif, flexShrink: 0 }}>{init}</div>
    );
  };

  const monthInvoices = [
    { num: 'AT-2026-051', client: 'Lemon Aviation', clientId: 0, date: '12 mai 2026', mission: 'Refonte site corporate · Jalon 2', ht: '4 200 €', tva: '840 €', ttc: '5 040 €', status: 'Payée', statusColor: SB.green, statusSoft: SB.greenSoft },
    { num: 'AT-2026-050', client: 'Cobalt Studio', clientId: 1, date: '8 mai 2026', mission: 'Brand identity Q2 · Jalon 1', ht: '2 800 €', tva: '560 €', ttc: '3 360 €', status: 'En attente', statusColor: SB.amber, statusSoft: SB.amberSoft },
    { num: 'AT-2026-049', client: 'Maison Vega', clientId: 2, date: '5 mai 2026', mission: 'Audit SEO technique · Final', ht: '3 600 €', tva: '720 €', ttc: '4 320 €', status: 'Payée', statusColor: SB.green, statusSoft: SB.greenSoft },
    { num: 'AT-2026-048', client: 'Doctolib', clientId: 3, date: '2 mai 2026', mission: 'Design system v3 · Workshop', ht: '1 800 €', tva: '360 €', ttc: '2 160 €', status: 'Payée', statusColor: SB.green, statusSoft: SB.greenSoft },
  ];
  const archivedMonths = [
    { month: 'avril 2026', count: 6, total: '21 480 €' },
    { month: 'mars 2026', count: 4, total: '14 200 €' },
    { month: 'février 2026', count: 5, total: '18 940 €' },
    { month: 'janvier 2026', count: 3, total: '9 600 €' },
  ];

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SB.bg, fontFamily: SB.sans, color: SB.text }}>
      <FSidebar active="invoices" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <FTopbar />
        <div style={{ flex: 1, overflow: 'auto', padding: '32px 40px', maxWidth: 1240, width: '100%', margin: '0 auto' }}>

          {/* En-tête de page */}
          <div style={{ marginBottom: 22, display: 'flex', alignItems: 'flex-end', gap: 16 }}>
            <div style={{ flex: 1 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 6 }}>
                <BI name="Receipt" size={20} />
                <h2 style={{ fontFamily: SB.serif, fontSize: 32, margin: 0, fontWeight: 500, letterSpacing: '-0.02em' }}>Tes factures</h2>
              </div>
              <p style={{ fontSize: 13.5, color: SB.textMute, margin: 0 }}>Tout est généré automatiquement à chaque jalon validé. Tu n'as rien à faire.</p>
            </div>
            <button style={{ background: '#fff', color: SB.text, border: `1px solid ${SB.borderStrong}`, padding: '9px 16px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}><BI name="Save" size={14} /> Exporter en CSV</button>
          </div>

          {/* Mois en cours — featured table */}
          <div style={{ background: '#fff', border: `1px solid ${SB.border}`, borderRadius: 16, marginBottom: 24, overflow: 'hidden' }}>
            <div style={{ padding: '18px 24px', borderBottom: `1px solid ${SB.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: SB.bg }}>
              <div>
                <div style={{ fontSize: 11, color: SB.accent, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase', marginBottom: 4 }}>Ce mois-ci</div>
                <h3 style={{ fontFamily: SB.serif, fontSize: 22, margin: 0, fontWeight: 500, letterSpacing: '-0.015em' }}>Factures de mai 2026</h3>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: SB.textMute }}>
                <span>Trier par</span>
                <button style={{ background: '#fff', border: `1px solid ${SB.border}`, padding: '6px 12px', fontSize: 12, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4, color: SB.text }}>Date <BI name="ChevronDown" size={12} /></button>
              </div>
            </div>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
              <thead>
                <tr style={{ borderBottom: `1px solid ${SB.border}`, color: SB.textMute, fontSize: 11, letterSpacing: '0.04em', textTransform: 'uppercase', fontWeight: 600 }}>
                  <th style={{ textAlign: 'left', padding: '14px 24px' }}>N°</th>
                  <th style={{ textAlign: 'left' }}>Client</th>
                  <th style={{ textAlign: 'left' }}>Mission · jalon</th>
                  <th style={{ textAlign: 'left' }}>Date</th>
                  <th style={{ textAlign: 'right' }}>HT</th>
                  <th style={{ textAlign: 'right' }}>TVA</th>
                  <th style={{ textAlign: 'right' }}>TTC</th>
                  <th style={{ textAlign: 'left', paddingLeft: 16 }}>Statut</th>
                  <th style={{ paddingRight: 24 }}></th>
                </tr>
              </thead>
              <tbody>
                {monthInvoices.map((inv, i) => (
                  <tr key={i} style={{ borderBottom: i < monthInvoices.length - 1 ? `1px solid ${SB.border}` : 'none', cursor: 'pointer' }}>
                    <td style={{ padding: '16px 24px', fontFamily: SB.mono, fontSize: 12, fontWeight: 600 }}>{inv.num}</td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                        <ClientChip id={inv.clientId} name={inv.client} />
                        <span style={{ fontWeight: 600 }}>{inv.client}</span>
                      </div>
                    </td>
                    <td style={{ color: SB.textMute, maxWidth: 220 }}>{inv.mission}</td>
                    <td style={{ color: SB.textMute, fontFamily: SB.mono, fontSize: 12 }}>{inv.date}</td>
                    <td style={{ textAlign: 'right', fontWeight: 500 }}>{inv.ht}</td>
                    <td style={{ textAlign: 'right', color: SB.textMute, fontSize: 12 }}>{inv.tva}</td>
                    <td style={{ textAlign: 'right', fontFamily: SB.serif, fontSize: 16, fontWeight: 600 }}>{inv.ttc}</td>
                    <td style={{ paddingLeft: 16 }}><span style={{ fontSize: 11, fontWeight: 700, color: inv.statusColor, display: 'inline-flex', alignItems: 'center', gap: 5, background: inv.statusSoft, padding: '4px 10px', borderRadius: 999 }}><span style={{ width: 5, height: 5, borderRadius: '50%', background: inv.statusColor }} /> {inv.status}</span></td>
                    <td style={{ paddingRight: 24, textAlign: 'right' }}>
                      <div style={{ display: 'flex', gap: 4, justifyContent: 'flex-end' }}>
                        <button style={{ width: 30, height: 30, borderRadius: 8, background: '#fff', border: `1px solid ${SB.border}`, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><BI name="Eye" size={13} /></button>
                        <button style={{ width: 30, height: 30, borderRadius: 8, background: '#fff', border: `1px solid ${SB.border}`, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><BI name="Save" size={13} /></button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Archives */}
          <div>
            <h3 style={{ fontFamily: SB.serif, fontSize: 22, margin: 0, fontWeight: 500, marginBottom: 14, letterSpacing: '-0.015em' }}>Archives</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {archivedMonths.map((m, i) => (
                <div key={i} style={{ background: '#fff', border: `1px solid ${SB.border}`, borderRadius: 14, padding: '16px 22px', display: 'flex', alignItems: 'center', gap: 16, cursor: 'pointer' }}>
                  <BI name="ChevronRight" size={14} />
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 14, fontWeight: 600, fontFamily: SB.serif, textTransform: 'capitalize' }}>{m.month}</div>
                    <div style={{ fontSize: 12, color: SB.textMute, marginTop: 1 }}>{m.count} factures</div>
                  </div>
                  <div style={{ fontFamily: SB.serif, fontSize: 18, fontWeight: 600 }}>{m.total}</div>
                  <button style={{ background: '#fff', border: `1px solid ${SB.border}`, padding: '6px 12px', fontSize: 11, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 5 }}><BI name="Save" size={12} /> ZIP du mois</button>
                </div>
              ))}
            </div>
          </div>

        </div>
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════
// 3) PROFIL DE FACTURATION — strict app
// ═══════════════════════════════════════════════════════════════════
function SoleilBillingProfile() {
  const Section = ({ title, children, intro }) => (
    <div style={{ background: '#fff', border: `1px solid ${SB.border}`, borderRadius: 16, padding: 26, marginBottom: 14 }}>
      <h3 style={{ fontFamily: SB.serif, fontSize: 18, margin: 0, fontWeight: 600, letterSpacing: '-0.01em', marginBottom: intro ? 6 : 18 }}>{title}</h3>
      {intro && <p style={{ fontSize: 12.5, color: SB.textMute, margin: '0 0 18px', lineHeight: 1.55 }}>{intro}</p>}
      {children}
    </div>
  );
  const Field = ({ label, value, optional, prefix, placeholder }) => (
    <div>
      <label style={{ fontSize: 12, color: SB.text, fontWeight: 500, marginBottom: 6, display: 'block' }}>
        {label}{optional && <span style={{ color: SB.textSubtle, fontWeight: 400 }}> (optionnel)</span>}
      </label>
      <div style={{ background: '#fff', border: `1px solid ${SB.border}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, color: value ? SB.text : SB.textSubtle, display: 'flex', alignItems: 'center', gap: 8, minHeight: 42 }}>
        {prefix && <span style={{ color: SB.textMute, fontFamily: SB.mono, fontSize: 12 }}>{prefix}</span>}
        <span style={{ flex: 1 }}>{value || placeholder}</span>
      </div>
    </div>
  );

  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SB.bg, fontFamily: SB.sans, color: SB.text }}>
      <FSidebar active="payout" />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <FTopbar />
        <div style={{ flex: 1, overflow: 'auto', padding: '32px 40px', maxWidth: 960, width: '100%', margin: '0 auto' }}>

          {/* Header */}
          <div style={{ marginBottom: 16 }}>
            <h2 style={{ fontFamily: SB.serif, fontSize: 32, margin: 0, fontWeight: 500, letterSpacing: '-0.02em', marginBottom: 8 }}>Profil de facturation</h2>
            <p style={{ fontSize: 13.5, color: SB.textMute, margin: 0, lineHeight: 1.6, maxWidth: 640 }}>Ces informations apparaissent sur les factures que la plateforme émet à ton organisation. Elles doivent être complètes pour pouvoir retirer ton solde et souscrire à un abonnement Premium.</p>
          </div>

          {/* Sync bar */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20, gap: 12 }}>
            <div style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 12, color: SB.green, fontWeight: 600 }}>
              <BI name="CheckCircle" size={14} /> Synchronisé depuis Stripe le 1 mai 2026
            </div>
            <button style={{ background: '#fff', color: SB.text, border: `1px solid ${SB.borderStrong}`, padding: '8px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
              <BI name="ArrowRight" size={13} /> Pré-remplir depuis Stripe
            </button>
          </div>

          {/* Pays */}
          <Section title="Pays" intro="Choisis d'abord ton pays — les autres champs s'adaptent en conséquence (SIRET pour la France, n° TVA intracom pour l'UE, adresse seule ailleurs).">
            <label style={{ fontSize: 12, color: SB.text, fontWeight: 500, marginBottom: 6, display: 'block' }}>Pays de facturation</label>
            <div style={{ background: '#fff', border: `1px solid ${SB.border}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, display: 'flex', alignItems: 'center', gap: 10, justifyContent: 'space-between', cursor: 'pointer' }}>
              <span style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ fontSize: 18 }}>🇫🇷</span>
                <span style={{ fontWeight: 500 }}>France</span>
              </span>
              <BI name="ChevronDown" size={14} />
            </div>
          </Section>

          {/* Adresse */}
          <Section title="Adresse">
            <div style={{ marginBottom: 14 }}>
              <div style={{ background: SB.bg, border: `1px solid ${SB.border}`, borderRadius: 10, padding: '11px 14px', fontSize: 14, color: SB.textSubtle, display: 'flex', alignItems: 'center', gap: 8 }}>
                <BI name="Search" size={15} />
                <span>Commencez à taper votre adresse...</span>
              </div>
            </div>
            <div style={{ display: 'grid', gap: 14 }}>
              <Field label="Adresse" value="115 Cours Gambetta" />
              <Field label="Complément d'adresse" optional value="" placeholder="Bâtiment, étage, …" />
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: 14 }}>
                <Field label="Code postal" value="69003" />
                <Field label="Ville" value="Lyon" />
              </div>
            </div>
          </Section>

          {/* Type de profil */}
          <Section title="Type de profil">
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              {[
                { name: 'Particulier', selected: true },
                { name: 'Entreprise', selected: false },
              ].map((t, i) => (
                <div key={i} style={{ padding: '14px 16px', borderRadius: 12, border: `1.5px solid ${t.selected ? SB.accent : SB.border}`, background: t.selected ? SB.accentSoft : '#fff', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 12 }}>
                  <div style={{ width: 18, height: 18, borderRadius: 999, border: `2px solid ${t.selected ? SB.accent : SB.borderStrong}`, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                    {t.selected && <div style={{ width: 8, height: 8, borderRadius: 999, background: SB.accent }} />}
                  </div>
                  <span style={{ fontSize: 14, fontWeight: 600, color: t.selected ? SB.accentDeep : SB.text }}>{t.name}</span>
                </div>
              ))}
            </div>
          </Section>

          {/* Identité légale */}
          <Section title="Identité légale">
            <div style={{ display: 'grid', gap: 14 }}>
              <Field label="Raison sociale ou nom légal" value="" placeholder="Élise Marchand" />
              <Field label="SIRET" value="" placeholder="14 chiffres" />
              <Field label="N° TVA intracommunautaire" optional value="" placeholder="FR + 11 chiffres" />
            </div>
          </Section>

          {/* Actions */}
          <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 22 }}>
            <button style={{ background: '#fff', color: SB.text, border: `1px solid ${SB.border}`, padding: '12px 22px', fontSize: 13, fontWeight: 600, borderRadius: 999, cursor: 'pointer' }}>Annuler</button>
            <button style={{ background: SB.accent, color: '#fff', border: 'none', padding: '12px 26px', fontSize: 13, fontWeight: 700, borderRadius: 999, cursor: 'pointer', boxShadow: '0 4px 12px rgba(232,93,74,0.3)' }}>Enregistrer</button>
          </div>

        </div>
      </div>
    </div>
  );
}

window.SoleilWallet = SoleilWallet;
window.SoleilInvoices = SoleilInvoices;
window.SoleilBillingProfile = SoleilBillingProfile;
