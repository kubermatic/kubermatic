package node

import (
	"sort"
)

type Node struct {
	children      map[uint64]*Node
	subscriptions map[string]subscriptionInfo
	shards        map[int64]string
	rand          func(int64) int64
}

type subscriptionInfo struct {
	deterministicRoutingCount int
	envelopes                 []SubscriptionEnvelope
}

type SubscriptionEnvelope struct {
	Subscription func(interface{})
	id           int64
	dName        string
}

func New(int63n func(n int64) int64) *Node {
	return &Node{
		children:      make(map[uint64]*Node),
		subscriptions: make(map[string]subscriptionInfo),
		shards:        make(map[int64]string),
		rand:          int63n,
	}
}

func (n *Node) AddChild(key uint64) *Node {
	if n == nil {
		return nil
	}

	if child, ok := n.children[key]; ok {
		return child
	}

	child := New(n.rand)
	n.children[key] = child
	return child
}

func (n *Node) FetchChild(key uint64) *Node {
	if n == nil {
		return nil
	}

	if child, ok := n.children[key]; ok {
		return child
	}

	return nil
}

func (n *Node) DeleteChild(key uint64) {
	if n == nil {
		return
	}

	delete(n.children, key)
}

func (n *Node) ChildLen() int {
	return len(n.children)
}

func (n *Node) AddSubscription(s func(interface{}), shardID, deterministicRoutingName string) int64 {
	if n == nil {
		return 0
	}

	id := n.createAndSetID(shardID)

	si := n.subscriptions[shardID]
	si.envelopes = append(si.envelopes, SubscriptionEnvelope{
		Subscription: s,
		id:           id,
		dName:        deterministicRoutingName,
	})
	if deterministicRoutingName != "" {
		si.deterministicRoutingCount++
	}

	sort.Sort(envelopes(si.envelopes))
	n.subscriptions[shardID] = si

	return id
}

func (n *Node) DeleteSubscription(id int64) {
	if n == nil {
		return
	}

	shardID, ok := n.shards[id]
	if !ok {
		return
	}

	delete(n.shards, id)

	s := n.subscriptions[shardID]
	for i, ss := range s.envelopes {
		if ss.id != id {
			continue
		}

		if ss.dName != "" {
			s.deterministicRoutingCount--
		}

		s.envelopes = append(s.envelopes[:i], s.envelopes[i+1:]...)
		break
	}

	n.subscriptions[shardID] = s

	if len(n.subscriptions[shardID].envelopes) == 0 {
		delete(n.subscriptions, shardID)
	}
}

func (n *Node) SubscriptionLen() int {
	return len(n.shards)
}

func (n *Node) ForEachSubscription(f func(shardID string, isDeterministic bool, s []SubscriptionEnvelope)) {
	if n == nil {
		return
	}

	for shardID, s := range n.subscriptions {
		f(shardID, s.deterministicRoutingCount > 0, s.envelopes)
	}
}

func (n *Node) createAndSetID(shardID string) int64 {
	id := n.rand(0x7FFFFFFFFFFFFFFF)
	for {
		if _, ok := n.shards[id]; ok {
			id++
			continue
		}
		n.shards[id] = shardID
		return id
	}
}

type envelopes []SubscriptionEnvelope

func (e envelopes) Len() int {
	return len(e)
}

func (e envelopes) Less(a, b int) bool {
	return e[a].dName < e[b].dName
}

func (e envelopes) Swap(a, b int) {
	tmp := e[a]
	e[a] = e[b]
	e[b] = tmp
}
