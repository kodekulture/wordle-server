// Package badgr is an adapter for the badgerDB
package badgr

import (
	"encoding/json"

	"github.com/dgraph-io/badger"
	"github.com/google/uuid"
	"github.com/kodekulture/wordle-server/game"
)

type HubRepo struct {
	db *badger.DB
}

// Dump implements repository.CacheDB.
func (r *HubRepo) Dump(hub map[uuid.UUID]*game.Room) error {
	for id, room := range hub {
		err := r.db.Update(func(txn *badger.Txn) error {
			b, err := json.Marshal(room.Game())
			if err != nil {
				return err
			}
			e := badger.NewEntry([]byte(id.String()), b)
			err = txn.SetEntry(e)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Load implements repository.CacheDB.
func (r *HubRepo) Load(conv func(g *game.Game) *game.Room) (map[uuid.UUID]*game.Room, error) {
	hub := make(map[uuid.UUID]*game.Room)
	err := r.db.View(func(txn *badger.Txn) error {
		// set badger options
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		// iterate over all items
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			// get key
			key := item.Key()
			uid, err := uuid.Parse(string(key))
			if err != nil {
				return err
			}
			// get value
			var g game.Game
			err = item.Value(func(v []byte) error {
				err := json.Unmarshal(v, &g)
				return err
			})
			if err != nil {
				return err
			}
			// set value
			hub[uid] = conv(&g)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hub, err
}

func (r *HubRepo) Drop() error {
	return r.db.DropAll()
}

func New(db *badger.DB) *HubRepo {
	return &HubRepo{db: db}
}
