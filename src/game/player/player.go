package player

import (
    "util"
    "github.com/syndtr/goleveldb/leveldb"
    "encoding/gob"
    "bytes"
)

type Player struct {
    Human int
    Money int
    Power int
    Home util.Point // spawn point
    Initialized bool
    UpdateTime int64
}

type PlayerDB struct {
    *leveldb.DB
}

func NewPlayerDB(path string) (pdb *PlayerDB, err error) {
    db, err := leveldb.OpenFile(path, nil)
    if err != nil {
        return
    }

    pdb = &PlayerDB { db }

    return
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

func (pdb *PlayerDB) Put(key string, value Player) error {
    var buffer bytes.Buffer

    enc := gob.NewEncoder(&buffer)
    enc.Encode(value)

    return pdb.DB.Put([]byte(key), buffer.Bytes(), nil)
}
