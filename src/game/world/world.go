package world

import (
    "log"
    "util"
    "time"
    "errors"
    "io/ioutil"
    "encoding/json"
)

// World size in chunks
var WorldSize util.Size = util.Size { 50, 50 }

// Chunk size in blocks
var ChunkSize util.Size = util.Size { 16, 16 }

var StructMap map[int]Structure

type Block struct {
    Pos     util.Point
    Terrain TerrainType
    Empty   bool
}

type Chunk struct {
    Owner       string
    Pos         util.Point
    Size        util.Size
    Blocks      [][]Block
    Structures  []Structure
    UpdateTime  int64       // Unix time
}

func init() {
    StructMap = make(map[int]Structure)
    err := loadStructures("/home/hmkrl/Documents/NCKU_Lectures/GO/LifeGamer-RTS/src/game/world/structures.json")
    if err != nil {
        log.Println(err)
    }
}

func NewChunk(pos util.Point) *Chunk {
    blocks := make([][]Block, ChunkSize.W)

    for x := range blocks {
        blocks[x] = make([]Block, ChunkSize.H)

        for y := range blocks[x] {
            blocks[x][y] = Block { Pos: util.Point { x, y }, Empty: true }
        }
    }

    return &Chunk { "", pos, ChunkSize, blocks, []Structure {}, time.Now().Unix() }
}

func loadStructures(filename string) (err error) {
    type strProto struct {
        ID int
        Name string
        Terrain []int
        Cost int
        Power int
        Human int
        Money int
        Size uint
    }

    protoList := struct {
        Structures []strProto
    } {}

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
        structure.Human = s.Human
        structure.Money = s.Money

        for _, t := range s.Terrain {
            structure.Terrain |= t
        }

        // TODO: Support two-dimention size
        structure.Size = util.Size { W: s.Size, H: s.Size }

        StructMap[structure.ID] = structure
    }

    return
}

func (chunk *Chunk) Update() {
    current := time.Now().Unix()
    chunk.UpdateTime = current
    return
}

func (chunk Chunk) Accepts(str Structure) (ok bool, err error) {
    var i uint
    var j uint

    // Check terrain accepts structure
    var terr_ok bool = false

    for i = 0; i < str.Size.H; i++ {
        for j = 0; j < str.Size.W; j++ {
            // Check available space
            if !chunk.Blocks[str.Pos.X][str.Pos.Y].Empty {
                err = errors.New("Map occupied")
                return
            }

            // Check terrain
            if !terr_ok && chunk.Blocks[str.Pos.X][str.Pos.Y].Terrain.Accepts(str.Terrain) {
                terr_ok = true
            }
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

func BuildStructure(wdb *WorldDB, str Structure, owner string) (err error) {
    target_chunk, err := wdb.Get(str.Chunk.String())

    // WorldDB get error
    if err != nil {
        return
    }

    // Permission denied
    if target_chunk.Owner != owner {
        err = errors.New("User do not own the chunk")
        return
    }

    // Check available space & terrain
    if ok, err := target_chunk.Accepts(str); ok {
        //TODO: Build structure
    } else {
        return err
    }

    return
}
