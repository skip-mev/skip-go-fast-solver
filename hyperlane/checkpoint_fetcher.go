package hyperlane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/skip-mev/go-fast-solver/hyperlane/types"
)

type CheckpointFetcher interface {
	LatestIndex(ctx context.Context) (uint64, error)
	Checkpoint(ctx context.Context, index uint64) (*types.SignedCheckpoint, error)
	Validator() string
}

const (
	latestIndexFilePathLocalStorage = "index.json"
	latestIndexFilePathS3           = "checkpoint_latest_index.json"
)

var (
	ErrCheckpointDoesNotExist = fmt.Errorf("checkpoint does not exist")
)

func checkpointFilePathLocalStorage(index uint64) string {
	return fmt.Sprintf("%d_with_id.json", index)
}

func checkpointFilePathS3(index uint64) string {
	return fmt.Sprintf("checkpoint_%d_with_id.json", index)
}

func NewCheckpointFetcherFromStorageLocation(storageLocation string, validator string) (CheckpointFetcher, error) {
	if strings.HasPrefix(storageLocation, "file://") {
		return NewLocalFileFetcher(strings.TrimPrefix(storageLocation, "file://"), validator), nil
	}
	if strings.HasPrefix(storageLocation, "s3://") {
		return NewS3Fetcher(storageLocation, validator)
	}
	return nil, fmt.Errorf("no fetcher type found for storage location %s", storageLocation)
}

type LocalFileFetcher struct {
	path      string
	validator string
}

func NewLocalFileFetcher(path string, validator string) *LocalFileFetcher {
	return &LocalFileFetcher{path: path, validator: validator}
}

func (f *LocalFileFetcher) LatestIndex(ctx context.Context) (uint64, error) {
	path := path.Join(f.path, latestIndexFilePathLocalStorage)
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading file at %s: %w", path, err)
	}

	var index uint64
	if err = json.Unmarshal(contents, &index); err != nil {
		return 0, fmt.Errorf("unmarshaling latest index file %s contents: %w", path, err)
	}
	return index, nil
}

func (f *LocalFileFetcher) Checkpoint(ctx context.Context, index uint64) (*types.SignedCheckpoint, error) {
	path := path.Join(f.path, checkpointFilePathLocalStorage(index))
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, ErrCheckpointDoesNotExist
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file at %s: %w", path, err)
	}

	var checkpoint types.SignedCheckpoint
	if err = json.Unmarshal(contents, &checkpoint); err != nil {
		return nil, fmt.Errorf("unmarshaling checkpoint file %s contents: %w", path, err)
	}

	return &checkpoint, nil
}

func (f *LocalFileFetcher) Validator() string {
	return f.validator
}

type S3Fetcher struct {
	url       string
	validator string
	client    *http.Client
}

func NewS3Fetcher(storageLocation string, validator string) (*S3Fetcher, error) {
	locationString := strings.TrimPrefix(storageLocation, "s3://")
	locationSplit := strings.Split(locationString, "/")
	if len(locationSplit) < 2 {
		return nil, fmt.Errorf("invalid s3 storage location %s", storageLocation)
	}
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", locationSplit[0], locationSplit[1])
	if len(locationSplit) > 2 {
		url += "/" + strings.Join(locationSplit[2:], "/")
	}
	return &S3Fetcher{url: url, validator: validator, client: http.DefaultClient}, nil
}

func (f *S3Fetcher) LatestIndex(ctx context.Context) (uint64, error) {
	u, err := url.JoinPath(f.url, latestIndexFilePathS3)
	if err != nil {
		return 0, fmt.Errorf("joining base url %s and path %s: %w", f.url, latestIndexFilePathS3, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response body: %w", err)
	}
	var index uint64
	if err = json.Unmarshal(body, &index); err != nil {
		return 0, fmt.Errorf("unmarshaling latest index file %s contents: %w", u, err)
	}
	return index, nil
}

func (f *S3Fetcher) Checkpoint(ctx context.Context, index uint64) (*types.SignedCheckpoint, error) {
	u, err := url.JoinPath(f.url, checkpointFilePathS3(index))
	if err != nil {
		return nil, fmt.Errorf("joining base url %s and path %s: %w", f.url, checkpointFilePathS3(index), err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrCheckpointDoesNotExist
		}
		return nil, fmt.Errorf("unexpected status code %d fetching s3 checkpoint from %s", resp.StatusCode, u)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	var checkpoint types.SignedCheckpoint
	if err = json.Unmarshal(body, &checkpoint); err != nil {
		return nil, fmt.Errorf("unmarshaling checkpoint file %s contents: %w", u, err)
	}
	// we do this because for some reason the validator strips leading and trailing 0s from the R and S values when it serializes
	// the checkpoint and puts it in s3
	checkpoint.Signature.R = strings.TrimPrefix(checkpoint.SerializedSignature, "0x")[:64]
	checkpoint.Signature.S = strings.TrimPrefix(checkpoint.SerializedSignature, "0x")[64:128]
	return &checkpoint, nil
}

func (f *S3Fetcher) Validator() string {
	return f.validator
}
