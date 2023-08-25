# Cadence-lint

The linter for Cadence. Find programming errors, bugs, stylistic errors and suspicious constructs.

## How to Build

```shell
go build ./cmd/lint
```

### Analyzing contracts of an account

To analyze all contracts of an account, specify the network and address.
This requires you have the [Flow CLI](https://docs.onflow.org/flow-cli/) installed and configured (run `flow init`).

For example:

```shell
./lint -network mainnet -address 0x1654653399040a61
```

### Analyzing a transaction

To analyze a transaction, specify the network and transaction ID.
This requires you have the [Flow CLI](https://docs.onflow.org/flow-cli/) installed and configured (run `flow init`).

For example:

```shell
./lint -network mainnet -transaction 44fd8475eeded90d74e7594b10cf456b0866c78221e7f230fcfd4ba1155c542f
```

### Only running some analyzers

By default, all available analyzers are run.

To list all available analyzers, run:

```shell
./lint -help
```

For example, to only run the `reference-to-optional` and the `external-mutation` analyzers, run:

```shell
./lint -network mainnet -address 0x1654653399040a61 \
    -analyze reference-to-optional \
    -analyze external-mutation
```

### Analyzing contracts in a directory

To analyze all contracts in a directory, specify the path.

For example:

```bash
./lint -directory contracts
```

The files must be named with the `.cdc` extension and by their location ID of the program:
- Contracts in accounts have the format `A.<address>.<name>`,
  e.g. `A.e467b9dd11fa00df.FlowStorageFees`, where
    - `address`: Address in hex format, e.g. `e467b9dd11fa00df`
    - `name`: The name of the contract, e.g `FlowStorageFees`
- Transactions have the format `t.<ID>`, where
    - `id`: The ID of the transaction (its hash)
- Scripts have the format `s.<ID>`, where
    - `id`: The ID of the script (its hash)


### Analyzing contracts in a CSV file

To analyze all contracts in a CSV file, specify the path to the file.

For example:

```shell
./lint -csv contracts.csv
```

The CSV file must be in the following format:

- Header: `location,code`
- Columns:
  - `location`: The location ID of the program
     - Contracts in accounts have the format `A.<address>.<name>`,
        e.g. `A.e467b9dd11fa00df.FlowStorageFees`, where
         - `address`: Address in hex format, e.g. `e467b9dd11fa00df`
         - `name`: The name of the contract, e.g `FlowStorageFees`
     - Transactions have the format `t.<ID>`, where
       - `id`: The ID of the transaction (its hash)
     - Scripts have the format `s.<ID>`, where
       - `id`: The ID of the script (its hash)
  - `code`: The code of the contract, e.g. `access(all) contract Test {}`

Full example:

```csv
location,code
t.0000000000000000,"
import 0x1

transaction {
    prepare(signer: AuthAccount) {
        Test.hello()
    }
}
"
A.0000000000000001.Test,"
access(all) contract Test {

    access(all) fun hello() {
      log(""Hello, world!"")
    }
}
"
```
