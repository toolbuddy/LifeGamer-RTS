package game

import (
    "log"
    "time"
    "comm"
    "game/player"
    "game/world"
    "encoding/json"
)

// This file is used to notify when data updates

type Notifier struct {
    online_players  map[string] map[int] chan<- string
    playerDB        *player.PlayerDB                    // Receive data changed players from MessageHandler
    worldDB         *world.WorldDB
    pChanged        <-chan ClientInfo
    pLogin          <-chan ClientInfo
    pLogout         <-chan ClientInfo                   // used to send latest player data to servers
    mbus            *comm.MBusNode
}

func NewNotifier(playerDB *player.PlayerDB, worldDB *world.WorldDB, mbus *comm.MBusNode, pChanged <-chan ClientInfo, pLogin <-chan ClientInfo, pLogout <-chan ClientInfo) (notifier *Notifier) {
    online_players := make(map[string] map[int] chan<- string)

    notifier = &Notifier {
        online_players: online_players,
        playerDB: playerDB,
        worldDB: worldDB,
        pChanged: pChanged,
        pLogin: pLogin,
        pLogout: pLogout,
        mbus: mbus,
    }

    return
}

func (notifier Notifier) Start() {
    // Handles user login and logout
    go func() {
        for {
            select {
            case client_info := <-notifier.pLogout:
                cid := client_info.cid
                username := client_info.username

                // Delete if user exist
                if user, ok := notifier.online_players[username]; ok {
                    if client_ch, ok := user[cid]; ok {
                        close(client_ch)
                        delete(user, cid)
                    }
                }
            case client_info := <-notifier.pLogin:
                cid := client_info.cid
                username := client_info.username

                // Add if user not exist
                user, ok := notifier.online_players[username]
                if !ok {
                    user = make(map[int] chan<- string)
                    notifier.online_players[username] = user
                }

                if _, ok := user[cid]; !ok {
                    ch := make(chan string, 1)
                    user[cid] = ch

                    go notify_loop(ch, client_info, notifier.mbus, notifier.playerDB)
                }
            case client_info := <-notifier.pChanged:
                cid := client_info.cid
                username := client_info.username

                // Update user status if user still online
                if user, ok := notifier.online_players[username]; ok {
                    if client_ch, ok := user[cid]; ok {
                        client_ch <- "update"
                    }
                }
            }
        }
    }()

    // start ticker
    //TODO: Check if here have race condition
    go func() {
        for range time.NewTicker(time.Second).C {
            for _, user := range notifier.online_players {
                for _, client_ch := range user {
                    client_ch <- "tick"
                }
            }
        }
    }()
}

func notify_loop(ch <-chan string, client_info ClientInfo, mbus *comm.MBusNode, db *player.PlayerDB) {
    cid := client_info.cid
    username := client_info.username

    player_data, err := db.Get(username)
    if err != nil {
        log.Println(err)
        return
    }

    // Update player's data
    player_data.Update()

    for m := range ch {
        switch m {
        case "tick":
            player_data.Update()
        case "update":
            player_data, err = db.Get(username)
            player_data.Update()
        }

        // TODO: use real communication format
        b, err := json.Marshal(PlayerDataPayload { comm.Payload { comm.PlayerDataResponse, username }, player_data })
        if err != nil {
            log.Println(err)
            continue
        }

        msg := comm.MessageWrapper { cid, username, comm.SendByClient, b }

        mbus.Write("ws", msg)
    }

    log.Println("Notifier: Notify loop stopped")
}
