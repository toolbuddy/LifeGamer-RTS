package world

import (
	"encoding/json"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
)

type WorldDB struct {
	*leveldb.DB
	mapLock map[string]*sync.Mutex

	Updated chan string // indicate which data have been changed
}

func NewWorldDB(path string) (wdb *WorldDB, err error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return
	}

	mapLock := make(map[string]*sync.Mutex)

	wdb = &WorldDB{db, mapLock, make(chan string, 256)}
	return
}

func (wdb WorldDB) Close() error {
	close(wdb.Updated)
	return wdb.DB.Close()
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

func (wdb WorldDB) Put(key string, value Chunk) (err error) {
	err = wdb.Load(key, value)
	if err != nil {
		return
	}

	wdb.Updated <- key
	return
}

func (wdb WorldDB) Load(key string, value Chunk) (err error) {
	if _, ok := wdb.mapLock[key]; !ok {
		wdb.mapLock[key] = new(sync.Mutex)
	}

	b, err := json.Marshal(value)
	if err != nil {
		return
	}

	err = wdb.DB.Put([]byte(key), b, nil)
	if err != nil {
		return
	}

	return
}

func (wdb WorldDB) Lock(key string) {
	_, ok := wdb.mapLock[key]
	if !ok {
		wdb.mapLock[key] = new(sync.Mutex)
	}

	wdb.mapLock[key].Lock()
}

func (wdb WorldDB) Unlock(key string) {
	_, ok := wdb.mapLock[key]
	if !ok {
		wdb.mapLock[key] = new(sync.Mutex)
	} else {
		wdb.mapLock[key].Unlock()
	}
}
