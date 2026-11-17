package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	var absent []string
	empty := make([]string, 0)
	for _, example := range []struct {
		name  string
		items []string
	}{{"nil slice", absent}, {"empty slice", empty}} {
		data, _ := json.Marshal(example.items)
		fmt.Printf("%s: %s\n", example.name, data)
	}
}
