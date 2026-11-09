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
		"lifetime ticks for a person": `SELECT COUNT(*) FROM habits h
			JOIN ticks t ON t.habit_id = h.id WHERE h.person_id = 1`,
		"ticks for a person over some days": `SELECT t.habit_id, t.tick_date FROM habits h
			JOIN ticks t ON t.habit_id = h.id
			WHERE h.person_id = 1 AND t.tick_date >= '2026-01-01' AND t.tick_date < '2026-02-01'`,
		"ticks per person on a day": `SELECT h.person_id, COUNT(*) FROM ticks t
			JOIN habits h ON h.id = t.habit_id
			WHERE t.tick_date >= '2026-01-01' AND t.tick_date < '2026-01-02' AND h.person_id IS NOT NULL
			GROUP BY h.person_id`,
		"ticks by habit for a person": `SELECT h.id, COUNT(t.id) FROM habits h
			JOIN ticks t ON t.habit_id = h.id WHERE h.person_id = 1 GROUP BY h.id`,
		"habits for a person":        `SELECT id FROM habits WHERE person_id = 1 ORDER BY display_order ASC, id ASC`,
		"adjustments for a person":   `SELECT SUM(amount) FROM point_adjustments WHERE person_id = 1`,
		"has tick on a day":          `SELECT COUNT(*) FROM ticks WHERE habit_id = 1 AND date(tick_date) = '2026-01-01'`,
		"rewards spent":              `SELECT SUM(cost) FROM rewards WHERE person_id = 1 AND achieved_at IS NOT NULL`,
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
