// Lot F — Messagerie + Team mobile
// Même contenu/hierarchie que desktop, juste réagencé pour 390px.

const SFM = window.S;
const SFMI = window.SI;
const SFMPortrait = window.Portrait;
const _MobileFrame = window.MobileFrame;
const _MobileHeader = window.MobileHeader;
const _MobileBottomNav = window.MobileBottomNav;
const _FAKE_CONVS = window.FAKE_CONVS;
const _FAKE_THREAD = window.FAKE_THREAD;
const _TEAM_MEMBERS = window.TEAM_MEMBERS;
const _TEAM_PERMISSIONS = window.TEAM_PERMISSIONS;
const _MsgBubble = window.MsgBubble;
const _Toggle = window.Toggle;

// ─── MESSAGERIE — liste des conversations ────────────────────────
function SoleilMessagerieListMobile() {
  return (
    <_MobileFrame>
      <_MobileHeader title="Messages" subtitle="3 conversations actives" />

      {/* Tabs filtres */}
      <div style={{ flexShrink: 0, padding: '8px 16px 12px', background: '#fff', borderBottom: `1px solid ${SFM.border}`, display: 'flex', gap: 6, overflowX: 'auto' }}>
        {['Tous', 'Agence', 'Freelance', 'Apporteur', 'Entreprise'].map((t, i) => (
          <span key={t} style={{ padding: '5px 11px', background: i === 0 ? SFM.text : SFM.bg, color: i === 0 ? '#fff' : SFM.textMute, fontSize: 11.5, fontWeight: 600, borderRadius: 999, whiteSpace: 'nowrap' }}>{t}</span>
        ))}
      </div>

      {/* Search */}
      <div style={{ flexShrink: 0, padding: '12px 16px', background: '#fff', borderBottom: `1px solid ${SFM.border}` }}>
        <div style={{ position: 'relative' }}>
          <div style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: SFM.textMute }}><SFMI name="Search" size={14} /></div>
          <input placeholder="Rechercher…" style={{ width: '100%', padding: '9px 12px 9px 34px', background: SFM.bg, border: `1px solid ${SFM.border}`, borderRadius: 10, fontSize: 13, outline: 'none' }} />
        </div>
      </div>

      {/* Liste */}
      <div style={{ flex: 1, overflow: 'auto', background: '#fff' }}>
        {_FAKE_CONVS.map((c) => (
          <div key={c.id} style={{ padding: '12px 16px', display: 'flex', gap: 12, borderBottom: `1px solid ${SFM.border}` }}>
            <div style={{ position: 'relative', flexShrink: 0 }}>
              <SFMPortrait id={c.portrait} size={44} />
              {c.online ? <div style={{ position: 'absolute', bottom: 0, right: 0, width: 11, height: 11, borderRadius: '50%', background: '#3aa66b', border: '2px solid #fff' }} /> : null}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 6 }}>
                <div style={{ fontSize: 13.5, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{c.name}</div>
                <div style={{ fontSize: 10.5, color: SFM.textMute, flexShrink: 0, fontFamily: SFM.mono }}>{c.time}</div>
              </div>
              <div style={{ fontSize: 11.5, color: c.unread ? SFM.text : SFM.textMute, fontWeight: c.unread ? 500 : 400, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', marginTop: 2 }}>{c.last}</div>
            </div>
            {c.unread ? <div style={{ alignSelf: 'center', background: SFM.accent, color: '#fff', fontSize: 9.5, fontWeight: 700, padding: '1px 6px', borderRadius: 999 }}>{c.unread}</div> : null}
          </div>
        ))}
      </div>

      <_MobileBottomNav active="msg" role="enterprise" />
    </_MobileFrame>
  );
}

// ─── MESSAGERIE — conversation active ─────────────────────────────
function SoleilMessagerieThreadMobile() {
  return (
    <_MobileFrame>
      {/* Header conv custom (back + avatar + actions) */}
      <div style={{ flexShrink: 0, padding: '10px 12px 12px', background: '#fff', borderBottom: `1px solid ${SFM.border}`, display: 'flex', alignItems: 'center', gap: 10 }}>
        <button style={{ width: 34, height: 34, borderRadius: '50%', background: SFM.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}><SFMI name="ArrowLeft" size={15} /></button>
        <SFMPortrait id={0} size={36} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 13.5, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>Camille Dubois</div>
          <div style={{ fontSize: 10.5, color: '#3aa66b', fontWeight: 500, display: 'flex', alignItems: 'center', gap: 4 }}>
            <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#3aa66b' }} />En ligne
          </div>
        </div>
        <button style={{ width: 34, height: 34, borderRadius: '50%', background: SFM.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SFMI name="Phone" size={14} /></button>
        <button style={{ width: 34, height: 34, borderRadius: '50%', background: SFM.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SFMI name="MoreH" size={14} /></button>
      </div>

      {/* Sticky CTA proposer projet */}
      <div style={{ flexShrink: 0, padding: '8px 12px', background: '#fff', borderBottom: `1px solid ${SFM.border}` }}>
        <button style={{ width: '100%', padding: '9px', background: SFM.accentSoft, color: SFM.accentDeep, border: `1px solid ${SFM.border}`, borderRadius: 10, fontSize: 12.5, fontWeight: 600, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}>
          <SFMI name="Plus" size={13} /> Proposer un projet
        </button>
      </div>

      {/* Thread */}
      <div style={{ flex: 1, overflow: 'auto', padding: '14px 14px', display: 'flex', flexDirection: 'column', gap: 4, background: SFM.bg }}>
        <div style={{ alignSelf: 'center', fontSize: 10.5, color: SFM.textMute, fontFamily: SFM.mono, padding: '4px 0', marginBottom: 6 }}>— Aujourd'hui —</div>
        {_FAKE_THREAD.slice(0, 6).map((m, i) => <_MsgBubble key={i} msg={m} />)}

        <div style={{ alignSelf: 'flex-start', maxWidth: '90%', background: '#fff8ec', border: '1px solid #f4dba0', borderRadius: 14, padding: 12, marginTop: 8 }}>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center', marginBottom: 8 }}>
            <div style={{ width: 30, height: 30, borderRadius: '50%', background: '#f4dba0', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#a86a14' }}><SFMI name="Star" size={14} /></div>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12.5, fontWeight: 600 }}>Mission terminée</div>
              <div style={{ fontSize: 10.5, color: SFM.textMute }}>Laissez votre avis</div>
            </div>
          </div>
          <button style={{ width: '100%', background: SFM.text, color: '#fff', border: 'none', padding: '8px', fontSize: 12, fontWeight: 600, borderRadius: 8 }}>Évaluer →</button>
        </div>
      </div>

      {/* Composer */}
      <div style={{ flexShrink: 0, background: '#fff', borderTop: `1px solid ${SFM.border}`, padding: '10px 12px', display: 'flex', gap: 8, alignItems: 'center' }}>
        <button style={{ width: 34, height: 34, borderRadius: '50%', background: SFM.bg, border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SFM.textMute, flexShrink: 0 }}><SFMI name="Paperclip" size={14} /></button>
        <input placeholder="Message…" style={{ flex: 1, padding: '9px 12px', background: SFM.bg, border: `1px solid ${SFM.border}`, borderRadius: 999, fontSize: 13, outline: 'none', minWidth: 0 }} />
        <button style={{ width: 34, height: 34, borderRadius: '50%', background: SFM.accent, color: '#fff', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}><SFMI name="Send" size={13} /></button>
      </div>
    </_MobileFrame>
  );
}

// ─── TEAM — mobile ───────────────────────────────────────────────
function SoleilTeamMobile() {
  return (
    <_MobileFrame>
      <_MobileHeader title="Équipe" subtitle="Nova Studio · 4 membres" />

      <div style={{ flex: 1, overflow: 'auto' }}>
        {/* Header card */}
        <div style={{ background: '#fff', padding: '16px', borderBottom: `1px solid ${SFM.border}`, display: 'flex', alignItems: 'center', gap: 12 }}>
          <div style={{ width: 44, height: 44, borderRadius: 12, background: SFM.accentSoft, display: 'flex', alignItems: 'center', justifyContent: 'center', color: SFM.accentDeep, flexShrink: 0 }}>
            <SFMI name="Building" size={20} />
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontFamily: SFM.serif, fontSize: 18, fontWeight: 600, letterSpacing: '-0.01em' }}>Nova Studio</div>
            <div style={{ fontSize: 11.5, color: SFM.textMute }}>Entreprise · 4 membres</div>
          </div>
          <button style={{ width: 36, height: 36, borderRadius: '50%', background: SFM.accent, color: '#fff', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}><SFMI name="Plus" size={16} /></button>
        </div>

        {/* Section Membres */}
        <div style={{ padding: '14px 16px 8px', fontSize: 10.5, fontWeight: 700, fontFamily: SFM.mono, letterSpacing: '0.08em', textTransform: 'uppercase', color: SFM.textMute }}>Membres</div>
        <div style={{ background: '#fff', borderTop: `1px solid ${SFM.border}`, borderBottom: `1px solid ${SFM.border}` }}>
          {_TEAM_MEMBERS.map((m, i) => (
            <div key={i} style={{ padding: '12px 16px', display: 'flex', gap: 12, alignItems: 'center', borderBottom: i === _TEAM_MEMBERS.length - 1 ? 'none' : `1px solid ${SFM.border}` }}>
              <SFMPortrait id={m.portrait} size={40} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13.5, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6 }}>
                  {m.name}
                  {m.you ? <span style={{ fontSize: 9.5, padding: '1px 5px', background: SFM.bg, borderRadius: 999, color: SFM.textMute, fontWeight: 600 }}>vous</span> : null}
                </div>
                <div style={{ fontSize: 11, color: SFM.textMute, marginTop: 2, display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={{ fontWeight: 600, color: m.role === 'Owner' ? '#a86a14' : SFM.text, display: 'inline-flex', alignItems: 'center', gap: 3 }}>
                    {m.role === 'Owner' ? <SFMI name="Star" size={10} /> : null}{m.role}
                  </span>
                  {m.title !== '—' ? <><span>·</span><span>{m.title}</span></> : null}
                </div>
              </div>
              <button style={{ width: 30, height: 30, borderRadius: '50%', background: 'transparent', border: 'none', color: SFM.textMute, opacity: m.you ? 0.3 : 1 }}><SFMI name="MoreH" size={14} /></button>
            </div>
          ))}
        </div>

        {/* Section Rôles & permissions — collapsé en accordéon */}
        <div style={{ padding: '14px 16px 8px', fontSize: 10.5, fontWeight: 700, fontFamily: SFM.mono, letterSpacing: '0.08em', textTransform: 'uppercase', color: SFM.textMute }}>Rôles & permissions</div>

        <div style={{ background: '#fff', borderTop: `1px solid ${SFM.border}` }}>
          {/* Tabs roles */}
          <div style={{ display: 'flex', borderBottom: `1px solid ${SFM.border}` }}>
            {['Admin', 'Member', 'Viewer'].map((r, i) => (
              <div key={r} style={{ flex: 1, padding: '12px 8px', textAlign: 'center', fontSize: 12, fontWeight: 600, color: i === 0 ? SFM.accent : SFM.textMute, borderBottom: i === 0 ? `2px solid ${SFM.accent}` : '2px solid transparent', marginBottom: -1 }}>
                {r}
              </div>
            ))}
          </div>

          <div style={{ padding: 16, fontSize: 12, color: SFM.text, lineHeight: 1.5, background: SFM.bg, borderBottom: `1px solid ${SFM.border}` }}>
            Opérateur de confiance avec droits opérationnels complets. Ne peut pas transférer la propriété ni sortir d'argent du portefeuille.
          </div>

          {_TEAM_PERMISSIONS.map((g, gi) => (
            <div key={gi}>
              <div style={{ padding: '14px 16px 6px', fontSize: 10, fontWeight: 700, fontFamily: SFM.mono, letterSpacing: '0.08em', textTransform: 'uppercase', color: SFM.textMute, background: SFM.bg }}>{g.group}</div>
              {g.items.map((p, i) => (
                <div key={i} style={{ padding: '12px 16px', display: 'flex', gap: 12, alignItems: 'center', borderBottom: `1px solid ${SFM.border}` }}>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 12.5, fontWeight: 600 }}>{p.label}</div>
                    <div style={{ fontSize: 11, color: SFM.textMute, marginTop: 2, lineHeight: 1.4 }}>{p.desc}</div>
                  </div>
                  <_Toggle on={p.on} />
                </div>
              ))}
            </div>
          ))}

          {/* Owner-only locked */}
          <div style={{ margin: 16, background: SFM.bg, border: `1px solid ${SFM.border}`, borderRadius: 12, padding: 14 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
              <SFMI name="Lock" size={12} />
              <div style={{ fontSize: 12.5, fontWeight: 600 }}>Réservé au propriétaire</div>
            </div>
            <div style={{ fontSize: 11, color: SFM.textMute, lineHeight: 1.4 }}>Transfert, KYC, suppression — non délégables.</div>
          </div>
        </div>

        {/* Pending */}
        <div style={{ padding: '14px 16px 8px', fontSize: 10.5, fontWeight: 700, fontFamily: SFM.mono, letterSpacing: '0.08em', textTransform: 'uppercase', color: SFM.textMute }}>Invitations en attente</div>
        <div style={{ background: '#fff', padding: '24px 16px', textAlign: 'center', color: SFM.textMute, fontSize: 12.5, borderTop: `1px solid ${SFM.border}`, borderBottom: `1px solid ${SFM.border}` }}>
          Aucune invitation
        </div>

        {/* Transfer */}
        <div style={{ margin: 16, background: '#fff8ec', border: '1px solid #f4dba0', borderRadius: 14, padding: 14 }}>
          <div style={{ fontSize: 13, fontWeight: 700, color: '#a86a14', marginBottom: 4 }}>Transférer la propriété</div>
          <div style={{ fontSize: 11.5, color: SFM.text, marginBottom: 12, lineHeight: 1.5 }}>Cédez Nova Studio à un Admin existant. Vous serez rétrogradé(e).</div>
          <button style={{ background: '#fff', border: '1px solid #f4dba0', color: '#a86a14', padding: '7px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 8 }}>Démarrer le transfert</button>
        </div>

        <div style={{ height: 16 }} />
      </div>

      <_MobileBottomNav active="profile" role="enterprise" />
    </_MobileFrame>
  );
}

window.SoleilMessagerieListMobile = SoleilMessagerieListMobile;
window.SoleilMessagerieThreadMobile = SoleilMessagerieThreadMobile;
window.SoleilTeamMobile = SoleilTeamMobile;
