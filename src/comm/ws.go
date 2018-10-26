package comm

import (
    "log"
    "net/http"
    "fmt"
    "github.com/gorilla/websocket"
)

var manager WsClientManager
var mbus MBusNode

type WsClient struct {
    username string
    socket  *websocket.Conn
}

// TODO: add read/write

type WsClientManager struct {
    clients     map[string]*WsClient
    register    chan *WsClient
}

func (manager *WsClientManager) start() {
    for {
        select {
        case c := <-manager.register:
            manager.clients[c.username] = c
            // TODO: send username to browser
        }
    }
}

func WsServerStart(port int) {
    // starting
    log.Println("Starting Websocket server listener")

    manager = WsClientManager {
        clients:    make(map[string]*WsClient),
        register:   make(chan *WsClient),
    }

    go manager.start()

    mbus = NewMBusNode("ws")

    http.HandleFunc("/", mainHandler)

    // listening
    log.Printf("Websocket server listening on port %d", port)

    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
    upgrader := websocket.Upgrader { CheckOrigin: func(r *http.Request) bool { return true } }
    conn, err := upgrader.Upgrade(w, r, nil)

    if err != nil {
        log.Println(err)
        return
    }

    // assume token is a string, not json
    // TODO: use ReadJSON
    _, token, err:= conn.ReadMessage()

    if err != nil {
        log.Println(err)
        return
    }

    username, err := Login(string(token))

    if err != nil {
        log.Println(err)
        return
    }

    manager.register <- &WsClient { username: username, socket: conn }
}
