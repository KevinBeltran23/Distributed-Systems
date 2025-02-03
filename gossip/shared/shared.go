package shared

import (
	"fmt"
	"math/rand"
	"time"
	"log"
)

const (
	MAX_NODES = 8
)

// Node struct represents a computing node.
type Node struct {
	ID        int
	Hbcounter int
	Time      float64
	Alive     bool
}

// Generate random crash time from 10-60 seconds
func (n Node) CrashTime() int {
	rand.Seed(time.Now().UnixNano())
	max := 60
	min := 10
	return rand.Intn(max-min) + min
}

func (n Node) InitializeNeighbors(id int) [2]int {
	neighbor1 := RandInt()
	for neighbor1 == id {
		neighbor1 = RandInt()
	}
	neighbor2 := RandInt()
	for neighbor1 == neighbor2 || neighbor2 == id {
		neighbor2 = RandInt()
	}
	return [2]int{neighbor1, neighbor2}
}

func RandInt() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(MAX_NODES-1+1) + 1
}

/*---------------*/

func LogInfo(nodeID int, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	log.Printf("[Node %d] %s - %s", nodeID, time.Now().Format("15:04:05.000"), msg)
}

/*---------------*/


// Membership struct represents participanting nodes
type Membership struct {
	Members map[int]Node
}

// Returns a new instance of a Membership (pointer).
func NewMembership() *Membership {
	return &Membership{
		Members: make(map[int]Node),
	}
}

// Adds a node to the membership list.
func (m *Membership) Add(payload Node, reply *Node) error {
	//TODO
	m.Members[payload.ID] = payload
	*reply = payload
	LogInfo(payload.ID, "Added to membership list. HB=%d, Time=%.2f.", payload.Hbcounter, payload.Time)
	return nil
}

// Updates a node in the membership list.
func (m *Membership) Update(payload Node, reply *Node) error {
	//TODO
	if node, exists := m.Members[payload.ID]; exists {
		node.Hbcounter = payload.Hbcounter
		node.Time = payload.Time
		m.Members[payload.ID] = node
		*reply = node
		return nil
	}
	LogInfo(payload.ID, "Updated. New HB=%d, Time=%.2f.", payload.Hbcounter, payload.Time)
	return nil	
}

// Returns a node with specific ID.
func (m *Membership) Get(payload int, reply *Node) error {
	//TODO
	if node, exists := m.Members[payload]; exists {
		*reply = node
		return nil
	}
	LogInfo(payload, "Node not found in membership list.")
	return fmt.Errorf("Node %d not found", payload)	
}

/*---------------*/

// Request struct represents a new message request to a client
type Request struct {
	ID    int
	Table Membership
}

// Requests struct represents pending message requests
type Requests struct {
	Pending map[int]Membership
}

// Returns a new instance of a Membership (pointer).
func NewRequests() *Requests {
	//TODO
	return &Requests{Pending: make(map[int]Membership)}
}

// Adds a new message request to the pending list
func (req *Requests) Add(payload Request, reply *bool) error {
	//TODO
	req.Pending[payload.ID] = payload.Table
	*reply = true
	return nil
}

// Listens to communication from neighboring nodes.
func (req *Requests) Listen(ID int, reply *Membership) error {
	//TODO
	if membership, exists := req.Pending[ID]; exists {
		*reply = membership
		LogInfo(ID, "Received Listen() request. Returning membership table.")
		delete(req.Pending, ID)
		return nil
	}
	return fmt.Errorf("No messages for node %d", ID)
}

func combineTables(table1 *Membership, table2 *Membership) *Membership {
	//TODO
	combined := NewMembership()
	for id, node := range table1.Members {
		combined.Members[id] = node
	}
	for id, node := range table2.Members {
		if existing, exists := combined.Members[id]; !exists || node.Time > existing.Time {
			combined.Members[id] = node
		}
	}
	return combined
}

