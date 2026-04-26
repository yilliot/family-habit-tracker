# Habit Tracker - Idea

> **STATUS: CLOSED — Shipped 2026-04-26**
> Full feature delivered: 6 models, 19 routes, 5 pages, newspaper aesthetic, optimistic ticking, multi-sprint carry-forward rewards. Production route-cache bug fixed post-deploy.
> See [CLOSURE.md](./CLOSURE.md) for full delivery notes.

> Brainstormed on: 2026-04-26

## The Problem

The family currently runs habit-tracking on a printed paper sheet (`Family Habit Tracker.html`, 28-day Apr 19 – May 16, 2026 ledger). Paper works but has friction:

- Ticks get lost when the sheet is taken down or moved
- Totals must be tallied by hand; no live progress to "the next reward"
- Big rewards (e.g., a 250pt LEGO set, ~3 sprints' worth) can't be tracked across multiple printed sheets
- New sprints mean re-printing and re-writing everyone's habits and rewards from scratch
- No history — a finished sheet either gets pinned somewhere or thrown away

We want a tablet-resident digital version that keeps the family-ledger feel but adds live point tracking, multi-sprint reward goals, sprint history, and easy reconfiguration at family meetings.

## The Idea

A single-tablet, no-auth web app that mirrors the printed habit ledger and adds live state. The home screen shows the **full sprint grid** (people × habits × N days, weekend shading, weekly groupings) — anyone in the family can walk up and tap today's box. Beyond the grid:

- **Sprints** are explicit, family-meeting-bounded periods with a start/end date
- Each sprint is configured with a flexible roster (anyone in the family-wide People list) and per-person habits (any number, freeform names)
- Each person has **one active reward** — a freeform name + point cost (e.g., "Lego Minecraft · 250pt"). 1 tick = 1 point. They earn toward it across however many sprints it takes.
- Points balance carries forward across sprints **until the reward goal is achieved**. When achieved, surplus seeds the next reward they pick.
- A **settings** page handles people, sprint dates, habits, and rewards. A **history** page archives past sprints. **Lifetime totals** track points and rewards earned across the family's whole run.

Visual language: keep the newspaper aesthetic from the print HTML (paper background, terracotta accent, Noto Serif SC + Fraunces italic, masthead with sprint dates, hairline grid). Layout is responsive — landscape shows the full grid, portrait collapses to a stacked per-person view.

## Key Terminology

| Term | Definition |
|------|------------|
| **Sprint** | A bounded period (start date → end date) decided at a family meeting. Holds a snapshot of participating people, each person's habits for that sprint, and the daily tick records. Started and ended manually. |
| **Person** | A family member. Stored globally (persists across sprints). Each sprint has an opt-in roster drawn from this global list — flexible (a person can sit out a sprint). |
| **Habit** | A specific behavior tracked for one person within one sprint. Freeform name (any language). Sprint-scoped — can change between sprints. |
| **Tick / Check-in** | A single day's "done" mark on one habit. Worth 1 point. Boolean (done/not-done) per (habit × day). Past days within the current sprint are tappable; future days are locked. |
| **Reward** | A freeform-named goal with a point cost (e.g., "Lego Minecraft · 250pt"). Each person has one **active** reward at a time. When the person's points balance reaches the cost, the reward is marked **achieved**. |
| **Points balance** | Per person. Sum of ticks earned minus points spent on achieved rewards. Carries across sprints when a reward goal hasn't been hit yet; surplus rolls onto the next reward when one is achieved. |
| **Family meeting** | The (offline) ritual that opens and closes sprints. Not a software feature — but the UX is shaped around it (clear start/end, easy reconfiguration). |

## User Stories

- As a **family member walking up to the tablet**, I want to tap today's box for one of my habits in two taps so daily check-in is friction-free.
- As a **family member**, I want to see how close I am to my current reward at a glance, so the daily ticks feel meaningful.
- As a **parent running the family meeting**, I want to set a sprint date range, edit each person's habits for the sprint, and confirm each person's active reward — all on one settings page.
- As a **family member**, I want to backfill yesterday's tick if I forgot, so a missed tap doesn't undermine the streak.
- As a **family member with a big reward goal (250pt)**, I want my unspent points to carry across sprints so I can keep saving toward the same reward over multiple months.
- As a **family member who just hit my goal**, I want a clear celebration on the tablet and the ability to pick my next reward whenever I'm ready (could be immediately, could be the next family meeting).
- As a **family**, I want to look back at past sprints to remember who hit which goals and to spot habit trends over time.

## Explored Alternatives

### Option A: Reward-centric (chosen)
- **Approach:** Reward is the primary goal. Each person has one active reward with a point cost. 1 tick = 1 point. Hit the cost = achieved. Surplus rolls. No separate "sprint goal" concept — the reward IS the goal, and its cost may exceed one sprint.
- **Pros:** Single mental model. Matches how the family already talks about it ("I want the Lego set, that's 250pt"). Naturally accommodates multi-sprint goals via point carry-forward.
- **Cons:** A person between rewards (just achieved one, hasn't picked a new one) earns ticks that pile up as orphaned surplus until they choose. Acceptable.

### Option B: Sprint-target + reward
- **Approach:** Each sprint has an explicit "points target" per person plus a separate reward for hitting the target. Two-tier model.
- **Pros:** Sprint feedback is sharper (achieved sprint goal vs not).
- **Cons:** Doesn't match user's mental model. Adds complexity. Multi-sprint big rewards become awkward.

### Option C: Percentage-completion goal
- **Approach:** Goal is a completion rate (e.g., 80% of habit boxes ticked over the sprint).
- **Pros:** Self-balancing for sprints of different lengths.
- **Cons:** Doesn't translate to a point bank for big rewards. Doesn't match the family's existing language.

### Chosen Direction
**Option A.** It collapses "goal" and "reward" into one concept (a reward with a point cost), aligns with the family's actual language, and naturally supports multi-sprint big rewards via carry-forward.

---

## Actors & Goals

| Actor | Goal | Priority |
|-------|------|----------|
| Any family member at the tablet | Tap today's habit box in 1–2 taps | High |
| Family member | See progress to current reward | High |
| Family meeting facilitator | Start a new sprint with date range and habits | High |
| Family meeting facilitator | Set / change a person's active reward | High |
| Family member | Backfill a missed-tap day | Medium |
| Family member | Pick the next reward after hitting one | Medium |
| Family | Browse past sprints and lifetime totals | Low |

## Scope Boundaries

### IN Scope (MVP)

- [ ] **Global People list** — CRUD on family members (single freeform name field)
- [ ] **Sprints** — create with start/end date; one active sprint at a time; archive on end; "start new sprint" pre-fills last sprint's roster + habits as defaults
- [ ] **Per-sprint roster** — choose which people participate this sprint
- [ ] **Per-sprint, per-person habits** — flexible count (1–N), single freeform name field per habit
- [ ] **Daily check-in grid** — full sprint shown as the home screen; rows = (person × habit), columns = days; tap a past-or-today cell to toggle a tick; future cells locked; weekend shading; month-boundary visual
- [ ] **Per-person active reward** — freeform name + integer point cost
- [ ] **Live points balance per person** — `total_ticks - points_spent_on_achieved_rewards`
- [ ] **Reward achievement** — when balance ≥ cost, mark achieved with timestamp; show a celebration banner; surplus persists toward next reward
- [ ] **Pick next reward** — anytime (no forced prompt); during the gap, ticks accrue as surplus
- [ ] **Mid-sprint edits** — anything (people, habits, rewards) is editable while a sprint is active; ticks for deleted habits archive (read-only history) rather than vanish
- [ ] **Carry-forward** — when a sprint ends without the goal hit, the same active reward and the points balance roll into the next sprint
- [ ] **Per-person progress bar** — visible on the grid view (header strip or sidebar) showing balance vs current reward cost
- [ ] **Settings page** — people, sprint dates / start-end actions, habits per person, rewards
- [ ] **Sprint history page** — read-only list of past sprints, drill into the archived grid + outcomes
- [ ] **Lifetime totals page** — per person: total points earned, count of rewards achieved
- [ ] **Newspaper aesthetic** — paper bg, terracotta accent, Noto Serif SC + Fraunces italic, masthead with sprint dates, hairline borders (match the reference HTML)
- [ ] **Responsive layout** — landscape shows the full grid; portrait collapses to a stacked per-person view (today + current week, with a way to expand to full sprint)

### OUT of Scope (Future)

- Authentication / user accounts — Reason: single shared tablet, no per-user privacy needed
- Multi-device sync / cloud backend — Reason: single tablet, local SQLite is sufficient
- Bilingual / structured i18n field schema — Reason: user opted for a single freeform name field; users can type Chinese, English, or mixed as they wish
- Custom point values per habit — Reason: user wants 1pt/tick uniformly to keep comparisons fair
- Multiple active rewards / wishlist per person — Reason: one-at-a-time matches the family's mental model; a wishlist is a future enhancement
- Per-sprint points target separate from reward cost — Reason: collapsed into the reward-cost model
- Notifications, reminders, push, email — Reason: ambient tablet on the fridge does not need notifications
- Streak tracking, badges, gamification beyond the reward — Reason: keep the editorial / understated tone of the print version
- Photo upload for rewards — Reason: text-only is enough; reduces scope
- Auto-scheduled "rollover" cron — Reason: sprint transitions are manual / family-meeting events
- Export to CSV / printable PDF of the live grid — Reason: a print stylesheet may be a v2 follow-up; not blocking
- Editing past sprints (after archive) — Reason: history is read-only by design

### Dependencies & Integration Infrastructure

| Dependency | Type | Notes |
|------------|------|-------|
| Laravel 13 + Inertia v3 + React 19 | Existing | Project foundation; pages live in `resources/js/pages/` |
| Tailwind v4 | Existing | For newspaper aesthetic, custom CSS variables for paper / terracotta palette |
| SQLite | Existing | Local DB; tables: `people`, `sprints`, `sprint_participants`, `habits`, `ticks`, `rewards` |
| Wayfinder | Existing | Type-safe route bindings for the React frontend |
| Pint, PHPUnit, Pest-free | Existing | Code style + tests |
| Laravel Fortify | Existing in project, **NOT used** | Auth is intentionally bypassed for this app — needs deciding whether to remove the dependency or just not register routes |
| Custom Google Fonts (Noto Serif SC, Fraunces, Inter, Noto Sans SC) | New | Already loaded via Google Fonts CDN in the reference HTML; replicate in app layout |

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Tablet runs the app offline / on a flaky network → ticks lost | Medium | Medium | Server is on local network (Herd), tablet is on same Wi-Fi. Acceptable for v1; consider service-worker offline cache later if real. |
| Mid-sprint edits create confusing data (deleting a habit erases ticks) | Medium | Medium | Soft-delete habits; archived habits keep their tick history but no longer appear in the grid. Document this clearly in the settings UI. |
| Surplus-points-after-achievement creates an "earned with no goal" gap | Low | High | Show clear "Pick next reward" CTA on celebration banner and on the progress bar. Surplus is preserved either way. |
| User changes a reward's cost mid-sprint after person already passed it | Low | Low | If new cost ≤ balance, immediately show as achieved. If new cost > balance, person is back below threshold (still saving). Acceptable. |
| Per-person flexible habit count breaks the symmetric grid layout | Medium | High | Use rowspan-style grouping per person (matches print). Grid handles uneven row counts naturally. |
| Anyone can tap as anyone (no auth) → child fakes a parent's tick | Low | Medium | Out of scope by design. Family trust is the model. A "who's tapping" PIN could be a future enhancement. |
| Single device dies → all data gone | Medium | Low | Standard SQLite backup script (cron / periodic copy of `database.sqlite`) — a tiny ops task. Could be documented in a follow-up. |

## Assumptions

- Single tablet on the local network, served by Laravel Herd at `http://family-habit-tracker.test`
- One sprint is active at a time (creating a new sprint while one is active either auto-ends the old one or is blocked — TBD in spec)
- "Today" is determined by the device's local date (the tablet's clock)
- Each tick is per (habit, day) — toggling re-toggles; no multi-tick-per-day
- Reward point costs are positive integers
- The family is okay with the "celebration" being a simple banner / overlay (no sound, no confetti unless trivially added)
- People can be re-ordered for grid display purposes (parents first, kids next, mirror the print)

## Unresolved Items

> Every item is tagged `[RESOLVED]` or `[DEFERRED]`.

- `[RESOLVED]` Goal model — **Decision:** Goal = the points cost of the active reward. 1 tick = 1 point. No separate sprint-target.
- `[RESOLVED]` Reward structure — **Decision:** Freeform name + integer point cost. One active reward per person at a time.
- `[RESOLVED]` Carry-forward semantics — **Decision:** Points balance and active reward both carry to the next sprint when goal isn't hit. Surplus after achievement rolls onto the next reward chosen.
- `[RESOLVED]` Sprint mechanics — **Decision:** Family-meeting bounded. Start/end dates. Manual "start new sprint" action. Last sprint's roster + habits pre-fill new sprint.
- `[RESOLVED]` Habits per person — **Decision:** Flexible count per person per sprint (no cap). Always 1pt per tick.
- `[RESOLVED]` People scope — **Decision:** People are global; habits are per-sprint.
- `[RESOLVED]` Retroactive ticks — **Decision:** Any past day in the current sprint is tappable. Future days locked.
- `[RESOLVED]` Mid-sprint edits — **Decision:** Anything is editable mid-sprint (people, habits, rewards). Soft-delete preserves tick history of removed habits.
- `[RESOLVED]` Reward achievement flow — **Decision:** Mark achieved + celebration banner. Person picks next reward whenever they want; surplus rolls.
- `[RESOLVED]` Visual aesthetic — **Decision:** Newspaper aesthetic from the reference HTML (paper bg, terracotta, Noto Serif SC, Fraunces italic, masthead).
- `[RESOLVED]` Bilingual fields — **Decision:** Single freeform name field per person and per habit; users type whatever language they prefer.
- `[RESOLVED]` Tablet orientation — **Decision:** Responsive — landscape shows full grid, portrait stacks per-person.
- `[RESOLVED]` Auth — **Decision:** No auth. Single shared tablet. Fortify is installed but unused for this app.
- `[RESOLVED]` Module name — **Decision:** `habit-tracker`.
- `[RESOLVED]` Additional views — **Decision:** Per-person progress bars on grid, settings page, sprint history page, lifetime totals page — all in MVP.
- `[DEFERRED]` Should starting a new sprint while one is active auto-end the old one, or block until the user explicitly ends it? — **Reason:** UX nuance better decided during spec/wireframe, not blocking idea-level scope. — **Owner / When:** decide in `/dev-spec-writer` phase.
- `[DEFERRED]` Where exactly to render the per-person progress bar (masthead, side rail, per-row chip, or a header strip) — **Reason:** Layout decision tied to the visual mockup. — **Owner / When:** decide in spec / first wireframe pass.
- `[DEFERRED]` Whether to retain the unused Fortify dependency or remove it — **Reason:** Tidy-up call, not feature-blocking; keeping it costs nothing. — **Owner / When:** decide in spec phase or post-MVP cleanup.
- `[DEFERRED]` Print-friendly export of a finished sprint to recreate the original paper sheet — **Reason:** Nice-to-have, not MVP. — **Owner / When:** post-MVP enhancement.
- `[DEFERRED]` Backup strategy for the SQLite file — **Reason:** Ops concern, not feature-blocking. — **Owner / When:** post-MVP follow-up.

## Related Existing Code

- `Users/el/Desktop/family habit tracker/Family Habit Tracker.html` — the print reference; lift the CSS for newspaper aesthetic, masthead structure, grid markup, weekend/Sunday markers, reward column layout
- `app/`, `resources/js/`, `routes/` — standard Laravel + Inertia layout; new pages go in `resources/js/pages/habit-tracker/` (or top-level — TBD in spec)
- `database/migrations/` — new migrations for `people`, `sprints`, `sprint_participants`, `habits`, `ticks`, `rewards` tables
- Default Laravel Fortify wiring — present but **not used** here (single-tablet, no-auth context)

---

## Next Step

Run `/dev-spec-writer habit-tracker` to create technical specifications.

---

# Spec-Writer Enrichment (2026-04-26)

> Appended after `/dev-spec-writer`. Resolves remaining technical decisions for the dev sub-agents.

## Decisions Made During Spec Writing

- **Sprint overlap:** Blocked. A new sprint cannot start while one is active — the user must explicitly end the current sprint first. Sprints never overlap and never auto-mutate.
- **Backfill scope:** Only the currently-active sprint. Once archived, a sprint's grid is read-only.
- **Progress bar location:** Header strip above the grid, one card per participant. Always visible.
- **Celebration UX:** Full-screen dismissable overlay when a tick pushes a person's balance ≥ their active reward's cost. Buttons: "Pick next reward" → settings, "Maybe later" → dismiss.
- **Default sprint length:** 28 days (matches the print). User can override.
- **Archived habits' ticks:** Keep counting toward balance and lifetime totals (already-earned points are real).
- **Lifetime totals scope:** Per person only.
- **Navigation:** Compact top-nav inside the masthead with 4 links: Tracker, History, Totals, Settings.
- **Sprint title:** Derived from start–end dates (e.g., "Apr 19 – May 16, 2026"). No separate name field.
- **Field bounds:** Person name max 50 chars; Habit name max 80 chars; Reward name max 80 chars; Reward cost is a positive integer (1–9999).
- **Sprint dates:** End date must be on or after start date. End date inclusive (last tick day = end date).
- **Soft-delete:** People and habits soft-delete (hidden from active config but preserved for history). Sprints and ticks are not soft-deleted (they only become read-only on sprint end).
- **Display ordering:** People and habits each have an integer `display_order` for grid sequencing; new entries append to end.
- **Server timezone:** Application runs at the device's configured timezone (set in `config/app.php`). "Today" is computed from the server clock (the tablet hits a local Herd server, so local time matches device).

## User Flows

### Flow: Daily check-in
1. Family member walks up to tablet (already on `/` showing the active sprint grid).
2. They find their row and today's column.
3. Tap the cell → tick appears (terracotta check). Tap again to undo.
4. Header strip updates that person's points balance and progress bar in real time.
5. If the tick pushed their balance ≥ active reward's cost → celebration overlay appears.

### Flow: Backfill a missed day
1. Family member opens the tracker.
2. Locates their row + the day they missed (any past day in the active sprint).
3. Taps the empty cell → tick appears.
4. Header strip recalculates immediately. Future cells remain locked (visually disabled).

### Flow: Reward achieved
1. A check-in pushes balance ≥ cost.
2. Full-screen overlay shows: person name, reward name, points earned, "Pick next reward" / "Maybe later".
3. The reward is recorded as `achieved_at = now`. Points equal to the cost are debited; remaining surplus stays in the balance.
4. If "Pick next reward" → goes to Settings → Rewards section, focused on this person's reward form.
5. If "Maybe later" → returns to grid; person now has no active reward; surplus accrues into a "pending" pool until they pick a new reward.

### Flow: Family meeting — start a new sprint
1. From Settings, family taps "Start new sprint" (only enabled if no active sprint).
2. Form pre-fills: start date = today; end date = today + 28 days; roster = participants from the most recent sprint; habits per person = copied from the most recent sprint; rewards = each person's currently-active reward (or carry-forward target).
3. Family edits any of: dates, roster (toggle people in/out), habits (add/remove/rename per person), reward (name + cost per person).
4. Submit → new sprint becomes active. Carry-forward balances seed the new sprint's per-person points balance.

### Flow: Family meeting — end the current sprint
1. From Settings, family taps "End sprint".
2. Confirmation dialog reminds: "Sprint becomes read-only. Unspent points will carry to your next sprint."
3. Confirm → sprint is archived (`ended_at = now`). For each participant: if their active reward was not achieved this sprint, the carry-forward state stores their current balance and active reward to seed the next sprint.

### Flow: Manage people (global)
1. Settings → People tab shows the family roster.
2. Add a person (name + display order). They appear globally available to opt into future sprints.
3. Edit person's name. Soft-delete a person (hidden from new sprints; existing history preserved).

### Flow: Browse history
1. Tap "History" in top-nav.
2. Index shows past sprints (date range, who participated, who hit their goal). Most recent first.
3. Tap a sprint → archived read-only grid with the same masthead and tick layout.

### Flow: Lifetime totals
1. Tap "Totals" in top-nav.
2. One card per person (sorted by display order). Each card shows: total points ever earned, total rewards achieved, count of sprints participated in.

## Data Concepts

- **Person**: A family member. Has a freeform name and a display order. Soft-deletable. Independent of any sprint — exists across the whole app's history.
- **Sprint**: A bounded period with a start date, end date, and lifecycle state (`active` or `archived`). At most one `active` sprint exists at a time. When archived, becomes read-only (ticks/edits locked).
- **Sprint Participant**: A person opting into a specific sprint. Holds the participant's per-sprint state: starting balance (carried forward from previous sprint), active reward at sprint start.
- **Habit**: A specific behavior tracked for one participant within one sprint. Belongs to (sprint × person). Has a freeform name and display order. Soft-deletable mid-sprint (becomes hidden in active grid; ticks preserved).
- **Tick**: A check-in record for one (habit, day) pair. Boolean — exists or doesn't. Tapping a cell toggles existence. Worth 1 point each.
- **Reward**: A goal a person is saving toward. Has a freeform name and integer point cost. Each person has at most one **active** (un-achieved, un-superseded) reward at a time. When `achieved_at` is set, the reward is final; the person can then create a new active reward.
- **Points Balance** (derived): For an active sprint participant, balance = (carried-forward seed) + (sum of their ticks in this sprint) − (cost of any rewards they've achieved during the sprint or earlier where surplus has not yet been spent on a new reward). Computed on read; never stored as a column.
- **Achieved Reward**: A reward whose `achieved_at` is set. Permanent record. Counts toward lifetime totals.
- **Pending Surplus**: When a reward is achieved and the person hasn't picked a new active reward, the remaining points accrue into the next reward they pick. Conceptually "balance with no goal" — derived, not stored.

## Edge Cases

- **No active sprint:** Tracker page shows a friendly empty state with a "Start your first sprint" CTA. History/Totals/Settings still accessible.
- **No people yet:** Settings → People shows an "Add your first family member" empty state. New-sprint button is disabled until at least one person exists.
- **Person has no active reward:** Their progress strip card shows "No reward set" with an "Add reward" link. Ticks still earn points; they accrue as pending surplus.
- **Reward cost lowered after person already passed it:** Achievement check runs on every cost-edit. If new cost ≤ balance, the reward is immediately marked achieved (with `achieved_at = now` and a note "marked on cost edit").
- **Person opts out of a sprint mid-way:** Not supported in v1 — once a participant is on a sprint, they stay. They can be added to the sprint mid-way (their habits start from the day they're added; carry-forward balance still seeds them).
- **Tick on a future day:** Server rejects with 422 — frontend must lock future cells visually. Today is the server-side current date.
- **Tick outside sprint bounds:** Server rejects.
- **Tick on archived sprint:** Server rejects.
- **End sprint while a person has no active reward:** Allowed. Their balance carries forward as pending surplus; next sprint they (or family) pick a reward.
- **Start sprint with end_date < start_date:** Validation error.
- **Start sprint while one is active:** Blocked at validation; UI hides the button.
- **Habit deleted mid-sprint:** Soft-delete. Existing ticks remain in points balance and lifetime totals. Habit row vanishes from active grid; reappears in history view.
- **Person deleted globally while they have ticks:** Soft-delete only. Past ticks preserved; person disappears from new-sprint roster picker but remains visible in history with their original name.
- **Reward name edited after achievement:** Allowed (it's a label). Reward cost edits after achievement are blocked.
- **Cost edited so high that previously-achieved reward is no longer ≥ cost:** Cost edits to achieved rewards are blocked entirely (history-integrity).
- **Two people simultaneously tapping the same cell on different rows:** No conflict — different (habit, day) primary keys. Inertia router handles each as a separate request.
- **Same cell tapped twice quickly:** Server toggles based on current state; debounce on the client; an idempotency safeguard via unique constraint on (habit_id, tick_date) prevents duplicate inserts.

## Infrastructure Needs

This is a **brand-new project** with no `docs/integration/*` skills yet. The infrastructure needs are:

- **Database (SQLite, existing default):** All persistence. No queues, no cache layer, no external services.
- **Inertia v3 + React 19 (existing):** SPA-style page navigation between Tracker / History / Totals / Settings.
- **Wayfinder (existing):** Type-safe route bindings between Laravel routes and React.
- **Tailwind v4 (existing):** Newspaper aesthetic implemented via custom CSS variables (paper, terracotta, ink hierarchy) layered onto Tailwind.
- **Google Fonts (CDN):** `Noto Serif SC`, `Fraunces`, `Inter`, `Noto Sans SC` loaded in the root layout, mirroring the print HTML.
- **No auth / no Fortify routes:** Fortify is installed by the starter but is not used by this app. Auth routes should not be registered for end users; any Fortify scaffolding pages remain inactive.
- **No notifications, no jobs, no queue, no realtime.** Single device, synchronous request/response is sufficient.
- **No money / serial / multi-tenant skills apply.**

