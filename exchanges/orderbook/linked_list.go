package orderbook

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// linkedList defines a depth linked list
type linkedList struct {
	length int
	head   *Node
	tail   *Node
}

// Add adds a new node to the linked list
func (ll *linkedList) Add(node *Node) {
	fmt.Printf("Current Head: %p\n", ll.head)
	fmt.Printf("Current Tail: %p\n", ll.tail)
	if ll.head == nil {
		ll.head = node
		ll.tail = node
	} else {
		node.prev = ll.tail
		ll.tail.next = node
		ll.tail = node
	}

	fmt.Printf("After Head: %p\n", ll.head)
	fmt.Printf("After prev: %p\n", ll.head.prev)
	fmt.Printf("After next: %p\n", ll.head.next)
	fmt.Printf("After Tail: %p\n", ll.tail)
	fmt.Printf("After Tail prev: %p\n", ll.tail.prev)
	fmt.Printf("After Tail next: %p\n", ll.tail.next)
	fmt.Println()

	ll.length++
}

// RemoveByPrice removes depth level by price and returns the node to be pushed
// onto the stack
func (ll *linkedList) RemoveByPrice(price float64) (*Node, error) {
	tip := ll.head
	for tip != nil {
		if tip.value.Price == price {
			tip.prev.next = tip.next
			if tip.next != nil {
				tip.next.prev = tip.prev
			}
			return tip, nil
		}
		tip = tip.next
	}
	return nil, errors.New("not found cannot remove")
}

// Liquidity returns total depth liquitidy
func (ll *linkedList) Liquidity() (Liquidity float64) {
	tip := ll.head
	for tip != nil {
		Liquidity += tip.value.Amount
		tip = tip.next
	}
	return
}

// Value returns total value on price.amount on full depth
func (ll *linkedList) Value() (value float64) {
	tip := ll.head
	for tip != nil {
		value += tip.value.Amount * tip.value.Price
		tip = tip.next
	}
	return
}

// Display displays depth content
func (ll *linkedList) Display() {
	tip := ll.head
	for tip != nil {
		fmt.Printf("-> %+v ", tip.value)
		tip = tip.next
	}
	fmt.Println()
}

// Node defines a linked list node for an orderbook item
type Node struct {
	value Item
	next  *Node
	prev  *Node

	// Denotes time pushed to stack, this will influence cleanup routine when
	// there is a pause or minimal actions during period
	shelfed time.Time
}

// Depth defines a linked list of orderbook items
type Depth struct {
	ask linkedList
	bid linkedList

	// TODO: Determine performance of shared to bid/ask stack
	stack Stack
	sync.Mutex
}

// LenAsk returns length of asks
func (d *Depth) LenAsk() int {
	d.Lock()
	defer d.Unlock()
	return d.ask.length
}

// LenBids returns length of bids
func (d *Depth) LenBids() int {
	d.Lock()
	defer d.Unlock()
	return d.bid.length
}

// AddBid adds a bid to the list
func (d *Depth) AddBid(item Item) error {
	d.Lock()
	defer d.Unlock()
	n := d.stack.Pop()
	n.value = item
	d.bid.Add(n)
	return nil
}

// RemoveBidByPrice removes a bid
func (d *Depth) RemoveBidByPrice(price float64) error {
	d.Lock()
	defer d.Unlock()
	n, err := d.bid.RemoveByPrice(price)
	if err != nil {
		return err
	}
	d.stack.Push(n)
	return nil
}

// DisplayBids does a helpful display!!! YAY!
func (d *Depth) DisplayBids() {
	d.Lock()
	defer d.Unlock()
	d.bid.Display()
}

// Stack defines a FIFO list of reusable nodes
type Stack struct {
	nodes []*Node
	count int32
}

// NewStack returns a ptr to a new Stack instance
func NewStack() *Stack {
	return &Stack{}
}

// Push pushes a node pointer into the stack to be reused
func (s *Stack) Push(n *Node) {
	n.shelfed = time.Now()
	n.next = nil // cleanup
	s.nodes = append(s.nodes[:s.count], n)
	s.count++
}

// Pop returns the last pointer off the stack and reduces the count and if empty
// will produce a lovely fresh node
func (s *Stack) Pop() *Node {
	if s.count == 0 {
		// Create an empty node
		return &Node{}
	}
	s.count--
	return s.nodes[s.count]
}

// cleanupKrew test cleanup every 30 seconds
func (s *Stack) cleanupKrew() {
	t := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-t.C:
			for i := 0; i < len(s.nodes); i++ {
				if time.Since(s.nodes[i].shelfed) > time.Second*30 {
					copy(s.nodes[i:], s.nodes[i+1:])
					s.nodes[len(s.nodes)-1] = nil
					s.nodes = s.nodes[:len(s.nodes)-1]
					i--
				}
			}
		}
	}
}
