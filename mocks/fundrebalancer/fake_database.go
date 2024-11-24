package fundrebalancer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
)

type FakeTransfer struct {
	ID                 int64
	TxHash             string
	SourceChainID      string
	DestinationChainID string
	Amount             string
	Status             string
	CreatedAt          time.Time
}

type FakeDatabase struct {
	db     []*FakeTransfer
	dbLock *sync.RWMutex
}

func NewFakeDatabase() *FakeDatabase {
	return &FakeDatabase{
		db:     make([]*FakeTransfer, 0),
		dbLock: new(sync.RWMutex),
	}
}

func (fdb *FakeDatabase) GetPendingRebalanceTransfersToChain(ctx context.Context, destinationChainID string) ([]db.GetPendingRebalanceTransfersToChainRow, error) {
	fdb.dbLock.RLock()
	defer fdb.dbLock.RUnlock()

	var pendingTransfers []db.GetPendingRebalanceTransfersToChainRow
	for _, transfer := range fdb.db {
		if transfer.Status == "PENDING" && transfer.DestinationChainID == destinationChainID {
			pendingTransfers = append(pendingTransfers, db.GetPendingRebalanceTransfersToChainRow{
				ID:                 transfer.ID,
				TxHash:             transfer.TxHash,
				SourceChainID:      transfer.SourceChainID,
				DestinationChainID: transfer.DestinationChainID,
				Amount:             transfer.Amount,
			})
		}
	}
	return pendingTransfers, nil
}

func (fdb *FakeDatabase) InsertRebalanceTransfer(ctx context.Context, arg db.InsertRebalanceTransferParams) (int64, error) {
	fdb.dbLock.Lock()
	defer fdb.dbLock.Unlock()

	var nextID int64 = 0
	if len(fdb.db) > 0 {
		nextID = fdb.db[len(fdb.db)-1].ID + 1
	}

	fdb.db = append(fdb.db, &FakeTransfer{
		ID:                 nextID,
		TxHash:             arg.TxHash,
		SourceChainID:      arg.SourceChainID,
		DestinationChainID: arg.DestinationChainID,
		Amount:             arg.Amount,
		Status:             "PENDING",
		CreatedAt:          time.Now(),
	})

	return nextID, nil
}

func (fdb *FakeDatabase) GetAllPendingRebalanceTransfers(ctx context.Context) ([]db.GetAllPendingRebalanceTransfersRow, error) {
	fdb.dbLock.RLock()
	defer fdb.dbLock.RUnlock()

	var pendingTransfers []db.GetAllPendingRebalanceTransfersRow
	for _, transfer := range fdb.db {
		if transfer.Status == "PENDING" {
			pendingTransfers = append(pendingTransfers, db.GetAllPendingRebalanceTransfersRow{
				ID:                 transfer.ID,
				TxHash:             transfer.TxHash,
				SourceChainID:      transfer.SourceChainID,
				DestinationChainID: transfer.DestinationChainID,
				Amount:             transfer.Amount,
				CreatedAt:          transfer.CreatedAt,
			})
		}
	}
	return pendingTransfers, nil
}

func (fdb *FakeDatabase) UpdateTransferStatus(ctx context.Context, arg db.UpdateTransferStatusParams) error {
	fdb.dbLock.Lock()
	defer fdb.dbLock.Unlock()

	for _, transfer := range fdb.db {
		if transfer.ID == arg.ID {
			transfer.Status = arg.Status
		}
	}
	return nil
}

func (fdb *FakeDatabase) GetDBContents() []*FakeTransfer {
	return fdb.db
}

func (fdb *FakeDatabase) UpdateTransferCreatedAt(ctx context.Context, id int64, createdAt time.Time) error {
	fdb.dbLock.Lock()
	defer fdb.dbLock.Unlock()

	for _, transfer := range fdb.db {
		if transfer.ID == id {
			transfer.CreatedAt = createdAt
			return nil
		}
	}
	return fmt.Errorf("transfer with id %d not found", id)
}
