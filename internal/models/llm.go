package models

// ChatCompletionRequest represents a request to the \/api\/chat\/completions endpoint.
type ChatCompletionRequest struct {
	Model            string              `json:"model"`
	Messages         []ChatMessage       `json:"messages"`
	Stream           bool                `json:"stream,omitempty"`
	Temperature      *float32            `json:"temperature,omitempty"`
	TopP             *float32            `json:"top_p,omitempty"`
	MaxTokens        *int                `json:"max_tokens,omitempty"`
	Stop             []string            `json:"stop,omitempty"`
	FrequencyPenalty *float32            `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float32            `json:"presence_penalty,omitempty"`
	Tools            []ChatTool          `json:"tools,omitempty"`
	ToolChoice       any                 `json:"tool_choice,omitempty"`
	ResponseFormat   *ResponseFormatSpec `json:"response_format,omitempty"`
	User             string              `json:"user,omitempty"`
} // @name ChatCompletionRequest

// ChatMessage represents a message in a chat conversation.
type ChatMessage struct {
	Role      string            `json:"role"`
	Content   string            `json:"content,omitempty"`
	Name      string            `json:"name,omitempty"`
	ToolCalls []ToolCall        `json:"tool_calls,omitempty"`
	ToolID    string            `json:"tool_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
} // @name ChatMessage

// ChatTool represents a callable tool (function) definition.
type ChatTool struct {
	Type     string         `json:"type"`
	Function ToolDefinition `json:"function"`
} // @name ChatTool

// ToolDefinition describes the schema for a function/tool.
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
} // @name ToolDefinition

// ToolCall represents a tool invocation request.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
} // @name ToolCall

// ToolCallFunction defines the details of a function call by the assistant.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
} // @name ToolCallFunction

// ResponseFormatSpec controls structured output.
type ResponseFormatSpec struct {
	Type string `json:"type"`
} // @name ResponseFormatSpec

// ChatCompletionResponse represents the full response to a chat completion request.
type ChatCompletionResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	SystemFingerprint string                 `json:"system_fingerprint"`
	Choices           []ChatCompletionChoice `json:"choices"`
	Usage             *ChatCompletionUsage   `json:"usage,omitempty"`
} // @name ChatCompletionResponse

// ChatCompletionChoice represents a single response choice.
type ChatCompletionChoice struct {
	Index        int                     `json:"index"`
	Message      ChatMessage             `json:"message"`
	FinishReason string                  `json:"finish_reason"`
	Delta        *ChatMessage            `json:"delta,omitempty"`
	LogProbs     *ChatCompletionLogProbs `json:"logprobs,omitempty"`
} // @name ChatCompletionChoice

// ChatCompletionLogProbs provides token logprob data (optional).
type ChatCompletionLogProbs struct {
	Content []LogProb `json:"content"`
} // @name ChatCompletionLogProbs

// LogProb represents the log probability for a token.
type LogProb struct {
	Token       string  `json:"token"`
	LogProb     float64 `json:"logprob"`
	TopLogProbs []struct {
		Token   string  `json:"token"`
		LogProb float64 `json:"logprob"`
	} `json:"top_logprobs"`
} // @name LogProb

// ChatCompletionUsage shows token usage statistics.
type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
} // @name ChatCompletionUsage

// CompletionRequest represents a request to the /api/chat/completions endpoint.
type CompletionRequest struct {
	Model            string   `json:"model"`
	Prompt           any      `json:"prompt"` // string or []string
	Suffix           string   `json:"suffix,omitempty"`
	MaxTokens        *int     `json:"max_tokens,omitempty"`
	Temperature      *float32 `json:"temperature,omitempty"`
	TopP             *float32 `json:"top_p,omitempty"`
	N                *int     `json:"n,omitempty"`
	Stream           bool     `json:"stream,omitempty"`
	LogProbs         *int     `json:"logprobs,omitempty"`
	Echo             bool     `json:"echo,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	PresencePenalty  *float32 `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32 `json:"frequency_penalty,omitempty"`
	BestOf           *int     `json:"best_of,omitempty"`
	User             string   `json:"user,omitempty"`
} // @name CompletionRequest

// CompletionResponse represents a response from the /api/chat/completions endpoint.
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   *CompletionUsage   `json:"usage,omitempty"`
} // @name CompletionResponse

// CompletionChoice represents a single generated completion.
type CompletionChoice struct {
	Text         string              `json:"text"`
	Index        int                 `json:"index"`
	LogProbs     *CompletionLogProbs `json:"logprobs,omitempty"`
	FinishReason string              `json:"finish_reason"`
} // @name CompletionChoice

// CompletionLogProbs contains token log probabilities.
type CompletionLogProbs struct {
	Tokens        []string             `json:"tokens"`
	TokenLogProbs []float64            `json:"token_logprobs"`
	TopLogProbs   []map[string]float64 `json:"top_logprobs"`
	TextOffset    []int                `json:"text_offset"`
} // @name CompletionLogProbs

// CompletionUsage reports token usage.
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
} // @name CompletionUsage

// EmbeddingsRequest represents a request to the /api/embeddings endpoint.
type EmbeddingsRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"` // string or []string
	User  string `json:"user,omitempty"`
} // @name EmbeddingsRequest

// EmbeddingsResponse represents a response from the /api/embeddings endpoint.
type EmbeddingsResponse struct {
	Object string           `json:"object"`
	Model  string           `json:"model"`
	Data   []EmbeddingData  `json:"data"`
	Usage  *EmbeddingsUsage `json:"usage,omitempty"`
} // @name EmbeddingsResponse

// EmbeddingData contains a single embedding vector and its metadata.
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
} // @name EmbeddingData

// EmbeddingsUsage reports token usage.
type EmbeddingsUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
} // @name EmbeddingsUsage
