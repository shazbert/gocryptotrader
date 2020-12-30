package orderbook

import (
	"errors"
	"fmt"
	"testing"
)

var bid = Items{
	Item{Price: 1336, Amount: 1},
	Item{Price: 1335, Amount: 1},
	Item{Price: 1334, Amount: 1},
	Item{Price: 1333, Amount: 1},
	Item{Price: 1332, Amount: 1},
	Item{Price: 1331, Amount: 1},
	Item{Price: 1330, Amount: 1},
	Item{Price: 1329, Amount: 1},
	Item{Price: 1328, Amount: 1},
	Item{Price: 1327, Amount: 1},
	Item{Price: 1326, Amount: 1},
	Item{Price: 1325, Amount: 1},
	Item{Price: 1324, Amount: 1},
	Item{Price: 1323, Amount: 1},
	Item{Price: 1322, Amount: 1},
	Item{Price: 1321, Amount: 1},
	Item{Price: 1320, Amount: 1},
	Item{Price: 1319, Amount: 1},
	Item{Price: 1318, Amount: 1},
	Item{Price: 1317, Amount: 1},
}

var ask = Items{
	Item{Price: 1337, Amount: 1},
	Item{Price: 1338, Amount: 1},
	Item{Price: 1339, Amount: 1},
	Item{Price: 1340, Amount: 1},
	Item{Price: 1341, Amount: 1},
	Item{Price: 1342, Amount: 1},
	Item{Price: 1343, Amount: 1},
	Item{Price: 1344, Amount: 1},
	Item{Price: 1345, Amount: 1},
	Item{Price: 1346, Amount: 1},
	Item{Price: 1347, Amount: 1},
	Item{Price: 1348, Amount: 1},
	Item{Price: 1349, Amount: 1},
	Item{Price: 1350, Amount: 1},
	Item{Price: 1351, Amount: 1},
	Item{Price: 1352, Amount: 1},
	Item{Price: 1353, Amount: 1},
	Item{Price: 1354, Amount: 1},
	Item{Price: 1355, Amount: 1},
	Item{Price: 1356, Amount: 1},
}

func TestLinkedListLoad(t *testing.T) {
	tests := []struct {
		Name   string
		Error  error
		Items  Items
		Change Items
		Stack  *Stack
	}{
		{
			Name:  "No Items",
			Error: errNoOrderbookItems,
		},
		{
			Name:  "No Stack",
			Error: errNoStack,
			Items: bid,
		},
		{
			Name:  "Load Bids",
			Error: nil,
			Items: bid,
		},
		{
			Name:  "Load Bids then load less items",
			Error: nil,
			Items: bid,
			Change: Items{
				Item{Price: 1337, Amount: 1},
				Item{Price: 1338, Amount: 1},
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		ll := linkedList{}
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			err := ll.Load(tt.Items, tt.Stack)
			if tt.Error != nil && !errors.Is(err, tt.Error) {
				t.Fatalf("expecting error %v but received %v", tt.Error, err)
			}

			items := ll.Retrieve()
			for x := range tt.Items {
				for y := range items {
					if tt.Items[x] != items[y] {
						t.Fatal("load functionality failure")
					}
				}
			}

			if tt.Change != nil {
				err = ll.Load(tt.Change, tt.Stack)
				if tt.Error != nil && !errors.Is(err, tt.Error) {
					t.Fatalf("expecting error %v but received %v", tt.Error, err)
				}
				items = ll.Retrieve()
				for x := range tt.Change {
					for y := range items {
						if tt.Change[x] != items[y] {
							t.Fatal("load functionality failure for change")
						}
					}
				}
			}
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		Name   string
		Error  error
		Items  Items
		Remove byDecision
		Stack  *Stack
	}{
		{
			Name:   "No Items",
			Error:  errNoOrderbookItems,
			Items:  bid,
			Remove: func(i Item) bool { return i.Price == 1336 },
			Stack:  &Stack{},
		},
	}

	for i := range tests {
		tt := tests[i]
		ll := linkedList{}
		err := ll.Load(tt.Items, tt.Stack)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			n, err := ll.Remove(tt.Remove, tt.Stack)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println(n)
		})
	}

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
