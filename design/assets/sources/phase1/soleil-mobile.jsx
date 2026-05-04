// Soleil — primitives mobile (responsive web, pas app native)
// Le "frame" ici = juste un viewport CSS de largeur ~390 px, présenté dans le canvas comme une page web mobile classique.

const SM = window.S;
const SMI = window.SI;
const SMPortrait = window.Portrait;

// ─── Frame mobile ─────────────────────────────────────────────
// Présente l'écran dans un viewport 390px avec un faux URL bar de navigateur mobile (purement décoratif pour le canvas).
function MobileFrame({ url = 'atelier.fr', children, hideUrlBar = false, bg }) {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', justifyContent: 'center', background: '#e8e2d4', padding: 0, fontFamily: SM.sans }}>
      <div style={{ width: 390, height: '100%', background: bg || SM.bg, display: 'flex', flexDirection: 'column', boxShadow: '0 0 0 1px rgba(42,31,21,0.06)', overflow: 'hidden', position: 'relative' }}>
        {!hideUrlBar ? (
          <div style={{ flexShrink: 0, height: 36, background: '#f6f2ea', borderBottom: `1px solid ${SM.border}`, display: 'flex', alignItems: 'center', padding: '0 14px', gap: 10, fontSize: 12, color: SM.textMute }}>
            <SMI name="Lock" size={11} />
            <span style={{ fontFamily: SM.mono, fontSize: 11.5 }}>{url}</span>
            <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
              <SMI name="Refresh" size={13} />
              <SMI name="More" size={13} />
            </div>
          </div>
        ) : null}
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          {children}
        </div>
      </div>
    </div>
  );
}

// ─── Header mobile ─────────────────────────────────────────────
// Titre + actions. Variante "back" pour les écrans secondaires.
function MobileHeader({ title, back, action, subtitle, transparent, menu = true }) {
  return (
    <div style={{ flexShrink: 0, padding: '14px 16px 12px', background: transparent ? 'transparent' : '#fff', borderBottom: transparent ? 'none' : `1px solid ${SM.border}`, display: 'flex', alignItems: 'center', gap: 12 }}>
      {back ? (
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SM.bg, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}><SMI name="ArrowLeft" size={16} /></button>
      ) : null}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontFamily: SM.serif, fontSize: 19, fontWeight: 600, letterSpacing: '-0.01em', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{title}</div>
        {subtitle ? <div style={{ fontSize: 11.5, color: SM.textMute, fontStyle: 'italic', fontFamily: SM.serif, marginTop: 1 }}>{subtitle}</div> : null}
      </div>
      {action ? action : null}
      {menu && !action ? (
        <button style={{ width: 36, height: 36, borderRadius: '50%', background: SM.bg, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }} title="Menu"><SMI name="Menu" size={16} /></button>
      ) : null}
    </div>
  );
}

// Drawer menu — full menu accessible via hamburger top-right
function MobileDrawer({ role = 'freelancer', open = false }) {
  if (!open) return null;
  const items = role === 'freelancer' ? [
    { group: 'Activité', items: [
      { icon: 'Home', label: 'Tableau de bord' },
      { icon: 'Inbox', label: 'Opportunités', badge: 3 },
      { icon: 'Briefcase', label: 'Mes candidatures' },
      { icon: 'Folder', label: 'Missions' },
    ]},
    { group: 'Argent', items: [
      { icon: 'Wallet', label: 'Portefeuille' },
      { icon: 'Receipt', label: 'Factures' },
      { icon: 'Card', label: 'Infos paiement' },
    ]},
    { group: 'Profil', items: [
      { icon: 'User', label: 'Profil prestataire' },
      { icon: 'Settings', label: 'Paramètres' },
      { icon: 'Help', label: 'Aide' },
      { icon: 'LogOut', label: 'Se déconnecter' },
    ]},
  ] : [
    { group: 'Activité', items: [
      { icon: 'Home', label: 'Tableau de bord' },
      { icon: 'Briefcase', label: 'Annonces' },
      { icon: 'Folder', label: 'Projets' },
      { icon: 'Search', label: 'Trouver des freelances' },
      { icon: 'Users', label: 'Équipe' },
    ]},
    { group: 'Argent', items: [
      { icon: 'Wallet', label: 'Portefeuille' },
      { icon: 'Receipt', label: 'Factures' },
    ]},
    { group: 'Compte', items: [
      { icon: 'Building', label: 'Profil entreprise' },
      { icon: 'Settings', label: 'Paramètres' },
      { icon: 'Help', label: 'Aide' },
      { icon: 'LogOut', label: 'Se déconnecter' },
    ]},
  ];
  return (
    <div style={{ position: 'absolute', inset: 0, background: 'rgba(42,31,21,0.4)', display: 'flex', justifyContent: 'flex-end', zIndex: 20 }}>
      <div style={{ width: 320, maxWidth: '85%', background: '#fff', height: '100%', display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: '16px 18px', borderBottom: `1px solid ${SM.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ fontFamily: SM.serif, fontSize: 18, fontWeight: 600 }}>Menu</div>
          <button style={{ width: 32, height: 32, borderRadius: '50%', background: SM.bg, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SMI name="X" size={14} /></button>
        </div>
        <div style={{ flex: 1, overflow: 'auto', padding: '8px 0' }}>
          {items.map((g, gi) => (
            <div key={gi} style={{ padding: '12px 0' }}>
              <div style={{ padding: '0 18px 6px', fontSize: 10, fontWeight: 700, letterSpacing: '0.08em', textTransform: 'uppercase', color: SM.textMute, fontFamily: SM.mono }}>{g.group}</div>
              {g.items.map((it, i) => (
                <div key={i} style={{ padding: '12px 18px', display: 'flex', alignItems: 'center', gap: 14, cursor: 'pointer' }}>
                  <SMI name={it.icon} size={18} />
                  <div style={{ flex: 1, fontSize: 14, fontWeight: 500 }}>{it.label}</div>
                  {it.badge ? <span style={{ background: SM.accent, color: '#fff', fontSize: 10, fontWeight: 700, padding: '2px 7px', borderRadius: 999 }}>{it.badge}</span> : null}
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── Bottom nav ─────────────────────────────────────────────────
// Position fixe bas. 4-5 items max selon le rôle.
function MobileBottomNav({ active, role = 'freelancer' }) {
  const items = role === 'freelancer' ? [
    { id: 'home', icon: 'Home', label: 'Accueil' },
    { id: 'opp', icon: 'Inbox', label: 'Opportunités', badge: 3 },
    { id: 'msg', icon: 'Chat', label: 'Messages', badge: 2 },
    { id: 'wallet', icon: 'Wallet', label: 'Argent' },
    { id: 'profile', icon: 'User', label: 'Profil' },
  ] : role === 'enterprise' ? [
    { id: 'home', icon: 'Home', label: 'Accueil' },
    { id: 'jobs', icon: 'Briefcase', label: 'Annonces' },
    { id: 'msg', icon: 'Chat', label: 'Messages', badge: 3 },
    { id: 'find', icon: 'Search', label: 'Trouver' },
    { id: 'profile', icon: 'User', label: 'Compte' },
  ] : [
    { id: 'home', icon: 'Home', label: 'Accueil' },
    { id: 'msg', icon: 'Chat', label: 'Messages' },
    { id: 'profile', icon: 'User', label: 'Compte' },
  ];

  return (
    <div style={{ flexShrink: 0, background: '#fff', borderTop: `1px solid ${SM.border}`, padding: '8px 4px 12px', display: 'flex', justifyContent: 'space-around', boxShadow: '0 -2px 16px rgba(42,31,21,0.04)' }}>
      {items.map(it => {
        const sel = active === it.id;
        return (
          <div key={it.id} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3, cursor: 'pointer', padding: '4px 0', position: 'relative' }}>
            <div style={{ position: 'relative', color: sel ? SM.accent : SM.textMute }}>
              <SMI name={it.icon} size={20} />
              {it.badge ? <span style={{ position: 'absolute', top: -4, right: -8, background: SM.accent, color: '#fff', fontSize: 9.5, fontWeight: 700, padding: '1px 5px', borderRadius: 999, minWidth: 14, textAlign: 'center', lineHeight: '12px' }}>{it.badge}</span> : null}
            </div>
            <div style={{ fontSize: 10, fontWeight: 600, color: sel ? SM.accent : SM.textMute }}>{it.label}</div>
          </div>
        );
      })}
    </div>
  );
}

// ─── Sheet modal — apparaît du bas ─────────────────────────────
function MobileSheet({ title, children, open = true }) {
  if (!open) return null;
  return (
    <div style={{ position: 'absolute', inset: 0, background: 'rgba(42,31,21,0.4)', display: 'flex', alignItems: 'flex-end', zIndex: 10 }}>
      <div style={{ width: '100%', background: '#fff', borderTopLeftRadius: 20, borderTopRightRadius: 20, padding: '12px 18px 24px', maxHeight: '70%', overflow: 'auto' }}>
        <div style={{ width: 40, height: 4, background: SM.borderStrong, borderRadius: 999, margin: '0 auto 14px' }} />
        {title ? <div style={{ fontFamily: SM.serif, fontSize: 18, fontWeight: 600, marginBottom: 12 }}>{title}</div> : null}
        {children}
      </div>
    </div>
  );
}

// ─── Card list item ─────────────────────────────────────────────
function MobileListItem({ leading, title, subtitle, trailing, onClick, accent }) {
  return (
    <div onClick={onClick} style={{ background: '#fff', padding: '14px 16px', display: 'flex', alignItems: 'center', gap: 12, cursor: onClick ? 'pointer' : 'default', borderBottom: `1px solid ${SM.border}`, position: 'relative' }}>
      {accent ? <div style={{ position: 'absolute', left: 0, top: 8, bottom: 8, width: 3, background: SM.accent, borderRadius: 999 }} /> : null}
      {leading ? <div style={{ flexShrink: 0 }}>{leading}</div> : null}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 2, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{title}</div>
        {subtitle ? <div style={{ fontSize: 12, color: SM.textMute, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{subtitle}</div> : null}
      </div>
      {trailing ? <div style={{ flexShrink: 0 }}>{trailing}</div> : null}
    </div>
  );
}

// ─── Segmented control ─────────────────────────────────────────
function MobileSegmented({ items, active }) {
  return (
    <div style={{ display: 'flex', gap: 4, padding: 4, background: SM.bg, borderRadius: 999 }}>
      {items.map((it, i) => (
        <div key={i} style={{ flex: 1, padding: '7px 10px', borderRadius: 999, background: i === active ? '#fff' : 'transparent', boxShadow: i === active ? '0 1px 3px rgba(0,0,0,0.08)' : 'none', textAlign: 'center', fontSize: 12, fontWeight: 600, color: i === active ? SM.text : SM.textMute, cursor: 'pointer', whiteSpace: 'nowrap' }}>{it}</div>
      ))}
    </div>
  );
}

// ─── Floating Action Button ─────────────────────────────────────
function MobileFab({ icon = 'Plus', label }) {
  return (
    <button style={{ position: 'absolute', bottom: 80, right: 16, background: SM.text, color: '#fff', border: 'none', height: 52, padding: label ? '0 22px 0 18px' : 0, width: label ? 'auto' : 52, borderRadius: 999, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, boxShadow: '0 8px 24px rgba(42,31,21,0.3)', fontSize: 13.5, fontWeight: 600, fontFamily: SM.sans }}>
      <SMI name={icon} size={18} />
      {label}
    </button>
  );
}

window.MobileFrame = MobileFrame;
window.MobileHeader = MobileHeader;
window.MobileDrawer = MobileDrawer;
window.MobileBottomNav = MobileBottomNav;
window.MobileSheet = MobileSheet;
window.MobileListItem = MobileListItem;
window.MobileSegmented = MobileSegmented;
window.MobileFab = MobileFab;
