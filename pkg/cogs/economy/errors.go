package economy

import "errors"

var ErrAccountAlreadyExist = errors.New("account already exists")
var ErrInsufficintBalance = errors.New("account has insufficent funds")
var ErrSameSenderAndReceiver = errors.New("the transmitter and receiver are the same account")
var ErrNoAccount = errors.New("the account does not exist")
