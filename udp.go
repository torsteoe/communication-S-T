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
	clients    []*Client
	messages   chan []byte
    listener   *net.UDPConn
}

type Client struct {
    addr *net.UDPAddr
	data   chan []byte
}
func postNotification(notify *notificator.Notificator, title string, textContent string) error {
    err := notify.Push(title, textContent, "/home/user/icon.png", notificator.UR_CRITICAL)

    if err != nil {
        fmt.Println("Error", err)
        return err
    }
    return nil
}

func (manager *ClientManager) start(notify *notificator.Notificator) {
    defer manager.listener.Close()
    go manager.receive(notify)
	for {
		select {
		case msg := <-manager.messages:
            fmt.Println("we are about to call sendToAll")
            manager.sendToAll(msg)
        }
    }
}

func (manager *ClientManager) receive(notify *notificator.Notificator) {
	for {
		message := make([]byte, 4096)
        n, addr, err :=  manager.listener.ReadFromUDP(message)
		if err != nil {
            fmt.Println(err)
            return 
		}
        hasClient := false
        for _, client := range manager.clients {
            if client.addr.String() == addr.String() {
                hasClient = true
            }
        }
        if !hasClient {
            client := &Client{
                addr: addr,
            }
            manager.clients = append(manager.clients, client)
        }
		if n > 0 {
            manager.messages <- message
            fmt.Println(string(message), n)
            b := bytes.Trim(message, "\x00")
            textContent := string(b)
            err := postNotification(notify,"Server",  textContent)

            if err != nil {
                fmt.Println("Error", err)
                return
            }
		}
    }
}

func (manager *ClientManager) sendToAll(message []byte) {
    for _, client := range manager.clients {
        fmt.Println("Sending to client: ", client.addr.String())
        manager.listener.WriteTo([]byte("Hello, this is the server talking"), client.addr)
    }
}

func startServerMode(notify *notificator.Notificator, addr string) {
	fmt.Println("Starting server...")
    s, err := net.ResolveUDPAddr("udp4", addr)
    if err != nil {
            fmt.Println(err)
            return
    }
    listener, err := net.ListenUDP("udp4", s)
	if err != nil {
		fmt.Println(err)
        return
	}
	manager := ClientManager{
		clients:    make([]*Client, 0),
		messages:  make(chan []byte),
        listener: listener,
	}

	manager.start(notify)
}

//Now we shift focus to the client side

func (client *Client) receive(connection *net.UDPConn, notify *notificator.Notificator) {
	for {
		message := make([]byte, 4096)
		length, addr,  err := connection.ReadFromUDP(message)
        fmt.Println("address", addr)
		if err != nil {
            fmt.Println(err)
            return
		}
		if length > 0 {
			fmt.Println("RECEIVED: " + string(message))
		}
	}
}

func startClientMode(notify *notificator.Notificator, addr string) {
	fmt.Println("Starting client...")
    s, err := net.ResolveUDPAddr("udp4", addr)
    connection, err := net.DialUDP("udp4", nil, s)
	if err != nil {
		fmt.Println(err)
        return
	}
	client := &Client{}
	go client.receive(connection, notify)
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

func fetchAddr() string {
    cmd := "ipconfig"
    args := []string{"getifaddr", "en0" }
    command := exec.Command(cmd, args[0], args[1])
    out, err := command.Output()

    if err != nil {
        fmt.Println("Error", err)
        panic(err)
        return ""
    }

    port := "12345"
    return strings.TrimSpace(string(out)) + ":" + port
}
func main() {

    notify = notificator.New(notificator.Options{
        DefaultIcon: "icon/default.png",
        AppName:     "My test App",
    })

    //Arguments 
    addr := fetchAddr()
	flagMode := flag.String("mode", "server", "start in client or server mode")
	flag.Parse()
	if strings.ToLower(*flagMode) == "server" {
		startServerMode(notify, addr)
	} else {
        startClientMode(notify, "62.16.226.210:500")
	}

}

