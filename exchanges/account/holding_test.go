package account

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

var holdingTest = Holding{
	total:  one,
	locked: decimal.NewFromFloat(0.5),
	free:   decimal.NewFromFloat(0.5),
}

func TestGetTotal(t *testing.T) {
	if holdingTest.GetTotal() != 1 {
		t.Fatal("unexpected value")
	}
}

func TestGetLocked(t *testing.T) {
	if holdingTest.GetLocked() != .5 {
		t.Fatal("unexpected value")
	}
}

func TestPending(t *testing.T) {
	if holdingTest.GetPending() != 0 {
		t.Fatal("unexpected value")
	}
}

func TestGetFree(t *testing.T) {
	if holdingTest.GetFree() != .5 {
		t.Fatal("unexpected value")
	}
}

func TestValidateAmount(t *testing.T) {
	// err := holdingTest.ValidateAmount(10)
	// if !errors.Is(err, errAmountExceedsHoldings) {
	// 	t.Fatalf("expected: %v but received: %v", errAmountExceedsHoldings, err)
	// }
	// err = holdingTest.ValidateAmount(.5)
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// err = holdingTest.ValidateAmount(.45)
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
}

func TestClaim(t *testing.T) {
	h := Holding{
		total:  one,
		locked: decimal.NewFromFloat(0.5),
		free:   decimal.NewFromFloat(0.5),
	}

	_, err := h.Claim(.6, true)
	if !errors.Is(err, errAmountExceedsHoldings) {
		t.Fatalf("expected: %v but received: %v", errAmountExceedsHoldings, err)
	}

	c1, err := h.Claim(.1, false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if c1.GetAmount() != .1 {
		t.Fatal("unexpected amount")
	}

	c2, err := h.Claim(.5, false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if c2.GetAmount() != .4 {
		t.Fatal("unexpected amount")
	}

	if h.GetFree() != 0 {
		t.Fatal("unexpected amount")
	}

	err = c2.Release()
	if err != nil {
		t.Fatal(err)
	}

	if h.GetFree() != .4 {
		t.Fatal("unexpected amount")
	}

	c2, err = h.Claim(.5, false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	claimedAmount := c2.GetAmount()

	err = c2.ReleaseToPending()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if h.GetFree() != 0 {
		t.Fatal("unexpected value, should have been released to pending")
	}

	if h.GetPending() != claimedAmount {
		t.Fatal("unexpected value", h.GetPending())
	}
}

func TestSetAmounts(t *testing.T) {
	ten := decimal.NewFromInt(10)
	h := &Holding{}

	// Standard deployment
	h.setAmounts(ten, decimal.Zero)
	checkValues(h, 10, 0, 10, 0, 0, t)

	// Standard keep total but lock as if a limit order has been accepted
	// exchange front end
	h.setAmounts(ten, decimal.NewFromInt(1))
	checkValues(h, 10, 1, 9, 0, 0, t)

	// Standard reduce total and locked as if the limit order has been filled
	h.setAmounts(decimal.NewFromInt(9), decimal.Zero)
	checkValues(h, 9, 0, 9, 0, 0, t)

	// Algo system --- claim 1
	cl1, err := h.Claim(1, true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(h, 9, 0, 8, 0, 1, t)

	err = cl1.ReleaseToPending() // Successfully sent a limit order
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(h, 9, 0, 8, 1, 0, t)

	// reduce total by exchange acknowlaging the limit order sell
	h.setAmounts(decimal.NewFromInt(8), decimal.Zero)
	checkValues(h, 8, 0, 8, 0, 0, t)

	// claim another amount
	cl1, err = h.Claim(1, true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(h, 8, 0, 7, 0, 1, t)

	err = cl1.ReleaseToPending() // Successfully sent a limit order but only partial fill 50%
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(h, 8, 0, 7, 1, 0, t)

	h.setAmounts(decimal.NewFromFloat(7.5), decimal.NewFromFloat(.5)) // Sold .5 so reduce holdings to 7.5, .5 still on books so lock that.
	checkValues(h, 7.5, .5, 7, .5, 0, t)

	// claim another amount
	cl1, err = h.Claim(1, true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(h, 7.5, .5, 6, 0.5, 1, t)

	err = cl1.ReleaseToPending()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(h, 7.5, .5, 6, 1.5, 0, t)

	// partial fill of first claim and fill second claim
	h.setAmounts(decimal.NewFromFloat(6.25), decimal.NewFromFloat(.25))
	checkValues(h, 6.25, .25, 6, .25, 0, t)

	// claim a whole bunch of stuff
	cl1, err = h.Claim(6, true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(h, 6.25, .25, 0, .25, 6, t)
	// total limit orders matched
	h.setAmounts(decimal.NewFromFloat(6), decimal.Zero)
	checkValues(h, 6, 0, 0, 0, 6, t)

	err = cl1.ReleaseToPending()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(h, 6, 0, 0, 6, 0, t)

	// For when a change comes in but doesn't have all the limit order
	// information
	h.setAmounts(decimal.NewFromFloat(5), decimal.Zero) // 1 sold, 5 free because timing and response
	checkValues(h, 5, 0, 0, 5, 0, t)                    // Should still lock out everything and no free amount

	// err = cl1.Release()
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(h, 6, 0, 6, 0, 0, t)
}

func checkValues(h *Holding, total, locked, free, pending, claims float64, t *testing.T) {
	t.Helper()
	if h == nil {
		t.Fatal("holding is nil")
	}
	var e bool
	if h.GetTotal() != total {
		e = true
		t.Errorf("Total amount error - expected value: %f but received: %f",
			total,
			h.GetTotal())
	}
	if h.GetLocked() != locked {
		e = true
		t.Errorf("Locked amount error - expected value: %f but received: %f",
			locked,
			h.GetLocked())
	}
	if h.GetFree() != free {
		e = true
		t.Errorf("Free amount error - expected value: %f but received: %f",
			free,
			h.GetFree())
	}
	if h.GetPending() != pending {
		e = true
		t.Errorf("Pending amount error - expected value: %f but received: %f",
			pending,
			h.GetPending())
	}
	if h.GetTotalClaims() != claims {
		e = true
		t.Errorf("Claim amount error - expected value: %f but received: %f",
			claims,
			h.GetTotalClaims())
	}
	if e {
		t.Fatal("check values failed")
	}
}
