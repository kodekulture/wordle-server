package temp

import (
	"encoding/json"
	"github.com/Chat-Map/wordle-server/game"
	"github.com/dgraph-io/badger"
	"github.com/google/uuid"
)

type HubRepo struct {
	db *badger.DB
}

// Dump implements repository.CacheDB.
func (r *HubRepo) Dump(hub map[uuid.UUID]*game.Game) error {
	for id, g := range hub {
		err := r.db.Update(func(txn *badger.Txn) error {
			b, err := json.Marshal(g)
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
func (r *HubRepo) Load() (map[uuid.UUID]*game.Game, func() error, error) {
	hub := make(map[uuid.UUID]*game.Game)
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
			hub[uid] = &g
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return hub, r.wipe, err
}

func (r *HubRepo) wipe() error {
	return r.db.DropAll()
}

func NewHubRepo(db *badger.DB) *HubRepo {
	return &HubRepo{db: db}
}
