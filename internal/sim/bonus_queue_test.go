package sim

import "testing"

func TestBonusQueuePaysGrossAndPayrollTaxOnce(t *testing.T) {
	queue := newBonusQueue()
	if err := queue.add(2, bonusDue{Gross: 10, PayrollTax: 2.5}); err != nil {
		t.Fatalf("add bonus: %v", err)
	}
	paid, err := queue.takeDue(2)
	if err != nil {
		t.Fatalf("take due bonus: %v", err)
	}
	if paid.Gross != 10 || paid.PayrollTax != 2.5 || queue.outstandingBalance().total() != 0 {
		t.Fatalf("paid/outstanding = %+v/%+v", paid, queue.outstandingBalance())
	}
	paidAgain, err := queue.takeDue(2)
	if err != nil {
		t.Fatalf("take due bonus again: %v", err)
	}
	if paidAgain.total() != 0 {
		t.Fatalf("bonus was paid twice: %+v", paidAgain)
	}
}

func TestBonusQueuePreservesSmallFutureEntryAcrossScales(t *testing.T) {
	queue := newBonusQueue()
	if err := queue.add(1, bonusDue{Gross: 1e20, PayrollTax: 1e19}); err != nil {
		t.Fatalf("add large bonus: %v", err)
	}
	if err := queue.add(2, bonusDue{Gross: 1, PayrollTax: 0.25}); err != nil {
		t.Fatalf("add small future bonus: %v", err)
	}
	if _, err := queue.takeDue(1); err != nil {
		t.Fatalf("take large bonus: %v", err)
	}
	outstanding := queue.outstandingBalance()
	if outstanding.Gross != 1 || outstanding.PayrollTax != 0.25 {
		t.Fatalf("small future bonus was lost: %+v", outstanding)
	}
}
