package main

import (
	"encoding/json"
	"strings"

	"github.com/HughNian/nmid/pkg/direct"
)

type Params struct {
	Name string `json:"name"`
}

func main() {
	s := direct.NewServer("tcp", "0.0.0.0:7900")
	s.Register("ToUpper", func(payload []byte) ([]byte, error) {
		var p Params
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		return []byte(strings.ToUpper(p.Name)), nil
	})
	_ = s.ListenAndServe()
}

