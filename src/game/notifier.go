package game

import (
    "log"
    "comm"
    "game/player"
)

// This file is used to notify when data updates

type Notifier struct {
    online_players  map[string] player.Player
    playerDB        *player.PlayerDB            // Receive data changed players from MessageHandler
    pChanged        <-chan string
    pLogin          <-chan string
    pLogout         <-chan string               // used to send latest player data to servers
    mbus            *comm.MBusNode
}

func NewNotifier(playerDB *player.PlayerDB, mbus *comm.MBusNode, pChanged <-chan string, pLogin <-chan string, pLogout <-chan string) (notifier *Notifier) {
    online_players := make(map[string] player.Player)

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
    go func() {
        for {
            select {
            case user := <-notifier.pLogout:
                // Delete if user exist
                if _, ok := notifier.online_players[user]; ok {
                    delete(notifier.online_players, user)
                }
            case user := <-notifier.pLogin:
                // Add if user not exist
                if _, ok := notifier.online_players[user]; !ok {
                    p, err := notifier.playerDB.Get(user)
                    if err != nil {
                        log.Println(err)
                        continue
                    }

                    notifier.online_players[user] = p
                }
            case user := <-notifier.pChanged:
                // Update user status if user online
                if _, ok := notifier.online_players[user]; ok {
                    p, err := notifier.playerDB.Get(user)
                    if err != nil {
                        log.Println(err)
                        continue
                    }

                    notifier.online_players[user] = p
                }
            }
        }
    }()
}
