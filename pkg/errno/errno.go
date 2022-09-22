package errno

import (
	"encoding/json"
	"log"
)

type Errno struct {
	State int    `json:"state"`
	Msg   string `json:"msg"`
}

func (e Errno) Error() string {
	return e.Msg
}

func (e Errno) Add(s string) *Errno {
	e.Msg += ": " + s
	return &e
}

func (e *Errno) Encode() []byte {
	b, err := json.Marshal(e)
	if err != nil {
		log.Println(err)
		return nil
	}
	return b
}

func (e *Errno) String() string {
	return string(e.Encode())
}
