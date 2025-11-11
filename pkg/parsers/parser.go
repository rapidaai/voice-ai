package parsers

import "github.com/rapidaai/pkg/types"

type Parser[In, Out any] interface {
	Parse(u In, argument map[string]interface{}) Out
}

type StringTemplateParser interface {
	Parser[string, string]
}

type MessageTemplateParser interface {
	Parser[*types.Message, *types.Message]
}
