-- Schema is compatible with the Laravel/Eloquent database it replaces. Every
-- table, column and foreign key matches the shape produced by the original
-- migrations, so an existing production database file opens unchanged and
-- `CREATE TABLE IF NOT EXISTS` is a no-op against it.
--
-- Two indexes are added that the original never had (marked below). An index is
-- invisible to anything reading the tables, so they change nothing about
-- compatibility; they are created on the first open of an inherited file.

CREATE TABLE IF NOT EXISTS "people" (
    "id"            integer primary key autoincrement not null,
    "name"          varchar not null,
    "display_order" integer not null default '0',
    "created_at"    datetime,
    "updated_at"    datetime,
    "deleted_at"    datetime
);
CREATE INDEX IF NOT EXISTS "people_display_order_index" on "people" ("display_order");
CREATE INDEX IF NOT EXISTS "people_deleted_at_index" on "people" ("deleted_at");

CREATE TABLE IF NOT EXISTS "sprints" (
    "id"         integer primary key autoincrement not null,
    "start_date" date not null,
    "end_date"   date not null,
    "status"     varchar not null default 'active',
    "started_at" datetime,
    "ended_at"   datetime,
    "created_at" datetime,
    "updated_at" datetime
);
CREATE INDEX IF NOT EXISTS "sprints_status_index" on "sprints" ("status");
CREATE INDEX IF NOT EXISTS "sprints_start_date_index" on "sprints" ("start_date");

CREATE TABLE IF NOT EXISTS "rewards" (
    "id"          integer primary key autoincrement not null,
    "person_id"   integer not null,
    "name"        varchar not null,
    "cost"        integer not null,
    "achieved_at" datetime,
    "created_at"  datetime,
    "updated_at"  datetime,
    foreign key("person_id") references "people"("id") on delete cascade
);
CREATE INDEX IF NOT EXISTS "rewards_person_id_achieved_at_index" on "rewards" ("person_id", "achieved_at");

CREATE TABLE IF NOT EXISTS "sprint_participants" (
    "id"                    integer primary key autoincrement not null,
    "sprint_id"             integer not null,
    "person_id"             integer not null,
    "carry_forward_balance" integer not null default '0',
    "active_reward_id"      integer,
    "display_order"         integer not null default '0',
    "created_at"            datetime,
    "updated_at"            datetime,
    foreign key("sprint_id")        references "sprints"("id") on delete cascade,
    foreign key("person_id")        references "people"("id")  on delete restrict,
    foreign key("active_reward_id") references "rewards"("id") on delete set null
);
CREATE UNIQUE INDEX IF NOT EXISTS "sprint_participants_sprint_id_person_id_unique" on "sprint_participants" ("sprint_id", "person_id");
CREATE INDEX IF NOT EXISTS "sprint_participants_display_order_index" on "sprint_participants" ("display_order");
-- Added. SQLite does not index foreign keys on its own, and every lifetime
-- balance joins through this column. Without it, each balance scanned every
-- enrolment ever recorded, so pages slowed as the years of history grew.
CREATE INDEX IF NOT EXISTS "sprint_participants_person_id_index" on "sprint_participants" ("person_id");

CREATE TABLE IF NOT EXISTS "habits" (
    "id"                    integer primary key autoincrement not null,
    "sprint_participant_id" integer,
    "name"                  varchar not null,
    "display_order"         integer not null default '0',
    "created_at"            datetime,
    "updated_at"            datetime,
    "deleted_at"            datetime,
    foreign key("sprint_participant_id") references "sprint_participants"("id") on delete set null
);
CREATE INDEX IF NOT EXISTS "habits_display_order_index" on "habits" ("display_order");
CREATE INDEX IF NOT EXISTS "habits_deleted_at_index" on "habits" ("deleted_at");
-- Added. Every grid render and every tap joins ticks to habits through this
-- column; without it each one scanned every habit ever created.
CREATE INDEX IF NOT EXISTS "habits_sprint_participant_id_index" on "habits" ("sprint_participant_id");

CREATE TABLE IF NOT EXISTS "ticks" (
    "id"         integer primary key autoincrement not null,
    "habit_id"   integer not null,
    "tick_date"  date not null,
    "created_at" datetime not null default CURRENT_TIMESTAMP,
    foreign key("habit_id") references "habits"("id") on delete cascade
);
CREATE UNIQUE INDEX IF NOT EXISTS "ticks_habit_id_tick_date_unique" on "ticks" ("habit_id", "tick_date");
CREATE INDEX IF NOT EXISTS "ticks_tick_date_index" on "ticks" ("tick_date");

CREATE TABLE IF NOT EXISTS "point_adjustments" (
    "id"                    integer primary key autoincrement not null,
    "sprint_participant_id" integer not null,
    "amount"                integer not null,
    "reason"                varchar,
    "created_at"            datetime not null default CURRENT_TIMESTAMP,
    foreign key("sprint_participant_id") references "sprint_participants"("id") on delete cascade
);
CREATE INDEX IF NOT EXISTS "point_adjustments_sprint_participant_id_index" on "point_adjustments" ("sprint_participant_id");
