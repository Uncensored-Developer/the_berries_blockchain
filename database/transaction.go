package database

type Account string

func NewAccount(value string) Account {
	return Account(value)
}

type Txn struct {
	From  Account `json:"from"`
	To    Account `json:"to"`
	Value uint    `json:"value"`
	Data  string  `json:"data"`
}

func (t Txn) IsReward() bool {
	return t.Data == "reward"
}

func NewTxn(from Account, to Account, value uint, data string) Txn {
	return Txn{from, to, value, data}
}
