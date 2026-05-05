# W-09 + M-09 — Création annonce

## Visual diff notes

### Web — `/[locale]/(app)/jobs/create`

Before (origin/main):
- Plain heading "Créer une offre" + grey ghost buttons (cancel/publish), generic `text-2xl font-bold tracking-tight`.
- Section accordions in white cards with grey rounded-2xl borders, rose-500 number badge when complete.
- Inputs use `bg-gray-50` with `focus:border-rose-500`. Currency € positioned on the LEFT of the input.
- Bottom of budget section duplicated the publish CTA inside the accordion.
- Applicant type selector: vertical stack of pill rows (rose-50 fill / rose-500 border on active).
- Credits modal: rose-100 icon chips on slate text inside a rounded-2xl shadow-xl card.

After (`feat/design-creation-annonce`):
- Editorial Soleil header: corail mono eyebrow `ATELIER · NOUVELLE ANNONCE` (uppercase, tracking 0.12em),
  Fraunces 30/42px display title `Publie ta *nouvelle annonce.*` with the second sentence in italic corail,
  tabac subtitle on a max-width 620px column. Back-arrow link `Toutes mes annonces` in font-mono micro-text.
- Two accordion cards on Soleil ivoire surface (`rounded-[20px]`, `border-border` -> `border-border-strong`
  when open, `shadow-[0_2px_12px_rgba(42,31,21,0.04)]`), section number rendered as a corail-soft pill with
  `font-mono` digit; complete = corail-filled circle with white check.
- Section title rendered in Fraunces 18px (titleMedium-equivalent).
- Inputs: rounded-2xl, `border-border-strong`, currency € on the RIGHT in serif at tabac, focus state =
  corail border + corail-soft ring.
- Applicant type selector: 3-column responsive grid, each option a rounded-2xl card with lucide icon (Users,
  User, Building2), corail-soft fill + corail border + radio dot when active, encre/sable otherwise. Mono
  uppercase header `QUI PEUT POSTULER ?`.
- Budget section: corail pill toggles for project type and payment frequency (active = corail bg + white
  text + soft glow), euro inputs use Geist Mono numerals, mono uppercase labels.
- Footer actions row: `Annuler` ghost (mute) on the left, `Publier l'annonce` corail rounded-full pill with
  `ArrowRight` on the right.
- Credits info modal: ivoire surface, corail mono eyebrow at the top, Fraunces title, 4 corail-soft icon
  chips (Ticket/RefreshCw/Trophy/TrendingUp), close = corail rounded-full pill at the bottom.

Behavioural diffs: NONE. Same react-hook-form-less local state, same `useCreateJob` mutation, same
validation rules, same agency role auto-coercion, same routing. The footer-of-section duplicate publish CTA
inside the budget accordion was removed because the global footer CTA already covers it (matches the maquette
A3 layout in `design/assets/sources/phase1/soleil-lotA.jsx`).

### Mobile — `lib/features/job/presentation/screens/create_job_screen.dart`

Before (origin/main):
- Plain Material `AppBar` titled "Créer une offre", FilledButton in app bar actions area (rose), bottom
  ElevatedButton + TextButton "Cancel".
- Accordion cards with rose-500 icon-on-soft-tint header and Material `SegmentedButton` for budget/applicant
  pickers.
- Skill chips on rose-tint with rose border; description-type segmented in standard Material 3.

After (`feat/design-creation-annonce`):
- Soleil AppBar with Fraunces title (`Nouvelle annonce` / `Modifier l'annonce`).
- Editorial hero scrolls inside the form: corail mono eyebrow, Fraunces displayM `Publie ta *nouvelle
  annonce.*` in italic corail, tabac subtitle.
- Accordion cards: ivoire surface, rounded `radius2xl`, corail-soft circular badge with mono digit.
- Applicant type picker: vertical stack of corail-bordered tiles with icon + label + radio dot, no Material
  SegmentedButton.
- Budget type picker: corail-filled pill toggle (rounded-full StadiumBorder).
- Description type picker (text/video/both): corail-filled pill toggle inside an ivoire section card.
- Skill chips: corail-soft fill + corail-deep label, deletable; "+" button is corail StadiumBorder.
- Sticky bottom action bar: ivoire surface with top-border, ghost `Annuler` outline pill on the left, corail
  `Publier l'annonce` filled pill (StadiumBorder) on the right with `arrow_forward` icon.

No `Color(0xFF...)` literals in new code — every color goes through `colorScheme` or `AppColors` extension.

## Screenshots

The agent runs in a headless container without an X display server and without a connected Android
emulator/device. Screenshots are deferred to the orchestrator's review pass on a workstation with
`flutter run` capability:

```
flutter run -d <device>
# Navigate to /jobs/create in the app
flutter screenshot --out=design/diffs/W-09-M-09/after-android.png
```

For the web side, run `npm run dev` in `/web` and capture
`http://localhost:3000/fr/jobs/create` (logged in as enterprise user).

## Source maquettes

- Web: `design/assets/sources/phase1/soleil-lotA.jsx` § `SoleilJobCreate` (lines 336–498).
- Mobile: `design/assets/sources/phase1/soleil-jobcreate-mobile.jsx` (lines 9–173).
