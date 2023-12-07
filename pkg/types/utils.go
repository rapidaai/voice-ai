package types

import (
	"encoding/json"
)

func Cast(orig interface{}, dst interface{}) error {
	orignalObj, err := json.Marshal(orig)
	if err != nil {
		return err
	}
	json.Unmarshal(orignalObj, dst)
	return nil
}
