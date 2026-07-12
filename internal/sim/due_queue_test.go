package sim

import (
	"testing"

	"simworkshare/internal/domain"
)

func TestDueQueuePaysEachAmountOnce(t *testing.T) {
	queue := newDueQueue()
	if err := queue.add(2, 10); err != nil {
		t.Fatalf("add first amount: %v", err)
	}
	if err := queue.add(2, 5); err != nil {
		t.Fatalf("add second amount: %v", err)
	}
	if !domain.MoneyAlmostEqual(queue.outstandingBalance(), 15) {
		t.Fatalf("outstanding = %v, want 15", queue.outstandingBalance())
	}

	paid, err := queue.takeDue(2)
	if err != nil {
		t.Fatalf("take due: %v", err)
	}
	if paid != 15 || queue.outstandingBalance() != 0 {
		t.Fatalf("paid/outstanding = %v/%v, want 15/0", paid, queue.outstandingBalance())
	}
	paidAgain, err := queue.takeDue(2)
	if err != nil {
		t.Fatalf("take due again: %v", err)
	}
	if paidAgain != 0 {
		t.Fatalf("second payment = %v, want 0", paidAgain)
	}
}

func TestDueQueueDoesNotRoundTinyAmounts(t *testing.T) {
	queue := newDueQueue()
	const tiny domain.Money = 1e-9
	if err := queue.add(3, tiny); err != nil {
		t.Fatalf("add tiny amount: %v", err)
	}
	paid, err := queue.takeDue(3)
	if err != nil {
		t.Fatalf("take tiny amount: %v", err)
	}
	if paid != tiny {
		t.Fatalf("paid = %.12g, want %.12g", paid, tiny)
	}
}

func TestDueQueuePreservesSmallFutureAmountAcrossScales(t *testing.T) {
	queue := newDueQueue()
	if err := queue.add(1, 1e20); err != nil {
		t.Fatalf("add large amount: %v", err)
	}
	if err := queue.add(2, 1); err != nil {
		t.Fatalf("add small future amount: %v", err)
	}

	paid, err := queue.takeDue(1)
	if err != nil {
		t.Fatalf("take large due amount: %v", err)
	}
	if paid != 1e20 {
		t.Fatalf("large payment = %.12g, want 1e20", paid)
	}
	if outstanding := queue.outstandingBalance(); outstanding != 1 {
		t.Fatalf("outstanding after large payment = %.12g, want 1", outstanding)
	}

	paid, err = queue.takeDue(2)
	if err != nil {
		t.Fatalf("take small due amount: %v", err)
	}
	if paid != 1 || queue.outstandingBalance() != 0 {
		t.Fatalf("small payment/outstanding = %.12g/%.12g, want 1/0", paid, queue.outstandingBalance())
	}
}
