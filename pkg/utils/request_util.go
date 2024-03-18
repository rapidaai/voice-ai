package utils

import (
	"encoding/json"

	web_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

func Cast(orig interface{}, dst interface{}) error {
	orignalObj, err := json.Marshal(orig)
	if err != nil {
		return err
	}
	err = json.Unmarshal(orignalObj, dst)
	if err != nil {
		return err
	}
	return nil
}

func IndexFunc[S ~[]E, E any](s S, f func(E) bool) int {
	for i := range s {
		if f(s[i]) {
			return i
		}
	}
	return -1
}

func Error[R any](message, humanMessage string) *R {
	data := struct {
		Code    int32
		Success bool
		Error   *web_api.Error
	}{
		Code:    400,
		Success: false,
		Error: &web_api.Error{
			ErrorCode:    400,
			ErrorMessage: message,
			HumanMessage: humanMessage,
		}}

	var result R
	b, _ := json.Marshal(&data)
	_ = json.Unmarshal(b, &result)
	return &result
}

func FromError[R any](message error) *R {
	data := struct {
		Code    int32
		Success bool
		Error   *web_api.Error
	}{
		Code:    400,
		Success: false,
		Error: &web_api.Error{
			ErrorCode:    400,
			ErrorMessage: message.Error(),
			HumanMessage: message.Error(),
		}}

	var result R
	b, _ := json.Marshal(&data)
	_ = json.Unmarshal(b, &result)
	return &result
}
