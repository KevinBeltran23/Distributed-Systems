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
	
	// Raft-specific constants
	FOLLOWER  = 0
	CANDIDATE = 1
	LEADER    = 2
	
	MIN_ELECTION_TIMEOUT = 150 * time.Millisecond
	MAX_ELECTION_TIMEOUT = 300 * time.Millisecond
	VOTE_WAIT_TIMEOUT = 200 * time.Millisecond
	LEADER_HB_INTERVAL = 50 * time.Millisecond
)

type Node struct {
    ID            int
    Status        int
    Role          int       // FOLLOWER, CANDIDATE, or LEADER
    HBcount       int64
    Timestamp     time.Time
    Term          int       // Current term
    VotedFor      int      // ID of voted-for candidate
    CurrentLeader int      // Current leader's ID
}

type Args struct {
    ID int
    TableList []Node
}

type NodeServer struct {
	mu              sync.Mutex
	self            *Node        // Pointer to this server's own Node entry
	Table           []Node
	HBChannel       chan Args           // Regular gossip heartbeats
	LeaderHBChannel chan LeaderHeartbeat // Leader heartbeats for Raft
	VoteChannel     chan VoteRequest    // Incoming vote requests
	VoteResponses   chan VoteResponse   // Responses to vote requests
}

type LeaderHeartbeat struct {
	Term     int
	LeaderID int
}

type VoteRequest struct {
	CandidateID int
	Term        int
}

type VoteResponse struct {
	VoterID     int
	Term        int
	VoteGranted bool
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
		args := Args{ID: server.self.ID, TableList: server.Table}
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
		args := Args{ID: server.self.ID, TableList: server.Table}
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

//TODO
// I need to make this go on a channel just in case 
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

// Helper method for consistent log formatting
func (server *NodeServer) logRaft(format string, args ...interface{}) {
    roleStr := map[int]string{
        FOLLOWER:  "FOLLOWER",
        CANDIDATE: "CANDIDATE",
        LEADER:    "LEADER",
    }[server.self.Role]
    
    prefix := fmt.Sprintf("[Node %d][%s][Term %d] ", 
        server.self.ID, 
        roleStr, 
        server.self.Term)
    log.Printf(prefix+format, args...)
}

// Update membership table logging with Raft status
func logMembershipTable(server *NodeServer) {
    server.mu.Lock()
    defer server.mu.Unlock()

    roleStr := map[int]string{
        FOLLOWER:  "FOLLOWER",
        CANDIDATE: "CANDIDATE",
        LEADER:    "LEADER",
    }[server.self.Role]

    fmt.Printf("\n=== Node %d Status ===\n", server.self.ID)
    fmt.Printf("Role: %s\n", roleStr)
    fmt.Printf("Term: %d\n", server.self.Term)
    if server.self.CurrentLeader != -1 {
        fmt.Printf("Current Leader: %d\n", server.self.CurrentLeader)
    } else {
        fmt.Printf("No leader elected\n")
    }
    fmt.Printf("\n=== Membership Table ===\n")
    for _, value := range server.Table {
        fmt.Printf("Node %d: Status=%d, HBCount=%d, LastSeen=%s\n",
            value.ID,
            value.Status,
            value.HBcount,
            value.Timestamp.Format("15:04:05"))
    }
    fmt.Println("=====================\n")
}

// start elections if leader does not communicate
func startElectionTimeout(server *NodeServer) {
    for {
        // Random timeout between 150-300ms
        timeout := time.Duration(rand.Intn(150)+150) * time.Millisecond
        timer := time.NewTimer(timeout)

        select {
        case leaderHB := <-server.LeaderHBChannel:
            server.mu.Lock()
            if server.self.Role != LEADER && leaderHB.Term >= server.self.Term {
                server.self.Term = leaderHB.Term
                server.self.Role = FOLLOWER
                server.self.CurrentLeader = leaderHB.LeaderID
                server.self.VotedFor = -1
            }
            server.mu.Unlock()
            timer.Reset(timeout)

        case <-timer.C:
            server.mu.Lock()
            // Only followers should start elections
            if server.self.Role == FOLLOWER {
                server.logRaft("Starting election for term %d", server.self.Term+1)
                server.self.Term++
                server.self.Role = CANDIDATE
                server.self.VotedFor = server.self.ID
                server.self.CurrentLeader = -1
                go server.startElection()
            }
            server.mu.Unlock()
        }
    }
}

func (server *NodeServer) startElection() {
    server.mu.Lock()
    currentTerm := server.self.Term
    server.logRaft("Starting election for term %d", currentTerm+1)
    server.self.Term++
    server.self.Role = CANDIDATE
    server.self.VotedFor = server.self.ID
    server.self.CurrentLeader = -1
    server.mu.Unlock()

    votes := 1 // Vote for self
    votesNeeded := (len(server.Table) + 1) / 2 + 1
    
    // Create channels for vote collection
    voteResponses := make(chan VoteResponse, len(server.Table))
    timeout := time.After(VOTE_WAIT_TIMEOUT)

    // Request votes from all nodes
    for _, node := range server.Table {
        go func(peer int) {
            client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%d", peer))
            if err != nil {
                return
            }
            defer client.Close()

            request := VoteRequest{
                CandidateID: server.self.ID,
                Term:       currentTerm,
            }
            
            var response VoteResponse
            if err := client.Call("NodeServer.RequestVote", request, &response); err == nil {
                voteResponses <- response
            }
        }(node.ID)
    }

    // Collect votes
    for {
        select {
        case response := <-voteResponses:
            server.mu.Lock()
            // If we've seen a higher term, step down
            if response.Term > server.self.Term {
                server.self.Term = response.Term
                server.self.Role = FOLLOWER
                server.self.VotedFor = -1
                server.mu.Unlock()
                return
            }

            if response.VoteGranted {
                votes++
                if votes >= votesNeeded {
                    if server.self.Role == CANDIDATE {
                        log.Printf("Node %d: Won election for term %d with %d votes", 
                            server.self.ID, server.self.Term, votes)
                        server.self.Role = LEADER
                        server.self.CurrentLeader = server.self.ID
                        go server.startLeaderHeartbeat()
                    }
                    server.mu.Unlock()
                    return
                }
            }
            server.mu.Unlock()

        case <-timeout:
            server.mu.Lock()
            if server.self.Role == CANDIDATE {
                server.logRaft("Election timed out - reverting to follower")
                server.self.Role = FOLLOWER
            }
            server.mu.Unlock()
            return
        case <-server.LeaderHBChannel:
            server.logRaft("Received leader heartbeat during election - stepping down")
        }
    }
}

// respond to vote requests from candidates
func (server *NodeServer) RequestVote(request *VoteRequest, response *VoteResponse) error {
    server.mu.Lock()
    defer server.mu.Unlock()

    response.Term = server.self.Term
    response.VoterID = server.self.ID
    response.VoteGranted = false

    if request.Term < server.self.Term {
        return nil
    }

    if request.Term > server.self.Term {
        server.self.Term = request.Term
        server.self.Role = FOLLOWER
        server.self.VotedFor = -1
    }

    // Grant vote if we haven't voted in this term or already voted for this candidate
    if (server.self.VotedFor == -1 || server.self.VotedFor == request.CandidateID) && 
        request.Term >= server.self.Term {
        response.VoteGranted = true
        server.self.VotedFor = request.CandidateID
        server.self.Term = request.Term
    }

    return nil
}

// send the leader heartbeats to all other nodes
func (server *NodeServer) startLeaderHeartbeat() {
    ticker := time.NewTicker(LEADER_HB_INTERVAL)
    defer ticker.Stop()

    for range ticker.C {
        server.mu.Lock()
        if server.self.Role != LEADER {
            server.mu.Unlock()
            return
        }
        currentTerm := server.self.Term
        server.mu.Unlock()

        // Send heartbeat to all nodes
        for _, node := range server.Table {
            go func(nodeID int) {
                client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%d", nodeID))
                if err != nil {
                    return
                }
                defer client.Close()

                heartbeat := LeaderHeartbeat{
                    Term:     currentTerm,
                    LeaderID: server.self.ID,
                }
                
                var reply bool
                client.Call("NodeServer.ReceiveLeaderHeartbeat", heartbeat, &reply)
            }(node.ID)
        }
    }
}

// receive the heartbeats from the leader
func (server *NodeServer) ReceiveLeaderHeartbeat(heartbeat LeaderHeartbeat, reply *bool) error {
    server.mu.Lock()
    defer server.mu.Unlock()

    // If we're the leader, ignore heartbeats
    if server.self.Role == LEADER {
        *reply = true
        return nil
    }

    if heartbeat.Term < server.self.Term {
        *reply = false
        return nil
    }

    if heartbeat.Term > server.self.Term {
        server.self.Term = heartbeat.Term
        server.self.Role = FOLLOWER
        server.self.VotedFor = -1
    }

    server.self.CurrentLeader = heartbeat.LeaderID
    *reply = true

    // Forward to the heartbeat channel for election timeout reset
    select {
    case server.LeaderHBChannel <- heartbeat:
    default:
    }

    return nil
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
		self: &Node{ID: nodeID, Status: 1, HBcount: 0, Timestamp: time.Now()},
		Table: []Node{
			{ID: nodeID, Status: 1, HBcount: 0, Timestamp: time.Now()},
		},
		HBChannel:       make(chan Args, BUFFER),
		LeaderHBChannel: make(chan LeaderHeartbeat, BUFFER),
		VoteChannel:     make(chan VoteRequest, BUFFER),
		VoteResponses:   make(chan VoteResponse, BUFFER),
	}

	// Start all goroutines
	go createServer(port, server)
	go sendHeartbeat(server, peers)
	go sendTable(server, peers)
	go detectFailures(server)
	go processHeartbeats(server)
	go startElectionTimeout(server)

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