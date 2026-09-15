package db_test

import (
	"strings"
	"testing"

	"github.com/el/family-habit-tracker/internal/db"
)

// The tables below grow for as long as the family uses the app. A query that
// scans one of them is fast in year one and slow in year twenty, and nothing
// else in the suite would notice. This test fails the moment a hot query stops
// using an index, whether because a query changed or an index was dropped.
//
// The statements mirror the ones in internal/store; keep them in step.
func TestHotQueriesUseIndexes(t *testing.T) {
	sqlDB, err := db.Open(t.Context(), t.TempDir()+"/plan.sqlite")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	queries := map[string]string{
		"lifetime ticks for a person": `SELECT COUNT(*) FROM ticks t
			JOIN habits h ON h.id = t.habit_id
			JOIN sprint_participants sp ON sp.id = h.sprint_participant_id
			WHERE sp.person_id = 1`,
		"sprint ticks for a participant": `SELECT COUNT(*) FROM ticks t
			JOIN habits h ON h.id = t.habit_id WHERE h.sprint_participant_id = 1`,
		"tick dates for a participant": `SELECT t.habit_id, t.tick_date FROM ticks t
			JOIN habits h ON h.id = t.habit_id
			WHERE h.sprint_participant_id = 1 AND h.deleted_at IS NULL`,
		"habits for a participant": `SELECT id FROM habits
			WHERE sprint_participant_id = 1 AND deleted_at IS NULL ORDER BY display_order ASC, id ASC`,
		"lifetime adjustments for a person": `SELECT SUM(pa.amount) FROM point_adjustments pa
			JOIN sprint_participants sp ON sp.id = pa.sprint_participant_id WHERE sp.person_id = 1`,
		"adjustments for a participant": `SELECT SUM(amount) FROM point_adjustments WHERE sprint_participant_id = 1`,
		"sprints a person joined":       `SELECT COUNT(*) FROM sprint_participants WHERE person_id = 1`,
		"participants in a sprint":      `SELECT id FROM sprint_participants WHERE sprint_id = 1 ORDER BY display_order ASC, id ASC`,
		"has tick on a day":             `SELECT COUNT(*) FROM ticks WHERE habit_id = 1 AND date(tick_date) = '2026-01-01'`,
		"rewards spent since": `SELECT SUM(cost) FROM rewards WHERE person_id = 1
			AND achieved_at IS NOT NULL AND datetime(achieved_at) >= datetime('2026-01-01 00:00:00')`,
		"active reward for a person": `SELECT id FROM rewards WHERE person_id = 1 AND achieved_at IS NULL ORDER BY id ASC LIMIT 1`,
	}

	for name, query := range queries {
		rows, err := sqlDB.QueryContext(t.Context(), "EXPLAIN QUERY PLAN "+query)
		if err != nil {
			t.Fatalf("%s: explain: %v", name, err)
		}

		var plan []string
		for rows.Next() {
			var id, parent, unused int
			var detail string

			if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
				t.Fatalf("%s: scan plan: %v", name, err)
			}

			plan = append(plan, detail)
		}
		rows.Close()

		for _, step := range plan {
			if strings.HasPrefix(step, "SCAN") {
				t.Errorf("%s scans a growing table:\n  %s", name, strings.Join(plan, "\n  "))

				break
			}
		}
	}
}
