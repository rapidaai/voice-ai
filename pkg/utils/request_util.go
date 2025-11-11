package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	web_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func Error[R any](err error, humanMessage string) (*R, error) {
	return ErrorWithCode[R](400, err, humanMessage)
}

func AuthenticateError[R any]() (*R, error) {
	return ErrorWithCode[R](401, errors.New("unauthenticated request"), "Unauthenticated requet, please try again with valid authentication.")
}

func JustError(code int32, err error, humanMessage string) *web_api.Error {
	return &web_api.Error{
		ErrorCode:    uint64(code),
		ErrorMessage: err.Error(),
		HumanMessage: humanMessage,
	}
}

func ErrorWithCode[R any](code int32, err error, humanMessage string) (*R, error) {
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

	if code == 200 {
		return &result, nil
	}
	return &result, err
}

func PaginatedSuccess[R any, T any](totalItem, currentPage uint32, out T) (*R, error) {
	data := struct {
		Code      int32
		Success   bool
		Paginated *web_api.Paginated
		Data      T
	}{
		Paginated: &web_api.Paginated{
			TotalItem:   totalItem,
			CurrentPage: currentPage,
		},
		Code:    200,
		Success: true,
		Data:    out,
	}

	var result R
	b, _ := json.Marshal(&data)
	_ = json.Unmarshal(b, &result)
	return &result, nil
}

func Success[R any, T any](out T) (*R, error) {
	data := struct {
		Code    int32
		Success bool
		Data    T
	}{
		Code:    200,
		Success: true,
		Data:    out,
	}

	var result R
	b, _ := json.Marshal(&data)
	_ = json.Unmarshal(b, &result)
	return &result, nil
}

func JustSuccess() (*web_api.BaseResponse, error) {
	return &web_api.BaseResponse{
		Success: true,
		Code:    200,
	}, nil
}

func Cast(orig interface{}, dst interface{}) error {
	orignalObj, err := json.Marshal(orig)
	if err != nil {
		fmt.Printf("error while castig %v", err)
		return err
	}
	err = json.Unmarshal(orignalObj, dst)
	if err != nil {
		fmt.Printf("error while castig %v", err)
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

func ToString(in []string) string {
	builder := new(strings.Builder)
	err := json.NewEncoder(builder).Encode(in)
	if err != nil {
		fmt.Printf("error while converting array to string %v", err)
		return ""
	}
	return builder.String()
}

func Uint64SliceToString(in []uint64) string {
	builder := new(strings.Builder)
	err := json.NewEncoder(builder).Encode(in)
	if err != nil {
		fmt.Printf("error while converting uint64 slice to string: %v", err)
		return ""
	}
	return builder.String()
}

func MapToStruct(m map[string]interface{}) *structpb.Struct {
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}

func ProtoJson(m proto.Message) string {
	return protojson.Format(m)
}
