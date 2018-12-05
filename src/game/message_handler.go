package game

import (
    "game/world"
    "log"
    "encoding/json"
    "comm"
    "time"
    "util"
    "errors"
)

// Function pointer type for common onMessage function
type OnMessageFunc func(comm.MessageWrapper)

// Use `NewMessageHandler` to constract a new message handler
type MessageHandler struct {
    GameDB
    CommonData

    onMessage   map[comm.MsgType] OnMessageFunc
    mbus        *comm.MBusNode
}

func NewMessageHandler(gameDB GameDB, common_data CommonData, mbus *comm.MBusNode) *MessageHandler {
    mHandler := &MessageHandler {
        gameDB,
        common_data,
        make(map[comm.MsgType] OnMessageFunc),
        mbus,
    }

    mHandler.onMessage[comm.LoginRequest]      = mHandler.onLoginRequest
    mHandler.onMessage[comm.LogoutRequest]     = mHandler.onLogoutRequest
    mHandler.onMessage[comm.HomePointResponse] = mHandler.onHomePointResponse
    mHandler.onMessage[comm.MapDataRequest]    = mHandler.onMapDataRequest
    mHandler.onMessage[comm.BuildRequest]      = mHandler.onBuildRequest

    return mHandler
}

func (mHandler MessageHandler) start() {
    go func() {
        for msg_wrapper := range mHandler.mbus.ReaderChan {
            var payload comm.Payload

            if err := json.Unmarshal(msg_wrapper.Data, &payload); err != nil {
                log.Println(err)
                continue
            }

            go mHandler.onMessage[payload.Msg_type](msg_wrapper)
        }
    }()
}

func (mHandler MessageHandler) startPlayerDataUpdate(client_info ClientInfo) {
    username := client_info.username

    // Add user to online list if user not exist, and start player data updating
    mHandler.playerLock.Lock()
    if _, ok := mHandler.online_players[username]; !ok {
        user_ch := make(chan string, 1)
        mHandler.online_players[username] = user_ch

        go playerDataUpdate(client_info, user_ch, mHandler.mbus, mHandler.playerDB)
    }
    mHandler.playerLock.Unlock()
}

func (mHandler MessageHandler) onLoginRequest(request comm.MessageWrapper) {
    username := request.Username

    _, err := mHandler.playerDB.Get(username)
    if err != nil {
        // Username not found! Send HomePointRequest to client
        b, err := json.Marshal(comm.Payload { comm.HomePointRequest, username })
        if err != nil {
            log.Println(err)
        }

        msg := request
        msg.SendTo = comm.SendToClient
        msg.Data = b

        mHandler.mbus.Write("ws", msg)
        return
    }

    mHandler.startPlayerDataUpdate(ClientInfo { request.Cid, request.Username })
}

func (mHandler MessageHandler) onLogoutRequest(request comm.MessageWrapper) {
    username := request.Username

    mHandler.playerLock.Lock()
    // Delete user if user exist
    if user_ch, ok := mHandler.online_players[username]; ok {
        close(user_ch)
        delete(mHandler.online_players, username)
    }
    mHandler.playerLock.Unlock()
}

func (mHandler MessageHandler) onHomePointResponse(response comm.MessageWrapper) {
    var payload struct {
        comm.Payload
        Home util.Point
    }

    if err := json.Unmarshal(response.Data, &payload); err != nil {
        log.Println(err)
        return
    }

    username := response.Username
    log.Printf("User %s select (%v) as homepoint", username, payload.Home)

    player_data, err := mHandler.playerDB.Get(username)
    if err != nil {
        // Check chunk is available
        chunk, err := mHandler.worldDB.Get(payload.Home.String())
        if err != nil {
            log.Println(err)
            return
        }

        if chunk.Owner != "" {
            log.Println("Chunk occupied, cannot use here as new home.")
            return
        }

        // Write chunk change into DB
        chunk.Owner = username
        if err := mHandler.worldDB.Put(chunk.Key(), chunk); err != nil {
            log.Println(err)
            return
        }

        // Username not found! Create new player in PlayerDB
        player_data.Home = payload.Home
        player_data.UpdateTime = time.Now().Unix()

        // TODO: Set default money! this is for testing
        player_data.Money = 100000
        player_data.MoneyRate = 100

        if err := mHandler.playerDB.Put(username, player_data); err != nil {
            log.Println(err)
            return
        }

        mHandler.startPlayerDataUpdate(ClientInfo { response.Cid, response.Username })
    }
}

func (mHandler MessageHandler) onMapDataRequest(request comm.MessageWrapper) {
    var payload struct {
        comm.Payload
        ChunkPos []util.Point
    }

    if err := json.Unmarshal(request.Data, &payload); err != nil {
        log.Println(err)
        return
    }

    chunks := []world.Chunk {}

    for _, pos := range payload.ChunkPos {
        chunk, err := mHandler.worldDB.Get(pos.String())
        if err != nil {
            chunk = *world.NewChunk(pos)

            if err := mHandler.worldDB.Put(pos.String(), chunk); err != nil {
                log.Println(err)
            }
        }

        chunks = append(chunks, chunk)
    }

    payload.Msg_type = comm.MapDataResponse
    map_data := MapDataPayload { payload.Payload, chunks }

    b, err := json.Marshal(map_data)
    if err != nil {
        log.Println(err)
        return
    }

    msg := request
    msg.SendTo = comm.SendToClient
    msg.Data = b

    // Update view point map if the message sending successful
    if mHandler.mbus.Write("ws", msg) {
        client_info := ClientInfo { request.Cid, request.Username }

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
                mHandler.chunk2Clients[pos] = []ClientInfo { client_info }
            } else {
                mHandler.chunk2Clients[pos] = append(infos, client_info)
            }
            mHandler.chunkLock.Unlock()
        }

        // Update where the client is watching
        mHandler.clientLock.Lock()
        mHandler.client2Chunks[client_info] = payload.ChunkPos
        mHandler.clientLock.Unlock()

        // TODO: move this part to mapDataUpdate (minimap update part)
/*
        msg = request
        msg.SendTo = comm.SendToUser
        var mmap_data MinimapDataPayload
        mmap_data.Msg_type = comm.MinimapDataResponse
        mmap_data.Size = util.Size{50, 50}
        mmap_data.Terrain = make([][]world.TerrainType, 50)
        mmap_data.Owner = make([][]string, 50)
        for i := 0;i < 50;i++ {
            mmap_data.Terrain[i] = make([]world.TerrainType, 50)
            mmap_data.Owner[i] = make([]string, 50)
            for j := 0;j < 50;j++ {
                mmap_data.Terrain[i][j] = world.Grass
                mmap_data.Owner[i][j] = "korea"
            }
        }
        b, err = json.Marshal(mmap_data)
        if err != nil {
            log.Println(err)
            return
        }

        msg.Data = b
        mHandler.mbus.Write("ws", msg)
*/
    }
}

func (mHandler *MessageHandler) onBuildRequest(request comm.MessageWrapper) {
    var payload BuildingPayload
    var err error

    if err = json.Unmarshal(request.Data, &payload); err != nil {
        log.Println(err)
        return
    }

    log.Printf("world: %s request %s at chunk (%s), pos (%s)", payload.Username, string(payload.Action), payload.Structure.Chunk.String(), payload.Structure.Pos.String())

    // Retrieve info
    world.CompleteStructure(&payload.Structure)

    user, err := mHandler.playerDB.Get(payload.Username)
    if err != nil {
        log.Println(err)
        return
    }
    user.Update()

    chunk, err := mHandler.worldDB.Get(payload.Structure.Chunk.String())
    if err != nil {
        log.Println(err)
        return
    }

    // Check user's permission
    if payload.Username != chunk.Owner {
        log.Println("User do not own the chunk.")
        return
    }

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
        err = world.BuildStructure(mHandler.worldDB, payload.Structure)

        // TODO: only power and money part finished
        if payload.Structure.Power > 0 {
            user.PowerMax += int64(payload.Structure.Power)
        } else {
            user.Power += int64(-(payload.Structure.Power))
        }

        user.Money -= int64(payload.Structure.Cost)
        // TODO: Wait build finished
        user.MoneyRate += int64(payload.Structure.Money)
    //case Upgrade:
    case Destruct:
        err = world.DestuctStructure(mHandler.worldDB, payload.Structure)

        if payload.Structure.Power > 0 {
            user.PowerMax -= int64(payload.Structure.Power)
        } else {
            user.Power -= int64(-(payload.Structure.Power))
        }

        // Money back when destruct
        // TODO: calculate upgrade money
        user.Money += int64(payload.Structure.Cost) / 2
        user.MoneyRate -= int64(payload.Structure.Money)
    //case Repair:
    //case Restart:
    }

    // Handle world error
    if err != nil {
        log.Println(err)
        return
    }

    // Write user into database if no world error happened
    mHandler.playerDB.Put(payload.Username, user)
}
