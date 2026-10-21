package storage

import "context"

func (store *Store) Initialize(ctx context.Context) error {
	if err := store.initializePingTable(ctx); err != nil {
		return err
	}
	if err := store.initializeResultTable(ctx); err != nil {
		return err
	}
	return store.initializeResultAggregationTable(ctx)
}