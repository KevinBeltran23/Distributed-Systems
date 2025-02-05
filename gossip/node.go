package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
	"io"
)

const (
	BUFFER = 100
	MINARGS = 2
	TIMEOUT = 10 * time.Second
	INTERVAL = 2 * time.Second
)

type Node struct {
    ID int
    Status int
    HBcount int64
    Timestamp time.Time
}

type Args struct {
    ID int
    TableList []Node
}

type NodeServer struct{
	mu sync.Mutex
	ID int
	Table []Node
}

// random ID from 1000-9000 using the current time as a seed
func initID() int {
    rand.Seed(time.Now().UnixNano())  
    nodeID := rand.Intn(9000) + 1000  
    return nodeID
}

func createServer(port string, server *NodeServer){
    rError := rpc.RegisterName("NodeServer", server)
    if rError != nil {
        log.Fatal("ERROR in {Create Server} Failed to register: ", rError)
    }

    listener, lError := net.Listen("tcp", ":" + port)
    if lError != nil {
        log.Fatal("ERROR in {Create Server} Failed to listen: ", lError)
    }
    defer listener.Close()

    log.Println("Node listening on port: " + port)

    var wg sync.WaitGroup
    for {
        connection, cError := listener.Accept()
        if cError != nil {
            log.Printf("Error in {Create Server} Failed to Accept: %v\n", cError)
            continue
        }
		log.Println("Successfully accepted on port: " + port)
        wg.Add(1)
        go func() {
            defer wg.Done()
            rpc.ServeConn(connection)
            connection.Close()
        }()
    }
}

// Sends heartbeat messages to a random peer
func sendHeartbeat(server *NodeServer, peers []string) {
	for {
		time.Sleep(INTERVAL) // Heartbeat interval

		if len(peers) == 0 {
			continue
		}

		peerIndex := rand.Intn(len(peers))
		peer := peers[peerIndex]

		client, err := rpc.Dial("tcp", peer)
		if err != nil {
			log.Printf("Failed to connect to peer %s: %v", peer, err)
			continue
		}

		server.mu.Lock()
		args := Args{ID: server.ID, TableList: server.Table}
		server.mu.Unlock()

		var reply bool
		done := make(chan *rpc.Call, 1) // Buffered channel to avoid goroutine leak
		call := client.Go("NodeServer.ReceiveHeartbeat", args, &reply, done)

		select {
		case <-call.Done:
			if call.Error != nil {
				log.Println("RPC call failed:", call.Error)
			}
		case <-time.After(TIMEOUT):
			log.Println("RPC call timed out")
		}

		client.Close()
	}
}

// Receives heartbeats and updates the membership table
func (server *NodeServer) ReceiveHeartbeat(args Args, reply *bool) error {
	server.mu.Lock()
	defer server.mu.Unlock()

	log.Printf("Heartbeat received from Node %d", args.ID)

	// Update or add node in the table
	exists := false
	for i, entry := range server.Table {
		if entry.ID == args.ID {
			server.Table[i].HBcount++
			server.Table[i].Timestamp = time.Now()
			exists = true
			break
		}
	}

	// add to membership table if not already in
	if !exists {
		server.Table = append(server.Table, Node{
			ID:        args.ID,
			Status:    1,
			HBcount:   1,
			Timestamp: time.Now(),
		})
	}

	*reply = true
	return nil
}

//func updateNeighbors {/
//
//}

//func receiveTable {
//
//}

// Detects failed nodes based on timeout
func detectFailures(server *NodeServer) {
	for {
		time.Sleep(5 * time.Second) // Failure detection interval

		server.mu.Lock()
		now := time.Now()
		for i, entry := range server.Table {
			if now.Sub(entry.Timestamp) > TIMEOUT {
				log.Printf("Node %d marked as failed", entry.ID)
				server.Table[i].Status = 0 // Mark as failed
			}
		}
		server.mu.Unlock()
	}
}

func logMembershipTable(server *NodeServer) {
	server.mu.Lock()
	defer server.mu.Unlock()

	fmt.Println("\n\nTable List for Node: " + strconv.Itoa(server.ID))
	fmt.Println("-------------------------------------")
	for _, value := range server.Table {
		fmt.Printf("NodeID: %d\n", value.ID)
		fmt.Printf("Status: %d\n", value.Status)
		fmt.Printf("HBCount: %d\n", value.HBcount)
		fmt.Println("Timestamp: " + value.Timestamp.Format("15:04:05"))
		fmt.Println()
	}
	fmt.Println("-------------------------------------\n")
}

func main(){
	if len(os.Args) < MINARGS {
        log.Fatal("Usage: go run main.go <port> <other node ports ... >")
    }

	port := os.Args[1]
	peers := os.Args[2:]

	nodeID = initID()
	server := &NodeServer{
		ID: nodeID,
		Table: []Node{
			{ID: nodeID, Status: 1, HBcount: 0, Timestamp: time.Now()},
		},
	}

	go createServer(port, server)
	go sendHeartbeat(server, peers)
	//go detectFailures(server)
	for {
		time.Sleep(INTERVAL) 
		logMembershipTable(server)
	}
	select {}

}
