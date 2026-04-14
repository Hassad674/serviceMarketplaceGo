"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { ArrowDown, ArrowUp, Check, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  EXPERTISE_DOMAIN_KEYS,
  type ExpertiseDomainKey,
  getMaxExpertiseForOrgType,
  isExpertiseDomainKey,
  orgTypeSupportsExpertise,
} from "../constants/expertise"
import { useUpdateExpertiseDomains } from "../hooks/use-update-expertise"

interface ExpertiseEditorProps {
  domains: string[] | undefined
  orgType: string | undefined
  readOnly?: boolean
  // Optional override for the save pipeline. When absent the editor
  // uses the legacy /api/v1/profile/expertise mutation via the
  // provider hook — that's the agency path. The freelance and
  // referrer pages pass their own persona-specific mutations through
  // this prop so a single component surface covers every persona
  // without cross-feature coupling.
  onSaveOverride?: (next: string[]) => Promise<void>
  savingOverride?: boolean
}

// The editor is the single React owner of the "currently picked list"
// while in edit mode. On Save the value is sent to the backend; the
// optimistic mutation immediately reflects the new list in the shared
// profile cache, so sibling views (public profile, read-only display)
// re-render with the new value without waiting for the network.
export function ExpertiseEditor({
  domains,
  orgType,
  readOnly = false,
  onSaveOverride,
  savingOverride,
}: ExpertiseEditorProps) {
  const t = useTranslations("profile.expertise")
  const tCommon = useTranslations("common")
  const maxDomains = getMaxExpertiseForOrgType(orgType)
  const legacyMutation = useUpdateExpertiseDomains()
  const saveFn = onSaveOverride ?? ((next: string[]) => legacyMutation.mutateAsync(next))
  const isSaving = savingOverride ?? legacyMutation.isPending

  const persisted = useMemo<ExpertiseDomainKey[]>(
    () => (domains ?? []).filter(isExpertiseDomainKey),
    [domains],
  )

  const [selected, setSelected] = useState<ExpertiseDomainKey[]>(persisted)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  // Re-sync local draft whenever the persisted list changes from
  // outside (e.g. another tab, server refresh, optimistic rollback).
  useEffect(() => {
    setSelected(persisted)
  }, [persisted])

  const isDirty = useMemo(
    () => !arraysEqual(selected, persisted),
    [selected, persisted],
  )
  const atMax = selected.length >= maxDomains

  const toggleDomain = useCallback(
    (key: ExpertiseDomainKey) => {
      setErrorMessage(null)
      setSelected((current) => {
        if (current.includes(key)) {
          return current.filter((entry) => entry !== key)
        }
        if (current.length >= maxDomains) return current
        return [...current, key]
      })
    },
    [maxDomains],
  )

  const moveDomain = useCallback(
    (key: ExpertiseDomainKey, direction: -1 | 1) => {
      setErrorMessage(null)
      setSelected((current) => {
        const index = current.indexOf(key)
        const target = index + direction
        if (index === -1 || target < 0 || target >= current.length) {
          return current
        }
        const next = [...current]
        next[index] = current[target]
        next[target] = key
        return next
      })
    },
    [],
  )

  const resetDraft = useCallback(() => {
    setErrorMessage(null)
    setSelected(persisted)
  }, [persisted])

  const handleSave = useCallback(async () => {
    setErrorMessage(null)
    try {
      await saveFn(selected)
    } catch (caught) {
      setErrorMessage(mapErrorToMessage(caught, t))
    }
  }, [saveFn, selected, t])

  if (readOnly && persisted.length === 0) return null
  if (!orgTypeSupportsExpertise(orgType)) return null

  return (
    <section
      aria-labelledby="expertise-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="expertise-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
        {!readOnly && (
          <p className="text-sm text-muted-foreground">
            {t("sectionSubtitle", { max: maxDomains })}
          </p>
        )}
      </header>

      {readOnly ? (
        <ReadOnlyPillList selected={persisted} />
      ) : (
        <EditableEditorBody
          state={{
            selected,
            atMax,
            maxDomains,
            isDirty,
            isSaving,
            errorMessage,
          }}
          actions={{
            toggleDomain,
            moveDomain,
            save: handleSave,
            reset: resetDraft,
          }}
          tCommon={tCommon}
        />
      )}
    </section>
  )
}

// ----- Internal sub-components -------------------------------------------

function ReadOnlyPillList({ selected }: { selected: ExpertiseDomainKey[] }) {
  const t = useTranslations("profile.expertise.domains")
  if (selected.length === 0) return null
  return (
    <ul className="flex flex-wrap gap-2" aria-label="expertise list">
      {selected.map((key) => (
        <li key={key}>
          <span className="inline-flex items-center rounded-full bg-primary/10 text-primary px-3 py-1 text-sm font-medium border border-primary/20">
            {t(key)}
          </span>
        </li>
      ))}
    </ul>
  )
}

type EditableEditorState = {
  selected: ExpertiseDomainKey[]
  atMax: boolean
  maxDomains: number
  isDirty: boolean
  isSaving: boolean
  errorMessage: string | null
}

type EditableEditorActions = {
  toggleDomain: (key: ExpertiseDomainKey) => void
  moveDomain: (key: ExpertiseDomainKey, direction: -1 | 1) => void
  save: () => void
  reset: () => void
}

type EditableEditorBodyProps = {
  state: EditableEditorState
  actions: EditableEditorActions
  tCommon: ReturnType<typeof useTranslations>
}

function EditableEditorBody({
  state,
  actions,
  tCommon,
}: EditableEditorBodyProps) {
  const { selected, atMax, maxDomains, isDirty, isSaving, errorMessage } = state
  const { toggleDomain, moveDomain, save, reset } = actions
  const t = useTranslations("profile.expertise")
  const tDomains = useTranslations("profile.expertise.domains")

  return (
    <div className="space-y-4">
      <SelectedDomainsList
        selected={selected}
        onMove={moveDomain}
        onToggle={toggleDomain}
      />

      <DomainPickerGrid
        selected={selected}
        atMax={atMax}
        onToggle={toggleDomain}
        tDomains={tDomains}
      />

      {/* Polite live region — counter update announced to assistive tech. */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <p
          aria-live="polite"
          className={cn(
            "text-xs",
            atMax ? "text-primary font-medium" : "text-muted-foreground",
          )}
        >
          {atMax
            ? t("maxReached", { max: maxDomains })
            : t("counter", { count: selected.length, max: maxDomains })}
        </p>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={reset}
            disabled={!isDirty || isSaving}
            className="rounded-md h-9 px-4 text-sm font-medium text-foreground hover:bg-muted transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {tCommon("cancel")}
          </button>
          <button
            type="button"
            onClick={save}
            disabled={!isDirty || isSaving}
            className="bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium hover:opacity-90 transition-opacity duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50 disabled:cursor-not-allowed inline-flex items-center gap-2"
          >
            {isSaving ? (
              <Loader2
                className="w-4 h-4 animate-spin"
                aria-hidden="true"
              />
            ) : (
              <Check className="w-4 h-4" aria-hidden="true" />
            )}
            {isSaving ? t("saving") : t("save")}
          </button>
        </div>
      </div>

      {errorMessage ? (
        <p
          role="alert"
          className="text-sm text-destructive bg-destructive/5 border border-destructive/20 rounded-md px-3 py-2"
        >
          {errorMessage}
        </p>
      ) : null}
    </div>
  )
}

type SelectedDomainsListProps = {
  selected: ExpertiseDomainKey[]
  onMove: (key: ExpertiseDomainKey, direction: -1 | 1) => void
  onToggle: (key: ExpertiseDomainKey) => void
}

function SelectedDomainsList({
  selected,
  onMove,
  onToggle,
}: SelectedDomainsListProps) {
  const t = useTranslations("profile.expertise")
  const tDomains = useTranslations("profile.expertise.domains")

  if (selected.length === 0) {
    return (
      <p className="text-sm text-muted-foreground italic">
        {t("emptyPrivate")}
      </p>
    )
  }

  return (
    <ol
      aria-label={t("selectedListLabel")}
      className="flex flex-col gap-2"
    >
      {selected.map((key, index) => (
        <li
          key={key}
          className="flex items-center justify-between gap-3 rounded-md border border-border bg-muted/40 px-3 py-2"
        >
          <span className="text-sm font-medium text-foreground truncate">
            <span className="text-muted-foreground mr-2">{index + 1}.</span>
            {tDomains(key)}
          </span>
          <span className="flex items-center gap-1 shrink-0">
            <IconActionButton
              label={t("moveUp", { label: tDomains(key) })}
              disabled={index === 0}
              onClick={() => onMove(key, -1)}
              icon={<ArrowUp className="w-4 h-4" aria-hidden="true" />}
            />
            <IconActionButton
              label={t("moveDown", { label: tDomains(key) })}
              disabled={index === selected.length - 1}
              onClick={() => onMove(key, 1)}
              icon={<ArrowDown className="w-4 h-4" aria-hidden="true" />}
            />
            <button
              type="button"
              onClick={() => onToggle(key)}
              aria-label={t("remove", { label: tDomains(key) })}
              className="inline-flex items-center justify-center w-8 h-8 rounded-md text-destructive hover:bg-destructive/10 transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
            >
              <span aria-hidden="true" className="text-lg leading-none">
                &times;
              </span>
            </button>
          </span>
        </li>
      ))}
    </ol>
  )
}

type DomainPickerGridProps = {
  selected: ExpertiseDomainKey[]
  atMax: boolean
  onToggle: (key: ExpertiseDomainKey) => void
  tDomains: ReturnType<typeof useTranslations>
}

function DomainPickerGrid({
  selected,
  atMax,
  onToggle,
  tDomains,
}: DomainPickerGridProps) {
  return (
    <div
      role="group"
      aria-label="expertise domain picker"
      className="flex flex-wrap gap-2"
    >
      {EXPERTISE_DOMAIN_KEYS.map((key) => {
        const isSelected = selected.includes(key)
        const isDisabled = !isSelected && atMax
        return (
          <button
            key={key}
            type="button"
            onClick={() => onToggle(key)}
            aria-pressed={isSelected}
            disabled={isDisabled}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium border transition-all duration-150",
              "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
              isSelected
                ? "bg-primary text-primary-foreground border-primary shadow-sm hover:opacity-90"
                : "bg-background text-foreground border-border hover:border-primary/60 hover:bg-muted",
              isDisabled && "opacity-50 cursor-not-allowed hover:border-border hover:bg-background",
            )}
          >
            {isSelected ? (
              <Check className="w-3.5 h-3.5" aria-hidden="true" />
            ) : null}
            {tDomains(key)}
          </button>
        )
      })}
    </div>
  )
}

type IconActionButtonProps = {
  label: string
  disabled?: boolean
  onClick: () => void
  icon: React.ReactNode
}

function IconActionButton({
  label,
  disabled = false,
  onClick,
  icon,
}: IconActionButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      className="inline-flex items-center justify-center w-8 h-8 rounded-md border border-transparent text-muted-foreground hover:bg-muted hover:text-foreground transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-40 disabled:cursor-not-allowed"
    >
      {icon}
    </button>
  )
}

// ----- Helpers -----------------------------------------------------------

function arraysEqual(
  a: readonly ExpertiseDomainKey[],
  b: readonly ExpertiseDomainKey[],
): boolean {
  if (a.length !== b.length) return false
  for (let index = 0; index < a.length; index += 1) {
    if (a[index] !== b[index]) return false
  }
  return true
}

function mapErrorToMessage(
  error: unknown,
  t: ReturnType<typeof useTranslations>,
): string {
  if (error instanceof Error && "status" in error) {
    const status = (error as Error & { status?: number }).status
    if (status === 403) return t("errorForbidden")
    if (status === 400) return t("errorValidation")
  }
  return t("errorGeneric")
}
