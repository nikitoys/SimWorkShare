package sim

import "testing"

func TestAmountQueuePaysOnceAndKeepsTail(t *testing.T) {
	queue := NewAmountQueue()
	if err := queue.Add(2, 10); err != nil {
		t.Fatal(err)
	}
	if err := queue.Add(2, 5); err != nil {
		t.Fatal(err)
	}
	if err := queue.Add(4, 7); err != nil {
		t.Fatal(err)
	}
	if got := queue.TakeDue(2); got != 15 {
		t.Fatalf("due = %g, want 15", got)
	}
	if got := queue.TakeDue(2); got != 0 {
		t.Fatalf("second due = %g, want 0", got)
	}
	if got := queue.Balance(); got != 7 {
		t.Fatalf("tail = %g, want 7", got)
	}
}

func TestDistributionQueuePreservesComponents(t *testing.T) {
	queue := NewDistributionQueue()
	if err := queue.Add(3, DistributionDue{Gross: 100, PayrollTax: 25}); err != nil {
		t.Fatal(err)
	}
	due := queue.TakeDue(3)
	if due.Gross != 100 || due.PayrollTax != 25 || queue.Balance().Total() != 0 {
		t.Fatalf("due/balance = %+v/%+v", due, queue.Balance())
	}
}
