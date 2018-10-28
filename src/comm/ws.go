package comm

import (
    "fmt"
    "log"
    "net/http"
    "github.com/gorilla/websocket"
)

// Struct to store connected client
type WsClient struct {
    Username string
    Socket  *websocket.Conn
}

// Struct for websocket server
type WsServer struct {
    clients     map[string] []*WsClient // map username to actual ws connection
    reg_queue   chan *WsClient          // ws clients to be register
    mbus        MBusNode
}

func NewWsServer() (server WsServer, err error) {
    // Initialize WsServer component
    server.clients = make(map[string] []*WsClient)
    server.reg_queue = make(chan *WsClient)
    server.mbus, err = NewMBusNode("ws")

    return
}

// handles client registration
func (server *WsServer) Start(port int) {
    // starting
    log.Println("Starting Websocket server listener")

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        upgrader := websocket.Upgrader { CheckOrigin: func(r *http.Request) bool { return true } }
        conn, err := upgrader.Upgrade(w, r, nil)

        if err != nil {
            log.Println(err)
            return
        }

        // Read JSON string send from client, use token to login
        // TODO: Support GitLab Private token and Access token
        var login_data struct {
            Token_type string
            Token string
        }

        err = conn.ReadJSON(&login_data)
        if err != nil {
            conn.Close()
            log.Println(err)
            return
        }

        username, err := Login(login_data.Token)

        if err != nil {
            // Close connection when login failed
            conn.Close()
            log.Println(err)
            return
        }

        server.reg_queue <- &WsClient { Username: username, Socket: conn }
    })

    // listening
    log.Printf("Websocket server listening on port %d", port)
    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

    // Handle message from MBus
    go func() {
        for m := range server.mbus.ReaderChan {
            switch msg := m.(type) {
            case BasePayload:
                fmt.Println(msg.Username)
            case PlayerDataPayload:
                fmt.Println(msg.Human)
            case MapDataPayload:
                continue
            case BuildPayload:
                continue
            }
        }
    }()

    go func() {
        for {
            select {
                case c := <-server.reg_queue:  // New client to register
                // Append new client into channel list
                user_conns, ok := server.clients[c.Username]
                if ok {
                    server.clients[c.Username] = append(user_conns, c)
                } else {
                    server.clients[c.Username] = []*WsClient { c }
                }

                // TODO: Start goroutine to handle massage from each websocket client

                // Send username to browser
                c.Socket.WriteJSON( BasePayload { Msg_type: LoginResult, Username: c.Username, Msg: "Login Successful" } )
                // TODO: unregister
            }
        }
    }()
}
