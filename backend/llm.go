package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net"
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
	Pane        string `json:"pane"`
	Action      string `json:"action"`
	Value       string `json:"value"`
	Done        bool   `json:"done"`
}

type AgentPlan struct {
	Issue    string   `json:"issue"`
	MenuPath string   `json:"menu_path"`
	Verify   string   `json:"verify"`
	Summary  string   `json:"summary"`
	Steps    []string `json:"steps"`
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
		detail = "low"
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

func (c *LLMClient) Plan(ctx context.Context, goal string) (*AgentPlan, error) {
	if err := c.Ready(); err != nil {
		return nil, err
	}
	body := map[string]any{
		"model": c.model,
		"instructions": `You are the planner for a keyboard-only IP-KVM agent on a Lenovo ThinkPad T14s Gen 6 (AMD) UEFI BIOS.
The operator goal is messy natural language. Your job is NOT to guess keys from the wording alone.
1. Diagnose: map the goal to a BIOS *issue* using the PAGE ENCYCLOPEDIA (which tab, which submenu, why).
2. Pick menu_path from real pages (Main / Config / Date/Time / Security / Setup=Startup / Restart).
3. Emit HID steps the executor will follow one key at a time.

Rules:
- Use ONLY menus/submenus in the encyclopedia + live overlay. Do not invent top-level tabs.
- Live HDMI labels this unit's boot tab "Setup"; docs say "Startup". Same page.
- PRIMARY DEMO: anything about boot drive, boot order, Windows driver, Windows Boot Manager, which disk boots first → issue is boot priority. menu_path MUST be Setup → Boot. Follow the PRIMARY DEMO recipe. Never Config.
- USB installer won't boot → still Setup (UEFI/Legacy, Boot) and maybe Security → Secure Boot — say so in issue.
- Fan/USB charge/display → Config (Power / USB / Display).
- Passwords, TPM, Secure Boot, camera missing → Security.
- Left-nav travel is one UP/DOWN per step. The executor HID layer sends ENTER after each navbar UP/DOWN so the blue fill (open tab) moves. Do NOT list ENTER after every DOWN.
- Use ENTER only to open a right-pane submenu (e.g. Boot) or a dialog.
- verify: filled blue background on the target left-nav tab (not a thin hover border).`,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": t14sPlannerContext() + "\n\nOperator goal:\n" + goal,
					},
				},
			},
		},
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "agent_plan",
				"strict": true,
				"schema": agentPlanSchema(),
			},
		},
	}
	raw, err := c.postJSON(ctx, "https://api.openai.com/v1/responses", body)
	if err != nil {
		return nil, err
	}
	text, err := extractOutputText(raw)
	if err != nil {
		return nil, err
	}
	var plan AgentPlan
	if err := json.Unmarshal([]byte(text), &plan); err != nil {
		return nil, fmt.Errorf("parse plan JSON: %w", err)
	}
	return &plan, nil
}

func agentPlanSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"issue", "menu_path", "verify", "summary", "steps"},
		"properties": map[string]any{
			"issue": map[string]any{
				"type":        "string",
				"description": "BIOS problem in one or two sentences, citing the encyclopedia page (e.g. boot priority lives under Startup/Setup → Boot, not Config).",
			},
			"menu_path": map[string]any{
				"type":        "string",
				"description": "Exact path, e.g. Setup → Boot.",
			},
			"verify": map[string]any{
				"type":        "string",
				"description": "What the screen must show when the goal is complete.",
			},
			"summary": map[string]any{
				"type":        "string",
				"description": "Short navigation plan for the executor.",
			},
			"steps": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "Ordered HID steps.",
			},
		},
	}
}

func (c *LLMClient) Decide(ctx context.Context, goal string, jpeg []byte, history []AgentDecision, plan *AgentPlan) (*AgentDecision, []byte, error) {
	if err := c.Ready(); err != nil {
		return nil, nil, err
	}

	goalText := fmt.Sprintf("Operator goal:\n%s", goal)
	if plan != nil && (plan.Summary != "" || plan.Issue != "" || len(plan.Steps) > 0) {
		goalText += "\n\nAgreed diagnosis + plan (follow this; recover with LEFT/ESC if the JPEG disagrees):\n"
		if plan.Issue != "" {
			goalText += "Issue: " + plan.Issue + "\n"
		}
		if plan.MenuPath != "" {
			goalText += "Menu path: " + plan.MenuPath + "\n"
		}
		if plan.Verify != "" {
			goalText += "Done when: " + plan.Verify + "\n"
		}
		if plan.Summary != "" {
			goalText += plan.Summary + "\n"
		}
		for i, s := range plan.Steps {
			goalText += fmt.Sprintf("%d. %s\n", i+1, s)
		}
	}
	if len(history) > 0 {
		goalText += "\n\nPrior steps this run (newest last):\n"
		for i, h := range history {
			goalText += fmt.Sprintf("%d) pane=%s action=%s value=%q done=%v — %s\n", i+1, h.Pane, h.Action, h.Value, h.Done, h.Observation)
		}
	}

	content := []map[string]any{
		{
			"type": "input_text",
			"text": goalText,
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
		"model":        c.model,
		"instructions": t14sExecutorContext(),
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
		"required":             []string{"observation", "thought", "pane", "action", "value", "done"},
		"properties": map[string]any{
			"observation": map[string]any{
				"type":        "string",
				"description": "Which left-nav tab has the filled blue background (open page). Mention a thin hover border only if visible.",
			},
			"thought": map[string]any{
				"type":        "string",
				"description": "Short reasoning for the next action.",
			},
			"pane": map[string]any{
				"type":        "string",
				"enum":        []string{"left_nav", "right_content"},
				"description": "left_nav if UP/DOWN would move the top-level tab hover; right_content if focus is in the page.",
			},
			"action": map[string]any{
				"type": "string",
				"enum": []string{"type", "key", "wait", "done"},
			},
			"value": map[string]any{
				"type":        "string",
				"description": "Text to type, key name, or empty for wait/done.",
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

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 800 * time.Millisecond):
			}
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
			req.Header.Set("Content-Type", "application/json")
		}

		res, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if !retryableNetErr(err) {
				return nil, err
			}
			continue
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			lastErr = err
			if retryableNetErr(err) {
				continue
			}
			return nil, err
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			if res.StatusCode >= 500 || res.StatusCode == http.StatusTooManyRequests {
				lastErr = fmt.Errorf("openai %s: %s", res.Status, truncate(string(body), 800))
				continue
			}
			return nil, fmt.Errorf("openai %s: %s", res.Status, truncate(string(body), 800))
		}
		return json.RawMessage(body), nil
	}
	return nil, lastErr
}

func retryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection refused")
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

const llmMaxWidth = 720
const llmJPEGQuality = 62

// compressForLLM downscales and re-encodes for the OpenAI POST only (live MJPEG unchanged).
func compressForLLM(jpegIn []byte) []byte {
	if len(jpegIn) == 0 {
		return jpegIn
	}
	img, err := jpeg.Decode(bytes.NewReader(jpegIn))
	if err != nil {
		return jpegIn
	}
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW <= 0 || srcH <= 0 {
		return jpegIn
	}
	dstW, dstH := srcW, srcH
	if dstW > llmMaxWidth {
		dstH = dstH * llmMaxWidth / dstW
		dstW = llmMaxWidth
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		sy := b.Min.Y + y*srcH/dstH
		for x := 0; x < dstW; x++ {
			sx := b.Min.X + x*srcW/dstW
			dst.Set(x, y, img.At(sx, sy))
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: llmJPEGQuality}); err != nil {
		return jpegIn
	}
	if buf.Len() == 0 || buf.Len() >= len(jpegIn) {
		return jpegIn
	}
	return buf.Bytes()
}
