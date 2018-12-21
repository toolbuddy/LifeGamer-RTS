package comm

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

// Struct to store connected client
type WsClient struct {
	*websocket.Conn

	cid      int
	username string
	token    string
}

// Struct for websocket server
type WsServer struct {
	clients      map[string][]WsClient // map username to actual ws connection
	login_queue  chan WsClient         // ws clients who are going to login
	logout_queue chan WsClient         // ws clients who are going to logout
	mbus         *MBusNode
	clientsLock  *sync.RWMutex
}

func NewWsServer() (server *WsServer, err error) {
	// Initialize WsServer component
	clients := make(map[string][]WsClient)
	login_queue := make(chan WsClient)
	logout_queue := make(chan WsClient)

	mbus, err := NewMBusNode("ws")
	if err != nil {
		return
	}

	server = &WsServer{clients, login_queue, logout_queue, mbus, new(sync.RWMutex)}
	return
}

// Handle client connection
func (server WsServer) Start(port int) {
	log.Println("[INFO] Starting Websocket server listener")

	// client id generator
	cid_generator := func() func() int {
		id := 0
		return func() int {
			id++
			return id
		}
	}()

	// http handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("[ERROR]", err)
			return
		}

		// Read JSON string send from client, use token to login
		// TODO: Support GitLab Private token and Access token
		var login_data struct {
			Token_type string
			Token      string
		}

		if err := conn.ReadJSON(&login_data); err != nil {
			log.Println("[ERROR]", err)
			conn.Close()
			return
		}

		// Close connection when login failed
		username, err := Login(login_data.Token, login_data.Token_type)
		if err != nil {
			log.Println("[ERROR]", err)
			conn.Close()
			return
		}

		server.login_queue <- WsClient{conn, cid_generator(), username, login_data.Token}
	})

	// listening
	log.Println("[INFO] Websocket server listening on port", port)

	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	// Handle client login & logout
	go func() {
		for {
			select {
			case client := <-server.login_queue: // login
				server.loginHandler(client)
			case client := <-server.logout_queue: // logout
				server.logoutHandler(client)
			}
		}
	}()

	// Handle message from MBus ( WsClient write )
	go server.write2client()
}

func (server WsServer) loginHandler(client WsClient) {
	cid := client.cid
	username := client.username

	// Add new client to client list
	server.clientsLock.Lock()
	if user_clients, ok := server.clients[username]; !ok {
		server.clients[username] = []WsClient{client}
	} else {
		server.clients[username] = append(user_clients, client)
	}
	server.clientsLock.Unlock()

	log.Printf("[INFO] New client of user %s connected (cid: %v)", username, cid)

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

			server.mbus.Write("game", MessageWrapper{Cid: cid, Username: username, Data: msg})
		}
	}()

	// Send username to browser
	client.WriteJSON(UsernamePayload{Payload{LoginResponse}, username})

	b, err := json.Marshal(Payload{LoginRequest})
	if err != nil {
		log.Println("[WARNING]", err)
		return
	}

	server.mbus.Write("game", MessageWrapper{Cid: cid, Username: username, Data: b})
}

func (server WsServer) logoutHandler(client WsClient) {
	cid := client.cid
	username := client.username

	find := func(clients []WsClient, cid int) int {
		for i, client := range clients {
			if client.cid == cid {
				return i
			}
		}

		return -1
	}

	server.clientsLock.Lock()
	if user_clients, ok := server.clients[username]; ok {
		if i := find(user_clients, client.cid); i != -1 {
			server.clients[username] = append(user_clients[:i], user_clients[i+1:]...)
		}

		b, err := json.Marshal(LogoutPayload{Payload{LogoutRequest}, len(user_clients) == 0})
		if err != nil {
			log.Println("[WARNING]", err)
			server.clientsLock.Unlock()
			return
		}

		server.mbus.Write("game", MessageWrapper{Cid: cid, Username: username, Data: b})
	}
	server.clientsLock.Unlock()
}

func (server WsServer) write2client() {
	for msg_wrapper := range server.mbus.ReaderChan {
		cid := msg_wrapper.Cid
		username := msg_wrapper.Username
		send_to := msg_wrapper.SendTo

		find := func(clients []WsClient, cid int) int {
			for i, client := range clients {
				if client.cid == cid {
					return i
				}
			}

			return -1
		}

		// check the username and cid for security
		server.clientsLock.RLock()
		if send_to != Broadcast {
			if user_clients, ok := server.clients[username]; send_to == SendToUser && !ok {
				server.clientsLock.RUnlock()
				continue
			} else if send_to == SendToClient && (!ok || find(user_clients, cid) == -1) {
				server.clientsLock.RUnlock()
				continue
			}
		}
		server.clientsLock.RUnlock()

		server.clientsLock.RLock()
		switch send_to {
		case Broadcast:
			for _, user_clients := range server.clients {
				for _, client := range user_clients {
					err := client.WriteMessage(websocket.TextMessage, msg_wrapper.Data)
					if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						log.Println("[WARNING]", err)
					}
				}
			}
		case SendToUser:
			if user_clients, ok := server.clients[username]; ok {
				for _, client := range user_clients {
					err := client.WriteMessage(websocket.TextMessage, msg_wrapper.Data)
					if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						log.Println("[WARNING]", err)
					}
				}
			}
		case SendToClient:
			if user_clients, ok := server.clients[username]; ok {
				if i := find(user_clients, cid); i != -1 {
					err := user_clients[i].WriteMessage(websocket.TextMessage, msg_wrapper.Data)
					if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						log.Println("[WARNING]", err)
					}
				}
			}
		}
		server.clientsLock.RUnlock()
	}
}
