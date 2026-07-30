package admin

import (
	"github.com/woodleighschool/woodgate/internal/store/db"
	"github.com/woodleighschool/woodgate/internal/store/postgres"
)

type Store struct {
	store   *postgres.Store
	queries *db.Queries
}

func New(store *postgres.Store) *Store {
	return &Store{
		store:   store,
		queries: store.Queries(),
	}
}
