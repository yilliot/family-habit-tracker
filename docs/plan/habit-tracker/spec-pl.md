# Habit Tracker — Platform Specification

> Date: 2026-04-26
> Idea: docs/plan/habit-tracker/idea.md

## Required Skills

> The dev agent MUST `Read`/activate every skill below BEFORE writing code.

- **`laravel-best-practices`** (Skill) — controllers, Form Requests, route definitions, middleware, query patterns.
- **`wayfinder-development`** (Skill) — typed route exposure to React; `wayfinder:generate`; `@/actions` and `@/routes` imports for Frontend.
- **`inertia-react-development`** (Skill) — for the props-shape contract and how server-side renders pages for the Inertia client.
- **`CLAUDE.md`** project file — `=== inertia-laravel/core rules ===`, `=== wayfinder/core rules ===`, `=== laravel/core rules ===` blocks.
- **Laravel Boost MCP tools** — `search-docs`, `get-absolute-url`, route inspection.
- **`spec-be.md`** — for the available repository / mutation surface to call.

> This project is new and has no `docs/dev/*` or `docs/integration/*` skills yet. Defer to the Skill-tool skills above plus CLAUDE.md.

## Problem Statement

The data layer (Backend) provides queries and mutations. The user-facing tablet (Frontend) needs HTTP routes that:

1. Render each page with the right Inertia props.
2. Accept form submissions and tick toggles, validate them, and dispatch to Backend services.
3. Return appropriate flash messages and redirects so the React UI can present feedback.

There is **no authentication** in this app. The single shared tablet is the only client, so all routes are public to anyone on the local network. There is no permission gating, no feature flag, and no role-based access — those concepts do not apply to this project.

## Objective

Define the routes, controllers, request validation rules, and Inertia props shape that bind Backend services to the Frontend pages.

## Scope

### In Scope

- Inertia-rendering routes for: Tracker (home), History index, History show, Lifetime Totals, Settings (people, sprint config, rewards).
- Mutation routes: people CRUD, sprint start/end, participant management, habit CRUD, reward CRUD, tick toggle.
- Form Request classes for every mutation route, defining validation rules.
- Controllers that delegate query → repository, mutation → factory/service, then return `Inertia::render(...)` or `back()->with(...)` as appropriate.
- Wayfinder route registration (so Frontend imports stay typed).
- Top-level layout component wiring (the same root layout serves all 4 pages — Frontend defines the layout itself; Platform just renders the page component name).

### Out of Scope

- Auth (Fortify routes are not registered for end users; if scaffolding pages exist they remain inactive).
- Permissions / authorization / feature flags / roles — non-existent in this app.
- API resources / API tokens / Sanctum.
- WebSockets / Echo / realtime broadcasting.
- Job dispatching.
- Notifications.

## Endpoints

> All routes return Inertia responses (HTML on first load, JSON on subsequent visits). All mutation routes follow the back-redirect-with-flash convention. All routes are unauthenticated and ungated.

### `GET /` — Tracker (home)
- **Route name:** `tracker.show`
- **Purpose:** Render the active sprint's grid (or an empty state if none).
- **Authorization:** None.
- **Request Input:** None.
- **Response Output:** Inertia page `tracker/Show` with props:
  - `sprint`: null | `{ id, start_date, end_date, days: [{date, dow, is_weekend, is_month_start, is_today, month_label}], status }`
  - `participants`: array of `{ id, person: {id, name, display_order}, balance, carry_forward_balance, active_reward: null | {id, name, cost, achieved_at}, habits: [{ id, name, display_order, ticks: [tick_date, ...] }] }`
  - `today`: ISO date string (server's current date)
  - `flash`: `{ success?, error?, achievement?: { person_id, reward_id, person_name, reward_name } }` — `achievement` is set when the most recent tick toggle pushed someone over their reward; Frontend uses it to open the celebration overlay.
- **Behavior:**
  1. Call `getActiveSprint()`.
  2. If null → render with `sprint = null`, `participants = []`. (Frontend shows "Start your first sprint" empty state.)
  3. Else call `getActiveSprintGrid()` and shape into the props above.
  4. Forward any `achievement` flash from session to the page.
- **Error Cases:** None — always 200.

### `POST /ticks` — Toggle a tick
- **Route name:** `ticks.toggle`
- **Purpose:** Toggle the existence of a tick for one (habit, day).
- **Authorization:** None.
- **Request Input:**
  - `habit_id` (int, required) — must reference a non-trashed habit on the active sprint.
  - `tick_date` (date, required) — must be ≤ today (server-clock) and within active sprint's date range.
- **Response Output:** Redirect back to `tracker.show` with flash:
  - On success: `flash.success = "Tick updated"`. If `just_achieved_reward` is non-null, additionally set `flash.achievement = { person_id, reward_id, person_name, reward_name }`.
- **Behavior:**
  1. Resolve Habit by id; reject 404 if missing.
  2. Validate (sprint active, date in range, date ≤ today).
  3. Call `toggleTick(habit, tick_date)`.
  4. Set flash. Redirect back.
- **Error Cases:**
  - Habit on archived sprint → 422 with message "This sprint is archived."
  - tick_date in future → 422 with message "Cannot tick a future day."
  - tick_date outside sprint range → 422 "Date is outside the active sprint."
  - Habit soft-deleted → 422 "This habit was removed."

### `POST /sprints` — Start a new sprint
- **Route name:** `sprints.store`
- **Purpose:** Begin a new sprint after a family meeting.
- **Authorization:** None.
- **Request Input:**
  - `start_date` (date, required) — any date.
  - `end_date` (date, required, after_or_equal `start_date`).
  - `participants` (array, required, min 1) — each:
    - `person_id` (int, required, must exist in non-trashed people)
    - `habits` (array, required, min 0) — each: `name` (string 1–80), `display_order` (int, optional)
    - `reward` (object, nullable):
      - `mode` (string enum: `keep_current` | `new` | `none`, required)
      - `name` (string 1–80, required if mode = `new`)
      - `cost` (int 1–9999, required if mode = `new`)
- **Response Output:** Redirect to `tracker.show` with `flash.success = "Sprint started"`.
- **Behavior:**
  1. Validate via Form Request.
  2. Reject (422) if any active sprint exists.
  3. Call `startSprint(...)` factory which: creates Sprint, creates SprintParticipants (with carry-forward seed), creates Habits, creates Rewards (if `mode=new`) or links existing active reward (`mode=keep_current`) or null (`mode=none`).
  4. Redirect.
- **Error Cases:**
  - Active sprint exists → 422 "End the current sprint before starting a new one."
  - Validation failures → 422 with field-level messages.

### `POST /sprints/{sprint}/end` — End the active sprint
- **Route name:** `sprints.end`
- **Purpose:** Archive the active sprint at family-meeting close.
- **Authorization:** None.
- **Request Input:** None (sprint resolved via route).
- **Response Output:** Redirect to `tracker.show` (now empty state) with `flash.success = "Sprint archived. You can start a new one."`.
- **Behavior:**
  1. Reject if `sprint.status !== 'active'`.
  2. Call `endSprint(sprint)`.
  3. Redirect.
- **Error Cases:** Already archived → 422 "Sprint is already archived."

### `GET /history` — Past sprints index
- **Route name:** `history.index`
- **Purpose:** List archived sprints.
- **Authorization:** None.
- **Request Input:** None.
- **Response Output:** Inertia page `history/Index` with props:
  - `sprints`: array of `{ id, start_date, end_date, ended_at, participants_count, achievements_count }` (most recent first).
- **Behavior:** Call `listArchivedSprints()`. Render.

### `GET /history/{sprint}` — Past sprint detail
- **Route name:** `history.show`
- **Purpose:** Read-only view of an archived sprint's grid.
- **Authorization:** None.
- **Request Input:** Sprint resolved via route binding.
- **Response Output:** Inertia page `history/Show` with same shape as `tracker.show` but `sprint.status = archived`. All cells render read-only.
- **Behavior:**
  1. Reject (404) if sprint is not archived.
  2. Call `getArchivedSprintGrid(sprint)`.
  3. Render.

### `GET /totals` — Lifetime totals
- **Route name:** `totals.show`
- **Purpose:** Per-person totals across all sprints.
- **Authorization:** None.
- **Request Input:** None.
- **Response Output:** Inertia page `totals/Show` with props:
  - `people`: array of `{ id, name, display_order, total_ticks, total_rewards_achieved, sprints_participated }` (in display order).

### `GET /settings` — Settings page
- **Route name:** `settings.index`
- **Purpose:** Manage people, current sprint config, and per-person rewards.
- **Authorization:** None.
- **Request Input:** None.
- **Response Output:** Inertia page `settings/Index` with props:
  - `people`: array of `{ id, name, display_order, deleted_at }`
  - `active_sprint`: same shape as `tracker.show`'s `sprint` plus full participant/habit/reward editing context (or null).
  - `available_for_sprint`: people not currently participating in active sprint (if any) — the pool to add from.
  - `today`: ISO date string.
  - `next_sprint_defaults`: `{ start_date: today, end_date: today + 28 days, participants: [...prior sprint roster with habits and currently-active rewards] }` — used to pre-fill the "Start new sprint" form when no active sprint exists.

### `POST /people` — Create person
- **Route name:** `people.store`
- **Input:** `name` (string 1–50, required), `display_order` (int, optional, defaults to max+1).
- **Output:** Redirect to `settings.index` with `flash.success = "Person added"`.

### `PATCH /people/{person}` — Update person
- **Route name:** `people.update`
- **Input:** `name` (string 1–50, optional), `display_order` (int, optional).
- **Output:** Redirect to `settings.index` with `flash.success = "Person updated"`.

### `DELETE /people/{person}` — Soft-delete person
- **Route name:** `people.destroy`
- **Input:** None.
- **Output:** Redirect to `settings.index` with `flash.success = "Person removed"`.
- **Behavior:** Soft-deletes the Person; if they're in active sprint, also detach (cascade soft-delete their habits in active sprint).

### `POST /sprints/{sprint}/participants` — Add participant mid-sprint
- **Route name:** `sprint-participants.store`
- **Input:** `person_id` (int, required), `habits` (array of `{name, display_order?}`, optional), `reward` (object same shape as in sprint create, nullable).
- **Output:** Redirect to `settings.index` with `flash.success = "Participant added"`.

### `POST /sprint-participants/{sprintParticipant}/habits` — Add habit
- **Route name:** `habits.store`
- **Input:** `name` (string 1–80, required), `display_order` (int, optional).
- **Output:** Redirect to `settings.index` with `flash.success = "Habit added"`.

### `PATCH /habits/{habit}` — Update habit
- **Route name:** `habits.update`
- **Input:** `name` (string 1–80, optional), `display_order` (int, optional).
- **Output:** Redirect to `settings.index` with `flash.success = "Habit updated"`.

### `DELETE /habits/{habit}` — Soft-delete habit
- **Route name:** `habits.destroy`
- **Output:** Redirect to `settings.index` with `flash.success = "Habit removed"`.

### `POST /people/{person}/rewards` — Create reward
- **Route name:** `rewards.store`
- **Input:** `name` (string 1–80, required), `cost` (int 1–9999, required).
- **Output:** Redirect to `settings.index` with `flash.success = "Reward set"`.
- **Error:** 422 "This person already has an active reward" if one exists.

### `PATCH /rewards/{reward}` — Update reward
- **Route name:** `rewards.update`
- **Input:** `name` (string 1–80, optional), `cost` (int 1–9999, optional).
- **Output:** Redirect to `settings.index` with `flash.success = "Reward updated"`. If the cost edit triggered immediate achievement, additionally set `flash.achievement`.
- **Error:** 422 if attempting to edit cost of an achieved reward.

### `DELETE /rewards/{reward}` — Delete an unachieved reward
- **Route name:** `rewards.destroy`
- **Output:** Redirect to `settings.index` with `flash.success = "Reward deleted"`.
- **Error:** 422 if reward is achieved or is the active_reward of an active-sprint participant.

### `POST /sprint-participants/{sprintParticipant}/active-reward` — Swap active reward
- **Route name:** `sprint-participants.update-reward`
- **Input:** `reward_id` (int, required, must belong to this participant's person, must not be achieved).
- **Output:** Redirect to `settings.index` with `flash.success = "Active reward updated"`.

## Integration Contract

### With Backend

- Uses repository methods listed in `spec-be.md` → **Queries Required**.
- Uses factory/action methods listed in `spec-be.md` → **Mutations Required**.
- Each Form Request reuses validation rule sets exposed by Backend models/services (so rules don't drift between backend invariants and frontend validation).
- Route-model binding resolves `Person`, `Sprint`, `SprintParticipant`, `Habit`, `Reward` directly. Soft-deleted Person/Habit are excluded from binding by default.

### With Frontend

- **Inertia props** are the only way data flows server→client. Shapes documented per route above.
- **Flash messages** in shared Inertia props: `flash.success` (string), `flash.error` (string), `flash.achievement` (object | null).
- **Wayfinder** generates typed route helpers (`@/routes/...`) and typed action helpers (`@/actions/...`) for every controller method. The Frontend imports these instead of hardcoding URLs. Run `npm run build` (or the Wayfinder Vite plugin in dev) regenerates them.
- **Forms** are submitted via Inertia's `useForm` / `<Form>`. Server validation errors come back as field-level errors; the Frontend renders them inline.
- **Tick toggles** can be sent via `router.post(...)` with `preserveScroll: true` and `preserveState: true` so the grid view doesn't jump on each tap.

## Infrastructure Integration

This is a brand-new project with no integration skills configured. Apply only what's relevant:

- [ ] **Roles/Permissions:** **Not applicable.** No auth in this app. No permission gating on any route. No `can(...)` middleware. The standard "every route must be permission-gated" rule does not apply because there is no concept of users, roles, or permissions in this single-tablet app.
- [ ] **Feature Flag:** Not applicable.
- [ ] **Realtime:** Not applicable.
- [ ] **Permission rollout (manual grant by school admin):** Not applicable — no permissions exist.
- [ ] **Tier 2 release seeder:** Not applicable.
- [ ] **CSRF protection:** Default Laravel CSRF middleware applies to all `POST/PATCH/DELETE`.
- [ ] **Wayfinder generation:** Run on every route file change so Frontend stays in sync.
- [ ] **Fortify routes:** **Do not register** for this app. Either remove the FortifyServiceProvider from `bootstrap/providers.php` (preferred for cleanliness) or leave it but ensure no Fortify route is exposed in user navigation. The dev agent should pick one approach and document it; the brainstorm marked this `[DEFERRED]`. Recommended default: leave FortifyServiceProvider registered (already on by starter) but ensure no Fortify routes appear in `routes/auth.php` or are added to user navigation; create a simple bypass so any direct hit on a Fortify URL falls through to a 404 or redirects to `/`. The dev agent may simplify by simply not adding any Fortify-related links anywhere; the routes that exist by default are harmless on a single-tablet local network.

## Acceptance Criteria

- [ ] All 17 routes above exist with the named-route names listed, registered in `routes/web.php` (or a `routes/tracker.php` file included from `web.php`).
- [ ] Each mutation route has a Form Request class enforcing the validation rules above.
- [ ] Controllers are thin: validate (via Form Request), call Backend service, redirect-or-render. No business logic in controllers.
- [ ] Inertia props match the documented shapes exactly. Frontend should not need to massage data.
- [ ] Route-model binding works for all bound parameters, with soft-deleted models excluded.
- [ ] Wayfinder typed routes/actions are generated and committed (or the dev agent confirms they regenerate at build time via `vite.config.ts`).
- [ ] Flash messages are populated correctly for: success of every mutation, achievement-celebration on tick that crosses threshold, errors for every documented error case.
- [ ] PHPUnit feature tests cover every route's success case + each documented error case + the achievement-flash flow.
- [ ] All code passes `vendor/bin/pint --dirty --format agent` with no diffs.
- [ ] Tablet can navigate Tracker ↔ History ↔ Totals ↔ Settings using top-nav links, all served via Inertia (no full page reloads).
