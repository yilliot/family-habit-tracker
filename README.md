# Family Habit Tracker · 家庭习惯表

A single-tablet family habit ledger. One binary, one SQLite file, no
JavaScript build step, no runtime dependencies.

It began as a Laravel + Inertia + React application and was rewritten in Go;
the database file carried across both moves without losing a tick.

## How it works

**1 tick = 1 point.** Each person has their own page, which opens on today: a
list of their habits, one big box each. A device remembers whose page it last
showed, so a child's own tablet opens straight onto their day; the names along
the top switch person in one tap.

- **Habits run until someone stops them.** Anyone can add one at the foot of a
  person's day; it counts from today. Stopping one takes it off the list and
  keeps its ticks and points. Restarting it later continues the same line in the
  history, because habits are matched by name.
- **Today and the six days before it can be ticked.** Older days open read-only:
  they show the habits that person had then, exactly as recorded.
- **Goals:** one at a time per person, leftover points roll on, and reaching one
  is celebrated on the tick that does it. Picking a goal you can already afford
  earns it on the spot.
- **History:** a month calendar per person, back to the first habit, with the
  month's ticks by habit, goals reached and deductions.

Nothing is behind a login. Anyone who walks up to the tablet can tap their own
ticks — the same trust model as the paper sheet on the fridge.

Until November 2026 the app ran in four-week sprints that a family meeting had
to start; between sprints nobody could tick. Sprints are gone. Their rows stay
in the database file as a record and nothing reads them.

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

`-tz` matters: it decides which day is today, and so which boxes are tappable. Set it to the family's
own timezone (`Asia/Singapore`), not the server's. Timestamps are still stored
in UTC; only the date boundary is local.

With `-tz Asia/Kuala_Lumpur` a tick at 6am lands on the right day rather than
on yesterday in UTC.

## Layout

```
cmd/server/          entry point, flags, graceful shutdown
internal/db/         SQLite open, schema steps (migrate.go), legacy date formats
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

### Schema steps and rolling back

`internal/db/migrate.go` lists the schema steps; the file records how far it has
come in SQLite's `user_version`, and every start runs whatever is missing, each
step in one transaction. Step 2 (November 2026) moved habits from sprints onto
people: each habit row got its owner and the days it ran, deductions got their
owner, and no tick moved.

A binary only opens files at or below its own version. To go back to an older
build after a step has run, restore the snapshot `make deploy` took before the
swap (`/var/lib/habit-tracker/pre-deploy/`), as under Restoring below. Snapshots
keep `user_version`, so restoring one never repeats a step.

The Laravel code lives on in this repository's history at the tag
`laravel-final`.

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

## Design notes

- **One person, one day per page.** A page is about 15 KB; the sprint grid it
  replaced was 356 KB, four people by 33 days of forms.
- **A tap swaps only its own row**, with the count, week, points, month and
  family strip updated alongside from the same fresh read. Taps on other rows
  queue behind it, so fast tapping through a list never loses one.
- **Chinese text uses the device's own fonts**; only Fraunces and Inter load.
  Static files are linked with a content hash and cached for a year.
- **Archiving a person keeps everything**, and they can be brought back.
- **The whole app works without JavaScript.** Every box is a real form and
  every mutation is a plain POST; htmx only upgrades a tap into an in-place
  swap. The box flips instantly and rolls back with an explanation if the
  server disagrees.
