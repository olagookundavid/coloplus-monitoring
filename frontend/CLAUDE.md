# Frontend guide

Standing rules for all code under `frontend/`. Follow them without being asked. Add each new rule the owner gives here, and keep this file succinct.

**Scope:** when you edit a file, apply every rule to the whole file (text icons, native `confirm()`/`alert()`, responsiveness gaps, hand-written tables, dashboard headers, first-time usability, §7 controls, §11-§13 trust and provenance rules), and to the components a page renders for responsiveness. Don't edit other files just for these rules.

**Two portals, one app.** `frontend/` is a single React app with two permission-gated, lazily-loaded route groups: `pages/customer/` (Airtel viewer and reviewer — the read-mostly monitoring portal) and `pages/ops/` (Coloplus administrator, field officer, operations supervisor, model administrator — entry, approval and formula management, PRD §4). Both share the same `DataTable`, `ConfirmDialog`, dashboard chrome, domain types and API base. `usePermissions()` (architecture standard §4) decides what a signed-in user sees and can do; never a separate build, and never a client-side-only check. See DECISIONS 024.

## 1. No text icons

Never use emoji, Unicode symbols or text as icons (`✅ 🔍 ⚡ ⭐ ✕ ✓ ← → ▲ ▼ ●`, an `x` or `>` for an icon, `...` as a loading indicator) in JSX, labels, headings, `toast()` or dialog strings. Currency signs, the `—` empty-cell placeholder and punctuation are typography and stay. `—` has one meaning here: *not reported*. Never replace it with a fabricated `0`, a repeated last-known value, or any other inferred number (§11).

1. **Heroicons** (`@heroicons/react`): `24/outline`, or `24/solid` for filled or active states. Check `node_modules/@heroicons/react/24/outline` before deciding one doesn't exist.
2. **Custom SVG**, only when no heroicon depicts the exact thing (a site mast, a fuel drop), never a loosely related one. Outline style: `viewBox="0 0 24 24"`, `fill="none"`, `stroke="currentColor"`, `strokeWidth={1.5}`, round caps and joins, sized with `className`. Used in more than one place? Make it a component in `src/components/icons/`.

- Size `w-4 h-4` beside text, `w-5 h-5` in headings and larger controls. Align with `inline-flex items-center gap-1.5`.
- Decorative icons get `aria-hidden="true"`. Icon-only buttons use `IconButton` (§7), which requires a `label`.
- Loading: `ArrowPathIcon` with `animate-spin`, or `<Spinner />` (`src/components/ui/`).

## 2. Confirm key writes, and logout

Key or destructive actions — approve, publish to Airtel, activate a formula version, correct or supersede an approved record, reopen a closed reporting period, deactivate a site or user, assign or resolve an incident, raise a dispute, revoke access — and logout run only after a confirm dialog. Plain saves/updates, and reading, filtering, sorting, paging, switching tabs, opening a form and marking notifications read, don't.

- `ConfirmDialog` (`src/components/ui/ConfirmDialog.tsx`) props: `open`, `title`, `description?`, `body?`, `confirmLabel`, `tone`, `icon` (a heroicon), `pending`, `maxWidth?` (`'md'` unless the body needs room), `dontAskAgainKey?`, `dontAskAgainLabel?`, `onConfirm`, `onCancel`. Name the affected record with a `ConfirmSummary` (`src/components/ui/ConfirmSummary.tsx`) in `body` — the site, period or record, never just "this item."
- `dontAskAgainKey` (a stable, page+operation-scoped string like `"ops-incidents:assign"`) offers a "don't ask me again" checkbox; once checked and confirmed, later opens for that key skip straight to `onConfirm`. Opt-in per call site, and never offered when `tone="danger"` — enforced inside the component itself, so a correction or any other irreversible action can never be silenced even by mistake.
- Tone: `success` for positive completions (approve, publish), `info` for neutral changes, `warning` for reopen/revoke/deactivate, `danger` for anything that discards or overwrites a record — red, so it always reads as irreversible at a glance.
- Logout: title "Log out?", `confirmLabel` "Log out", `tone="info"`, `icon={ArrowRightOnRectangleIcon}`.
- Never use `confirm()`, `window.confirm()` or `alert()`. Report results with the shared `toast` helper (`src/utils/toast.ts` + `src/components/ui/Toast.tsx`/`ToastViewport.tsx`, mounted once in `main.tsx`) — a custom component, not `react-hot-toast` (DECISIONS 025).
- Pattern:
  1. The button stores the pending action (`setPendingAction({ kind, item })`) instead of calling the handler.
  2. One `<ConfirmDialog>` per page reads that state. Its title says what will happen, and its body names the record.
  3. `onConfirm` runs the handler with `pending` true while the request is in flight, then the dialog closes. `onCancel` clears the state.

## 3. Responsive at every size

Every page you change or build works from a 320px phone to a wide monitor. This isn't just polish: the daily entry and incident-reporting screens in `pages/ops/` are mobile-first by requirement, not default politeness — field officers fill them in on site, on a phone (PRD §5, §6).

- **Mobile first.** Base classes are for the smallest screen. Add `sm:` (640px), `md:` (768px), `lg:` (1024px), `xl:` (1280px), `2xl:` (1536px). Don't write desktop styles and undo them with `max-*:`.
- **No horizontal page scroll.**
  - No fixed pixel widths on containers. Use `w-full` with `max-w-*`, percentages, grid or flex.
  - Give flex and grid children whose text must shrink `min-w-0`.
  - Use `truncate`, `break-words` or `line-clamp-*` on long values, and `max-w-full` on media.
- **Reflow.** Grids collapse (`grid-cols-1 sm:grid-cols-2 lg:grid-cols-3`). Control rows wrap (`flex-wrap`) or stack (`flex-col sm:flex-row`). Spacing and type scale up with the breakpoint.
- **Tables:** see §4. Don't hide a column unless its information is offered another way.
- **Touch targets** are at least 44px (`min-h-11 min-w-11`, or padding), with room between them. Nothing that matters depends on hover — a field officer on a phone must be able to do everything a desk reviewer can.
- **Modals fit the viewport.**
  - Height: `max-h-[calc(100dvh-2rem)]` (prefer `dvh` to `vh`), with an `overflow-y-auto` body.
  - Width: `w-full` up to `max-w-*`, with a margin on small screens.
  - Footer buttons stack or wrap (`Modal`'s `footer` does this).
- **Fixed chrome.**
  - Layout provides the page width, the sticky navbar and the mobile bottom nav. Don't add a page-level max-width.
  - Keep anything pinned to a screen edge clear of the bottom nav and the notch (`env(safe-area-inset-*)`).
- **Check** at about 320, 375, 768, 1024 and 1440px before calling it done: no sideways scroll, nothing overlapping or clipped, every control reachable.

## 4. Tables: DataTable

Every table uses `DataTable` (`src/components/ui/DataTable.tsx`, built on TanStack Table). Never hand-write a `<table>`.

- **Columns** are `Column<T>` (`header`, `accessor`, `cell`, `sortable`, `sortValue`, `align`, `className`). Pages don't use TanStack Table directly.
  - Define columns once with `useMemo`. When a page shows the same columns twice, share them.
  - Keep `data` stable too: a module-level `const NO_ROWS: Row[] = []`, not an inline `?? []`.
- **Server-paginated lists** pass `serverPage`, `totalServerPages` and `onPageChange`, plus `onSearchSubmit` for server search. A sortable header there only reorders the loaded rows. Local lists let `DataTable` search, sort and page in the browser.
- **Other props:** `loading`, `paginated={false}`, `getRowId`, `emptyMessage`, `searchable={false}`, `compact` (no 800px minimum width, for tables in dialogs or narrow cards), `headerActions` (toolbar filters and buttons), `rowClassName`, `onRowClick`, `stickyHeader` with `maxHeight`. A column can take `onSort`/`sortActive` for a sort the page controls, and `headerClassName` for tighter header padding.
- **The same table on every screen.** On phones it scrolls sideways inside its card. No separate mobile card layout.
- **The first column is frozen,** so it must name the row (site, ticket, incident). Never lead with a narrow column like `#` or a bare code; fold it into the name cell or put it second.
- **Row navigation vs. row actions.** A table whose signed-in role has no mutating action on a row (the Airtel viewer's or reviewer's site list, ticket list) ends in a plain trailing chevron, and the whole row is clickable to the record's detail page — never an empty or disabled `RowActions` menu standing in for "nothing to do here." Rows with real mutating actions end in an `Actions` column (`align: 'right'`) holding `<RowActions>` (`src/components/ui/RowActions.tsx`): a three-dot button that opens a dropdown of that row's actions, no inline action buttons, an action that can't run stays in the menu disabled with a `hint`. This is the rule wherever a table exposes mutating actions to the signed-in role (`pages/ops/`: assign incident, approve entry, correct a record, deactivate a site or user). The same `DataTable` and column set can serve both roles — which ending a given table gets follows from whether `usePermissions()` grants any action on that row, not a per-page stylistic choice.
- **Missing a feature?** Add it to `DataTable` as an optional prop instead of hand-writing a table. Ask first if the change is large.

## 5. Dashboards

Dashboard layouts (`pages/customer/`, `pages/ops/`) render `DashboardShell` (`src/components/dashboard/`) with a nav file per portal (`customerNav.ts`, `opsNav.ts`).

Every dashboard page starts with its own `DashboardPageHeader` (`src/components/dashboard/DashboardPageHeader.tsx`). Never hand-write an `<h1>`, banner or card header there. The look lives in that component only; change it there.

- **Eyebrow:** a small uppercase, letter-tracked label in the brand teal above the title (e.g. "PORTFOLIO INTELLIGENCE", "SITE OPERATIONS") — a `DashboardPageHeader` prop, Coloplus's own addition. The title itself stays normal case; only the eyebrow is uppercase.
- **Title:** the page's sidebar label in Title Case (`Sites`, `Reports`), so page and sidebar agree. Detail pages (a site, an incident) use their own title plus `back={{ to, label }}`.
- **Subtitle:** one sentence on every page saying what it's for. Pass a `ReactNode` for live values.
- **Actions:** primary buttons go in `actions`, on the right from 640px up.
- **Filters and recency, below the header.** Every page that shows reported data (not a pure settings or admin form) carries a second row: a region/period filter, persisted across navigation, and a data-recency indicator — a dot plus "as of [time] WAT" or "Sample snapshot · [time] WAT," matched to what the environment actually is. This isn't decoration: PRD §5 requires the dashboard header to show the selected period, Africa/Lagos time zone, publication cutoff and last refresh on every data screen.
- No italics, icon tiles or per-page sizes beyond the eyebrow above.

## 6. First-time users

Every UI change and new page must be easy for someone using the app for the first time. Put UX and UI first.

- **Say what things are.** Plain-word labels, a subtitle saying what the page is for, a heading on each section.
- **Show, don't hide.** Put what people need to decide (which site, what status, who owns it) on the page, not behind a dialog or an extra click.
- **Obvious next step.** One primary action per page or section. Links look like links, and a record's name opens its detail page. A status badge is never a secret button.
- **Helpful empty states.** Say what's missing, why it matters, and give a link or button to fix it.
- **Predictable.** Reuse the shared components and patterns above so every page behaves the same way, in both portals.
- **Check** by walking the page as a newcomer: can they tell where they are, what they can do, and how to get back?

## 7. Controls

Buttons, form fields and dialogs come from the shared primitives in `src/components/ui/`. Never style a raw `<button>`, `<input>`, `<select>`, `<textarea>` or `fixed inset-0` overlay with page classes. Look changes go in the primitive; layout goes in `className`.

- **Button:** `variant` (`primary | secondary | outline | danger | ghost | link | success | info | warning`), `size` (`sm | md | lg`), `icon` with `iconPosition`, `loading` (spinner, disabled), `fullWidth`. Defaults to `type="button"`. `ButtonLink` looks the same on a router link, for navigation only.
- **IconButton:** icon-only. `icon` and `label` (required; becomes the `aria-label` and tooltip). `variant` `ghost | secondary | danger`.
- **Field:** `label`, `htmlFor`, `hint`, `error`.
- **Checkbox:** `label` and `hint` make a labelled row, and the whole row is the tap target.
- **Tabs:** an underline tab bar. `items` (`value`, `label`, optional `icon`), `value`, `onChange`, `aria-label`.
- **Input, Textarea, Select:** native controls with the shared look. `invalid` gives an error border.
- **Modal:** `title`, `subtitle`, `footer` (Cancel first, primary last; stacked on phones), `maxWidth`. Omit `onClose` for a gate that can't be dismissed. A form goes in the body, and its submit button goes in the footer as `form="<id>"`. Render a `ConfirmDialog` outside the Modal, or its backdrop clicks close the Modal.
- **SidePanel:** a right-anchored slide-over for reading one record's detail (a ticket, a site) without leaving the list — `eyebrow`, `title`, `subtitle` (caller composes the meta row: a location line, status badges), `headerActions` (e.g. a future "Create ticket" next to the close button), `maxWidth`. Shares its focus-trap/Escape/scroll-lock behavior with `Modal` via `dialogBehavior.ts`, so a change to that behavior is made once, not per-dialog. Use `Modal` for a form or a confirmation; use `SidePanel` when the row's own detail page would otherwise be the destination (frontend/CLAUDE.md §4's "clickable to the record's detail page" row convention — the panel *is* that detail page, reachable by a real `getRowHref` link, not `onRowClick`).
- **Pointer cursor:** set globally in `src/index.css` for buttons, selects, checkboxes and radios. Disabled controls show `not-allowed`. Don't add `cursor-pointer` per element.
- **Corners are round by default,** matching the stakeholder mockup; no page currently needs the sharp-corner exception Showtime reserves for its store. If one ever does, it's a `shape="square"` prop on the primitive, recorded in `docs/DECISIONS.md`, not a page override.
- **On a dark surface** (a dark modal header, a dark hero): pass `tone="dark"` to any control on it.
- **Text in controls** is `text-base` on phones and `sm:text-sm` above, because iOS zooms into smaller text.
- **Brand tokens.** Components reference semantic Tailwind v4 theme tokens (`primary`, `primary-soft`, `warning`, `danger`, `surface`, …), never a raw hex. Tokens are seeded in `src/index.css`'s `@theme` block when the app is scaffolded, provisionally from the stakeholder mockup pending an official brand kit (DECISIONS 024): primary teal ≈ Tailwind `teal-600`/`teal-700` (soft background `teal-50`/`100`), warning = `amber-500`/`100`, danger = `red-600`/`50`, neutrals = `slate`. Airtel's own red (their logo asset) is a fixed co-brand mark in the shell only — never reused as a semantic token, so it's never mistaken for an app danger or warning color. When the real brand kit lands, only `@theme` changes; no component touches.

## 8. Types

Every domain/API type lives in `src/types/<domain>.ts`: `site`, `incident`, `measurement` (power source intervals, daily summaries), `fuel`, `formula`, `report`, `ticket`, `dispute`, `user`. Never declare one inline in a component or page, and never in `services/api.ts` — that file only imports types back from `src/types/` for its own function signatures. Add a new domain file when none fits; cross-domain references import from the other domain file.

**Stays local, not moved:** `*Props` interfaces; local UI-state unions whose only job is one component's `useState` (`PendingAction`, `FormState`, tab/filter state); client-computed view types derived from already-fetched data; hardcoded static content shapes.

## 9. Barrel imports (index.ts)

Every folder under `src/components/` (plus `utils/`, `hooks/`, `types/`, `contexts/`, `constants/`, and the eagerly-loaded part of `src/pages/`) re-exports its contents through that folder's `index.ts` — not just components, every colocated helper too (`opsNav.ts`, `statusStyles.ts`, `useSiteActions.ts`, ...). A new file isn't done until its barrel export is added in the same change.

- **Cross-folder imports go through the barrel:** `from '../components'`, not `from '../components/DataTable'`.
- **Same-folder sibling imports stay direct** — the rule is about crossing folder boundaries, not internal organization.
- **`pages/index.ts` covers only eagerly-loaded top-level pages.** `pages/customer/` and `pages/ops/` are lazily loaded via `lazy(() => import('./pages/...'))` for code-splitting and intentionally bypass it.

## 10. Sitemap

No `public/sitemap.xml`. Coloplus has no public, evergreen pages — both `pages/customer/` and `pages/ops/` sit behind auth, org- and site-scoped. The whole app is disallowed in `robots.txt`. Don't add a sitemap "for consistency" with other Fusewall products; this product has the opposite shape — nothing to index.

## 11. Data trust and status language

A site's displayed status reflects its most recent *reported* state, never an assumption. An overdue or missing report shows as "Update overdue" or "Awaiting update" — never the last-known "Available," and never silently re-colored green by the passage of time. Status is always text *and* icon or dot alongside color, never color or a bare dot alone. On trend charts, reported and estimated series use visibly distinct line styles (solid vs. dashed, say), never the same style distinguished only by a tooltip. A gap in reporting stays a visible gap on the chart; never interpolate, zero-fill or otherwise paper over a missing interval. This applies to every chart, table cell, card and export — not just the portfolio dashboard.

## 12. Provenance and measurement basis

Every derived metric (not a raw passthrough) carries two independent badges, never merged into one pill: a **basis** (`Measured | Calculated | Estimated | Unavailable`) and a **review state** (`Draft | Submitted | Approved | Superseded | Disputed`). Approving a record never upgrades its basis — an approved estimate is still `Estimated`, just no longer `Draft`. Every derived metric exposes a "How calculated" affordance (`src/components/ui/`) showing input record IDs and revisions, coverage period, units, formula ID and version, parameter set and effective dates, execution time, output precision, warnings and approver. Use shared `BasisBadge` / `ReviewStateBadge` primitives everywhere a figure appears — never a one-off colored span.

## 13. Exports and reports

Every export or report surface discloses its scope, data basis, missing-data convention and time standard once per surface (a panel or banner), not repeated per row or per file: reporting scope (site IDs, region, period), data context (units, basis, observation timestamps), how missing data reads (blank, never zero or "available"), and the time standard (`Africa/Lagos · WAT (UTC+1)`). Totals exclude unavailable values while stating the count and scope excluded. Provisional, approved and reconciled are separate views and exports, never flattened into one. A screenshot is never the only way to get data out.

## 14. Scope and permissions

Org and site scope is enforced on every page, query, attachment and export — a copied URL must never reveal another org's or another site's data. Never fetch cross-org data and filter it client-side "for convenience"; the query is scoped server-side, and the frontend doesn't additionally rely on CSS to hide fields a response shouldn't have included. Internal-only content (draft commentary, staff names, internal cost figures) never renders on a `pages/customer/` surface, regardless of what a shared component shows elsewhere. The backend remains the authority; `usePermissions()` is UI convenience, not the security boundary.
