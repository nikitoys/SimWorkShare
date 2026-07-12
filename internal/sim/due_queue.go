package sim

import (
	"fmt"
	"math"
	"sort"

	"simworkshare/internal/domain"
)

// dueQueue stores amounts by due month. Taking a due amount deletes it,
// preventing a second payment. The outstanding balance is recomputed in due
// month order so that a small future entry cannot be lost when a much larger
// earlier entry is removed from a cached float sum.
type dueQueue struct {
	byMonth map[int]domain.Money
}

func newDueQueue() *dueQueue {
	return &dueQueue{byMonth: make(map[int]domain.Money)}
}

func (q *dueQueue) add(dueMonth int, amount domain.Money) error {
	if dueMonth < 1 {
		return fmt.Errorf("due month must be >= 1")
	}
	if math.IsNaN(float64(amount)) || math.IsInf(float64(amount), 0) {
		return fmt.Errorf("amount is not finite")
	}
	if amount < 0 {
		return fmt.Errorf("amount must be >= 0")
	}
	if amount == 0 {
		return nil
	}
	newDue := q.byMonth[dueMonth] + amount
	if math.IsInf(float64(newDue), 0) {
		return fmt.Errorf("queue balance overflow")
	}
	q.byMonth[dueMonth] = newDue
	return nil
}

func (q *dueQueue) takeDue(month int) (domain.Money, error) {
	opening := q.outstandingBalance()
	amount := q.byMonth[month]
	if domain.MoneyLess(opening, amount) {
		return 0, fmt.Errorf("due amount exceeds opening queue balance")
	}
	delete(q.byMonth, month)
	return amount, nil
}

func (q *dueQueue) outstandingBalance() domain.Money {
	months := make([]int, 0, len(q.byMonth))
	for month := range q.byMonth {
		months = append(months, month)
	}
	sort.Ints(months)

	var outstanding domain.Money
	for _, month := range months {
		outstanding += q.byMonth[month]
	}
	return outstanding
}
