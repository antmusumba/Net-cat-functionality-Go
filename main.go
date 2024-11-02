package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func readArt(filename string) (art string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		art += scanner.Text() + "\n"
	}
	if err := scanner.Err(); err != nil {
		log.Println("Error reading from file:", err)
	}
	return
}

func displayArt(conn net.Conn) {
	art := readArt("file.txt")
	conn.Write([]byte(art))
}

type Client struct {
	conn net.Conn
	name string
}

var (
	clients      = make(map[net.Conn]*Client)
	msgHistory   []string
	historyMutex sync.Mutex
	clientsMutex sync.Mutex
)

func main() {
	port := "8989" // Default port
	if len(os.Args) == 2 {
		port = os.Args[1]
	} else if len(os.Args) > 2 {
		fmt.Println("[USAGE]: ./TCPChat $port")
		return
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Error starting TCP server: %v", err)
	}
	defer listener.Close()
	fmt.Printf("Chat server started on port %s...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	displayArt(conn)
	fmt.Fprint(conn, "ENTER YOUR NAME: ")
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	name := scanner.Text()

	client := &Client{conn: conn, name: name}
	registerClient(client)
	defer unregisterClient(client)

	for scanner.Scan() {
		message := scanner.Text()
		if message == "" {
			continue
		}

		// Check for the /name command
		if len(message) > 6 && message[:6] == "/name " {
			newName := message[6:]
			if newName != "" {
				clientsMutex.Lock()
				oldName := client.name
				client.name = newName
				clientsMutex.Unlock()

				notification := fmt.Sprintf("%s has changed their name to %s", oldName, newName)
				broadcast(notification, client)
				fmt.Fprintf(conn, "Your name has been changed to %s\n", newName)
			} else {
				fmt.Fprintln(conn, "Name change failed. New name cannot be empty.")
			}
			continue
		}

		// Standard message handling
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		formattedMessage := fmt.Sprintf("[%s][%s]: %s", timestamp, client.name, message)
		logMessage(formattedMessage)
		broadcast(formattedMessage, client)
	}

	if scanner.Err() != nil {
		log.Printf("Error reading from client %s: %v", client.name, scanner.Err())
	}
}

func registerClient(client *Client) {
	clientsMutex.Lock()
	clients[client.conn] = client
	clientsMutex.Unlock()

	welcomeMessage := fmt.Sprintf("%s has joined the chat", client.name)
	broadcast(welcomeMessage, client)
	sendHistory(client)
}

func unregisterClient(client *Client) {
	clientsMutex.Lock()
	delete(clients, client.conn)
	clientsMutex.Unlock()

	farewellMessage := fmt.Sprintf("%s has left the chat", client.name)
	broadcast(farewellMessage, client)
	client.conn.Close()
}

func logMessage(message string) {
	historyMutex.Lock()
	msgHistory = append(msgHistory, message)
	historyMutex.Unlock()
}

func sendHistory(client *Client) {
	historyMutex.Lock()
	for _, msg := range msgHistory {
		fmt.Fprintln(client.conn, msg)
	}
	historyMutex.Unlock()
}

func broadcast(message string, sender *Client) {
	clientsMutex.Lock()
	for _, client := range clients {
		if client != sender {
			fmt.Fprintln(client.conn, message)
		}
	}
	clientsMutex.Unlock()
}
