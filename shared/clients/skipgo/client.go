package skipgo

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
)

type TxHash string

type SkipGoClient interface {
	Balance(
		ctx context.Context,
		chainID string,
		address string,
		denom string,
	) (string, error)

	Route(
		ctx context.Context,
		sourceAssetDenom string,
		sourceAssetChainID string,
		destAssetDenom string,
		destAssetChainID string,
		amountIn *big.Int,
	) (*RouteResponse, error)

	Msgs(
		ctx context.Context,
		sourceAssetDenom string,
		sourceAssetChainID string,
		sourceChainAddress string,
		destAssetDenom string,
		destAssetChainID string,
		destChainAddress string,
		amountIn *big.Int,
		amountOut *big.Int,
		addressList []string,
		operations []any,
	) ([]Tx, error)

	SubmitTx(
		ctx context.Context,
		tx []byte,
		chainID string,
	) (TxHash, error)

	TrackTx(
		ctx context.Context,
		txHash string,
		chainID string,
	) (TxHash, error)

	Status(
		ctx context.Context,
		tx TxHash,
		chainID string,
	) (*StatusResponse, error)
}

type skipGoClient struct {
	baseURL *url.URL
	http    *http.Client
}

func NewSkipGoClient(baseURL string) (SkipGoClient, error) {
	url, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing base url %s: %w", baseURL, err)
	}

	return &skipGoClient{
		baseURL: url,
		http:    http.DefaultClient,
	}, nil
}

func (s *skipGoClient) Balance(
	ctx context.Context,
	chainID string,
	address string,
	denom string,
) (string, error) {
	const endpoint = "/v2/info/balances"
	u, err := url.JoinPath(s.baseURL.String(), endpoint)
	if err != nil {
		return "", fmt.Errorf("joining base url to endpoint %s: %w", endpoint, err)
	}

	type BalancesRequestEntry struct {
		Address string   `json:"address"`
		Denoms  []string `json:"denoms"`
	}
	type BalancesRequest struct {
		Chains map[string]*BalancesRequestEntry `json:"chains"`
	}

	body := BalancesRequest{
		Chains: map[string]*BalancesRequestEntry{
			chainID: {
				Address: address,
				Denoms:  []string{denom},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling request body to bytes: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("creating http request: %w", err)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("making http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", handleError(resp.Body)
	}

	type BalancesResponseDenom struct {
		Amount          string `json:"amount"`
		Decimals        int    `json:"decimals"`
		FormattedAmount string `json:"formatted_amount"`
	}
	type BalancesResponseDenoms struct {
		Denoms map[string]BalancesResponseDenom `json:"denoms"`
	}
	type BalancesResponse struct {
		Chains map[string]BalancesResponseDenoms `json:"chains"`
	}

	var res BalancesResponse
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decoding response body: %w", err)
	}

	chainBalalances, ok := res.Chains[chainID]
	if !ok {
		return "", fmt.Errorf("no balance found for chain %s", chainID)
	}

	denomBalance, ok := chainBalalances.Denoms[denom]
	if !ok {
		return "", fmt.Errorf("no balance found for denom %s on chain %s", denom, chainID)
	}

	return denomBalance.Amount, nil
}

type RouteResponse struct {
	AmountIn               string   `json:"amount_in"`
	AmountOut              string   `json:"amount_out"`
	SourceAssetDenom       string   `json:"source_asset_denom"`
	SourceAssetChainID     string   `json:"source_asset_chain_id"`
	DestAssetDenom         string   `json:"dest_asset_denom"`
	DestAssetChainID       string   `json:"dest_asset_chain_id"`
	Operations             []any    `json:"operations"`
	ChainIDs               []string `json:"chain_ids"`
	RequiredChainAddresses []string `json:"required_chain_addresses"`
	DoesSwap               bool     `json:"does_swap"`
	EstimatedAmountOut     string   `json:"estimated_amount_out"`
	TxsRequired            int      `json:"txs_required"`
	USDAmountIn            string   `json:"usd_amount_in"`
	USDAmountOut           string   `json:"usd_amount_out"`
	SwapPriceImpactPercent string   `json:"swap_price_impact_percent"`
}

func (s *skipGoClient) Route(
	ctx context.Context,
	sourceAssetDenom string,
	sourceAssetChainID string,
	destAssetDenom string,
	destAssetChainID string,
	amountIn *big.Int,
) (*RouteResponse, error) {
	const endpoint = "/v2/fungible/route"
	u, err := url.JoinPath(s.baseURL.String(), endpoint)
	if err != nil {
		return nil, fmt.Errorf("joining base url to endpoint %s: %w", endpoint, err)
	}

	type SmartSwapOptions struct {
		EVMSwaps    bool `json:"evm_swaps"`
		SplitRoutes bool `json:"split_routes"`
	}

	type RouteRequest struct {
		SourceAssetDenom   string           `json:"source_asset_denom"`
		SourceAssetChainID string           `json:"source_asset_chain_id"`
		DestAssetDenom     string           `json:"dest_asset_denom"`
		DestAssetChainID   string           `json:"dest_asset_chain_id"`
		AmountIn           string           `json:"amount_in"`
		AllowMultiTx       bool             `json:"allow_multi_tx"`
		Bridges            []string         `json:"bridges"`
		AllowUnsafe        bool             `json:"allow_unsafe"`
		SmartSwapOptions   SmartSwapOptions `json:"smart_swap_options"`
	}

	body := RouteRequest{
		SourceAssetDenom:   sourceAssetDenom,
		SourceAssetChainID: sourceAssetChainID,
		DestAssetDenom:     destAssetDenom,
		DestAssetChainID:   destAssetChainID,
		AmountIn:           amountIn.String(),
		AllowMultiTx:       true,
		Bridges:            []string{"CCTP", "IBC"},
		AllowUnsafe:        true,
		SmartSwapOptions:   SmartSwapOptions{EVMSwaps: false, SplitRoutes: true},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body to bytes: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleError(resp.Body)
	}

	var res RouteResponse
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decoding response body: %w", err)
	}

	return &res, nil
}

type EVMTx struct {
	ChainID                string          `json:"chain_id"`
	To                     string          `json:"to"`
	Value                  string          `json:"value"`
	Data                   string          `json:"data"`
	SignerAddress          string          `json:"signer_address"`
	RequiredERC20Approvals []ERC20Approval `json:"required_erc20_approvals"`
}

type ERC20Approval struct {
	TokenContract string `json:"token_contract"`
	Spender       string `json:"spender"`
	Amount        string `json:"amount"`
}

type CosmosMessage struct {
	Msg        string `json:"msg"`
	MsgTypeURL string `json:"msg_type_url"`
}

type CosmosTx struct {
	ChainID       string          `json:"chain_id"`
	Path          []string        `json:"path"`
	SignerAddress string          `json:"signer_address"`
	Msgs          []CosmosMessage `json:"msgs"`
}

type Tx struct {
	EVMTx             *EVMTx    `json:"evm_tx"`
	CosmosTx          *CosmosTx `json:"cosmos_tx"`
	OperationsIndices []int     `json:"operations_indices"`
}

func (s *skipGoClient) Msgs(
	ctx context.Context,
	sourceAssetDenom string,
	sourceAssetChainID string,
	sourceChainAddress string,
	destAssetDenom string,
	destAssetChainID string,
	destChainAddress string,
	amountIn *big.Int,
	amountOut *big.Int,
	addressList []string,
	operations []any,
) ([]Tx, error) {
	const endpoint = "/v2/fungible/msgs"
	u, err := url.JoinPath(s.baseURL.String(), endpoint)
	if err != nil {
		return nil, fmt.Errorf("joining base url to endpoint %s: %w", endpoint, err)
	}

	type MsgsRequest struct {
		SourceAssetDenom         string   `json:"source_asset_denom"`
		SourceAssetChainID       string   `json:"source_asset_chain_id"`
		DestAssetDenom           string   `json:"dest_asset_denom"`
		DestAssetChainID         string   `json:"dest_asset_chain_id"`
		AmountIn                 string   `json:"amount_in"`
		AmountOut                string   `json:"amount_out"`
		SlippageTolerancePercent string   `json:"slippage_tolerance_percent"`
		AddressList              []string `json:"address_list"`
		Operations               []any    `json:"operations"`
	}

	body := MsgsRequest{
		SourceAssetDenom:         sourceAssetDenom,
		SourceAssetChainID:       sourceAssetChainID,
		DestAssetDenom:           destAssetDenom,
		DestAssetChainID:         destAssetChainID,
		AmountIn:                 amountIn.String(),
		AmountOut:                amountOut.String(),
		SlippageTolerancePercent: "3",
		AddressList:              addressList,
		Operations:               operations,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body to bytes: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleError(resp.Body)
	}

	type MsgsResponse struct {
		Txs []Tx `json:"txs"`
	}
	var res MsgsResponse
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decoding response body: %w", err)
	}

	return res.Txs, nil
}

func (s *skipGoClient) SubmitTx(
	ctx context.Context,
	tx []byte,
	chainID string,
) (TxHash, error) {
	const endpoint = "/v2/tx/submit"
	u, err := url.JoinPath(s.baseURL.String(), endpoint)
	if err != nil {
		return "", fmt.Errorf("joining base url to endpoint %s: %w", endpoint, err)
	}

	type SubmitRequest struct {
		Tx      string `json:"tx"`
		ChainID string `json:"chain_id"`
	}

	encodedTx := base64.StdEncoding.EncodeToString(tx)
	body := SubmitRequest{
		Tx:      encodedTx,
		ChainID: chainID,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling request body to bytes: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("creating http request: %w", err)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("making http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code %d returned from Skip Go when submitting transaction %s: %w", resp.StatusCode, encodedTx, handleError(resp.Body))
	}

	type SubmitResponse struct {
		TxHash string `json:"tx_hash"`
	}
	var res SubmitResponse
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decoding response body: %w", err)
	}

	return TxHash(res.TxHash), nil
}

func (s *skipGoClient) TrackTx(
	ctx context.Context,
	txHash string,
	chainID string,
) (TxHash, error) {
	const endpoint = "/v2/tx/track"
	u, err := url.JoinPath(s.baseURL.String(), endpoint)
	if err != nil {
		return "", fmt.Errorf("joining base url to endpoint %s: %w", endpoint, err)
	}

	type TrackRequest struct {
		TxHash  string `json:"tx_hash"`
		ChainID string `json:"chain_id"`
	}

	body := TrackRequest{
		TxHash:  txHash,
		ChainID: chainID,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling request body to bytes: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("creating http request: %w", err)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("making http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		e := handleError(resp.Body)
		if strings.Contains(e.Error(), "failed to fetch tx from chain") {
			return "", ErrTxNotFound{
				TxHash:  txHash,
				ChainID: chainID,
			}
		}
		return "", fmt.Errorf("status code %d returned from Skip Go when submitting transaction to /track %s: %w", resp.StatusCode, txHash, e)
	}

	type TrackResponse struct {
		TxHash string `json:"tx_hash"`
	}
	var res TrackResponse
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decoding response body: %w", err)
	}

	return TxHash(res.TxHash), nil
}

type StatusResponse struct {
	Transfers []Transfer `json:"transfers"`
}

type Transfer struct {
	State                TransactionState     `json:"state"`
	TransferSequence     []TransferSequence   `json:"transfer_sequence"`
	NextBlockingTransfer *TransferSequence    `json:"next_blocking_transfer,omitempty"`
	TransferAssetRelease TransferAssetRelease `json:"transfer_asset_release,omitempty"`
	Error                *string              `json:"error,omitempty"`
}

type TransferSequence struct {
	IBCTransfer       *IBCTransfer     `json:"ibc_transfer,omitempty"`
	AxelarTransfer    *AxelarTransfer  `json:"axelar_transfer,omitempty"`
	CCTPTransfer      *GenericTransfer `json:"cctp_transfer,omitempty"`
	HyperlaneTransfer *GenericTransfer `json:"hyperlane_transfer,omitempty"`
	OpinitTransfer    *GenericTransfer `json:"opinit_transfer,omitempty"`
}

type TransferAssetRelease struct {
	ChainID  string `json:"chain_id"`
	Denom    string `json:"denom"`
	Released bool   `json:"released"`
}

type StatusTrackingTx struct {
	ChainID      string `json:"chain_id"`
	ExplorerLink string `json:"explorer_link"`
	TxHash       string `json:"tx_hash"`
}

type IBCTransfer struct {
	FromChainID string    `json:"from_chain_id"`
	ToChainID   string    `json:"to_chain_id"`
	State       string    `json:"state"`
	PacketTxs   PacketTxs `json:"packet_txs"`
}

type PacketTxs struct {
	SendTx        *StatusTrackingTx `json:"send_tx"`
	ReceiveTx     *StatusTrackingTx `json:"receive_tx"`
	AcknowledgeTx *StatusTrackingTx `json:"acknowledge_tx"`
	TimeoutTx     *StatusTrackingTx `json:"timeout_tx,omitempty"`
	Error         *string           `json:"error,omitempty"`
}

type AxelarTransfer struct {
	FromChainID    string       `json:"from_chain_id"`
	ToChainID      string       `json:"to_chain_id"`
	Type           string       `json:"type"`
	State          string       `json:"state"`
	Txs            SendTokenTxs `json:"txs"`
	AxelarScanLink string       `json:"axelar_scan_link"`
}

type SendTokenTxs struct {
	SendTx    *StatusTrackingTx `json:"send_tx"`
	ConfirmTx *StatusTrackingTx `json:"confirm_tx,omitempty"`
	ExecuteTx *StatusTrackingTx `json:"execute_tx"`
	Error     *string           `json:"error,omitempty"`
}

type GenericTransfer struct {
	ToChain   string           `json:"to_chain"`
	FromChain string           `json:"from_chain"`
	State     string           `json:"state"`
	SendTx    StatusTrackingTx `json:"send_tx"`
	ReceiveTx StatusTrackingTx `json:"receive_tx"`
}

type TransactionState string

const (
	STATE_SUBMITTED         TransactionState = "STATE_SUBMITTED"
	STATE_PENDING           TransactionState = "STATE_PENDING"
	STATE_COMPLETED_SUCCESS TransactionState = "STATE_COMPLETED_SUCCESS"
	STATE_COMPLETED_ERROR   TransactionState = "STATE_COMPLETED_ERROR"
	STATE_ABANDONED         TransactionState = "STATE_ABANDONED"
	STATE_PENDING_ERROR     TransactionState = "STATE_PENDING_ERROR"
)

func (s TransactionState) IsCompleted() bool {
	return s == STATE_COMPLETED_SUCCESS || s == STATE_COMPLETED_ERROR || s == STATE_ABANDONED || s == STATE_PENDING_ERROR
}

func (s TransactionState) IsCompletedError() bool {
	return s == STATE_COMPLETED_ERROR || s == STATE_ABANDONED || s == STATE_PENDING_ERROR
}

func (s *skipGoClient) Status(
	ctx context.Context,
	tx TxHash,
	chainID string,
) (*StatusResponse, error) {
	const endpoint = "/v2/tx/status"

	query := url.Values{}
	query.Add("tx_hash", string(tx))
	query.Add("chain_id", chainID)
	queryStr := query.Encode()

	u, err := url.JoinPath(s.baseURL.String(), endpoint)
	if err != nil {
		return nil, fmt.Errorf("joining base url to endpoint %s: %w", endpoint, err)
	}
	u += "?" + queryStr

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleError(resp.Body)
	}
	var res StatusResponse
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decoding response body: %w", err)
	}

	return &res, nil
}

type SkipGoError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

func (s SkipGoError) Error() string {
	return fmt.Sprintf("Skip Go Error: Code %d, Message %s, Details %+v", s.Code, s.Message, s.Details)
}

func handleError(body io.Reader) error {
	var e SkipGoError
	if err := json.NewDecoder(body).Decode(&e); err != nil {
		return fmt.Errorf("decoding skip go error: %w", err)
	}
	return e
}

type ErrTxNotFound struct {
	TxHash  string
	ChainID string
}

func (e ErrTxNotFound) Error() string {
	return fmt.Sprintf("failed to fetch tx %s from chain %s", e.TxHash, e.ChainID)
}
