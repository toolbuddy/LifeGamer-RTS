package world

import (
    "util"
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
    blocks := make([][]Block, ChunkSize.W)

    for x := range blocks {
        blocks[x] = make([]Block, ChunkSize.H)

        for y := range blocks[x] {
            blocks[x][y] = Block { Pos: util.Point { x, y } }
        }
    }

    return &Chunk { "", pos, ChunkSize, blocks, []Structure {} }
}
