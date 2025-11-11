package exceptions

import (
	"encoding/json"
	"errors"

	web_api "github.com/rapidaai/protos"
)

const (
	Unauthorized    = 401
	APIUnauthorized = 403
	NotFound        = 404
	InternalServer  = 500
	BadRequest      = 400
)

// ErrorMessages maps HTTP status codes to error messages.
var ErrorMessages = map[int]string{
	Unauthorized:    "Unauthenticated request, please try again with valid authentication.",
	APIUnauthorized: "Invalid API key, please provide a valid key.",

	NotFound:       "Resource not found, please check the endpoint and try again.",
	InternalServer: "Internal server error, please try again later.",
}

func APIAuthenticationError[R any]() (*R, error) {
	return ErrorWithCode[R](APIUnauthorized, errors.New("unauthenticated request"), ErrorMessages[APIUnauthorized]), nil
}

func AuthenticationError[R any]() (*R, error) {
	return ErrorWithCode[R](Unauthorized, errors.New("unauthenticated request"), ErrorMessages[Unauthorized]), nil
}

func BadRequestError[R any](err string) (*R, error) {
	return ErrorWithCode[R](BadRequest, errors.New("bad request"), err), nil
}

func InternalServerError[R any](err error, msg string) (*R, error) {
	return ErrorWithCode[R](InternalServer, err, msg), nil
}

func ErrorWithCode[R any](code int32, err error, humanMessage string) *R {
	data := struct {
		Code    int32
		Success bool
		Error   *web_api.Error
	}{
		Code:    code,
		Success: false,
		Error: &web_api.Error{
			ErrorCode:    uint64(code),
			ErrorMessage: err.Error(),
			HumanMessage: humanMessage,
		}}

	var result R
	b, _ := json.Marshal(&data)
	_ = json.Unmarshal(b, &result)
	return &result
}
