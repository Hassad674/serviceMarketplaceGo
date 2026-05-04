// Lot B mobile — Wallet · Factures · Profil facturation · Stripe Connect
const SBM = window.S;
const SBMI = window.SI;
const { MobileFrame, MobileHeader, MobileBottomNav, MobileSegmented } = window;

// ─── BM1 — Wallet (portefeuille) ─────────────────────────────
function SoleilWalletMobile() {
  return (
    <MobileFrame url="atelier.fr/wallet">
      <MobileHeader title="Portefeuille" />
      <div style={{ flex: 1, overflow: 'auto', padding: '14px 14px 90px' }}>
        {/* Balance hero */}
        <div style={{ background: SBM.text, color: '#fff', borderRadius: 16, padding: 20, marginBottom: 12, position: 'relative', overflow: 'hidden' }}>
          <div style={{ position: 'absolute', top: -30, right: -30, width: 160, height: 160, borderRadius: '50%', background: 'radial-gradient(circle, rgba(232,93,74,0.4), transparent 70%)' }} />
          <div style={{ position: 'relative' }}>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.6)', letterSpacing: '0.06em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 6 }}>Solde disponible</div>
            <div style={{ fontFamily: SBM.serif, fontSize: 38, fontWeight: 500, letterSpacing: '-0.025em', lineHeight: 1, marginBottom: 12 }}>3 240,80 €</div>
            <button style={{ background: '#fff', color: SBM.text, border: 'none', padding: '10px 18px', fontSize: 12.5, fontWeight: 600, borderRadius: 999, display: 'inline-flex', alignItems: 'center', gap: 6 }}><SBMI name="Send" size={13} /> Virer sur mon compte</button>
          </div>
        </div>

        {/* En séquestre + ce mois */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 18 }}>
          <div style={{ background: '#fff', border: `1px solid ${SBM.border}`, borderRadius: 12, padding: 14 }}>
            <div style={{ fontSize: 10.5, color: SBM.textMute, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 4 }}>En séquestre</div>
            <div style={{ fontFamily: SBM.serif, fontSize: 20, fontWeight: 500, letterSpacing: '-0.02em' }}>1 800 €</div>
            <div style={{ fontSize: 10.5, color: SBM.textSubtle, marginTop: 2 }}>1 jalon Nova</div>
          </div>
          <div style={{ background: '#fff', border: `1px solid ${SBM.border}`, borderRadius: 12, padding: 14 }}>
            <div style={{ fontSize: 10.5, color: SBM.textMute, letterSpacing: '0.05em', textTransform: 'uppercase', fontWeight: 600, marginBottom: 4 }}>Ce mois</div>
            <div style={{ fontFamily: SBM.serif, fontSize: 20, fontWeight: 500, letterSpacing: '-0.02em' }}>4 920 €</div>
            <div style={{ fontSize: 10.5, color: SBM.green, marginTop: 2 }}>+ 12 % vs avril</div>
          </div>
        </div>

        {/* Activité */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 10 }}>
          <div style={{ fontSize: 12, fontWeight: 700, letterSpacing: '0.06em', textTransform: 'uppercase', color: SBM.textMute }}>Activité récente</div>
          <a style={{ fontSize: 11.5, color: SBM.accent, fontWeight: 600 }}>Tout voir</a>
        </div>
        <div style={{ background: '#fff', border: `1px solid ${SBM.border}`, borderRadius: 12, overflow: 'hidden' }}>
          {[
            { kind: 'in', l: 'Versement Memo Bank', d: 'Aujourd\'hui · jalon validé', amt: '+ 1 800 €', color: SBM.green },
            { kind: 'out', l: 'Virement vers Qonto', d: 'Hier · IBAN ••2381', amt: '− 2 500 €', color: SBM.text },
            { kind: 'in', l: 'Versement Doctolib', d: '3 mai · facture FA-202', amt: '+ 2 400 €', color: SBM.green },
            { kind: 'fee', l: 'Frais plateforme', d: '3 mai · 5 % de 2 400 €', amt: '− 120 €', color: SBM.textMute },
            { kind: 'in', l: 'Versement Lydia', d: '28 avr · jalon validé', amt: '+ 720 €', color: SBM.green },
          ].map((t, i, arr) => (
            <div key={i} style={{ padding: '13px 14px', borderBottom: i < arr.length - 1 ? `1px solid ${SBM.border}` : 'none', display: 'flex', alignItems: 'center', gap: 12 }}>
              <div style={{ width: 32, height: 32, borderRadius: '50%', background: t.kind === 'in' ? SBM.greenSoft : t.kind === 'out' ? SBM.bg : SBM.bg, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <SBMI name={t.kind === 'in' ? 'ArrowDown' : 'ArrowUp'} size={13} />
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 1 }}>{t.l}</div>
                <div style={{ fontSize: 10.5, color: SBM.textMute }}>{t.d}</div>
              </div>
              <div style={{ fontSize: 13, fontWeight: 700, color: t.color, fontFamily: SBM.serif }}>{t.amt}</div>
            </div>
          ))}
        </div>
      </div>
      <MobileBottomNav active="wallet" role="freelancer" />
    </MobileFrame>
  );
}

// ─── BM2 — Factures ──────────────────────────────────────────
function SoleilInvoicesMobile() {
  const [tab, setTab] = React.useState(0);
  return (
    <MobileFrame url="atelier.fr/factures">
      <MobileHeader title="Factures" action={<button style={{ background: SBM.text, color: '#fff', border: 'none', padding: '7px 14px', fontSize: 11.5, fontWeight: 600, borderRadius: 999, display: 'flex', alignItems: 'center', gap: 5 }}><SBMI name="Plus" size={11} /> Émettre</button>} />
      <div style={{ padding: '12px 14px', flexShrink: 0, background: '#fff', borderBottom: `1px solid ${SBM.border}` }}>
        <MobileSegmented items={['En cours · 3', 'Payées · 24', 'Brouillons']} active={tab} />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '12px 14px 90px' }}>
        {[
          { num: 'FA-204', client: 'Nova', amt: '1 800 €', date: 'Émise hier', status: 'En attente', kind: 'pending' },
          { num: 'FA-203', client: 'Memo Bank', amt: '2 100 €', date: 'Émise 3 mai', status: 'En attente', kind: 'pending' },
          { num: 'FA-202', client: 'Doctolib', amt: '2 400 €', date: 'Payée 3 mai', status: 'Payée', kind: 'paid' },
          { num: 'FA-201', client: 'Lydia', amt: '720 €', date: 'Payée 28 avr', status: 'Payée', kind: 'paid' },
          { num: 'FA-200', client: 'BlaBlaCar', amt: '4 200 €', date: 'Payée 22 avr', status: 'Payée', kind: 'paid' },
        ].map((inv, i) => (
          <div key={i} style={{ background: '#fff', border: `1px solid ${SBM.border}`, borderRadius: 12, padding: 14, marginBottom: 8 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 6 }}>
              <div>
                <div style={{ fontSize: 11, fontFamily: SBM.mono, color: SBM.textMute, marginBottom: 2 }}>{inv.num}</div>
                <div style={{ fontSize: 14, fontWeight: 600, fontFamily: SBM.serif }}>{inv.client}</div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{ fontFamily: SBM.serif, fontSize: 17, fontWeight: 600, letterSpacing: '-0.015em' }}>{inv.amt}</div>
                <span style={{ fontSize: 10.5, padding: '2px 8px', background: inv.kind === 'paid' ? SBM.greenSoft : SBM.amberSoft, color: inv.kind === 'paid' ? SBM.green : SBM.amber, borderRadius: 999, fontWeight: 700, marginTop: 4, display: 'inline-block' }}>{inv.status}</span>
              </div>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingTop: 10, borderTop: `1px solid ${SBM.border}`, marginTop: 8 }}>
              <div style={{ fontSize: 11, color: SBM.textMute }}>{inv.date}</div>
              <div style={{ display: 'flex', gap: 6 }}>
                <button style={{ width: 30, height: 30, borderRadius: '50%', background: SBM.bg, border: 'none' }}><SBMI name="Download" size={12} /></button>
                <button style={{ background: SBM.bg, border: 'none', padding: '6px 12px', fontSize: 11.5, fontWeight: 600, borderRadius: 999 }}>Voir</button>
              </div>
            </div>
          </div>
        ))}
      </div>
      <MobileBottomNav active="wallet" role="freelancer" />
    </MobileFrame>
  );
}

// ─── BM3 — Profil de facturation ─────────────────────────────
function SoleilBillingProfileMobile() {
  return (
    <MobileFrame url="atelier.fr/facturation">
      <MobileHeader title="Profil de facturation" back />
      <div style={{ flex: 1, overflow: 'auto', padding: '14px 14px 24px' }}>
        <div style={{ fontSize: 12.5, color: SBM.textMute, marginBottom: 12, fontFamily: SBM.serif, fontStyle: 'italic', lineHeight: 1.5 }}>Ces informations apparaissent sur toutes tes factures émises depuis Atelier.</div>

        {/* Statut */}
        <div style={{ background: '#fff', border: `1px solid ${SBM.border}`, borderRadius: 12, padding: 16, marginBottom: 10 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 12 }}>Statut juridique</div>
          <div style={{ marginBottom: 10 }}>
            <label style={{ fontSize: 11, color: SBM.textMute, display: 'block', marginBottom: 4, fontWeight: 600 }}>Forme</label>
            <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap' }}>
              {[{ l: 'Micro-entreprise', sel: true }, { l: 'EURL' }, { l: 'SASU' }, { l: 'Salarié porté' }].map((s, i) => (
                <span key={i} style={{ fontSize: 11.5, padding: '6px 12px', background: s.sel ? SBM.text : '#fff', color: s.sel ? '#fff' : SBM.text, border: s.sel ? 'none' : `1px solid ${SBM.border}`, borderRadius: 999, fontWeight: 600 }}>{s.l}</span>
              ))}
            </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
            <div>
              <label style={{ fontSize: 11, color: SBM.textMute, display: 'block', marginBottom: 4, fontWeight: 600 }}>SIRET</label>
              <input defaultValue="892 451 783 00012" style={{ width: '100%', border: `1px solid ${SBM.borderStrong}`, borderRadius: 8, padding: '9px 11px', fontSize: 12.5, fontFamily: SBM.mono, outline: 'none' }} />
            </div>
            <div>
              <label style={{ fontSize: 11, color: SBM.textMute, display: 'block', marginBottom: 4, fontWeight: 600 }}>TVA intra.</label>
              <input defaultValue="Non applicable" style={{ width: '100%', border: `1px solid ${SBM.borderStrong}`, borderRadius: 8, padding: '9px 11px', fontSize: 12.5, color: SBM.textMute, outline: 'none' }} />
            </div>
          </div>
        </div>

        {/* Adresse */}
        <div style={{ background: '#fff', border: `1px solid ${SBM.border}`, borderRadius: 12, padding: 16, marginBottom: 10 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 12 }}>Adresse de facturation</div>
          <input defaultValue="Élise Marchand" style={{ width: '100%', border: `1px solid ${SBM.borderStrong}`, borderRadius: 8, padding: '9px 11px', fontSize: 12.5, outline: 'none', marginBottom: 8 }} />
          <input defaultValue="14 rue de Belleville" style={{ width: '100%', border: `1px solid ${SBM.borderStrong}`, borderRadius: 8, padding: '9px 11px', fontSize: 12.5, outline: 'none', marginBottom: 8 }} />
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: 8 }}>
            <input defaultValue="75020" style={{ border: `1px solid ${SBM.borderStrong}`, borderRadius: 8, padding: '9px 11px', fontSize: 12.5, outline: 'none' }} />
            <input defaultValue="Paris" style={{ border: `1px solid ${SBM.borderStrong}`, borderRadius: 8, padding: '9px 11px', fontSize: 12.5, outline: 'none' }} />
          </div>
        </div>

        {/* Mentions */}
        <div style={{ background: '#fff', border: `1px solid ${SBM.border}`, borderRadius: 12, padding: 16, marginBottom: 14 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 4 }}>Mentions légales</div>
          <div style={{ fontSize: 11.5, color: SBM.textMute, marginBottom: 10, fontStyle: 'italic', fontFamily: SBM.serif }}>Ajoutées en bas de chaque facture.</div>
          <textarea defaultValue="TVA non applicable, art. 293 B du CGI. Dispensé d'immatriculation au RCS et au répertoire des métiers." style={{ width: '100%', minHeight: 76, border: `1px solid ${SBM.borderStrong}`, borderRadius: 8, padding: '9px 11px', fontSize: 12, fontFamily: SBM.sans, outline: 'none', lineHeight: 1.5, resize: 'vertical' }} />
        </div>

        <button style={{ width: '100%', background: SBM.text, color: '#fff', border: 'none', padding: '13px', fontSize: 13, fontWeight: 600, borderRadius: 999 }}>Enregistrer les modifications</button>
      </div>
    </MobileFrame>
  );
}

// ─── BM4 — Stripe Connect (paiement) ─────────────────────────
function SoleilStripeConnectMobile() {
  return (
    <MobileFrame url="atelier.fr/paiement">
      <MobileHeader title="Infos de paiement" back />
      <div style={{ flex: 1, overflow: 'auto', padding: '14px 14px 24px' }}>
        {/* Statut Stripe */}
        <div style={{ background: SBM.greenSoft, border: `1px solid ${SBM.green}`, borderRadius: 12, padding: 16, marginBottom: 12, display: 'flex', gap: 12, alignItems: 'flex-start' }}>
          <div style={{ width: 36, height: 36, borderRadius: '50%', background: SBM.green, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
            <SBMI name="Verified" size={18} />
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 13, fontWeight: 700, color: SBM.green, marginBottom: 3 }}>Compte Stripe vérifié</div>
            <div style={{ fontSize: 11.5, color: SBM.text, lineHeight: 1.5 }}>Tu peux recevoir des paiements et émettre des virements. Les fonds arrivent sous 2 jours ouvrés.</div>
          </div>
        </div>

        {/* IBAN */}
        <div style={{ background: '#fff', border: `1px solid ${SBM.border}`, borderRadius: 12, padding: 16, marginBottom: 10 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
            <div style={{ fontSize: 13, fontWeight: 700 }}>Compte de virement</div>
            <button style={{ background: 'transparent', border: `1px solid ${SBM.border}`, padding: '5px 11px', fontSize: 11.5, fontWeight: 600, borderRadius: 999 }}>Modifier</button>
          </div>
          <div style={{ fontSize: 12, color: SBM.textMute, marginBottom: 4 }}>Qonto · Élise Marchand</div>
          <div style={{ fontSize: 13.5, fontFamily: SBM.mono, color: SBM.text, letterSpacing: '0.04em' }}>FR76 •••• •••• 2381</div>
        </div>

        {/* Identité */}
        <div style={{ background: '#fff', border: `1px solid ${SBM.border}`, borderRadius: 12, padding: 16, marginBottom: 10 }}>
          <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 10 }}>Identité vérifiée</div>
          {[
            ['Pièce d\'identité', 'CNI · Vérifiée'],
            ['Justificatif domicile', 'Quittance · Vérifiée'],
            ['SIRET', '892 451 783 00012'],
          ].map(([l, v], i, arr) => (
            <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '9px 0', borderBottom: i < arr.length - 1 ? `1px solid ${SBM.border}` : 'none' }}>
              <span style={{ fontSize: 12.5, color: SBM.textMute }}>{l}</span>
              <span style={{ fontSize: 12.5, fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: 5 }}><SBMI name="CheckCircle" size={12} /> {v}</span>
            </div>
          ))}
        </div>

        {/* Frais plateforme */}
        <div style={{ background: SBM.bg, borderRadius: 12, padding: 14, fontSize: 11.5, color: SBM.textMute, lineHeight: 1.6 }}>
          <strong style={{ color: SBM.text }}>Frais Atelier — 5 %</strong> prélevés à la validation de chaque jalon. Frais Stripe inclus, pas de frais cachés.
        </div>
      </div>
    </MobileFrame>
  );
}

window.SoleilWalletMobile = SoleilWalletMobile;
window.SoleilInvoicesMobile = SoleilInvoicesMobile;
window.SoleilBillingProfileMobile = SoleilBillingProfileMobile;
window.SoleilStripeConnectMobile = SoleilStripeConnectMobile;
