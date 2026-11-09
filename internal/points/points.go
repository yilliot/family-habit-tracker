// Package points derives every balance in the application. Nothing here is
// persisted: a balance is always recomputed from the ticks, rewards and manual
// adjustments that produced it.
package points

import (
	"context"

	"github.com/el/family-habit-tracker/internal/store"
)

// Calculator computes balances.
type Calculator struct {
	store *store.Store
}

// New builds a Calculator over a store.
func New(s *store.Store) *Calculator { return &Calculator{store: s} }

// For returns a Calculator bound to a different store, used to run a
// calculation inside an open transaction so it sees that transaction's writes.
func (c *Calculator) For(s *store.Store) *Calculator { return &Calculator{store: s} }

// Balance is what a person has to spend:
//
//	every point ever earned
//	  − the cost of every reward ever achieved
//	  + every manual adjustment
//
// While the app ran in sprints each sprint opened with this same figure carried
// in, so the number a person sees did not change when sprints were removed.
func (c *Calculator) Balance(ctx context.Context, personID int64) (int, error) {
	earned, err := c.store.TicksEarnedByPerson(ctx, personID)
	if err != nil {
		return 0, err
	}

	spent, err := c.store.SpentByPerson(ctx, personID)
	if err != nil {
		return 0, err
	}

	adjusted, err := c.store.AdjustmentTotalByPerson(ctx, personID)
	if err != nil {
		return 0, err
	}

	return earned - spent + adjusted, nil
}
