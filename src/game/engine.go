package game

import (
    "comm"
    "game/player"
)

type GameEngine struct {
    playerDB    *player.PlayerDB
    handler     *MessageHandler
    notifier    *Notifier
    mbus        *comm.MBusNode
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

    pChanged := make(chan string, 256)
    pLogin := make(chan string, 256)
    pLogout := make(chan string, 256)

    handler := NewMessageHandler(playerDB, mbus, pChanged, pLogin, pLogout)
    notifier := NewNotifier(playerDB, mbus, pChanged, pLogin, pLogout)

    engine = &GameEngine { playerDB: playerDB, handler: handler, notifier: notifier, mbus: mbus }

    return
}

func (engine GameEngine) Start() {
    engine.handler.Start()
    engine.notifier.Start()
}
