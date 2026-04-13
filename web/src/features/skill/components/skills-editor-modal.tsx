"use client"

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react"
import { createPortal } from "react-dom"
import {
  ArrowDown,
  ArrowUp,
  Check,
  Loader2,
  X,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { ApiError } from "@/shared/lib/api-client"
import { ALL_EXPERTISE_DOMAIN_KEYS } from "../constants"
import { useProfileSkills } from "../hooks/use-profile-skills"
import { useUpdateProfileSkills } from "../hooks/use-update-profile-skills"
import type { SkillResponse } from "../types"
import { ExpertisePanel } from "./expertise-panel"
import { PopularSkillsRow } from "./popular-skills-row"
import { SkillSearchBar } from "./skill-search-bar"

interface SkillsEditorModalProps {
  open: boolean
  onClose: () => void
  expertiseKeys: string[]
  maxSkills: number
}

type DraftSkill = {
  skill_text: string
  display_text: string
}

// Modal that owns the editing session. While open, the "draft" of
// the user's skill list lives entirely in local state; nothing is
// persisted until the user hits Save. Closing the modal discards the
// draft — a deliberate contract so canceling is always safe.
export function SkillsEditorModal({
  open,
  onClose,
  expertiseKeys,
  maxSkills,
}: SkillsEditorModalProps) {
  const t = useTranslations("profile.skills")
  const { data: persisted } = useProfileSkills()
  const mutation = useUpdateProfileSkills()
  const [draft, setDraft] = useState<DraftSkill[]>([])
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const dialogRef = useRef<HTMLDivElement>(null)

  // Re-seed the draft whenever the modal opens so the user always
  // starts from the latest persisted value.
  useEffect(() => {
    if (!open) return
    setErrorMessage(null)
    setDraft(
      (persisted ?? []).map((entry) => ({
        skill_text: entry.skill_text,
        display_text: entry.display_text,
      })),
    )
  }, [open, persisted])

  // ESC closes the modal, matching the project-wide dialog contract.
  useEffect(() => {
    if (!open) return
    function handleKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose()
    }
    document.addEventListener("keydown", handleKey)
    return () => document.removeEventListener("keydown", handleKey)
  }, [open, onClose])

  const alreadySelected = useMemo(
    () => new Set(draft.map((d) => d.skill_text)),
    [draft],
  )
  const atMax = draft.length >= maxSkills
  const isDirty = useMemo(
    () => !sameOrderedList(draft, persisted ?? []),
    [draft, persisted],
  )

  const addSkill = useCallback(
    (skill: SkillResponse) => {
      setErrorMessage(null)
      setDraft((current) => {
        if (current.some((d) => d.skill_text === skill.skill_text)) {
          return current
        }
        if (current.length >= maxSkills) return current
        return [
          ...current,
          {
            skill_text: skill.skill_text,
            display_text: skill.display_text,
          },
        ]
      })
    },
    [maxSkills],
  )

  const removeSkill = useCallback((skillText: string) => {
    setErrorMessage(null)
    setDraft((current) => current.filter((d) => d.skill_text !== skillText))
  }, [])

  const moveSkill = useCallback((skillText: string, direction: -1 | 1) => {
    setErrorMessage(null)
    setDraft((current) => {
      const index = current.findIndex((d) => d.skill_text === skillText)
      const target = index + direction
      if (index === -1 || target < 0 || target >= current.length) return current
      const next = [...current]
      next[index] = current[target]
      next[target] = current[index]
      return next
    })
  }, [])

  const handleSave = useCallback(async () => {
    setErrorMessage(null)
    try {
      await mutation.mutateAsync(draft.map((d) => d.skill_text))
      onClose()
    } catch (caught) {
      setErrorMessage(mapErrorToMessage(caught, t, maxSkills))
    }
  }, [draft, mutation, onClose, t, maxSkills])

  if (!open) return null
  if (typeof window === "undefined") return null

  const body = (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/40 backdrop-blur-sm p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="skills-editor-title"
        onClick={(event) => event.stopPropagation()}
        className="flex h-[90vh] max-h-[720px] w-full max-w-2xl flex-col rounded-xl border border-border bg-background shadow-xl"
      >
        <ModalHeader onClose={onClose} />
        <div className="flex flex-col gap-3 border-b border-border px-4 py-3 sm:gap-4 sm:px-6 sm:py-4">
          <SkillSearchBar
            alreadySelected={alreadySelected}
            onAdd={addSkill}
            disabled={atMax}
          />
          <SelectedSkillsList
            draft={draft}
            onRemove={removeSkill}
            onMove={moveSkill}
          />
          <div className="flex items-center justify-between">
            <p
              aria-live="polite"
              className={cn(
                "text-xs",
                atMax ? "text-primary font-medium" : "text-muted-foreground",
              )}
            >
              {t("counter", { count: draft.length, max: maxSkills })}
            </p>
          </div>
        </div>
        <ModalBrowseBody
          expertiseKeys={expertiseKeys}
          alreadySelected={alreadySelected}
          onAdd={addSkill}
        />
        <ModalFooter
          isDirty={isDirty}
          isSaving={mutation.isPending}
          errorMessage={errorMessage}
          onCancel={onClose}
          onSave={handleSave}
        />
      </div>
    </div>
  )

  return createPortal(body, document.body)
}

// ----- Sub-components ---------------------------------------------------

function ModalHeader({ onClose }: { onClose: () => void }) {
  const t = useTranslations("profile.skills")
  return (
    <div className="flex items-center justify-between border-b border-border px-6 py-4">
      <h2
        id="skills-editor-title"
        className="text-lg font-semibold text-foreground"
      >
        {t("modalTitle")}
      </h2>
      <button
        type="button"
        onClick={onClose}
        aria-label={t("close")}
        className="rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
      >
        <X className="h-5 w-5" aria-hidden="true" />
      </button>
    </div>
  )
}

type SelectedSkillsListProps = {
  draft: DraftSkill[]
  onRemove: (skillText: string) => void
  onMove: (skillText: string, direction: -1 | 1) => void
}

function SelectedSkillsList({
  draft,
  onRemove,
  onMove,
}: SelectedSkillsListProps) {
  const t = useTranslations("profile.skills")
  if (draft.length === 0) {
    return (
      <p className="text-sm italic text-muted-foreground">
        {t("selectedEmpty")}
      </p>
    )
  }
  return (
    // Cap the selected list height so it never pushes the browse
    // panels off-screen. Responsive: tight on mobile (≈ 2 rows
    // visible so the top zone stays under ~30% of a 667px viewport)
    // and looser on sm+ desktops (≈ 4 rows).
    <ol
      className="flex max-h-[120px] flex-col gap-1.5 overflow-y-auto pr-1 sm:max-h-[220px] sm:gap-2"
      aria-label={t("selectedListLabel")}
    >
      {draft.map((entry, index) => (
        <li
          key={entry.skill_text}
          className="flex items-center justify-between gap-2 rounded-md border border-border bg-muted/30 px-2.5 py-1"
        >
          <span className="truncate text-sm font-medium text-foreground">
            <span className="mr-2 text-muted-foreground">{index + 1}.</span>
            {entry.display_text}
          </span>
          <span className="flex shrink-0 items-center gap-1">
            <IconButton
              label={t("moveUp", { label: entry.display_text })}
              disabled={index === 0}
              onClick={() => onMove(entry.skill_text, -1)}
              icon={<ArrowUp className="h-4 w-4" aria-hidden="true" />}
            />
            <IconButton
              label={t("moveDown", { label: entry.display_text })}
              disabled={index === draft.length - 1}
              onClick={() => onMove(entry.skill_text, 1)}
              icon={<ArrowDown className="h-4 w-4" aria-hidden="true" />}
            />
            <button
              type="button"
              onClick={() => onRemove(entry.skill_text)}
              aria-label={t("remove", { label: entry.display_text })}
              className="inline-flex h-8 w-8 items-center justify-center rounded-md text-destructive hover:bg-destructive/10 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
            >
              <X className="h-4 w-4" aria-hidden="true" />
            </button>
          </span>
        </li>
      ))}
    </ol>
  )
}

type ModalBrowseBodyProps = {
  expertiseKeys: string[]
  alreadySelected: Set<string>
  onAdd: (skill: SkillResponse) => void
}

function ModalBrowseBody({
  expertiseKeys,
  alreadySelected,
  onAdd,
}: ModalBrowseBodyProps) {
  const t = useTranslations("profile.skills")
  // The "Popular in your domains" row is keyed to the user's declared
  // expertise when they have any — it feels like curated guidance. When
  // they haven't declared any yet, we fall back to development +
  // design_ui_ux as a reasonable default so the row always has content.
  const popularKeys = expertiseKeys.length > 0
    ? expertiseKeys
    : (["development", "design_ui_ux"] as string[])

  // The "Browse by domain" section always lists the full 15 expertise
  // domains so users can pick skills from any area, not just the
  // domains they declared on their profile. The first panel is open
  // by default to reduce the initial scroll.
  const allKeys = ALL_EXPERTISE_DOMAIN_KEYS

  return (
    <div className="flex-1 overflow-y-auto px-6 py-4">
      <section className="mb-6">
        <h3 className="mb-3 text-sm font-semibold text-foreground">
          {t("popularHeading")}
        </h3>
        <PopularSkillsRow
          expertiseKeys={popularKeys}
          alreadySelected={alreadySelected}
          onAdd={onAdd}
        />
      </section>
      <section>
        <h3 className="mb-3 text-sm font-semibold text-foreground">
          {t("browseHeading")}
        </h3>
        <div className="flex flex-col gap-3">
          {allKeys.map((key, index) => (
            <ExpertisePanel
              key={key}
              expertiseKey={key}
              alreadySelected={alreadySelected}
              onAdd={onAdd}
              defaultOpen={index === 0}
            />
          ))}
        </div>
      </section>
    </div>
  )
}

type ModalFooterProps = {
  isDirty: boolean
  isSaving: boolean
  errorMessage: string | null
  onCancel: () => void
  onSave: () => void
}

function ModalFooter({
  isDirty,
  isSaving,
  errorMessage,
  onCancel,
  onSave,
}: ModalFooterProps) {
  const t = useTranslations("profile.skills")
  return (
    <div className="border-t border-border px-6 py-4">
      {errorMessage ? (
        <p
          role="alert"
          className="mb-3 rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive"
        >
          {errorMessage}
        </p>
      ) : null}
      <div className="flex items-center justify-end gap-2">
        <button
          type="button"
          onClick={onCancel}
          disabled={isSaving}
          className="rounded-md h-9 px-4 text-sm font-medium text-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
        >
          {t("cancel")}
        </button>
        <button
          type="button"
          onClick={onSave}
          disabled={!isDirty || isSaving}
          className="inline-flex items-center gap-2 rounded-md bg-primary h-9 px-4 text-sm font-medium text-primary-foreground hover:opacity-90 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
        >
          {isSaving ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : (
            <Check className="h-4 w-4" aria-hidden="true" />
          )}
          {isSaving ? t("saving") : t("save")}
        </button>
      </div>
    </div>
  )
}

type IconButtonProps = {
  label: string
  disabled?: boolean
  onClick: () => void
  icon: React.ReactNode
}

function IconButton({ label, disabled, onClick, icon }: IconButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      className="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-40 disabled:cursor-not-allowed"
    >
      {icon}
    </button>
  )
}

// ----- Helpers ----------------------------------------------------------

function sameOrderedList(
  a: Array<{ skill_text: string }>,
  b: Array<{ skill_text: string }>,
): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i += 1) {
    if (a[i].skill_text !== b[i].skill_text) return false
  }
  return true
}

function mapErrorToMessage(
  error: unknown,
  t: ReturnType<typeof useTranslations>,
  maxSkills: number,
): string {
  if (error instanceof ApiError) {
    if (error.code === "too_many_skills") {
      return t("errorTooMany", { max: maxSkills })
    }
    if (error.status === 403) return t("errorDisabled")
  }
  return t("errorGeneric")
}
