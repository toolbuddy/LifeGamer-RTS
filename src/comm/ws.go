package comm

import (
    "fmt"
    "log"
    "net/http"
    "github.com/gorilla/websocket"
    "encoding/json"
    "reflect"
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

        server.reg_queue <- &WsClient { conn, username }
    })

    // listening
    log.Printf("Websocket server listening on port %d", port)
    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

    // Handle message from MBus ( WsClient write )
    go func() {
        for m := range server.mbus.ReaderChan {
            b, err := json.Marshal(m)

            if err != nil {
                log.Println(err)
                continue
            }

            username := reflect.ValueOf(m).FieldByName("Username").String()
            user_conns, ok := server.clients[username]

            if ok {
                for _, c := range user_conns {
                    if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
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
            case c := <-server.reg_queue:   // register
                // Append new client into channel list
                user_conns, ok := server.clients[c.Username]
                if ok {
                    server.clients[c.Username] = append(user_conns, c)
                } else {
                    server.clients[c.Username] = []*WsClient { c }
                }

                // Start goroutine to handle massage from each websocket client ( WsClient read )
                go func() {
                    for {
                        _, msg, err := c.ReadMessage()

                        if err != nil {
                            if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
                                log.Println(err)
                            }

                            c.Close()
                            server.unreg_queue <- c

                            return
                        }

                        var base BasePayload

                        if err := json.Unmarshal(msg, &base); err != nil {
                            log.Println(err)
                            continue
                        }

                        switch base.Msg_type {
                        case PlayerData:
                            var data PlayerDataPayload

                            if err := json.Unmarshal(msg, &data); err != nil {
                                log.Println(err)
                                continue
                            }

                            server.mbus.Write("game", data)

                        case MapData:
                            var data MapDataPayload

                            if err := json.Unmarshal(msg, &data); err != nil {
                                log.Println(err)
                                continue
                            }

                            server.mbus.Write("game", data)
                        }
                    }
                }()

                // Send username to browser
                c.WriteJSON( BasePayload { Msg_type: LoginResult, Username: c.Username, Msg: "Login Successful" } )

            case c := <-server.unreg_queue: // unregister
                if user_conns, ok := server.clients[c.Username]; ok {
                    for i, conn := range user_conns {
                        if conn == c {
                            server.clients[c.Username] = append(user_conns[:i], user_conns[i+1:]...)
                        }
                    }
                }
            }
        }
    }()
}
