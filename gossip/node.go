package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	BUFFER = 100
	MINARGS = 2
	TIMEOUT = 10 * time.Second
	HBINTERVAL = 2 * time.Second
	TABLEINTERVAL = 7 * time.Second
	FAILTIME = 30 * time.Second
	DEATHTIME = 2 * FAILTIME
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
	HBChannel chan Args
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
		time.Sleep(HBINTERVAL) // Heartbeat interval

		if len(peers) == 0 {
			continue
		}

		peerIndex := rand.Intn(len(peers))
		peer := peers[peerIndex]
		peerAddress := "localhost:" + peers[peerIndex] // Ensure correct address format

		client, err := rpc.Dial("tcp", peerAddress)
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

// Receives heartbeats and enqueues them on to the channel for processing
func (server *NodeServer) ReceiveHeartbeat(args Args, reply *bool) error {
    select {
    case server.HBChannel <- args: 
        *reply = true
    default:
        log.Println("Heartbeat dropped due to full buffer")
        *reply = false
    }
    return nil
}

// updates the table with received heartbeats
func processHeartbeats(server *NodeServer) {
    for args := range server.HBChannel {
        server.mu.Lock()

        exists := false
        for i, entry := range server.Table {
            if entry.ID == args.ID {
                server.Table[i].HBcount++
                server.Table[i].Timestamp = time.Now()
                server.Table[i].Status = 1
                exists = true
                break
            }
        }
        
        if !exists {
            server.Table = append(server.Table, Node{
                ID:        args.ID,
                Status:    1,
                HBcount:   1,
                Timestamp: time.Now(),
            })
        }
        server.mu.Unlock()
    }
}


func sendTable(server *NodeServer, peers []string) {
	for {
		time.Sleep(TABLEINTERVAL) // Periodic table exchange interval

		if len(peers) == 0 {
			continue
		}

		numPeers := len(peers)
		numToSend := rand.Intn(numPeers) + 1 

		shuffledPeers := make([]string, len(peers))
		copy(shuffledPeers, peers)
		rand.Shuffle(len(shuffledPeers), func(i, j int) { shuffledPeers[i], shuffledPeers[j] = shuffledPeers[j], shuffledPeers[i] })

		selectedPeers := shuffledPeers[:numToSend]

		server.mu.Lock()
		args := Args{ID: server.ID, TableList: server.Table}
		server.mu.Unlock()

		for _, peer := range selectedPeers {
			peerAddress := "localhost:" + peer
			client, err := rpc.Dial("tcp", peerAddress)
			if err != nil {
				log.Printf("Failed to connect to peer %s: %v", peerAddress, err)
				continue
			}

			var reply bool
			done := make(chan *rpc.Call, 1)
			client.Go("NodeServer.ReceiveTable", args, &reply, done)

			go func() {
				select {
				case res := <-done:
					if res.Error != nil {
						log.Println("Table exchange RPC failed:", res.Error)
					} else {
						log.Printf("Successfully sent table to Node %s", peer)
					}
				case <-time.After(TIMEOUT):
					log.Println("Table exchange RPC timed out")
				}
				client.Close()
			}()
		}
	}
}

func (server *NodeServer) ReceiveTable(args Args, reply *bool) error {
	server.mu.Lock()
	defer server.mu.Unlock()
	newTable := []Node{}
	deadNode := false

	log.Printf("Received table from Node %d", args.ID)
	count := 0
	for _, incomingNode := range args.TableList {
		exists := false
		for i, existingNode := range server.Table {
			if existingNode.ID == incomingNode.ID {
				// Prioritize updates based on heartbeat count
				if incomingNode.HBcount > existingNode.HBcount ||
					(incomingNode.HBcount == existingNode.HBcount && incomingNode.Timestamp.After(existingNode.Timestamp)) {
					server.Table[i].Timestamp = incomingNode.Timestamp
					server.Table[i].HBcount = incomingNode.HBcount
				}
				exists = true
				break
			}
		}
		
		if incomingNode.Status == 0{
			deadNode = true
		}
		count += 1
		// If the node is not in the table, add it
		// Check for status code 0 to prevent bad nodes from being added back
		if (!exists && incomingNode.Status != 0){
			server.Table = append(server.Table, incomingNode)
		}
	}
	
	if deadNode == true{
		//remove the zeroed out Node(s) from the table
		for _, entry := range server.Table{
			if entry.Status == 1{
				newTable = append(newTable, entry)
			}
		}
		//update server table once loop is done
		server.Table = newTable
	}
	*reply = true
	return nil
}


// Detects failed nodes based on timeout
func detectFailures(server *NodeServer) {
    for {
        time.Sleep(FAILTIME) 

        server.mu.Lock()
        now := time.Now()
        newTable := []Node{}

        for _, entry := range server.Table {
            elapsed := now.Sub(entry.Timestamp)

            if elapsed > DEATHTIME {
                log.Printf("Node %d removed from table due to inactivity", entry.ID)
                continue // Skip adding this node to the new table
            }

            if elapsed > FAILTIME {
                log.Printf("Node %d marked as failed", entry.ID)
                entry.Status = 0 
            }

            newTable = append(newTable, entry)
        }

        server.Table = newTable // Update the table with only active nodes
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
	
	nodeID, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Invalid port number %s: %v", port, err)
	}

	if (os.Args[len(os.Args) - 1] == "bad"){
        //don't include last arg
        peers = os.Args[2:(len(os.Args) - 2)]
    }

	server := &NodeServer{
		ID: nodeID,
		Table: []Node{
			{ID: nodeID, Status: 1, HBcount: 0, Timestamp: time.Now()},
		},
		HBChannel: make(chan Args, BUFFER),
	}

	go createServer(port, server)
	go sendHeartbeat(server, peers)
	go sendTable(server, peers)
	go detectFailures(server)
	go processHeartbeats(server) 

	num_loops := 0
	rand.Seed(time.Now().UnixNano()) 
	FAILURE := rand.Intn(16) + 5

    for {
        time.Sleep(HBINTERVAL) 
        logMembershipTable(server)
        num_loops += 1
        if ((os.Args[len(os.Args) - 1] == "bad") && (num_loops > FAILURE)){
            fmt.Println("Node failing!")
            return
        }
    }

	select {}
}