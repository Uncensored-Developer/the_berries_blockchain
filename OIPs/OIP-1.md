# Dynamic Transaction Cost
## Current Context
Every OPB transfer costs miner's `txFee` hardcoded to 20 OPB tokens to prevent users from spamming the network.

```go
func (t Txn) TotalCost() uint {
	return t.Value + TxnFee
}
```

There are a few downsides to this approach:
- Depending on the price of a single OPB token, the $ cost of spending 20 OPB tokens could make the network too expensive to use
- The opposite also applies, if a single OPB token price is too low, the network is susceptible to DOS (denial of service) attack
- Hardcoded cost limits the network to only one type of operations and its costs, transfers (maybe we define a new OIP later and implement Smart Contracts via a Virtual Machine).


### What Ethereum does

The official [go-ethereum/core/types/transaction.go](https://github.com/ethereum/go-ethereum/blob/57feabea663496109e59df669238398239438fb1/core/types/transaction.go#L296) has several fee/gas related methods:
```go
// Cost returns gas * gasPrice + value.
func (tx *Transaction) Cost() *big.Int {
    total := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
    total.Add(total, tx.Value())
    return total
}

// Gas returns the gas limit of the transaction.
func (tx *Transaction) Gas() uint64 { return tx.inner.gas() }

// GasPrice returns the gas price of the transaction.
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.inner.gasPrice()) }

// GasTipCap returns the gasTipCap per gas of the transaction.
func (tx *Transaction) GasTipCap() *big.Int { return new(big.Int).Set(tx.inner.gasTipCap()) }

// GasFeeCap returns the fee cap per gas of the transaction.
func (tx *Transaction) GasFeeCap() *big.Int { return new(big.Int).Set(tx.inner.gasFeeCap()) }
```

The final Ethereum `tx.Cost()` is calculated as **gas * gasPrice + value**.
```go
// Cost returns gas * gasPrice + value.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
	total.Add(total, tx.Value())
	return total
}
```

#### Example

The legacy Ethereum TX transfer from account A to account B costs 21 000 gas. The Price of Gas depends on the network usage. If the Price is 2 GWei per 1 Gas, the Gas Price would be 2 * 21 000 = 42 000 Gwei or 0.000042 ETH.

The Ether currency structure with **Wei** being the smallest unit:
```go
const (
    Wei   = 1
    GWei  = 1e9
    Ether = 1e18
)
```

## New Specification
The OPB protocol will define how much **Gas** each action will require, and the user and network economics will decide the **Gas Price**.

| Action | Gas Required |
|--------|--------------|
| Transfer | 10           |

Each Transaction will require two new attributes:
- **Gas:** user must set this value to 10
- **GasPrice:** user decides how much he wants to spend, miner chooses whenever it's enough to include this transaction into a block and cover the mining costs

The default **GasPrice value will be 1 OPB** token for simplicity (1 OPB token is the smallest unit).

Miners can create a new The OnePieceBerriesBlockchain Improvement Proposal and define according to what criteria the **GasPrice** will be sufficient to pay for a transaction to get included into a block.

```go
type Txn struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	
	Gas      uint           `json:"gas"`
	GasPrice uint           `json:"gasPrice"`
	
	Value    uint           `json:"value"`
	Nonce    uint           `json:"nonce"`
	Data     string         `json:"data"`
	Time     uint64         `json:"time"`
}
```

## Proposed Consensus Fork Number
Block number 10.