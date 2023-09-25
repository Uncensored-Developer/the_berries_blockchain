package database

type Account string

type Txn struct {
	From  Account `json:"from"`
	To    Account `json:"to"`
	Value uint    `json:"value"`
	Data  string  `json:"data"`
}

func (t Txn) IsReward() bool {
	return t.Data == "reward"
}
