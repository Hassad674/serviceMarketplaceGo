import { Wallet, Plus, Loader2 } from "lucide-react"

import { Button } from "@/shared/components/ui/button"
import { Card, CardHeader, CardTitle, CardContent } from "@/shared/components/ui/card"
import { cn } from "@/shared/lib/utils"

import type { AIBudgetSummary } from "../types"
import { useIncreaseAIBudget } from "../hooks/use-disputes"

const TIER_LABELS: Record<string, string> = {
  S: "Standard",
  M: "Significatif",
  L: "Important",
  XL: "Critique",
}

interface AIBudgetPanelProps {
  disputeId: string
  budget: AIBudgetSummary
}

export function AIBudgetPanel({ disputeId, budget }: AIBudgetPanelProps) {
  const increaseMutation = useIncreaseAIBudget(disputeId)

  const summaryRatio = ratio(budget.summary_used_tokens, budget.summary_max_tokens)
  const chatRatio = ratio(budget.chat_used_tokens, budget.chat_max_tokens)
  const summaryHardCapReached = summaryRatio >= 1.0
  const chatHardCapReached = chatRatio >= 1.0
  const showIncreaseButton = summaryHardCapReached || chatHardCapReached

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Wallet className="h-5 w-5 text-violet-500" />
          Budget IA
          <span className="ml-2 rounded-md bg-violet-50 px-2 py-0.5 text-xs font-medium text-violet-700">
            Tier {budget.tier} — {TIER_LABELS[budget.tier] ?? budget.tier}
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <BudgetBar
          label="Resume"
          used={budget.summary_used_tokens}
          max={budget.summary_max_tokens}
        />
        <BudgetBar
          label="Chat"
          used={budget.chat_used_tokens}
          max={budget.chat_max_tokens}
        />

        <div className="flex items-center justify-between border-t pt-3 text-sm">
          <span className="text-muted-foreground">Cumul total</span>
          <span className="font-medium text-foreground">
            {budget.total_used_tokens.toLocaleString("fr-FR")} tokens
          </span>
        </div>
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Cout cumule</span>
          <span className="font-mono font-semibold text-foreground">
            {budget.total_cost_eur.toFixed(4)} EUR
          </span>
        </div>

        {budget.bonus_tokens > 0 && (
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>Bonus accorde</span>
            <span>+{budget.bonus_tokens.toLocaleString("fr-FR")} tokens</span>
          </div>
        )}

        {showIncreaseButton && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={increaseMutation.isPending}
            onClick={() => increaseMutation.mutate()}
            className="w-full"
          >
            {increaseMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin mr-2" />
            ) : (
              <Plus className="h-4 w-4 mr-2" />
            )}
            Augmenter le budget (+25 000 tokens)
          </Button>
        )}
      </CardContent>
    </Card>
  )
}

interface BudgetBarProps {
  label: string
  used: number
  max: number
}

function BudgetBar({ label, used, max }: BudgetBarProps) {
  const r = ratio(used, max)
  const pct = Math.min(r * 100, 100)
  // Tone the bar based on usage: green < 70%, orange 70-100%, red > 100%.
  const barColor =
    r >= 1.0 ? "bg-destructive" : r >= 0.7 ? "bg-warning" : "bg-success"
  const labelColor =
    r >= 1.0
      ? "text-destructive"
      : r >= 0.7
        ? "text-warning"
        : "text-muted-foreground"

  return (
    <div>
      <div className="mb-1 flex items-center justify-between text-xs">
        <span className="font-medium text-foreground">{label}</span>
        <span className={cn("font-mono", labelColor)}>
          {used.toLocaleString("fr-FR")} / {max.toLocaleString("fr-FR")}
        </span>
      </div>
      <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
        <div
          className={cn("h-full transition-all duration-300", barColor)}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

function ratio(used: number, max: number): number {
  if (max <= 0) return 0
  return used / max
}
