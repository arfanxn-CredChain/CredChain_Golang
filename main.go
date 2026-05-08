package main

import (
	"log"

	"CredChain_Golang/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
