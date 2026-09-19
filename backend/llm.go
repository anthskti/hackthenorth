package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultOpenAIModel = "gpt-4o"

var errNoAPIKey = errors.New("OPENAI_API_KEY is not set")

type AgentDecision struct {
	Observation string `json:"observation"`
	Thought     string `json:"thought"`
	Action      string `json:"action"`
	Value       string `json:"value"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Done        bool   `json:"done"`
}

type LLMClient struct {
	apiKey      string
	model       string
	imageDetail string
	http        *http.Client
}

func NewLLMClient() *LLMClient {
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = defaultOpenAIModel
	}
	detail := strings.TrimSpace(os.Getenv("OPENAI_IMAGE_DETAIL"))
	if detail == "" {
		detail = "auto"
	}
	return &LLMClient{
		apiKey:      strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		model:       model,
		imageDetail: detail,
		http:        &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *LLMClient) Ready() error {
	if c == nil || c.apiKey == "" {
		return errNoAPIKey
	}
	return nil
}

func (c *LLMClient) Decide(ctx context.Context, goal string, jpeg []byte) (*AgentDecision, []byte, error) {
	if err := c.Ready(); err != nil {
		return nil, nil, err
	}

	content := []map[string]any{
		{
			"type": "input_text",
			"text": fmt.Sprintf("Operator goal:\n%s", goal),
		},
	}
	if len(jpeg) > 0 {
		content = append(content, map[string]any{
			"type":      "input_image",
			"detail":    c.imageDetail,
			"image_url": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpeg),
		})
	} else {
		content = append(content, map[string]any{
			"type": "input_text",
			"text": "No screen frame is available. Infer the most likely next KVM action from the goal alone.",
		})
	}

	body := map[string]any{
		"model": c.model,
		"instructions": `You are a vision agent driving a remote machine over IP-KVM.
You receive the operator's goal and a JPEG of the current HDMI screen.
Return JSON for the single next action. Do not execute anything yourself.
Use action "key" for keyboard (value like ENTER, F2, a).
Use "mouse_click" / "mouse_move" with x,y in frame pixels.
Use "wait" when the screen is mid-transition.
Use "done" when the goal is already complete.`,
		"input": []map[string]any{
			{
				"role":    "user",
				"content": content,
			},
		},
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "agent_decision",
				"strict": true,
				"schema": agentDecisionSchema(),
			},
		},
	}

	raw, err := c.postJSON(ctx, "https://api.openai.com/v1/responses", body)
	if err != nil {
		return nil, nil, err
	}

	text, err := extractOutputText(raw)
	if err != nil {
		return nil, raw, err
	}

	var decision AgentDecision
	if err := json.Unmarshal([]byte(text), &decision); err != nil {
		return nil, []byte(text), fmt.Errorf("parse model JSON: %w", err)
	}
	pretty, _ := json.MarshalIndent(decision, "", "  ")
	return &decision, pretty, nil
}

func agentDecisionSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"observation", "thought", "action", "value", "x", "y", "done"},
		"properties": map[string]any{
			"observation": map[string]any{
				"type":        "string",
				"description": "What is visible on the current screen.",
			},
			"thought": map[string]any{
				"type":        "string",
				"description": "Short reasoning for the next action.",
			},
			"action": map[string]any{
				"type": "string",
				"enum": []string{"key", "mouse_click", "mouse_move", "wait", "done"},
			},
			"value": map[string]any{
				"type":        "string",
				"description": "Key name, mouse button, or wait hint. Empty string if unused.",
			},
			"x": map[string]any{
				"type":        "integer",
				"description": "Mouse X in frame pixels. 0 if unused.",
			},
			"y": map[string]any{
				"type":        "integer",
				"description": "Mouse Y in frame pixels. 0 if unused.",
			},
			"done": map[string]any{
				"type":        "boolean",
				"description": "True if the goal is complete and no further action is needed.",
			},
		},
	}
}

func (c *LLMClient) postJSON(ctx context.Context, url string, payload any) (json.RawMessage, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("openai %s: %s", res.Status, truncate(string(body), 800))
	}
	return json.RawMessage(body), nil
}

func extractOutputText(raw json.RawMessage) (string, error) {
	var envelope struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", err
	}
	if envelope.Error != nil && envelope.Error.Message != "" {
		return "", errors.New(envelope.Error.Message)
	}
	if text := strings.TrimSpace(envelope.OutputText); text != "" {
		return text, nil
	}
	var parts []string
	for _, item := range envelope.Output {
		for _, c := range item.Content {
			if c.Type == "output_text" || c.Type == "text" {
				if t := strings.TrimSpace(c.Text); t != "" {
					parts = append(parts, t)
				}
			}
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("openai response had no text: %s", truncate(string(raw), 800))
	}
	return strings.Join(parts, "\n"), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
