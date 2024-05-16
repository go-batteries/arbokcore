package utils

import (
	"encoding/json"
	"log"
)

func Dump(stuff any) {
	b, err := json.Marshal(stuff)
	if err != nil {
		log.Printf("%T %v", stuff, stuff)
		return
	}

	log.Println("dump:", string(b), stuff)
}
