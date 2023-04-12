package model

var (
	UserRole      = "user"
	SystemRole    = "system"
	AssistantRole = "assistant"
)

var (
	ModelGPT3_5_Turbo = "gpt-3.5-turbo"
	ModelTextDavinci  = "text-davinci-003"
)

type Conversation struct {
	Role string
	Text string
}

type ChatGPTFunctionInput struct {
	SystemPrompt   string         `json:"system_prompt"`
	Conversation   []Conversation `json:"conversation"`
	SlackChannelID string         `json:"slack_channel_id"`
	SlackTs        string         `json:"slack_ts"`
	ImagePrompt    string         `json:"image_prompt"`
}
