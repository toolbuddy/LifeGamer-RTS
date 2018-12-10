package game

import (
    "util"
    "comm"
    "game/player"
    "game/world"
    "io/ioutil"
    "encoding/json"
    "sync"
    "log"
    "config"
)

type ClientInfo struct {
    cid         int
    username    string
}

type GameDB struct {
    playerDB    *player.PlayerDB
    worldDB     *world.WorldDB
}

// Must use refrence type
type CommonData struct {
    online_players  map[string] chan<- string
    chunk2Clients   map[util.Point] []ClientInfo    // store clients who are watching this chunk
    client2Chunks   map[ClientInfo] []util.Point    // store chunks where the client is watching
    owner_changed   chan string
    minimap         *MinimapData

    playerLock      *sync.RWMutex
    chunkLock       *sync.RWMutex
    clientLock      *sync.RWMutex
    minimapLock     *sync.RWMutex
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
    playerDB, err := player.NewPlayerDB(config.PdbPath)
    if err != nil {
        return
    }

    worldDB, err := world.NewWorldDB(config.WdbPath)
    if err != nil {
        return
    }

    gameDB := GameDB { playerDB, worldDB }

    online_players := make(map[string] chan<- string)
    chunk2Clients  := make(map[util.Point] []ClientInfo)
    client2Chunks  := make(map[ClientInfo] []util.Point)
    owner_changed  := make(chan string, 256)

    var mmap_data MinimapData

    common_data := CommonData {
        online_players,
        chunk2Clients,
        client2Chunks,
        owner_changed,
        &mmap_data,
        new(sync.RWMutex),
        new(sync.RWMutex),
        new(sync.RWMutex),
        new(sync.RWMutex),
    }

    mbus, err := comm.NewMBusNode("game")
    if err != nil {
        return
    }

    handler  := NewMessageHandler(gameDB, common_data, mbus)
    notifier := NewNotifier(gameDB, common_data, mbus)

    engine = &GameEngine { gameDB, common_data, handler, notifier, mbus }
    return
}

func (engine GameEngine) Start() {
    log.Println("Initializing game engine")

    // initialize minimap data
    engine.CommonData.minimap.Size = util.Size { 50, 50 }
    engine.CommonData.minimap.Owner = make([][]string, 50)
    for i := 0;i < 50;i++ {
        engine.CommonData.minimap.Owner[i] = make([]string, 50)
        for j := 0;j < 50;j++ {
            chk, err := engine.worldDB.Get(util.Point{i - 25, j - 25}.String())
            if err != nil {
                log.Fatalln(err)
            }

            engine.CommonData.minimap.Owner[i][j] = chk.Owner
        }
    }

    log.Println("Starting message handler")
    engine.handler.start()

    log.Println("Starting notifier")
    engine.notifier.start()

    log.Println("Initializing game engine done")
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
        engine.worldDB.Load(p.String(), chunk)
    }

    return
}
