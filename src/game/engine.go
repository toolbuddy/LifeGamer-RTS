package game

import (
    "comm"
    "game/player"
)

type GameEngine struct {
    online_players  []player.Player
    playerDB        *player.PlayerDB
    handler         *MessageHandler
    mbus            *comm.MBusNode
}

func NewGameEngine() (engine *GameEngine, err error) {
    online_players = []player.Player {  }

    // TODO: use config to determine DB location
    playerDB, err := player.NewPlayerDB("/tmp/pdb")

    if err != nil {
        return
    }

    mbus, err = comm.NewMBusNode("game")

    if err != nil {
        return
    }

    handler = NewMessageHandler(playerDB, mbus)

    engine = &GameEngine { online_players: online_players, playerDB: playerDB, handler: handler, mbus: mbus }

    return
}

func (engine GameEngine) Start() {
    engine.handler.Start()
}
