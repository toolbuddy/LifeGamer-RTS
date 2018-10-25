package player

import (
    "util"
    "github.com/syndtr/goleveldb/leveldb"
    "encoding/gob"
    "bytes"
)

type Player struct {
    Id string       // student id
    Human int
    Money int
    Power int
    Home util.Point // spawn point
}

type PlayerDB struct {
    *leveldb.DB
}

func (pdb *PlayerDB) Close() error {
    return pdb.DB.Close()
}

func (pdb *PlayerDB) Delete(key string) error {
    return pdb.DB.Delete([]byte(key), nil)
}

func (pdb *PlayerDB) Get(key string) (value Player, err error) {
    v, err := pdb.DB.Get([]byte(key), nil)

    if err != nil {
        return
    }

    var buffer bytes.Buffer
    buffer.Write(v)

    dec := gob.NewDecoder(&buffer)
    dec.Decode(&value)

    return
}

func (pdb *PlayerDB) GetHome(key string) (home util.Point, err error) {
    p, err := pdb.Get(key)

    if err != nil {
        return
    }

    home = p.Home

    return
}

func (pdb *PlayerDB) GetHuman(key string) (human int, err error) {
    p, err := pdb.Get(key)

    if err != nil {
        return
    }

    human = p.Human

    return
}

func (pdb *PlayerDB) GetMoney(key string) (money int, err error) {
    p, err := pdb.Get(key)

    if err != nil {
        return
    }

    money = p.Money

    return
}

func (pdb *PlayerDB) GetPower(key string) (power int, err error) {
    p, err := pdb.Get(key)

    if err != nil {
        return
    }

    power = p.Power

    return
}

func (pdb *PlayerDB) Put(key string, value Player) error {
    var buffer bytes.Buffer

    enc := gob.NewEncoder(&buffer)
    enc.Encode(value)

    return pdb.DB.Put([]byte(key), buffer.Bytes(), nil)
}

func NewPlayerDB(path string) (pdb *PlayerDB, err error) {
    db, err := leveldb.OpenFile(path, nil)

    if err != nil {
        return
    }

    pdb = &PlayerDB { db }

    return
}
