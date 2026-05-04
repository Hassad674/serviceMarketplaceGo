# Design — Source files & Claude Design commands

> Where the Soleil v2 design comes from + how to fetch more details if needed.
> Read this if you need to inspect the original design beyond what's in `assets/sources/`.

---

## 1. Local working copies (versioned in this PR)

The full Claude Design output has been copied into `design/assets/` and committed to the repo. **This is the authoritative copy** — don't go fetching from external sources unless you specifically need a detail not present here.

```
design/assets/
├── sources/
│   ├── design-canvas.jsx          (the canvas wrapper component, used by all Atelier-*.html)
│   ├── screens-editorial.jsx      (legacy Direction A — superseded, kept for ref)
│   ├── screens-studio.jsx         (legacy Direction B — superseded, kept for ref)
│   ├── system-cards.jsx           (overview of A vs B vs Soleil — illustrates why Soleil won)
│   ├── phase1/                    (the Soleil v2 implementations — THIS is the source of truth)
│   │   ├── soleil.jsx             (the master file: tokens, shared primitives, ref dashboard/find/profile/messages)
│   │   ├── soleil-app.jsx         (mobile master)
│   │   ├── soleil-lotA.jsx        (web desktop · entreprise · jobs/projects)
│   │   ├── soleil-lotA-mobile.jsx (responsive web · idem)
│   │   ├── soleil-lotB.jsx        (web desktop · argent · wallet/invoices/billing)
│   │   ├── soleil-lotB-mobile.jsx
│   │   ├── soleil-lotC.jsx        (web desktop · freelance · dashboard/opportunities/applications)
│   │   ├── soleil-lotC-mobile.jsx
│   │   ├── soleil-lotD.jsx        (web desktop · profil prestataire pub/priv)
│   │   ├── soleil-lotD-mobile.jsx
│   │   ├── soleil-lotE.jsx        (web desktop · auth + stripe + account)
│   │   ├── soleil-lotE-mobile.jsx
│   │   ├── soleil-lotF.jsx        (web desktop · messagerie + team)
│   │   ├── soleil-lotF-mobile.jsx
│   │   ├── soleil-app-lot1.jsx    (mobile · activité = dashboards + missions + candidatures)
│   │   ├── soleil-app-lot2.jsx    (mobile · annonces entreprise)
│   │   ├── soleil-app-lot3.jsx    (mobile · argent)
│   │   ├── soleil-app-lot4.jsx    (mobile · communication = messages + notifications)
│   │   ├── soleil-app-lot5.jsx    (mobile · auth + compte)
│   │   ├── soleil-jobcreate-mobile.jsx  (extracted because too big for lotA-mobile)
│   │   ├── soleil-mobile.jsx      (mobile master responsive)
│   │   ├── icons.jsx              (icon set used across all Soleil components)
│   │   ├── ios-frame.jsx          (iOS frame wrapper for mobile previews)
│   │   ├── _bootstrap.js          (canvas runtime helpers — not part of the visual)
│   │   ├── intro.jsx, place.jsx, maison-*.jsx (legacy/exploration — ignore)
│   ├── Atelier - 1. Web Desktop.html       (canvas viewer for desktop)
│   ├── Atelier - 2. Responsive Web.html    (canvas viewer for responsive)
│   ├── Atelier - 3. App Native.html        (canvas viewer for app)
│   └── Atelier App* / Atelier Lot*.html    (per-lot canvas viewers, useful to isolate one lot)
└── pdf/
    ├── web-desktop.pdf      (visual proof — 31 pages, every desktop screen with section dividers)
    ├── web-responsive.pdf   (30 pages — every responsive web screen)
    └── app-native-ios.pdf   (19 pages — every mobile screen)
```

### How to use these files

**Tokens** — already extracted into [`DESIGN_SYSTEM.md`](./DESIGN_SYSTEM.md). Don't re-extract from `soleil.jsx` — the canonical values are in the tokens file.

**Per-screen layout** — open the matching `soleil-lot{X}.jsx` and find the function for your screen (e.g. `SoleilJobsList`, `SoleilFreelancerDashboard`). The JSX has inline styles — those are inspiration, not literal — re-implement using Tailwind classes (web/admin) or Material 3 + theme extension (mobile).

**Visual proof** — open the PDFs at the right page (PDF page numbers listed in `inventory.md` per screen). When the JSX is ambiguous, the PDF wins.

**Section/lot mapping** — see `inventory.md`. Each entry tells you which `.jsx` file + line range to read.

---

## 2. Local "Téléchargements" (Hassad's machine, NOT versioned)

The original drop from Claude Design landed at:

```
/home/hassad/Téléchargements/serviceGoLast/   — the working folder (jsx + html + uploads + phase1)
/home/hassad/Téléchargements/desktop.pdf       — copied to design/assets/pdf/web-desktop.pdf
/home/hassad/Téléchargements/Responsive.pdf    — copied to design/assets/pdf/web-responsive.pdf
/home/hassad/Téléchargements/mobile.pdf        — copied to design/assets/pdf/app-native-ios.pdf
```

These are Hassad's local copies — **not committed**. The committed copies in `design/assets/` are the canonical reference for the chantier. If for some reason `design/assets/` gets corrupted, Hassad can re-copy from `Téléchargements`.

---

## 3. Claude Design canvas URLs (remote source-of-truth)

Each canvas is hosted by Anthropic. **The committed `design/assets/` already contains everything** — these URLs are a fallback for fetching the latest version if the design ever gets updated.

If you need to re-fetch, send these prompts to Claude in the design context:

### Web Desktop

```
Fetch this design file, read its readme, and implement the relevant
aspects of the design.
https://api.anthropic.com/v1/design/h/eFM-np0Vos4mj9az64NhZw?open_file=Atelier+-+1.+Web+Desktop.html

Implement: Atelier - 1. Web Desktop.html
```

### Responsive Web

```
Fetch this design file, read its readme, and implement the relevant
aspects of the design.
https://api.anthropic.com/v1/design/h/cTkWzJEcSF6y5x3qPMg53w?open_file=Atelier+-+2.+Responsive+Web.html

Implement: Atelier - 2. Responsive Web.html
```

### App Native

```
Fetch this design file, read its readme, and implement the relevant
aspects of the design.
https://api.anthropic.com/v1/design/h/l1591IIUBzBlVCKSQ5xgzA?open_file=Atelier+-+3.+App+Native.html

Implement: Atelier - 3. App Native.html
```

### Important caveats

- **DO NOT** auto-fetch these URLs at every batch — the committed copies are the source of truth.
- **DO** fetch them if Hassad pushes a design update and asks you to refresh.
- **NEVER** copy the JSX literally into `web/src/...` or `mobile/lib/...` — the JSX uses inline `style={{}}` which violates our linting rules. Re-implement using the repo's idiomatic patterns (Tailwind utility classes for web, Material 3 + theme extension for mobile).

---

## 4. The "uploads/" folder (Hassad's screenshots reference)

`design/assets/sources/uploads/` (NOT committed — they're in Hassad's local Téléchargements) contains 9 screenshots Hassad sent to Claude Design as visual references during the iterative design process. They're pre-finalization snapshots — DON'T use them as source-of-truth. The final visual is in the JSX + PDFs.

---

## 5. When the design needs an update

If Hassad wants to add a new screen, modify a primitive, or refresh the palette:

1. He runs the matching prompt above against Claude Design.
2. He downloads the updated archive into `~/Téléchargements/`.
3. We re-copy the relevant files into `design/assets/` and commit.
4. We update `inventory.md` if new screens were added.
5. We update `tracking.md` to ⚪ those new entries.

Don't try to update the design from inside this repo — Claude Design is the authoring tool, this repo is the implementation.
