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
}

func TestClaim(t *testing.T) {
	var h Holding
	_, err := h.Claim(1, true)
	if !errors.Is(err, errNoBalance) {
		t.Fatalf("expected: %v but received: %v", errNoBalance, err)
	}

	h = Holding{
		total:  one,
		locked: decimal.NewFromFloat(0.5),
		free:   decimal.NewFromFloat(0.5),
	}

	_, err = h.Claim(.6, true)
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

func TestRelease_Holding(t *testing.T) {
	h := Holding{}
	err := h.Release(nil)
	if !errors.Is(err, errClaimIsNil) {
		t.Fatalf("expected: %v but received: %v", errClaimIsNil, err)
	}

	c := &Claim{}
	err = h.Release(c)
	if !errors.Is(err, errClaimInvalid) {
		t.Fatalf("expected: %v but received: %v", errClaimInvalid, err)
	}

	c.amount = decimal.NewFromFloat(1)
	err = h.Release(c)
	if !errors.Is(err, errUnableToReleaseClaim) {
		t.Fatalf("expected: %v but received: %v", errUnableToReleaseClaim, err)
	}

	h.claims = append(h.claims, c)
	h.verbose = true
	err = h.Release(c)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !h.free.Equal(decimal.NewFromFloat(1)) {
		t.Fatal("unexpected amount")
	}
}

func TestReleaseToPending_Holding(t *testing.T) {
	h := Holding{}
	err := h.ReleaseToPending(nil)
	if !errors.Is(err, errClaimIsNil) {
		t.Fatalf("expected: %v but received: %v", errClaimIsNil, err)
	}

	c := &Claim{}
	err = h.ReleaseToPending(c)
	if !errors.Is(err, errClaimInvalid) {
		t.Fatalf("expected: %v but received: %v", errClaimInvalid, err)
	}

	c.amount = decimal.NewFromFloat(1)
	err = h.ReleaseToPending(c)
	if !errors.Is(err, errUnableToReleaseClaim) {
		t.Fatalf("expected: %v but received: %v", errUnableToReleaseClaim, err)
	}

	h.claims = append(h.claims, c)
	h.verbose = true
	err = h.ReleaseToPending(c)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !h.pending.Equal(decimal.NewFromFloat(1)) {
		t.Fatal("unexpected amount")
	}
}

func TestAdjustByBalance(t *testing.T) {
	// addWithPending := &Holding{
	// 	total: decimal.NewFromFloat(1),
	// 	free:  decimal.NewFromFloat(1),
	// }
	// checkValues(addWithPending, 1, 0, 1, 0, 0, t)

	// err := addWithPending.adjustByBalance(0)
	// if !errors.Is(err, errAmountCannotBeZero) {
	// 	t.Fatalf("expected: %v but received: %v", errAmountCannotBeZero, err)
	// }

	// // Test with pending amounts - Add to holdings
	// c, err := addWithPending.Claim(.2, true) // execute internal claim on .2
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1, 0, .8, 0, .2, t)

	// err = c.ReleaseToPending() // simulate accepted market order in management
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1, 0, .8, .2, 0, t)

	// // simulate - in another market order an order to sell quote currency to increase
	// // this base currency balance
	// err = addWithPending.adjustByBalance(.2) // simulate an increase in balance from the exchange
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1, 0, 1, 0, 0, t)

	// c, err = addWithPending.Claim(.2, true) // execute internal claim on .2
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1, 0, .8, 0, .2, t)

	// err = c.ReleaseToPending() // simulate accepted market order in management
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1, 0, .8, .2, 0, t)

	// err = addWithPending.adjustByBalance(.2) // simulate an increase in balance from the exchange when order gets cancelled
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1, 0, 1, 0, 0, t)

	// c, err = addWithPending.Claim(.2, true) // execute internal claim on .2
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1, 0, .8, 0, .2, t)

	// err = c.ReleaseToPending() // simulate accepted market order in management
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1, 0, .8, .2, 0, t)

	// err = addWithPending.adjustByBalance(.4) // simulate an increase in balance from the exchange when order gets cancelled and another order executes and this balance increases.
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1.2, 0, 1.2, 0, 0, t)

	// c, err = addWithPending.Claim(.2, true) // execute internal claim on .2
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1.2, 0, 1, 0, .2, t)

	// err = c.ReleaseToPending() // simulate accepted market order in management
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1.2, 0, 1, .2, 0, t)

	// err = addWithPending.adjustByBalance(.1) // simulate reduce only order
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addWithPending, 1.2, 0, 1.1, .1, 0, t)

	//  Test without pending amounts add to holdings
	addNoPending := &Holding{
		total:  decimal.NewFromFloat(1),
		free:   decimal.NewFromFloat(.8),
		locked: decimal.NewFromFloat(.2), // Simulate an order already on the exchange when starting
	}
	checkValues(addNoPending, 1, .2, .8, 0, 0, t)

	err := addNoPending.adjustByBalance(.2) // Simulate order cancel
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(addNoPending, 1.2, 0, 1.2, 0, 0, t)

	addNoPending.free = decimal.NewFromFloat(1)
	addNoPending.locked = decimal.NewFromFloat(.2) // reset locked
	checkValues(addNoPending, 1.2, .2, 1, 0, 0, t)

	err = addNoPending.adjustByBalance(.3) // Simulate order cancel and another order being matched
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(addNoPending, 1.5, 0, 1.5, 0, 0, t)

	addNoPending.free = decimal.NewFromFloat(1.3)
	addNoPending.locked = decimal.NewFromFloat(.2) // reset locked
	checkValues(addNoPending, 1.5, .2, 1.3, 0, 0, t)

	err = addNoPending.adjustByBalance(.1) // Simulate partial cancel or reduce
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(addNoPending, 1.5, .1, 1.4, 0, 0, t)

	err = addNoPending.adjustByBalance(.05) // Simulate partial cancel or reduce
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(addNoPending, 1.5, .05, 1.45, 0, 0, t)

	err = addNoPending.adjustByBalance(0.15) // Simulate partial cancel or reduce
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(addNoPending, 1.6, 0, 1.6, 0, 0, t)

	// _, err = addNoPending.Claim(.2, true) // Claim but don't release
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addNoPending, 1.2, 0, 1, 0, .2, t)

	// err = addNoPending.adjustByBalance(.2)
	// if !errors.Is(err, nil) {
	// 	t.Fatalf("expected: %v but received: %v", nil, err)
	// }
	// checkValues(addNoPending, 1.4, 0, 1.2, 0, .2, t)
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
