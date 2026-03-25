# Marketplace Design System

A premium B2B marketplace design language — warm, confident, crafted. Inspired by Contra.com's boldness, Google Stitch's systematic rigor, and Airbnb's hospitality-driven aesthetic.

This document is the single source of truth for all visual decisions across web, admin, and mobile apps. Every component, every screen, every interaction follows these rules. No exceptions.

---

## Design Philosophy

Three principles guide every decision:

1. **Clarity over decoration.** Every pixel earns its place. If an element does not help the user complete a task or understand information, remove it.
2. **Warmth over corporate.** B2B does not mean boring. Rose as a primary color signals energy, approachability, and distinction in a sea of blue SaaS products.
3. **Motion with purpose.** Animations exist to communicate state changes and spatial relationships, never to show off. Every transition under 300ms. Every animation serves comprehension.

---

## 1. Color Palette

### Primary (Rose)

Rose conveys warmth, energy, and professionalism. It stands out in the B2B space dominated by blues and grays while remaining accessible at all contrast ratios.

| Token | Hex | Tailwind | Usage |
|-------|-----|----------|-------|
| primary-50 | #FFF1F2 | rose-50 | Backgrounds, hover states, selected row highlight |
| primary-100 | #FFE4E6 | rose-100 | Light backgrounds, selected states, avatar fallback bg |
| primary-200 | #FECDD3 | rose-200 | Borders on primary elements, interactive card hover border |
| primary-300 | #FDA4AF | rose-300 | Inactive/disabled primary elements |
| primary-400 | #FB7185 | rose-400 | Hover on primary buttons, dark mode primary |
| primary-500 | #F43F5E | rose-500 | **Primary actions, CTAs, links, active navigation** |
| primary-600 | #E11D48 | rose-600 | Active/pressed state on primary buttons, gradient endpoint |
| primary-700 | #BE123C | rose-700 | Dark variant, text on light primary backgrounds |
| primary-800 | #9F1239 | rose-800 | Very dark variant (text on light bg) |
| primary-900 | #881337 | rose-900 | Darkest variant |

### Neutral (Slate)

Slate provides a cool, professional neutral that pairs naturally with rose without competing.

| Token | Hex | Tailwind | Usage |
|-------|-----|----------|-------|
| foreground | #0F172A | slate-900 | Primary text, headings |
| foreground-secondary | #334155 | slate-700 | Strong secondary text |
| muted-foreground | #64748B | slate-500 | Secondary text, placeholders, timestamps |
| muted | #F1F5F9 | slate-100 | Backgrounds, disabled states, table stripes |
| border | #E2E8F0 | slate-200 | Borders, dividers, input borders |
| border-strong | #CBD5E1 | slate-300 | Active borders, stronger separators |
| card | #FFFFFF | white | Card backgrounds |
| background | #FFFFFF | white | Page background |
| sidebar | #FAFAFA | -- | Sidebar background (or glass effect) |

### Semantic

| Token | Hex | Tailwind | Usage |
|-------|-----|----------|-------|
| success | #22C55E | green-500 | Completed, verified, online, approved |
| success-light | #F0FDF4 | green-50 | Success background (badges, alerts) |
| warning | #F59E0B | amber-500 | Pending, attention needed, expiring soon |
| warning-light | #FFFBEB | amber-50 | Warning background |
| destructive | #EF4444 | red-500 | Errors, delete actions, danger states |
| destructive-light | #FEF2F2 | red-50 | Error background |
| info | #3B82F6 | blue-500 | Informational, external links |
| info-light | #EFF6FF | blue-50 | Info background |

### Role Colors

Each marketplace role has a dedicated color for instant visual identification.

| Role | Color | Hex | Tailwind | Badge bg | Badge text |
|------|-------|-----|----------|----------|------------|
| Agency | Blue | #3B82F6 | blue-500 | blue-50 | blue-700 |
| Enterprise | Purple | #8B5CF6 | violet-500 | violet-50 | violet-700 |
| Provider | Rose | #F43F5E | rose-500 | rose-50 | rose-700 |
| Admin | Slate | #64748B | slate-500 | slate-100 | slate-700 |

### Dark Mode

Same tokens, inverted values. Dark mode uses slate-900 as background and lighter shades for text.

| Token | Light | Dark |
|-------|-------|------|
| primary | #F43F5E | #FB7185 |
| background | #FFFFFF | #0F172A |
| foreground | #0F172A | #F8FAFC |
| foreground-secondary | #334155 | #CBD5E1 |
| muted-foreground | #64748B | #94A3B8 |
| muted | #F1F5F9 | #1E293B |
| border | #E2E8F0 | #334155 |
| border-strong | #CBD5E1 | #475569 |
| card | #FFFFFF | #1E293B |
| sidebar | #FAFAFA | #0F172A |
| success | #22C55E | #4ADE80 |
| success-light | #F0FDF4 | #052E16 |
| warning | #F59E0B | #FBBF24 |
| warning-light | #FFFBEB | #451A03 |
| destructive | #EF4444 | #F87171 |
| destructive-light | #FEF2F2 | #450A0A |
| info | #3B82F6 | #60A5FA |
| info-light | #EFF6FF | #172554 |

---

## 2. Gradients

Gradients add depth and visual hierarchy. Use them sparingly — a gradient should draw the eye to the most important element on the page.

### Gradient Tokens

| Token | CSS Value | Tailwind | Usage |
|-------|-----------|----------|-------|
| gradient-primary | `linear-gradient(135deg, #F43F5E 0%, #E11D48 100%)` | `from-rose-500 to-rose-600` | Primary buttons, primary CTAs |
| gradient-hero | `linear-gradient(135deg, #F43F5E 0%, #A855F7 50%, #6366F1 100%)` | `from-rose-500 via-purple-500 to-indigo-500` | Welcome banners, hero sections, onboarding |
| gradient-subtle | `linear-gradient(180deg, #FFF1F2 0%, #FFFFFF 100%)` | `from-rose-50 to-white` | Card backgrounds, section highlights |
| gradient-warm | `linear-gradient(135deg, #FFF1F2 0%, #EFF6FF 100%)` | `from-rose-50 to-blue-50` | Feature section backgrounds |
| gradient-dark | `linear-gradient(135deg, #0F172A 0%, #1E293B 100%)` | `from-slate-900 to-slate-800` | Dark mode hero, footer |

### Gradient Rules

- Maximum ONE gradient per viewport. Two gradients competing for attention creates visual noise.
- Gradients are reserved for hero sections, primary CTAs, and welcome banners. Never on body text backgrounds, table rows, or secondary UI.
- The hero gradient (rose-purple-indigo) is used ONLY for full-width welcome banners and onboarding screens. It is the most visually prominent element and must not be diluted.
- Buttons use gradient-primary (rose-500 to rose-600) for a subtle depth effect. The gradient should be barely noticeable — it adds polish, not drama.
- Dark mode gradients use the dark palette equivalents. Never show light gradients on dark backgrounds.

---

## 3. Typography

### Font Stack

| Role | Family | Fallback |
|------|--------|----------|
| Sans (primary) | Geist Sans | -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif |
| Mono (numbers, code) | Geist Mono | 'SF Mono', 'Fira Code', 'Fira Mono', monospace |

### Type Scale

| Level | Size | Weight | Line-height | Letter-spacing | Usage |
|-------|------|--------|-------------|----------------|-------|
| Display | 36px (2.25rem) | 800 | 1.1 | -0.025em | Landing hero headlines |
| H1 | 30px (1.875rem) | 700 | 1.2 | -0.025em | Page titles |
| H2 | 24px (1.5rem) | 600 | 1.3 | -0.02em | Section titles |
| H3 | 20px (1.25rem) | 600 | 1.4 | -0.015em | Card titles, dialog titles |
| H4 | 18px (1.125rem) | 500 | 1.4 | -0.01em | Subsections, sidebar group labels |
| Body Large | 18px (1.125rem) | 400 | 1.6 | 0 | Featured text, intros |
| Body | 15px (0.9375rem) | 400 | 1.6 | 0 | Default text, paragraphs |
| Body Small | 14px (0.875rem) | 400 | 1.5 | 0 | Secondary text, table cells |
| Caption | 13px (0.8125rem) | 500 | 1.4 | 0 | Labels, timestamps, metadata |
| Overline | 12px (0.75rem) | 600 | 1.4 | 0.05em | Category labels (uppercase) |
| Stat Number | 30px (1.875rem) | 700 | 1.1 | -0.02em | Dashboard stat values (use Geist Mono) |

### Typography Rules

- Body text is 15px, not 16px. The slightly smaller size creates a more refined, editorial feel while remaining perfectly readable. Forms keep 16px to prevent iOS zoom.
- Stat numbers and financial figures use **Geist Mono** for tabular alignment and visual distinction.
- Maximum 2 font weights visible on any single screen.
- Headings use slate-900 (foreground), body uses slate-700 (foreground-secondary) or slate-500 (muted-foreground).
- Links use primary-500, underline on hover only.
- Never use font sizes outside this scale.
- Responsive: Display drops to 30px on mobile, H1 drops to 24px.
- Tight letter-spacing on headings (-0.025em) creates a premium, editorial feel.

---

## 4. Spacing Scale

Base unit: **4px**. All spacing values are multiples of 4. Only use these values throughout the entire codebase.

| Token | Value | Tailwind | Common usage |
|-------|-------|----------|-------------|
| space-1 | 4px | p-1 / m-1 | Tight inline spacing, icon gaps |
| space-2 | 8px | p-2 / m-2 | Badge padding, compact lists |
| space-3 | 12px | p-3 / m-3 | Small card padding, mobile card gap |
| space-4 | 16px | p-4 / m-4 | Input padding, form field gap, mobile page padding |
| space-5 | 20px | p-5 / m-5 | Medium spacing |
| space-6 | 24px | p-6 / m-6 | Card padding (standard), tablet page padding, form section gap |
| space-8 | 32px | p-8 / m-8 | Desktop page padding, section spacing (mobile) |
| space-10 | 40px | p-10 / m-10 | Section spacing (tablet) |
| space-12 | 48px | p-12 / m-12 | Section spacing (desktop), desktop page padding (large) |
| space-16 | 64px | p-16 / m-16 | Large section separators |
| space-20 | 80px | p-20 / m-20 | Hero padding |
| space-24 | 96px | p-24 / m-24 | Major layout sections |
| space-32 | 128px | -- | Page-level vertical rhythm |

---

## 5. Border Radius

| Token | Value | Tailwind | Usage |
|-------|-------|----------|-------|
| radius-sm | 6px | rounded-md | Badges, tags, small elements |
| radius-md | 8px | rounded-lg | Buttons, inputs, dropdowns |
| radius-lg | 12px | rounded-xl | Cards, modals, sheets |
| radius-xl | 16px | rounded-2xl | Large cards, hero sections, images, stat cards |
| radius-full | 9999px | rounded-full | Avatars, pills, circular buttons |

### Radius Rules

- Never mix radius sizes within the same component (if a card uses xl, its inner elements use lg or md).
- Nested elements always use the same or smaller radius than their parent.
- Images inside cards: use the card's radius with `overflow-hidden` on the card container.
- Stat cards and dashboard cards use radius-xl (16px) for a softer, more modern appearance.

---

## 6. Shadows

### Shadow Scale

| Token | Value | Usage |
|-------|-------|-------|
| shadow-xs | `0 1px 2px rgba(0, 0, 0, 0.04)` | Subtle depth on inputs, badges |
| shadow-sm | `0 2px 4px rgba(0, 0, 0, 0.06)` | Cards at rest, static containers |
| shadow-md | `0 4px 12px rgba(0, 0, 0, 0.08)` | Cards on hover, raised elements, active dropdowns |
| shadow-lg | `0 8px 24px rgba(0, 0, 0, 0.12)` | Modals, popovers, toasts |
| shadow-xl | `0 16px 48px rgba(0, 0, 0, 0.16)` | Floating elements, command palette, large modals |
| shadow-glow | `0 0 20px rgba(244, 63, 94, 0.3)` | Primary CTA glow effect on hover |

### Shadow Rules

- Cards at rest: shadow-sm. On hover: transition to shadow-md.
- Floating UI (modals, dropdowns, popovers): shadow-lg.
- The glow shadow is reserved for the single most important CTA on a page. Never use it on more than one element at a time.
- Dark mode: shadows use `rgba(0, 0, 0, 0.4)` base for visibility against dark backgrounds.
- Never stack or combine shadows. One shadow per element.
- Inputs use shadow-xs for subtle depth, transitioning to no shadow on focus (the focus ring replaces the shadow).

---

## 7. Animation System

### Timing Tokens

| Token | Duration | Easing | Usage |
|-------|----------|--------|-------|
| duration-fast | 150ms | ease-out | Button states, toggles, icon transitions |
| duration-normal | 200ms | ease-out | Card hover, dropdown open, focus ring |
| duration-slow | 300ms | ease-out | Modal enter, page transitions, content reveal |
| duration-shimmer | 1.5s | linear (infinite) | Skeleton loading shimmer |

### Default Transition

All interactive elements use: `transition-all duration-200 ease-out`

This is the baseline. Override duration only when the component warrants it.

### Keyframe Animations

| Animation | Keyframes | Duration | Usage |
|-----------|-----------|----------|-------|
| shimmer | Gradient sweep left-to-right | 1.5s infinite | Skeleton loading placeholders |
| slideUp | opacity 0 + translateY(8px) to visible | 300ms ease-out | Content entering the viewport, list items |
| scaleIn | opacity 0 + scale(0.95) to visible | 200ms ease-out | Modals, dropdowns, popovers |
| glow | box-shadow pulse 20px to 30px | 2s infinite | Primary CTA attention pulse |
| fadeIn | opacity 0 to 1 | 200ms ease-out | Generic fade entrance |

### Micro-interactions

| Element | Interaction | CSS |
|---------|-------------|-----|
| Button (press) | Scale down slightly | `active:scale-[0.98]` |
| Button (primary hover) | Glow shadow appears | `hover:shadow-glow` |
| Card (hover) | Lift up with shadow upgrade | `hover:-translate-y-0.5 hover:shadow-md` |
| Card (interactive hover) | Border tints rose | `hover:border-rose-200` |
| Toggle | Smooth slide with spring | `transition-all duration-200 ease-out` |
| Modal (enter) | Scale from 95% + fade | `animate-scale-in` |
| Toast (enter) | Slide down from top-right | `animate-slide-up` |
| Page content | Staggered fade + slide up | `animate-slide-up` with stagger-50 delay |
| Avatar (hover) | Rose ring appears | `hover:ring-2 hover:ring-rose-500 hover:ring-offset-2` |
| Stat number | Count up on load | CSS or JS counter animation, 500ms |
| Skeleton | Shimmer gradient sweep | `animate-shimmer` |

### Stagger Pattern

When multiple items enter the viewport (card grids, list items, stat cards), stagger their entrance animations by 50ms per item:

```
Item 1: delay 0ms
Item 2: delay 50ms
Item 3: delay 100ms
Item 4: delay 150ms
```

Maximum stagger: 5 items (250ms total). Beyond 5, show all remaining items at the 250ms mark. Users should never wait more than 550ms (300ms animation + 250ms stagger) for all content to appear.

### Motion Rules

- Respect `prefers-reduced-motion`: disable all animations except opacity fades.
- Never animate layout properties (width, height, top, left) -- use `transform` and `opacity`.
- Keep all animations under 300ms -- users should never wait for an animation.
- Loading states appear instantly (0ms delay), never after a timer.
- Exit animations are faster than enter animations (150ms vs 200-300ms).

---

## 8. Glass Effect

Glass morphism adds depth and sophistication to persistent UI elements like sidebars and headers.

### Glass Tokens

| Token | CSS | Usage |
|-------|-----|-------|
| glass | `background: rgba(255, 255, 255, 0.8); backdrop-filter: blur(20px);` | Sidebar, header, floating panels |
| glass-strong | `background: rgba(255, 255, 255, 0.9); backdrop-filter: blur(24px);` | Command palette, critical overlays |
| glass-subtle | `background: rgba(255, 255, 255, 0.6); backdrop-filter: blur(12px);` | Tooltip backgrounds, hover cards |

### Glass Rules

- Glass is used ONLY on elements that overlay scrollable content (sidebar, sticky header, floating panels).
- Always pair glass with a subtle border: `border: 1px solid rgba(255, 255, 255, 0.2)` (light) or `border: 1px solid rgba(255, 255, 255, 0.05)` (dark).
- Dark mode glass: `background: rgba(15, 23, 42, 0.8)` (slate-900 at 80%).
- Test glass effects on both solid and patterned/image backgrounds to ensure readability.
- Always include `-webkit-backdrop-filter` for Safari support.

---

## 9. Component Patterns

### Buttons

5 variants, 3 sizes. All buttons use radius-md (8px).

**Variants:**

| Variant | Background | Text | Border | Hover | Active |
|---------|------------|------|--------|-------|--------|
| primary | gradient-primary (rose-500 to rose-600) | white | none | shadow-glow | scale-[0.98] + rose-700 bg |
| secondary | white | slate-900 | 1px slate-200 | bg-gray-50 shadow-sm to shadow-md | scale-[0.98] |
| outline | transparent | slate-700 | 1px slate-200 | bg-gray-50 border-slate-300 | scale-[0.98] |
| ghost | transparent | slate-700 | none | bg-gray-100 | scale-[0.98] |
| destructive | red-500 | white | none | red-600 shadow-md | scale-[0.98] |

**Sizes:**

| Size | Height | Padding | Font size | Icon size |
|------|--------|---------|-----------|-----------|
| sm | 32px (h-8) | px-3 | 13px | 16px |
| md | 40px (h-10) | px-4 | 14px (text-sm) | 18px |
| lg | 48px (h-12) | px-6 | 15px | 20px |

**States:**
- Default: as defined above
- Hover: as defined in Hover column + `transition-all duration-200 ease-out`
- Active/pressed: `scale-[0.98]` transform + slightly darker shade
- Disabled: `opacity-50 cursor-not-allowed`, no hover effects, no pointer events
- Loading: spinner replaces text content, same dimensions maintained, `pointer-events-none`. Spinner uses `currentColor` to match the button's text color.

**Icon-only buttons:** 40px circle (md size), `aria-label` required, tooltip on hover with 500ms delay.

**Icon button hover:** `hover:bg-gray-100 rounded-full transition-all duration-200`

### Cards

**Default Card:**
- Background: white
- Border: 1px solid border (slate-100 for softer appearance)
- Radius: radius-xl (16px) -- rounded-2xl
- Shadow: shadow-sm at rest
- Padding: p-6 (24px) standard

**Interactive Card (clickable):**
- Hover: `shadow-md border-rose-200 -translate-y-0.5`
- Transition: `all 200ms ease-out`
- Focus-visible: `ring-2 ring-primary-500 ring-offset-2`
- Cursor: `pointer`

**Featured Card (highlighted/promoted):**
- Border: 2px solid with gradient (rose-500 to purple-500)
- Shadow: `shadow-glow` at rest
- Optional: subtle rose-50 background

**Glass Card (overlay contexts):**
- Background: `rgba(255, 255, 255, 0.8)`
- Backdrop: `blur(20px)`
- Border: `1px solid rgba(255, 255, 255, 0.2)`
- No shadow (the blur provides depth)

**Card Rules:**
- Never nest cards inside cards.
- Card headers: flex with title (H3) left, actions right.
- Card footers: border-t border-slate-100, pt-4.
- Card grids use gap-6 (24px) between cards.

### Stat Cards

Stat cards are the primary data visualization element on dashboards. They deserve special attention.

**Structure:**
```
+--------------------------------------------------+
|  [icon-circle]                          [trend]   |
|                                                    |
|  Label (caption, muted-foreground)                |
|  Value (stat-number, 30px, font-bold)             |
|  Description (body-small, muted-foreground)       |
+--------------------------------------------------+
```

**Specifications:**
- Container: bg-white rounded-2xl border border-slate-100 shadow-sm p-6
- Icon circle: 48px (w-12 h-12), rounded-full, uses semantic color at 10% opacity as background (e.g., `bg-rose-50` for primary, `bg-green-50` for success), icon at 24px in the corresponding solid color
- Value: 30px (text-3xl), font-bold, slate-900, Geist Mono for tabular numbers
- Label: 13px (caption), font-medium, slate-500, positioned ABOVE the value
- Trend badge: pill shape (`px-2 py-0.5 rounded-full text-xs font-medium`), green-50/green-700 for positive, red-50/red-700 for negative. Arrow icon (TrendingUp/TrendingDown) at 14px before the percentage.
- Description: 14px, slate-500, optional subtitle below the value
- Hover: `shadow-md -translate-y-0.5 transition-all duration-200`
- Subtle gradient background: optional `from-white to-rose-50/30` for primary stats

### Hero Banner / Welcome Banner

The welcome banner is the first element users see on their dashboard. It sets the emotional tone.

**Structure:**
```
+----------------------------------------------------------------------+
|  gradient-hero (rose-500 via purple-500 to indigo-500)               |
|                                                                       |
|  "Welcome back, {first_name}"  (H1, white, font-bold)               |
|  "{role-specific subtitle}"    (body-large, white/80)                |
|                                                                       |
|  [Primary CTA: white bg]  [Secondary CTA: white/20 border]          |
|                                                    [CSS pattern/dots] |
+----------------------------------------------------------------------+
```

**Specifications:**
- Full-width within the content area, rounded-2xl
- Background: `gradient-hero` (rose-500 via purple-500 to indigo-500 at 135deg)
- Padding: `px-8 py-10` (desktop), `px-6 py-8` (mobile)
- Heading: H1 size (30px), font-bold, white
- Subtitle: body-large (18px), `text-white/80`
- Primary CTA button: white background, slate-900 text, hover shadow-md
- Secondary CTA button: `bg-white/20` background, white text, `border border-white/30`, hover `bg-white/30`
- Right side decoration: CSS-only decorative pattern (grid of dots, circles, or geometric shapes) using `bg-white/10`. This is purely decorative and adds visual texture.
- Dark mode: same gradient, slightly adjusted opacity

### Avatars

5 sizes, always `rounded-full`.

| Size | Dimensions | Tailwind | Usage |
|------|------------|----------|-------|
| xs | 24px | w-6 h-6 | Inline mentions, compact lists |
| sm | 32px | w-8 h-8 | Comment threads, table rows |
| md | 40px | w-10 h-10 | Cards, navigation, default |
| lg | 48px | w-12 h-12 | Profile headers, detail views |
| xl | 64px | w-16 h-16 | Profile pages, hero sections |

**Fallback:** When no image is available, show initials (first letter of first name + first letter of last name) on `primary-100` background with `primary-700` text. Font size scales with avatar size.

**Hover state:** `ring-2 ring-rose-500 ring-offset-2 transition-all duration-200` -- the ring appears smoothly on hover.

**Status dot:** Absolute positioned at bottom-right. 10px circle with 2px white border (ring). Green for online, slate-300 for offline, amber for away.

**Completion ring:** A circular progress indicator (SVG stroke-dasharray) around the avatar showing profile completion percentage. Stroke color is rose-500, track is slate-100. Used on profile pages.

**Group display:** Overlapping avatars with `-ml-2` offset, `ring-2 ring-white` on each to create separation. Max 4 visible + "+N" indicator.

### Forms and Inputs

**Upgraded Input:**

| State | Border | Ring | Background | Additional |
|-------|--------|------|------------|------------|
| Default | border-gray-200 | none | white | shadow-xs |
| Hover | border-gray-300 | none | white | shadow-xs |
| Focus | border-rose-500 | ring-4 ring-rose-500/10 | white | no shadow-xs (ring replaces it) |
| Error | border-red-500 | ring-4 ring-red-500/10 | white | error message below |
| Disabled | border-gray-200 | none | gray-50 | cursor-not-allowed, opacity-60 |

- Height: h-10 (40px) for md, h-8 (32px) for sm
- Radius: radius-md (8px)
- Padding: px-3 (12px)
- Text: 16px (to prevent iOS zoom)
- Transition: `border-color 200ms ease-out, box-shadow 200ms ease-out`

**With icon:** Leading icon (left side) inside the input at 18px. Add `pl-10` to the input for padding. Icon color: slate-400, transitions to slate-600 on focus.

**Password input:** Trailing eye icon (EyeOff/Eye) as a ghost button inside the input. Toggles visibility. Icon at 18px, `pr-10` on input.

**Labels:**
- Always above the input, never floating or placeholder-only
- Size: 13px (caption), font-medium (500)
- Required fields: asterisk in destructive color after label text
- Spacing: gap-1.5 (6px) between label and input, gap-4 (16px) between fields, gap-6 (24px) between sections

**Textarea:** minimum h-24 (96px), resize-y only, same border/focus styling as inputs.

**Select:** Same styling as input, with ChevronDown icon right-aligned at 16px, slate-400 color.

### Badges and Status

**Role badges:**

| Role | Background | Text | Border |
|------|------------|------|--------|
| Agency | blue-50 | blue-700 | blue-200 |
| Enterprise | violet-50 | violet-700 | violet-200 |
| Provider | rose-50 | rose-700 | rose-200 |
| Admin | slate-100 | slate-700 | slate-300 |

**Status badges:**

| Status | Background | Text | Dot color |
|--------|------------|------|-----------|
| Success | green-50 | green-700 | green-500 |
| Warning | amber-50 | amber-700 | amber-500 |
| Error | red-50 | red-700 | red-500 |
| Info | blue-50 | blue-700 | blue-500 |
| Neutral | slate-100 | slate-600 | slate-400 |

**Badge styling:** `px-2.5 py-0.5 text-xs font-medium rounded-full`, optional leading dot (`w-1.5 h-1.5 rounded-full`).

**Trend badge (new):** Used on stat cards. Pill shape with directional arrow.
- Positive: `bg-green-50 text-green-700` with TrendingUp icon (14px) + value (e.g., "+18%")
- Negative: `bg-red-50 text-red-700` with TrendingDown icon (14px) + value (e.g., "-5%")
- Neutral: `bg-slate-100 text-slate-600` with Minus icon (14px) + "0%"

### Toast Notifications

- Position: top-right corner, 16px from edges
- Max width: 420px
- Stack: newest on top, max 3 visible, older ones auto-dismissed

**Structure:**
```
+---+------------------------------------------+---+
| | |  Title (14px, font-semibold)              | X |
| B |  Description (13px, muted-foreground)     |   |
| | |                                            |   |
+---+------------------------------------------+---+
|  [progress bar - auto-dismiss timer]             |
+--------------------------------------------------+
```

**Specifications:**
- Container: white bg, rounded-xl, shadow-lg, overflow-hidden
- Left border: 4px solid in semantic color
- Icon: semantic color, 20px, positioned left of text content
- Close button: X icon, ghost button, top-right corner
- Progress bar: 2px height at bottom, semantic color, animates from 100% to 0% width over the auto-dismiss duration
- Enter animation: slide down from top-right + fade in (200ms ease-out)
- Exit animation: fade out + slide right (150ms ease-in)

| Type | Left border | Icon | Icon color | Auto-dismiss |
|------|-------------|------|------------|--------------|
| Success | green-500 | CheckCircle | green-500 | 3s |
| Error | red-500 | XCircle | red-500 | 5s (critical: manual dismiss required) |
| Warning | amber-500 | AlertTriangle | amber-500 | 4s |
| Info | blue-500 | Info | blue-500 | 4s |

---

## 10. Layout Standards

### Dashboard Layout

The dashboard layout uses glass effects for a premium, modern feel.

**Sidebar:**
- Width: 280px (desktop), 0px collapsed (tablet/mobile)
- Background: glass effect (`bg-white/80 backdrop-blur-xl`) with `border-r border-gray-100`
- Logo area: h-16, flex center, border-b border-gray-100
- Nav items: `px-3 py-2 rounded-lg text-sm font-medium`, hover `bg-gray-100`, active `bg-rose-50 text-rose-700 font-semibold`
- Active indicator: 3px left border in rose-500 on the active nav item, or filled background
- Bottom section: user avatar + name + role badge, border-t, p-4
- Mobile: slides in from left as a sheet with overlay

**Header:**
- Height: h-16 (64px)
- Background: glass effect (`bg-white/80 backdrop-blur-xl`)
- Position: `sticky top-0 z-50`
- Border: `border-b border-gray-100`
- Content: breadcrumb left, search/notifications/avatar right
- Mobile: hamburger menu button replaces sidebar toggle

**Content Area:**
- Max width: `max-w-7xl` (1280px)
- Centering: `mx-auto`
- Padding: `px-6 py-8` (desktop), `px-4 py-6` (mobile)
- Cards grid: `gap-6`

### Welcome Banner Layout

The welcome banner sits at the top of the dashboard content area, above the stats grid.

```
+-- Content Area (max-w-7xl mx-auto px-6 py-8) ----+
|                                                     |
|  [Welcome Banner - full width, rounded-2xl]        |
|                                                     |
|  [Stat Card] [Stat Card] [Stat Card] [Stat Card]  |
|                                                     |
|  [Main Content Section]                            |
|                                                     |
+----------------------------------------------------+
```

### Grid System

| Context | Mobile (<640px) | Tablet (640-1024px) | Desktop (>1024px) |
|---------|-----------------|---------------------|---------------------|
| Stat cards | 1 column, full-width | 2 columns | 4 columns |
| Content cards | 1 column | 2 columns | 3 columns |
| Form layout | Single column | Single column (max-w-lg centered) | Single column (max-w-lg centered) |
| Card gap | 16px (gap-4) | 20px (gap-5) | 24px (gap-6) |
| Page padding | 16px | 24px | 24-48px |

### Content Width Constraints

| Content type | Max width | Reason |
|-------------|-----------|--------|
| Prose/text | 65ch (~700px) | Optimal reading line length |
| Forms | 640px (max-w-lg) | Inputs should not stretch too wide |
| Dashboards | 1280px (max-w-7xl) | Full use of screen |
| Settings | 768px (max-w-3xl) | Focused, form-heavy pages |

### Responsive Breakpoints

| Name | Min width | Tailwind prefix | Typical devices |
|------|-----------|-----------------|-----------------|
| Mobile | 0px | (default) | Phones |
| sm | 640px | sm: | Large phones, small tablets |
| md | 768px | md: | Tablets portrait |
| lg | 1024px | lg: | Tablets landscape, small laptops |
| xl | 1280px | xl: | Laptops, desktops |
| 2xl | 1536px | 2xl: | Large desktops |

Design mobile-first. Use `min-width` breakpoints (Tailwind default). Never use `max-width` unless absolutely necessary.

---

## 11. Interaction Patterns

| Interaction | Web | Mobile |
|-------------|-----|--------|
| Hover | shadow upgrade + border tint + translateY | N/A (no hover on touch devices) |
| Focus | ring-2 ring-primary-500 ring-offset-2 | N/A (use active state) |
| Active/Press | scale-[0.98] + darker shade | scale-[0.97] + haptic feedback (light impact) |
| Transition | 200ms ease-out (default) | 200ms ease-out |
| Loading (page) | Skeleton matching content shape with shimmer | Skeleton + shimmer effect |
| Loading (button) | Spinner replacing content, same dimensions | Same |
| Empty state | Illustration + message + CTA button | Same |
| Pull to refresh | N/A | CircularProgressIndicator in primary-500 |
| Error state | Inline message + retry button | Same |

### Skeleton Loading

- Match the exact shape of the content being loaded (card skeleton looks like a card, stat skeleton looks like a stat card).
- Background: `bg-slate-200/60` (slightly transparent for softer appearance).
- Shimmer: animated gradient sweep left-to-right using `animate-shimmer` class. The gradient is `transparent -> white/40 -> transparent` moving across a 200% width background.
- Never use a full-page spinner for content loading.
- Spinners are acceptable only for button loading states and inline actions.
- Show skeleton immediately (0ms delay), transition to content with a fade (150ms).
- Skeleton corner radius matches the content it represents.

---

## 12. Iconography

- Icon set: **Lucide** (web and admin), **Lucide equivalent or Material** (Flutter)
- Default size: 18px for inline, 20px for buttons, 24px for standalone
- Stroke width: 1.5px (default Lucide)
- Color: inherit from parent text color (currentColor)
- Interactive icons: wrapped in a ghost button (40px touch target) with `aria-label` and tooltip

### Icon Usage Reference

| Action | Icon | Notes |
|--------|------|-------|
| Search | `Search` | Always in search input left slot |
| Close | `X` | Modals, toasts, sheets |
| Menu | `Menu` | Mobile hamburger |
| Back | `ChevronLeft` | Navigation back |
| More | `MoreHorizontal` | Context menu trigger |
| Edit | `Pencil` | Edit actions |
| Delete | `Trash2` | Destructive actions (always red) |
| Add | `Plus` | Create new |
| Filter | `SlidersHorizontal` | Filter panels |
| Sort | `ArrowUpDown` | Table sort |
| Check | `Check` | Success, selected |
| Alert | `AlertCircle` | Warnings, errors |
| Info | `Info` | Informational |
| User | `User` | Profile, account |
| Settings | `Settings` | Configuration |
| Logout | `LogOut` | Sign out |
| Trend Up | `TrendingUp` | Positive stat change |
| Trend Down | `TrendingDown` | Negative stat change |
| Eye | `Eye` / `EyeOff` | Password visibility toggle |
| Notification | `Bell` | Notification center |
| Calendar | `Calendar` | Dates, scheduling |
| Money | `DollarSign` | Financial, billing |

---

## 13. Platform Specificities

### Web

- Hover states on all interactive elements (buttons, cards, links, table rows)
- Keyboard focus indicators on every focusable element (WCAG 2.1 AA)
- `cursor-pointer` on all clickable elements
- Tooltips on icon-only buttons (delay: 500ms)
- Breadcrumbs on all nested pages
- Glass effects on sidebar and header
- Staggered entrance animations on card grids

### Mobile (Flutter)

- Touch targets: minimum 48x48px (even if the visual element is smaller, the hit area must be 48px)
- Bottom sheet instead of modals for secondary actions (up to 3 options)
- Swipe gestures on list items (archive, delete) with colored background reveal
- Pull to refresh on all scrollable list screens
- Safe area padding: respect notch (top) and home indicator (bottom)
- Haptic feedback: light impact on primary button press, medium on destructive actions
- Splash screen: gradient-hero background, white logo centered
- Offline state: banner at top "You are offline", gray overlay on actions, cached data still visible
- iOS: native-feeling navigation transitions (slide from right)
- Android: Material-style transitions (fade through)

### Admin Panel

- Denser information display: body-small (14px) as default text size
- Wider tables: horizontal scroll on smaller screens
- Batch actions: checkbox column on tables, floating action bar
- Keyboard shortcuts for common actions (documented in a help modal)
- Minimal animation -- admin users prioritize speed over polish

---

## 14. Do / Don't Rules

### DO

- Use the spacing scale (multiples of 4px) for ALL spacing values
- Use semantic colors (success/warning/destructive/info) for status indicators
- Use skeleton loading with shimmer animation for all async content
- Use consistent padding within cards (always p-6)
- Use role-specific colors for badges (agency=blue, enterprise=purple, provider=rose)
- Use `transition-all duration-200 ease-out` on all interactive state changes
- Use `ring-2 ring-primary-500 ring-offset-2` for focus indicators
- Use gradient-primary on primary buttons for subtle depth
- Use glass effects on persistent overlay elements (sidebar, header)
- Use staggered animations on card grids and list entries
- Pair destructive actions with a confirmation dialog
- Truncate long text with ellipsis and provide full text via tooltip or expand
- Use empty state illustrations with a clear message and actionable CTA
- Use radius-xl (16px) on dashboard cards for a modern, spacious feel
- Use Geist Mono for stat numbers and financial figures

### DON'T

- Never use more than 2 font weights on the same screen
- Never use inline styles -- always Tailwind classes or CSS tokens
- Never use hardcoded color hex values -- always use tokens
- Never use more than ONE gradient per viewport
- Never use a full-page spinner for content loading -- always skeleton
- Never mix rounded corners within a component (parent xl, children lg or smaller)
- Never use pure black (#000000) for text -- use foreground (slate-900)
- Never put primary-colored text on primary-colored background
- Never use disabled buttons without explaining why (tooltip or helper text)
- Never auto-dismiss error toasts -- user must acknowledge or it stays 5s minimum
- Never use browser alerts/confirms -- always custom modals
- Never use horizontal scroll for primary content (tables are the exception)
- Never truncate critical information (prices, dates, statuses) -- only descriptions
- Never use shadow-glow on more than one element per viewport
- Never animate layout properties (width, height) -- use transform and opacity
- Never use animations longer than 300ms -- users should not wait for UI

---

## 15. Quick Reference Card

Copy-paste these patterns. They represent the baseline for every new component.

**Card:**
```
bg-white rounded-2xl border border-slate-100 shadow-sm p-6
hover:shadow-md hover:border-rose-200 hover:-translate-y-0.5 transition-all duration-200
```

**Primary Button:**
```
gradient-primary text-white rounded-lg px-4 h-10 text-sm font-medium
shadow-sm hover:shadow-glow active:scale-[0.98] transition-all duration-200
```

**Input:**
```
w-full h-10 px-3 rounded-lg border border-gray-200 shadow-xs text-[15px]
focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:shadow-none
transition-all duration-200 placeholder:text-slate-400
```

**Stat Card:**
```
bg-white rounded-2xl border border-slate-100 shadow-sm p-6
[icon: w-12 h-12 rounded-full bg-{color}-50 flex items-center justify-center]
[label: text-[13px] font-medium text-slate-500]
[value: text-3xl font-bold text-slate-900 font-mono]
[trend: px-2 py-0.5 rounded-full text-xs font-medium bg-green-50 text-green-700]
```

**Glass Sidebar:**
```
w-[280px] h-screen bg-white/80 backdrop-blur-xl border-r border-gray-100
```

**Glass Header:**
```
h-16 bg-white/80 backdrop-blur-xl border-b border-gray-100 sticky top-0 z-50
```

**Welcome Banner:**
```
gradient-hero rounded-2xl px-8 py-10 text-white
[title: text-3xl font-bold]
[subtitle: text-lg text-white/80]
```

**Skeleton:**
```
bg-slate-200/60 rounded-{matching-radius} animate-shimmer
```
