package game

import (
	"comm"
	"config"
	"encoding/json"
	"game/player"
	"game/world"
	"io/ioutil"
	"log"
	"path"
	"sync"
	"time"
	"util"
)

type ClientInfo struct {
	cid      int
	username string
}

type GameDB struct {
	playerDB *player.PlayerDB
	worldDB  *world.WorldDB
}

// Must use refrence type
type CommonData struct {
	online_players map[string]chan<- string
	chunk2Clients  map[util.Point][]ClientInfo // store clients who are watching this chunk
	client2Chunks  map[ClientInfo][]util.Point // store chunks where the client is watching
	owner_changed  chan string
	minimap        *MinimapData

	playerLock  *sync.RWMutex
	chunkLock   *sync.RWMutex
	clientLock  *sync.RWMutex
	minimapLock *sync.RWMutex
}

type GameEngine struct {
	GameDB
	CommonData

	handler  *MessageHandler
	notifier *Notifier
	mbus     *comm.MBusNode
}

func NewGameEngine() (engine *GameEngine, err error) {
	playerDB, err := player.NewPlayerDB(path.Join(config.DBDir, "pdb"))
	if err != nil {
		return
	}

	worldDB, err := world.NewWorldDB(path.Join(config.DBDir, "wdb"))
	if err != nil {
		return
	}

	gameDB := GameDB{playerDB, worldDB}

	online_players := make(map[string]chan<- string)
	chunk2Clients := make(map[util.Point][]ClientInfo)
	client2Chunks := make(map[ClientInfo][]util.Point)
	owner_changed := make(chan string, 256)

	var mmap_data MinimapData

	common_data := CommonData{
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

	handler := NewMessageHandler(gameDB, common_data, mbus)
	notifier := NewNotifier(gameDB, common_data, mbus)

	engine = &GameEngine{gameDB, common_data, handler, notifier, mbus}
	return
}

func (engine GameEngine) Start() {
	log.Println("[INFO] Starting game engine")

	// initialize minimap data & run timer for unfinished structure operation
	engine.minimap.Size = util.Size{50, 50}
	engine.minimap.Owner = make([][]string, 50)
	for i := 0; i < 50; i++ {
		engine.minimap.Owner[i] = make([]string, 50)
		for j := 0; j < 50; j++ {
			chk, err := engine.worldDB.Get(util.Point{i - 25, j - 25}.String())
			if err != nil {
				log.Fatalf("[ERROR] Map data corrupted at %s\n", util.Point{i - 25, j - 25}.String())
			}

			engine.minimap.Owner[i][j] = chk.Owner

			// Unfinished structures
			for _, s := range chk.Structures {
				currentTime := time.Now().Unix()
				if s.Status == world.Building || s.Status == world.Destructing {
					if s.UpdateTime+s.BuildTime <= currentTime {
						go UpdateChunk(engine.GameDB, chk.Key())
					} else {
						go func() {
							select {
							case <-time.After(time.Duration(s.BuildTime-(currentTime-s.UpdateTime)) * time.Second):
								UpdateChunk(engine.GameDB, chk.Key())
							}
						}()
					}
				}
			}
		}
	}

	log.Println("[INFO] Starting message handler")
	engine.handler.start()

	log.Println("[INFO] Starting notifier")
	engine.notifier.start()

	log.Println("[INFO] Game engine service available")
}

func (engine GameEngine) LoadTerrain(from util.Point, to util.Point, filename string) (err error) {
	mapdata := struct {
		Unit [][]world.TerrainType
	}{}

	mapjson, _ := ioutil.ReadFile(filename)
	json.Unmarshal(mapjson, &mapdata)

	for _, p := range util.InRange(from, to) {
		chunk, err := engine.worldDB.Get(p.String())
		if err != nil {
			chunk = *world.NewChunk(p)
		}

		for i := 0; uint(i) < world.ChunkSize.H; i++ {
			for j := 0; uint(j) < world.ChunkSize.W; j++ {
				real_X := (p.X+int(world.WorldSize.W)/2)*int(world.ChunkSize.W) + i
				real_Y := (p.Y+int(world.WorldSize.H)/2)*int(world.ChunkSize.H) + j
				chunk.Blocks[i][j].Terrain = mapdata.Unit[real_Y][real_X]
			}
		}

		chunk.Update()
		engine.worldDB.Load(p.String(), chunk)
	}

	return
}

// TODO: Maybe world lock & user lock needed to prevent something modify DB during this
func UpdateChunk(db GameDB, key string) (err error) {
	var username string
	var owner player.Player
	currentTime := time.Now().Unix()

	chunk, err := db.worldDB.Get(key)
	if err != nil {
		return
	}

	need_update := func() []world.Structure {
		var res []world.Structure
		for _, s := range chunk.Structures {
			if s.UpdateTime+s.BuildTime <= currentTime {
				res = append(res, s)
			}
		}

		return res
	}()

	if len(need_update) > 0 {
		username = chunk.Owner
		owner, err = db.playerDB.Get(username)
		if err != nil {
			return
		}

		for _, s := range need_update {
			index, _ := world.GetStructure(chunk, s)
			chunk.Structures[index].UpdateTime = currentTime
			chunk.Structures[index].BuildTime = 0
			switch s.Status {
			case world.Building:
				// Calcutate power
				if s.Power > 0 {
					owner.PowerMax += int64(s.Power)
				} else {
					owner.Power += int64(-(s.Power))
				}

				// Calculate money
				owner.MoneyRate += int64(s.Money)

				// Calculate population
				owner.PopulationCap += int64(s.PopulationCap)

				if s.Population > 0 {
					owner.PopulationRate += int64(s.Population)
				}
				chunk.Structures[index].Status = world.Running
			case world.Destructing:
				err = world.DestructStructure(&chunk, s)
				if err != nil {
					return
				}

				if s.Power > 0 {
					owner.PowerMax -= int64(s.Power)
				} else {
					owner.Power -= int64(-(s.Power))
				}

				if s.Population > 0 {
					owner.PopulationRate -= int64(s.Population)
				}

				// Change max population
				owner.PopulationCap -= int64(s.PopulationCap)

				// Money back when destruct
				// TODO: calculate upgrade money
				owner.Money += int64(s.Cost) / 2
				owner.MoneyRate -= int64(s.Money)
			}
		}

		db.playerDB.Put(username, owner)
		db.worldDB.Put(chunk.Key(), chunk)
	}

	return nil
}
