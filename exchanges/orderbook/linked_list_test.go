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

var bids = Items{
	Item{Price: 1336, Amount: 1},
	Item{Price: 1337, Amount: 1},
	Item{Price: 1338, Amount: 1},
	Item{Price: 1339, Amount: 1},
}

func TestLoad(t *testing.T) {
	a := bids
	d := Depth{}
	err := d.bid.Load(a, &d.stack)
	if err != nil {
		t.Fatal(err)
	}

	d.bid.Display()
	fmt.Println("Liguidity:", d.bid.Liquidity())
	fmt.Println("Value:", d.bid.Value())

	b := Items{
		Item{Price: 1336, Amount: 1},
		Item{Price: 1337, Amount: 1},
	}

	err = d.bid.Load(b, &d.stack)
	if err != nil {
		t.Fatal(err)
	}

	d.bid.Display()
	fmt.Println("Liguidity:", d.bid.Liquidity())
	fmt.Println("Value:", d.bid.Value())
}

//  158	   9521717 ns/op	 9600104 B/op	  100001 allocs/op
func BenchmarkWithoutStack(b *testing.B) {
	var n *Node
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			n = new(Node)
			n.value.Price = 1337
		}
	}
}

//  949	   1427820 ns/op	       0 B/op	       0 allocs/op
func BenchmarkWithStack(b *testing.B) {
	var n *Node
	stack := NewStack()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			n = stack.Pop()
			n.value.Price = 1337
			stack.Push(n)
		}
	}
}
