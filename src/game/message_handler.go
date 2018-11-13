package game

import (
    pler "game/player"
    "game/world"
    "log"
    "encoding/json"
    "comm"
    "time"
    "util"
)

// Function pointer type for common OnMessage function
type OnMessageFunc func([]byte)

// Use `NewMessageHandler` to constract a new message handler
type MessageHandler struct {
    OnMessage   map[comm.MsgType]OnMessageFunc
    playerDB    *pler.PlayerDB
    worldDB     *world.WorldDB
    mbus        *comm.MBusNode
    pChanged    chan<- string
    pLogin      chan<- string
    pLogout     chan<- string
}

func NewMessageHandler(playerDB *pler.PlayerDB, worldDB *world.WorldDB, mbus *comm.MBusNode, pChanged chan<- string, pLogin chan<- string, pLogout chan<- string) *MessageHandler {
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
        for msg := range mHandler.mbus.ReaderChan {
            payload := new(comm.Payload)

            if err := json.Unmarshal(msg, payload); err != nil {
                log.Println(err)
            } else {
                mHandler.OnMessage[payload.Msg_type](msg)
            }
        }
    }()
}

func toPayload(b []byte) (payload comm.Payload, err error) {
    err = json.Unmarshal(b, &payload)
    return
}

func (mHandler MessageHandler) OnLoginRequest(request []byte) {
    mHandler.OnPlayerDataRequest(request)

    payload, err := toPayload(request)
    if err != nil {
        log.Println(err)
        return
    }

    mHandler.pLogin <- payload.Username
}

func (mHandler MessageHandler) OnPlayerDataRequest(request []byte) {
    payload, err := toPayload(request)
    if err != nil {
        log.Println(err)
        return
    }

    username := payload.Username
    player, err := mHandler.playerDB.Get(username)
    if err != nil {
        // Username not found! Create new player in PlayerDB
        player.UpdateTime = time.Now().Unix()

        if err := mHandler.playerDB.Put(username, player); err != nil {
            log.Println(err)
        }
    }

    if !player.Initialized {
        if b, err := json.Marshal(comm.Payload { comm.HomePointRequest, username, "Please select the home point" }); err != nil {
            log.Println(err)
        } else {
            defer mHandler.mbus.Write("ws", b)
        }
    }

    // Send player data to WsServer
    payload.Msg_type = comm.PlayerDataResponse
    payload.Message = ""
    player_data := PlayerDataPayload { payload, player }

    if b, err := json.Marshal(player_data); err != nil {
        log.Println(err)
        return
    } else {
        mHandler.mbus.Write("ws", b)
    }
}

func (mHandler MessageHandler) OnLogoutRequest(request []byte) {
    payload, err := toPayload(request)
    if err != nil {
        log.Println(err)
        return
    }

    mHandler.pLogout <- payload.Username
}

func (mHandler MessageHandler) OnHomePointResponse(response []byte) {
    player_data := new(PlayerDataPayload)

    if err := json.Unmarshal(response, player_data); err != nil {
        log.Println(err)
        return
    }

    username := player_data.Username
    player, err := mHandler.playerDB.Get(username)
    if err != nil {
        log.Println(err)
        return
    }

    player.Home = player_data.Home
    player.Initialized = true
    player.UpdateTime = time.Now().Unix()

    if err := mHandler.playerDB.Put(username, player); err != nil {
        log.Println(err)
        return
    }

    mHandler.pChanged <- username
}

func (mHandler MessageHandler) OnMapDataRequest(request []byte) {
    var payload struct {
        comm.Payload
        Poss []util.Point
    }

    if err := json.Unmarshal(request, &payload); err != nil {
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
    payload.Message = ""
    map_data := MapDataPayload { payload.Payload, chunks }

    if b, err := json.Marshal(map_data); err != nil {
        log.Println(err)
        return
    } else {
        mHandler.mbus.Write("ws", b)
    }
}
