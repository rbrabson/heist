package main

import (
	"fmt"

	"github.com/rbrabson/heist/pkg/store"
)

func main() {
	db := "./store/"
	store := store.NewStore(db)

	idList := store.ListDocuments("heist")
	for _, id := range idList {
		fmt.Println(id)
	}
}
