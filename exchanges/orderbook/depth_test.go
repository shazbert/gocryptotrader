package orderbook

import (
	"fmt"
	"testing"
)

func TestDepth(t *testing.T) {
	depth := Depth{}
	depth.AddBid(Item{Price: 1336, Amount: 1})
	depth.DisplayBids()
	err := depth.RemoveBidByPrice(1336)
	if err != nil {
		t.Fatal(err)
	}
	depth.DisplayBids()

	depth.AddBid(Item{Price: 1336, Amount: 1})
	depth.DisplayBids()
	err = depth.RemoveBidByPrice(1336)
	if err != nil {
		t.Fatal(err)
	}
	depth.DisplayBids()

	depth.AddBid(Item{Price: 1337, Amount: 1})
	depth.AddBid(Item{Price: 1338, Amount: 1})
	depth.AddBid(Item{Price: 1339, Amount: .3444})

	depth.DisplayBids()

	err = depth.RemoveBidByPrice(1337)
	if err != nil {
		t.Fatal(err)
	}
	depth.DisplayBids()
	err = depth.RemoveBidByPrice(1338)
	if err != nil {
		t.Fatal(err)
	}

	depth.DisplayBids()
	// fmt.Println("Stack PTR to reuse:", depth.stack.nodes)

	depth.AddBid(Item{Price: 1340, Amount: 1})
	depth.DisplayBids()

	fmt.Println("Stack PTR to reuse:", depth.stack.nodes)

	err = depth.RemoveBidByPrice(1340)
	if err != nil {
		t.Fatal(err)
	}

	// depth.DisplayBids()
	// fmt.Println("Stack PTR to reuse:", depth.stack.nodes)

	err = depth.AddBid(Item{Price: 1340, Amount: .5})
	if err != nil {
		t.Fatal(err)
	}

	err = depth.RemoveBidByPrice(1340)
	if err != nil {
		t.Fatal(err)
	}

	depth.AddBid(Item{Price: 1337, Amount: 1})
	depth.AddBid(Item{Price: 1338, Amount: 1})

	err = depth.RemoveBidByPrice(1337)
	if err != nil {
		t.Fatal(err)
	}

	depth.DisplayBids()
	fmt.Println("Stack PTR to reuse:", depth.stack.nodes)

	fmt.Println("Liquidity Bids:", depth.bid.Liquidity())
	fmt.Println("Value Bids:", depth.bid.Value())

}
