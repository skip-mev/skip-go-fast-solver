package fundrebalancer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	evmtxsubmission "github.com/skip-mev/go-fast-solver/shared/txexecutor/evm"

	"github.com/ethereum/go-ethereum/common"
	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/shared/metrics"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/contracts/usdc"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/signing"
	"github.com/skip-mev/go-fast-solver/shared/signing/evm"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

const (
	initialRebalancerLoopDelay = 1 * time.Nanosecond
	rebalancerLoopDelay        = 1 * time.Minute
)

type Database interface {
	GetPendingRebalanceTransfersToChain(ctx context.Context, destinationChainID string) ([]db.GetPendingRebalanceTransfersToChainRow, error)
	InsertRebalanceTransfer(ctx context.Context, arg db.InsertRebalanceTransferParams) (int64, error)
	GetAllPendingRebalanceTransfers(ctx context.Context) ([]db.GetAllPendingRebalanceTransfersRow, error)
	UpdateTransferStatus(ctx context.Context, arg db.UpdateTransferStatusParams) error
}

type FundRebalancer struct {
	chainIDToPrivateKey map[string]string
	skipgo              skipgo.SkipGoClient
	evmClientManager    evmrpc.EVMRPCClientManager
	config              map[string]config.FundRebalancerConfig
	database            Database
	trasferTracker      *TransferTracker
	evmTxExecutor       evmtxsubmission.EVMTxExecutor
}

func NewFundRebalancer(
	ctx context.Context,
	keysPath string,
	skipgo skipgo.SkipGoClient,
	evmClientManager evmrpc.EVMRPCClientManager,
	database Database,
	evmTxExecutor evmtxsubmission.EVMTxExecutor,
) (*FundRebalancer, error) {
	chainIDToPriavateKey, err := loadChainIDToPrivateKeyMap(keysPath)
	if err != nil {
		return nil, fmt.Errorf("loading chain id to private key map from %s: %w", keysPath, err)
	}

	return &FundRebalancer{
		chainIDToPrivateKey: chainIDToPriavateKey,
		skipgo:              skipgo,
		evmClientManager:    evmClientManager,
		config:              config.GetConfigReader(ctx).Config().FundRebalancer,
		database:            database,
		trasferTracker:      NewTransferTracker(skipgo, database),
		evmTxExecutor:       evmTxExecutor,
	}, nil
}

// Run is the main loop of the fund rebalancer.
func (r *FundRebalancer) Run(ctx context.Context) {
	if r.config == nil {
		lmt.Logger(ctx).Warn("no fund rebalancer config found, no funds will be rebalanced across chains")
		return
	}

	go r.trasferTracker.TrackPendingTransfers(ctx)

	ticker := time.NewTicker(initialRebalancerLoopDelay)
	lmt.Logger(ctx).Info("fund rebalancer starting to monitor chains for fund imbalances")
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ticker.Stop()

			r.Rebalance(ctx)

			ticker.Reset(rebalancerLoopDelay)
		}
	}
}

// Rebalance performs a single rebalancing of funds across all chains. It will
// iterate over configured chains unit it finds a chain in need of funds, and
// will then take funds from chains that have usdc to spare in their balances.
// If multiple chains are in need of a rebalance, this function will attempt to
// rebalance all of them.
func (r *FundRebalancer) Rebalance(ctx context.Context) {
	for chainID := range r.config {
		usdcNeeded, err := r.USDCNeeded(ctx, chainID)
		if err != nil {
			lmt.Logger(ctx).Error("error getting usdc needed on chain", zap.Error(err), zap.String("chainID", chainID))
			continue
		}
		if usdcNeeded.Cmp(big.NewInt(0)) <= 0 {
			// no usdc needed on this chain, continue checking if
			// others need usdc
			continue
		}

		lmt.Logger(ctx).Info(
			"found chain in need of usdc",
			zap.String("chainID", chainID),
			zap.String("usdcNeeded", usdcNeeded.String()),
		)

		txns, totalUSDCMoved, err := r.MoveFundsToChain(
			ctx,
			chainID,
			usdcNeeded,
		)
		if err != nil {
			lmt.Logger(ctx).Error("error moving funds to chain", zap.Error(err), zap.String("chainID", chainID))
			continue
		}

		if len(txns) > 0 {
			lmt.Logger(ctx).Info(
				"submitted transactions to rebalance usdc to chain",
				zap.String("chainID", chainID),
				zap.String("usdcNeeded", usdcNeeded.String()),
				zap.String("totalUSDCRebalanced", totalUSDCMoved.String()),
				zap.Int("totalTxnsToRebalance", len(txns)),
			)
		}
	}
}

func loadChainIDToPrivateKeyMap(keysPath string) (map[string]string, error) {
	keysBytes, err := os.ReadFile(keysPath)
	if err != nil {
		return nil, err
	}

	rawKeysMap := make(map[string]map[string]string)
	if err := json.Unmarshal(keysBytes, &rawKeysMap); err != nil {
		return nil, err
	}

	keysMap := make(map[string]string)
	for key, value := range rawKeysMap {
		keysMap[key] = value["private_key"]
	}

	return keysMap, nil
}

// USDCNeeded gets the amount of usdc a chain needs in order to reach its
// configured target amount balance
func (r *FundRebalancer) USDCNeeded(
	ctx context.Context,
	chainID string,
) (*big.Int, error) {
	currentBalance, err := r.usdcBalance(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("getting usdc balance on chain %s: %w", chainID, err)
	}

	// There is a race condition special case here leading to potentially
	// seeing a chains balance as greater than it actually is. If the transfer
	// monitor has not seen that a transaction arrived on chain yet (i.e. it is
	// still marked as pending), but in reality it has arrived on chain and the
	// amount is reflected in the on chain balance, that amount is counted
	// twice (once via the on chain balance, and once via the pending balance).
	//
	// This may result in a chains balance looking higher than it is for some
	// time until the transfer monitor sees that the transfer has completed
	// successfully. However, since this is eventually consistent with the
	// correct balance, this will only result in slightly delayed rebalances to
	// a chain.
	pendingBalance, err := r.pendingUSDCBalance(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("getting pending balance on chain %s: %w", chainID, err)
	}
	currentBalance.Add(currentBalance, pendingBalance)

	minAllowedAmount, ok := new(big.Int).SetString(r.config[chainID].MinAllowedAmount, 10)
	if !ok {
		return nil, fmt.Errorf("could not convert min allowed amount %s to *big.Int for chain %s", r.config[chainID].MinAllowedAmount, chainID)
	}

	if currentBalance.Cmp(minAllowedAmount) >= 0 {
		// usdc allocation on this chain is > min allowed amount, no usdc needed
		return big.NewInt(0), nil
	}

	targetAmount, ok := new(big.Int).SetString(r.config[chainID].TargetAmount, 10)
	if !ok {
		return nil, fmt.Errorf("could not convert target amount %s to *big.Int for chain %s", r.config[chainID].TargetAmount, chainID)
	}

	return new(big.Int).Sub(targetAmount, currentBalance), nil
}

// MoveFundsToChain moves usdc from each chain in the fund rebalancers config
// that has usdc to spare (i.e. the chains usdc balance is > configured target
// balance), until usdcToReachTarget usdc has been transfered to
// rebalanceToChain.
func (r *FundRebalancer) MoveFundsToChain(
	ctx context.Context,
	rebalanceToChain string,
	usdcToReachTarget *big.Int,
) ([]skipgo.TxHash, *big.Int, error) {
	var hashes []skipgo.TxHash
	totalUSDCcMoved := big.NewInt(0)
	remainingUSDCNeeded := usdcToReachTarget
	for rebalanceFromChainID := range r.config {
		if rebalanceFromChainID == rebalanceToChain {
			// do not try and rebalance funds from the same chain
			continue
		}

		usdcToSpare, err := r.USDCToSpare(ctx, rebalanceFromChainID)
		if err != nil {
			return nil, nil, fmt.Errorf("could not get amount of usdc to spare from chain %s: %w", rebalanceFromChainID, err)
		}

		if usdcToSpare.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		neededVsSpareDiff := new(big.Int).Sub(remainingUSDCNeeded, usdcToSpare)

		var usdcToRebalance *big.Int
		if neededVsSpareDiff.Cmp(big.NewInt(0)) <= 0 {
			// usdc to spare on this chain is greater than or equal to the
			// remaining amount needed, so only move the remaining amount so we
			// dont wipe out a chains funds uncessarily
			usdcToRebalance = remainingUSDCNeeded
		} else {
			// usdc to spare on this chain is less than the remaining amount
			// needed, so move all that it has to spare and then continue
			// moving from other over allocated chains
			usdcToRebalance = usdcToSpare
		}

		txns, err := r.GetRebalanceTxns(ctx, usdcToRebalance, rebalanceFromChainID, rebalanceToChain)
		if err != nil {
			return nil, nil, fmt.Errorf("getting txns required for fund rebalancing: %w", err)
		}
		if len(txns) != 1 {
			return nil, nil, fmt.Errorf("only single transaction transfers are supported")
		}
		txn := txns[0]

		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(rebalanceFromChainID)
		if err != nil {
			return nil, nil, fmt.Errorf("getting chain config for gas threshold check: %w", err)
		}

		if chainConfig.MaxRebalancingGasThreshold != 0 {
			gasAcceptable, totalRebalancingGas, err := r.isGasAcceptable(txns, chainConfig.MaxRebalancingGasThreshold)
			if err != nil {
				return nil, nil, fmt.Errorf("checking if gas amount is acceptable: %w", err)
			}
			if !gasAcceptable {
				lmt.Logger(ctx).Info(
					"skipping rebalance from chain "+rebalanceFromChainID+" due to rebalancing txs exceeding gas threshold",
					zap.String("sourceChainID", rebalanceFromChainID),
					zap.String("destinationChainID", rebalanceToChain),
					zap.Uint64("estimatedGas", totalRebalancingGas),
					zap.Uint64("gasThreshold", chainConfig.MaxRebalancingGasThreshold),
				)
				continue
			}
		}

		if err = r.ERC20Approval(ctx, txn); err != nil {
			return nil, nil, fmt.Errorf("approving usdc erc20 spend on chain %s for %suusdc: %w", rebalanceFromChainID, usdcToRebalance.String(), err)
		}

		txHash, err := r.SignAndSubmitTxn(ctx, txn)
		if err != nil {
			return nil, nil, fmt.Errorf("signing and submitting transaction: %w", err)
		}
		metrics.FromContext(ctx).IncFundsRebalanceTransferStatusChange(rebalanceFromChainID, rebalanceToChain, dbtypes.RebalanceTransactionStatusPending)

		totalUSDCcMoved = new(big.Int).Add(totalUSDCcMoved, usdcToRebalance)
		hashes = append(hashes, txHash)

		// if there is no more usdc needed, we are done rebalancing
		remainingUSDCNeeded = new(big.Int).Sub(remainingUSDCNeeded, usdcToRebalance)
		if remainingUSDCNeeded.Cmp(big.NewInt(0)) <= 0 {
			return hashes, totalUSDCcMoved, nil
		}
	}

	// we have moved all available funds from all available chains
	return hashes, totalUSDCcMoved, nil
}

// USDCToSpare returns a chains current balance - a chains target amount of
// usdc, or 0 if this value is negative. This does not take into account any
// pending rebalance transactions in the db that are bound for this chain.
func (r *FundRebalancer) USDCToSpare(
	ctx context.Context,
	chainID string,
) (*big.Int, error) {
	currentBalance, err := r.usdcBalance(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("getting usdc balance on chain %s: %w", chainID, err)
	}

	targetAmountBig, ok := new(big.Int).SetString(r.config[chainID].TargetAmount, 10)
	if !ok {
		return nil, fmt.Errorf("converting target amount to *big.Int")
	}

	if currentBalance.Cmp(targetAmountBig) <= 0 {
		return big.NewInt(0), nil
	}

	return new(big.Int).Sub(currentBalance, targetAmountBig), nil
}

// usdcBalance gets the balance on chainID in uusdc.
func (r *FundRebalancer) usdcBalance(ctx context.Context, chainID string) (*big.Int, error) {
	usdcDenom, err := config.GetConfigReader(ctx).GetUSDCDenom(chainID)
	if err != nil {
		return nil, fmt.Errorf("getting usdc denom for chain %s: %w", chainID, err)
	}

	var currentBalance *big.Int
	chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, fmt.Errorf("getting chain type for chain id %s: %w", chainID, err)
	}
	switch chainConfig.Type {
	case config.ChainType_EVM:
		client, err := r.evmClientManager.GetClient(ctx, chainID)
		if err != nil {
			return nil, fmt.Errorf("getting evm client for chain %s: %w", chainID, err)
		}

		currentBalance, err = client.GetUSDCBalance(ctx, usdcDenom, chainConfig.SolverAddress)
		if err != nil {
			return nil, fmt.Errorf("fetching balance for address %s on chain %s for denom %s: %w", chainConfig.SolverAddress, chainID, usdcDenom, err)
		}
	case config.ChainType_COSMOS:
		balance, err := r.skipgo.Balance(ctx, chainID, chainConfig.SolverAddress, usdcDenom)
		if err != nil {
			return nil, fmt.Errorf("fetching balance for address %s on chain %s for denom %s: %w", chainConfig.SolverAddress, chainID, usdcDenom, err)
		}

		var ok bool
		currentBalance, ok = new(big.Int).SetString(balance, 10)
		if !ok {
			return nil, fmt.Errorf("could not convert balance %s to *big.Int", balance)
		}
	}

	return currentBalance, nil
}

// pendingUSDCBalance gets the amount of USDC in pending transactions that are
// being sent to chainID as the destination chain.
func (r *FundRebalancer) pendingUSDCBalance(ctx context.Context, chainID string) (*big.Int, error) {
	pendingTransfers, err := r.database.GetPendingRebalanceTransfersToChain(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("getting pending rebalance transfers to chain from db: %w", err)
	}

	balance := big.NewInt(0)
	for _, inboundTransfer := range pendingTransfers {
		inboundAmount, ok := new(big.Int).SetString(inboundTransfer.Amount, 10)
		if !ok {
			return nil, fmt.Errorf("could not convert pending transfer amount %s from db to *big.Int", inboundTransfer.Amount)
		}
		balance = balance.Add(balance, inboundAmount)
	}

	return balance, nil
}

type SkipGoTxnWithMetadata struct {
	tx                 skipgo.Tx
	sourceChainID      string
	destinationChainID string
	amount             *big.Int
	gasEstimate        uint64
}

// GetRebalanceTxns gets transaction msgs/data from Skip Go that can be signed
// and submitted on chain in order to rebalance the solvers funds.
func (r *FundRebalancer) GetRebalanceTxns(
	ctx context.Context,
	amount *big.Int,
	sourceChainID string,
	destChainID string,
) ([]SkipGoTxnWithMetadata, error) {
	rebalanceFromDenom, err := config.GetConfigReader(ctx).GetUSDCDenom(sourceChainID)
	if err != nil {
		return nil, fmt.Errorf("getting usdc denom for chain %s: %w", sourceChainID, err)
	}
	rebalanceToDenom, err := config.GetConfigReader(ctx).GetUSDCDenom(destChainID)
	if err != nil {
		return nil, fmt.Errorf("getting usdc denom for chain %s: %w", destChainID, err)
	}

	sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(sourceChainID)
	if err != nil {
		return nil, fmt.Errorf("getting source chain config for chain %s: %w", sourceChainID, err)
	}
	rebalanceFromAddress := sourceChainConfig.SolverAddress

	destinationChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(destChainID)
	if err != nil {
		return nil, fmt.Errorf("getting destination chain config for chain %s: %w", destChainID, err)
	}
	rebalanceToAddress := destinationChainConfig.SolverAddress

	// get the route that will be used to rebalance funds
	route, err := r.skipgo.Route(
		ctx,
		rebalanceFromDenom,
		sourceChainID,
		rebalanceToDenom,
		destChainID,
		amount,
	)
	if err != nil {
		return nil, fmt.Errorf("getting rebalancing route from Skip Go: %w", err)
	}

	// create addres list from required chain addreses response field
	var addresses []string
	for _, requiredChainAddress := range route.RequiredChainAddresses {
		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(requiredChainAddress)
		if err != nil {
			return nil, fmt.Errorf("getting chain config for chain %s: %w", requiredChainAddress, err)
		}

		addresses = append(addresses, chainConfig.SolverAddress)
	}

	amountOut, ok := new(big.Int).SetString(route.AmountOut, 10)
	if !ok {
		return nil, fmt.Errorf("converting amount out %s to *bit.Int", route.AmountOut)
	}

	// get txn data from for the route to be executed
	txns, err := r.skipgo.Msgs(
		ctx,
		rebalanceFromDenom,
		sourceChainID,
		rebalanceFromAddress,
		rebalanceToDenom,
		destChainID,
		rebalanceToAddress,
		amount,
		amountOut,
		addresses,
		route.Operations,
	)
	if err != nil {
		return nil, fmt.Errorf("getting rebalancing txn operations from Skip Go: %w", err)
	}

	txnsWithMetadata := make([]SkipGoTxnWithMetadata, 0, len(txns))
	for _, txn := range txns {
		var gasEstimate uint64
		if txn.EVMTx != nil {
			client, err := r.evmClientManager.GetClient(ctx, txn.EVMTx.ChainID)
			if err != nil {
				return nil, fmt.Errorf("getting evm client for chain %s: %w", txn.EVMTx.ChainID, err)
			}

			decodedData, err := hex.DecodeString(txn.EVMTx.Data)
			if err != nil {
				return nil, fmt.Errorf("hex decoding evm call data: %w", err)
			}

			txBuilder := evm.NewTxBuilder(client)
			estimate, err := txBuilder.EstimateGasForTx(
				ctx,
				sourceChainConfig.SolverAddress,
				txn.EVMTx.To,
				txn.EVMTx.Value,
				decodedData,
			)
			if err != nil {
				return nil, fmt.Errorf("estimating gas: %w", err)
			}
			gasEstimate = estimate
		}
		txnsWithMetadata = append(txnsWithMetadata, SkipGoTxnWithMetadata{
			tx:                 txn,
			sourceChainID:      sourceChainID,
			destinationChainID: destChainID,
			amount:             amount,
			gasEstimate:        gasEstimate,
		})
	}

	return txnsWithMetadata, nil
}

// SignAndSubmitTxn signs and submits txs to chain
func (r *FundRebalancer) SignAndSubmitTxn(
	ctx context.Context,
	txn SkipGoTxnWithMetadata,
) (skipgo.TxHash, error) {
	// convert the Skip Go txHash into a signable data structure for
	// each chain type
	switch {
	case txn.tx.EVMTx != nil:
		signer, err := signing.NewSigner(ctx, txn.sourceChainID, r.chainIDToPrivateKey)
		if err != nil {
			return "", fmt.Errorf("creating signer for chain %s: %w", txn.sourceChainID, err)
		}

		txData, err := hex.DecodeString(txn.tx.EVMTx.Data)
		if err != nil {
			return "", fmt.Errorf("decoding hex data from Skip Go: %w", err)
		}

		txHash, err := r.evmTxExecutor.ExecuteTx(
			ctx,
			txn.sourceChainID,
			txn.tx.EVMTx.SignerAddress,
			txData,
			txn.tx.EVMTx.Value,
			txn.tx.EVMTx.To,
			signer,
		)
		if err != nil {
			return "", fmt.Errorf("submitting evm txn to chain %s: %w", txn.sourceChainID, err)
		}

		lmt.Logger(ctx).Info(
			"submitted txHash to Skip Go to rebalance funds",
			zap.String("sourceChainID", txn.sourceChainID),
			zap.String("destChainID", txn.destinationChainID),
			zap.String("txnHash", txHash),
		)

		args := db.InsertRebalanceTransferParams{
			TxHash:             txHash,
			SourceChainID:      txn.sourceChainID,
			DestinationChainID: txn.destinationChainID,
			Amount:             txn.amount.String(),
		}
		if _, err := r.database.InsertRebalanceTransfer(ctx, args); err != nil {
			return "", fmt.Errorf("inserting rebalance txHash with hash %s into db: %w", txHash, err)
		}

		return skipgo.TxHash(txHash), nil
	case txn.tx.CosmosTx != nil:
		return "", fmt.Errorf("cosmos txns not supported yet")
	default:
		return "", fmt.Errorf("no valid txHash types returned from Skip Go")
	}
}

func (r *FundRebalancer) ERC20Approval(ctx context.Context, txn SkipGoTxnWithMetadata) error {
	if txn.tx.EVMTx == nil {
		// if this isnt an evm tx, no erc20 approvals are required
		return nil
	}
	evmTx := txn.tx.EVMTx
	if len(evmTx.RequiredERC20Approvals) == 0 {
		// if no approvals are required, return with no error
		return nil
	}
	if len(evmTx.RequiredERC20Approvals) > 1 {
		// only support single approval
		return fmt.Errorf("expected 1 required erc20 approval but got %d", len(evmTx.RequiredERC20Approvals))
	}
	approval := evmTx.RequiredERC20Approvals[0]

	chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(evmTx.ChainID)
	if err != nil {
		return fmt.Errorf("getting config for chain %s: %w", evmTx.ChainID, err)
	}
	usdcDenom, err := config.GetConfigReader(ctx).GetUSDCDenom(evmTx.ChainID)
	if err != nil {
		return fmt.Errorf("fetching usdc denom on chain %s: %w", evmTx.ChainID, err)
	}

	// sanity check on the address being returned to be what the solver expects
	if !strings.EqualFold(approval.TokenContract, usdcDenom) {
		return fmt.Errorf("expected required approval for usdc token contract %s, but got %s", usdcDenom, approval.TokenContract)
	}

	signer, err := signing.NewSigner(ctx, evmTx.ChainID, r.chainIDToPrivateKey)
	if err != nil {
		return fmt.Errorf("creating signer for chain %s: %w", evmTx.ChainID, err)
	}

	spender := common.HexToAddress(approval.Spender)

	amount, ok := new(big.Int).SetString(approval.Amount, 10)
	if !ok {
		return fmt.Errorf("error converting erc20 approval amount %s on chain %s to *big.Int", approval.Amount, evmTx.ChainID)
	}

	abi, err := usdc.UsdcMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("getting usdc contract abi: %w", err)
	}

	input, err := abi.Pack("approve", spender, amount)
	if err != nil {
		return fmt.Errorf("packing input to erc20 approval tx: %w", err)
	}

	_, err = r.evmTxExecutor.ExecuteTx(
		ctx,
		evmTx.ChainID,
		chainConfig.SolverAddress,
		input,
		"0",
		approval.TokenContract,
		signer,
	)
	if err != nil {
		return fmt.Errorf("executing erc20 approve for %s at contract %s for spender %s on %s: %w", amount.String(), approval.TokenContract, approval.Spender, evmTx.ChainID, err)
	}

	return nil
}

func (r *FundRebalancer) estimateTotalGas(txns []SkipGoTxnWithMetadata) (uint64, error) {
	var totalGas uint64
	for _, txn := range txns {
		totalGas += txn.gasEstimate
	}
	return totalGas, nil
}

func (r *FundRebalancer) isGasAcceptable(txns []SkipGoTxnWithMetadata, maxRebalancingGasThreshold uint64) (bool, uint64, error) {
	// Check if total gas needed exceeds threshold to rebalance funds from this chain
	totalRebalancingGas, err := r.estimateTotalGas(txns)
	if err != nil {
		return false, 0, fmt.Errorf("estimating total gas for transactions: %w", err)
	}

	if totalRebalancingGas > maxRebalancingGasThreshold {
		return false, totalRebalancingGas, nil
	}

	return true, totalRebalancingGas, nil
}
