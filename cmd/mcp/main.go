package main

import (
	"context"
	"log"
	"os"
	"stockmind/internal/mcp"
)

func main() {
	protocol := "stdio"
	if len(os.Args) > 1 {
		protocol = os.Args[1]
	}
	
	if err := mcp.Start(context.Background(), protocol); err != nil {
		log.Fatal(err)
	}
}
