package main

import (
	"fmt"
	"lab3/shared"
	"net/rpc"
)

func main() {
	server, err := rpc.DialHTTP("tcp", "localhost:9005")
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}

	node := shared.Node{ID: 1, Hbcounter: 0, Time: 0, Alive: true}
	var reply shared.Node
	err = server.Call("Membership.Add", node, &reply)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Node added:", reply)
	}
}
