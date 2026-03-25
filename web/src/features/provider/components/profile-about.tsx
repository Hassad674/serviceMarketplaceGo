"use client"

import { useRef, useState, useCallback, useEffect } from "react"
import { Edit2, Loader2 } from "lucide-react"

const MAX_CHARS = 1000

interface ProfileAboutProps {
  content: string
  onSave: (text: string) => Promise<void>
  saving?: boolean
  label?: string
  placeholder?: string
}

export function ProfileAbout({
  content,
  onSave,
  saving = false,
  label = "About",
  placeholder = "Describe your activity...",
}: ProfileAboutProps) {
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

  function startEditing() {
    setDraft(content)
    setEditing(true)
  }

  function cancelEditing() {
    setDraft(content)
    setEditing(false)
  }

  async function handleSave() {
    await onSave(draft.trim())
    setEditing(false)
  }

  return (
    <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-foreground">{label}</h2>
        {!editing && (
          <button
            type="button"
            onClick={startEditing}
            aria-label={`Edit ${label.toLowerCase()}`}
            className="rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
          >
            <Edit2 className="w-[18px] h-[18px]" aria-hidden="true" />
          </button>
        )}
      </div>

      {editing ? (
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
              {draft.length} / {MAX_CHARS} characters
            </span>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={cancelEditing}
                disabled={saving}
                className="rounded-md h-9 px-4 text-sm font-medium text-foreground hover:bg-muted transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSave}
                disabled={saving}
                className="bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium hover:opacity-90 transition-opacity duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50 inline-flex items-center gap-2"
              >
                {saving && <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />}
                Save
              </button>
            </div>
          </div>
        </div>
      ) : content ? (
        <p className="text-sm text-foreground whitespace-pre-line">{content}</p>
      ) : (
        <p className="text-sm text-muted-foreground italic">
          Click the edit button to add a description
        </p>
      )}
    </section>
  )
}
