# Design System — Atelier · Direction Soleil v2

The full token reference for the Atelier marketplace. Source-of-truth values come from [`assets/sources/phase1/soleil.jsx`](./assets/sources/phase1/soleil.jsx) (the original Claude Design implementation). This file is the canonical translation into web (Tailwind 4) + admin (same) + mobile (Flutter Material 3) tokens.

> **Rule**: tokens defined here are the only allowed values. No magic hex in components, no inline `TextStyle(fontSize: ...)` with arbitrary numbers, no off-system spacings.

---

## 1. Color palette

### Semantic tokens (web + admin via Tailwind `@theme`)

| CSS variable | Hex | Soleil name | Usage |
|--------------|-----|-------------|-------|
| `--color-background` | `#fffbf5` | ivoire | Page background |
| `--color-surface` | `#ffffff` | blanc pur | Card backgrounds |
| `--color-border` | `#f0e6d8` | sable clair | Faible borders, dividers |
| `--color-border-strong` | `#e0d3bc` | sable foncé | Forte borders, outlined buttons |
| `--color-foreground` | `#2a1f15` | encre | Primary text, headings |
| `--color-muted-foreground` | `#7a6850` | tabac | Secondary text, captions |
| `--color-subtle-foreground` | `#a89679` | sable | Mono labels, very subtle text |
| `--color-primary` | `#e85d4a` | corail | CTAs, accent words, active state |
| `--color-primary-soft` | `#fde9e3` | rose pâle | Soft backgrounds, active pill bg |
| `--color-primary-deep` | `#c43a26` | corail foncé | Hover, error |
| `--color-pink` | `#f08aa8` | rose chaud | Gradient stop |
| `--color-pink-soft` | `#fde6ed` | rose pâle | Gradient stop |
| `--color-success` | `#5a9670` | sapin | Success, availability dot |
| `--color-success-soft` | `#e8f2eb` | sapin pâle | Pills "Disponible" bg |
| `--color-amber` | `#d4924a` | ambre | Warnings (rare) |

### Mobile Material 3 mapping

| `colorScheme.*` | Soleil hex |
|-----------------|------------|
| `surface` | `#fffbf5` |
| `surfaceContainerLowest` | `#ffffff` |
| `outline` | `#f0e6d8` |
| `outlineVariant` | `#e0d3bc` |
| `onSurface` | `#2a1f15` |
| `onSurfaceVariant` | `#7a6850` |
| `primary` | `#e85d4a` |
| `primaryContainer` | `#fde9e3` |
| `error` | `#c43a26` |

Plus a `SoleilColors` ThemeExtension exposing the non-Material tokens (subtle-foreground, success/successSoft, pink/pinkSoft, amber).

### Gradients (used decoratively)

```css
--gradient-warm: linear-gradient(135deg, #fde9e3, #fde6ed, #fbf0dc);
--gradient-coral: linear-gradient(135deg, #fde9e3, #fde6ed);
```

Used on: cover/hero zones (login right column, profile cover band, dashboard "Cette semaine" right column when present), with optional radial blobs of `rgba(232, 93, 74, 0.28)` and `rgba(240, 138, 168, 0.35)`.

---

## 2. Typography

### Font families

| Token | Family | Fallback | Usage |
|-------|--------|----------|-------|
| `--font-serif` | Fraunces | Georgia, serif | Display, page titles, editorial accents, italic-quoted citations |
| `--font-sans` | Inter Tight | system-ui, sans-serif | UI, body, labels |
| `--font-mono` | Geist Mono | monospace | Numbers, IDs, dates metadata, mono labels |

Fonts loaded via Google Fonts (Fraunces opsz 9-144, Inter Tight 400-700, Geist Mono 400-500). Web: `next/font` with `display: 'swap'`. Mobile: `google_fonts` package (Flutter).

### Scale (sample)

| Use | Size (px) | Family | Weight | Notes |
|-----|-----------|--------|--------|-------|
| Display L (page hero) | 38-44 | serif | 400-500 | letter-spacing -0.025em |
| Display M (card hero) | 30 | serif | 400-500 | letter-spacing -0.02em |
| H1 / page title | 28-32 | serif | 500 | |
| H2 / section title | 22-24 | serif | 500 | |
| H3 / sub-section | 18 | serif | 500-600 | |
| Body L | 15 | sans | 400 | line-height 1.6 |
| Body | 14 | sans | 400-500 | |
| Body S | 13 | sans | 500 | |
| Caption | 12 | sans | 400-500 | |
| Micro | 11 | sans / mono | 600-700 | uppercase + 0.05-0.12em letter-spacing for mono labels |
| Stat number | 30-44 | serif | 500 | letter-spacing -0.02em |
| Mono amount | 16-18 | mono | 400-500 | |

**Editorial signature**: large display in `font-serif` with one or two words wrapped in `font-style: italic; color: var(--color-primary)`. Example: « Bonjour Nova, *belle journée* en perspective. »

---

## 3. Radii

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | 6px | Small inline badges |
| `--radius-md` | 10px | Inputs, dense cards |
| `--radius-lg` | 14px | Sidebar pill background, dense cards |
| `--radius-xl` | 16-18px | Cards, freelance cards, content sections |
| `--radius-2xl` | 20px | Large hero cards, profile header |
| `--radius-full` | 9999px | Buttons, pills, badges, avatars (when round) |

Mobile: matches via `BorderRadius.circular(N)` constants in `SoleilRadii`.

---

## 4. Spacing scale

Used (in pixels): `4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 28, 32, 36, 40, 44, 48, 56, 64, 80`.

Tailwind: maps onto the default scale (so `p-4` = 16px, `p-6` = 24px, etc.). The non-Tailwind values (10, 14, 18, 22, 28, 36, 44) appear in `soleil.jsx` and we keep them where the design demands tightness — use arbitrary values (`p-[18px]`) sparingly, with a comment.

---

## 5. Shadows

| Token | CSS | Usage |
|-------|-----|-------|
| `--shadow-card` | `0 4px 24px rgba(42, 31, 21, 0.04)` | Profile header card, hero cards |
| `--shadow-card-strong` | `0 8px 24px rgba(0, 0, 0, 0.12)` | Floating portraits in hero |
| `--shadow-portrait` | `0 2px 12px rgba(42, 31, 21, 0.06)` | Profile photo frame |
| `--shadow-message` | `0 2px 12px rgba(232, 93, 74, 0.08)` | Proposal-card-in-chat highlight |

No `shadow-glow` (legacy from previous direction). No heavy elevations — Soleil is calm.

---

## 6. Components — atomic primitives

These primitives are shared across all surfaces. **They are stable APIs** — once Phase 0 freezes them, screen-level agents only consume, never edit.

### `Portrait`

Stylized SVG portrait, 6 deterministic palettes (corail / vert olive / rose / ambre / lilas / bleu). Used **everywhere a person is referenced** — never initials, never emojis.

```tsx
<Portrait id={2} size={48} rounded="50%" />
```

Web: `web/src/shared/components/ui/portrait.tsx`.
Mobile: `mobile/lib/core/widgets/portrait.dart`.

Reference implementation: `assets/sources/phase1/soleil.jsx` lines 27-52.

### `Button`

Variants: `primary` (encre bg, white text), `accent` (corail bg, white text), `outline` (transparent bg, encre border), `ghost` (transparent), `danger` (corail-deep bg).
Sizes: `sm` (h-9, text-xs), `md` (h-10, text-sm), `lg` (h-12, text-base).
Radii: always `rounded-full` (Soleil signature).

### `Card`

`bg-white border border-border rounded-xl shadow-card` baseline. Padding 22-32px. Hover variants raise shadow + slight border darkening.

### `Pill` / `Badge`

`rounded-full px-3 py-1 text-xs font-medium`. Variants:
- `available` — `bg-success-soft text-success` with leading dot
- `accent` — `bg-primary-soft text-primary-deep` with leading-trail filter mark
- `neutral` — `bg-muted text-muted-foreground` for tags

### `Sidebar` (web/admin only)

256px wide, white bg, role-aware items (entreprise vs freelancer/provider), section divider, "Découvrir" sub-section, bottom Premium CTA card. See `assets/sources/phase1/soleil.jsx` lines 67-137.

### `Topbar` (web/admin only)

64px tall, white bg, search pill (left, max 480px), publish CTA (right), bell, portrait. See `soleil.jsx` lines 139-154.

### Mobile-specific: tab bar, status bar, iOS frame

Standard Material 3 + Cupertino-flavored elements. See `assets/sources/phase1/soleil-app.jsx` and `ios-frame.jsx`.

---

## 7. Motion

Every interactive element transitions in 150-200ms ease-out. No long animations. No bouncing. The marketplace feels calm and confident, not flashy.

| Pattern | Duration | Easing |
|---------|----------|--------|
| Color/bg/border transitions | 150ms | ease-out |
| Card hover (translate -2px + shadow up) | 200ms | ease-out |
| Modal/dropdown enter | 200ms | ease-out (scale 0.96 → 1, fade) |
| Skeleton shimmer | 1.5s infinite | ease-in-out |

**Forbidden**: any `animation-glow`, `pulse`, or "wiggle" effect from previous direction.

---

## 8. Iconography

Web: `lucide-react` icons, stroke-width 1.6-2, sizes 14-20px (inline 14, button 18, standalone 20).
Mobile: `material_symbols_icons` (rounded variant) for parity.

Mapping cheat-sheet for the icons used in `soleil.jsx`:

| Soleil name | Lucide | Material |
|-------------|--------|----------|
| Home | Home | home_rounded |
| Chat | MessageCircle | chat_bubble_rounded |
| Folder | Folder | folder_rounded |
| Briefcase | Briefcase | work_rounded |
| Users | Users | groups_rounded |
| Search | Search | search_rounded |
| Bell | Bell | notifications_rounded |
| Inbox | Inbox | inbox_rounded |
| Sparkle | Sparkles | auto_awesome_rounded |
| Layers | Layers | layers_rounded |
| Plus | Plus | add_rounded |
| Verified | BadgeCheck | verified_rounded |
| MapPin | MapPin | place_rounded |
| Star | Star | star_rounded |
| Globe | Globe | language_rounded |
| Clock | Clock | schedule_rounded |
| Send | Send | send_rounded |
| Phone | Phone | call_rounded |
| Video | Video | videocam_rounded |
| Sliders | SlidersHorizontal | tune_rounded |
| ChevronDown | ChevronDown | keyboard_arrow_down_rounded |
| Bookmark | Bookmark | bookmark_rounded |
| CheckCircle | CheckCircle2 | check_circle_rounded |
| Smiley | Smile | mood_rounded |
| Paperclip | Paperclip | attach_file_rounded |
| Mic | Mic | mic_rounded |

---

## 9. French language conventions

The product speaks French and tutoie. Lock these phrasings:

- Greetings: "Bonjour", "Salut".
- Time: "à l'instant", "il y a 14 min", "il y a 1 h", "ce matin", "hier soir", "mardi dernier".
- Currency: "1 234 €" (space thousand separator, € suffix).
- Amount on cards: stat-style serif weight 500 → `"3 213 €"` not `"€3,213"`.
- Calls-to-action: "Publier une annonce", "Démarrer un projet", "Contacter", "Voir le détail", "Continuer →" (always with arrow on continue).
- Section labels (mono): UPPERCASE letter-spacing 0.08-0.12em — `"BON RETOUR"`, `"ATELIER · LE MOT JUSTE"`.

A French formatter helper `formatRelativeFr(date)` ships in `web/src/shared/lib/format.ts` and `mobile/lib/core/format/relative_fr.dart`.

---

## 10. Mapping to existing repo

This section grows as Phase 0 progresses.

### Web tokens — `web/src/styles/globals.css`

```css
@theme inline {
  --color-background: #fffbf5;
  --color-card: #ffffff;
  --color-border: #f0e6d8;
  --color-border-strong: #e0d3bc;
  --color-foreground: #2a1f15;
  --color-muted-foreground: #7a6850;
  --color-subtle-foreground: #a89679;
  --color-primary: #e85d4a;
  --color-primary-soft: #fde9e3;
  --color-primary-deep: #c43a26;
  --color-pink: #f08aa8;
  --color-pink-soft: #fde6ed;
  --color-success: #5a9670;
  --color-success-soft: #e8f2eb;
  --color-amber: #d4924a;

  --font-sans: 'Inter Tight', system-ui, sans-serif;
  --font-serif: 'Fraunces', Georgia, serif;
  --font-mono: 'Geist Mono', monospace;

  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 14px;
  --radius-xl: 18px;
  --radius-2xl: 20px;
}
```

### Admin tokens — `admin/src/index.css`

Same `@theme` block. Admin shares the web tokens 1:1.

### Mobile tokens — `mobile/lib/core/theme/soleil_theme.dart`

```dart
class SoleilColors extends ThemeExtension<SoleilColors> {
  static const ivoire = Color(0xFFFFFBF5);
  static const encre = Color(0xFF2A1F15);
  static const tabac = Color(0xFF7A6850);
  static const sable = Color(0xFFA89679);
  static const corail = Color(0xFFE85D4A);
  static const corailDeep = Color(0xFFC43A26);
  static const corailSoft = Color(0xFFFDE9E3);
  static const sapin = Color(0xFF5A9670);
  static const sapinSoft = Color(0xFFE8F2EB);
  // ... etc
}
```

ThemeData wires Material 3 `colorScheme` from these constants, plus the extension for non-Material tokens.

---

## Source-of-truth references

When in doubt, the **JSX source** wins:
- Direction file: [`assets/sources/phase1/soleil.jsx`](./assets/sources/phase1/soleil.jsx)
- Per-screen implementations: `assets/sources/phase1/soleil-lot{A,B,C,D,E,F}.jsx` (web desktop), `soleil-lot{A,B,C,D,E,F}-mobile.jsx` (responsive), `soleil-app-lot{1,2,3,4,5}.jsx` (native iOS).
- Visual proof: [`assets/pdf/web-desktop.pdf`](./assets/pdf/web-desktop.pdf), `web-responsive.pdf`, `app-native-ios.pdf`.

The HTML files in `assets/sources/` are canvas wrappers for visualizing the JSX. Not authoritative.
