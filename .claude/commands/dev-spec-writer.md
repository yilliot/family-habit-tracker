---
model: opus
allowed_tools: ["Bash", "Glob", "Grep", "Read", "Edit", "Write", "Agent", "AskUserQuestion"]
---

You are running the **Spec Writer** step of the 4-step dev workflow.

**Module / Domain:** $ARGUMENTS

---

## Your Job

1. **Enrich `idea.md`** — append technical clarifications (user flows, data concepts, edge cases, infrastructure needs) that brainstorm left open.
2. **Write 3 spec files** for the dev sub-agents, each telling them **WHAT** to build and **WHICH SKILLS to Read** before coding.

Output location: `docs/plan/{domain-slug}/`

```
docs/plan/{domain-slug}/
├── idea.md       # from brainstorm, enriched by you
├── spec-be.md    # backend spec   (for role-dev-backend)
├── spec-pl.md    # platform spec  (for role-dev-platform)
└── spec-fe.md    # frontend spec  (for role-dev-frontend)
```

---

## Non-Negotiable Rules

1. **No implementation details in specs.** No PHP, no JS, no SQL, no migration code, no component code. Behavior, contracts, and data shapes only.
2. **Every spec MUST start with `## Required Skills`** — a bullet list of file paths from `docs/dev/*` and `docs/integration/*` that the dev agent must Read before coding. Only list skills that truly apply to this spec.
3. **Infrastructure is integrated, not duplicated.** If `docs/integration/infra-*.md` already covers a need, spec the integration — never a new implementation.
4. **Dev agents are subagents. They cannot ask questions.** Anything they need must be in the spec. Clarify with the user up-front.
5. **Never hardcode tenant role names.** Roles like `school-admin` and `teacher` are conceptual — each tenant school uses its own custom role names (e.g., "Owner", "Academic Admin", "Finance"). Never name roles by string anywhere in the spec.
6. **Default: manual permission grant after deploy, not automated.** For new features with new permissions, the default rollout is: (a) `PermissionSeeder` (Setup tier) auto-creates permission rows per school on `php artisan release`, (b) each school admin manually grants the new permissions to whichever roles they want via the Role/Permission UI. **Do NOT author a Tier 2 release seeder** to auto-grant permissions to existing roles unless the user EXPLICITLY requests that behaviour during brainstorm. "Automatic grant to existing roles" is an opt-in optimisation — ask the user during spec-writer if unsure. Default is manual.
7. **Every house-related link, menu, and route MUST be permission-gated** (or for student/guardian portals, user-type + feature-flag-gated). A user whose role lacks the new permission must see zero UI entry points for the feature — no menu item, no button, no link, no accessible route. Spec must explicitly call out permission gating on every UI entry point and every route.

---

## Process

### 1. Read the idea
- `docs/plan/{domain-slug}/idea.md` is your primary input.
- Pay attention to **Existing System Context** — brainstorm already explored overlapping modules and patterns.
- If `idea.md` is missing, ask the user for the domain slug and suggest `/dev-brainstorm` first.

### 2. Load the skill map
- Read `docs/dev/index.md` to see every layer skill and its file-pattern trigger.
- Read `docs/integration/index.md` to see every infrastructure skill.

### 3. Enrich `idea.md` (append, do not rewrite)

Dev agents cannot ask questions. Resolve the open technical details now and append back into `idea.md`:

```markdown
## User Flows
### Flow: {name}
1. {step}
2. {step}

## Data Concepts
- **{Entity}**: {what it represents, states, relationships — conceptual, not schema}

## Edge Cases
- {boundary / null / conflict case}

## Infrastructure Needs
- {which `docs/integration/infra-*.md` apply and why}
```

Ask the user what's still unclear. Good starter questions:

| Layer | Ask about |
|-------|-----------|
| Backend | domain module home, integration points with other modules, background jobs, notifications |
| Platform | route naming (RESTful vs nested), authorization (roles/policies), Inertia props shape |
| Frontend | portal (school-admin/teacher/guardian/student), layout choice, existing components to reuse, page title, breadcrumb relation |

### 4. Define integration contracts
- What does Backend provide to Platform? (Repository methods, Factory methods)
- What does Platform provide to Frontend? (Inertia props, flash messages)
- What infrastructure does each layer integrate with?

### 5. Write the 3 specs (in this order)
1. `spec-be.md` — data foundation + infra integration
2. `spec-pl.md` — orchestration + auth + realtime
3. `spec-fe.md` — UI + i18n + display

### 6. Present for user review
Summarize each spec's scope and its `Required Skills` list. Wait for approval before suggesting `/dev-process`.

---

## Agent Scope Boundaries

### Backend (`spec-be.md`) — owns data and business logic
- Database schema (tables, columns, relationships)
- Business rules and validation logic
- Data transformation rules
- Background job requirements
- Notification requirements

**Receives from Platform:** entity objects, validated data
**Provides to Platform:** query results, mutation results

### Platform (`spec-pl.md`) — owns HTTP and orchestration
- API endpoints (routes, methods, naming)
- Request validation rules
- Response data structure
- Authorization (who can do what)
- Job dispatching triggers

**Receives from Frontend:** HTTP requests, form data
**Provides to Frontend:** Inertia props, flash messages

### Frontend (`spec-fe.md`) — owns UX
- Pages and layouts
- User interactions and flows
- Form fields and error messages
- Display formats (lists, details, badges)
- User feedback (toasts, confirmations)

**Receives from Platform:** props via Inertia
**Provides to Platform:** form submissions, navigation

---

## Spec Templates

### spec-be.md
```markdown
# {Feature} — Backend Specification

> Date: {YYYY-MM-DD}
> Idea: docs/plan/{slug}/idea.md

## Required Skills
> The dev agent MUST `Read` every file below BEFORE writing code.

 - this project is new and has no skills yet

## Problem Statement
{What data / business problem}

## Objective
{What data capabilities are needed}

## Scope
### In Scope
- …
### Out of Scope
- …

## Data Requirements
### Entity: {Name}
- Purpose:
- Fields: `field_name` (type): description, constraints
- Relationships:
- Business Rules:

### Queries Required
- {name}: what to retrieve, filters, sorting

### Mutations Required
- {name}: what changes, validation, state transitions

### Jobs Required (if any)
- {name}: trigger, what it does, completion behavior

### Notifications Required (if any)
- {name}: audience (Parent/Teacher/User), channels, trigger

## Integration Contract (with Platform)
### Provides to Platform
- Repository method `{name}` → {what data}
- Factory method `{name}` → {what mutation}

### Receives from Platform
- entity objects (already resolved)
- validated attribute arrays

## Infrastructure Integration
- [ ] Notification: {type and triggers}
- [ ] Job Queue: {async needs}
- [ ] Money: {currency handling}
- [ ] Serial: {serial number use}
- [ ] School Settings: {config}
- [ ] User Settings: {preferences}
- [ ] New PermissionName enum cases (declare only — Setup-tier `PermissionSeeder` auto-creates per school; role-granting is integrator scope)
- [ ] New SettingKey enum case for feature flag (declare only — Setup-tier `SettingSeeder` auto-seeds; portal-sharing is integrator scope)

## Acceptance Criteria
- [ ] …
```

### spec-pl.md
```markdown
# {Feature} — Platform Specification

> Date: {YYYY-MM-DD}
> Idea: docs/plan/{slug}/idea.md

## Required Skills

 - this project is new and has no skills yet

## Problem Statement
{What orchestration / API problem}

## Objective
{What endpoints / flows}

## Scope
### In Scope / Out of Scope
- …

## Endpoints
### {METHOD} {route name}
- Purpose:
- Authorization: {permission, feature flag}
- Request Input: `field` (type, required/optional): description
- Response Output: {props / JSON shape}
- Behavior: {ordered steps}
- Error Cases: {condition → response}

## Integration Contract
### With Backend
- Uses `{Repository}` for …
- Uses `{Factory}` for …

### With Frontend
- Provides Inertia props: `{name}` = {shape}
- Receives form: `{fields}` for {purpose}
- Flash messages: success / error

## Infrastructure Integration
- [ ] Roles/Permissions: {rules, per-portal gating strategy — every menu item, button, and route must be gated by `can('perm.name')` or, for student/guardian portals, by portal-middleware + feature-flag}
- [ ] Feature Flag: {toggle, per-school or global, default on/off}
- [ ] Realtime: {events}
- [ ] Permission rollout: **Manual grant by school admin post-deploy** (default). `PermissionSeeder` auto-creates the permission rows; school admins then tick the new permissions on the roles they want via Role/Permission UI.
- [ ] Tier 2 release seeder: **NOT required** unless brainstorm explicitly asked for automatic permission grant to existing roles. If required, describe the capability-based trigger here (e.g., "Grant to any role already holding `student.manage`") — NEVER by role-name string. Default: omit this row.

## Acceptance Criteria
- [ ] …
```

### spec-fe.md
```markdown
# {Feature} — Frontend Specification

> Date: {YYYY-MM-DD}
> Idea: docs/plan/{slug}/idea.md

## Required Skills

 - this project is new and has no skills yet


## Problem Statement
{What UX problem}

## Objective
{What UI is needed}

## Scope
### In Scope / Out of Scope
- …

## Pages
### {PageName} ({Index | Show | Create | Edit})
- Route: {named route}
- Container: {max-w-7xl | max-w-5xl | …}
- Purpose:
- Elements:
- User Actions:
- Feedback (success / error):
- Empty / loading / error states:

## Forms (if applicable)
### {FormName}
- Fields: `name` (input type): label, validation message
- Submit Behavior:
- Disabled states:

## Display / Copy Rules
- i18n: keys OR hardcoded English (per school-admin rule)
- Date format: `->format('…')` (timezone-aware)
- Money format: via `currency()` helper
- Name format: per `person-info.md`

## Integration Contract (with Platform)
### Receives from Platform
- prop `{name}`: {shape}

### Sends to Platform
- Form submission to `{route}` with: {fields}

## Infrastructure Integration
- [ ] i18n: {which keys}
- [ ] Money: {display rules}
- [ ] Realtime: {events to listen for}
- [ ] PDF: {download triggers}
- [ ] Excel: {import/export UI}

```

---

## Key Reminders

- **Enrich `idea.md` first** — specs build on top of the enriched idea.
- **Never assume** — if unclear, ask the user (AskUserQuestion).
- **Be specific** — vague specs produce wrong code.
- **Clear boundaries** — each agent knows exactly what to build.
- **Define contracts** — integration points must be explicit.
- **Include skills** — every spec lists required skills with `docs/dev/*` or `docs/integration/*` paths.
- **No code** — specs describe WHAT; skills guide HOW.

---

## Workflow Position

```
[1] /dev-brainstorm   → idea.md
[2] /dev-spec-writer  → idea.md (enriched) + spec-be/pl/fe.md   ← YOU ARE HERE
[3] /dev-process      → working code
[4] /dev-closure      → archive + skill lessons
```

When all 3 specs are approved, tell the user: `Next step: /dev-process {domain-slug}`.
