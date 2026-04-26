# Habit Tracker — Frontend Specification

> **STATUS: CLOSED — Shipped 2026-04-26**
> 5 pages, newspaper aesthetic (paper/ink/terracotta), optimistic ticking, celebration overlay, all forms. Three intentional omissions: instructions micro-strip, named footer signatures, and portrait stacked view (horizontal-scroll fallback used instead).
> See [CLOSURE.md](./CLOSURE.md) for full delivery notes.

> Date: 2026-04-26
> Idea: docs/plan/habit-tracker/idea.md

## Required Skills

> The dev agent MUST `Read`/activate every skill below BEFORE writing code.

- **`inertia-react-development`** (Skill) — Inertia v3 React patterns: pages, `<Link>`, `<Form>`, `useForm`, `useHttp`, layouts, deferred props, optimistic updates.
- **`wayfinder-development`** (Skill) — typed `@/actions/*` and `@/routes/*` imports; never hardcode URLs.
- **`tailwindcss-development`** (Skill) — Tailwind v4 utility classes, responsive grid layouts, dark/light variants (only paper-light here), arbitrary values, custom CSS variable integration.
- **`ui-ux-pro-max:ui-ux-pro-max`** (Skill) — for typography pairing (Noto Serif SC + Fraunces + Inter), color systems, layout, and tablet ergonomics in landscape + portrait.
- **`fe-feedbackloop`** (Skill) — every user manipulation needs visual feedback (success/failure with message).
- **`CLAUDE.md`** project file — `=== inertia-react/core rules ===`, `=== inertia-laravel/core rules ===`, `=== wayfinder/core rules ===`.
- **Reference document:** `Users/el/Desktop/family habit tracker/Family Habit Tracker.html` — the visual language source. Lift the CSS variables, fonts, masthead structure, grid markup, weekend / Sunday / month-start markers, and reward column layout. Adapt for live tap interaction and React.
- **`spec-pl.md`** — for the Inertia props shapes and route names.

> This project is new and has no `docs/dev/*` or `docs/integration/*` skills yet. Defer to the Skill-tool skills above plus CLAUDE.md and the reference HTML.

## Problem Statement

Build the tablet UX that mirrors the printed family-habit-tracker but adds live state: tap-to-tick, live points balance, progress bar to the active reward, celebration on goal-hit, easy reconfiguration at the family meeting, and visibility into history + lifetime totals. The aesthetic must feel like the print piece — paper background, terracotta accents, Noto Serif SC + Fraunces italic — not like a generic SaaS dashboard.

## Objective

Render four pages (Tracker / History / Totals / Settings) plus dialogs (celebration overlay, end-sprint confirmation) with a single shared layout that includes the masthead, top-nav, and footer.

## Scope

### In Scope

- **Root layout** with: masthead (sprint dates, family name, top-nav), footer (small print). Newspaper aesthetic.
- **Tracker page** — the home grid, with progress strip above the grid.
- **History index + History show** pages.
- **Totals page** — per-person lifetime cards.
- **Settings page** — tabs for People, Current Sprint (or New Sprint form when no active sprint), Rewards.
- **Forms** for: add/edit person, start new sprint, add/edit/remove participant, add/edit/remove habit, add/edit/remove reward, swap active reward.
- **Celebration overlay** — full-screen, dismissable; offers "Pick next reward" → settings, or "Maybe later".
- **End-sprint confirmation dialog**.
- **Empty states** for: no active sprint, no people, no history, no rewards.
- **Responsive layout** — landscape full grid; portrait collapses to per-person stacked view (today's columns + week navigator).
- **Weekend, Sunday, month-start visual markers** lifted from print HTML.
- **i18n / language**: pure freeform — single `name` field per entity. UI labels are English (with optional Chinese subtitles in the masthead per the print).

### Out of Scope

- Authentication UI (no login, no profile).
- Push notifications / sounds.
- Animations beyond simple cell-tick transition and overlay fade-in.
- A wishlist / multiple-active-rewards UX (one active reward only).
- Dark mode.
- Mobile-phone-specific layout (tablet-first; phone is a tolerated fallback through portrait stacked).

## Pages

### Tracker — `tracker/Show`
- **Route name:** `tracker.show`
- **Container:** Full bleed for the grid; outer page max-width = `1680px` matching the print's `.page` width on landscape; centered with paper background.
- **Purpose:** Daily check-in surface. Anyone walks up and ticks their habits.
- **Elements:**
  - **Masthead** (top): "The Family Ledger · No. {sprint sequence}" eyebrow, big title `家庭习惯表 · A Family Habit Tracker`, sprint dates ("Begins / Ends"), top-nav links (Tracker · History · Totals · Settings).
  - **Progress strip** (between masthead and grid): horizontal scroll-free row of cards, one per participant. Each card shows person name (large serif), current points balance / cost as fraction (e.g., "47 / 250"), reward name in Fraunces italic, progress bar (terracotta fill on light track), "Pick reward" link if no active reward.
  - **Instructions micro-strip** (small, optional, taken from print): "How it works · Legend · Total ticks possible".
  - **Grid table** matching the print column structure: Person · Habit · {N day cells} · Reward summary cell. Row groups per person (rowspan on the person cell). Habit row contains habit name + the day cells.
  - **Day cells:** weekend gets `weekend` background; Sundays get the small terracotta dot top-left; first-of-month gets a thicker left border. Today's column has a subtle terracotta column highlight.
  - **Tick interaction:** tapping a (habit, day) cell where `day ≤ today` and sprint active toggles the tick (an Inertia POST to `ticks.toggle`). Optimistic UI: tick appears immediately; on server confirmation it stays; on error it reverts and a toast shows the error.
  - **Future cells:** rendered with a subdued background and `pointer-events: none`.
  - **Reward summary cell** (rightmost, rowspan = participant's habit count): shows the participant's current reward name + cost + balance + small progress visual, like an inline mini-card.
- **User Actions:**
  - Tap a cell → toggle tick.
  - Tap "Pick reward" link → navigate to Settings → Rewards section anchored to that person.
  - Tap top-nav links to navigate.
- **Feedback:**
  - Success: tick visually flips state + small terracotta check appears with a 150ms scale-in.
  - Error: cell reverts, toast shows server error message at the top of the page.
  - Achievement: when `flash.achievement` is set in props, open the celebration overlay.
- **Empty / loading / error states:**
  - **No active sprint:** masthead shows generic title (no sprint dates); progress strip and grid replaced with a centered "Start your first sprint" card with a CTA button → Settings → New Sprint form.
  - **Loading:** Inertia handles by default; for tick toggles, optimistic UI means no spinner needed.
  - **Error (network down):** toast at top with retry hint.

### History — `history/Index`
- **Route name:** `history.index`
- **Container:** `max-w-5xl` centered.
- **Purpose:** Browse archived sprints.
- **Elements:** Vertical list of sprint cards, most recent first. Each card: date range (large serif), participants count, count of rewards achieved that sprint, "View" link → `history.show`.
- **Empty state:** "No archived sprints yet. Your first sprint is in progress." with a Tracker link.

### History Show — `history/Show`
- **Route name:** `history.show`
- **Container:** Same as Tracker (full bleed, 1680px landscape).
- **Purpose:** Read-only archived grid.
- **Elements:** Same as Tracker but:
  - Masthead shows "Archived" stamp.
  - Day cells have `cursor: default`; tapping does nothing (no toggle handler attached).
  - No celebration overlay possible.
  - "Back to history" link in masthead.

### Totals — `totals/Show`
- **Route name:** `totals.show`
- **Container:** `max-w-5xl` centered.
- **Purpose:** Lifetime stats per family member.
- **Elements:** Grid of cards, one per non-trashed person (in display order). Each card:
  - Person name (large serif).
  - Three stat blocks: Total points · Rewards earned · Sprints participated. Big numerals in Fraunces, small label in Inter caps.
  - Optional: last reward achieved as a small line at the bottom.
- **Empty state:** "No data yet — first ticks coming soon."

### Settings — `settings/Index`
- **Route name:** `settings.index`
- **Container:** `max-w-5xl` centered.
- **Purpose:** Manage people, current sprint, rewards.
- **Elements:** Tabbed interface (3 tabs):
  1. **People tab** — list of people with inline edit (name, display order). "Add person" inline form. Soft-delete button per row (with confirm).
  2. **Current Sprint tab:**
     - **If no active sprint:** "Start a new sprint" form (see Forms below) with `next_sprint_defaults` pre-filled.
     - **If active sprint:** sprint dates header (read-only — dates can't be changed once active), participant manager (add person from `available_for_sprint`, swap their active reward, add/edit/soft-delete habits inline), and a prominent "End sprint" button at the bottom (opens the End-Sprint confirmation dialog).
  3. **Rewards tab** — per person, list their rewards (active = highlighted, achieved = with `achieved_at` timestamp). Inline form to add a new reward (only when person has no active one). Inline edit for active rewards. Soft-delete button (only when not achieved and not in active sprint).

## Forms

### Form: `AddPerson`
- Fields:
  - `name` (text input, max 50): label "Name", validation message "Name is required (max 50 chars)".
  - `display_order` (optional, hidden — server defaults).
- Submit: POST → `people.store`.
- Disabled states: while submitting.

### Form: `EditPerson`
- Fields: same as `AddPerson`, inline.
- Submit: PATCH → `people.update`.

### Form: `StartNewSprint`
- Fields:
  - `start_date` (date picker, default = today): label "Start date".
  - `end_date` (date picker, default = today + 28 days): label "End date", validation "must be on or after start date".
  - `participants` (dynamic list — one block per family member, each toggleable in/out of the sprint):
    - `person_id` (hidden)
    - `included` (checkbox, default true if person was in last sprint)
    - `habits` (dynamic list of `{name, display_order}`): label "Habits"; "Add habit" button below; remove button next to each.
    - `reward.mode` (radio: `keep_current` | `new` | `none`):
      - `keep_current` only available if person has an existing active reward.
      - `new`: shows `reward.name` (text, max 80) + `reward.cost` (number, 1–9999).
      - `none`: ticks accrue as pending surplus.
- Submit: POST → `sprints.store`.
- Disabled states: while submitting; "Start sprint" disabled if no participants are checked.

### Form: `AddHabit` (mid-sprint)
- Fields: `name` (text, max 80).
- Submit: POST → `habits.store` with `sprintParticipant` route param.

### Form: `EditHabit`
- Fields: `name`, `display_order`.
- Submit: PATCH → `habits.update`.

### Form: `AddReward`
- Fields: `name` (text, max 80), `cost` (number, 1–9999).
- Submit: POST → `rewards.store` with `person` route param.
- Server-side rejected if person already has active reward → show field-level error: "This person already has an active reward. Edit or finish that one first."

### Form: `EditReward`
- Fields: `name`, `cost` (cost disabled if reward is achieved).
- Submit: PATCH → `rewards.update`.

### Form: `SwapActiveReward`
- A select on the participant settings: list of person's non-achieved rewards. On change → POST → `sprint-participants.update-reward`.

### Form: `AddParticipantMidSprint`
- Choose a person from `available_for_sprint`, optionally seed habits + reward.
- Submit: POST → `sprint-participants.store`.

### Confirmation: `EndSprint`
- Modal. Body: "End the sprint dated {start} – {end}? Sprint becomes read-only. Unspent points will carry to your next sprint." Buttons: "End sprint" (terracotta) → POST `sprints.end`; "Cancel".

### Confirmation: `DeletePerson` / `DeleteHabit` / `DeleteReward`
- Modal with reasoned warning text and confirmation button.

## Display / Copy Rules

- **Language:** English UI labels with Chinese subtitle in masthead matching the print. Free-form name fields support any language without transformation.
- **Date format:** `Sun · Apr 19, 2026` for masthead labels; `Apr 19` for day-column headers (DOW + day-of-month, like print).
- **Money / points:** Points are unitless integers; format as `47 / 250` for fractions, `+13` for surplus carryover. Use the Fraunces serif for the numeric balance, Inter for the slash and labels.
- **Name format:** Render person/habit/reward names as-is (no transformation, no truncation under 30 chars; truncate-with-ellipsis above).
- **Color:**
  - Paper background `#fbf8f2`, secondary paper `#f3efe6`, weekend tint `#f2efe7`.
  - Ink `#111`, ink-2 `#3a3a3a`, ink-3 `#6b6b6b`.
  - Terracotta accent `#c94a2b`, soft accent `#f2dcd2`.
  - Hairline `#cdc9c3`, lighter hairline `#e8e4dc`.
- **Typography:**
  - Display: `Noto Serif SC` 600 (Chinese & big serif moments).
  - Italic accent: `Fraunces` 400 italic (sub-titles, numerals, small labels).
  - Body / nav / labels: `Inter` 500 (caps, letter-spaced).
  - Body Chinese: `Noto Sans SC` 400.
- **Tick check:** terracotta `✓` rendered as a CSS pseudo-element (rotated border like print's `.box.tick::after`).
- **Day cell minimum tap target:** 44×44 px on landscape (matches the print's 36px col scaled up for finger taps). Portrait can be larger.

## Integration Contract (with Platform)

### Receives from Platform

- All Inertia prop shapes documented in `spec-pl.md`. Frontend should treat these as the canonical source and not synthesize state from elsewhere.
- `flash.success` / `flash.error` / `flash.achievement` from shared Inertia props.

### Sends to Platform

- All form submissions go through Wayfinder typed actions (e.g., `import { ticks } from '@/actions'` then `ticks.toggle.form()`). Never hardcode URLs.
- Tick toggle uses `router.post(...)` with `preserveScroll: true`, `preserveState: true`, and `only: ['participants', 'flash']` to minimize re-renders.

## Infrastructure Integration

This is a brand-new project with no integration skills set up yet. Apply only what's needed:

- [ ] **i18n:** **Not applicable.** All UI labels are English; entity names (person/habit/reward) are freeform user input rendered as-is. No translation layer.
- [ ] **Money:** Not applicable.
- [ ] **Realtime:** Not applicable.
- [ ] **PDF:** Not applicable in v1 (a print stylesheet for archived sprints is a `[DEFERRED]` item — leave a `@media print` stub if convenient).
- [ ] **Excel:** Not applicable.
- [ ] **Tailwind v4 custom theme:** Configure CSS variables for paper, ink, accent (above) in the root layout's `<style>` or via Tailwind's `@theme` block.
- [ ] **Google Fonts:** `<link>` to `Noto Serif SC`, `Fraunces`, `Inter`, `Noto Sans SC` in the root document head (mirror the print HTML's `<link>` tags exactly).
- [ ] **Permission gating on UI entry points:** **Not applicable.** No auth in this app, so every menu item, button, and link is visible to anyone using the tablet. The standard "permission-gate every link" rule does not apply because there are no permissions.

## Acceptance Criteria

- [ ] Four pages render at the four routes (`tracker.show`, `history.index`, `history.show`, `totals.show`, `settings.index` — five in total counting history-show).
- [ ] Tracker grid visually matches the reference HTML at landscape 1680px width. Weekend tint, Sunday dot, month-start divider all present.
- [ ] Tap-to-tick works with optimistic UI: cell flips immediately, server response confirms, error reverts.
- [ ] Future-day cells are visually disabled and not tappable.
- [ ] Past-day cells in the active sprint are tappable to backfill.
- [ ] Archived sprint grids are read-only (no toggle handler attached).
- [ ] Progress strip above the grid shows each participant's points balance / reward cost as a fraction with a terracotta progress bar; updates immediately after a tick.
- [ ] Celebration overlay opens automatically when `flash.achievement` is set; offers "Pick next reward" (navigates to Settings → Rewards anchored to that person) and "Maybe later" (dismisses).
- [ ] Top-nav navigates between Tracker / History / Totals / Settings via Inertia (no full page reloads).
- [ ] Settings page lets the family: add/edit/soft-delete people; start a sprint when none active (with `next_sprint_defaults` pre-fill); end the active sprint via confirmation dialog; add/edit/remove participants, habits, rewards mid-sprint.
- [ ] Empty states present for: no active sprint (Tracker), no people (Settings), no archived sprints (History), no people for totals.
- [ ] Layout is responsive: full grid in landscape, stacked per-person view in portrait.
- [ ] All form submissions and tick toggles use Wayfinder typed routes/actions (no hardcoded URLs anywhere).
- [ ] All form submissions provide visible feedback (toast or inline) on both success and error per `fe-feedbackloop`.
- [ ] Pint (PHP) and Prettier/ESLint (JS/TS) clean.
