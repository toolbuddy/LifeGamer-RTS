package game

import (
    "log"
    "time"
    "comm"
    "strconv"
    "game/player"
)

// This file is used to notify when data updates

type Notifier struct {
    online_players  map[string] chan<- string
    playerDB        *player.PlayerDB            // Receive data changed players from MessageHandler
    pChanged        <-chan string
    pLogin          <-chan string
    pLogout         <-chan string               // used to send latest player data to servers
    mbus            *comm.MBusNode
}

func NewNotifier(playerDB *player.PlayerDB, mbus *comm.MBusNode, pChanged <-chan string, pLogin <-chan string, pLogout <-chan string) (notifier *Notifier) {
    online_players := make(map[string] chan<- string)

    notifier = &Notifier {
        online_players: online_players,
        playerDB: playerDB,
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
            case user := <-notifier.pLogout:
                // Delete if user exist
                if pch, ok := notifier.online_players[user]; ok {
                    delete(notifier.online_players, user)
                    close(pch)
                }
            case user := <-notifier.pLogin:
                // Add if user not exist
                if _, ok := notifier.online_players[user]; !ok {
                    pch := make(chan string, 1)
                    go notify_loop(pch, user, notifier.mbus, notifier.playerDB)
                    notifier.online_players[user] = pch
                }
            case user := <-notifier.pChanged:
                // Update user status if user still online
                if pch, ok := notifier.online_players[user]; ok {
                    pch <- "update"
                }
            }
        }
    }()

    // start ticker
    //TODO: Check if here have race condition
    go func() {
        for range time.NewTicker(time.Second).C {
            for _, ch := range notifier.online_players {
                ch <- "tick"
            }
        }
    }()
}

func notify_loop(ch <-chan string, user string, mbus *comm.MBusNode, db *player.PlayerDB) {
    p, err := db.Get(user)
    if err != nil {
        log.Println(err)
        return
    }

    // Update player's data
    p.Update()

    for m := range ch {
        switch m {
        case "tick":
            p.Update()
        case "update":
            p, err = db.Get(user)
            p.Update()
        }
        // TODO: use real communication format
        mbus.Write("ws", []byte(user + "'s money: " + strconv.FormatInt(p.Money, 10)))
    }
    log.Println("Notify loop stopped")
}
