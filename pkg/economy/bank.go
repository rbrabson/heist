package economy

import "time"

type Banks struct {
	Banks map[string]Bank
}

type Bank struct {
	ServerID string
	Accounts map[string]*Account
}

type Account struct {
	UserID    string
	UserName  string
	Balance   uint
	CreatedAt time.Time
}

func NewBank(serverID string) *Bank {
	b := Bank{
		ServerID: serverID,
		Accounts: make(map[string]*Account),
	}
	return &b
}

func NewAccount(userID string, userName string, initialBalance uint) *Account {
	a := Account{
		UserID:    userID,
		UserName:  userName,
		Balance:   initialBalance,
		CreatedAt: time.Now(),
	}
	return &a
}

func (b *Bank) CreateAccount(userID string, userName string, initialBalance uint) error {
	if _, ok := b.Accounts[userID]; ok {
		return ErrAccountAlreadyExist
	}

	a := NewAccount(userID, userName, initialBalance)
	b.Accounts[a.UserID] = a

	return nil
}

func (b *Bank) GetAccount(userID string) (*Account, error) {
	a, ok := b.Accounts[userID]
	if !ok {
		return nil, ErrNoAccount
	}
	return a, nil
}

func (b *Bank) WipeBank() {
	b.Accounts = make(map[string]*Account)
}

func (a *Account) WithdrawCredits(amount uint) error {
	if a.Balance < amount {
		return ErrInsufficintBalance
	}

	a.Balance -= amount
	return nil
}

func (a *Account) DepositCredits(amount uint) {
	a.Balance += amount
}

func (a *Account) TransferCredits(receiver *Account, amount uint) error {
	if a.UserID == receiver.UserID {
		return ErrSameSenderAndReceiver
	}
	if a.Balance < amount {
		return ErrInsufficintBalance
	}

	a.WithdrawCredits(amount)
	receiver.DepositCredits(amount)
	return nil
}

func (a *Account) CanSpend(amount uint) bool {
	return a.Balance >= amount
}

func (a *Account) GetBalance() uint {
	return a.Balance
}
