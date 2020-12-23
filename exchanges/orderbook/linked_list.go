package orderbook

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// var stack audit

// type audit []*Stack

// func (a *audit) Clean() {}

// func (a *audit) Register() *Stack {
// 	s := new(Stack)
// 	a = append(a)
// 	return s

// }

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
type outOfOrder func(float64, float64) bool

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

// Load iterates across new items and refreshes linked list
func (ll *linkedList) Load(items Items, stack *Stack) error {
	if ll.head == nil {
		ll.head = stack.Pop()
		ll.tail = ll.head
	}

	tip := &ll.head
	var prev *Node
	for i := 0; i < len(items); i++ {
		if *tip == nil {
			// Exceeded node length, pop from stack and reference to previous
			// node
			// fmt.Println("POP")
			n := stack.Pop()
			*tip = n
			prev.next = n
		}

		// fmt.Printf("Current ADDR: %p\n", *tip)
		// fmt.Printf("PREV: %+v\n", prev)

		(*tip).value = items[i] // Change node value
		(*tip).prev = prev

		prev = *tip
		ll.tail = *tip

		// fmt.Printf("current tail %p\n", ll.tail)

		tip = &(*tip).next
	}

	// Push unused nodes back onto stack
	for *tip != nil {
		// Prune reference to previous
		fmt.Printf("PUSH %+v\n", (*tip).prev)
		stack.Push(*tip)
		(*tip).prev.next = nil
		if (*tip).next == nil {
			break
		}
		tip = &(*tip).next
	}
	return nil
}

// Liquidity returns total depth liquidity
func (ll *linkedList) Liquidity() (liquidity float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		liquidity += tip.value.Amount
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

// Amount returns total depth liquidity and value
func (ll *linkedList) Amount() (liquidity, value float64) {
	for tip := ll.head; tip != nil; tip = tip.next {
		liquidity += tip.value.Amount
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

// Stack defines a FIFO list of reusable nodes
type Stack struct {
	nodes []*Node
	s     *uint32
	count int32
}

// NewStack returns a ptr to a new Stack instance
func NewStack() *Stack {
	return &Stack{}
}

// Push pushes a node pointer into the stack to be reused
func (s *Stack) Push(n *Node) {
	*n = Node{shelfed: time.Now()} // purge and insert timing
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
