package player

import (
	"encoding/json"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
)

type PlayerDB struct {
	*leveldb.DB
	playerLock map[string]*sync.Mutex

	Updated chan string // indicate which data have been changed
}

func NewPlayerDB(path string) (pdb *PlayerDB, err error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return
	}

	playerLock := make(map[string]*sync.Mutex)

	pdb = &PlayerDB{db, playerLock, make(chan string, 256)}
	return
}

func (pdb PlayerDB) Close() error {
	close(pdb.Updated)
	return pdb.DB.Close()
}

func (pdb PlayerDB) Delete(key string) error {
	return pdb.DB.Delete([]byte(key), nil)
}

func (pdb PlayerDB) Get(key string) (value Player, err error) {
	v, err := pdb.DB.Get([]byte(key), nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(v, &value)
	return
}

func (pdb PlayerDB) Put(key string, value Player) (err error) {
	b, err := json.Marshal(value)
	if err != nil {
		return
	}

	err = pdb.DB.Put([]byte(key), b, nil)
	if err != nil {
		return
	}

	pdb.Updated <- key
	return
}

func (pdb PlayerDB) Lock(key string) {
	_, ok := pdb.playerLock[key]
	if !ok {
		pdb.playerLock[key] = new(sync.Mutex)
	}

	pdb.playerLock[key].Lock()
}

func (pdb PlayerDB) Unlock(key string) {
	_, ok := pdb.playerLock[key]
	if !ok {
		pdb.playerLock[key] = new(sync.Mutex)
	} else {
		pdb.playerLock[key].Unlock()
	}
}
