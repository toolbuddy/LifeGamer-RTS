package world

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"time"
	"util"
)

// World size in chunks
var WorldSize util.Size = util.Size{50, 50}

// Chunk size in blocks
var ChunkSize util.Size = util.Size{16, 16}

var StructMap map[int]Structure

type Block struct {
	Pos     util.Point
	Terrain TerrainType
	Empty   bool
}

type Chunk struct {
	Owner          string
	Pos            util.Point
	Size           util.Size
	Blocks         [][]Block
	Structures     []Structure
	Population     int64 // Population on this chunk
	PopulationRate int64 // Population Rate of this chunk
	UpdateTime     int64 // Unix time
}

// Get db key of chunk
func (chunk Chunk) Key() string {
	return chunk.Pos.String()
}

func init() {
	StructMap = make(map[int]Structure)
	err := loadStructures("src/game/world/structures.json")
	if err != nil {
		log.Fatalln("[ERROR] Unable to load structure data")
	}
}

func NewChunk(pos util.Point) *Chunk {
	blocks := make([][]Block, ChunkSize.W)

	for x := range blocks {
		blocks[x] = make([]Block, ChunkSize.H)

		for y := range blocks[x] {
			blocks[x][y] = Block{Pos: util.Point{x, y}, Empty: true}
		}
	}

	return &Chunk{"", pos, ChunkSize, blocks, []Structure{}, 0, 0, time.Now().Unix()}
}

func loadStructures(filename string) (err error) {
	type strProto struct {
		ID            int
		Name          string
		Terrain       []int
		Cost          int
		Power         int
		Population    int
		Money         int
		PopulationCap int
		Size          uint
		MaxLevel      int
		BuildTime     int64
	}

	protoList := struct {
		Structures []strProto
	}{}

	structjson, _ := ioutil.ReadFile(filename)

	if err = json.Unmarshal(structjson, &protoList); err != nil {
		return
	}

	for _, s := range protoList.Structures {
		var structure Structure

		structure.ID = s.ID
		structure.Name = s.Name
		structure.Cost = s.Cost
		structure.Power = s.Power
		structure.Population = s.Population
		structure.Money = s.Money
		structure.PopulationCap = s.PopulationCap
		structure.Level = 1
		structure.MaxLevel = s.MaxLevel
		structure.BuildTime = s.BuildTime

		for _, t := range s.Terrain {
			structure.Terrain |= t
		}

		// TODO: Support two-dimention size
		structure.Size = util.Size{W: s.Size, H: s.Size}

		StructMap[structure.ID] = structure
	}

	return
}

func (chunk Chunk) Accepts(str Structure) (ok bool, err error) {
	var terr_ok bool

	for _, point := range util.InSizeRange(str.Pos, str.Size) {
		if uint(point.X) >= ChunkSize.W || uint(point.Y) >= ChunkSize.H {
			err = errors.New("Structure out of chunk")
			return
		}
		// Check available space
		if !chunk.Blocks[point.X][point.Y].Empty {
			err = errors.New("Map occupied")
			return
		}

		// Check terrain
		if !terr_ok && chunk.Blocks[point.X][point.Y].Terrain.Accepts(str.Terrain) {
			terr_ok = true
		}
	}

	if !terr_ok {
		err = errors.New("Terrain check failed")
		return
	} else {
		ok = true
		return
	}
}

func (chunk Chunk) PopulationNeed() (needed int64) {
	for _, str := range chunk.Structures {
		if str.Population < 0 {
			needed += int64(-str.Population)
		}
	}

	return
}

// Fill remained part of struct from client
func CompleteStructure(str *Structure) {
	chunk := str.Chunk
	pos := str.Pos

	// Load default values
	*str = StructMap[str.ID]

	// Restore position
	str.Chunk = chunk
	str.Pos = pos
}

func GetStructure(chunk Chunk, str Structure) (str_index int, err error) {
	// Check structure exists, returns -1 if not found
	str_index = func() int {
		for index, s := range chunk.Structures {
			if s.Pos == str.Pos && s.ID == str.ID {
				return index
			}
		}
		return -1
	}()

	if str_index == -1 {
		err = errors.New("Structure not found")
	}

	return
}

func BuildStructure(chunk *Chunk, str Structure) (err error) {
	// Check available space & terrain
	if ok, err := chunk.Accepts(str); !ok {
		return err
	}

	// Check finished, build the structure

	// Set map occupied
	for _, block := range util.InSizeRange(str.Pos, str.Size) {
		chunk.Blocks[block.X][block.Y].Empty = false
	}

	str.UpdateTime = time.Now().Unix()

	// Add structure
	chunk.Structures = append(chunk.Structures, str)

	return
}

func DestructStructure(chunk *Chunk, str Structure) (err error) {
	index, err := GetStructure(*chunk, str)

	if err != nil {
		return
	}

	// Set map free
	for _, block := range util.InSizeRange(str.Pos, str.Size) {
		chunk.Blocks[block.X][block.Y].Empty = true
	}

	// delete structure
	chunk.Structures = append(chunk.Structures[:index], chunk.Structures[index+1:]...)

	return
}
