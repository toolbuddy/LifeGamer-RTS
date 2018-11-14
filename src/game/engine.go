package game

import (
    "comm"
    "game/player"
    "game/world"
)

type ClientInfo struct {
    cid         int
    username    string
}

type GameEngine struct {
    playerDB    *player.PlayerDB
    worldDB     *world.WorldDB
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

    worldDB, err := world.NewWorldDB("/tmp/wdb")
    if err != nil {
        return
    }

    mbus, err := comm.NewMBusNode("game")
    if err != nil {
        return
    }

    pChanged := make(chan ClientInfo, 256)
    pLogin   := make(chan ClientInfo, 256)
    pLogout  := make(chan ClientInfo, 256)

    handler  := NewMessageHandler(playerDB, worldDB, mbus, pChanged, pLogin, pLogout)
    notifier := NewNotifier(playerDB, worldDB, mbus, pChanged, pLogin, pLogout)

    engine = &GameEngine { playerDB: playerDB, worldDB: worldDB, handler: handler, notifier: notifier, mbus: mbus }
    return
}

func (engine GameEngine) Start() {
    engine.handler.Start()
    engine.notifier.Start()
}
