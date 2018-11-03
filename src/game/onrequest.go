package game

import (
    pler "game/player"
    "log"
    "encoding/json"
    "comm"
)

var playerDB *pler.PlayerDB
var mbus *comm.MBusNode

func Test() {
    playerDB, _ = pler.NewPlayerDB("./PlayerDB")
    mbus, _ = comm.NewMBusNode("game")

    go func() {
        for msg := range mbus.ReaderChan {
            payload := new(comm.Payload)

            if err := json.Unmarshal(msg, payload); err != nil {
                log.Println(err)
            } else {
                switch payload.Msg_type {
                case comm.PlayerDataRequest:
                    go OnPlayerDataRequest(msg)
                case comm.HomePointResponse:
                    go OnHomePointResponse(msg)
                }
            }
        }
    }()
}

func toPayload(b []byte) (payload comm.Payload, err error) {
    err = json.Unmarshal(b, &payload)
    return
}

func OnPlayerDataRequest(request []byte) {
    payload, err := toPayload(request)

    if err != nil {
        log.Println(err)
        return
    }

    username := payload.Username
    player, err := playerDB.Get(username)

    if err != nil {
        // Username not found! Create new player in PlayerDB
        if err := playerDB.Put(username, player); err != nil {
            log.Println(err)
        }
    }

    if !player.Initialized {
        if b, err := json.Marshal(comm.Payload { comm.HomePointRequest, username, "Please select the home point" }); err != nil {
            log.Println(err)
        } else {
            defer mbus.Write("ws", b)
        }
    }

    // Send player data to WsServer
    payload.Msg_type = comm.PlayerDataResponse
    playerData := comm.PlayerDataPayload { payload, player.Home, player.Human, player.Money, player.Power }

    if b, err := json.Marshal(playerData); err != nil {
        log.Println(err)
        return
    } else {
        mbus.Write("ws", b)
    }
}

func OnHomePointResponse(response []byte) {
    playerData := new(comm.PlayerDataPayload)

    if err := json.Unmarshal(response, playerData); err != nil {
        log.Println(err)
        return
    }

    username := playerData.Username
    player, err := playerDB.Get(username)

    if err != nil {
        log.Println(err)
        return
    }

    player.Home = playerData.Home
    player.Initialized = true

    if err := playerDB.Put(username, player); err != nil {
        log.Println(err)
        return
    }
}
