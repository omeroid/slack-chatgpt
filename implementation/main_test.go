package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/omeroid/chat-gpt/shared/model"
)

func TestChatGPT3_5Turbo(t *testing.T) {
	// テストケースを作成
	testCases := []struct {
		conversations []model.Conversation
		systemComment string
	}{
		//		{
		//			systemComment: "あなたは小学生の先生です",
		//			conversations: []model.Conversation{
		//				{
		//					Role: model.UserRole,
		//					Text: `地球とはなんですか？`,
		//				},
		//			},
		//		},
		//		{
		//			conversations: []model.Conversation{
		//				{
		//					Role: model.UserRole,
		//					Text: `地球とはなんですか？`,
		//				},
		//				{
		//					Role: model.AssistantRole,
		//					Text: `地球は、太陽系の惑星の1つであり、自転と公転によって空間を移動しています。地球は、岩石質の地殻、マントル、核を持つ岩石惑星であり、生命の存在が確認されている唯一の惑星です。地球は、大気、水、生物、地形、気候、気候変動など、多くの自然現象に影響されています。`,
		//				},
		//				{
		//					Role: model.UserRole,
		//					Text: `その影響とはなんですか？`,
		//				},
		//			},
		//		},
	}

	// テストケースを実行
	ctx := context.Background()
	for _, tc := range testCases {
		// テストケースの実行
		re, err := ChatGPT3_5Turbo(ctx, tc.systemComment, tc.conversations)
		if err != nil {
			t.Fatalf("failed test %#v", err)
		}

		fmt.Println(re)
	}
}

func TestChatGPTImage(t *testing.T) {
	// テストケースを作成
	testCases := []struct {
		prompt string
	}{
		// {
		// 	prompt: `PC好きの若者の1人部屋の室内`,
		// },
		// {
		// 	prompt: `パソコンを見て悩んでいる人`,
		// },
	}

	// テストケースを実行
	ctx := context.Background()
	for _, tc := range testCases {
		// テストケースの実行
		re, err := ChatGPTImage(ctx, tc.prompt)
		if err != nil {
			t.Fatalf("failed test %#v", err)
		}

		fmt.Println(re)
	}
}

func TestChatGPTModeration(t *testing.T) {
	// テストケースを作成
	testCases := []struct {
		prompt string
	}{
		{
			prompt: `I will kill you`,
		},
		{
			prompt: `You are cute`,
		},
	}

	// テストケースを実行
	ctx := context.Background()
	for _, tc := range testCases {
		// テストケースの実行
		re, err := ChatGPTModeration(ctx, tc.prompt)
		if err != nil {
			t.Fatalf("failed test %#v", err)
		}

		fmt.Println(re)
	}
}
