package game

import (
    "game/player"
    "game/world"
    "log"
    "encoding/json"
    "comm"
    "time"
    "util"
)

// Function pointer type for common onMessage function
type OnMessageFunc func(comm.MessageWrapper)

// Use `NewMessageHandler` to constract a new message handler
type MessageHandler struct {
    onMessage       map[comm.MsgType] OnMessageFunc
    playerDB        *player.PlayerDB
    worldDB         *world.WorldDB
    mbus            *comm.MBusNode
    dChanged        chan<- ClientInfo
    pLogin          chan<- ClientInfo
    pLogout         chan<- ClientInfo
    chunk2Clients   map[util.Point] []ClientInfo        // store clients who are watching this chunk
    client2Chunks   map[ClientInfo] []util.Point        // store chunks where the client is watching
}

func NewMessageHandler(playerDB *player.PlayerDB, worldDB *world.WorldDB, mbus *comm.MBusNode, dChanged chan<- ClientInfo, pLogin chan<- ClientInfo, pLogout chan<- ClientInfo) *MessageHandler {
    mHandler := &MessageHandler {
        onMessage: make(map[comm.MsgType] OnMessageFunc),
        playerDB: playerDB,
        worldDB: worldDB,
        mbus: mbus,
        dChanged: dChanged,
        pLogin: pLogin,
        pLogout: pLogout,
        chunk2Clients: make(map[util.Point] []ClientInfo),
        client2Chunks: make(map[ClientInfo] []util.Point),
    }

    mHandler.onMessage[comm.LoginRequest]      = mHandler.onLoginRequest
    mHandler.onMessage[comm.PlayerDataRequest] = mHandler.onPlayerDataRequest
    mHandler.onMessage[comm.LogoutRequest]     = mHandler.onLogoutRequest
    mHandler.onMessage[comm.HomePointResponse] = mHandler.onHomePointResponse
    mHandler.onMessage[comm.MapDataRequest]    = mHandler.onMapDataRequest

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

            mHandler.onMessage[payload.Msg_type](msg_wrapper)
        }
    }()
}

func (mHandler MessageHandler) onLoginRequest(request comm.MessageWrapper) {
    mHandler.onPlayerDataRequest(request)
    mHandler.pLogin <- ClientInfo { request.Cid, request.Username }
}

func (mHandler MessageHandler) onPlayerDataRequest(request comm.MessageWrapper) {
    username := request.Username

    player_data, err := mHandler.playerDB.Get(username)
    if err != nil {
        // Username not found! Create new player in PlayerDB
        player_data.UpdateTime = time.Now().Unix()

        if err := mHandler.playerDB.Put(username, player_data); err != nil {
            log.Println(err)
        }
    }

    if !player_data.Initialized {
        if b, err := json.Marshal(comm.Payload { comm.HomePointRequest, username }); err != nil {
            log.Println(err)
        } else {
            msg := request
            msg.SendTo = comm.SendToClient
            msg.Data = b

            defer mHandler.mbus.Write("ws", msg)
        }
    }

    // Send player data to WsServer
    payload := PlayerDataPayload { comm.Payload { comm.PlayerDataResponse, username }, player_data }

    b, err := json.Marshal(payload)
    if err != nil {
        log.Println(err)
        return
    }

    msg := request
    msg.SendTo = comm.SendToClient
    msg.Data = b

    mHandler.mbus.Write("ws", msg)
}

func (mHandler MessageHandler) onLogoutRequest(request comm.MessageWrapper) {
    mHandler.pLogout <- ClientInfo { request.Cid, request.Username }
}

func (mHandler MessageHandler) onHomePointResponse(response comm.MessageWrapper) {
    username := response.Username

    player_data, err := mHandler.playerDB.Get(username)
    if err != nil {
        log.Println(err)
        return
    }

    var payload struct {
        comm.Payload
        Home util.Point
    }

    if err := json.Unmarshal(response.Data, &payload); err != nil {
        log.Println(err)
        return
    }

    player_data.Update()
    player_data.Home = payload.Home
    player_data.Initialized = true

    if err := mHandler.playerDB.Put(username, player_data); err != nil {
        log.Println(err)
        return
    }

    mHandler.dChanged <- ClientInfo { response.Cid, response.Username }
}

func (mHandler *MessageHandler) onMapDataRequest(request comm.MessageWrapper) {
    var payload struct {
        comm.Payload
        Poss []util.Point
    }

    if err := json.Unmarshal(request.Data, &payload); err != nil {
        log.Println(err)
        return
    }

    var chunks []world.Chunk = []world.Chunk {}

    for _, pos := range payload.Poss {
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
        if poss, ok := mHandler.client2Chunks[client_info]; ok {
            for _, pos := range poss {
                // Remove client from chunk watching list
                if infos, ok := mHandler.chunk2Clients[pos]; ok {
                    for i, info := range infos {
                        if info == client_info {
                            mHandler.chunk2Clients[pos] = append(infos[:i], infos[i+1:]...)
                        }
                    }
                }
            }
        }

        // Update clients who are watching this chunk
        for _, pos := range payload.Poss {
            if infos, ok := mHandler.chunk2Clients[pos]; !ok {
                mHandler.chunk2Clients[pos] = []ClientInfo { client_info }
            } else {
                mHandler.chunk2Clients[pos] = append(infos, client_info)
            }
        }

        // Update where the client is watching
        mHandler.client2Chunks[client_info] = payload.Poss
    }
}

func (mHandler MessageHandler) mapDataUpdate(poss []util.Point) {
    client2chunk := make(map[ClientInfo] []world.Chunk)

    for _, pos := range poss {
        if infos, ok := mHandler.chunk2Clients[pos]; ok {
            chunk, err := mHandler.worldDB.Get(pos.String())
            if err != nil {
                log.Println(err)
                continue
            }

            for _, info := range infos {
                if chunks, ok := client2chunk[info]; !ok {
                    client2chunk[info] = []world.Chunk { chunk }
                } else {
                    client2chunk[info] = append(chunks, chunk)
                }
            }
        }
    }

    for info, chunks := range client2chunk {
        payload := comm.Payload { Msg_type: comm.MapDataResponse, Username: info.username }
        map_data := MapDataPayload { payload, chunks }

        b, err := json.Marshal(map_data)
        if err != nil {
            log.Println(err)
            continue
        }

        msg := comm.MessageWrapper { info.cid, info.username, comm.SendToClient, b }

        mHandler.mbus.Write("ws", msg)
    }
}
