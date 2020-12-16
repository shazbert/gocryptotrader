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
	if ll.head == nil {
		ll.head = node
		ll.tail = node
	} else {
		node.prev = ll.tail
		ll.tail.next = node
		ll.tail = node
	}
	ll.length++
}

type byDecision func(Item) bool

// RemoveByPrice removes depth level by price and returns the node to be pushed
// onto the stack
func (ll *linkedList) Remove(fn byDecision) (*Node, error) {
	for tip := ll.head; tip != nil; tip = tip.next {
		if fn(tip.value) {
			if tip.prev == nil { // tip is at head
				ll.head = tip.next
				if tip.next != nil {
					tip.next.prev = nil
				}
				return tip, nil
			}
			if tip.next == nil { // tip is at tail
				ll.tail = tip.prev
				tip.prev.next = nil
				return tip, nil
			}
			// Split reference
			tip.prev.next = tip.next
			tip.next.prev = tip.prev
			return tip, nil
		}
	}
	return nil, errors.New("not found cannot remove")
}

// Liquidity returns total depth liquitidy
func (ll *linkedList) Liquidity() (Liquidity float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		Liquidity += tip.value.Amount
	}
	return
}

// Value returns total value on price.amount on full depth
func (ll *linkedList) Value() (value float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		value += tip.value.Amount * tip.value.Price
	}
	return
}

// Display displays depth content
func (ll *linkedList) Display() {
	for tip := ll.head; tip != nil; tip = tip.next {
		fmt.Printf("-> %+v ", tip.value)
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
	sync.Pool
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
	n, err := d.bid.Remove(func(i Item) bool { return i.Price == price })
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
	// fmt.Printf("Stack insert %+v ADDR: %p\n\n", n, n)
	*n = Node{shelfed: time.Now()} // purge and insert timing
	s.nodes = append(s.nodes[:s.count], n)
	s.count++
}

// Pop returns the last pointer off the stack and reduces the count and if empty
// will produce a lovely fresh node
func (s *Stack) Pop() *Node {
	if s.count == 0 {
		// Create an empty node
		// fmt.Println("Stack popped new ADDR")
		return &Node{}
	}
	s.count--
	// fmt.Printf("Stack popped ADDR %p\n", s.nodes[s.count])
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
