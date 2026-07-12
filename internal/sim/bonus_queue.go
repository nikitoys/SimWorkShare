package sim

import (
	"fmt"
	"math"
	"sort"

	"simworkshare/internal/domain"
)

type bonusDue struct {
	Gross      domain.Money
	PayrollTax domain.Money
}

func (d bonusDue) total() domain.Money {
	return d.Gross + d.PayrollTax
}

type bonusQueue struct {
	byMonth map[int]bonusDue
}

func newBonusQueue() *bonusQueue {
	return &bonusQueue{byMonth: make(map[int]bonusDue)}
}

func (q *bonusQueue) add(dueMonth int, amount bonusDue) error {
	if dueMonth < 1 {
		return fmt.Errorf("due month must be >= 1")
	}
	for _, item := range []struct {
		name  string
		value domain.Money
	}{
		{"gross", amount.Gross},
		{"payroll tax", amount.PayrollTax},
	} {
		if math.IsNaN(float64(item.value)) || math.IsInf(float64(item.value), 0) {
			return fmt.Errorf("%s amount is not finite", item.name)
		}
		if item.value < 0 {
			return fmt.Errorf("%s amount must be >= 0", item.name)
		}
	}
	if amount.Gross == 0 && amount.PayrollTax == 0 {
		return nil
	}

	current := q.byMonth[dueMonth]
	next := bonusDue{
		Gross:      current.Gross + amount.Gross,
		PayrollTax: current.PayrollTax + amount.PayrollTax,
	}
	if math.IsInf(float64(next.Gross), 0) || math.IsInf(float64(next.PayrollTax), 0) ||
		math.IsInf(float64(next.total()), 0) {
		return fmt.Errorf("queue balance overflow")
	}
	q.byMonth[dueMonth] = next
	return nil
}

func (q *bonusQueue) takeDue(month int) (bonusDue, error) {
	opening := q.outstandingBalance()
	amount := q.byMonth[month]
	if domain.MoneyLess(opening.Gross, amount.Gross) ||
		domain.MoneyLess(opening.PayrollTax, amount.PayrollTax) {
		return bonusDue{}, fmt.Errorf("due amount exceeds opening queue balance")
	}
	delete(q.byMonth, month)
	return amount, nil
}

func (q *bonusQueue) outstandingBalance() bonusDue {
	months := make([]int, 0, len(q.byMonth))
	for month := range q.byMonth {
		months = append(months, month)
	}
	sort.Ints(months)

	var outstanding bonusDue
	for _, month := range months {
		outstanding.Gross += q.byMonth[month].Gross
		outstanding.PayrollTax += q.byMonth[month].PayrollTax
	}
	return outstanding
}
