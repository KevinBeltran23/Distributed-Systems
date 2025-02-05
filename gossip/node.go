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

type NodeServer int64

var hbChannel = make(chan Args, BUFFER)


// random ID from 1000-9000 using the current time as a seed
func initID() int {
    rand.Seed(time.Now().UnixNano())  
    nodeID := rand.Intn(9000) + 1000  
    return nodeID
}

func createServer(port string){
    // create a server for this node
    server := new(NodeServer)
    rpc.Register(server)

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
    }
}

func detectFailures() {
    for {
        time.Sleep(5 * time.Second) // Failure detection interval
        
        now := time.Now()
        for i, entry := range table {
            if now.Sub(entry.Timestamp) > 10*time.Second { // Timeout threshold
                log.Printf("Node %d marked as failed", entry.NodeID)
                table[i].Status = 0 // Mark as failed
            }
        }
    }
}

// does the remote calling using rpc
func sendHeartbeat(nodeID int, peers []string, table []MembershipTable) {
    for {
        time.Sleep(2 * time.Second) 
        
        peerIndex := rand.Intn(len(peers))
        peer := peers[peerIndex]

        client, err := rpc.Dial("tcp", peer)
        if err != nil {
            log.Printf("Failed to connect to peer %s: %v", peer, err)
            continue
        }

        args := Args{NodeID: nodeID, HBTableList: table}
        var reply bool
        err = client.Call("NodeServer.ReceiveHeartbeat", args, &reply)
        if err != nil {
            log.Printf("RPC call failed: %v", err)
        }
        client.Close()
    }
}


// remotely called 
func (node *NodeServer) ReceiveHeartbeat(args *Args, reply *bool) error{
	log.Printf("Heartbeat received from Node %d", args.NodeID)

	hbChannel <- *args
    // Update membership table
    for i, entry := range table {
        if entry.NodeID == args.NodeID {
            table[i].HBcount++
            table[i].Timestamp = time.Now()
            return nil
        }
    }

    // If the node is new, add it to the table
    table = append(table, MembershipTable{
        NodeID:    args.NodeID,
        Status:    1, // Active
        HBcount:   1,
        Timestamp: time.Now(),
    })

	*reply = true
	return nil
}

func updateMembershipTable(){

}

func main(){
	if len(os.Args) < MINARGS {
        log.Fatal("Usage: go run main.go <port> <other node ports ... >")
    }

	port := os.Args[1]
	peers := os.Args[2:]

	go createServer(port)

    var membershipTable []Node = []Node{}
    mu := sync.Mutex{}

    // Create and initialize the current node in the membership table
	nodeID := initID()
    membershipTable = append(membershipTable, Node{
        ID:   nodeID,
        Status:   0,
        HBcount:  0,
        Timestamp: time.Now(),
    })

    go detectFailures()
	go sendHeartbeat(nodeID, peers, membershipTable)

}
