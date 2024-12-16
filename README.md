# Skip Go Fast Solver

![Solver Flow](./img/solver_flow.png)
_(note: running the skip go fast transfer solver binary deploys both the solver and hyperlane relayer components needed)_

### Solver Service Description

Solvers monitor for user transfer order creation events emitted by the Skip Go Fast Transfer contract on the source chain. A solver assesses whether fulfilling the transfer aligns with their
risk/reward profile and if they have the required resources on the destination chain to fulfill it.

### User Transfer Intent Fulfillment Flow

1. **Order Submission**: A user on the source chain wants to transfer assets along with a message payload to a destination chain within seconds. They initiate this process by calling the `submitOrder` function on the
   Skip Go Fast Transfer Protocol contract deployed on the source chain. The user provides the necessary assets, message payload, any associated fees, and desired funds destination address on the destination chain. This action
   triggers an event that solvers monitor, and a unique ID is generated to map directly to the user’s transfer order intent.
2. **Solver Detection**: Solvers monitor for intent events emitted by the Skip Go Fast Transfer contract on the source chain. A solver assesses whether fulfilling the intent aligns with their
   risk/reward profile depending on the order size and if they have the required resources on the destination chain to fulfill the intent.
3. **Fulfillment**: The solver proceeds by calling the `fillOrder` function on the Skip Go Fast Transfer Protocol contract deployed on the destination chain, executing the transfer of assets and processing the message payload
   as specified. The contract on the destination chain then executes the intended action, whether it be transferring assets to a specified address or executing a contract call with the provided assets and payload. The solver’s
   address is recorded, and a unique ID is generated to link the solver’s fulfillment to the intent solved.
4. **Completion**: From the user’s perspective, the intended transfer and message execution (if applicable) are complete, and they have successfully achieved their goal on the destination chain.

### Solver Funds Settlement Flow

1. **Settlement Initiation**: After fulfilling one or more intents, the solver on the destination chain seeks to recover any assets they fronted, along with the fees earned. The solver initiates the settlement process by
   calling the `initiateSettlement` function on the Skip Go contract deployed on the destination chain, specifying the intent IDs they fulfilled and the address on the source chain where they wish to receive their compensation.
2. **Cross-Chain Verification**: The Skip Go contract on the destination chain verifies the solver’s fulfillment of the specified IDs and dispatches a cross-chain message via a message-passing protocol. The message-passing
   protocol’s verification service ingests the payload and attests to its validity for processing on the source chain.
3. **Relay and Settlement**: A relayer from the message-passing protocol takes the attested payload and relays it to the contracts deployed on the source chain. The `handle` function is then called on the Skip Go contract
   on the source chain, which verifies that the specified intents were accurately fulfilled based on the signing set’s attestation. Upon successful verification, the assets and fees are released to the solver’s specified
   address on the source chain.
4. **Completion**: The solver has now received the assets they fronted for the user, along with the service fee they earned, completing the settlement process.

### Latest Fast Transfer contract addresses

Use these addresses in the solver config and when using the CLI tool to submit transfers

- Arbitrum: https://arbiscan.io/address/0x23cb6147e5600c23d1fb5543916d3d5457c9b54c
- Optimism: https://optimistic.etherscan.io/address/0x0f479de4fd3144642f1af88e3797b1821724f703
- Polygon: https://polygonscan.com/address/0x3ffaf8d0d33226302e3a0ae48367cf1dd2023b1f
- Base: https://basescan.org/address/0x43d090025aaa6c8693b71952b910ac55ccb56bbb
- Avalanche: https://snowtrace.io/address/0xD415B02A7E91dBAf92EAa4721F9289CFB7f4E1cF
- Ethereum: https://etherscan.io/address/0xe7935104c9670015b21c6300e5b95d2f75474cda
- Osmosis: https://celatone.osmosis.zone/osmosis-1/contracts/osmo1vy34lpt5zlj797w7zqdta3qfq834kapx88qtgudy7jgljztj567s73ny82

### How to start server

1. Update [config/local/keys.json](config/local/keys.json) with the corresponding solver private keys and addresses.
2. Update [config/local/config.yml](config/local/config.yml) with the needed config values (solver addresses, chain rpc links, etc.). Values that need to be set are in `<>` brackets. Reference the [shared/config/config.go](shared/config/config.go) file for more info about the config fields.
   - The [config/local/config.yml](config/local/config.yml) config file is pre-filled with recommended values. To customize your solver deployment, reference [config/sample/config.yml](config/sample/config.yml) to see which config values can be modified.

```shell
make build # build solver server binary
# quickstart mode determines whether solver starts monitoring for user intent events from latest chain block height,
# or from the last block the solver has previously processed (set to true by default)
make run-solver
```

### How to run Solver docker image

```shell
# Choose right platform to build Docker image
docker build --platform <linux/amd64|linux/arm64> -t skip-go-fast-solver .
docker run skip-go-fast-solver
```

### How to run tests

```shell
make test # run all tests
```

### Database access

```shell
make db-exec  # access sqlite shell
make db-clean # clean all existing db entries
```

### CLI Tool

Build the CLI tool from the project root directory:

```shell
make build-cli
```

To make the solver command available system-wide, copy it to your PATH:

```shell
cp ./build/solvercli /usr/local/bin/solver && chmod 755 /usr/local/bin/solver
```

Now you can run the solver commands from anywhere. Available commands:

**submit-transfer**: Submit a fast transfer order to transfer USDC from EVM -> Osmosis

```shell
solver submit-transfer \
  --config <configFilePath> \
  --token <usdc address> \
  --recipient <destination address> \
  --amount <usdc amount> \
  --source-chain-id <source chain id> \
  --destination-chain-id <destination chain id> \
  --gateway <gateway contract> \
  --private-key <private key> \
  --deadline-hours <timeout in hours>
```

**relay**: Manually relay a hyperlane transaction

```shell
solver relay \
  --config <configFilePath> \
  --keys <keysFilePath> \
  --key-store-type <store type> \
  --aes-key-hex <hex key> \
  --origin-chain-id <chain id> \
  --originTxHash <tx hash> \
  --checkpoint-storage-location-override <storage path>
```

**balances**: Get current on-chain balances (USDC, gas token, and custom assets requested)

```shell
solver balances --custom-assets '{"osmosis-1":["uosmo","uion"],"celestia-1":["utia"]}'
```

**inventory**: Get complete solver inventory including balances, pending settlements, and pending rebalance transfers

```shell
solver inventory --custom-assets '{"osmosis-1":["uosmo","uion"],"celestia-1":["utia"]}'
```

**rebalances**: Get pending rebalance transfers

```shell
solver rebalances
```

**settlements**: Get pending order settlements

```shell
solver settlements
```

### Main Project Modules

- transfer monitor: monitors for user transfer intent events and creates pending order fills in the solver database
- order filler: monitors for pending user order fills and fulfills them
- order settler: monitors for completed order fills and initiates process to settle solver funds
- tx verifier: verifies the status of any pending transactions related to user transfers on chain and updates the solver database
  with their latest status
- fund rebalancer: constantly checks if any configured chains are below a specified funds threshold, ands tops up funds if needed
  from other chains that have spare funds
- hyperlane: used for cross chain communication during funds settlement to validate that the user transfer has been successfully fulfilled

### Hyperlane Docs

- [Hyperlane Docs Link](https://docs.hyperlane.xyz/)

### Key Management

Besides storing keys in a plaintext config file, users can also:

- Run the solver with the flag `--key-store-type env` and store a json string of the keys map in the environment variable `SOLVER_KEYS`.
  The keys map json string should be formed like [this](config/local/keys.json).
- Store keys in an encrypted file and run the solver with the flags `--key-store-type encrypted-file --keys <path to encrypted keys file>`.
  Must set the hex encoded aes encryption key in the environment variable `AES_KEY_HEX`.
  An example of how the encrypted file should be formed can be found [here](examples/encrypted_key_store/main.go).
