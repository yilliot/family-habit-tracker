# Family Habit Tracker · 家庭习惯表

A single-tablet family habit ledger. One binary, one SQLite file, no
JavaScript build step, no runtime dependencies.

This is a Go rewrite of a Laravel + Inertia + React application. It keeps the
newspaper design and every rule of the original, reads the original's database
unchanged, and adds the things that were deferred in the first version.

## How it works

**1 tick = 1 point.** Each person has one reward with a point cost, and earns
toward it across as many sprints as it takes. A **sprint** is a date range the
family commits to at a family meeting; exactly one is active at a time. When a
reward is reached it is marked achieved, and any surplus rolls onto whatever
they pick next.

Nothing is behind a login. Anyone who walks up to the tablet can tap their own
ticks — the same trust model as the paper sheet on the fridge.

## Running it

```sh
make build                  # produces ./habit-tracker
./habit-tracker -seed       # optional: add 爸爸 / 妈妈 / 可遇 / 奇乐
./habit-tracker             # serves on 127.0.0.1:8080
```

| Flag           | Environment      | Default          | Meaning                                         |
| -------------- | ---------------- | ---------------- | ----------------------------------------------- |
| `-db`          | `HT_DB`          | `habits.sqlite`  | Database file; created if absent                |
| `-addr`        | `HT_ADDR`        | `127.0.0.1:8080` | Listen address                                  |
| `-tz`          | `HT_TZ`          | `Local`          | Timezone deciding which day is "today"          |
| `-backup-dir`  | `HT_BACKUP_DIR`  | off              | Directory for automatic daily snapshots         |
| `-backup-keep` | `HT_BACKUP_KEEP` | `14`             | How many daily snapshots to keep                |
| `-backup`      | —                | —                | Write one verified snapshot to a path, then exit |
| `-seed`        | —                | off              | Seed the family, then exit                      |

`-tz` matters: it decides which grid cells are tappable. Set it to the family's
own timezone (`Asia/Singapore`), not the server's. Timestamps are still stored
in UTC; only the date boundary is local.

The Laravel version ran on UTC. In Singapore that meant today's cells stayed
locked from midnight until 8am, because it was still yesterday in UTC. With
`-tz Asia/Singapore` a tick at 6am lands on the right day.

## Layout

```
cmd/server/          entry point, flags, graceful shutdown
internal/db/         SQLite open, schema, legacy date-format handling
internal/domain/     entities and validation rules
internal/store/      every SQL statement
internal/points/     balance calculations (never persisted)
internal/service/    business rules and read models
internal/web/        routing, request parsing, handlers
internal/web/views/  templ templates
internal/web/static/ CSS, JS, htmx, favicon — embedded into the binary
```

Rules live in `internal/service`. Handlers only parse and render; the store
only talks to SQLite. Balances are always derived at read time, so a corrected
tick or an edited reward cost can never leave a stale total behind.

## Development

```sh
make check   # gofmt, vet, and the full test suite
make run     # run against a local database
make dist    # static linux/amd64 and linux/arm64 binaries
```

Templates are `.templ` files compiled to Go. After editing one, run
`make generate` (every other target does it for you). The `templ` CLI is pinned
in `go.mod` as a tool dependency, so it never drifts from the committed output.

## Deployment

Production runs on the Axon EC2 at <https://family-habit-tracker.axon.commun.cloud>.
Everything the server needs is in `deploy/`:

```sh
make deploy   # check + dist, then on the server: snapshot, swap binary, restart, nginx
make status   # service state, public /healthz, newest snapshots
```

`make dist` produces a static binary with no libc dependency — the pure-Go
SQLite driver means there is no cgo and therefore no cross-compilation setup.
Always build locally: the server's Go is older than `go.mod` requires.

`deploy/install.sh` runs on the server and can be re-run safely. The settings at
its top — listen address `127.0.0.1:7778`, timezone `Asia/Kuala_Lumpur`, and
the site name — are the single source of truth: it renders
`deploy/habit-tracker.service` and `deploy/nginx.conf` from them. It takes a
verified snapshot before replacing a running binary, and only switches nginx
once the new process answers `/healthz`.

| What                        | Where on the server                      |
| --------------------------- | ---------------------------------------- |
| Binary                      | `/opt/habit-tracker/bin/habit-tracker`   |
| Database                    | `/var/lib/habit-tracker/habits.sqlite`   |
| Daily snapshots (last 14)   | `/var/lib/habit-tracker/backups/`        |
| Pre-deploy snapshots (10)   | `/var/lib/habit-tracker/pre-deploy/`     |
| Service and logs            | `habit-tracker.service`, `journalctl -u habit-tracker` |
| nginx site                  | `/etc/nginx/sites-available/family-habit-tracker.axon.commun.cloud` |

`GET /healthz` returns `ok` only if the database answers, and `503` otherwise,
so point an uptime monitor at it: it catches a wedged database as well as a
dead process.

### Migrating from the Laravel version

The schema is compatible: same tables, columns and foreign keys. Copy the old
database across and start the binary. Every `CREATE` is `IF NOT EXISTS`, so
existing data is left exactly as it was. The only change on first open is two
added indexes, which make no difference to anything reading the tables.

```sh
scp server:family-habit-tracker/database/database.sqlite ./habits.sqlite
./habit-tracker -db habits.sqlite
```

The Laravel code lives on in this repository's history at the tag
`laravel-final`.

Back up first. Reads tolerate every timestamp format Eloquent produced; writes
use the canonical one, so new rows are indistinguishable from old ones.

The Laravel `users` table and its auth columns are simply unused — the tracker
itself was always public, and the login only ever guarded starter-kit pages
that this version does not have.

## Backups

With `-backup-dir` set, the server keeps a snapshot of the database from each of
the last 14 days. Each one is written under a temporary name, checked with
SQLite's integrity check, flushed to disk, and only then renamed into place,
so every file in that directory is a complete, verified database. A snapshot
is taken once a day, judged from the files already there. Restarting the
service therefore does not take extra ones and push older days out.

Before a deploy or migration, take one by hand. It refuses to overwrite an
existing file:

```sh
./habit-tracker -db /var/lib/habit-tracker/habits.sqlite -backup ~/before-deploy.sqlite
```

**Snapshots on the same disk do not survive losing that disk.** They cover a bad
deploy, an accidental deletion, or a damaged file. They do not cover a failed
volume or a terminated instance. Copy them somewhere else as well, for
example with a nightly cron job (adapt the destination to what you use):

```sh
0 4 * * * aws s3 sync /var/lib/habit-tracker/backups s3://your-bucket/habit-tracker/
```

### Restoring

```sh
systemctl stop habit-tracker
cd /var/lib/habit-tracker
mv habits.sqlite habits.sqlite.broken
rm -f habits.sqlite-wal habits.sqlite-shm
cp backups/snapshot-20260915T030000Z.sqlite habits.sqlite
systemctl start habit-tracker
```

Remove the `-wal` and `-shm` files. They belong to the database being
replaced, not to the snapshot.

## Operating it for years

- **Keep an off-disk copy of the backups.** This is the only step here that
  the app cannot do for you.
- **Rebuild when Go ships a security release.** The binary is static, so
  fixes to Go's HTTP server reach it only through a rebuild and redeploy:
  `make dist`, then copy the binary across.
- **Monitor `/healthz`.** systemd restarts a crashed process but not one that
  is running and unable to serve.
- Growth is not a concern. A family of four adds roughly 400 KB a year, and a
  tap takes the same time in year twenty as in year one.

## Notes on the port

Everything the original did, this does. A few things changed deliberately:

- **Archiving a person no longer loses their points.** The original archived
  their habits and then deleted the sprint-participant row. Because that row is
  the only link from a tick back to a person, deleting it orphaned every point
  they had earned — despite a comment saying the opposite. The enrolment row is
  now kept and the person is filtered out of the grid instead.
- **Two indexes the original lacked.** SQLite does not index foreign keys, and
  every balance joins through `habits.sprint_participant_id` and
  `sprint_participants.person_id`. Without them each render scanned every
  habit ever created, so a tap slowed from 1ms to 6ms over 20 years of
  history. With them it stays under 1ms.
- **"No reward" now means no reward.** Choosing it used to silently re-link the
  person's existing goal.
- **Picking a goal you can already afford earns it immediately**, rather than
  waiting for the next unrelated tick to notice.
- **The settings tab lives in the URL**, so it can be linked to and bookmarked.
- **The person and habit columns stay pinned** while the days scroll, which is
  what makes a long sprint usable on a narrow screen.
- **A finished sprint prints** as a clean landscape ledger.
- **The whole app works without JavaScript.** Every cell is a real form and
  every mutation is a plain POST; htmx only upgrades a tap into an in-place
  swap. On the tablet the cell still flips instantly, and rolls back with an
  explanation if the server disagrees.
