package main

import (
	"fmt"
	"lab3/shared"
	"math/rand"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	MAX_NODES  = 8
	X_TIME     = 1
	Y_TIME     = 2
	Z_TIME_MAX = 100
	Z_TIME_MIN = 10
)
var self_node shared.Node

// Send the current membership table to a neighboring node with the provided ID
func sendMessage(server rpc.Client, id int, membership shared.Membership) {
	//TODO
	var success bool
	request := shared.Request{ID: id, Table: membership} 
	err := server.Call("Requests.Add", request, &success)
	if err != nil {
		shared.LogInfo(id, "Failed to send message to Node %d: %v", id, err)
	} else {
		shared.LogInfo(id, "Successfully sent message to Node %d.", id)
	}
	
}

// Read incoming messages from other nodes
func readMessages(server rpc.Client, id int, membership shared.Membership) *shared.Membership {
	//TODO
	var received shared.Membership
	err := server.Call("Requests.Listen", id, &received)
	if err != nil {
		shared.LogInfo(id, "No messages received.")
		return nil		
	}
	shared.LogInfo(id, "Received updated membership table from Node %d.", id)
	shared.LogMembershipTable(id, &membership)
	return &received
	
}

func calcTime() float64 {
	//TODO
	return float64(time.Now().UnixNano()) / 1e9
}

var wg = &sync.WaitGroup{}

func main() {
	rand.Seed(time.Now().UnixNano())
	Z_TIME := rand.Intn(Z_TIME_MAX - Z_TIME_MIN) + Z_TIME_MIN

	// Connect to RPC server
	server, _ := rpc.DialHTTP("tcp", "localhost:9005")

	args := os.Args[1:]

	// Get ID from command line argument
	if len(args) == 0 {
		fmt.Println("No args given")
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("Found Error", err)
	}

	shared.LogInfo(id, "Starting. Will fail after %d seconds.", Z_TIME)

	currTime := calcTime()
	// Construct self
	self_node = shared.Node{ID: id, Hbcounter: 0, Time: currTime, Alive: true}
	var self_node_response shared.Node // Allocate space for a response to overwrite this

	// Add node with input ID
	if err := server.Call("Membership.Add", self_node, &self_node_response); err != nil {
		fmt.Println("Error:2 Membership.Add()", err)
	} else {
		fmt.Printf("Success: Node created with id= %d\n", id)
	}

	neighbors := self_node.InitializeNeighbors(id)
	fmt.Println("Neighbors:", neighbors)

	membership := shared.NewMembership()
	membership.Add(self_node, &self_node)

	sendMessage(*server, neighbors[0], *membership)

	// crashTime := self_node.CrashTime()

	time.AfterFunc(time.Second*X_TIME, func() { runAfterX(server, &self_node, &membership, id) })
	time.AfterFunc(time.Second*Y_TIME, func() { runAfterY(server, neighbors, &membership, id) })
	time.AfterFunc(time.Second*time.Duration(Z_TIME), func() { runAfterZ(server, id) })

	wg.Add(1)
	wg.Wait()
}

func runAfterX(server *rpc.Client, node *shared.Node, membership **shared.Membership, id int) {
	//TODO
	node.Hbcounter++
	node.Time = calcTime()
	var response shared.Node
	server.Call("Membership.Update", *node, &response)
	shared.LogInfo(id, "Heartbeat incremented to %d. Updated node timestamp.", node.Hbcounter)
}

func runAfterY(server *rpc.Client, neighbors [2]int, membership **shared.Membership, id int) {
	//TODO
	for _, neighbor := range neighbors {
		shared.LogInfo(id, "Sending membership table to neighbor Node %d.", neighbor)
		sendMessage(*server, neighbor, **membership)
	}	
}

func runAfterZ(server *rpc.Client, id int) {
	//TODO
	var node shared.Node
	server.Call("Membership.Get", id, &node)
	node.Alive = false
	var response shared.Node
	server.Call("Membership.Update", node, &response)
	shared.LogInfo(id, "Node marked as FAILED.")
}




func printMembership(m shared.Membership){
	for _, val := range m.Members {
		status := "is Alive"
		if !val.Alive {
			status = "is Dead"
		}
		fmt.Printf("Node %d has hb %d, time %.1f and %s\n", val.ID, val.Hbcounter, val.Time, status)
	}
	fmt.Println("")
}