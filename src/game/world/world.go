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
    blocks := make([][]Block, ChunkSize.H)

    for r := range blocks {
        blocks[r] = make([]Block, ChunkSize.W)

        for c := range blocks[r] {
            blocks[r][c] = Block { Pos: util.Point { c, r } }
        }
    }

    return &Chunk { "", pos, ChunkSize, blocks, []Structure {} }
}
