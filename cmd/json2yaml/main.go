package main

import (
	"encoding/json"
	"os"

	yaml "gopkg.in/yaml.v2"
)

func main() {
	in := make(map[string]interface{})
	decoder := json.NewDecoder(os.Stdin)
	err := decoder.Decode(&in)
	if err != nil {
		panic(err)
	}
	out, err := yaml.Marshal(&in)
	if err != nil {
		panic(err)
	}
	_, err = os.Stdout.Write(out)
	if err != nil {
		panic(err)
	}
}
