# Marketplace Design System

Professional B2B marketplace design — warm, distinctive, inspired by Malt and Airbnb's rose/pink branding.

This document is the single source of truth for all visual decisions across web, admin, and mobile apps. Every component, every screen, every interaction must follow these rules.

---

## 1. Color Palette

### Primary (Rose)

The primary color conveys warmth, energy, and professionalism. Rose stands out in the B2B space dominated by blues and grays.

| Token | Hex | Tailwind | Usage |
|-------|-----|----------|-------|
| primary-50 | #FFF1F2 | rose-50 | Backgrounds, hover states, selected row highlight |
| primary-100 | #FFE4E6 | rose-100 | Light backgrounds, selected states, avatar fallback bg |
| primary-200 | #FECDD3 | rose-200 | Borders on primary elements, interactive card hover border |
| primary-300 | #FDA4AF | rose-300 | Inactive/disabled primary elements |
| primary-400 | #FB7185 | rose-400 | Hover on primary buttons, dark mode primary |
| primary-500 | #F43F5E | rose-500 | **Primary actions, CTAs, links, active navigation** |
| primary-600 | #E11D48 | rose-600 | Active/pressed state on primary buttons |
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
| sidebar | #FAFAFA | — | Sidebar background |

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

## 2. Typography

Font family: **Geist Sans** (web), system default (mobile). Clean, modern, professional.

Fallback stack: `'Geist Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`

| Level | Size | Weight | Line-height | Letter-spacing | Usage |
|-------|------|--------|-------------|----------------|-------|
| Display | 36px (2.25rem) | 800 | 1.1 | -0.025em | Landing hero headlines |
| H1 | 30px (1.875rem) | 700 | 1.2 | -0.025em | Page titles |
| H2 | 24px (1.5rem) | 600 | 1.3 | -0.02em | Section titles |
| H3 | 20px (1.25rem) | 600 | 1.4 | -0.015em | Card titles, dialog titles |
| H4 | 18px (1.125rem) | 500 | 1.4 | -0.01em | Subsections, sidebar labels |
| Body Large | 18px (1.125rem) | 400 | 1.6 | 0 | Featured text, intros |
| Body | 16px (1rem) | 400 | 1.5 | 0 | Default text, paragraphs |
| Body Small | 14px (0.875rem) | 400 | 1.5 | 0 | Secondary text, table cells |
| Caption | 12px (0.75rem) | 400 | 1.4 | 0 | Labels, timestamps, metadata |
| Overline | 12px (0.75rem) | 600 | 1.4 | 0.05em | Category labels (uppercase) |

### Typography rules

- Maximum 2 font weights visible on any single screen
- Headings use slate-900 (foreground), body uses slate-700 (foreground-secondary) or slate-500 (muted-foreground)
- Links use primary-500, underline on hover only
- Never use font sizes outside this scale
- Responsive: Display drops to 30px on mobile, H1 drops to 24px

---

## 3. Spacing Scale

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
| space-32 | 128px | — | Page-level vertical rhythm |

---

## 4. Border Radius

| Token | Value | Tailwind | Usage |
|-------|-------|----------|-------|
| radius-sm | 6px | rounded-md | Badges, tags, small elements |
| radius-md | 8px | rounded-lg | Buttons, inputs, dropdowns |
| radius-lg | 12px | rounded-xl | Cards, modals, sheets |
| radius-xl | 16px | rounded-2xl | Large cards, hero sections, images |
| radius-full | 9999px | rounded-full | Avatars, pills, circular buttons |

### Radius rules

- Never mix radius sizes within the same component (if a card uses lg, its inner elements use md or sm)
- Nested elements always use the same or smaller radius than their parent
- Images inside cards: use radius-lg minus the card padding, or `overflow-hidden` on the card

---

## 5. Shadows

| Token | Value | Usage |
|-------|-------|-------|
| shadow-sm | `0 1px 2px rgba(0, 0, 0, 0.05)` | Subtle elevation: cards at rest, static containers |
| shadow-md | `0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1)` | Interactive cards on hover, raised elements |
| shadow-lg | `0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1)` | Modals, dropdowns, popovers, toasts |

### Shadow rules

- Only 3 shadow levels. Never create custom shadows.
- Cards at rest: shadow-sm. On hover: transition to shadow-md.
- Floating UI (modals, dropdowns, popovers): shadow-lg.
- Dark mode: shadows use `rgba(0, 0, 0, 0.3)` instead of `0.1` for visibility.
- Never combine shadows (no `shadow-sm shadow-md`).

---

## 6. Component Patterns

### Buttons

5 variants, 3 sizes. All buttons use radius-md (8px).

**Variants:**

| Variant | Background | Text | Border | Usage |
|---------|------------|------|--------|-------|
| primary | primary-500 | white | none | Main CTAs: "Submit", "Create", "Save" |
| secondary | slate-100 | slate-900 | none | Secondary actions: "Cancel", "Back" |
| outline | transparent | slate-900 | 1px slate-200 | Tertiary actions: "Edit", "Filter" |
| ghost | transparent | slate-900 | none | Inline actions: "View all", toolbar buttons |
| destructive | red-500 | white | none | Danger: "Delete", "Remove", "Revoke" |

**Sizes:**

| Size | Height | Padding | Font size | Icon size |
|------|--------|---------|-----------|-----------|
| sm | 32px (h-8) | px-3 | 14px (text-sm) | 16px |
| md | 40px (h-10) | px-4 | 14px (text-sm) | 18px |
| lg | 48px (h-12) | px-6 | 16px (text-base) | 20px |

**States:**
- Default: as defined above
- Hover: darken background 10% (primary-600 for primary, slate-200 for secondary)
- Active/pressed: `scale-[0.98]` transform
- Disabled: `opacity-50`, `cursor-not-allowed`, no hover effect
- Loading: spinner replacing text or icon, same dimensions, `pointer-events-none`

**Icon-only buttons:** square (same width as height), `aria-label` required, tooltip on hover (web).

### Cards

- Background: white (card token)
- Border: 1px solid border (slate-200)
- Radius: radius-lg (12px)
- Shadow: shadow-sm at rest
- Padding: p-6 (24px) standard

**Interactive cards** (clickable):
- Hover: shadow-md, border transitions to primary-200, `cursor-pointer`
- Transition: `all 150ms ease-out`
- Focus-visible: ring-2 ring-primary-500 ring-offset-2

**Rules:**
- Never nest cards inside cards
- Card headers: flex with title (H3) left, actions right
- Card footers: border-t border-slate-200, pt-4

### Forms and Inputs

- Height: h-10 (40px) for md, h-8 (32px) for sm
- Border: 1px border-slate-200, radius-md
- Padding: px-3 (12px)
- Background: white
- Text: body size (16px) to prevent iOS zoom

**States:**
- Default: border-slate-200
- Hover: border-slate-300
- Focus: `ring-2 ring-primary-500 ring-offset-2` (MANDATORY for accessibility)
- Error: `border-destructive`, error message below in destructive color
- Disabled: `bg-slate-50 text-muted-foreground cursor-not-allowed`

**Labels:**
- Always above the input, never floating or placeholder-only
- Size: text-sm (14px), font-medium (500)
- Required fields: asterisk in destructive color after label text
- Spacing: gap-1.5 (6px) between label and input, gap-4 (16px) between fields, gap-6 (24px) between sections

**Textarea:** minimum h-24 (96px), resize-y only, same border/focus styling as inputs.

**Select:** same styling as input, with chevron-down icon right-aligned.

### Avatars

5 sizes, always `rounded-full`.

| Size | Dimensions | Tailwind | Usage |
|------|------------|----------|-------|
| xs | 24px | w-6 h-6 | Inline mentions, compact lists |
| sm | 32px | w-8 h-8 | Comment threads, table rows |
| md | 40px | w-10 h-10 | Cards, navigation, default |
| lg | 48px | w-12 h-12 | Profile headers, detail views |
| xl | 64px | w-16 h-16 | Profile pages, hero sections |

**Fallback:** when no image is available, show initials (first letter of first name + first letter of last name) on `primary-100` background with `primary-700` text. Font size scales with avatar size.

**Selection state:** `ring-2 ring-primary-500 ring-offset-2`.

**Group display:** overlapping avatars with `-ml-2` offset, `ring-2 ring-white` on each to create separation.

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

**Badge styling:** `px-2 py-0.5 text-xs font-medium rounded-full`, optional leading dot (`w-1.5 h-1.5 rounded-full`).

### Toast Notifications

- Position: top-right corner, 16px from edges
- Max width: 400px
- Stack: newest on top, max 3 visible, older ones dismissed

**Structure:** white bg, left border 4px in semantic color, shadow-lg, radius-lg.

| Type | Left border | Icon color | Duration |
|------|-------------|------------|----------|
| Success | green-500 | green-500 | 3s |
| Error | red-500 | red-500 | 5s (requires manual dismiss for critical) |
| Warning | amber-500 | amber-500 | 4s |
| Info | blue-500 | blue-500 | 4s |

**Animation:** slide in from right (200ms), fade out (150ms). Dismiss on click or swipe right (mobile).

---

## 7. Interaction Patterns

| Interaction | Web | Mobile |
|-------------|-----|--------|
| Hover | opacity-90 or bg darken 10% | N/A (no hover on touch devices) |
| Focus | ring-2 ring-primary-500 ring-offset-2 | N/A (use active state) |
| Active/Press | scale-[0.98] | scale-[0.97] + haptic feedback (light impact) |
| Transition duration | 150ms ease-out | 200ms ease-out |
| Loading (page) | Skeleton matching content shape | Skeleton + shimmer effect |
| Loading (button) | Spinner replacing content, same size | Same |
| Empty state | Illustration + message + CTA button | Same |
| Pull to refresh | N/A | CircularProgressIndicator in primary-500 |
| Error state | Inline message + retry button | Same |

### Skeleton Loading

- Match the exact shape of the content being loaded (card skeleton looks like a card)
- Use `bg-slate-200` with a subtle shimmer animation (`animate-pulse` or custom shimmer)
- Never use a full-page spinner for content loading
- Spinners are acceptable only for button loading states and inline actions
- Show skeleton immediately (0ms delay), transition to content with a fade (150ms)

---

## 8. Layout Patterns

| Pattern | Mobile (<640px) | Tablet (640-1024px) | Desktop (>1024px) |
|---------|-----------------|---------------------|---------------------|
| Page padding | 16px | 24px | 32-48px |
| Content max-width | 100% | 100% | 1280px (max-w-7xl) |
| Card grid | 1 column | 2 columns | 3-4 columns |
| Sidebar | Hidden (hamburger/drawer) | Collapsed (72px icons) | Expanded (260px) |
| Bottom nav | Yes (mobile app only) | No | No |
| Card gap | 12px | 16px | 24px |
| Section spacing | 32px | 40px | 48px |
| Table | Card list (stacked) | Horizontal scroll | Full table |

### Page Layout

```
Desktop:
+--sidebar(260px)--+--------content(max 1280px, centered)--------+
|                  |  breadcrumb                                  |
|  logo            |  page-title              action-buttons      |
|  nav-items       |  ------------------------------------------ |
|                  |  content area                                |
|                  |                                              |
+------------------+----------------------------------------------+

Mobile:
+--full-width--+
|  header bar  |
|  page-title  |
|  content     |
|              |
|  bottom-nav  |
+--------------+
```

### Content width constraints

| Content type | Max width | Reason |
|-------------|-----------|--------|
| Prose/text | 65ch (~700px) | Optimal reading line length |
| Forms | 640px (max-w-lg) | Inputs should not stretch too wide |
| Dashboards | 1280px (max-w-7xl) | Full use of screen |
| Settings | 768px (max-w-3xl) | Focused, form-heavy pages |

---

## 9. Platform Specificities

### Web only

- Hover states on all interactive elements (buttons, cards, links, table rows)
- Keyboard focus indicators on every focusable element (WCAG 2.1 AA)
- `cursor-pointer` on all clickable elements
- Tooltips on icon-only buttons (delay: 500ms)
- Breadcrumbs on all nested pages
- Right-click context menus where appropriate (table rows)
- Cmd/Ctrl+K command palette for power users (future)

### Mobile only

- Touch targets: minimum 48x48px (even if the visual element is smaller, the hit area must be 48px)
- Bottom sheet instead of modals for secondary actions (up to 3 options)
- Swipe gestures on list items (archive, delete) with colored background reveal
- Pull to refresh on all scrollable list screens
- Safe area padding: respect notch (top) and home indicator (bottom)
- Haptic feedback: light impact on primary button press, medium on destructive actions
- Splash screen: primary-500 background, white logo centered
- Offline state: banner at top "You are offline", gray overlay on actions, cached data still visible
- iOS: native-feeling navigation transitions (slide from right)
- Android: Material-style transitions (fade through)

### Admin panel

- Denser information display: body-small (14px) as default text size
- Wider tables: horizontal scroll on smaller screens
- Batch actions: checkbox column on tables, floating action bar
- Keyboard shortcuts for common actions (documented in a help modal)

---

## 10. Do / Don't Rules

### DO

- Use the spacing scale (multiples of 4px) for ALL spacing values
- Use semantic colors (success/warning/destructive/info) for status indicators
- Use skeleton loading matching the content shape for all async content
- Use consistent padding within cards (always p-6)
- Use role-specific colors for badges (agency=blue, enterprise=purple, provider=rose)
- Use transitions (150ms ease-out) on all interactive state changes
- Use `ring-2 ring-primary-500 ring-offset-2` for focus indicators
- Pair destructive actions with a confirmation dialog
- Truncate long text with ellipsis and provide a way to see the full text (tooltip or expand)
- Use empty state illustrations with a clear message and actionable CTA

### DON'T

- Never use more than 2 font weights on the same screen
- Never use inline styles — always Tailwind classes or CSS tokens
- Never use hardcoded color hex values — always use tokens
- Never create custom shadows — use the 3 defined levels (sm, md, lg)
- Never use a full-page spinner for content loading — always skeleton
- Never mix rounded corners within a component (parent lg, children lg or smaller)
- Never use pure black (#000000) for text — use foreground (slate-900)
- Never put primary-colored text on primary-colored background
- Never use disabled buttons without explaining why (tooltip or helper text)
- Never auto-dismiss error toasts — user must acknowledge or it stays 5s minimum
- Never use alerts/confirms from the browser — always custom modals
- Never use horizontal scroll for primary content (tables are the exception)
- Never truncate critical information (prices, dates, statuses) — only descriptions

---

## 11. Animation and Motion

| Animation | Duration | Easing | Usage |
|-----------|----------|--------|-------|
| Button state | 150ms | ease-out | Hover, active, focus transitions |
| Card hover | 150ms | ease-out | Shadow and border transitions |
| Modal enter | 200ms | ease-out | Scale from 0.95 + fade in |
| Modal exit | 150ms | ease-in | Scale to 0.95 + fade out |
| Toast enter | 200ms | ease-out | Slide from right + fade in |
| Toast exit | 150ms | ease-in | Fade out |
| Sidebar toggle | 200ms | ease-in-out | Width transition |
| Skeleton shimmer | 1.5s | linear (infinite) | Left-to-right gradient sweep |
| Page transition | 200ms | ease-out | Fade (web), slide (mobile) |

### Motion rules

- Respect `prefers-reduced-motion`: disable all animations except opacity fades
- Never animate layout properties (width, height, top, left) — use transform and opacity
- Keep all animations under 300ms — users should never wait for an animation
- Loading states appear instantly (0ms delay), never after a timer

---

## 12. Iconography

- Icon set: **Lucide** (web and admin), **Lucide equivalent or Material** (Flutter)
- Default size: 18px for inline, 20px for buttons, 24px for standalone
- Stroke width: 1.5px (default Lucide)
- Color: inherit from parent text color (currentColor)
- Interactive icons: wrapped in a ghost button with proper hit area and aria-label

### Common icons

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

---

## 13. Responsive Breakpoints

| Name | Min width | Tailwind prefix | Typical devices |
|------|-----------|-----------------|-----------------|
| Mobile | 0px | (default) | Phones |
| sm | 640px | sm: | Large phones, small tablets |
| md | 768px | md: | Tablets portrait |
| lg | 1024px | lg: | Tablets landscape, small laptops |
| xl | 1280px | xl: | Laptops, desktops |
| 2xl | 1536px | 2xl: | Large desktops |

Design mobile-first. Use `min-width` breakpoints (Tailwind default). Never use `max-width` unless absolutely necessary.
