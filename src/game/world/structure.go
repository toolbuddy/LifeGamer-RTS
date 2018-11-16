package world

import (
    "util"
    "encoding/json"
)

const (
    Running = "Running"         // Running normally
    Building = "Building"       // building or upgrading
    Destucting = "Destucting"   // Being destruting by player
    Destroyed = "Destroyed"     // Destroyed after war
    Halted = "Halted"           // Halt because insufficient power
)

type Structure struct {
    ID          int
    Name        string
    Status      string

    Human       int             // + for provide, - for occupy
    Money       int             // + for produce, - for consume
    Power       int             // + for generate, - for consume

    Cost        int             // Money required for build
    // Upgrade cost: 1->2 = 1 * Cost
    //               2->3 = 2 * Cost
    //               3->4 = 4 * Cost
    //               4->5 = 8 * Cost

    Level       int             // Building's current level
    MaxLevel    int

    Chunk       util.Point
    Pos         util.Point
    Size        util.Size

    Terrain     int            // vaild construct terrain
}

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

type strProtoList struct {
    Structures []strProto
}

func LoadDefinition(structDef []byte) (structList []Structure, err error) {
    var protoList strProtoList
    if err = json.Unmarshal(structDef, &protoList); err != nil {
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

        structure.Size = util.Size { W: s.Size, H: s.Size }

        structList = append(structList, structure)
    }

    return
}
