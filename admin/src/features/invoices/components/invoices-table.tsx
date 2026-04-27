import { FileText, FileMinus } from "lucide-react"
import { Badge } from "@/shared/components/ui/badge"
import { formatCurrency, formatDate } from "@/shared/lib/utils"
import { openInvoicePDF } from "../api/invoicing-api"
import type { AdminInvoiceRow } from "../types"

type InvoicesTableProps = {
  rows: AdminInvoiceRow[]
}

const TYPE_LABELS: Record<string, string> = {
  subscription: "Abonnement",
  monthly_commission: "Commission",
}

const STATUS_LABELS: Record<string, string> = {
  issued: "Emise",
  credited: "Avoir applique",
  draft: "Brouillon",
  credit_note: "Avoir",
}

// InvoicesTable renders the unified invoice + credit-note grid. Click
// a row to open its PDF in a new tab — the click handler hits the
// admin redirect endpoint and pops the resolved presigned URL via
// window.open.
export function InvoicesTable({ rows }: InvoicesTableProps) {
  async function handleOpenPDF(row: AdminInvoiceRow) {
    try {
      const url = await openInvoicePDF(row.id, row.is_credit_note)
      window.open(url, "_blank", "noopener,noreferrer")
    } catch (e) {
      // Best-effort: surface a console error so the operator can
      // inspect via devtools. A toast system is out of scope here.
      // eslint-disable-next-line no-console
      console.error("Failed to open invoice PDF", e)
    }
  }

  return (
    <div className="overflow-hidden rounded-xl border border-gray-100 bg-white shadow-sm">
      <table className="w-full" aria-label="Liste des factures emises">
        <thead>
          <tr className="border-b border-border bg-muted/50">
            <Th>Type</Th>
            <Th>Numero</Th>
            <Th>Destinataire</Th>
            <Th>Emise le</Th>
            <Th>Montant TTC</Th>
            <Th>Regime TVA</Th>
            <Th>Statut</Th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {rows.length === 0 ? (
            <tr>
              <td
                colSpan={7}
                className="px-4 py-12 text-center text-sm text-muted-foreground"
              >
                Aucune facture ne correspond aux filtres.
              </td>
            </tr>
          ) : (
            rows.map((row) => (
              <tr
                key={row.id}
                onClick={() => handleOpenPDF(row)}
                className="cursor-pointer transition-colors duration-150 hover:bg-muted/50"
                data-testid={`invoice-row-${row.id}`}
              >
                <td className="px-4 py-3 text-sm">
                  <TypeBadge row={row} />
                </td>
                <td className="px-4 py-3 text-sm font-mono text-foreground">
                  {row.number}
                </td>
                <td className="px-4 py-3 text-sm text-foreground">
                  <div className="font-medium">
                    {row.recipient_legal_name || "(sans nom)"}
                  </div>
                  <div className="text-xs text-muted-foreground font-mono">
                    {row.recipient_org_id}
                  </div>
                </td>
                <td className="px-4 py-3 text-sm text-muted-foreground">
                  {formatDate(row.issued_at)}
                </td>
                <td className="px-4 py-3 text-sm font-mono text-foreground">
                  {formatCurrency(row.amount_incl_tax_cents / 100)}
                </td>
                <td className="px-4 py-3 text-xs text-muted-foreground">
                  {row.tax_regime}
                </td>
                <td className="px-4 py-3 text-sm">
                  <StatusBadge status={row.status} />
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}

function Th({ children }: { children: React.ReactNode }) {
  return (
    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
      {children}
    </th>
  )
}

function TypeBadge({ row }: { row: AdminInvoiceRow }) {
  if (row.is_credit_note) {
    return (
      <span className="inline-flex items-center gap-1.5 rounded-md bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700">
        <FileMinus className="h-3.5 w-3.5" />
        Avoir
      </span>
    )
  }
  const label = TYPE_LABELS[row.source_type ?? ""] ?? "Facture"
  return (
    <span className="inline-flex items-center gap-1.5 rounded-md bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700">
      <FileText className="h-3.5 w-3.5" />
      {label}
    </span>
  )
}

function StatusBadge({ status }: { status: string }) {
  const label = STATUS_LABELS[status] ?? status
  switch (status) {
    case "issued":
      return <Badge variant="success">{label}</Badge>
    case "credited":
      return <Badge variant="warning">{label}</Badge>
    case "credit_note":
      return <Badge variant="default">{label}</Badge>
    default:
      return <Badge variant="outline">{label}</Badge>
  }
}
