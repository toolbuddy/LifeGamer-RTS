package comm

import (
    "log"
    "net/http"
    "io/ioutil"
    "fmt"
    "github.com/gorilla/websocket"
    "strconv"
    "time"
)

type Client struct {
    socket  *websocket.Conn
    sending chan []byte
}

func (c *Client) read() {
    defer func() {
        manager.unregister <- c
        c.socket.Close()
    }()

    for {
        _, msg, err := c.socket.ReadMessage()

        if err != nil {
            log.Println(err)
            return
        }

        fmt.Printf("%v\n", string(msg))
    }
}

func (c *Client) write() {
    defer func() {
        c.socket.Close()
    }()

    for {
        msg, ok := <-c.sending

        if !ok {
            c.socket.WriteMessage(websocket.CloseMessage, []byte{})
            return
        }

        //log.Println(string(msg))

        c.socket.WriteMessage(websocket.TextMessage, msg)
    }
}

type ClientManager struct {
    clients     map[*Client]bool
    register    chan *Client
    unregister  chan *Client
    broadcast   chan []byte
}

func (manager *ClientManager) start() {
    for {
        select {
        case c := <-manager.register:
            manager.clients[c] = true
        case c := <-manager.unregister:
            if _, ok := manager.clients[c]; ok {
                close(c.sending)
                delete(manager.clients, c)
            }
        case msg := <-manager.broadcast:
            for k, _ := range manager.clients {
                k.sending <- msg[:len(msg)]
            }
        }
    }
}

var manager ClientManager

var mbus MBusClient

func WsServerStart(port int) {
    // starting
    log.Println("Starting WebSocket server listener")

    manager = ClientManager {
        clients:    make(map[*Client]bool),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        broadcast:  make(chan []byte),
    }

    mbus = NewClient("ws")

    http.HandleFunc("/", mainHandler)

    go manager.start()

    // listening
    log.Printf("WebSocket server listening on port %d", port)

    go func() {
        timer := time.NewTicker(time.Second)
        defer timer.Stop()

        for money := 0;; {
            select {
            case <-timer.C:
                if len(manager.clients) != 0 {
                    for k, _  := range manager.clients {
                        s := strconv.Itoa(money)
                        b := []byte(s)
                        k.sending <- b[:len(b)]

                        //test part
                        s, ok := mbus.Get()
                        if ok {
                            b = []byte(s)
                            k.sending <- b[:len(b)]
                        }
                    }

                    money += 100
                }
            }
        }
    }()

    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
    if b, err := ioutil.ReadAll(r.Body); err == nil {
        fmt.Printf("%s\n", b)
    } else {
        log.Println(err)
        return
    }

    upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
    conn, err := (&upgrader).Upgrade(w, r, nil)

    if err != nil {
        log.Println(err)
        return
    }

    client := &Client{socket: conn, sending: make(chan []byte)}

    manager.register <- client

    go client.read()
    go client.write()
}
