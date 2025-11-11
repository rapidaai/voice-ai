package parsers

import (
	"github.com/flosch/pongo2/v6"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

type pongo2TemplateParser struct {
	logger commons.Logger
}

type pongo2StringTemplateParser struct {
	pongo2TemplateParser
}

type pongo2MessageTemplateParser struct {
	pongo2TemplateParser
}

func NewPongo2StringTemplateParser(logger commons.Logger) StringTemplateParser {
	return &pongo2StringTemplateParser{
		pongo2TemplateParser: pongo2TemplateParser{logger: logger},
	}
}

func NewPongo2MessageTemplateParser(logger commons.Logger) MessageTemplateParser {
	return &pongo2MessageTemplateParser{
		pongo2TemplateParser: pongo2TemplateParser{logger: logger},
	}
}

func (stp *pongo2StringTemplateParser) Parse(template string, argument map[string]interface{}) string {
	tpl, err := pongo2.FromString(template)
	if err != nil {
		stp.logger.Errorf("error while parsing the template with pongo2: %v", err)
		return template
	}

	formattedTemplate, err := tpl.Execute(pongo2.Context(argument))
	if err != nil {
		stp.logger.Errorf("error while executing the template with pongo2: %v", err)
		return template
	}

	return formattedTemplate
}

func (stp *pongo2MessageTemplateParser) Parse(template *types.Message, argument map[string]interface{}) *types.Message {
	for ix, v := range template.Contents {
		if commons.ResponseContentType(v.GetContentType()) == commons.TEXT_CONTENT &&
			commons.ResponseContentFormat(v.GetContentFormat()) == commons.TEXT_CONTENT_FORMAT_RAW {
			tpl, err := pongo2.FromString(string(v.Content))
			if err != nil {
				stp.logger.Errorf("error while parsing the template with pongo2: %v", err)
				continue
			}

			formattedTemplate, err := tpl.Execute(pongo2.Context(argument))
			if err != nil {
				stp.logger.Errorf("error while executing the template with pongo2: %v", err)
				continue
			}
			template.Contents[ix].Content = []byte(formattedTemplate)
		}
	}
	return template
}
