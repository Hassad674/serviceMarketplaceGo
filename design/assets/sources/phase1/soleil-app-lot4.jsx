// App Lot 4 — Communication : Liste conversations, Notifications
const SL4 = window.S;
const SL4I = window.SI;
const _AppFrame_L4 = window.AppFrame;
const _AppTabBar_L4 = window.AppTabBar;
const SL4Portrait = window.Portrait;

function AppConversations() {
  const convos = [
    { id: 4, name: 'Sofia Lambert', preview: 'Super, je vous envoie le portfolio aujourd\'hui.', time: '14:32', unread: 2, online: true, project: 'Product designer' },
    { id: 2, name: 'Léa Bertrand', preview: 'Vous · Voici les maquettes finales du jalon 2', time: '12:18', read: true, project: 'Refonte Helio' },
    { id: 3, name: 'Théo Martinet', preview: 'Bonjour, est-ce que la mission est toujours…', time: 'Hier', unread: 1, project: 'Motion designer' },
    { id: 0, name: 'Camille Dubois', preview: 'Vous · Merci, c\'est noté ✓', time: 'Hier', read: true, project: 'Brand designer' },
    { id: 5, name: 'Marc Olivier', preview: 'Le contrat est-il prêt pour signature ?', time: 'Lun', unread: 1, project: 'Dev back-end' },
    { id: 1, name: 'Marion Lefèvre', preview: 'Vous · D\'accord, on fait comme ça', time: '15 mai', read: true, project: 'Identité visuelle' },
  ];
  return (
    <_AppFrame_L4>
      <div style={{ flexShrink: 0, padding: '6px 20px 14px', background: SL4.bg, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <div style={{ fontFamily: SL4.serif, fontSize: 26, fontWeight: 600, letterSpacing: '-0.02em', color: SL4.text }}>Messages</div>
          <div style={{ fontSize: 12.5, color: SL4.textMute, fontFamily: SL4.serif, fontStyle: 'italic', marginTop: 2 }}>4 nouveaux · 6 conversations</div>
        </div>
        <button style={{ width: 38, height: 38, borderRadius: '50%', background: SL4.text, color: '#fff', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL4I name="Edit" size={15} />
        </button>
      </div>

      {/* Search */}
      <div style={{ flexShrink: 0, padding: '0 20px 12px', background: SL4.bg }}>
        <div style={{ background: '#fff', border: `1px solid ${SL4.border}`, borderRadius: 12, padding: '10px 14px', display: 'flex', alignItems: 'center', gap: 10 }}>
          <SL4I name="Search" size={16} />
          <span style={{ fontSize: 13, color: SL4.textMute }}>Rechercher dans les messages…</span>
        </div>
      </div>

      {/* Tabs */}
      <div style={{ flexShrink: 0, padding: '0 20px 8px', background: SL4.bg, display: 'flex', gap: 6 }}>
        {[{ l: 'Toutes', n: 6, active: true }, { l: 'Non lues', n: 4 }, { l: 'Projets', n: 3 }, { l: 'Archives', n: 12 }].map(t => (
          <span key={t.l} style={{ padding: '6px 11px', borderRadius: 999, fontSize: 12, fontWeight: 600, whiteSpace: 'nowrap', background: t.active ? SL4.text : '#fff', color: t.active ? '#fff' : SL4.textMute, border: t.active ? 'none' : `1px solid ${SL4.border}` }}>{t.l} <span style={{ opacity: 0.6 }}>{t.n}</span></span>
        ))}
      </div>

      {/* Liste */}
      <div style={{ flex: 1, overflow: 'auto', background: '#fff', borderTop: `1px solid ${SL4.border}` }}>
        {convos.map((c, i, a) => (
          <div key={i} style={{ display: 'flex', gap: 11, padding: '12px 18px', borderBottom: i < a.length - 1 ? `1px solid ${SL4.border}` : 'none', alignItems: 'center', background: c.unread ? '#fffaf3' : '#fff' }}>
            <div style={{ position: 'relative', flexShrink: 0 }}>
              <SL4Portrait id={c.id} size={46} rounded={14} />
              {c.online ? <div style={{ position: 'absolute', bottom: -1, right: -1, width: 12, height: 12, borderRadius: '50%', background: SL4.green, border: '2px solid #fff' }} /> : null}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 6 }}>
                <div style={{ fontSize: 13.5, fontWeight: c.unread ? 700 : 600, color: SL4.text, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{c.name}</div>
                <div style={{ fontSize: 11, color: c.unread ? SL4.accent : SL4.textMute, fontWeight: c.unread ? 700 : 500, flexShrink: 0 }}>{c.time}</div>
              </div>
              <div style={{ fontSize: 11, color: SL4.accentDeep, marginTop: 2, fontFamily: SL4.serif, fontStyle: 'italic' }}>{c.project}</div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8, marginTop: 4 }}>
                <div style={{ fontSize: 12, color: c.unread ? SL4.text : SL4.textMute, fontWeight: c.unread ? 600 : 400, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', flex: 1 }}>{c.preview}</div>
                {c.unread ? <div style={{ minWidth: 18, height: 18, borderRadius: 9, background: SL4.accent, color: '#fff', fontSize: 10, fontWeight: 700, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '0 5px', flexShrink: 0 }}>{c.unread}</div> : c.read ? <SL4I name="Check" size={13} /> : null}
              </div>
            </div>
          </div>
        ))}
      </div>

      <_AppTabBar_L4 active="chat" />
    </_AppFrame_L4>
  );
}

function AppNotifications() {
  return (
    <_AppFrame_L4>
      <div style={{ flexShrink: 0, padding: '6px 20px 14px', background: SL4.bg, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <div style={{ fontFamily: SL4.serif, fontSize: 26, fontWeight: 600, letterSpacing: '-0.02em', color: SL4.text }}>Notifications</div>
          <div style={{ fontSize: 12.5, color: SL4.textMute, fontFamily: SL4.serif, fontStyle: 'italic', marginTop: 2 }}>5 non lues · tout marquer lu</div>
        </div>
        <button style={{ width: 38, height: 38, borderRadius: '50%', background: '#fff', border: `1px solid ${SL4.border}`, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <SL4I name="Settings" size={16} />
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '0 20px 20px' }}>
        {/* Aujourd'hui */}
        <div style={{ fontSize: 11, color: SL4.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', margin: '6px 4px 10px' }}>Aujourd'hui</div>
        <div style={{ background: '#fff', border: `1px solid ${SL4.border}`, borderRadius: 14, overflow: 'hidden' }}>
          {[
            { type: 'msg', icon: 'Chat', accent: 'accent', title: 'Sofia Lambert vous a répondu', sub: '« Super, je vous envoie le portfolio aujourd\'hui. »', time: '14:32', unread: true, portrait: 4 },
            { type: 'cand', icon: 'Briefcase', accent: 'green', title: 'Nouvelle candidature', sub: 'Théo Martinet sur "Motion designer"', time: '12:18', unread: true, portrait: 3 },
            { type: 'pay', icon: 'Wallet', accent: 'green', title: 'Paiement reçu', sub: '+ 2 400 € · Jalon 1 Refonte Helio', time: '10:04', unread: true },
          ].map((n, i, a) => (
            <NotifRow key={i} n={n} last={i === a.length - 1} />
          ))}
        </div>

        {/* Hier */}
        <div style={{ fontSize: 11, color: SL4.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', margin: '20px 4px 10px' }}>Hier</div>
        <div style={{ background: '#fff', border: `1px solid ${SL4.border}`, borderRadius: 14, overflow: 'hidden' }}>
          {[
            { type: 'milestone', icon: 'Check', accent: 'green', title: 'Jalon validé', sub: 'Léa Bertrand a validé le jalon 1 de Refonte Helio', time: 'Hier 17:42', unread: true },
            { type: 'review', icon: 'Star', accent: 'accent', title: 'Vous avez reçu un avis', sub: 'Marie Lambert · 5 étoiles', time: 'Hier 11:08', unread: true, portrait: 1 },
            { type: 'job', icon: 'Sparkle', accent: 'accent', title: '3 nouvelles opportunités', sub: 'Match avec votre profil ≥ 90 %', time: 'Hier 09:00' },
          ].map((n, i, a) => <NotifRow key={i} n={n} last={i === a.length - 1} />)}
        </div>

        {/* Cette semaine */}
        <div style={{ fontSize: 11, color: SL4.textSubtle, letterSpacing: '0.06em', fontWeight: 700, textTransform: 'uppercase', margin: '20px 4px 10px' }}>Cette semaine</div>
        <div style={{ background: '#fff', border: `1px solid ${SL4.border}`, borderRadius: 14, overflow: 'hidden' }}>
          {[
            { type: 'profile', icon: 'Eye', accent: 'mute', title: 'Votre profil a été vu 24 fois', sub: '+ 18 % vs semaine précédente', time: 'Lun' },
            { type: 'system', icon: 'Shield', accent: 'mute', title: 'Vérification d\'identité confirmée', sub: 'Votre badge "Vérifié" est actif', time: 'Lun' },
          ].map((n, i, a) => <NotifRow key={i} n={n} last={i === a.length - 1} />)}
        </div>
      </div>

      <_AppTabBar_L4 active="home" />
    </_AppFrame_L4>
  );
}

function NotifRow({ n, last }) {
  const colors = {
    'accent': { bg: SL4.accentSoft, fg: SL4.accent },
    'green': { bg: SL4.greenSoft, fg: SL4.green },
    'mute': { bg: SL4.bg, fg: SL4.textMute },
  };
  const c = colors[n.accent];
  return (
    <div style={{ display: 'flex', gap: 11, padding: '12px 14px', borderBottom: last ? 'none' : `1px solid ${SL4.border}`, alignItems: 'flex-start', background: n.unread ? '#fffaf3' : '#fff' }}>
      {n.portrait !== undefined ? (
        <div style={{ position: 'relative', flexShrink: 0 }}>
          <SL4Portrait id={n.portrait} size={36} />
          <div style={{ position: 'absolute', bottom: -2, right: -2, width: 18, height: 18, borderRadius: '50%', background: c.bg, color: c.fg, display: 'flex', alignItems: 'center', justifyContent: 'center', border: '2px solid #fff' }}>
            <SL4I name={n.icon} size={9} />
          </div>
        </div>
      ) : (
        <div style={{ width: 36, height: 36, borderRadius: 11, background: c.bg, color: c.fg, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
          <SL4I name={n.icon} size={15} />
        </div>
      )}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 6 }}>
          <div style={{ fontSize: 13, fontWeight: n.unread ? 700 : 600, color: SL4.text, lineHeight: 1.3 }}>{n.title}</div>
          <div style={{ fontSize: 10.5, color: SL4.textMute, flexShrink: 0 }}>{n.time}</div>
        </div>
        <div style={{ fontSize: 11.5, color: SL4.textMute, marginTop: 2, lineHeight: 1.4 }}>{n.sub}</div>
      </div>
      {n.unread ? <div style={{ width: 7, height: 7, borderRadius: '50%', background: SL4.accent, marginTop: 6, flexShrink: 0 }} /> : null}
    </div>
  );
}

Object.assign(window, { AppConversations, AppNotifications });
