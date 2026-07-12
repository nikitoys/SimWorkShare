package sim

import (
	"fmt"
	"math"
	"sort"
)

// AmountQueue is a deterministic due-month ledger used by AR, tax, member
// redemption and capacity activation. Entries are deleted when taken, so a
// due amount cannot be settled twice.
type AmountQueue struct {
	byMonth map[int]float64
}

func NewAmountQueue() *AmountQueue {
	return &AmountQueue{byMonth: make(map[int]float64)}
}

func (q *AmountQueue) Add(dueMonth int, amount float64) error {
	if dueMonth < 1 {
		return fmt.Errorf("due month must be >= 1")
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return fmt.Errorf("amount must be finite")
	}
	if amount < 0 {
		return fmt.Errorf("amount must be >= 0")
	}
	if amount == 0 {
		return nil
	}
	next := q.byMonth[dueMonth] + amount
	if math.IsInf(next, 0) {
		return fmt.Errorf("queue balance overflow")
	}
	q.byMonth[dueMonth] = next
	return nil
}

func (q *AmountQueue) TakeDue(month int) float64 {
	amount := q.byMonth[month]
	delete(q.byMonth, month)
	return amount
}

func (q *AmountQueue) Balance() float64 {
	months := make([]int, 0, len(q.byMonth))
	for month := range q.byMonth {
		months = append(months, month)
	}
	sort.Ints(months)
	var total float64
	for _, month := range months {
		total += q.byMonth[month]
	}
	return total
}

func (q *AmountQueue) DueMonths() []int {
	months := make([]int, 0, len(q.byMonth))
	for month := range q.byMonth {
		months = append(months, month)
	}
	sort.Ints(months)
	return months
}

type DistributionDue struct {
	Gross      float64
	PayrollTax float64
}

func (d DistributionDue) Total() float64 {
	return d.Gross + d.PayrollTax
}

func (d DistributionDue) Scale(factor float64) DistributionDue {
	return DistributionDue{Gross: d.Gross * factor, PayrollTax: d.PayrollTax * factor}
}

type DistributionQueue struct {
	byMonth map[int]DistributionDue
}

func NewDistributionQueue() *DistributionQueue {
	return &DistributionQueue{byMonth: make(map[int]DistributionDue)}
}

func (q *DistributionQueue) Add(dueMonth int, amount DistributionDue) error {
	if dueMonth < 1 {
		return fmt.Errorf("due month must be >= 1")
	}
	for name, value := range map[string]float64{"gross": amount.Gross, "payroll tax": amount.PayrollTax} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("%s must be finite", name)
		}
		if value < 0 {
			return fmt.Errorf("%s must be >= 0", name)
		}
	}
	if amount.Total() == 0 {
		return nil
	}
	current := q.byMonth[dueMonth]
	next := DistributionDue{Gross: current.Gross + amount.Gross, PayrollTax: current.PayrollTax + amount.PayrollTax}
	if math.IsInf(next.Gross, 0) || math.IsInf(next.PayrollTax, 0) || math.IsInf(next.Total(), 0) {
		return fmt.Errorf("queue balance overflow")
	}
	q.byMonth[dueMonth] = next
	return nil
}

func (q *DistributionQueue) TakeDue(month int) DistributionDue {
	amount := q.byMonth[month]
	delete(q.byMonth, month)
	return amount
}

func (q *DistributionQueue) Balance() DistributionDue {
	months := make([]int, 0, len(q.byMonth))
	for month := range q.byMonth {
		months = append(months, month)
	}
	sort.Ints(months)
	var total DistributionDue
	for _, month := range months {
		entry := q.byMonth[month]
		total.Gross += entry.Gross
		total.PayrollTax += entry.PayrollTax
	}
	return total
}
