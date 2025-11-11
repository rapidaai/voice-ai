package parsers

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestPongo2MessageTemplateParser_Parse(t *testing.T) {
	parser := NewPongo2MessageTemplateParser(commons.NewApplicationLogger())
	tests := []struct {
		name     string
		template *types.Message
		argument map[string]interface{}
		expected *types.Message
	}{
		{
			name: "Simple template parsing",
			template: &types.Message{
				Contents: []*types.Content{
					{
						Content:       []byte("Hello, {{ name }}!"),
						ContentType:   commons.TEXT_CONTENT.String(),
						ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
					},
				},
			},
			argument: map[string]interface{}{
				"name": "John",
			},
			expected: &types.Message{
				Contents: []*types.Content{
					{
						Content:       []byte("Hello, John!"),
						ContentType:   commons.TEXT_CONTENT.String(),
						ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
					},
				},
			},
		},
		{
			name: "Multiple contents with mixed types",
			template: &types.Message{
				Contents: []*types.Content{
					{
						Content:       []byte("Hello, {{ name }}!"),
						ContentType:   commons.TEXT_CONTENT.String(),
						ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
					},
					{
						Content:       []byte("Age: {{ age }} {% if age > 20 %}You are 20 year old{% endif %}"),
						ContentType:   commons.TEXT_CONTENT.String(),
						ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
					},
				},
			},
			argument: map[string]interface{}{
				"name": "Alice",
				"age":  30,
			},
			expected: &types.Message{
				Contents: []*types.Content{
					{
						Content:       []byte("Hello, Alice!"),
						ContentType:   commons.TEXT_CONTENT.String(),
						ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
					},
					{
						Content:       []byte("Age: 30 You are 20 year old"),
						ContentType:   commons.TEXT_CONTENT.String(),
						ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.template, tt.argument)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPongo2StringTemplateParser_Parse(t *testing.T) {
	tests := []struct {
		name     string
		template string
		argument map[string]interface{}
		expected string
	}{
		{
			name:     "Simple template",
			template: "Hello, {{ name }}!",
			argument: map[string]interface{}{"name": "World"},
			expected: "Hello, World!",
		},
		{
			name:     "Template with multiple variables",
			template: "{{ greeting }} {{ name }}! You are {{ age }} years old.",
			argument: map[string]interface{}{
				"greeting": "Hi",
				"name":     "Alice",
				"age":      30,
			},
			expected: "Hi Alice! You are 30 years old.",
		},
	}

	parser := NewPongo2StringTemplateParser(commons.NewApplicationLogger())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.template, tt.argument)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPongo2TemplateParser_Parse(t *testing.T) {
	tests := []struct {
		name     string
		template string
		argument map[string]interface{}
		expected string
	}{

		{
			name: "Coaching session template with multiple messages",
			template: `IMPORTANT: If the message does not clearly relate to any of the growth areas listed above, set "primary_area" to "None" and explain why in the reasoning.
Consider the conversation context when determining the growth area - sometimes the meaning is clearer with context.
Output ONLY a JSON object with this format:
{
"growth_area_analysis": {
"primary_area": "THE EXACT NAME OF ONE OF THE GROWTH AREAS LISTED ABOVE OR 'None'",
"reasoning": "Brief explanation of your analysis that considers both the message and context"
}
}
<COACHING_SESSION>
{% for message in messages %}
{{ message.role }}: {{ message.text }}
{% endfor %}
</COACHING_SESSION>`,
			argument: map[string]interface{}{
				"messages": []map[string]string{
					{
						"text": "Hello how are you doing?",
						"role": "user",
					},
					{
						"text": "Hello Akshay! It's great to meet you! I'm doing well, thank you for asking. How are you doing today? Is there anything I can help you with to make your day a bit easier?",
						"role": "assistant",
					},
					{
						"text": "I wanted to discuss about change in my job",
						"role": "user",
					},
					{
						"text": "Of course, Akshay! Job changes can be a significant topic to think through. I'm here to listen and help however I can.\nAre you considering making a change from your current job, or are you dealing with changes happening within your current role? Feel free to share what's on your mind - whether it's about new opportunities you're exploring, challenges you're facing, or just needing someone to bounce ideas off of.\nWhat aspects of this job change would you like to discuss?",
						"role": "assistant",
					},
					{
						"text": "It's significant they are expecting me to do alot of thing with so less money",
						"role": "user",
					},
					{
						"text": "I hear you, Akshay. That's a really frustrating situation to be in - when the workload and expectations keep increasing but the compensation doesn't match the effort and value you're bringing to the table. It's one of the most common yet challenging workplace issues.\nA few thoughts that might help:\n1. **Document everything** - Keep track of all the additional responsibilities you've taken on. This will be valuable if you decide to negotiate for better pay or look elsewhere.\n2. **Consider having a conversation** - If you haven't already, it might be worth scheduling a meeting with your manager to discuss your expanded role and compensation. Come prepared with specific examples of your increased responsibilities.\n3. **Know your worth** - Research what others in similar roles with your responsibilities are earning. This gives you leverage whether negotiating or job hunting.\n4. **Set boundaries** - While looking for solutions, try to maintain work-life balance and avoid burnout.\nHow long has this been going on, and have you had any conversations with your management about adjusting your compensation to match your responsibilities?",
						"role": "assistant",
					},
					{
						"text": "Thank",
						"role": "user",
					},
					{
						"text": "You're welcome, Akshay! I know it's not an easy situation to deal with. If you want to continue talking about it or need help with anything else - whether it's preparing for that conversation with your manager, updating your resume, or just venting more about the situation - I'm here for you.\nTake care of yourself, and remember that recognizing your worth is the first step toward positive change. Feel free to reach out anytime you need support or advice!",
						"role": "assistant",
					},
				},
			},
			expected: `IMPORTANT: If the message does not clearly relate to any of the growth areas listed above, set "primary_area" to "None" and explain why in the reasoning.
Consider the conversation context when determining the growth area - sometimes the meaning is clearer with context.
Output ONLY a JSON object with this format:
{
"growth_area_analysis": {
"primary_area": "THE EXACT NAME OF ONE OF THE GROWTH AREAS LISTED ABOVE OR 'None'",
"reasoning": "Brief explanation of your analysis that considers both the message and context"
}
}
<COACHING_SESSION>
user: Hello how are you doing?
assistant: Hello Akshay! It's great to meet you! I'm doing well, thank you for asking. How are you doing today? Is there anything I can help you with to make your day a bit easier?
user: I wanted to discuss about change in my job
assistant: Of course, Akshay! Job changes can be a significant topic to think through. I'm here to listen and help however I can.
Are you considering making a change from your current job, or are you dealing with changes happening within your current role? Feel free to share what's on your mind - whether it's about new opportunities you're exploring, challenges you're facing, or just needing someone to bounce ideas off of.
What aspects of this job change would you like to discuss?
user: It's significant they are expecting me to do alot of thing with so less money
assistant: I hear you, Akshay. That's a really frustrating situation to be in - when the workload and expectations keep increasing but the compensation doesn't match the effort and value you're bringing to the table. It's one of the most common yet challenging workplace issues.
A few thoughts that might help:
1. **Document everything** - Keep track of all the additional responsibilities you've taken on. This will be valuable if you decide to negotiate for better pay or look elsewhere.
2. **Consider having a conversation** - If you haven't already, it might be worth scheduling a meeting with your manager to discuss your expanded role and compensation. Come prepared with specific examples of your increased responsibilities.
3. **Know your worth** - Research what others in similar roles with your responsibilities are earning. This gives you leverage whether negotiating or job hunting.
4. **Set boundaries** - While looking for solutions, try to maintain work-life balance and avoid burnout.
How long has this been going on, and have you had any conversations with your management about adjusting your compensation to match your responsibilities?
user: Thank
assistant: You're welcome, Akshay! I know it's not an easy situation to deal with. If you want to continue talking about it or need help with anything else - whether it's preparing for that conversation with your manager, updating your resume, or just venting more about the situation - I'm here for you.
Take care of yourself, and remember that recognizing your worth is the first step toward positive change. Feel free to reach out anytime you need support or advice!
</COACHING_SESSION>`,
		},
	}

	parser := NewPongo2StringTemplateParser(commons.NewApplicationLogger())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.template, tt.argument)
			assert.Equal(t, tt.expected, result)
		})
	}
}
