package main

import (
	"fmt"
	"time"
)

func main() {
	for {
		fmt.Println("processing jobs...")
		time.Sleep(time.Second)
	}
}
