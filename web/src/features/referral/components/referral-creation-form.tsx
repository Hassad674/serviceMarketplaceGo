"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Loader2, Send } from "lucide-react"

import { useCreateReferral } from "../hooks/use-referrals"
import type { CreateReferralInput, SnapshotToggles } from "../types"
import { ClientPicker, type ClientPickerSelection } from "./client-picker"
import {
  ProviderPicker,
  type ProviderPickerSelection,
} from "./provider-picker"

const DEFAULT_TOGGLES: SnapshotToggles = {
  include_expertise: true,
  include_experience: true,
  include_rating: true,
  include_pricing: true,
  include_region: true,
  include_languages: true,
  include_availability: true,
}

// ReferralCreationForm renders the full intro creation surface as a single
// page (V1). A future iteration can split it into a multi-step wizard if
// the field count grows. The form intentionally keeps the rate slider close
// to the messages so the apporteur can think holistically about the deal.
//
// Party selection goes through dedicated pickers instead of raw UUIDs:
//  - Provider: searchable dropdown over freelances + agences from the
//    marketplace's public search (filters by name client-side).
//  - Client: pickable from the apporteur's existing conversations with
//    enterprises — cold intro is not supported on purpose.
export function ReferralCreationForm() {
  const router = useRouter()
  const create = useCreateReferral()

  const [provider, setProvider] = useState<ProviderPickerSelection | null>(null)
  const [client, setClient] = useState<ClientPickerSelection | null>(null)
  const [ratePct, setRatePct] = useState(5)
  const [durationMonths, setDurationMonths] = useState(6)
  const [pitchProvider, setPitchProvider] = useState("")
  const [pitchClient, setPitchClient] = useState("")
  const [toggles, setToggles] = useState<SnapshotToggles>(DEFAULT_TOGGLES)
  const [submitError, setSubmitError] = useState<string | null>(null)

  function update<K extends keyof SnapshotToggles>(key: K) {
    return (next: boolean) => setToggles((t) => ({ ...t, [key]: next }))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitError(null)

    if (!provider) {
      setSubmitError("Sélectionnez un prestataire dans la liste.")
      return
    }
    if (!client) {
      setSubmitError("Sélectionnez un client parmi vos conversations.")
      return
    }
    if (ratePct < 0 || ratePct > 50) {
      setSubmitError("Le taux doit être compris entre 0 et 50 %.")
      return
    }
    if (!pitchProvider.trim() || !pitchClient.trim()) {
      setSubmitError("Les deux messages d'introduction sont requis.")
      return
    }

    const payload: CreateReferralInput = {
      provider_id: provider.userId,
      client_id: client.userId,
      rate_pct: ratePct,
      duration_months: durationMonths,
      intro_message_provider: pitchProvider.trim(),
      intro_message_client: pitchClient.trim(),
      snapshot_toggles: toggles,
    }

    try {
      const created = await create.mutateAsync(payload)
      router.push(`/referrals/${created.id}`)
    } catch (err) {
      setSubmitError(
        err instanceof Error
          ? err.message
          : "Une erreur est survenue lors de la création de l'intro.",
      )
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="mx-auto flex max-w-3xl flex-col gap-6"
    >
      <Section
        title="1 — Les parties à mettre en relation"
        description="Cherchez un prestataire par nom, puis choisissez un client parmi vos conversations existantes. Vous ne pouvez introduire qu'un client avec qui vous avez déjà échangé."
      >
        <ProviderPicker value={provider} onChange={setProvider} />
        <ClientPicker value={client} onChange={setClient} />
      </Section>

      <Section
        title="2 — Termes de l'apport"
        description="Le client ne verra jamais le taux. Il est négocié uniquement entre vous et le prestataire."
      >
        <Field
          label={`Commission : ${ratePct.toFixed(ratePct % 1 === 0 ? 0 : 1)} %`}
        >
          <input
            type="range"
            min={0}
            max={30}
            step={0.5}
            value={ratePct}
            onChange={(e) => setRatePct(parseFloat(e.target.value))}
            className="w-full accent-rose-500"
          />
          <div className="mt-1 flex justify-between text-xs text-slate-500">
            <span>0 %</span>
            <span>15 %</span>
            <span>30 %</span>
          </div>
        </Field>
        <Field label="Durée d'exclusivité">
          <select
            value={durationMonths}
            onChange={(e) => setDurationMonths(parseInt(e.target.value, 10))}
            className="w-full rounded-lg border border-slate-300 px-4 py-2 text-sm focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-100"
          >
            {[3, 6, 9, 12, 18, 24].map((n) => (
              <option key={n} value={n}>
                {n} mois
              </option>
            ))}
          </select>
        </Field>
      </Section>

      <Section
        title="3 — Champs révélés sur le profil du prestataire"
        description="Avant l'acceptation, le client verra une carte anonyme du prestataire avec uniquement les champs que vous cochez."
      >
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <ToggleRow
            label="Domaines d'expertise"
            checked={toggles.include_expertise}
            onChange={update("include_expertise")}
          />
          <ToggleRow
            label="Années d'expérience"
            checked={toggles.include_experience}
            onChange={update("include_experience")}
          />
          <ToggleRow
            label="Notation moyenne"
            checked={toggles.include_rating}
            onChange={update("include_rating")}
          />
          <ToggleRow
            label="Fourchette tarifaire"
            checked={toggles.include_pricing}
            onChange={update("include_pricing")}
          />
          <ToggleRow
            label="Région"
            checked={toggles.include_region}
            onChange={update("include_region")}
          />
          <ToggleRow
            label="Langues"
            checked={toggles.include_languages}
            onChange={update("include_languages")}
          />
          <ToggleRow
            label="Disponibilité"
            checked={toggles.include_availability}
            onChange={update("include_availability")}
          />
        </div>
      </Section>

      <Section
        title="4 — Vos messages d'introduction"
        description="Le prestataire et le client recevront chacun un mot personnalisé."
      >
        <Field label="Mot pour le prestataire">
          <textarea
            value={pitchProvider}
            onChange={(e) => setPitchProvider(e.target.value)}
            rows={3}
            maxLength={2000}
            placeholder="Ex : Ce client a un projet de refonte branding qui colle parfaitement à ton positionnement."
            className="w-full rounded-lg border border-slate-300 px-4 py-2 text-sm focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-100"
            required
          />
        </Field>
        <Field label="Mot pour le client">
          <textarea
            value={pitchClient}
            onChange={(e) => setPitchClient(e.target.value)}
            rows={3}
            maxLength={2000}
            placeholder="Ex : Voici un prestataire avec qui je travaille depuis trois ans, parfait pour votre besoin."
            className="w-full rounded-lg border border-slate-300 px-4 py-2 text-sm focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-100"
            required
          />
        </Field>
      </Section>

      {submitError && (
        <div
          role="alert"
          className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700"
        >
          {submitError}
        </div>
      )}

      <button
        type="submit"
        disabled={create.isPending}
        className="inline-flex items-center justify-center gap-2 self-end rounded-lg bg-rose-500 px-6 py-2.5 text-sm font-medium text-white shadow-sm transition hover:bg-rose-600 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {create.isPending ? (
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        ) : (
          <Send className="h-4 w-4" aria-hidden="true" />
        )}
        Envoyer l&rsquo;introduction
      </button>
    </form>
  )
}

interface SectionProps {
  title: string
  description: string
  children: React.ReactNode
}

function Section({ title, description, children }: SectionProps) {
  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <header className="mb-4">
        <h2 className="text-base font-semibold text-slate-900">{title}</h2>
        <p className="mt-1 text-sm text-slate-500">{description}</p>
      </header>
      <div className="space-y-4">{children}</div>
    </section>
  )
}

interface FieldProps {
  label: string
  children: React.ReactNode
}

function Field({ label, children }: FieldProps) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-sm font-medium text-slate-700">
        {label}
      </span>
      {children}
    </label>
  )
}

interface ToggleRowProps {
  label: string
  checked: boolean
  onChange: (next: boolean) => void
}

function ToggleRow({ label, checked, onChange }: ToggleRowProps) {
  return (
    <label className="flex items-center gap-3 rounded-lg border border-slate-200 px-3 py-2 text-sm">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 rounded border-slate-300 text-rose-500 focus:ring-rose-500"
      />
      <span className="text-slate-700">{label}</span>
    </label>
  )
}
