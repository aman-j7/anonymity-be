package questions

import (
	urls "anonymity/constants"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

const (
	modelName = "deepseek/deepseek-chat"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ResponseBody struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type AIResponse struct {
	Questions []string `json:"questions"`
}

type OpenRouter struct {
	apiKey string
}

func InitOpenRouter(apiKey string) *OpenRouter {
	return &OpenRouter{
		apiKey: apiKey,
	}
}

func (service *OpenRouter) GenerateTemplatesByGenre(genre string, count int) ([]string, error) {
	if service.apiKey == "" {
		return nil, fmt.Errorf("missing OPEN_ROUTER_API_KEY")
	}

	prompt := service.buildPrompt(genre, count)

	raw, err := service.callOpenRouter(context.Background(), prompt, service.apiKey)
	if err != nil {
		return nil, err
	}

	questions, err := service.parseAIResponse(raw)
	if err != nil {
		return nil, err
	}

	questions = service.filterValidQuestions(questions)

	if len(questions) == 0 {
		return nil, fmt.Errorf("no valid templates generated")
	}

	return questions, nil
}

func (service *OpenRouter) buildPrompt(genre string, count int) string {
	return fmt.Sprintf(`
		Generate %d funny party game question templates for the genre: "%s".

		Rules:
		- Use placeholder {player}
		- Keep questions short and humorous
		- No explanations
		- Return ONLY valid JSON

		Format:
		{
		"questions": ["...", "..."]
		}
		`,
		count, genre)
}

func (service *OpenRouter) callOpenRouter(ctx context.Context, prompt string, apiKey string) (string, error) {
	reqBody := RequestBody{
		Model: modelName,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", urls.OpenRouterURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost")
	req.Header.Set("X-Title", "Template Generator")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openrouter error: %s", string(respBytes))
	}

	var result ResponseBody
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	return result.Choices[0].Message.Content, nil
}

func (service *OpenRouter) parseAIResponse(raw string) ([]string, error) {
	jsonStr := service.extractJSON(raw)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var obj AIResponse
	if err := json.Unmarshal([]byte(jsonStr), &obj); err == nil && len(obj.Questions) > 0 {
		return obj.Questions, nil
	}

	var arr []string
	if err := json.Unmarshal([]byte(jsonStr), &arr); err == nil {
		return arr, nil
	}

	return nil, fmt.Errorf("failed to parse AI response")
}

func (service *OpenRouter) extractJSON(raw string) string {

	obj := regexp.MustCompile(`\{[\s\S]*\}`).FindString(raw)
	if obj != "" {
		return obj
	}

	arr := regexp.MustCompile(`\[[\s\S]*\]`).FindString(raw)
	return arr
}

func (service *OpenRouter) filterValidQuestions(questions []string) []string {
	valid := make([]string, 0, len(questions))

	for _, q := range questions {
		if service.containsPlayer(q) {
			valid = append(valid, q)
		}
	}

	return valid
}

func (service *OpenRouter) containsPlayer(q string) bool {
	return regexp.MustCompile(`\{player\}`).MatchString(q)
}
