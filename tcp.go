package main

import (
    "bytes"
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
    "github.com/0xAX/notificator"
    "strings"
    "os/exec"
)

var notify *notificator.Notificator


type ClientManager struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	socket net.Conn
	data   chan []byte
}

func (manager *ClientManager) start(notify *notificator.Notificator) {
	for {
		select {
		case connection := <-manager.register:
			manager.clients[connection] = true
            textContent := "A new connection"
            notification := fmt.Sprintf("display notification \"%s\" with title \"Hi\"", textContent)
            err := exec.Command("osascript", "-e", notification).Run()
            if err != nil {
                fmt.Println(err)
                return
            }
			fmt.Println("Added new connection!")
            if err != nil {
                fmt.Println(err)
                return
            }
		case connection := <-manager.unregister:
			if _, ok := manager.clients[connection]; ok {
				close(connection.data)
				delete(manager.clients, connection)
				fmt.Println("A connection has terminated!")
			}
		case message := <-manager.broadcast:
			for connection := range manager.clients {
				select {
				case connection.data <- message:
				default:
					close(connection.data)
					delete(manager.clients, connection)
				}
			}
		}
	}
}

func (manager *ClientManager) receive(client *Client, notify *notificator.Notificator) {
	for {
		message := make([]byte, 4096)
		length, err := client.socket.Read(message)
		if err != nil {
			manager.unregister <- client
			client.socket.Close()
			break
		}
		if length > 0 {

            //textContent := strings.TrimSpace(string(message))

            //fmt.Println(">>>"+textContent+"<<<")
            b := bytes.Trim(message, "\x00")
            fmt.Printf("%x", b)
            textContent := string(b)
            notification := fmt.Sprintf("display notification \"%s\" with title \"Hi\"", textContent)

            //args := []string{string(message)}
            //cmd := "ls"
            //flag := "-ld"
            //args := string(message)
           // command := exec.Command(cmd, flag, args[0])
           // fmt.Println(command.Stdout)
           // err := command.Run()
            err := exec.Command("osascript", "-e", notification).Run()

            if err != nil {
                fmt.Println("Error", err)
            //    fmt.Println("Argument ", notification)
                return
            }
			fmt.Println("RECEIVED: " + string(message))
			manager.broadcast <- message
		}
	}
}

func (manager *ClientManager) send(client *Client) {
	defer client.socket.Close()
	for {
		select {
		case message, ok := <-client.data:
			if !ok {
				return
			}
			client.socket.Write(message)
		}
	}
}

func startServerMode(notify *notificator.Notificator) {
	fmt.Println("Starting server...")
    listener, error := net.Listen("tcp", "10.22.32.242:12345")
	if error != nil {
		fmt.Println(error)
	}
	manager := ClientManager{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go manager.start(notify)
	for {
		connection, _ := listener.Accept()
		if error != nil {
			fmt.Println(error)
		}
		client := &Client{socket: connection, data: make(chan []byte)}
		manager.register <- client
		go manager.receive(client, notify)
		go manager.send(client)
	}
}

//Now we shift focus to the client side

func (client *Client) receive(notify *notificator.Notificator) {
	for {
		message := make([]byte, 4096)
		length, err := client.socket.Read(message)
		if err != nil {
			client.socket.Close()
			break
		}
		if length > 0 {
			fmt.Println("RECEIVED: " + string(message))
            var text string = string(message)
            notify.Push("Client: New message", text, "/home/user/icon.png", notificator.UR_CRITICAL)
		}
	}
}

func startClientMode(notify *notificator.Notificator) {
	fmt.Println("Starting client...")
    connection, error := net.Dial("tcp", "10.22.32.242:12345")
	if error != nil {
		fmt.Println(error)
        return
	}
	client := &Client{socket: connection}
	go client.receive(notify)
	for {
		reader := bufio.NewReader(os.Stdin)
		message, err := reader.ReadString('\n')
        if err != nil {
            fmt.Println(err)
            return
        }
		if string(message) == "end\n" {
			terminationMessage := "Client user has terminated the session"
			connection.Write([]byte(strings.TrimRight(terminationMessage, "\n")))
			fmt.Println("Terminating client")
			break
		}
		connection.Write([]byte(strings.TrimRight(message, "\n")))
	}
}

func main() {

    notify = notificator.New(notificator.Options{
        DefaultIcon: "icon/default.png",
        AppName:     "My test App",
    })

    //Arguments 
	flagMode := flag.String("mode", "server", "start in client or server mode")
	flag.Parse()
	if strings.ToLower(*flagMode) == "server" {
		startServerMode(notify)
	} else {
		startClientMode(notify)
	}

}
