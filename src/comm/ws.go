package comm

import (
    "fmt"
    "log"
    "net/http"
    "github.com/gorilla/websocket"
    "encoding/json"
)

// Struct to store connected client
type WsClient struct {
    *websocket.Conn

    Username string
}

// Struct for websocket server
type WsServer struct {
    clients     map[string] []*WsClient // map username to actual ws connection
    reg_queue   chan *WsClient          // ws clients to be registered
    unreg_queue chan *WsClient          // ws clients to be unregistered
    mbus        *MBusNode
}

func NewWsServer() (server *WsServer, err error) {
    // Initialize WsServer component
    clients     := make(map[string] []*WsClient)
    reg_queue   := make(chan *WsClient)
    unreg_queue := make(chan *WsClient)
    mbus, err   := NewMBusNode("ws")

    if err != nil {
        return
    }

    server = &WsServer { clients, reg_queue, unreg_queue, mbus }

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

        if err := conn.ReadJSON(&login_data); err != nil {
            log.Println(err)
            conn.Close()
            return
        }

        // Close connection when login failed
        if username, err := Login(login_data.Token); err != nil {
            log.Println(err)
            conn.Close()
            return
        } else {
            server.reg_queue <- &WsClient { conn, username }
        }
    })

    // listening
    log.Println("Websocket server listening on port", port)

    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

    // Handle message from MBus ( WsClient write )
    go func() {
        for msg := range server.mbus.ReaderChan {
            payload := new(Payload)

            if err := json.Unmarshal(msg, payload); err != nil {
                log.Println(err)
                continue
            }

            username := payload.Username

            if user_conns, ok := server.clients[username]; ok {
                for _, conn := range user_conns {
                    if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
                        if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
                            log.Println(err)
                        }
                    }
                }
            }
        }
    }()

    go func() {
        for {
            select {
            case conn := <-server.reg_queue: // register
                username := conn.Username

                // Append new client into channel list
                if user_conns, ok := server.clients[username]; ok {
                    server.clients[username] = append(user_conns, conn)
                } else {
                    server.clients[username] = []*WsClient { conn }
                }

                // Start goroutine to handle massage from each websocket client ( WsClient read )
                go func() {
                    for {
                        _, msg, err := conn.ReadMessage()

                        if err != nil {
                            if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
                                log.Println(err)
                            }

                            conn.Close()
                            server.unreg_queue <- conn

                            return
                        }

                        server.mbus.Write("game", msg)
                    }
                }()

                // Send username to browser
                conn.WriteJSON( Payload { LoginResponse, username, "Login Successful" } )

                if b, err := json.Marshal( Payload { Msg_type: PlayerDataRequest, Username: username } ); err != nil {
                    log.Println(err)
                } else {
                    server.mbus.Write("game", b)
                }

            case conn := <-server.unreg_queue: // unregister
                username := conn.Username

                if user_conns, ok := server.clients[username]; ok {
                    for i, c := range user_conns {
                        if c == conn {
                            server.clients[username] = append(user_conns[:i], user_conns[i+1:]...)
                        }
                    }
                }
            }
        }
    }()
}
