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

import {
  Backend,
  BackendCopyModelRequest,
  BackendCreateModelRequest,
  BackendDeleteModelRequest,
  BackendOperationResponse,
  BackendPullModelRequest,
  BackendPushModelRequest,
  BackendShowModelRequest,
  BackendShowModelResponse,
  BackendTagsResponse,
  BackendVersionResponse,
  ChatCompletionRequest,
  ChatCompletionResponse,
  CompletionRequest,
  CompletionResponse,
  CreatePersonalAccessTokenRequest,
  EmbeddingsRequest,
  EmbeddingsResponse,
  Model,
  ModelList,
  PersonalAccessToken,
  PersonalAccessTokenResponse,
  ProcessModelResponse,
  User,
} from "./data-contracts";
import { ContentType, HttpClient, RequestParams } from "./http-client";

export class Api<
  SecurityDataType = unknown,
> extends HttpClient<SecurityDataType> {
  /**
   * No description
   *
   * @tags Backends
   * @name BackendsList
   * @summary List registered backends
   * @request GET:/api/backends
   * @secure
   */
  backendsList = (params: RequestParams = {}) =>
    this.request<Backend[], Record<string, string>>({
      path: `/api/backends`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Backends
   * @name BackendsCopyCreate
   * @summary Copy a model on the specified backend
   * @request POST:/api/backends/{backendID}/copy
   * @secure
   */
  backendsCopyCreate = (
    backendId: string,
    request: BackendCopyModelRequest,
    params: RequestParams = {},
  ) =>
    this.request<BackendOperationResponse, Record<string, string>>({
      path: `/api/backends/${backendId}/copy`,
      method: "POST",
      body: request,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Backends
   * @name BackendsCreateCreate
   * @summary Create a model on the specified backend
   * @request POST:/api/backends/{backendID}/create
   * @secure
   */
  backendsCreateCreate = (
    backendId: string,
    request: BackendCreateModelRequest,
    params: RequestParams = {},
  ) =>
    this.request<BackendOperationResponse, Record<string, string>>({
      path: `/api/backends/${backendId}/create`,
      method: "POST",
      body: request,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Backends
   * @name BackendsDeleteDelete
   * @summary Delete a model from the specified backend
   * @request DELETE:/api/backends/{backendID}/delete
   * @secure
   */
  backendsDeleteDelete = (
    backendId: string,
    request: BackendDeleteModelRequest,
    params: RequestParams = {},
  ) =>
    this.request<BackendOperationResponse, Record<string, string>>({
      path: `/api/backends/${backendId}/delete`,
      method: "DELETE",
      body: request,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * @description Forwards the request to the backend's /api/ps endpoint.
   *
   * @tags Backends
   * @name BackendsPsList
   * @summary List running models on a backend
   * @request GET:/api/backends/{backendID}/ps
   * @secure
   */
  backendsPsList = (backendId: string, params: RequestParams = {}) =>
    this.request<ProcessModelResponse[], Record<string, string>>({
      path: `/api/backends/${backendId}/ps`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Backends
   * @name BackendsPullCreate
   * @summary Pull a model on the specified backend
   * @request POST:/api/backends/{backendID}/pull
   * @secure
   */
  backendsPullCreate = (
    backendId: string,
    request: BackendPullModelRequest,
    params: RequestParams = {},
  ) =>
    this.request<BackendOperationResponse, Record<string, string>>({
      path: `/api/backends/${backendId}/pull`,
      method: "POST",
      body: request,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Backends
   * @name BackendsPushCreate
   * @summary Push a model from the specified backend
   * @request POST:/api/backends/{backendID}/push
   * @secure
   */
  backendsPushCreate = (
    backendId: string,
    request: BackendPushModelRequest,
    params: RequestParams = {},
  ) =>
    this.request<BackendOperationResponse, Record<string, string>>({
      path: `/api/backends/${backendId}/push`,
      method: "POST",
      body: request,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Backends
   * @name BackendsShowCreate
   * @summary Show model details on the specified backend
   * @request POST:/api/backends/{backendID}/show
   * @secure
   */
  backendsShowCreate = (
    backendId: string,
    request: BackendShowModelRequest,
    params: RequestParams = {},
  ) =>
    this.request<BackendShowModelResponse, Record<string, string>>({
      path: `/api/backends/${backendId}/show`,
      method: "POST",
      body: request,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Backends
   * @name BackendsTagsList
   * @summary List available models on a backend
   * @request GET:/api/backends/{backendID}/tags
   * @secure
   */
  backendsTagsList = (backendId: string, params: RequestParams = {}) =>
    this.request<BackendTagsResponse, Record<string, string>>({
      path: `/api/backends/${backendId}/tags`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Backends
   * @name BackendsVersionList
   * @summary Retrieve Ollama version of specified backend
   * @request GET:/api/backends/{backendID}/version
   * @secure
   */
  backendsVersionList = (backendId: string, params: RequestParams = {}) =>
    this.request<BackendVersionResponse, Record<string, string>>({
      path: `/api/backends/${backendId}/version`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags LLM
   * @name ChatCompletionsCreate
   * @summary Proxy chat completions
   * @request POST:/api/chat/completions
   * @secure
   */
  chatCompletionsCreate = (
    request: ChatCompletionRequest,
    params: RequestParams = {},
  ) =>
    this.request<ChatCompletionResponse, Record<string, string>>({
      path: `/api/chat/completions`,
      method: "POST",
      body: request,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags LLM
   * @name CompletionsCreate
   * @summary Proxy legacy completions
   * @request POST:/api/completions
   * @secure
   */
  completionsCreate = (
    request: CompletionRequest,
    params: RequestParams = {},
  ) =>
    this.request<CompletionResponse, Record<string, string>>({
      path: `/api/completions`,
      method: "POST",
      body: request,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags LLM
   * @name EmbeddingsCreate
   * @summary Proxy embeddings
   * @request POST:/api/embeddings
   * @secure
   */
  embeddingsCreate = (request: EmbeddingsRequest, params: RequestParams = {}) =>
    this.request<EmbeddingsResponse, Record<string, string>>({
      path: `/api/embeddings`,
      method: "POST",
      body: request,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Models
   * @name ModelsList
   * @summary List available models
   * @request GET:/api/models
   * @secure
   */
  modelsList = (params: RequestParams = {}) =>
    this.request<ModelList, Record<string, string>>({
      path: `/api/models`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Models
   * @name ModelsDetail
   * @summary Get metadata for a single model
   * @request GET:/api/models/{modelID}
   * @secure
   */
  modelsDetail = (modelId: string, params: RequestParams = {}) =>
    this.request<Model, Record<string, string>>({
      path: `/api/models/${modelId}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Users
   * @name ProfileList
   * @summary Get authenticated user profile
   * @request GET:/api/profile
   * @secure
   */
  profileList = (params: RequestParams = {}) =>
    this.request<User, Record<string, string>>({
      path: `/api/profile`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Profile
   * @name ProfileTokensList
   * @summary List personal access tokens
   * @request GET:/api/profile/tokens
   * @secure
   */
  profileTokensList = (params: RequestParams = {}) =>
    this.request<PersonalAccessToken[], Record<string, string>>({
      path: `/api/profile/tokens`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Profile
   * @name ProfileTokensCreate
   * @summary Create a personal access token
   * @request POST:/api/profile/tokens
   * @secure
   */
  profileTokensCreate = (
    payload: CreatePersonalAccessTokenRequest,
    params: RequestParams = {},
  ) =>
    this.request<PersonalAccessTokenResponse, Record<string, string>>({
      path: `/api/profile/tokens`,
      method: "POST",
      body: payload,
      secure: true,
      type: ContentType.Json,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Profile
   * @name ProfileTokensDetail
   * @summary Get personal access token metadata
   * @request GET:/api/profile/tokens/{tokenID}
   * @secure
   */
  profileTokensDetail = (tokenId: string, params: RequestParams = {}) =>
    this.request<PersonalAccessToken, Record<string, string>>({
      path: `/api/profile/tokens/${tokenId}`,
      method: "GET",
      secure: true,
      format: "json",
      ...params,
    });
  /**
   * No description
   *
   * @tags Profile
   * @name ProfileTokensDelete
   * @summary Revoke a personal access token
   * @request DELETE:/api/profile/tokens/{tokenID}
   * @secure
   */
  profileTokensDelete = (tokenId: string, params: RequestParams = {}) =>
    this.request<string, Record<string, string>>({
      path: `/api/profile/tokens/${tokenId}`,
      method: "DELETE",
      secure: true,
      ...params,
    });
}
