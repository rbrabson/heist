package store

import (
	"github.com/rbrabson/heist/pkg/economy"
)

type mongo struct {
}

func newMongoStore() Store {
	m := &mongo{}
	return m
}

func (m *mongo) SaveHeistState([]byte) {

}

func (m *mongo) SaveEnconomyState(economy.Banks) {

}
