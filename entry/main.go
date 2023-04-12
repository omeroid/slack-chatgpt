package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	runtime_lamnbda "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/kr/pretty"
	"github.com/omeroid/chat-gpt/pkg/util"
	"github.com/omeroid/chat-gpt/shared/model"
	"github.com/slack-go/slack"
)

func main() {
	ctx := context.Background()
	runtime_lamnbda.StartWithContext(ctx, Handler)
}

var (
	mentionRegexp       = regexp.MustCompile(`<@.*>`)
	systemCommentRegexp = regexp.MustCompile(`^あなたは.*です$`)
	imageCommentRegexp  = regexp.MustCompile(`\[image\]`)
	ignoreCommentRegexp = regexp.MustCompile(`\[ignore\]`)
)

type LambdaResponse struct {
	Message string `json:"Answer:"`
}

type SlackEvent struct {
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
	Type      string `json:"type"`
	TeamID    string `json:"team_id"`
	APIAppID  string `json:"api_app_id"`
	Event     struct {
		ClientMsgID string `json:"client_msg_id"`
		Type        string `json:"type"`
		Text        string `json:"text"`
		User        string `json:"user"`
		Ts          string `json:"ts"`
		ThreadTs    string `json:"thread_ts"`
		Team        string `json:"team"`
		Channel     string `json:"channel"`
		EventTs     string `json:"event_ts"`
	} `json:"event"`
	EventID        string `json:"event_id"`
	EventTime      int    `json:"event_time"`
	Authorizations []struct {
		EnterpriseID        interface{} `json:"enterprise_id"`
		TeamID              string      `json:"team_id"`
		UserID              string      `json:"user_id"`
		IsBot               bool        `json:"is_bot"`
		IsEnterpriseInstall bool        `json:"is_enterprise_install"`
	} `json:"authorizations"`
	IsExtSharedChannel bool   `json:"is_ext_shared_channel"`
	EventContext       string `json:"event_context"`
}

func Handler(ctx context.Context, request events.LambdaFunctionURLRequest) (res LambdaResponse, err error) {
	// https://dev.classmethod.jp/articles/slack-resend-matome/
	// slack eventはレスポンスが遅いと再送されるので、再送を無視する
	if _, ok := request.Headers["x-slack-retry-num"]; ok {
		return LambdaResponse{}, nil
	}
	var slackBotID string
	var chatGPTFunctionInput model.ChatGPTFunctionInput
	// interactive の場合
	if request.IsBase64Encoded {
		var body []byte
		if body, err = base64.StdEncoding.DecodeString(request.Body); err != nil {
			return LambdaResponse{}, err
		}
		var payload string
		if payload, err = url.QueryUnescape(string(body)); err != nil {
			return LambdaResponse{}, err
		}
		payload = strings.Replace(payload, "payload=", "", 1)
		pretty.Printf("%+v", payload)
		var interactive slack.InteractionCallback
		if err = json.Unmarshal([]byte(payload), &interactive); err != nil {
			return LambdaResponse{}, err
		}

		if imageCommentRegexp.MatchString(strings.TrimSpace(RemoveMention(interactive.Message.Text))) {
			chatGPTFunctionInput = model.ChatGPTFunctionInput{
				ImagePrompt:    imageCommentRegexp.ReplaceAllString(strings.TrimSpace(RemoveMention(interactive.Message.Text)), ""),
				SlackChannelID: interactive.Channel.ID,
				SlackTs:        interactive.MessageTs,
			}
		}

		// あなたは***ですというコメントが最初にあれば、3.5-turboを利用する
		if systemCommentRegexp.MatchString(strings.TrimSpace(RemoveMention(interactive.Message.Text))) {
			if err = SendSlack(interactive.Channel.ID, interactive.MessageTs, util.IgnoreText("はい。メンションをつけて質問をどうぞ")); err != nil {
				return LambdaResponse{}, err
			}
			return LambdaResponse{}, err
		} else {
			chatGPTFunctionInput = model.ChatGPTFunctionInput{
				Conversation: []model.Conversation{
					{
						Role: model.UserRole,
						Text: interactive.Message.Text,
					},
				},
				SlackChannelID: interactive.Channel.ID,
				SlackTs:        interactive.MessageTs,
			}
		}
	} else {
		pretty.Printf("%+v", request.Body)
		var event SlackEvent
		if err = json.Unmarshal([]byte(request.Body), &event); err != nil {
			return LambdaResponse{}, err
		}

		if slackBotID, err = GetBotUserID(ctx); err != nil {
			return LambdaResponse{}, SendSlack(event.Event.Channel, event.Event.Ts, util.IgnoreText(err.Error()))
		}

		switch event.Type {
		case "url_verification":
			return LambdaResponse{Message: event.Challenge}, nil
		case "event_callback":
			// botへの話しかけでなければスルー
			if !strings.Contains(event.Event.Text, slackBotID) {
				return LambdaResponse{}, nil
			}

			prompt := RemoveMention(event.Event.Text)
			if len(prompt) == 0 {
				return LambdaResponse{}, SendSlack(event.Event.Channel, event.Event.Ts, util.IgnoreText("ChatGPTに何か質問してみてください。\n最大10回の会話で回答します。(ただし回答の文字数に制限があります)\n回答が途中で途切れた時は「続きを教えて」と書いてみてください。\n[image]と最初につけて指示をすると、画像ファイルの生成ができます"))
			}
			var conversations []model.Conversation
			if 0 < len(event.Event.ThreadTs) {
				if conversations, err = GetThreadConversations(event.Event.Channel, event.Event.ThreadTs, slackBotID); err != nil {
					return LambdaResponse{}, SendSlack(event.Event.Channel, event.Event.Ts, util.IgnoreText(err.Error()))
				}

				chatGPTFunctionInput = model.ChatGPTFunctionInput{
					Conversation:   conversations,
					SlackChannelID: event.Event.Channel,
					SlackTs:        event.Event.Ts,
				}
			} else {
				if imageCommentRegexp.MatchString(prompt) {
					chatGPTFunctionInput = model.ChatGPTFunctionInput{
						ImagePrompt:    imageCommentRegexp.ReplaceAllString(prompt, ""),
						SlackChannelID: event.Event.Channel,
						SlackTs:        event.Event.Ts,
					}
				} else if systemCommentRegexp.MatchString(prompt) {
					if err = SendSlack(event.Event.Channel, event.Event.Ts, util.IgnoreText("はい。メンションをつけて質問をどうぞ")); err != nil {
						return LambdaResponse{}, err
					}
					return LambdaResponse{}, err
				} else {
					conversations = append(conversations, model.Conversation{
						Role: model.UserRole,
						Text: prompt,
					})
					chatGPTFunctionInput = model.ChatGPTFunctionInput{
						Conversation:   conversations,
						SlackChannelID: event.Event.Channel,
						SlackTs:        event.Event.Ts,
					}
				}
			}
		}

	}
	if err = InvokeChatGPTFunction(ctx, chatGPTFunctionInput); err != nil {
		return LambdaResponse{}, err
	}
	return LambdaResponse{}, nil
}

func InvokeChatGPTFunction(ctx context.Context, input model.ChatGPTFunctionInput) (err error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	var inputBytes []byte
	if inputBytes, err = json.Marshal(input); err != nil {
		return err
	}

	client := lambda.NewFromConfig(cfg)
	if _, err = client.InvokeAsync(ctx, &lambda.InvokeAsyncInput{
		FunctionName: util.ToPtr("ChatGPTFunction"),
		InvokeArgs:   bytes.NewReader(inputBytes),
	}); err != nil {
		return err
	}
	return nil
}

func GetThreadConversations(channel, threadTs, botUserID string) (conversations []model.Conversation, err error) {
	slackToken := os.Getenv("SLACK_TOKEN")
	conn := slack.New(slackToken)
	var msgs []slack.Message
	var hasMore bool
	if msgs, hasMore, _, err = conn.GetConversationReplies(&slack.GetConversationRepliesParameters{
		ChannelID: channel,
		Timestamp: threadTs,
		Inclusive: true,
		Limit:     20,
	}); err != nil {
		return nil, err
	}

	if hasMore {
		return nil, fmt.Errorf("20以上の会話を続けることはできません。新しいThreadで会話をしてください")
	}

	for i, msg := range msgs {
		pretty.Printf("%+v", msg)
		prompt := RemoveMention(msg.Text)

		if msg.User != botUserID && !strings.Contains(msg.Msg.Text, botUserID) {
			continue
		}

		// [ignore]という文字列が含まれている場合は無視する
		if ignoreCommentRegexp.MatchString(prompt) || imageCommentRegexp.MatchString(prompt) {
			continue
		}

		// 最初のメッセージが「あなたは〇〇です」というメッセージであれば、それをSystemとして追加する
		if i == 0 && systemCommentRegexp.MatchString(prompt) {
			conversations = append(conversations, model.Conversation{
				Role: model.SystemRole,
				Text: prompt,
			})
		}

		if msg.User == botUserID {
			conversations = append(conversations, model.Conversation{
				Role: model.AssistantRole,
				Text: prompt,
			})
			continue
		}

		if strings.Contains(msg.Msg.Text, botUserID) {
			conversations = append(conversations, model.Conversation{
				Role: model.UserRole,
				Text: prompt,
			})
			continue
		}
	}
	return conversations, nil
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

func GetBotUserID(ctx context.Context) (botUserID string, err error) {
	slackToken := os.Getenv("SLACK_TOKEN")
	conn := slack.New(slackToken)

	authTestResponse, err := conn.AuthTestContext(ctx)
	if err != nil {
		return "", err
	}
	return authTestResponse.UserID, nil
}

func RemoveMention(s string) string {
	return strings.TrimSpace(mentionRegexp.ReplaceAllString(s, ""))
}
