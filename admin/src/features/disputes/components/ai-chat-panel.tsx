import { useState } from "react"
import { Bot, Send, AlertTriangle, Loader2 } from "lucide-react"

import { Button } from "@/shared/components/ui/button"
import { Card, CardHeader, CardTitle, CardContent } from "@/shared/components/ui/card"
import { Textarea } from "@/shared/components/ui/textarea"
import { cn, formatRelativeDate } from "@/shared/lib/utils"

import type { AIChatMessage } from "../types"
import { useAskAIDispute } from "../hooks/use-disputes"

interface AIChatPanelProps {
  disputeId: string
  history: AIChatMessage[]
  budgetExceeded: boolean
}

export function AIChatPanel({ disputeId, history, budgetExceeded }: AIChatPanelProps) {
  // The chat history is server-persisted and lives on the dispute object.
  // The frontend just reflects it. The local input state is the only
  // ephemeral piece — the question being typed before submit.
  const [question, setQuestion] = useState("")
  const askMutation = useAskAIDispute(disputeId)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = question.trim()
    if (!trimmed || askMutation.isPending) return

    askMutation.mutate(trimmed, {
      onSuccess: () => {
        // The dispute query is invalidated by the hook → re-fetch happens
        // automatically → both turns appear from the persisted source.
        setQuestion("")
      },
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Bot className="h-5 w-5 text-violet-500" />
          Demander a l&apos;assistant IA
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {history.length > 0 && (
          <div className="max-h-96 space-y-3 overflow-y-auto rounded-lg border bg-muted/30 p-3">
            {history.map((msg) => (
              <ChatBubble key={msg.id} message={msg} />
            ))}
            {askMutation.isPending && (
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Loader2 className="h-3 w-3 animate-spin" />
                L&apos;assistant reflechit...
              </div>
            )}
          </div>
        )}

        {askMutation.isError && (
          <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-3 text-sm text-destructive">
            {extractErrorMessage(askMutation.error)}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-2">
          <Textarea
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            rows={2}
            placeholder="Posez votre question..."
            disabled={budgetExceeded || askMutation.isPending}
          />
          <div className="flex items-center justify-between">
            <p className="flex items-center gap-1 text-xs text-muted-foreground">
              <AlertTriangle className="h-3 w-3" />
              L&apos;IA peut se tromper. Verifiez toujours les sources avant
              de trancher.
            </p>
            <Button
              type="submit"
              variant="primary"
              size="sm"
              disabled={
                budgetExceeded || askMutation.isPending || !question.trim()
              }
            >
              {askMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : (
                <Send className="h-4 w-4 mr-2" />
              )}
              Envoyer
            </Button>
          </div>
        </form>

        {budgetExceeded && (
          <p className="text-xs text-warning">
            Budget IA epuise pour ce litige. Cliquez sur &quot;Augmenter le
            budget&quot; dans le panel ci-contre pour debloquer.
          </p>
        )}
      </CardContent>
    </Card>
  )
}

function ChatBubble({ message }: { message: AIChatMessage }) {
  const isUser = message.role === "user"
  const tokenCost = message.input_tokens + message.output_tokens
  return (
    <div className={cn("flex", isUser ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "max-w-[85%] rounded-lg px-3 py-2 text-sm",
          isUser
            ? "bg-primary text-primary-foreground"
            : "bg-card text-card-foreground border",
        )}
      >
        <p className="whitespace-pre-wrap">{message.content}</p>
        <div
          className={cn(
            "mt-1 flex items-center justify-between gap-2 text-[10px]",
            isUser ? "text-primary-foreground/70" : "text-muted-foreground",
          )}
        >
          <span>{formatRelativeDate(message.created_at)}</span>
          {!isUser && tokenCost > 0 && (
            <span className="font-mono">
              {tokenCost.toLocaleString("fr-FR")} tokens
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

function extractErrorMessage(err: unknown): string {
  if (err && typeof err === "object" && "message" in err) {
    const m = (err as { message: unknown }).message
    if (typeof m === "string") return m
  }
  return "Erreur inattendue lors de l'appel a l'IA."
}
