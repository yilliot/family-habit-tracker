# Habit Tracker — Backend Specification

> **STATUS: CLOSED — Shipped 2026-04-26**
> All 6 models, migrations, factories, 15 actions, repository, and 44 PHPUnit tests delivered. Key deviation: `habits.sprint_participant_id` uses `nullOnDelete` (not cascade) to preserve tick history when a participant is removed.
> See [CLOSURE.md](./CLOSURE.md) for full delivery notes.

> Date: 2026-04-26
> Idea: docs/plan/habit-tracker/idea.md

## Required Skills

> The dev agent MUST `Read`/activate every skill below BEFORE writing code.

- **`laravel-best-practices`** (Skill) — model design, migrations, factories, seeders, validation, query patterns, N+1 avoidance.
- **`CLAUDE.md`** project file — Laravel 13 / PHP 8.3 / Boost / Pint / PHPUnit conventions, especially the `=== php rules ===` and `=== laravel/core rules ===` blocks.
- **Laravel Boost MCP tools** — `database-schema`, `database-query`, `search-docs` per `=== boost rules ===` in CLAUDE.md.
- **Reference document:** `Users/el/Desktop/family habit tracker/Family Habit Tracker.html` — for understanding the printed-paper data model that this digital version mirrors.

> This project is new and has no `docs/dev/*` or `docs/integration/*` skills yet. Defer to the Skill-tool skills above plus CLAUDE.md.

## Problem Statement

The family currently runs habit-tracking on paper. Paper has no live point totals, no carry-forward across multi-month rewards, and no history. We need a persistence layer that:

1. Stores family members (people) globally, sprints with their dates and lifecycle, per-sprint habits, daily ticks, and per-person rewards with point costs.
2. Computes each participant's points balance on demand (no stale stored totals).
3. Supports carry-forward: when a sprint ends without a person hitting their reward goal, their balance and active reward roll into the next sprint.
4. Supports soft-deleting habits/people mid-life so existing tick history is preserved.
5. Enforces the business rules: at most one active sprint, no future-day ticks, no edits to archived sprints, no cost edits to achieved rewards.

## Objective

Provide the data layer (Eloquent models, migrations, factories, repositories, factory/service classes for mutations, business-rule enforcement) that the Platform layer can call to fulfill all routes specified in `spec-pl.md`.

## Scope

### In Scope

- Eloquent models, migrations, factories, seeders for: `Person`, `Sprint`, `SprintParticipant`, `Habit`, `Tick`, `Reward`.
- Soft-deletes on `Person` and `Habit`.
- Business-rule enforcement (validation methods invoked by Platform's Form Requests, plus DB-level constraints where applicable).
- Repository / query class methods that return data shapes the Platform layer needs.
- Factory / service class methods that perform mutations (create, update, delete, tick toggle, sprint end, sprint start with carry-forward).
- A "compute balance" service that returns the live points balance for a `(SprintParticipant)` or person.
- A "detect achievement" routine invoked after a tick is created that flips a reward to `achieved` if balance ≥ cost.
- Database seeder with the original 4-person family pre-populated (爸爸/妈妈/可遇/奇乐) for first-run convenience.
- PHPUnit feature tests for every business rule and edge case.

### Out of Scope

- HTTP controllers, routes, request validation classes — handled by Platform.
- Inertia props / view rendering — handled by Platform / Frontend.
- Notifications, jobs, queues — none.
- Authentication / authorization — none (single-tablet, no auth).
- Caching layer — not needed for single-device usage.

## Data Requirements

### Entity: Person

- **Purpose:** A family member. Global; persists across sprints. Soft-deletable.
- **Fields:**
  - `id` (bigint, pk)
  - `name` (string, max 50, required) — freeform, any language.
  - `display_order` (unsigned int, default 0) — for grid ordering.
  - `created_at`, `updated_at`, `deleted_at` (timestamps; `deleted_at` for soft-delete).
- **Relationships:**
  - `hasMany` SprintParticipant
  - `hasMany` Reward
  - `hasMany` Habit (via SprintParticipant)
- **Business Rules:**
  - Soft-delete only; ticks and history are preserved.
  - Display order is unique-per-active-row not enforced (ties broken by `id`).

### Entity: Sprint

- **Purpose:** A bounded period (start date → end date). Holds the per-sprint snapshot of participation. At most one `active` sprint at a time.
- **Fields:**
  - `id` (bigint, pk)
  - `start_date` (date, required) — inclusive.
  - `end_date` (date, required) — inclusive; must be ≥ start_date.
  - `status` (string enum: `active`, `archived`, default `active`)
  - `started_at` (timestamp, nullable) — when the sprint became active (= created_at typically).
  - `ended_at` (timestamp, nullable) — set when archived.
  - `created_at`, `updated_at`.
- **Relationships:**
  - `hasMany` SprintParticipant
  - `hasMany` Habit (via participants)
  - `hasMany` Tick (via habits)
- **Business Rules:**
  - Only one row may have `status = active` at any time. Enforce in mutation service (DB partial-unique-index also recommended where supported; otherwise app-level lock).
  - `end_date >= start_date`.
  - Once `archived`, all child writes (ticks, habit edits, reward cost edits within this sprint context) are rejected.
  - On archive: set `status = archived`, `ended_at = now`. Trigger carry-forward calculation for each participant.

### Entity: SprintParticipant

- **Purpose:** Pivot recording a person's participation in a sprint, with per-sprint state.
- **Fields:**
  - `id` (bigint, pk)
  - `sprint_id` (fk → sprints.id, cascading delete)
  - `person_id` (fk → people.id, restrict delete — soft-delete instead)
  - `carry_forward_balance` (unsigned int, default 0) — points seeded from previous sprint.
  - `active_reward_id` (fk → rewards.id, nullable) — the reward this person was working on at sprint start (snapshot; the actual `Reward` row is the source of truth for cost/achievement).
  - `display_order` (unsigned int, default 0) — for in-sprint ordering (defaults to the Person's display order on creation).
  - `created_at`, `updated_at`.
- **Relationships:**
  - `belongsTo` Sprint, `belongsTo` Person, `belongsTo` Reward (active_reward, nullable).
  - `hasMany` Habit.
- **Business Rules:**
  - Unique on `(sprint_id, person_id)`.
  - On delete of the Sprint, cascade.
  - On soft-delete of the Person, retain row (history).

### Entity: Habit

- **Purpose:** A specific behavior tracked for one participant within one sprint.
- **Fields:**
  - `id` (bigint, pk)
  - `sprint_participant_id` (fk → sprint_participants.id, cascading delete)
  - `name` (string, max 80, required) — freeform.
  - `display_order` (unsigned int, default 0)
  - `created_at`, `updated_at`, `deleted_at` (soft-delete).
- **Relationships:**
  - `belongsTo` SprintParticipant.
  - `hasMany` Tick.
- **Business Rules:**
  - Soft-delete only mid-sprint. Ticks remain for points-balance and history.
  - Cannot be created/edited on an archived sprint.

### Entity: Tick

- **Purpose:** One day's "done" mark on one habit. Unique per (habit, date).
- **Fields:**
  - `id` (bigint, pk)
  - `habit_id` (fk → habits.id, cascading delete on hard delete; preserved on soft delete)
  - `tick_date` (date, required)
  - `created_at`.
- **Relationships:**
  - `belongsTo` Habit.
- **Business Rules:**
  - Unique on `(habit_id, tick_date)` — prevents duplicates from rapid double-taps.
  - `tick_date` must be within the parent sprint's `[start_date, end_date]` inclusive.
  - `tick_date` must be ≤ today (server clock) — no future ticks.
  - Cannot be created or deleted on an archived sprint.
  - Toggle semantics: if a tick exists for (habit_id, tick_date), delete it; else create it.

### Entity: Reward

- **Purpose:** A goal a person is saving toward. Has a name and an integer cost.
- **Fields:**
  - `id` (bigint, pk)
  - `person_id` (fk → people.id, cascading on hard delete; preserved on soft delete of person)
  - `name` (string, max 80, required) — freeform.
  - `cost` (unsigned int, 1–9999, required).
  - `achieved_at` (timestamp, nullable) — set when balance ≥ cost.
  - `created_at`, `updated_at`.
- **Relationships:**
  - `belongsTo` Person.
- **Business Rules:**
  - At most one Reward per Person where `achieved_at IS NULL` (the "active" reward). Enforce in mutation service.
  - When `achieved_at` is set, the row is final: `cost` is locked from edits; `name` may still be edited (label).
  - Cost edits to a non-achieved reward trigger an immediate achievement check against the person's current balance — if balance ≥ new cost, mark achieved with current timestamp.

### Queries Required

> All return shapes expressed conceptually; the dev agent decides whether to use Eloquent collections, plain arrays, or DTOs.

- **`getActiveSprint()`** → the single active Sprint, or null. Eager-load: participants → person, participants → habits (non-trashed) → ticks, participants → activeReward.
- **`getActiveSprintGrid()`** → returns the data shape needed to render the tracker grid:
  - Sprint dates (start, end, list of every day with weekend flag, month-boundary flag).
  - Each participant in display order: person name, display order, current points balance (computed), carry-forward balance, current active reward (id, name, cost, achieved_at), and that participant's habits (in display order) each with the set of `tick_date`s they have.
- **`getPersonBalance(Person $person)`** → integer balance for a person at the moment of call, scoped by:
  - Current sprint (if active and they participate): carry-forward + ticks − cost-of-rewards-achieved-during-this-sprint.
  - For lifetime totals: all ticks ever earned − total cost of all rewards achieved.
- **`getSprintParticipantBalance(SprintParticipant $sp)`** → same as above but scoped to that sprint.
- **`listArchivedSprints()`** → list of archived sprints (most recent first), each with: dates, participant count, who achieved a reward in that sprint.
- **`getArchivedSprintGrid(Sprint $sprint)`** → same shape as `getActiveSprintGrid` but read-only.
- **`getLifetimeTotalsPerPerson()`** → for each non-trashed Person: `total_ticks_earned`, `total_rewards_achieved`, `sprints_participated_count`.
- **`listPeople()`** → all non-trashed people in display-order.
- **`listPersonRewards(Person $person)`** → all rewards (achieved + active), sorted achieved_at DESC nulls first.

### Mutations Required

- **`createPerson(name, display_order?)`** → Person. Validation: name 1–50 chars.
- **`updatePerson(Person $person, name?, display_order?)`** → updated Person.
- **`softDeletePerson(Person $person)`** → soft-deletes; if person is in active sprint, also detach them from the sprint (their SprintParticipant is removed; their habits soft-delete; their ticks remain via the soft-deleted habits).
- **`startSprint(start_date, end_date, participants[])`** where each participant is `{ person_id, habits: [{name, display_order}], reward: {name, cost} | null }`.
  - Reject if any active sprint exists.
  - Validate dates (`end_date >= start_date`, `start_date >= today` not strictly required — allow back-dating for the first sprint).
  - Validate each participant: person exists & not trashed, habits 0–N each (≥1 recommended), reward has 1–80 char name and 1–9999 cost when present.
  - For each participant: compute carry-forward seed = previous-sprint's ending balance for that person if they had an unfulfilled reward; otherwise the surplus from the most recent achieved reward not yet "spent" on a new one. Persist as `carry_forward_balance`.
  - If reward is provided and the person has no active reward, create the reward; if person already has an active reward, link the existing one as `active_reward_id`.
  - Returns: created Sprint with eager-loaded participants/habits.
- **`endSprint(Sprint $sprint)`** → archives. Sets status, ended_at. No further writes allowed.
- **`addParticipant(Sprint $sprint, person_id, habits[], reward?)`** → mid-sprint participant addition. Sprint must be active.
- **`updateParticipantReward(SprintParticipant $sp, reward_id)`** → swap which reward is the "active" one for this participant in this sprint. Validation: reward must belong to this participant's person and not be achieved.
- **`createHabit(SprintParticipant $sp, name, display_order?)`** → Habit. Sprint must be active.
- **`updateHabit(Habit $habit, name?, display_order?)`** → Habit. Sprint must be active.
- **`softDeleteHabit(Habit $habit)`** → Habit. Sprint must be active. Existing ticks preserved.
- **`toggleTick(Habit $habit, tick_date)`** → returns `{ tick: Tick|null, balance_after: int, just_achieved_reward: Reward|null }`.
  - Validate: sprint active, date in sprint range, date ≤ today.
  - If a tick exists, delete it; else create it.
  - After mutation, compute balance for the participant; if their active reward is non-null and `balance >= reward.cost` and `achieved_at is null`, set `achieved_at = now` and return that reward as `just_achieved_reward`.
- **`createReward(Person $person, name, cost)`** → Reward. Reject if person already has an active reward.
- **`updateReward(Reward $reward, name?, cost?)`** → Reward.
  - If reward is achieved: only `name` editable.
  - If reward is active and `cost` is changed: re-run achievement check against person's current balance; if balance ≥ new cost, mark achieved.
- **`deleteReward(Reward $reward)`** → only if `achieved_at IS NULL` and reward is not the `active_reward_id` of a SprintParticipant in an active sprint. Otherwise reject.

### Jobs Required

None.

### Notifications Required

None.

## Integration Contract (with Platform)

### Provides to Platform

- A repository class (or set of query objects) exposing every method in **Queries Required** above.
- A factory / service class (or set of action classes per Laravel-best-practices) exposing every method in **Mutations Required** above.
- Eloquent models with route-model binding ready (Person, Sprint, SprintParticipant, Habit, Reward; Tick is not bound — created/deleted via `toggleTick`).
- Form Request validation rules surfaced as static methods or constants on the model/service classes so the Platform's Form Requests can re-use them (e.g., `Person::nameRules()`).

### Receives from Platform

- Resolved Eloquent model instances from route-model binding.
- Validated attribute arrays (the Platform does the input validation; Backend trusts them but still enforces invariants like "no active sprint already exists").

## Infrastructure Integration

This is a brand-new project with no integration skills set up yet. Apply only what's needed:

- [ ] **Notifications:** None.
- [ ] **Job Queue:** None.
- [ ] **Money:** Not applicable — points are unitless integers.
- [ ] **Serial:** Not applicable.
- [ ] **School Settings:** Not applicable (no multi-tenant).
- [ ] **User Settings:** Not applicable (no auth).
- [ ] **Permissions enum / SettingKey enum:** Not applicable (no auth, no feature flags).
- [ ] **Database (SQLite):** Default `database/database.sqlite`. All migrations target this connection.
- [ ] **Soft-deletes:** `Person` and `Habit` use `SoftDeletes` trait.

## Acceptance Criteria

- [ ] Migrations exist for all 6 tables with correct columns, indexes, foreign keys, and soft-delete columns where specified.
- [ ] Eloquent models exist with correct relationships, casts (date casts on `start_date`, `end_date`, `tick_date`, `achieved_at`), fillables, and `SoftDeletes` trait where specified.
- [ ] Factories exist for each model. Seeder pre-populates the 4 family members (爸爸, 妈妈, 可遇, 奇乐) but creates no sprint by default.
- [ ] Service / action classes implement every mutation in **Mutations Required**, each with a single public entry method.
- [ ] Repository / query class implements every method in **Queries Required**.
- [ ] Toggle-tick is idempotent: tapping twice in rapid succession results in the cell being in the original state, not duplicated.
- [ ] Achievement detection runs on every successful tick creation and on every cost-lowering reward edit; never on cost-raising edits to an already-achieved reward (blocked).
- [ ] Sprint-archive triggers carry-forward computation; the next `startSprint` call seeds participants correctly.
- [ ] Future-day ticks (server-clock) and ticks outside sprint date range are rejected with a clear validation message.
- [ ] Edits to anything inside an archived sprint are rejected.
- [ ] PHPUnit feature tests cover: tick toggle (create + delete), achievement on tick, achievement on cost edit, carry-forward across sprints, soft-delete habit preserves ticks, soft-delete person preserves history, end-of-sprint while no reward set, two-sprint-attempt rejected, reward edit rules, and all listed edge cases in `idea.md`.
- [ ] All code passes `vendor/bin/pint --dirty --format agent` with no diffs.
- [ ] All tests pass via `php artisan test --compact`.
