package orderbook

import (
	"errors"
	"fmt"
	"sync"
)

// Node defines a linked list node for an orderbook item
type Node struct {
	value   Item
	next    *Node
	prev    *Node
	head    *Node
	headOpp *Node
	tail    *Node
}

func (n *Node) Next() *Node     { return n.next }
func (n *Node) Previous() *Node { return n.prev }
func (n *Node) Head() *Node     { return n.head }
func (n *Node) OppHead() *Node  { return n.headOpp }
func (n *Node) Tail() *Node     { return n.headOpp }

// Depth defines a linked list of orderbook items
type Depth struct {
	askLength, bidLength int
	headAsk, headBid     *Node
	stack                Stack
	sync.Mutex
}

// LenAsk returns length of asks
func (d *Depth) LenAsk() int {
	return d.askLength
}

// LenBids returns length of bids
func (d *Depth) LenBids() int {
	return d.bidLength
}

// AddBid adds a bid to the list
func (d *Depth) AddBid(item Item) error {
	n := d.stack.Pop()
	n.value = item
	n.headOpp = d.headAsk
	if d.headBid == nil {
		n.head = n
		d.headBid = n
		d.headBid.tail = n
		d.bidLength++
		return nil
	}

	n.head = d.headBid
	n.prev = d.headBid.tail
	d.headBid.tail.next = n
	d.headBid.tail = n
	return nil
}

// RemoveBidByPrice removes a bid
func (d *Depth) RemoveBidByPrice(price float64) error {
	if d.headBid == nil {
		return errors.New("head bids not found")
	}

	elem := d.headBid.Next()
	for elem != nil {
		if elem.value.Price == price {
			// Drop association
			elem.prev.next = elem.next
			// Push unused node onto stack
			d.stack.Push(elem)
			return nil
		}
		elem = elem.Next()
	}
	return errors.New("node not found")
}

// DisplayBids does a helpful display!!! YAY!
func (d *Depth) DisplayBids() {
	currentHead := d.headBid
	for currentHead != nil {
		fmt.Printf("%v ->", currentHead.value)
		currentHead = currentHead.next
	}
	fmt.Println()
}

// func (l *doublyLinkedList) PushBack(n *node) {
// 	if l.head == nil {
// 		l.head = n
// 		fmt.Println(l.head)
// 		l.tail = n
// 	} else {
// 		l.tail.next = n
// 		l.tail = n
// 	}
// 	l.length++
// }

// func (d *Depth) Delete(key int) {
// 	fmt.Println("deleting key:", key)
// 	fmt.Println(l.head)
// 	if l.head.value == key {
// 		l.head = l.head.next
// 		l.length--
// 		return
// 	}
// 	var prev *node
// 	curr := l.head
// 	for curr != nil && curr.value != key {
// 		prev = curr
// 		curr = curr.next
// 	}

// 	if curr == nil {
// 		fmt.Println("Key Not found")
// 	}
// 	prev.next = curr.next
// 	l.length--
// 	fmt.Println("node deleted")
// }

// Stack defines a FIFO list of empty nodes FIFO
type Stack struct {
	nodes []*Node
	count int
}

// NewStack returns a ptr to a new Stack instance
func NewStack() *Stack {
	return &Stack{}
}

// Push pushes a new node pointer into the stack
func (s *Stack) Push(n *Node) {
	// TODO: benchmark clean
	*n = Node{}
	s.nodes = append(s.nodes[:s.count], n)
	s.count++
}

// Pop returns the last pointer off the stack and reduces the count and if empty
// will produce a lovely fresh node
func (s *Stack) Pop() *Node {
	if s.count == 0 {
		// Create empty node
		return &Node{}
	}
	s.count--
	return s.nodes[s.count]
}
