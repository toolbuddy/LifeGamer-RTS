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
    online_players  map[string] chan<- string
    playerDB        *player.PlayerDB            // Receive data changed players from MessageHandler
    worldDB         *world.WorldDB
    dChanged        <-chan ClientInfo
    pLogin          <-chan ClientInfo
    pLogout         <-chan ClientInfo           // used to send latest player data to servers
    mbus            *comm.MBusNode
}

func NewNotifier(playerDB *player.PlayerDB, worldDB *world.WorldDB, mbus *comm.MBusNode, dChanged <-chan ClientInfo, pLogin <-chan ClientInfo, pLogout <-chan ClientInfo) (notifier *Notifier) {
    online_players := make(map[string] chan<- string)

    notifier = &Notifier {
        online_players: online_players,
        playerDB: playerDB,
        worldDB: worldDB,
        dChanged: dChanged,
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
                username := client_info.username

                // Delete if user exist
                if user_ch, ok := notifier.online_players[username]; ok {
                    close(user_ch)
                    delete(notifier.online_players, username)
                }
            case client_info := <-notifier.pLogin:
                username := client_info.username

                // Add if user not exist
                if _, ok := notifier.online_players[username]; !ok {
                    user_ch := make(chan string, 1)
                    notifier.online_players[username] = user_ch

                    go notify_loop(client_info, user_ch, notifier.mbus, notifier.playerDB)
                }
            case client_info := <-notifier.dChanged:
                username := client_info.username

                // Update user status if user still online
                if user_ch, ok := notifier.online_players[username]; ok {
                    user_ch <- "update"
                }
            }
        }
    }()

    // start ticker
    //TODO: Check if here have race condition
    go func() {
        for range time.NewTicker(time.Second).C {
            for _, user_ch := range notifier.online_players {
                user_ch <- "tick"
            }
        }
    }()
}

func notify_loop(client_info ClientInfo, user_ch <-chan string, mbus *comm.MBusNode, db *player.PlayerDB) {
    cid := client_info.cid
    username := client_info.username

    player_data, err := db.Get(username)
    if err != nil {
        log.Println(err)
        return
    }

    // Update player's data
    player_data.Update()

    for m := range user_ch {
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

        msg := comm.MessageWrapper { cid, username, comm.SendToUser, b }

        mbus.Write("ws", msg)
    }

    log.Println("Notifier: Notify loop stopped")
}
