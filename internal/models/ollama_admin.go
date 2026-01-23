package models

import "time"

// BackendCreateModelRequest matches Ollama's create API payload.
type BackendCreateModelRequest struct {
	Model string `json:"model"` // Name of the model to create.

	Stream *bool `json:"stream,omitempty"`

	Quantize string `json:"quantize,omitempty"` // Quantization target, e.g. "Q4_0".
	From     string `json:"from,omitempty"`

	RemoteHost string `json:"remote_host,omitempty"`

	Files    map[string]string `json:"files,omitempty"`
	Adapters map[string]string `json:"adapters,omitempty"`

	Template string `json:"template,omitempty"`

	License any `json:"license,omitempty"`

	System     string         `json:"system,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty"`

	Messages []BackendMessage `json:"messages,omitempty"`

	Renderer string `json:"renderer,omitempty"`
	Parser   string `json:"parser,omitempty"`
	Requires string `json:"requires,omitempty"`

	Info map[string]any `json:"info,omitempty"`

	Name         string `json:"name"`
	Quantization string `json:"quantization,omitempty"`
} // @name BackendCreateModelRequest

// BackendImageData represents raw image bytes (base64-encoded in JSON).
type BackendImageData []byte

// BackendToolCall describes a tool call associated with a message.
type BackendToolCall struct {
	ID       string                  `json:"id,omitempty"`
	Function BackendToolCallFunction `json:"function"`
} // @name BackendToolCall

// BackendToolCallFunction describes a callable tool in a tool call.
type BackendToolCallFunction struct {
	Index     int                              `json:"index"`
	Name      string                           `json:"name"`
	Arguments BackendToolCallFunctionArguments `json:"arguments"`
} // @name BackendToolCallFunction

// BackendToolCallFunctionArguments holds tool call arguments.
type BackendToolCallFunctionArguments map[string]any

// BackendMessage mirrors Ollama's chat message schema.
type BackendMessage struct {
	Role       string             `json:"role"`
	Content    string             `json:"content"`
	Thinking   string             `json:"thinking,omitempty"`
	Images     []BackendImageData `json:"images,omitempty"`
	ToolCalls  []BackendToolCall  `json:"tool_calls,omitempty"`
	ToolName   string             `json:"tool_name,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
} // @name BackendMessage

// BackendCopyModelRequest matches Ollama's copy API payload.
type BackendCopyModelRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
} // @name BackendCopyModelRequest

// BackendPullModelRequest matches Ollama's pull API payload.
type BackendPullModelRequest struct {
	Model    string `json:"model"`
	Insecure bool   `json:"insecure,omitempty"`
	Stream   bool   `json:"stream,omitempty"`
} // @name BackendPullModelRequest

// BackendPushModelRequest matches Ollama's push API payload.
type BackendPushModelRequest struct {
	Model    string `json:"model"`
	Insecure bool   `json:"insecure,omitempty"`
	Stream   bool   `json:"stream,omitempty"`
} // @name BackendPushModelRequest

// BackendDeleteModelRequest matches Ollama's delete API payload.
type BackendDeleteModelRequest struct {
	Model string `json:"model"`
} // @name BackendDeleteModelRequest

// BackendShowModelRequest matches Ollama's show API payload.
type BackendShowModelRequest struct {
	Model  string `json:"model"`
	System string `json:"system,omitempty"`
} // @name BackendShowModelRequest

// BackendShowModelResponse captures the important fields from Ollama's show response.
type BackendShowModelResponse struct {
	Model      string                  `json:"model"`
	License    string                  `json:"license,omitempty"`
	Modelfile  string                  `json:"modelfile,omitempty"`
	Template   string                  `json:"template,omitempty"`
	Parameters map[string]any          `json:"parameters,omitempty"`
	Details    BackendShowModelDetails `json:"details"`
	ModifiedAt time.Time               `json:"modified_at"`
} // @name BackendShowModelResponse

// BackendShowModelDetails describes metadata extracted from the model.
type BackendShowModelDetails struct {
	Family            string `json:"family,omitempty"`
	ParameterSize     string `json:"parameter_size,omitempty"`
	QuantizationLevel string `json:"quantization_level,omitempty"`
} // @name BackendShowModelDetails

// BackendOperationResponse represents the streaming status envelopes returned by Ollama's admin APIs.
type BackendOperationResponse struct {
	Status string `json:"status"`
	Model  string `json:"model,omitempty"`
	Digest string `json:"digest,omitempty"`
	Detail string `json:"detail,omitempty"`
} // @name BackendOperationResponse

// BackendVersionResponse mirrors Ollama's version endpoint.
type BackendVersionResponse struct {
	Version string `json:"version"`
} // @name BackendVersionResponse
