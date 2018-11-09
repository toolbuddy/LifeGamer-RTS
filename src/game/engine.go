package game

import (
    "comm"
    "game/player"
)

type GameEngine struct {
    playerDB        *player.PlayerDB
    handler         *MessageHandler
    mbus            *comm.MBusNode
}

func NewGameEngine() (engine *GameEngine, err error) {

    // TODO: use config to determine DB location
    playerDB, err := player.NewPlayerDB("/tmp/pdb")
    if err != nil {
        return
    }

    mbus, err := comm.NewMBusNode("game")
    if err != nil {
        return
    }

    pChanged := make(chan string)
    pLogin := make(chan string)
    pLogout := make(chan string)

    handler := NewMessageHandler(playerDB, mbus, pChanged, pLogin, pLogout)

    engine = &GameEngine { playerDB: playerDB, handler: handler, mbus: mbus }

    return
}

func (engine GameEngine) Start() {
    engine.handler.Start()
}
