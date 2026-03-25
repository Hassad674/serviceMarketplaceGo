import { FileText } from "lucide-react"

export function ProfileHistory() {
  return (
    <section className="bg-card border border-border rounded-xl p-6 shadow-sm">
      <div className="flex items-center gap-3 mb-4">
        <h2 className="text-lg font-semibold text-foreground">
          Project History
        </h2>
        <span className="rounded-full bg-muted text-muted-foreground px-2.5 py-0.5 text-xs font-medium">
          0 completed
        </span>
      </div>

      <div className="flex flex-col items-center justify-center py-8 text-center">
        <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-3">
          <FileText className="w-5 h-5 text-muted-foreground" aria-hidden="true" />
        </div>
        <p className="text-sm font-medium text-foreground mb-1">
          No completed projects
        </p>
        <p className="text-sm text-muted-foreground italic">
          Completed projects will appear here once finished.
        </p>
      </div>
    </section>
  )
}
