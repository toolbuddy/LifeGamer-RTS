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

    cid         int
    token       string
    username    string
}

// Struct for websocket server
type WsServer struct {
    clients         map[string] map[int] *WsClient  // map username to actual ws connection
    login_queue     chan *WsClient                  // ws clients who are going to login
    logout_queue    chan *WsClient                  // ws clients who are going to logout
    mbus            *MBusNode
}

func NewWsServer() (server *WsServer, err error) {
    // Initialize WsServer component
    clients      := make(map[string] map[int] *WsClient)
    login_queue  := make(chan *WsClient)
    logout_queue := make(chan *WsClient)

    mbus, err   := NewMBusNode("ws")
    if err != nil {
        return
    }

    server = &WsServer { clients, login_queue, logout_queue, mbus }
    return
}

// Handle client connection
func (server *WsServer) Start(port int) {
    // starting
    log.Println("Starting Websocket server listener")

    // client id generator
    cid_generator := func () (func () int) {
        id := 0

        return func () int {
            id++
            return id
        }
    }()

    // http handler
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
        username, err := Login(login_data.Token)
        if err != nil {
            log.Println(err)
            conn.Close()
            return
        }

        server.login_queue <- &WsClient { conn, cid_generator(), login_data.Token, username }
    })

    // listening
    log.Println("Websocket server listening on port", port)

    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

    // Handle message from MBus ( WsClient write )
    go func() {
        for msg_wrapper := range server.mbus.ReaderChan {
            cid := msg_wrapper.Cid
            username := msg_wrapper.Username

            // check the username and cid for security
            if user, ok := server.clients[username]; !ok {
                continue
            } else if _, ok := user[cid]; !ok {
                continue
            }

            switch msg_wrapper.SendTo {
            case SendToClient:
                if user, ok := server.clients[username]; ok {
                    if client, ok := user[cid]; ok {
                        err := client.WriteMessage(websocket.TextMessage, msg_wrapper.Data)
                        if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
                            log.Println(err)
                        }
                    }
                }
            case SendToUser:
                if user, ok := server.clients[username]; ok {
                    for _, client := range user {
                        err := client.WriteMessage(websocket.TextMessage, msg_wrapper.Data)
                        if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
                            log.Println(err)
                        }
                    }
                }
            case Broadcast:
                for _, user := range server.clients {
                    for _, client := range user {
                        err := client.WriteMessage(websocket.TextMessage, msg_wrapper.Data)
                        if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
                            log.Println(err)
                        }
                    }
                }
            }
        }
    }()

    // Handle client login & logout
    go func() {
        for {
            select {
            case client := <-server.login_queue: // login
                // Add new client to map
                cid := client.cid
                username := client.username

                user, ok := server.clients[username]
                if !ok {
                    user = make(map[int] *WsClient)
                    server.clients[username] = user
                }

                user[cid] = client
                log.Printf("Ws: New client for user %s connected (cid: %v)", username, cid)

                // Start goroutine to handle massage from each websocket client ( WsClient read )
                go func() {
                    for {
                        _, msg, err := client.ReadMessage()
                        if err != nil {
                            if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
                                log.Printf("%s's client, cid %d: %s", username, cid, err.Error())
                            }

                            server.logout_queue <- client
                            return
                        }

                        server.mbus.Write("game", MessageWrapper { Cid: cid, Username: username, Data: msg })
                    }
                }()

                // Send username to browser
                client.WriteJSON( Payload { LoginResponse, username } )

                b, err := json.Marshal( Payload { LoginRequest, username } )
                if err != nil {
                    log.Println(err)
                    continue
                }

                server.mbus.Write("game", MessageWrapper { Cid: cid, Username: username, Data: b })
            case client := <-server.logout_queue: // logout
                cid := client.cid
                username := client.username

                b, err := json.Marshal( Payload { LogoutRequest, username } )
                if err != nil {
                    log.Println(err)
                    continue
                }

                server.mbus.Write("game", MessageWrapper { Cid: cid, Username: username, Data: b })

                if user, ok := server.clients[username]; ok {
                    delete(user, cid)
                }
            }
        }
    }()
}
