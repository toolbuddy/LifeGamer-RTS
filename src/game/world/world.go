package world

import (
    "util"
    "github.com/syndtr/goleveldb/leveldb"
    "encoding/json"
)

var ChunkSize util.Size = util.Size { 16, 16 }

type Block struct {
    Pos     util.Point
    Terrain TerrainType
}

type Chunk struct {
    Owner       string
    Pos         util.Point
    Size        util.Size
    Blocks      [][]Block
    Structures  []Structure
}

func NewChunk(pos util.Point) *Chunk {
    blocks := make([][]Block, ChunkSize.H)

    for r := range blocks {
        blocks[r] = make([]Block, ChunkSize.W)

        for c := range blocks[r] {
            blocks[r][c] = Block { Pos: util.Point { c, r } }
        }
    }

    return &Chunk { "", pos, ChunkSize, blocks, []Structure {} }
}

type WorldDB struct {
    *leveldb.DB
}

func NewWorldDB(path string) (wdb *WorldDB, err error) {
    db, err := leveldb.OpenFile(path, nil)
    if err != nil {
        return
    }

    wdb = &WorldDB { db }
    return
}

func (wdb WorldDB) Delete(key string) error {
    return wdb.DB.Delete([]byte(key), nil)
}

func (wdb WorldDB) Get(key string) (value Chunk, err error) {
    v, err := wdb.DB.Get([]byte(key), nil)
    if err != nil {
        return
    }

    err = json.Unmarshal(v, &value)
    return
}

func (wdb WorldDB) Put(key string, value Chunk) error {
    b, err := json.Marshal(value)
    if err != nil {
        return err
    }

    return wdb.DB.Put([]byte(key), b, nil)
}
