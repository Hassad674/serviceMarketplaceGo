// Lot F — Messagerie + Team (desktop + mobile)
// Reprend exactement la structure des écrans utilisateur fournis, avec l'identité Soleil.

const SF = window.S;
const SFI = window.SI;
const SFPortrait = window.Portrait;
const SFSidebar = window.SSidebar;
const SFTopbar = window.STopbar;

// ═════════════════════════════════════════════════════════════════
// MESSAGERIE — DESKTOP
// 3 colonnes : sidebar app | liste convs | conversation active
// ═════════════════════════════════════════════════════════════════

const FAKE_CONVS = [
  { id: 0, name: 'Camille Dubois', last: 'Parfait, je reviens vers vous demain pour le brief final', time: 'à l\'instant', unread: 2, online: true, role: 'Freelance', portrait: 0 },
  { id: 1, name: 'Léa Martinez', last: 'Nouvelle proposition d\'apport d\'affaires', time: '12 min', unread: 1, online: true, role: 'Apporteur', portrait: 1 },
  { id: 2, name: 'Studio Carbone', last: 'On a relu le devis, OK pour démarrer', time: '1 h', unread: 0, online: false, role: 'Agence', portrait: 2 },
  { id: 3, name: 'Hugo Lefèvre', last: 'Voici les premiers wireframes en pièce jointe', time: '3 h', unread: 0, online: false, role: 'Freelance', portrait: 3 },
  { id: 4, name: 'Nova SaaS', last: 'Bonjour, suite à notre échange…', time: 'hier', unread: 0, online: false, role: 'Entreprise', portrait: 4 },
  { id: 5, name: 'Inès Bouchard', last: 'Mission terminée — laissez votre avis', time: 'lun.', unread: 0, online: false, role: 'Freelance', portrait: 5 },
  { id: 6, name: 'Marc Rousseau', last: 'Merci pour le retour, je m\'en occupe', time: '24 avr.', unread: 0, online: false, role: 'Freelance', portrait: 0 },
];

const FAKE_THREAD = [
  { who: 'them', text: 'Bonjour, j\'ai bien reçu votre brief pour la refonte de Nova v2', time: '10:14' },
  { who: 'them', text: 'Quelques questions avant de vous faire une proposition chiffrée 👇', time: '10:14' },
  { who: 'me', text: 'Bonjour Camille, allez-y avec plaisir', time: '10:21' },
  { kind: 'card', proposal: { from: 'Camille Dubois', amount: '3 200,00 €', dueDate: 'Non définie', docs: 'Aucun', status: 'En attente' }},
  { who: 'me', text: 'Top, je relis ce soir et je reviens vers vous', time: '15:02' },
  { kind: 'event', label: 'Proposition acceptée', sub: 'Refonte Nova v2 — 3 200,00 €', tone: 'green' },
  { kind: 'event', label: 'Paiement confirmé', sub: 'Séquestre — 3 200,00 €', tone: 'blue' },
  { kind: 'event', label: 'Mission terminée', sub: 'Refonte Nova v2 — 3 200,00 €', tone: 'green' },
  { who: 'them', text: 'Parfait, je reviens vers vous demain pour le brief final', time: 'à l\'instant' },
];

function MsgConvItem({ c, active }) {
  return (
    <div style={{ padding: '12px 16px', display: 'flex', gap: 12, cursor: 'pointer', borderLeft: `3px solid ${active ? SF.accent : 'transparent'}`, background: active ? SF.accentSoft : 'transparent' }}>
      <div style={{ position: 'relative', flexShrink: 0 }}>
        <SFPortrait id={c.portrait} size={42} />
        {c.online ? <div style={{ position: 'absolute', bottom: 0, right: 0, width: 11, height: 11, borderRadius: '50%', background: '#3aa66b', border: '2px solid #fff' }} /> : null}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 6 }}>
          <div style={{ fontSize: 13.5, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{c.name}</div>
          <div style={{ fontSize: 11, color: SF.textMute, flexShrink: 0, fontFamily: SF.mono }}>{c.time}</div>
        </div>
        <div style={{ fontSize: 12, color: c.unread ? SF.text : SF.textMute, fontWeight: c.unread ? 500 : 400, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', marginTop: 2 }}>{c.last}</div>
      </div>
      {c.unread ? <div style={{ flexShrink: 0, alignSelf: 'center', background: SF.accent, color: '#fff', fontSize: 10, fontWeight: 700, padding: '2px 7px', borderRadius: 999 }}>{c.unread}</div> : null}
    </div>
  );
}

function MsgBubble({ msg }) {
  if (msg.kind === 'card') {
    const p = msg.proposal;
    return (
      <div style={{ alignSelf: 'flex-start', maxWidth: 460, background: '#fff', border: `1px solid ${SF.border}`, borderRadius: 16, padding: 16, marginBottom: 12, boxShadow: '0 1px 2px rgba(42,31,21,0.04)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12, paddingBottom: 12, borderBottom: `1px solid ${SF.border}` }}>
          <div style={{ width: 32, height: 32, borderRadius: 8, background: SF.accentSoft, display: 'flex', alignItems: 'center', justifyContent: 'center', color: SF.accentDeep }}>
            <SFI name="Briefcase" size={16} />
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 12.5, fontWeight: 600 }}>Proposition de {p.from}</div>
            <div style={{ fontSize: 11, color: SF.textMute }}>Refonte Nova v2</div>
          </div>
          <span style={{ fontSize: 10.5, fontWeight: 600, padding: '3px 9px', background: '#fff5e8', color: '#a86a14', borderRadius: 999 }}>{p.status}</span>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, marginBottom: 14 }}>
          {[
            { label: 'Montant total', value: p.amount, icon: 'Euro' },
            { label: 'Date limite', value: p.dueDate, icon: 'Clock' },
            { label: 'Documents', value: p.docs, icon: 'File' },
          ].map(s => (
            <div key={s.label}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 5, fontSize: 10, color: SF.textMute, fontFamily: SF.mono, letterSpacing: '0.04em', textTransform: 'uppercase', marginBottom: 4 }}>
                <SFI name={s.icon} size={11} />{s.label}
              </div>
              <div style={{ fontSize: 13.5, fontWeight: 600, fontFamily: SF.serif }}>{s.value}</div>
            </div>
          ))}
        </div>
        <button style={{ width: '100%', padding: '10px', background: SF.bg, border: `1px solid ${SF.border}`, borderRadius: 10, fontSize: 13, fontWeight: 600, color: SF.text, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6 }}>
          <SFI name="Edit" size={13} /> Modifier la proposition
        </button>
        <div style={{ textAlign: 'center', marginTop: 8, fontSize: 11.5, color: SF.accent, fontWeight: 600, cursor: 'pointer' }}>Voir les détails</div>
      </div>
    );
  }
  if (msg.kind === 'event') {
    const tones = {
      green: { bg: '#e8f5ec', dot: '#3aa66b', icon: 'CheckCircle' },
      blue: { bg: '#e8eef9', dot: '#3a6aa6', icon: 'Euro' },
      orange: { bg: '#fde9e3', dot: SF.accent, icon: 'Star' },
    };
    const t = tones[msg.tone] || tones.green;
    return (
      <div style={{ alignSelf: 'flex-start', maxWidth: 320, background: t.bg, borderRadius: 14, padding: '12px 16px', marginBottom: 8, display: 'flex', alignItems: 'center', gap: 10 }}>
        <div style={{ color: t.dot }}><SFI name={t.icon} size={18} /></div>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 12.5, fontWeight: 600 }}>{msg.label}</div>
          <div style={{ fontSize: 11, color: SF.textMute, marginTop: 1 }}>{msg.sub}</div>
        </div>
      </div>
    );
  }
  const me = msg.who === 'me';
  return (
    <div style={{ alignSelf: me ? 'flex-end' : 'flex-start', maxWidth: 480, marginBottom: 4 }}>
      <div style={{
        background: me ? SF.accent : '#fff',
        color: me ? '#fff' : SF.text,
        padding: '10px 14px',
        borderRadius: me ? '18px 18px 4px 18px' : '18px 18px 18px 4px',
        fontSize: 13.5,
        lineHeight: 1.45,
        border: me ? 'none' : `1px solid ${SF.border}`,
      }}>{msg.text}</div>
      <div style={{ fontSize: 10, color: SF.textMute, marginTop: 3, textAlign: me ? 'right' : 'left', fontFamily: SF.mono }}>{msg.time}</div>
    </div>
  );
}

function SoleilMessagerie() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SF.bg, fontFamily: SF.sans, color: SF.text }}>
      <SFSidebar active="msg" role="enterprise" />

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SFTopbar />

        <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>

          {/* ─── Colonne convs (gauche) ─── */}
          <div style={{ width: 360, borderRight: `1px solid ${SF.border}`, background: '#fff', display: 'flex', flexDirection: 'column' }}>
            <div style={{ padding: '20px 20px 14px', borderBottom: `1px solid ${SF.border}` }}>
              <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', marginBottom: 12 }}>
                <h1 style={{ fontFamily: SF.serif, fontSize: 24, fontWeight: 500, margin: 0, letterSpacing: '-0.02em' }}>Messages</h1>
                <button style={{ background: 'none', border: 'none', cursor: 'pointer', color: SF.accent, fontSize: 12.5, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 4 }}>
                  <SFI name="Edit" size={13} /> Nouveau
                </button>
              </div>

              <div style={{ display: 'flex', gap: 6, marginBottom: 12, overflow: 'auto' }}>
                {[
                  { id: 'all', label: 'Tous', active: true },
                  { id: 'agency', label: 'Agence' },
                  { id: 'free', label: 'Freelances/Apporteurs' },
                  { id: 'ent', label: 'Entreprise' },
                ].map(t => (
                  <span key={t.id} style={{ padding: '5px 11px', background: t.active ? SF.text : SF.bg, color: t.active ? '#fff' : SF.textMute, fontSize: 11.5, fontWeight: 600, borderRadius: 999, whiteSpace: 'nowrap', cursor: 'pointer' }}>{t.label}</span>
                ))}
              </div>

              <div style={{ position: 'relative' }}>
                <div style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: SF.textMute }}><SFI name="Search" size={14} /></div>
                <input placeholder="Rechercher une conversation…" style={{ width: '100%', padding: '9px 12px 9px 34px', background: SF.bg, border: `1px solid ${SF.border}`, borderRadius: 10, fontSize: 13, outline: 'none' }} />
              </div>
            </div>

            <div style={{ flex: 1, overflow: 'auto' }}>
              {FAKE_CONVS.map((c, i) => <MsgConvItem key={c.id} c={c} active={i === 0} />)}
            </div>
          </div>

          {/* ─── Conversation active (droite) ─── */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>

            {/* Header conv */}
            <div style={{ height: 72, borderBottom: `1px solid ${SF.border}`, background: '#fff', display: 'flex', alignItems: 'center', padding: '0 24px', gap: 14, flexShrink: 0 }}>
              <SFPortrait id={0} size={42} />
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 14.5, fontWeight: 600 }}>Camille Dubois</div>
                <div style={{ fontSize: 11.5, color: '#3aa66b', fontWeight: 500, display: 'flex', alignItems: 'center', gap: 5 }}>
                  <span style={{ width: 7, height: 7, borderRadius: '50%', background: '#3aa66b' }} />En ligne
                </div>
              </div>
              <button style={{ background: SF.accent, color: '#fff', border: 'none', padding: '9px 16px', fontSize: 13, fontWeight: 600, borderRadius: 10, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
                <SFI name="Plus" size={14} /> Proposer un projet
              </button>
              <button style={{ width: 38, height: 38, borderRadius: '50%', background: SF.bg, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SFI name="Phone" size={16} /></button>
              <button style={{ width: 38, height: 38, borderRadius: '50%', background: SF.bg, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SFI name="Video" size={16} /></button>
              <button style={{ width: 38, height: 38, borderRadius: '50%', background: SF.bg, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SFI name="MoreH" size={16} /></button>
            </div>

            {/* Thread */}
            <div style={{ flex: 1, overflow: 'auto', padding: '24px 32px', display: 'flex', flexDirection: 'column', gap: 4, background: SF.bg }}>
              <div style={{ alignSelf: 'center', fontSize: 11, color: SF.textMute, fontFamily: SF.mono, padding: '6px 0', marginBottom: 8 }}>— Aujourd'hui —</div>
              {FAKE_THREAD.map((m, i) => <MsgBubble key={i} msg={m} />)}

              {/* Mission terminée — call to review */}
              <div style={{ alignSelf: 'flex-start', maxWidth: 460, background: '#fff8ec', border: '1px solid #f4dba0', borderRadius: 14, padding: 14, marginTop: 8, display: 'flex', gap: 12, alignItems: 'center' }}>
                <div style={{ width: 36, height: 36, borderRadius: '50%', background: '#f4dba0', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#a86a14' }}><SFI name="Star" size={18} /></div>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 13, fontWeight: 600 }}>Mission terminée — laissez votre avis</div>
                  <div style={{ fontSize: 11.5, color: SF.textMute, marginTop: 1 }}>Aidez d'autres entreprises à choisir Camille</div>
                </div>
                <button style={{ background: SF.text, color: '#fff', border: 'none', padding: '8px 14px', fontSize: 12.5, fontWeight: 600, borderRadius: 10, cursor: 'pointer' }}>Évaluer →</button>
              </div>
            </div>

            {/* Composer */}
            <div style={{ background: '#fff', borderTop: `1px solid ${SF.border}`, padding: '14px 24px', display: 'flex', gap: 10, alignItems: 'center', flexShrink: 0 }}>
              <button style={{ width: 38, height: 38, borderRadius: '50%', background: SF.bg, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SF.textMute }}><SFI name="Paperclip" size={16} /></button>
              <button style={{ width: 38, height: 38, borderRadius: '50%', background: SF.bg, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', color: SF.textMute }}><SFI name="File" size={16} /></button>
              <input placeholder="Écrivez votre message…" style={{ flex: 1, padding: '11px 14px', background: SF.bg, border: `1px solid ${SF.border}`, borderRadius: 999, fontSize: 13.5, outline: 'none' }} />
              <button style={{ width: 38, height: 38, borderRadius: '50%', background: SF.accent, color: '#fff', border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}><SFI name="Send" size={15} /></button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ═════════════════════════════════════════════════════════════════
// TEAM — DESKTOP
// Header My team + members table + collapsible roles & permissions
// ═════════════════════════════════════════════════════════════════

const TEAM_MEMBERS = [
  { name: 'Élise Moreau', email: 'elise@nova.fr', role: 'Owner', title: '—', portrait: 1, you: true },
  { name: 'Tom Garcia', email: 'tom@nova.fr', role: 'Admin', title: 'Head of Ops', portrait: 0 },
  { name: 'Sarah Levy', email: 'sarah@nova.fr', role: 'Member', title: 'Product Manager', portrait: 4 },
  { name: 'Maxime Roy', email: 'maxime@nova.fr', role: 'Viewer', title: 'Stakeholder', portrait: 2 },
];

const TEAM_ROLES = [
  { id: 'admin', label: 'Admin', desc: 'Opérateur de confiance avec droits opérationnels complets : peut gérer l\'équipe, les annonces, les propositions, le KYC, les paramètres de facturation, et répondre aux avis. Ne peut pas transférer la propriété, supprimer l\'organisation, ni sortir d\'argent du portefeuille.' },
  { id: 'member', label: 'Member' },
  { id: 'viewer', label: 'Viewer' },
];

const TEAM_PERMISSIONS = [
  { group: 'Équipe', items: [
    { label: 'Voir l\'équipe', desc: 'Accéder à la liste des membres et invitations en attente.', on: true },
    { label: 'Inviter des membres', desc: 'Envoyer des invitations e-mail pour rejoindre l\'organisation.', on: true },
    { label: 'Gérer l\'équipe', desc: 'Modifier les rôles et titres des membres, retirer un membre.', on: true },
  ]},
  { group: 'Profil public', items: [
    { label: 'Modifier le profil prestataire', desc: 'Mettre à jour le profil public côté marketplace (logo, à propos, vidéo).', on: true },
    { label: 'Modifier le profil client', desc: 'Mettre à jour le profil public côté client — celui que les freelances voient.', on: true },
  ]},
  { group: 'Wallet', items: [
    { label: 'Voir le portefeuille', desc: 'Consulter le solde de l\'organisation et l\'historique des transferts.', on: true },
  ]},
  { group: 'Facturation', items: [
    { label: 'Voir la facturation', desc: 'Consulter les factures et l\'historique de paiement de l\'organisation.', on: true },
    { label: 'Gérer la facturation', desc: 'Modifier les moyens de paiement et les paramètres de facturation.', on: false },
  ]},
];

function Toggle({ on, locked }) {
  return (
    <div style={{ width: 38, height: 22, borderRadius: 999, background: locked ? '#e0d8c8' : (on ? SF.accent : '#d8d0c0'), position: 'relative', flexShrink: 0, cursor: locked ? 'not-allowed' : 'pointer', opacity: locked ? 0.5 : 1 }}>
      <div style={{ position: 'absolute', top: 2, left: on ? 18 : 2, width: 18, height: 18, borderRadius: '50%', background: '#fff', boxShadow: '0 1px 3px rgba(0,0,0,0.2)', transition: 'left 0.15s' }} />
    </div>
  );
}

function PermRow({ p, locked }) {
  return (
    <div style={{ padding: '14px 0', borderBottom: `1px solid ${SF.border}`, display: 'flex', gap: 16, alignItems: 'flex-start' }}>
      <div style={{ flex: 1 }}>
        <div style={{ fontSize: 13.5, fontWeight: 600, marginBottom: 3 }}>{p.label}</div>
        <div style={{ fontSize: 12, color: SF.textMute, lineHeight: 1.45 }}>{p.desc}</div>
      </div>
      <Toggle on={p.on} locked={locked} />
    </div>
  );
}

function SoleilTeam() {
  return (
    <div style={{ width: '100%', height: '100%', display: 'flex', background: SF.bg, fontFamily: SF.sans, color: SF.text }}>
      <SFSidebar active="team" role="enterprise" />

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <SFTopbar />

        <div style={{ flex: 1, overflow: 'auto', padding: '32px 48px' }}>
          <div style={{ maxWidth: 920, margin: '0 auto' }}>

            {/* Header — My team */}
            <div style={{ background: '#fff', border: `1px solid ${SF.border}`, borderRadius: 16, padding: '20px 24px', display: 'flex', alignItems: 'center', gap: 16, marginBottom: 24 }}>
              <div style={{ width: 52, height: 52, borderRadius: 14, background: SF.accentSoft, display: 'flex', alignItems: 'center', justifyContent: 'center', color: SF.accentDeep }}>
                <SFI name="Building" size={24} />
              </div>
              <div style={{ flex: 1 }}>
                <h1 style={{ fontFamily: SF.serif, fontSize: 26, fontWeight: 500, margin: 0, letterSpacing: '-0.01em' }}>Mon équipe</h1>
                <div style={{ fontSize: 13, color: SF.textMute, marginTop: 2 }}>Nova Studio · Entreprise</div>
              </div>
              <span style={{ padding: '6px 12px', background: SF.bg, border: `1px solid ${SF.border}`, borderRadius: 999, fontSize: 12, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6 }}>
                <SFI name="Users" size={13} /> {TEAM_MEMBERS.length} membres
              </span>
            </div>

            {/* Members */}
            <div style={{ marginBottom: 32 }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
                <h2 style={{ fontFamily: SF.serif, fontSize: 20, fontWeight: 500, margin: 0, letterSpacing: '-0.01em' }}>Membres</h2>
                <button style={{ background: SF.accent, color: '#fff', border: 'none', padding: '9px 16px', fontSize: 13, fontWeight: 600, borderRadius: 10, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 }}>
                  <SFI name="Plus" size={14} /> Inviter
                </button>
              </div>
              <div style={{ background: '#fff', border: `1px solid ${SF.border}`, borderRadius: 14, overflow: 'hidden' }}>
                <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr 1fr 80px', gap: 16, padding: '12px 20px', background: SF.bg, fontSize: 10.5, fontWeight: 700, fontFamily: SF.mono, letterSpacing: '0.08em', textTransform: 'uppercase', color: SF.textMute, borderBottom: `1px solid ${SF.border}` }}>
                  <div>Membre</div><div>Rôle</div><div>Titre</div><div style={{ textAlign: 'right' }}>Actions</div>
                </div>
                {TEAM_MEMBERS.map((m, i) => (
                  <div key={i} style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr 1fr 80px', gap: 16, padding: '14px 20px', alignItems: 'center', borderBottom: i === TEAM_MEMBERS.length - 1 ? 'none' : `1px solid ${SF.border}` }}>
                    <div style={{ display: 'flex', gap: 12, alignItems: 'center', minWidth: 0 }}>
                      <SFPortrait id={m.portrait} size={36} />
                      <div style={{ minWidth: 0 }}>
                        <div style={{ fontSize: 13.5, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6 }}>{m.name}{m.you ? <span style={{ fontSize: 10, padding: '1px 6px', background: SF.bg, borderRadius: 999, color: SF.textMute, fontWeight: 600 }}>vous</span> : null}</div>
                        <div style={{ fontSize: 11.5, color: SF.textMute, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{m.email}</div>
                      </div>
                    </div>
                    <div>
                      <span style={{ fontSize: 12, fontWeight: 600, padding: '4px 10px', background: m.role === 'Owner' ? '#fff5e8' : SF.bg, color: m.role === 'Owner' ? '#a86a14' : SF.text, borderRadius: 999, display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                        {m.role === 'Owner' ? <SFI name="Star" size={11} /> : null}{m.role}
                      </span>
                    </div>
                    <div style={{ fontSize: 12.5, color: SF.textMute }}>{m.title}</div>
                    <div style={{ textAlign: 'right' }}>
                      <button style={{ width: 30, height: 30, borderRadius: '50%', background: 'transparent', border: 'none', cursor: 'pointer', color: SF.textMute, opacity: m.you ? 0.3 : 1 }}><SFI name="MoreH" size={15} /></button>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Roles & permissions — collapsible card */}
            <div style={{ background: '#fff', border: `1px solid ${SF.border}`, borderRadius: 14, overflow: 'hidden', marginBottom: 24 }}>
              <div style={{ padding: '16px 20px', borderBottom: `1px solid ${SF.border}`, display: 'flex', alignItems: 'center', gap: 12 }}>
                <div style={{ width: 36, height: 36, borderRadius: 10, background: SF.accentSoft, display: 'flex', alignItems: 'center', justifyContent: 'center', color: SF.accentDeep, flexShrink: 0 }}>
                  <SFI name="Shield" size={16} />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ fontFamily: SF.serif, fontSize: 16, fontWeight: 600 }}>Rôles et permissions</div>
                  <div style={{ fontSize: 12, color: SF.textMute, marginTop: 1 }}>Voir et personnaliser ce que chaque rôle peut faire dans votre organisation.</div>
                </div>
                <SFI name="ChevronDown" size={18} />
              </div>

              <div style={{ padding: '0 24px 24px' }}>
                {/* Tabs roles */}
                <div style={{ display: 'flex', gap: 0, borderBottom: `1px solid ${SF.border}`, marginBottom: 20 }}>
                  {TEAM_ROLES.map((r, i) => (
                    <div key={r.id} style={{ flex: 1, padding: '14px 12px', textAlign: 'center', fontSize: 12.5, fontWeight: 600, color: i === 0 ? SF.accent : SF.textMute, borderBottom: i === 0 ? `2px solid ${SF.accent}` : '2px solid transparent', marginBottom: -1, cursor: 'pointer' }}>
                      {r.label}
                    </div>
                  ))}
                </div>

                <div style={{ fontSize: 13, color: SF.text, lineHeight: 1.55, marginBottom: 20, padding: '14px 16px', background: SF.bg, borderRadius: 10 }}>
                  {TEAM_ROLES[0].desc}
                </div>

                {TEAM_PERMISSIONS.map(g => (
                  <div key={g.group} style={{ marginBottom: 18 }}>
                    <div style={{ fontSize: 10.5, fontWeight: 700, fontFamily: SF.mono, letterSpacing: '0.08em', textTransform: 'uppercase', color: SF.textMute, marginBottom: 4 }}>{g.group}</div>
                    {g.items.map((p, i) => <PermRow key={i} p={p} />)}
                  </div>
                ))}

                {/* Owner-only locked permissions */}
                <div style={{ background: SF.bg, border: `1px solid ${SF.border}`, borderRadius: 12, padding: '16px 18px', marginTop: 8 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                    <SFI name="Lock" size={14} />
                    <div style={{ fontSize: 13, fontWeight: 600 }}>Permissions réservées au propriétaire</div>
                  </div>
                  <div style={{ fontSize: 12, color: SF.textMute, marginBottom: 14, lineHeight: 1.45 }}>Pour des raisons de sécurité, ces permissions sont verrouillées. Seul le propriétaire de l'organisation peut les exercer — elles ne sont pas délégables.</div>
                  {[
                    { label: 'Transférer la propriété', desc: 'Initier le flux de transfert pour céder l\'organisation à un autre admin.' },
                    { label: 'Personnaliser les rôles', desc: 'Modifier les permissions de chaque rôle (Admin, Member, Viewer).' },
                    { label: 'Demander un virement', desc: 'Sortir de l\'argent du portefeuille vers le compte Stripe connecté.' },
                    { label: 'Gérer le KYC', desc: 'Compléter et mettre à jour la vérification Stripe Connect.' },
                    { label: 'Supprimer l\'organisation', desc: 'Action irréversible.' },
                  ].map((p, i) => (
                    <div key={i} style={{ padding: '10px 0', borderBottom: i === 4 ? 'none' : `1px solid ${SF.border}` }}>
                      <div style={{ fontSize: 12.5, fontWeight: 600 }}>{p.label}</div>
                      <div style={{ fontSize: 11.5, color: SF.textMute, marginTop: 2 }}>{p.desc}</div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Pending invitations */}
            <div style={{ marginBottom: 24 }}>
              <h2 style={{ fontFamily: SF.serif, fontSize: 20, fontWeight: 500, margin: '0 0 14px', letterSpacing: '-0.01em' }}>Invitations en attente</h2>
              <div style={{ background: '#fff', border: `1px solid ${SF.border}`, borderRadius: 14, padding: '40px 20px', textAlign: 'center', color: SF.textMute, fontSize: 13 }}>
                Aucune invitation en attente
              </div>
            </div>

            {/* Transfer ownership */}
            <div style={{ background: '#fff8ec', border: '1px solid #f4dba0', borderRadius: 14, padding: 20 }}>
              <div style={{ fontSize: 14, fontWeight: 700, color: '#a86a14', marginBottom: 4 }}>Transférer la propriété</div>
              <div style={{ fontSize: 12.5, color: SF.text, marginBottom: 14, lineHeight: 1.5 }}>Cédez votre organisation à un Admin existant. Vous serez rétrogradé(e) en Admin une fois qu'il/elle aura accepté.</div>
              <button style={{ background: '#fff', border: '1px solid #f4dba0', color: '#a86a14', padding: '8px 16px', fontSize: 13, fontWeight: 600, borderRadius: 10, cursor: 'pointer' }}>Démarrer le transfert</button>
            </div>

          </div>
        </div>
      </div>
    </div>
  );
}

window.SoleilMessagerie = SoleilMessagerie;
window.SoleilTeam = SoleilTeam;
window.FAKE_CONVS = FAKE_CONVS;
window.FAKE_THREAD = FAKE_THREAD;
window.TEAM_MEMBERS = TEAM_MEMBERS;
window.TEAM_PERMISSIONS = TEAM_PERMISSIONS;
window.MsgConvItem = MsgConvItem;
window.MsgBubble = MsgBubble;
window.PermRow = PermRow;
window.Toggle = Toggle;
