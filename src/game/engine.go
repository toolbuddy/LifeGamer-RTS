package game

import (
	"comm"
	"config"
	"encoding/json"
	"game/player"
	"game/world"
	"io/ioutil"
	"log"
	"math/rand"
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

	onlineLock  *sync.RWMutex
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
	rand.Seed(time.Now().UTC().UnixNano())

	// initialize minimap data & run timer for unfinished structure operation
	log.Println("[INFO] Initializing map data")
	engine.minimap.Size = util.Size{50, 50}
	engine.minimap.Owner = make([][]string, 50)
	engine.minimap.Terrain = make([][]world.TerrainType, 50)
	for i := 0; i < 50; i++ {
		engine.minimap.Owner[i] = make([]string, 50)
		engine.minimap.Terrain[i] = make([]world.TerrainType, 50)
		for j := 0; j < 50; j++ {
			chk, err := engine.worldDB.Get(util.Point{i - 25, j - 25}.String())
			if err != nil {
				log.Fatalf("[ERROR] Map data corrupted at %s\n", util.Point{i - 25, j - 25}.String())
			}

			engine.minimap.Owner[i][j] = chk.Owner

			// Calculate terrain
			engine.minimap.Terrain[i][j] = func() world.TerrainType {
				var t_most world.TerrainType
				var t_num int
				t_map := make(map[world.TerrainType]int)
				for ci := 0; ci < 16; ci++ {
					for cj := 0; cj < 16; cj++ {
						t_map[chk.Blocks[ci][cj].Terrain]++
					}
				}
				for k, v := range t_map {
					if v > t_num {
						t_most = k
					}
				}
				return t_most
			}()

			// ->unfinished-> structures
			for _, s := range chk.Structures {
				currentTime := time.Now().Unix()
				if s.Status == world.Building || s.Status == world.Destructing {
					if s.UpdateTime+s.BuildTime <= currentTime {
						go UpdateChunk(engine.GameDB, chk.Owner, chk.Key())
					} else {
						go func() {
							select {
							case <-time.After(time.Duration(s.BuildTime-(currentTime-s.UpdateTime)) * time.Second):
								UpdateChunk(engine.GameDB, chk.Owner, chk.Key())
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

	log.Println("[INFO] Starting population updater")
	go func() {
		for {
			// Update every 10 min
			<-time.After(time.Until(time.Now().Add(time.Second * 2).Truncate(time.Second * 2)))
			engine.UpdatePopulation()
		}
	}()

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

		for i := 0; uint(i) < world.ChunkSize.W; i++ {
			for j := 0; uint(j) < world.ChunkSize.H; j++ {
				real_X := (p.X+int(world.WorldSize.W)/2)*int(world.ChunkSize.W) + i
				real_Y := (p.Y+int(world.WorldSize.H)/2)*int(world.ChunkSize.H) + j
				chunk.Blocks[i][j].Terrain = mapdata.Unit[real_X][real_Y]
			}
		}

		engine.worldDB.Load(p.String(), chunk)
	}

	return
}

func (engine GameEngine) UpdatePopulation() {
	iter := engine.playerDB.NewIterator(nil, nil)
	for iter.Next() {
		username := string(iter.Key())

		engine.playerDB.Lock(username)

		owner, err := engine.playerDB.Get(username)
		if err != nil {
			continue
		}

		for _, pos := range owner.Territory {
			engine.worldDB.Lock(pos.String())

			chunk, err := engine.worldDB.Get(pos.String())
			if err != nil {
				engine.worldDB.Unlock(pos.String())
				log.Println("[WARNING]", err)
				continue
			}

			if owner.PopulationCap-owner.Population >= chunk.PopulationRate {
				// Population cap still enough
				chunk.Population += chunk.PopulationRate
				owner.Population += chunk.PopulationRate
			} else {
				chunk.Population += owner.PopulationCap - owner.Population
				owner.Population = owner.PopulationCap
			}

			engine.worldDB.Put(chunk.Key(), chunk)
			engine.worldDB.Unlock(pos.String())
		}

		engine.playerDB.Put(username, owner)
		engine.playerDB.Unlock(username)
	}
}

func UpdateChunk(db GameDB, username string, key string) (err error) {
	var owner player.Player
	currentTime := time.Now().Unix()

	db.playerDB.Lock(username)
	defer db.playerDB.Unlock(username)

	db.worldDB.Lock(key)
	defer db.worldDB.Unlock(key)

	chunk, err := db.worldDB.Get(key)
	if err != nil {
		return
	}

	// Ownership changed
	if chunk.Owner != username {
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
					chunk.PopulationRate += int64(s.Population)
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

				owner.MoneyRate -= int64(s.Money)

				if s.Population > 0 {
					chunk.PopulationRate -= int64(s.Population)
				}

				// Change max population
				owner.PopulationCap -= int64(s.PopulationCap)

				// Money back when destruct
				// TODO: calculate upgrade money
				owner.Money += int64(s.Cost) / 2
			}
		}

		db.playerDB.Put(username, owner)
		db.worldDB.Put(chunk.Key(), chunk)
	}

	return nil
}

// TODO: return error
func HaltPlayer(db GameDB, username string) {
	db.playerDB.Lock(username)

	owner, err := db.playerDB.Get(username)
	if err != nil {
		return
	}

	owner.Update()

	if owner.Money < 0 {
		owner.Money = 0
	}

	for _, pos := range owner.Territory {
		db.worldDB.Lock(pos.String())

		chunk, err := db.worldDB.Get(pos.String())
		if err != nil {
			db.worldDB.Unlock(pos.String())
			log.Println("[WARNING]", err)
			continue
		}

		for s_index, str := range chunk.Structures {
			if str.Status == world.Running {
				chunk.Structures[s_index].Status = world.Halted

				// Stop functions
				if str.Power > 0 {
					owner.PowerMax -= int64(str.Power)
				} else {
					owner.Power -= int64(-(str.Power))
				}

				owner.MoneyRate -= int64(str.Money)

				if str.Population > 0 {
					chunk.PopulationRate -= int64(str.Population)
				}

				owner.PopulationCap -= int64(str.PopulationCap)
			}
		}

		db.worldDB.Put(chunk.Key(), chunk)
		db.worldDB.Unlock(pos.String())
	}

	db.playerDB.Put(username, owner)
	db.playerDB.Unlock(username)
}

// TODO: return error
func HaltChunk(db GameDB, username string, key string) {
	log.Println("halt", key)
	db.playerDB.Lock(username)
	defer db.playerDB.Unlock(username)

	owner, err := db.playerDB.Get(username)
	if err != nil {
		return
	}

	owner.Update()

	db.worldDB.Lock(key)
	defer db.worldDB.Unlock(key)

	chunk, err := db.worldDB.Get(key)
	if err != nil {
		log.Println("[WARNING]", err)
	}

	var status_changed bool = false

	for s_index, str := range chunk.Structures {
		if str.Status == world.Running {
			chunk.Structures[s_index].Status = world.Halted
			status_changed = true

			// Stop functions
			if str.Power > 0 {
				owner.PowerMax -= int64(str.Power)
			} else {
				owner.Power -= int64(-(str.Power))
			}

			owner.MoneyRate -= int64(str.Money)

			if str.Population > 0 {
				chunk.PopulationRate -= int64(str.Population)
			}

			owner.PopulationCap -= int64(str.PopulationCap)
		}
	}

	// Only write DB on status changed to prevent halt loop
	if status_changed {
		db.worldDB.Put(chunk.Key(), chunk)
		db.playerDB.Put(username, owner)
	}
}

func Battle(troopAtk, troopDef int) (remainAtk, remainDef int) {
	// 1.60 : 1 = Def win
	// 1.80 : 1 = Atk win

	// Formula : base_damage + random * damage_buff
	defRatio := 0.14 + rand.Float64()*0.02 // 0.14 ~ 0.16
	atkRatio := 0.04 + rand.Float64()*0.02 // 0.04 ~ 0.06

	troopAtkF, troopDefF := float64(troopAtk), float64(troopDef)

	// Remaining number may vary
	remainAtk = int(troopAtkF - troopDefF*defRatio)
	remainDef = int(troopDefF - troopAtkF*atkRatio)

	if remainAtk < 0 {
		remainAtk = 0
	}

	if remainDef < 0 {
		remainDef = 0
	}

	return
}
