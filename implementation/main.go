package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	runtime_lamnbda "github.com/aws/aws-lambda-go/lambda"
	"github.com/kr/pretty"
	"github.com/omeroid/chat-gpt/pkg/util"
	"github.com/omeroid/chat-gpt/shared/model"
	"github.com/otiai10/openaigo"
	"github.com/slack-go/slack"
)

func main() {
	ctx := context.Background()
	runtime_lamnbda.StartWithContext(ctx, Handler)
}

type LambdaResponse struct {
	Message string `json:"Answer:"`
}

func Handler(ctx context.Context, request model.ChatGPTFunctionInput) (res LambdaResponse, err error) {
	var message string
	if 0 < len(request.ImagePrompt) {
		if message, err = ChatGPTImage(ctx, request.ImagePrompt); err != nil {
			return LambdaResponse{}, SendSlack(request.SlackChannelID, request.SlackTs, util.IgnoreText(err.Error()))
		}
	} else {
		if message, err = ChatGPT3_5Turbo(ctx, request.SystemPrompt, request.Conversation); err != nil {
			return LambdaResponse{}, SendSlack(request.SlackChannelID, request.SlackTs, util.IgnoreText(err.Error()))
		}
	}

	if err := SendSlack(request.SlackChannelID, request.SlackTs, message); err != nil {
		return LambdaResponse{}, err
	}
	return LambdaResponse{}, nil
}

func SendSlack(channel, ts, body string) error {
	slackToken := os.Getenv("SLACK_TOKEN")
	conn := slack.New(slackToken)
	_, _, err := conn.PostMessage(channel,
		slack.MsgOptionText(body, true),
		slack.MsgOptionTS(ts),
	)
	if err != nil {
		panic(err)
	}
	return nil
}

func ChatGPT3_5Turbo(ctx context.Context, systemComment string, conversations []model.Conversation) (re string, err error) {
	var messages []openaigo.ChatMessage
	if 0 < len(systemComment) {
		messages = []openaigo.ChatMessage{
			{
				Role:    model.SystemRole,
				Content: systemComment,
			},
		}
	}
	for _, conversation := range conversations {
		messages = append(messages, openaigo.ChatMessage{
			Role:    conversation.Role,
			Content: conversation.Text,
		})
	}

	// GPT-3のAPIを使うためのクライアントを作成する
	openAIAPI := os.Getenv("OPENAI_API")
	client := openaigo.NewClient(openAIAPI)
	request := openaigo.ChatCompletionRequestBody{
		Model:     model.ModelGPT3_5_Turbo,
		MaxTokens: 2048,
		Messages:  messages,
		User:      "test",
	}
	response, err := client.Chat(ctx, request)
	if err != nil {
		return "", err
	}

	var msgs []string
	for _, choice := range response.Choices {
		msgs = append(msgs, choice.Message.Content)
		// 最初のMessageのみ回答する
		break
	}
	return strings.Join(msgs, "\n"), nil
}

func ChatGPTImage(ctx context.Context, prompt string) (re string, err error) {
	openAIAPI := os.Getenv("OPENAI_API")
	client := openaigo.NewClient(openAIAPI)
	request := openaigo.ImageGenerationRequestBody{
		Prompt:         prompt,
		N:              1,
		Size:           "1024x1024",
		ResponseFormat: "url",
		User:           "test",
	}
	response, err := client.CreateImage(ctx, request)
	if err != nil {
		return "", err
	}

	var msgs []string
	for _, data := range response.Data {
		msgs = append(msgs, data.URL)
	}
	return strings.Join(msgs, "\n"), nil
}

func ChatGPTModeration(ctx context.Context, prompt string) (re string, err error) {
	openAIAPI := os.Getenv("OPENAI_API")
	client := openaigo.NewClient(openAIAPI)
	request := openaigo.ModerationCreateRequestBody{
		Input: prompt,
		Model: "text-moderation-latest",
	}
	response, err := client.CreateModeration(ctx, request)
	if err != nil {
		return "", err
	}

	pretty.Print(response)
	var msgs []string
	for _, data := range response.Results {
		hate := fmt.Sprintf("Hate(人種，民族等に対する憎悪): %f", data.CategoryScores.Hate)
		if data.Categories.Hate {
			hate = fmt.Sprintf("**%s**", hate)
		}

		hateThreatening := fmt.Sprintf("HateThreatening(いやらがせとかいじめの発言): %f", data.CategoryScores.HateThreatening)
		if data.Categories.HateThreatening {
			hateThreatening = fmt.Sprintf("**%s**", hateThreatening)
		}

		selfHarm := fmt.Sprintf("SelfHarm(自傷行為): %f", data.CategoryScores.SelfHarm)
		if data.Categories.SelfHarm {
			selfHarm = fmt.Sprintf("**%s**", selfHarm)
		}

		sexual := fmt.Sprintf("Sexual(エロ): %f", data.CategoryScores.Sexual)
		if data.Categories.Sexual {
			sexual = fmt.Sprintf("**%s**", sexual)
		}

		sexualMinors := fmt.Sprintf("SexualMinors(未成年のエロ): %f", data.CategoryScores.SexualMinors)
		if data.Categories.SexualMinors {
			sexualMinors = fmt.Sprintf("**%s**", sexualMinors)
		}

		violence := fmt.Sprintf("Violence(暴力): %f", data.CategoryScores.Violence)
		if data.Categories.Violence {
			violence = fmt.Sprintf("**%s**", violence)
		}

		violenceGraphic := fmt.Sprintf("ViolenceGraphic(欠損やフェイタリティ): %f", data.CategoryScores.ViolenceGraphic)
		if data.Categories.ViolenceGraphic {
			violenceGraphic = fmt.Sprintf("**%s**", violenceGraphic)
		}
		msgs = append(msgs, strings.Join([]string{hate, hateThreatening, selfHarm, sexual, sexualMinors, violence, violenceGraphic}, "\n"))
	}
	return strings.Join(msgs, "\n\n"), nil
}
