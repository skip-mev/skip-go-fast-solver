package cosmos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cosmwasm "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/skip-mev/go-fast-solver/hyperlane/types"
)

type ValidatorAnnounceQuerier struct {
	client  cosmwasm.QueryClient
	address string
}

func NewValidatorAnnounceQuerier(address string, client cosmwasm.QueryClient) *ValidatorAnnounceQuerier {
	return &ValidatorAnnounceQuerier{client, address}
}

type GetAnnounceValidatorStorageLocationsRequest struct {
	GetAnnounceStorageLocations GetAnnounceStorageLoctions `json:"get_announce_storage_locations"`
}

type GetAnnounceStorageLoctions struct {
	Validators []string `json:"validators"`
}

func (va *ValidatorAnnounceQuerier) GetAnnouncedValidatorStorageLocations(ctx context.Context, validators []string) ([]*types.ValidatorStorageLocation, error) {
	var stripped []string
	for _, v := range validators {
		stripped = append(stripped, strings.TrimPrefix(v, "0x"))
	}
	req := GetAnnounceValidatorStorageLocationsRequest{
		GetAnnounceStorageLoctions{
			Validators: stripped,
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling get announced validators storage locations request: %w", err)
	}

	resp, err := va.client.SmartContractState(ctx, &cosmwasm.QuerySmartContractStateRequest{
		Address:   va.address,
		QueryData: data,
	})
	if err != nil {
		return nil, fmt.Errorf("querying smart contract %s for announced validator storage locations: %w", va.address, err)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("got nil response when querying for announced validator storage locations")
	}
	type StorageLocationsResponse struct {
		StorageLocations [][]any `json:"storage_locations"`
	}
	var validatorLocations StorageLocationsResponse
	if err := json.Unmarshal(resp.Data, &validatorLocations); err != nil {
		return nil, fmt.Errorf("unmarshaling response bytes into storage locations json: %w", err)
	}

	var validatorStorageLocations []*types.ValidatorStorageLocation
	for _, validatorLocation := range validatorLocations.StorageLocations {
		// each entry in the array is a two item slice, the first is the
		// validator address as a string, the second is an array of storage
		// locations, we will simply take the last announced location as the
		// one the validator is intending to use
		if len(validatorLocation) != 2 {
			return nil, fmt.Errorf("expected two elements in validator location array, instead got %d", len(validatorLocation))
		}

		validator, ok := validatorLocation[0].(string)
		if !ok {
			return nil, fmt.Errorf("got unexpected type for first element of validator storage location, expected string")
		}
		locationsAny, ok := validatorLocation[1].([]any)
		if !ok {
			return nil, fmt.Errorf("got unexpected type for second element of validator storage location, expected []any")
		}
		if len(locationsAny) == 0 {
			return nil, fmt.Errorf("expected at least one storage location for validator %s, got none", validator)
		}
		location, ok := locationsAny[len(locationsAny)-1].(string)
		if !ok {
			return nil, fmt.Errorf("got unexpected type for second element of validator storage location, expected string")
		}

		validatorStorageLocation := &types.ValidatorStorageLocation{
			Validator:       validator,
			StorageLocation: location,
		}
		validatorStorageLocations = append(validatorStorageLocations, validatorStorageLocation)
	}

	return validatorStorageLocations, nil
}
