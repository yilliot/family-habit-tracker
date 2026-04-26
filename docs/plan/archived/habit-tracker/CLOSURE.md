# Habit Tracker — Closure Notes

**Status:** SHIPPED
**Closed:** 2026-04-26
**Decision:** No feature flag — deployed immediately as the app's only feature on a single shared family tablet with no authentication.

---

## What Was Shipped

| Component | Location | Notes |
|-----------|----------|-------|
| Person model | `app/Models/Person.php` | SoftDeletes, nameRules() helper |
| Sprint model | `app/Models/Sprint.php` | active/archived scopes, STATUS_* constants |
| SprintParticipant model | `app/Models/SprintParticipant.php` | carry_forward_balance, active_reward FK |
| Habit model | `app/Models/Habit.php` | SoftDeletes, ordered scope |
| Tick model | `app/Models/Tick.php` | unique (habit_id, tick_date) at DB level |
| Reward model | `app/Models/Reward.php` | active/achieved scopes, cost/name rule helpers |
| People migration | `database/migrations/2026_04_26_140000_create_people_table.php` | soft-delete |
| Sprints migration | `database/migrations/2026_04_26_140001_create_sprints_table.php` | |
| Rewards migration | `database/migrations/2026_04_26_140002_create_rewards_table.php` | |
| SprintParticipants migration | `database/migrations/2026_04_26_140003_create_sprint_participants_table.php` | unique (sprint_id, person_id) |
| Habits migration | `database/migrations/2026_04_26_140004_create_habits_table.php` | soft-delete; nullOnDelete on sprint_participant_id (see deviations) |
| Ticks migration | `database/migrations/2026_04_26_140005_create_ticks_table.php` | unique (habit_id, tick_date) |
| 6 factories | `database/factories/{Person,Sprint,SprintParticipant,Habit,Tick,Reward}Factory.php` | SprintFactory::archived(), RewardFactory::achieved() states |
| FamilySeeder | `database/seeders/FamilySeeder.php` | Idempotent seed of 爸爸/妈妈/可遇/奇乐 |
| HabitTrackerSprintSeeder | `database/seeders/HabitTrackerSprintSeeder.php` | Seeds Apr 19–May 16 2026 sprint with habits from print HTML |
| 15 action classes | `app/Actions/HabitTracker/` | One class per mutation; throw ValidationException for business-rule violations |
| PointsCalculator | `app/Support/HabitTracker/PointsCalculator.php` | Lifetime + sprint-scoped balance; never persisted |
| HabitTrackerRepository | `app/Repositories/HabitTracker/HabitTrackerRepository.php` | All 9 queries; eager-loaded |
| SettingsPresenter | `app/Support/HabitTracker/SettingsPresenter.php` | Builds available_for_sprint, next_sprint_defaults, rewards_by_person |
| 10 controllers | `app/Http/Controllers/HabitTracker/` | Thin; delegate to repo/actions |
| 10 Form Requests | `app/Http/Requests/HabitTracker/` | Reuse model rule helpers |
| 19 named routes | `routes/tracker.php` | spec listed "17" in prose, enumerated 19; all present |
| Flash sharing | `app/Http/Middleware/HandleInertiaRequests.php` | success/error/achievement shared on every response |
| Wayfinder generation | `resources/js/actions/`, `resources/js/routes/` | Typed helpers for all controllers/routes |
| Habit tracker layout | `resources/js/layouts/habit-tracker-layout.tsx` | Masthead, top-nav, footer; newspaper aesthetic |
| Tracker page | `resources/js/pages/tracker/Show.tsx` | Active sprint grid; optimistic ticking |
| History Index page | `resources/js/pages/history/Index.tsx` | Archived sprint list |
| History Show page | `resources/js/pages/history/Show.tsx` | Read-only archived grid |
| Totals page | `resources/js/pages/totals/Show.tsx` | Per-person lifetime cards |
| Settings page | `resources/js/pages/settings/Index.tsx` | 3-tab (Current Sprint / People / Rewards) |
| Sprint grid component | `resources/js/components/habit-tracker/sprint-grid.tsx` | Optimistic ticks, weekend/Sunday/month-start markers, 44px tap targets |
| Progress strip | `resources/js/components/habit-tracker/progress-strip.tsx` | Points balance, reward progress bar |
| Celebration overlay | `resources/js/components/habit-tracker/celebration-overlay.tsx` | Full-screen, fires on flash.achievement |
| End-sprint confirm | `resources/js/components/habit-tracker/end-sprint-confirm.tsx` | Modal |
| 8 form components | `resources/js/components/habit-tracker/forms/` | All mutations covered |
| Masthead / TopNav / Footer | `resources/js/components/habit-tracker/` | |
| HabitTracker types | `resources/js/types/habit-tracker.ts` | Inertia prop types |
| Tailwind theme tokens | `resources/css/app.css` | `@theme` block: paper/ink/terracotta/line color vars |
| Google Fonts | `resources/views/app.blade.php` | Noto Serif SC, Fraunces, Inter, Noto Sans SC |
| 44 backend tests | `tests/Feature/HabitTracker/{PersonTest,HabitMutationTest,ToggleTickTest,RewardTest,SprintLifecycleTest,RepositoryTest,FamilySeederTest}.php` | |
| 88 HTTP / platform tests | `tests/Feature/HabitTracker/Http/` | 9 controller test files |

---

## Key Design Decisions

### 1. Habits `nullOnDelete` instead of cascade
`habits.sprint_participant_id` uses `nullOnDelete` rather than cascade. Cascade would have hard-deleted habits (and therefore ticks) when a SprintParticipant row was removed (e.g. when soft-deleting a person mid-sprint). `nullOnDelete` keeps the habit rows (and their ticks) intact, satisfying the "archived ticks keep counting toward balance" requirement. Habits with a null `sprint_participant_id` are excluded from active-grid queries but preserved for history.

### 2. Carry-forward seed computed as lifetime balance
`carry_forward_balance` is seeded as `max(0, lifetimeBalanceForPerson)` — total ticks across all sprints minus total achieved-reward cost. This cleanly handles both "unfulfilled reward balance" and "post-achievement surplus" without storing per-sprint ending state.

### 3. Route load-order comment removed; redirect removed instead
The original workaround loaded `tracker.php` after `settings.php` to make `GET /settings` resolve to `settings.index` in uncached mode (last-registered wins). This failed in production where `php artisan route:cache` uses Symfony's first-registered-wins UrlMatcher. Fix: removed the conflicting `Route::redirect('settings', '/settings/profile')` line from `routes/settings.php` entirely. That redirect exists for logged-in users navigating to `/settings` — this app has no logged-in users.

### 4. Reward mode adapter in controllers
The `StoreSprintRequest` spec defined a `reward.mode` enum (`keep_current` / `new` / `none`) but the underlying `StartSprint` action takes a simpler `{name, cost} | null` shape and auto-links existing active rewards. Controllers translate the mode to the action's expected shape.

### 5. Instructions micro-strip omitted
The Tracker page omits the "How it works · Legend · Total ticks possible" strip from the print reference. The masthead + progress strip already orient users sufficiently; the strip would have duplicated information in the live app.

### 6. Footer: dynamic not named
The print reference has four named signature lines (爸爸 / 妈妈 / 可遇 / 奇乐). The digital footer uses a generic 3-column layout (family note / tablet blurb / 家 brand mark) because the roster is dynamically configured.

### 7. Portrait layout: horizontal-scroll grid, not dedicated stacked view
The spec called for "big tappable buttons + horizontal-scroll week" in portrait. The shipped implementation uses `overflow-x-auto` on the grid wrapper for portrait scrollability. A dedicated portrait-specific per-person stacked view (spec's ideal) is a future enhancement.

### 8. Settings tab: URL not updated on switch
Settings tabs are managed by local component state. The celebration overlay's deep link (`?tab=rewards&person=N`) is read once on render. Tab changes do not update the URL, so you cannot bookmark a specific tab.

---

## Acceptance Criteria Review

### Backend

- [x] Migrations exist for all 6 tables with correct columns, indexes, foreign keys, and soft-delete columns.
- [x] Eloquent models exist with correct relationships, casts, fillables, and `SoftDeletes` trait where specified.
- [x] Factories exist for each model. FamilySeeder pre-populates the 4 members; HabitTrackerSprintSeeder seeds the first sprint.
- [x] Service / action classes implement every mutation in Mutations Required, each with a single public entry method.
- [x] Repository / query class implements every method in Queries Required.
- [x] Toggle-tick is idempotent (unique constraint + toggle logic).
- [x] Achievement detection runs on every tick creation and every cost-lowering reward edit.
- [x] Sprint-archive triggers carry-forward; `startSprint` seeds participants correctly.
- [x] Future-day ticks and out-of-range ticks are rejected.
- [x] Edits to archived sprint data are rejected.
- [x] PHPUnit feature tests cover all listed scenarios (44 backend tests).
- [x] Pint clean.
- [x] All tests pass.

### Platform

- [x] All 19 named routes exist and registered.
- [x] Each mutation route has a Form Request class.
- [x] Controllers are thin (validate → call service → redirect/render).
- [x] Inertia props match documented shapes.
- [x] Route-model binding works; soft-deleted models excluded.
- [x] Wayfinder typed routes/actions generated.
- [x] Flash messages populated correctly (success, error, achievement).
- [x] PHPUnit feature tests cover all routes (88 HTTP tests).
- [x] Pint clean.
- [~] `Route::redirect('settings')` conflict — resolved post-deploy by removing the conflicting route line. Production `/settings` now returns 200.

### Frontend

- [x] Four pages render at correct routes.
- [x] Tracker grid visually follows print reference (weekend tint, Sunday dot, month-start divider).
- [x] Optimistic tick UI (cell flips immediately; reverts on error).
- [x] Future-day cells locked visually.
- [x] Past-day cells in active sprint are tappable.
- [x] Archived sprint grids are read-only.
- [x] Progress strip shows balance / reward cost with terracotta bar.
- [x] Celebration overlay opens on `flash.achievement`; "Pick next reward" + "Maybe later".
- [x] Top-nav navigates via Inertia (no full-page reloads).
- [x] Settings page — people / sprint / rewards tabs.
- [x] Empty states present.
- [~] Portrait layout — horizontal-scroll grid instead of dedicated stacked-per-person view.
- [x] All Wayfinder typed routes/actions used; no hardcoded URLs.
- [x] Flash feedback on all submissions.
- [x] Lint, format, types clean; `npm run build` succeeds.
- [-] Instructions micro-strip — intentionally omitted; masthead + progress strip suffice.
- [-] Footer named signatures — intentionally replaced with dynamic-friendly footer.
- [-] Settings URL updated on tab switch — deferred; local state only in v1.

---

## What Was NOT Delivered (Deferred)

| Item | Reason |
|------|--------|
| Portrait dedicated stacked-per-person view | Deferred; horizontal-scroll is a workable fallback. Revisit if family finds portrait cramped. |
| Settings tab URL sync | Low priority; local state is fine for v1. |
| Print-friendly export of a finished sprint | Marked deferred during brainstorm; `@media print` stub acceptable for v1. |
| SQLite backup strategy | Ops concern; not feature-blocking. |
| Streak tracking / badges | Explicitly out of scope (editorial tone). |
| Instructions micro-strip on Tracker | Intentionally omitted — content covered by progress strip. |
