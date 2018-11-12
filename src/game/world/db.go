package world

import (
    "github.com/syndtr/goleveldb/leveldb"
    "encoding/json"
)

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
