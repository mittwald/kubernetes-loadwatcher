package jsonpatch

import "encoding/json"

type PatchList []Patch

type Patch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (p Patch) ToJSON() []byte {
	j, err := json.Marshal(&p)
	if err != nil {
		panic(err)
	}

	return j
}

func (p PatchList) ToJSON() []byte {
	j, err := json.Marshal(&p)
	if err != nil {
		panic(err)
	}

	return j
}
