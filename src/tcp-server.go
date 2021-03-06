package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// Create a request based on the type
// TODO: Make pluggable
func createRequest(mti string) string {
	request := ""

	switch mti {
	case "0120":
		request = fmt.Sprintf("time=%s", time.Now().Format(time.RFC3339))
	}

	return request
}

// Send request to client
func sendRequest(c net.Conn, mti string) {
	connectionId := c.RemoteAddr().String()

	payload := createRequest(mti)
	request := fmt.Sprintf("%s:%s\n", mti, payload)

	log.Printf("SEND[%s] length=%d, request=%s", connectionId, len(request), request)
	c.Write([]byte(string(request)))
}

// Handle an MTI 820 by providing the response
// TODO: Make pluggable
func handleMTI820(payload string) string {
	return "0830:OK"
}

// Process a client request and prepare response
// TODO: Genericize known requests and dispatch for handling in a pluggable manner
func processRequest(mti string, payload string) string {
	response := "TODO"
	switch mti {
	case "0820":
		response = handleMTI820(payload)
	default:
		response = "0000:Unknown request"
	}

	return response
}

// Decode the client request and encode response
func handleRequest(req string) string {
	response := ""

	if len(req) > 0 {
		s := strings.Split(req, ":")
		if len(s) < 2 {
			response = "0001:Invalid request"
		} else {
			response = processRequest(s[0], s[1])
		}
	} else {
		response = "0000:Empty request"
	}

	return response
}

// Handle a new connection
func handleConnection(c net.Conn) {
	connectionId := c.RemoteAddr().String()
	log.Printf("ACCEPT[%s]\n", connectionId)

	// Prepare and start a timer to send requests to the client
	ticker := time.NewTicker(5000 * time.Millisecond)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// TODO Make request type pluggable
				sendRequest(c, "0120")
			}
		}
	}()

	// Read request from client
	for {
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			log.Printf("CLOSED[%s]\n", connectionId)
			ticker.Stop()
			done <- true
			break
		}

		// Process client request and send response
		request := strings.TrimSpace(string(netData))
		log.Printf("RECV[%s]: length=%d, request=%s\n", connectionId, len(request), request)
		response := handleRequest(request)
		response = fmt.Sprintf("%s\n", response)
		log.Printf("SEND[%s]: length=%d, response=%s", connectionId, len(response), response)

		c.Write([]byte(string(response)))
	}
	c.Close()
}

// Entry
func main() {
	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide a port number!")
		return
	}

	// Listenfor  connections on the designated port on all interfaces
	PORT := ":" + arguments[1]
	l, err := net.Listen("tcp4", PORT)
	if err != nil {
		log.Println(err)
		return
	}
	defer l.Close()

	// Indefinitely wait for connections and dispatch on a separate thread
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			return
		}
		go handleConnection(c)
	}
}
