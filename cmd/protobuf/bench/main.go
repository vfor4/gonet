package main

import (
	"encoding/json"
	"fmt"
	"log"

	userpb "github.com/vfor4/gonet/cmd/protobuf/bench/pbuser"
	"google.golang.org/protobuf/proto"
)

// JSON Struct
type UserJSON struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	// 1. JSON Example
	userJSON := UserJSON{Name: "Alice", Age: 25}
	jsonBytes, err := json.Marshal(userJSON)
	if err != nil {
		log.Fatalf("JSON Marshal failed: %v", err)
	}
	fmt.Println("JSON Text:", jsonBytes)
	fmt.Println("JSON Size:", len(jsonBytes), "bytes")

	// 2. Protobuf Example
	userProto := &userpb.User{
		Name: "Alice",
		Age:  25,
	}
	protoBytes, err := proto.Marshal(userProto)
	if err != nil {
		log.Fatalf("Protobuf Marshal failed: %v", err)
	}
	fmt.Println("Protobuf Bytes:", protoBytes)
	fmt.Println("Protobuf Size:", len(protoBytes), "bytes")
}
