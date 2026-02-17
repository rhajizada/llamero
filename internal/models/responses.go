package models

// ResponsesCreateRequest represents a request to the /api/responses endpoint.
type ResponsesCreateRequest struct {
	Model              string                  `json:"model"`
	Input              any                     `json:"input,omitempty"`
	Instructions       string                  `json:"instructions,omitempty"`
	Tools              []ResponsesTool         `json:"tools,omitempty"`
	Stream             bool                    `json:"stream,omitempty"`
	Temperature        *float32                `json:"temperature,omitempty"`
	TopP               *float32                `json:"top_p,omitempty"`
	MaxOutputTokens    *int                    `json:"max_output_tokens,omitempty"`
	PreviousResponseID string                  `json:"previous_response_id,omitempty"`
	Conversation       any                     `json:"conversation,omitempty"`
	Truncation         string                  `json:"truncation,omitempty"`
	Include            []string                `json:"include,omitempty"`
	Text               *ResponsesTextConfig    `json:"text,omitempty"`
	Reasoning          *ResponsesReasoningSpec `json:"reasoning,omitempty"`
} // @name ResponsesCreateRequest

// ResponsesCompactRequest represents a request to the /api/responses/compact endpoint.
type ResponsesCompactRequest struct {
	Model              string `json:"model"`
	Input              any    `json:"input,omitempty"`
	Instructions       string `json:"instructions,omitempty"`
	PreviousResponseID string `json:"previous_response_id,omitempty"`
} // @name ResponsesCompactRequest

// ResponsesTool represents a function-like tool definition for Responses API.
type ResponsesTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Strict      *bool          `json:"strict,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
} // @name ResponsesTool

// ResponsesTextConfig describes output formatting for Responses API.
type ResponsesTextConfig struct {
	Format *ResponsesTextFormat `json:"format,omitempty"`
} // @name ResponsesTextConfig

// ResponsesTextFormat describes text or schema mode for Responses API.
type ResponsesTextFormat struct {
	Type   string `json:"type"`
	Name   string `json:"name,omitempty"`
	Schema any    `json:"schema,omitempty"`
	Strict *bool  `json:"strict,omitempty"`
} // @name ResponsesTextFormat

// ResponsesReasoningSpec configures reasoning behavior.
type ResponsesReasoningSpec struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
} // @name ResponsesReasoningSpec

// ResponsesResponse represents a response from /api/responses.
type ResponsesResponse struct {
	ID                 string                      `json:"id"`
	Object             string                      `json:"object"`
	CreatedAt          int64                       `json:"created_at"`
	CompletedAt        *int64                      `json:"completed_at,omitempty"`
	Status             string                      `json:"status,omitempty"`
	Model              string                      `json:"model,omitempty"`
	Output             []ResponsesOutputItem       `json:"output,omitempty"`
	Usage              *ResponsesUsage             `json:"usage,omitempty"`
	Error              *ResponsesError             `json:"error,omitempty"`
	IncompleteDetails  *ResponsesIncompleteDetails `json:"incomplete_details,omitempty"`
	Instructions       any                         `json:"instructions,omitempty"`
	PreviousResponseID *string                     `json:"previous_response_id,omitempty"`
} // @name ResponsesResponse

// ResponsesDeleteResponse represents a delete acknowledgment payload.
type ResponsesDeleteResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
} // @name ResponsesDeleteResponse

// ResponsesCompactedResponse represents a compact response payload.
type ResponsesCompactedResponse struct {
	ID        string                `json:"id"`
	Object    string                `json:"object"`
	CreatedAt int64                 `json:"created_at"`
	Output    []ResponsesOutputItem `json:"output"`
	Usage     *ResponsesUsage       `json:"usage,omitempty"`
} // @name ResponsesCompactedResponse

// ResponsesOutputItem represents one item in the response output list.
type ResponsesOutputItem struct {
	ID               string                      `json:"id,omitempty"`
	Type             string                      `json:"type"`
	Role             string                      `json:"role,omitempty"`
	Status           string                      `json:"status,omitempty"`
	Content          []ResponsesOutputContent    `json:"content,omitempty"`
	CallID           string                      `json:"call_id,omitempty"`
	Name             string                      `json:"name,omitempty"`
	Arguments        string                      `json:"arguments,omitempty"`
	Summary          []ResponsesReasoningSummary `json:"summary,omitempty"`
	EncryptedContent string                      `json:"encrypted_content,omitempty"`
} // @name ResponsesOutputItem

// ResponsesOutputContent represents output content from assistant message items.
type ResponsesOutputContent struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Annotations []any  `json:"annotations,omitempty"`
	Logprobs    []any  `json:"logprobs,omitempty"`
} // @name ResponsesOutputContent

// ResponsesReasoningSummary represents a summary snippet for reasoning output.
type ResponsesReasoningSummary struct {
	Type string `json:"type"`
	Text string `json:"text"`
} // @name ResponsesReasoningSummary

// ResponsesUsage reports token usage for responses API calls.
type ResponsesUsage struct {
	InputTokens   int                          `json:"input_tokens"`
	OutputTokens  int                          `json:"output_tokens"`
	TotalTokens   int                          `json:"total_tokens"`
	InputDetails  *ResponsesInputUsageDetails  `json:"input_tokens_details,omitempty"`
	OutputDetails *ResponsesOutputUsageDetails `json:"output_tokens_details,omitempty"`
} // @name ResponsesUsage

// ResponsesInputUsageDetails reports cached input token details.
type ResponsesInputUsageDetails struct {
	CachedTokens int `json:"cached_tokens"`
} // @name ResponsesInputUsageDetails

// ResponsesOutputUsageDetails reports output token details.
type ResponsesOutputUsageDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
} // @name ResponsesOutputUsageDetails

// ResponsesError represents an error object from responses API payloads.
type ResponsesError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
} // @name ResponsesError

// ResponsesIncompleteDetails describes why a response is incomplete.
type ResponsesIncompleteDetails struct {
	Reason string `json:"reason"`
} // @name ResponsesIncompleteDetails
