package game

import (
	"comm"
	"encoding/json"
	"errors"
	"game/world"
	"log"
	"math/rand"
	"time"
	"util"
)

// Function pointer type for common onMessage function
type OnMessageFunc func(comm.MessageWrapper)

// Use `NewMessageHandler` to constract a new message handler
type MessageHandler struct {
	GameDB
	CommonData

	onMessage map[comm.MsgType]OnMessageFunc
	mbus      *comm.MBusNode
}

func NewMessageHandler(gameDB GameDB, common_data CommonData, mbus *comm.MBusNode) *MessageHandler {
	mHandler := &MessageHandler{
		gameDB,
		common_data,
		make(map[comm.MsgType]OnMessageFunc),
		mbus,
	}

	mHandler.onMessage[comm.LoginRequest] = mHandler.onLoginRequest
	mHandler.onMessage[comm.LogoutRequest] = mHandler.onLogoutRequest
	mHandler.onMessage[comm.HomePointResponse] = mHandler.onHomePointResponse
	mHandler.onMessage[comm.MapDataRequest] = mHandler.onMapDataRequest
	mHandler.onMessage[comm.BuildRequest] = mHandler.onBuildRequest
	mHandler.onMessage[comm.OccupyRequest] = mHandler.onOccupyRequest
	mHandler.onMessage[comm.Message] = mHandler.onBroadcastMessage

	return mHandler
}

func (mHandler MessageHandler) start() {
	go func() {
		for msg_wrapper := range mHandler.mbus.ReaderChan {
			var payload comm.Payload

			if err := json.Unmarshal(msg_wrapper.Data, &payload); err != nil {
				log.Println("[WARNING]", "Payload unmarshal failed.")
				continue
			}

			go mHandler.onMessage[payload.Msg_type](msg_wrapper)
		}
	}()
}

func (mHandler MessageHandler) startPlayerDataUpdate(client_info ClientInfo) {
	username := client_info.username

	// Send minimap data to user
	mHandler.minimapLock.RLock()
	payload := MinimapDataPayload{comm.Payload{Msg_type: comm.MinimapDataResponse}, *mHandler.minimap}
	mHandler.minimapLock.RUnlock()

	b, err := json.Marshal(payload)
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	msg := comm.MessageWrapper{Cid: client_info.cid, Username: username, SendTo: comm.SendToClient, Data: b}
	mHandler.mbus.Write("ws", msg)

	// Add user to online list if user not exist, and start player data updating
	mHandler.onlineLock.Lock()
	if _, ok := mHandler.online_players[username]; !ok {
		user_ch := make(chan string, 1)
		mHandler.online_players[username] = user_ch

		go playerDataUpdate(client_info, user_ch, mHandler.mbus, mHandler.GameDB)
	}
	mHandler.onlineLock.Unlock()
}

func (mHandler MessageHandler) onLoginRequest(request comm.MessageWrapper) {
	username := request.Username

	_, err := mHandler.playerDB.Get(username)
	if err != nil {
		// Username not found! Send HomePointRequest to client
		b, err := json.Marshal(comm.Payload{comm.HomePointRequest})
		if err != nil {
			log.Println("[ERROR]", err)
		}

		msg := request
		msg.SendTo = comm.SendToClient
		msg.Data = b

		mHandler.mbus.Write("ws", msg)
		return
	}

	mHandler.startPlayerDataUpdate(ClientInfo{request.Cid, request.Username})
}

func (mHandler MessageHandler) onLogoutRequest(request comm.MessageWrapper) {
	var username string = request.Username
	var payload comm.LogoutPayload

	if err := json.Unmarshal(request.Data, &payload); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	if payload.RemoveUser {
		mHandler.onlineLock.Lock()
		// Delete user if user exist
		if user_ch, ok := mHandler.online_players[username]; ok {
			close(user_ch)
			delete(mHandler.online_players, username)
		}
		mHandler.onlineLock.Unlock()
	}

	// Remove client from chunk watching list
	client_info := ClientInfo{request.Cid, request.Username}

	mHandler.clientLock.RLock()
	if poss, ok := mHandler.client2Chunks[client_info]; ok {
		for _, pos := range poss {
			mHandler.chunkLock.Lock()
			if infos, ok := mHandler.chunk2Clients[pos]; ok {
				for i, info := range infos {
					if info == client_info {
						mHandler.chunk2Clients[pos] = append(infos[:i], infos[i+1:]...)
						break
					}
				}
			}
			mHandler.chunkLock.Unlock()
		}
	}
	mHandler.clientLock.RUnlock()

	mHandler.clientLock.Lock()
	delete(mHandler.client2Chunks, client_info)
	mHandler.clientLock.Unlock()
}

func (mHandler MessageHandler) onMapDataRequest(request comm.MessageWrapper) {
	var payload struct {
		comm.Payload
		ChunkPos []util.Point
	}

	if err := json.Unmarshal(request.Data, &payload); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	chunks := []world.Chunk{}

	for _, pos := range payload.ChunkPos {
		chunk, err := mHandler.worldDB.Get(pos.String())
		if err != nil {
			chunk = *world.NewChunk(pos)

			if err := mHandler.worldDB.Put(pos.String(), chunk); err != nil {
				log.Println("[ERROR]", err)
			}
		}

		chunks = append(chunks, chunk)
	}

	payload.Msg_type = comm.MapDataResponse
	map_data := MapDataPayload{payload.Payload, chunks}

	b, err := json.Marshal(map_data)
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	msg := request
	msg.SendTo = comm.SendToClient
	msg.Data = b

	// Update view point map if the message sending successful
	if mHandler.mbus.Write("ws", msg) {
		client_info := ClientInfo{request.Cid, request.Username}

		// Get chunks where this client was watching
		mHandler.clientLock.RLock()
		if poss, ok := mHandler.client2Chunks[client_info]; ok {
			for _, pos := range poss {
				// Remove client from chunk watching list
				mHandler.chunkLock.Lock()
				if infos, ok := mHandler.chunk2Clients[pos]; ok {
					for i, info := range infos {
						if info == client_info {
							mHandler.chunk2Clients[pos] = append(infos[:i], infos[i+1:]...)
							break
						}
					}
				}
				mHandler.chunkLock.Unlock()
			}
		}
		mHandler.clientLock.RUnlock()

		// Update clients who are watching this chunk
		for _, pos := range payload.ChunkPos {
			mHandler.chunkLock.Lock()
			if infos, ok := mHandler.chunk2Clients[pos]; !ok {
				mHandler.chunk2Clients[pos] = []ClientInfo{client_info}
			} else {
				mHandler.chunk2Clients[pos] = append(infos, client_info)
			}
			mHandler.chunkLock.Unlock()
		}

		// Update where the client is watching
		mHandler.clientLock.Lock()
		mHandler.client2Chunks[client_info] = payload.ChunkPos
		mHandler.clientLock.Unlock()
	}
}

func (mHandler MessageHandler) onBuildRequest(request comm.MessageWrapper) {
	var payload BuildingPayload
	var err error

	if err = json.Unmarshal(request.Data, &payload); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	log.Printf("[INFO] %s request %s at chunk (%s), pos (%s)", request.Username, string(payload.Action), payload.Structure.Chunk.String(), payload.Structure.Pos.String())

	// Retrieve info from struct definition
	world.CompleteStructure(&payload.Structure)

	user, err := mHandler.playerDB.Get(request.Username)
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}
	user.Update()

	mHandler.playerDB.Lock(request.Username)
	defer mHandler.playerDB.Unlock(request.Username)

	mHandler.worldDB.Lock(payload.Structure.Chunk.String())
	defer mHandler.worldDB.Unlock(payload.Structure.Chunk.String())

	chunk, err := mHandler.worldDB.Get(payload.Structure.Chunk.String())
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	// Check user's permission
	if request.Username != chunk.Owner {
		log.Println("[INFO] User do not own the chunk.")
		return
	}

	// Check user's money
	switch payload.Action {
	case Build:
		if user.Money < int64(payload.Structure.Cost) {
			err = errors.New("User do not have enough money.")
		}
		//case Upgrade:
		//case Destruct:
		//case Repair:
		//case Restart:
	}

	// Handle player error
	if err != nil {
		log.Println(err)
		return
	}

	// Check world status & perform action
	switch payload.Action {
	case Build:
		payload.Structure.Status = world.Building
		err = world.BuildStructure(&chunk, payload.Structure)
		user.Money -= int64(payload.Structure.Cost)
		go func() {
			select {
			case <-time.After(time.Duration(world.StructMap[payload.Structure.ID].BuildTime) * time.Second):
				UpdateChunk(mHandler.GameDB, chunk.Owner, chunk.Key())
			}
		}()
	//case Upgrade:
	case Destruct:
		index, _ := world.GetStructure(chunk, payload.Structure)

		// Set properties to 0 to prevent minus after destruction
		if chunk.Structures[index].Status == world.Building || chunk.Structures[index].Status == world.Halted {
			chunk.Structures[index].Power = 0
			chunk.Structures[index].Money = 0
			chunk.Structures[index].Population = 0
			chunk.Structures[index].PopulationCap = 0
		}

		if chunk.Structures[index].Status != world.Destructing {
			chunk.Structures[index].Status = world.Destructing
			chunk.Structures[index].BuildTime = world.StructMap[payload.Structure.ID].BuildTime
			chunk.Structures[index].UpdateTime = time.Now().Unix()
			go func() {
				select {
				case <-time.After(time.Duration(world.StructMap[payload.Structure.ID].BuildTime) * time.Second):
					UpdateChunk(mHandler.GameDB, chunk.Owner, chunk.Key())
				}
			}()
		}
		//case Repair:
	case Restart:
		index, _ := world.GetStructure(chunk, payload.Structure)

		if chunk.Structures[index].Status == world.Halted {
			chunk.Structures[index].Status = world.Running

			// Calcutate power
			if chunk.Structures[index].Power > 0 {
				user.PowerMax += int64(chunk.Structures[index].Power)
			} else {
				user.Power += int64(-(chunk.Structures[index].Power))
			}

			// Calculate money
			user.MoneyRate += int64(chunk.Structures[index].Money)

			// Calculate population
			user.PopulationCap += int64(chunk.Structures[index].PopulationCap)

			if chunk.Structures[index].Population > 0 {
				chunk.PopulationRate += int64(chunk.Structures[index].Population)
			}
		}
	}

	// Handle world error
	if err != nil {
		log.Println(err)
		return
	}

	// Check chunk resource status(such as human not enough)
	human_needed := 0
	for _, b := range chunk.Structures {
		human_needed += -b.Population
	}

	// TODO: Change building status when human not enough
	//log.Println(human_needed, chunk.Population)

	// Write data into database if no world error happened
	mHandler.playerDB.Put(request.Username, user)
	mHandler.worldDB.Put(chunk.Key(), chunk)
}

// TODO: deal with long-distance move & boundary check
// TODO: move delay
func (mHandler MessageHandler) onOccupyRequest(request comm.MessageWrapper) {
	var payload struct {
		comm.Payload
		From   util.Point
		To     util.Point
		Amount int64
	}

	if err := json.Unmarshal(request.Data, &payload); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	log.Printf("[INFO] %s move %v from (%s) to (%s)", request.Username, payload.Amount, payload.From.String(), payload.To.String())

	username := request.Username

	// chunk operation
	mHandler.playerDB.Lock(username)
	defer mHandler.playerDB.Unlock(username)

	mHandler.worldDB.Lock(payload.From.String())
	defer mHandler.worldDB.Unlock(payload.From.String())

	mHandler.worldDB.Lock(payload.To.String())
	defer mHandler.worldDB.Unlock(payload.To.String())

	chunk_from, err := mHandler.worldDB.Get(payload.From.String())
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	chunk_to, err := mHandler.worldDB.Get(payload.To.String())
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	// check if player exist
	player_data, err := mHandler.playerDB.Get(username)
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	if chunk_from.Owner != username {
		log.Println("[ERROR]", "User do not own the chunk")
	}

	if chunk_from.Population < payload.Amount {
		log.Println("[ERROR]", "Source chunk population not enough")
	}

	if chunk_to.Owner != "" {
		// TODO: Fighting
	}

	// Check finished, move minions
	chunk_from.Population -= payload.Amount
	chunk_to.Population += payload.Amount
	chunk_to.Owner = username

	if err := mHandler.worldDB.Put(chunk_from.Key(), chunk_from); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	if err := mHandler.worldDB.Put(chunk_to.Key(), chunk_to); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	// player operation
	player_data.Territory = append(player_data.Territory, payload.To)

	if err := mHandler.playerDB.Put(username, player_data); err != nil {
		log.Println("[ERROR]", err)
		return
	}
}

func (mHandler MessageHandler) onHomePointResponse(request comm.MessageWrapper) {
	var payload struct {
		comm.Payload
		Pos util.Point
	}

	if err := json.Unmarshal(request.Data, &payload); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	username := request.Username

	point := func() {
		mHandler.minimapLock.RLock()
		defer mHandler.minimapLock.RUnlock()
		for {
			point_x := rand.Intn(50) - 25
			point_y := rand.Intn(50) - 25
			if mHandler.minimap.Owner[point_x][point_y] != "" {
				continue
			}
		}
	}

	log.Println(point)

	// chunk operation
	mHandler.playerDB.Lock(username)
	defer mHandler.playerDB.Unlock(username)

	mHandler.worldDB.Lock(payload.Pos.String())
	defer mHandler.worldDB.Unlock(payload.Pos.String())

	chunk, err := mHandler.worldDB.Get(payload.Pos.String())
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	// check if player exist
	player_data, err := mHandler.playerDB.Get(username)

	if chunk.Owner != "" {
		// Chunk is occupied
		if err != nil {
			// Username not found! Send HomePointRequest to client
			b, err := json.Marshal(comm.Payload{comm.HomePointRequest})
			if err != nil {
				log.Println("[ERROR]", err)
			}

			msg := request
			msg.SendTo = comm.SendToClient
			msg.Data = b

			mHandler.mbus.Write("ws", msg)
		}
		return
	} else {
		chunk.Owner = username
	}

	if err != nil {
		// Username not found! Create new player in PlayerDB
		// Set population provided by home chunk
		player_data.PopulationCap = 100

		// TODO: Set default money! this is for testing
		player_data.Money = 100000
		player_data.MoneyRate = 100

		player_data.Home = payload.Pos

		player_data.Initialized = true

		defer mHandler.startPlayerDataUpdate(ClientInfo{request.Cid, username})
	}

	// Provide initial population
	player_data.Population = 10
	chunk.Population = 10

	if err := mHandler.worldDB.Put(chunk.Key(), chunk); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	// player operation
	player_data.Territory = append(player_data.Territory, payload.Pos)
	player_data.UpdateTime = time.Now().Unix()

	if err := mHandler.playerDB.Put(username, player_data); err != nil {
		log.Println("[ERROR]", err)
		return
	}
}

func (mHandler MessageHandler) onBroadcastMessage(request comm.MessageWrapper) {
	request.SendTo = comm.Broadcast
	mHandler.mbus.Write("ws", request)
}
