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

// Function pointer type for common OnMessage function
type OnMessageFunc func(comm.MessageWrapper)

// Use `NewMessageHandler` to constract a new message handler
type MessageHandler struct {
    OnMessage   map[comm.MsgType] OnMessageFunc
    playerDB    *player.PlayerDB
    worldDB     *world.WorldDB
    mbus        *comm.MBusNode
    pChanged    chan<- ClientInfo
    pLogin      chan<- ClientInfo
    pLogout     chan<- ClientInfo
}

func NewMessageHandler(playerDB *player.PlayerDB, worldDB *world.WorldDB, mbus *comm.MBusNode, pChanged chan<- ClientInfo, pLogin chan<- ClientInfo, pLogout chan<- ClientInfo) *MessageHandler {
    mHandler := &MessageHandler {
        OnMessage: make(map[comm.MsgType]OnMessageFunc),
        playerDB: playerDB,
        worldDB: worldDB,
        mbus: mbus,
        pChanged: pChanged,
        pLogin: pLogin,
        pLogout: pLogout,
    }

    mHandler.OnMessage[comm.LoginRequest]      = mHandler.OnLoginRequest
    mHandler.OnMessage[comm.PlayerDataRequest] = mHandler.OnPlayerDataRequest
    mHandler.OnMessage[comm.LogoutRequest]     = mHandler.OnLogoutRequest
    mHandler.OnMessage[comm.HomePointResponse] = mHandler.OnHomePointResponse
    mHandler.OnMessage[comm.MapDataRequest]    = mHandler.OnMapDataRequest

    return mHandler
}

func (mHandler MessageHandler) Start() {
    go func() {
        for msg_wrapper := range mHandler.mbus.ReaderChan {
            var payload comm.Payload

            if err := json.Unmarshal(msg_wrapper.Data, &payload); err != nil {
                log.Println(err)
                continue
            }

            mHandler.OnMessage[payload.Msg_type](msg_wrapper)
        }
    }()
}

func (mHandler MessageHandler) OnLoginRequest(request comm.MessageWrapper) {
    mHandler.OnPlayerDataRequest(request)
    mHandler.pLogin <- ClientInfo { request.Cid, request.Username }
}

func (mHandler MessageHandler) OnPlayerDataRequest(request comm.MessageWrapper) {
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
            msg.Sendto = comm.SendToClient
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
    msg.Sendto = comm.SendToClient
    msg.Data = b

    mHandler.mbus.Write("ws", msg)
}

func (mHandler MessageHandler) OnLogoutRequest(request comm.MessageWrapper) {
    mHandler.pLogout <- ClientInfo { request.Cid, request.Username }
}

func (mHandler MessageHandler) OnHomePointResponse(response comm.MessageWrapper) {
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

    player_data.Home = payload.Home
    player_data.Initialized = true
    player_data.UpdateTime = time.Now().Unix()

    if err := mHandler.playerDB.Put(username, player_data); err != nil {
        log.Println(err)
        return
    }

    mHandler.pChanged <- ClientInfo { response.Cid, response.Username }
}

func (mHandler MessageHandler) OnMapDataRequest(request comm.MessageWrapper) {
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
        chunk, err := mHandler.worldDB.Get(pos.String());
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
    msg.Sendto = comm.SendToClient
    msg.Data = b

    mHandler.mbus.Write("ws", msg)
}
