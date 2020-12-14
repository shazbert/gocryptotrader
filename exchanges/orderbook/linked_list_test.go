package orderbook

import (
	"fmt"
	"testing"
)

func TestDepth(t *testing.T) {
	depth := Depth{}
	depth.AddBid(Item{Price: 1336, Amount: 1})
	depth.AddBid(Item{Price: 1337, Amount: 1})
	depth.AddBid(Item{Price: 1338, Amount: 1})
	depth.AddBid(Item{Price: 1339, Amount: 1})
	depth.DisplayBids()
	err := depth.RemoveBidByPrice(1338)
	if err != nil {
		t.Fatal(err)
	}

	depth.DisplayBids()
	fmt.Println("Stack PTR to reuse:", depth.stack.nodes)
	depth.AddBid(Item{Price: 1340, Amount: 1})

	depth.DisplayBids()
	fmt.Println("Stack PTR to reuse:", depth.stack.nodes)

	err = depth.RemoveBidByPrice(1340)
	if err != nil {
		t.Fatal(err)
	}

	depth.DisplayBids()
	fmt.Println("Stack PTR to reuse:", depth.stack.nodes)

	err = depth.AddBid(Item{Price: 1340, Amount: 1})
	if err != nil {
		t.Fatal(err)
	}

	depth.DisplayBids()
	fmt.Println("Stack PTR to reuse:", depth.stack.nodes)

	err = depth.RemoveBidByPrice(1340)
	if err != nil {
		t.Fatal(err)
	}
}
