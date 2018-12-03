package game

import (
    "util"
    "comm"
    "game/player"
    "game/world"
    "io/ioutil"
    "encoding/json"
)

type ClientInfo struct {
    cid         int
    username    string
}

type GameDB struct {
    playerDB    *player.PlayerDB
    worldDB     *world.WorldDB
 }

type CommonData struct {
    online_players  map[string] chan<- string
    chunk2Clients   map[util.Point] []ClientInfo    // store clients who are watching this chunk
    client2Chunks   map[ClientInfo] []util.Point    // store chunks where the client is watching
}

type GameEngine struct {
    GameDB
    CommonData

    handler     *MessageHandler
    notifier    *Notifier
    mbus        *comm.MBusNode
}

func NewGameEngine() (engine *GameEngine, err error) {
    // TODO: use config to determine DB location
    playerDB, err := player.NewPlayerDB("/tmp/gdb/pdb")
    if err != nil {
        return
    }

    worldDB, err := world.NewWorldDB("/tmp/gdb/wdb")
    if err != nil {
        return
    }

    gameDB := GameDB { playerDB, worldDB }

    online_players := make(map[string] chan<- string)
    chunk2Clients  := make(map[util.Point] []ClientInfo)
    client2Chunks  := make(map[ClientInfo] []util.Point)

    common_data := CommonData { online_players, chunk2Clients, client2Chunks }

    mbus, err := comm.NewMBusNode("game")
    if err != nil {
        return
    }

    dChanged := make(chan ClientInfo, 256)
    pLogin   := make(chan ClientInfo, 256)
    pLogout  := make(chan ClientInfo, 256)

    handler  := NewMessageHandler(gameDB, common_data, mbus, dChanged, pLogin, pLogout)
    notifier := NewNotifier(gameDB, common_data, mbus, dChanged, pLogin, pLogout)

    engine = &GameEngine { gameDB, common_data, handler, notifier, mbus }
    return
}

func (engine GameEngine) Start() {
    engine.handler.start()
    engine.notifier.Start()
}

func (engine GameEngine) LoadTerrain(from util.Point, to util.Point, filename string) (err error) {
    mapdata := struct {
        Unit [][]world.TerrainType
    } {}

    mapjson, _ := ioutil.ReadFile(filename)
    json.Unmarshal(mapjson, &mapdata)

    for _, p := range util.InRange(from, to) {
        chunk, err := engine.worldDB.Get(p.String())
        if err != nil {
            chunk = *world.NewChunk(p)
        }

        for i := 0; uint(i) < world.ChunkSize.H; i++ {
            for j := 0; uint(j) < world.ChunkSize.W; j++ {
                real_X := (p.X + int(world.WorldSize.W) / 2) * int(world.ChunkSize.W) + i
                real_Y := (p.Y + int(world.WorldSize.H) / 2) * int(world.ChunkSize.H) + j
                chunk.Blocks[i][j].Terrain = mapdata.Unit[real_Y][real_X]
            }
        }

        chunk.Update()
        engine.worldDB.Put(p.String(), chunk)
    }

    return
}
