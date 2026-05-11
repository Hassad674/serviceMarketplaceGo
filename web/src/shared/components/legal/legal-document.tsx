import type { ReactNode } from "react"

// LegalDocument — D4 (GDPR Phase C). Server-renderable presentation
// surface for long-form legal markdown documents shipped under
// /legal/* routes (registre, AIPD, DPA template, politique de
// confidentialité, CGU, CGV).
//
// We deliberately do NOT pull in `react-markdown` or any other parser:
// the markdown files in /legal/*.md are the authoritative source for
// DPO / legal counsel export. The web pages mirror that content in
// strict semantic HTML with i18n-keyed headings so the wording stays
// readable, accessible, and free of any third-party origin in the CSP.
//
// Each page receives a structured `LegalSections` array (typed below).
// LegalDocument renders the title, a meta strip (last updated, source
// link to the markdown), then walks the sections rendering headings,
// paragraphs, lists, and simple tables. Anything more elaborate (raw
// HTML, scripts) is intentionally not supported — keep the surface
// boring.

export type LegalParagraph = {
  type: "p"
  content: ReactNode
}

export type LegalList = {
  type: "ul" | "ol"
  items: ReactNode[]
}

export type LegalTable = {
  type: "table"
  caption?: string
  headers: string[]
  rows: ReactNode[][]
}

export type LegalCallout = {
  type: "callout"
  variant?: "info" | "warning"
  content: ReactNode
}

export type LegalBlock = LegalParagraph | LegalList | LegalTable | LegalCallout

export interface LegalSection {
  id: string
  heading: string
  blocks: LegalBlock[]
}

export interface LegalDocumentProps {
  title: string
  subtitle?: string
  lastUpdatedISO: string
  sourceHref?: string
  englishNotice?: string
  sections: LegalSection[]
}

function renderBlock(block: LegalBlock, index: number): ReactNode {
  if (block.type === "p") {
    return (
      <p key={index} className="text-sm leading-relaxed text-foreground">
        {block.content}
      </p>
    )
  }

  if (block.type === "ul" || block.type === "ol") {
    const ListTag = block.type
    return (
      <ListTag
        key={index}
        className={
          block.type === "ul"
            ? "ml-6 list-disc space-y-1 text-sm text-foreground"
            : "ml-6 list-decimal space-y-1 text-sm text-foreground"
        }
      >
        {block.items.map((item, i) => (
          <li key={i}>{item}</li>
        ))}
      </ListTag>
    )
  }

  if (block.type === "table") {
    return (
      <div
        key={index}
        className="overflow-x-auto rounded-2xl border border-border bg-card"
      >
        <table className="w-full text-left text-sm">
          {block.caption ? (
            <caption className="px-4 pt-3 text-xs text-muted-foreground">
              {block.caption}
            </caption>
          ) : null}
          <thead className="border-b border-border bg-muted/30 text-xs uppercase text-muted-foreground">
            <tr>
              {block.headers.map((header, i) => (
                <th key={i} scope="col" className="px-4 py-3">
                  {header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-border text-foreground">
            {block.rows.map((row, rIdx) => (
              <tr key={rIdx}>
                {row.map((cell, cIdx) => (
                  <td key={cIdx} className="px-4 py-3 text-sm">
                    {cell}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  }

  if (block.type === "callout") {
    return (
      <aside
        key={index}
        className={
          block.variant === "warning"
            ? "rounded-2xl border border-accent/40 bg-accent/10 p-4 text-sm text-foreground"
            : "rounded-2xl border border-border bg-muted/30 p-4 text-sm text-muted-foreground"
        }
      >
        {block.content}
      </aside>
    )
  }

  // Exhaustive — every LegalBlock variant is covered above.
  return null
}

export function LegalDocument({
  title,
  subtitle,
  lastUpdatedISO,
  sourceHref,
  englishNotice,
  sections,
}: LegalDocumentProps) {
  const formattedDate = new Date(lastUpdatedISO).toISOString().slice(0, 10)

  return (
    <article className="mx-auto w-full max-w-5xl space-y-8 py-12">
      <header className="space-y-3 border-b border-border pb-6">
        <h1 className="font-display text-3xl text-foreground sm:text-4xl">
          {title}
        </h1>
        {subtitle ? (
          <p className="text-muted-foreground">{subtitle}</p>
        ) : null}
        <p className="text-xs text-muted-foreground">
          <span>Version du {formattedDate}</span>
          {sourceHref ? (
            <>
              {" — "}
              <a
                href={sourceHref}
                className="text-accent underline-offset-4 hover:underline"
                rel="noopener"
              >
                source Markdown
              </a>
            </>
          ) : null}
        </p>
        {englishNotice ? (
          <p className="rounded-2xl border border-border bg-muted/30 p-3 text-xs text-muted-foreground">
            {englishNotice}
          </p>
        ) : null}
      </header>

      <div className="space-y-10">
        {sections.map((section) => (
          <section
            key={section.id}
            id={section.id}
            aria-labelledby={`${section.id}-heading`}
            className="space-y-4"
          >
            <h2
              id={`${section.id}-heading`}
              className="font-display text-xl text-foreground sm:text-2xl"
            >
              {section.heading}
            </h2>
            <div className="space-y-3">
              {section.blocks.map((block, i) => renderBlock(block, i))}
            </div>
          </section>
        ))}
      </div>
    </article>
  )
}
