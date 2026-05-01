"use client"

import { useCallback, useEffect, useRef, useState } from "react"
import { Edit2, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"

const MAX_CHARS = 1000

interface ProfileAboutCardProps {
  content: string
  label: string
  placeholder: string
  onSave?: (text: string) => Promise<void>
  saving?: boolean
  readOnly?: boolean
}

// ProfileAboutCard renders the editable "À propos" section used by
// both the freelance-profile and referrer-profile features. Owns its
// own draft/editing state so each persona page can compose it
// without re-implementing textarea plumbing. Stays under the 4-prop
// cap by accepting the optional handlers only as opt-in fields.
export function ProfileAboutCard(props: ProfileAboutCardProps) {
  const {
    content,
    label,
    placeholder,
    onSave,
    saving = false,
    readOnly = false,
  } = props
  const t = useTranslations("profile")
  const tCommon = useTranslations("common")
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(content)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const autoResize = useCallback(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = "auto"
    el.style.height = `${Math.min(el.scrollHeight, 200)}px`
  }, [])

  useEffect(() => {
    if (editing) autoResize()
  }, [editing, draft, autoResize])

  if (readOnly && !content) return null

  const startEditing = () => {
    setDraft(content)
    setEditing(true)
  }
  const cancelEditing = () => {
    setDraft(content)
    setEditing(false)
  }
  const handleSave = async () => {
    if (!onSave) return
    await onSave(draft.trim())
    setEditing(false)
  }

  return (
    <section className="bg-card border border-border rounded-xl p-6 shadow-sm min-w-0 w-full">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-foreground">{label}</h2>
        {!editing && !readOnly && onSave ? (
          <button
            type="button"
            onClick={startEditing}
            aria-label={`${tCommon("edit")} ${label.toLowerCase()}`}
            className="rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
          >
            <Edit2 className="w-[18px] h-[18px]" aria-hidden="true" />
          </button>
        ) : null}
      </div>

      {editing ? (
        <AboutEditor
          draft={draft}
          setDraft={setDraft}
          placeholder={placeholder}
          label={label}
          saving={saving}
          onCancel={cancelEditing}
          onSave={handleSave}
          textareaRef={textareaRef}
        />
      ) : content ? (
        <p className="text-sm text-foreground whitespace-pre-line break-words [overflow-wrap:anywhere] min-w-0">
          {content}
        </p>
      ) : (
        <p className="text-sm text-muted-foreground italic">
          {t("clickToEdit")}
        </p>
      )}
    </section>
  )
}

interface AboutEditorProps {
  draft: string
  setDraft: (next: string) => void
  placeholder: string
  label: string
  saving: boolean
  onCancel: () => void
  onSave: () => void
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
}

function AboutEditor(props: AboutEditorProps) {
  const {
    draft,
    setDraft,
    placeholder,
    label,
    saving,
    onCancel,
    onSave,
    textareaRef,
  } = props
  const t = useTranslations("profile")
  const tCommon = useTranslations("common")
  return (
    <div className="space-y-3">
      <textarea
        ref={textareaRef}
        value={draft}
        onChange={(e) => setDraft(e.target.value.slice(0, MAX_CHARS))}
        placeholder={placeholder}
        rows={4}
        className="w-full min-h-[100px] max-h-[200px] border border-border rounded-md p-3 text-sm text-foreground bg-background placeholder:text-muted-foreground resize-none focus:outline-none focus:ring-2 focus:ring-primary"
        aria-label={label}
      />
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">
          {draft.length} / {MAX_CHARS} {t("characters")}
        </span>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={onCancel}
            disabled={saving}
            className="rounded-md h-9 px-4 text-sm font-medium text-foreground hover:bg-muted transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
          >
            {tCommon("cancel")}
          </button>
          <button
            type="button"
            onClick={onSave}
            disabled={saving}
            className="bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium hover:opacity-90 transition-opacity duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50 inline-flex items-center gap-2"
          >
            {saving ? (
              <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
            ) : null}
            {tCommon("save")}
          </button>
        </div>
      </div>
    </div>
  )
}
