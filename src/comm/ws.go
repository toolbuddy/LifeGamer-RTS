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

var manager WsClientManager
var mbus MBusNode

type WsClient struct {
    socket  *websocket.Conn
    sending chan []byte
}

func (c *WsClient) read() {
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

        fmt.Printf(">> %s\n", string(msg))
    }
}

func (c *WsClient) write() {
    defer func() {
        c.socket.Close()
    }()

    for {
        msg, ok := <-c.sending

        if !ok {
            c.socket.WriteMessage(websocket.CloseMessage, []byte{})
            return
        }

        c.socket.WriteMessage(websocket.TextMessage, msg)
    }
}

type WsClientManager struct {
    clients     map[*WsClient]bool
    register    chan *WsClient
    unregister  chan *WsClient
    broadcast   chan []byte
}

func (manager *WsClientManager) start() {
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

func WsServerStart(port int) {
    // starting
    log.Println("Starting Websocket server listener")

    manager = WsClientManager {
        clients:    make(map[*WsClient]bool),
        register:   make(chan *WsClient),
        unregister: make(chan *WsClient),
        broadcast:  make(chan []byte),
    }

    go manager.start()

    mbus = NewMBusNode("ws")

    http.HandleFunc("/", mainHandler)

    // listening
    log.Printf("Websocket server listening on port %d", port)

    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
    if b, err := ioutil.ReadAll(r.Body); err == nil {
        fmt.Printf(">> %s\n", b)
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

    client := &WsClient{socket: conn, sending: make(chan []byte)}

    manager.register <- client

    go client.read()
    go client.write()
}
