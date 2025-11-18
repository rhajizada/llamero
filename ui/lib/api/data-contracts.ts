/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface Backend {
  address?: string;
  healthy?: boolean;
  id?: string;
  latency_ms?: number;
  /** Models currently running in Ollama. */
  loaded_models?: string[];
  /** Installed models available on disk. */
  models?: string[];
  tags?: string[];
  updated_at?: string;
}

export interface BackendCopyModelRequest {
  destination?: string;
  source?: string;
}

export interface BackendCreateModelRequest {
  keep_alive?: string;
  /** Name of the model to create. */
  model?: string;
  /** Inline Modelfile contents. */
  modelfile?: string;
  /** Optional path to an existing Modelfile. */
  path?: string;
  /** Quantization target, e.g. "Q4_0". */
  quantize?: string;
}

export interface BackendDeleteModelRequest {
  force?: boolean;
  model?: string;
}

export interface BackendOperationResponse {
  detail?: string;
  digest?: string;
  model?: string;
  status?: string;
}

export interface BackendPullModelRequest {
  insecure?: boolean;
  model?: string;
  stream?: boolean;
}

export interface BackendPushModelRequest {
  insecure?: boolean;
  model?: string;
  stream?: boolean;
}

export interface BackendShowModelDetails {
  family?: string;
  parameter_size?: string;
  quantization_level?: string;
}

export interface BackendShowModelRequest {
  model?: string;
  system?: string;
}

export interface BackendShowModelResponse {
  details?: BackendShowModelDetails;
  license?: string;
  model?: string;
  modelfile?: string;
  modified_at?: string;
  parameters?: Record<string, any>;
  template?: string;
}

export interface BackendTagsResponse {
  models?: OllamaTag[];
}

export interface BackendVersionResponse {
  version?: string;
}

export interface ChatCompletionChoice {
  delta?: ChatMessage;
  finish_reason?: string;
  index?: number;
  logprobs?: ChatCompletionLogProbs;
  message?: ChatMessage;
}

export interface ChatCompletionLogProbs {
  content?: LogProb[];
}

export interface ChatCompletionRequest {
  frequency_penalty?: number;
  max_tokens?: number;
  messages?: ChatMessage[];
  model?: string;
  presence_penalty?: number;
  response_format?: ResponseFormatSpec;
  stop?: string[];
  stream?: boolean;
  temperature?: number;
  tool_choice?: any;
  tools?: ChatTool[];
  top_p?: number;
  user?: string;
}

export interface ChatCompletionResponse {
  choices?: ChatCompletionChoice[];
  created?: number;
  id?: string;
  model?: string;
  object?: string;
  system_fingerprint?: string;
  usage?: ChatCompletionUsage;
}

export interface ChatCompletionUsage {
  completion_tokens?: number;
  prompt_tokens?: number;
  total_tokens?: number;
}

export interface ChatMessage {
  content?: string;
  metadata?: Record<string, string>;
  name?: string;
  role?: string;
  tool_calls?: ToolCall[];
  tool_id?: string;
}

export interface ChatTool {
  function?: ToolDefinition;
  type?: string;
}

export interface CompletionChoice {
  finish_reason?: string;
  index?: number;
  logprobs?: CompletionLogProbs;
  text?: string;
}

export interface CompletionLogProbs {
  text_offset?: number[];
  token_logprobs?: number[];
  tokens?: string[];
  top_logprobs?: Record<string, number>[];
}

export interface CompletionRequest {
  best_of?: number;
  echo?: boolean;
  frequency_penalty?: number;
  logprobs?: number;
  max_tokens?: number;
  model?: string;
  n?: number;
  presence_penalty?: number;
  /** string or []string */
  prompt?: any;
  stop?: string[];
  stream?: boolean;
  suffix?: string;
  temperature?: number;
  top_p?: number;
  user?: string;
}

export interface CompletionResponse {
  choices?: CompletionChoice[];
  created?: number;
  id?: string;
  model?: string;
  object?: string;
  usage?: CompletionUsage;
}

export interface CompletionUsage {
  completion_tokens?: number;
  prompt_tokens?: number;
  total_tokens?: number;
}

export interface CreatePersonalAccessTokenRequest {
  expires_in?: number;
  name?: string;
  scopes?: string[];
}

export interface EmbeddingData {
  embedding?: number[];
  index?: number;
  object?: string;
}

export interface EmbeddingsRequest {
  /** string or []string */
  input?: any;
  model?: string;
  user?: string;
}

export interface EmbeddingsResponse {
  data?: EmbeddingData[];
  model?: string;
  object?: string;
  usage?: EmbeddingsUsage;
}

export interface EmbeddingsUsage {
  prompt_tokens?: number;
  total_tokens?: number;
}

export interface LogProb {
  logprob?: number;
  token?: string;
  top_logprobs?: {
    logprob?: number;
    token?: string;
  }[];
}

export interface Model {
  created?: number;
  id?: string;
  object?: string;
  owned_by?: string;
}

export interface ModelDetails {
  families?: string[];
  family?: string;
  format?: string;
  parameter_size?: string;
  parent_model?: string;
  quantization_level?: string;
}

export interface ModelList {
  data?: Model[];
  object?: string;
}

export interface OllamaTag {
  digest?: string;
  modified_at?: string;
  name?: string;
  size?: number;
}

export interface PersonalAccessToken {
  created_at?: string;
  expires_at?: string;
  id?: string;
  jti?: string;
  last_used_at?: string;
  name?: string;
  revoked?: boolean;
  scopes?: string[];
  token_type?: string;
  updated_at?: string;
  user_id?: string;
}

export interface PersonalAccessTokenResponse {
  created_at?: string;
  expires_at?: string;
  id?: string;
  jti?: string;
  last_used_at?: string;
  name?: string;
  revoked?: boolean;
  scopes?: string[];
  token?: string;
  token_type?: string;
  updated_at?: string;
  user_id?: string;
}

export interface ProcessModelResponse {
  context_length?: number;
  details?: ModelDetails;
  digest?: string;
  expires_at?: string;
  model?: string;
  name?: string;
  size?: number;
  size_vram?: number;
}

export interface ResponseFormatSpec {
  type?: string;
}

export interface ToolCall {
  function?: ToolCallFunction;
  id?: string;
  type?: string;
}

export interface ToolCallFunction {
  arguments?: string;
  name?: string;
}

export interface ToolDefinition {
  description?: string;
  name?: string;
  parameters?: any;
}

export interface User {
  created_at?: string;
  display_name?: string;
  email?: string;
  groups?: string[];
  id?: string;
  last_login_at?: string;
  provider?: string;
  role?: string;
  scopes?: string[];
  sub?: string;
  updated_at?: string;
}
