package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

func Dump(stuff any) {
	b, err := json.MarshalIndent(stuff, "", " ")
	if err != nil {
		log.Printf("%T %v", stuff, stuff)
		return
	}

	log.Println("dump:", string(b))
	log.Println(stuff)
}

func GetSize(reader io.Reader) int {
	buf := &bytes.Buffer{}
	nRead, err := io.Copy(buf, reader)
	if err != nil {
		fmt.Println(err)
	}

	return int(nRead)
}
