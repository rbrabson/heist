package main

import (
	"fmt"

	"github.com/rbrabson/heist/pkg/store"
)

func main() {
	store := store.NewStore()

	idList := store.ListDocuments("heist")
	for _, id := range idList {
		fmt.Println(id)
	}
}
