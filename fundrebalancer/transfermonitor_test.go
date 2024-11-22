package fundrebalancer

import (
	"strconv"
	"testing"

	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	mock_database "github.com/skip-mev/go-fast-solver/mocks/fundrebalancer"
	mock_skipgo "github.com/skip-mev/go-fast-solver/mocks/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestTransferMonitor_UpdateTransfers(t *testing.T) {
	t.Run("pending transaction status is properly updated", func(t *testing.T) {
		ctx := context.Background()
		mockSkipGo := mock_skipgo.NewMockSkipGoClient(t)
		mockDatabse := mock_database.NewMockDatabase(t)

		// two osmosis pending tx's, one will fail and another will complete successfully
		mockDatabse.EXPECT().GetAllPendingRebalanceTransfers(mockContext).Return([]db.GetAllPendingRebalanceTransfersRow{
			{ID: 1, TxHash: "hash", SourceChainID: arbitrumChainID, DestinationChainID: osmosisChainID, Amount: strconv.Itoa(osmosisTargetAmount)},
			{ID: 2, TxHash: "hash2", SourceChainID: arbitrumChainID, DestinationChainID: osmosisChainID, Amount: strconv.Itoa(osmosisTargetAmount)},
		}, nil)

		mockSkipGo.EXPECT().TrackTx(mockContext, "hash", arbitrumChainID).Return("hash", nil)

		mockSkipGo.EXPECT().TrackTx(mockContext, "hash2", arbitrumChainID).Return("hash2", nil)

		mockSkipGo.EXPECT().Status(mockContext, skipgo.TxHash("hash"), arbitrumChainID).Return(&skipgo.StatusResponse{
			Transfers: []skipgo.Transfer{
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
			},
		}, nil)

		transferError := "ahhhh"
		mockSkipGo.EXPECT().Status(mockContext, skipgo.TxHash("hash2"), arbitrumChainID).Return(&skipgo.StatusResponse{
			Transfers: []skipgo.Transfer{
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_ERROR, Error: &transferError},
			},
		}, nil)

		mockDatabse.EXPECT().UpdateTransferStatus(mockContext, db.UpdateTransferStatusParams{
			ID:     1,
			Status: dbtypes.RebalanceTransactionStatusSuccess,
		}).Return(nil)
		mockDatabse.EXPECT().UpdateTransferStatus(mockContext, db.UpdateTransferStatusParams{
			ID:     2,
			Status: dbtypes.RebalanceTransactionStatusFailed,
		}).Return(nil)

		tm := NewTransferTracker(mockSkipGo, mockDatabse)

		assert.NoError(t, tm.UpdateTransfers(ctx))
	})
}
