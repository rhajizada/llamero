"use client";

import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createApiClient } from "@/lib/api-client";
import { useAuth } from "@/components/AuthProvider";
import type { Backend } from "@/lib/api/data-contracts";

const actionLabels = {
  ps: "List running models",
  version: "Get Ollama version",
  pull: "Pull model",
  push: "Push model",
  create: "Create model",
  copy: "Copy model",
  delete: "Delete model",
  show: "Inspect model",
  models: "List models",
} as const;

type AdminAction = keyof typeof actionLabels;

type ActionResult = string | null;

interface BackendActionFields {
  model?: string;
  destination?: string;
  source?: string;
  modelfile?: string;
  system?: string;
  keep_alive?: string;
  quantize?: string;
  force?: boolean;
}

type StringFieldKey = Extract<
  {
    [K in keyof BackendActionFields]: BackendActionFields[K] extends string | undefined
      ? K
      : never;
  }[keyof BackendActionFields],
  string
>;

interface FieldConfig {
  key: StringFieldKey;
  label: string;
  textarea?: boolean;
}

interface ActionRequest {
  method: string;
  path: string;
  body?: Record<string, unknown>;
  requiresBackend?: boolean;
}

const streamingActions = new Set<AdminAction>(["pull", "push", "create", "copy"]);

const buildActionRequest = (
  action: AdminAction,
  backendId: string,
  fields: BackendActionFields,
): ActionRequest => {
  switch (action) {
    case "ps":
      return {
        method: "GET",
        path: `/api/backends/${backendId}/ps`,
        requiresBackend: true,
      };
    case "version":
      return {
        method: "GET",
        path: `/api/backends/${backendId}/version`,
        requiresBackend: true,
      };
    case "pull":
      return {
        method: "POST",
        path: `/api/backends/${backendId}/pull`,
        body: { model: fields.model },
        requiresBackend: true,
      };
    case "push":
      return {
        method: "POST",
        path: `/api/backends/${backendId}/push`,
        body: { model: fields.model },
        requiresBackend: true,
      };
    case "create":
      return {
        method: "POST",
        path: `/api/backends/${backendId}/create`,
        body: {
          model: fields.model,
          modelfile: fields.modelfile,
          keep_alive: fields.keep_alive,
          quantize: fields.quantize,
        },
        requiresBackend: true,
      };
    case "copy":
      return {
        method: "POST",
        path: `/api/backends/${backendId}/copy`,
        body: {
          source: fields.source,
          destination: fields.destination,
        },
        requiresBackend: true,
      };
    case "delete":
      return {
        method: "DELETE",
        path: `/api/backends/${backendId}/delete`,
        body: {
          model: fields.model,
          force: Boolean(fields.force),
        },
        requiresBackend: true,
      };
    case "show":
      return {
        method: "POST",
        path: `/api/backends/${backendId}/show`,
        body: { model: fields.model, system: fields.system },
        requiresBackend: true,
      };
    case "models":
      // Models are handled via the typed client in onSubmit, but keep this for exhaustiveness.
      return {
        method: "GET",
        path: `/api/backends/${backendId}/tags`,
        requiresBackend: true,
      };
    default: {
      throw new Error(`Unhandled action: ${action satisfies never}`);
    }
  }
};

const sanitizeBody = (body?: Record<string, unknown>) => {
  if (!body) return undefined;
  const payload: Record<string, unknown> = {};
  Object.entries(body).forEach(([key, value]) => {
    if (value === undefined || value === "") {
      return;
    }
    payload[key] = value;
  });
  return Object.keys(payload).length ? payload : undefined;
};

const formatLatency = (value?: number) => {
  if (typeof value !== "number") return "—";
  return `${value.toFixed(0)} ms`;
};

const formatUpdated = (value?: string) => {
  if (!value) return "—";
  return new Intl.DateTimeFormat("en-US", {
    timeZone: "UTC",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(value));
};

export const BackendsConsole = () => {
  const { token } = useAuth();
  const [backends, setBackends] = useState<Backend[]>([]);
  const [selectedBackend, setSelectedBackend] = useState<string>("");
  const [action, setAction] = useState<AdminAction>("ps");
  const [fields, setFields] = useState<BackendActionFields>({});
  const [loading, setLoading] = useState(false);
  const [listLoading, setListLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<ActionResult>(null);
  const [isStreaming, setIsStreaming] = useState(false);
  const resultRef = useRef<HTMLPreElement | null>(null);

  useEffect(() => {
    if (isStreaming && resultRef.current) {
      resultRef.current.scrollTop = resultRef.current.scrollHeight;
    }
  }, [isStreaming, result]);

  const fetchBackends = useCallback(async () => {
    if (!token) return;
    setListLoading(true);
    setError(null);
    try {
      const api = createApiClient(token);
      const response = await api.backendsList();
      setBackends(response.data ?? []);
      setSelectedBackend((prev) => {
        if (prev) return prev;
        return response.data?.[0]?.id || "";
      });
    } catch (err) {
      console.error("load backends", err);
      setError("Unable to load backends");
    } finally {
      setListLoading(false);
    }
  }, [token]);

  useEffect(() => {
    fetchBackends();
  }, [fetchBackends]);

  const createRequestInit = useCallback(
    (request: ActionRequest): RequestInit => {
      if (!token) {
        throw new Error("missing token");
      }
      const payload = sanitizeBody(request.body);
      const shouldSendBody = request.method !== "GET" && payload !== undefined;
      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
      };
      if (shouldSendBody) {
        headers["Content-Type"] = "application/json";
      }
      return {
        method: request.method,
        headers,
        body: shouldSendBody ? JSON.stringify(payload) : undefined,
      };
    },
    [token],
  );

  const executeStandardRequest = useCallback(
    async (request: ActionRequest) => {
      const response = await fetch(request.path, createRequestInit(request));
      const text = await response.text();
      if (!response.ok) {
        throw new Error(text || response.statusText);
      }
      let printable = text || response.statusText;
      try {
        printable = text ? JSON.stringify(JSON.parse(text), null, 2) : response.statusText;
      } catch {
        printable = text || response.statusText;
      }
      setResult(printable);
    },
    [createRequestInit],
  );

  const executeStreamingRequest = useCallback(
    async (request: ActionRequest) => {
      const response = await fetch(request.path, createRequestInit(request));
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || response.statusText);
      }
      const reader = response.body?.getReader();
      if (!reader) {
        const fallback = await response.text();
        setResult(fallback || "Stream completed");
        return;
      }
      const decoder = new TextDecoder();
      let hasChunks = false;
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        const chunk = decoder.decode(value, { stream: true });
        hasChunks = true;
        setResult((prev) => (prev ? `${prev}${chunk}` : chunk));
      }
      if (!hasChunks) {
        setResult("Stream completed");
      }
    },
    [createRequestInit],
  );

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!token) {
      setError("Sign in before running console actions");
      return;
    }
    if (action === "models") {
      setLoading(true);
      setIsStreaming(false);
      setError(null);
      setResult("");
      try {
        const api = createApiClient(token);
        const resp = await api.backendsTagsList(selectedBackend);
        setResult(JSON.stringify(resp.data, null, 2));
      } catch (err) {
        const message = err instanceof Error ? err.message : "Action failed";
        setResult(message);
      } finally {
        setLoading(false);
        setIsStreaming(false);
      }
      return;
    }
    const request = buildActionRequest(action, selectedBackend, fields);
    if (request.requiresBackend && !selectedBackend) {
      setError("Select a backend before running an action");
      return;
    }
    const streams = streamingActions.has(action);
    setLoading(true);
    setIsStreaming(streams);
    setError(null);
    setResult("");
    try {
      if (streams) {
        await executeStreamingRequest(request);
      } else {
        await executeStandardRequest(request);
      }
    } catch (err) {
      console.error("backend action", err);
      const message = err instanceof Error ? err.message : "Action failed";
      setResult(message);
    } finally {
      setLoading(false);
      setIsStreaming(false);
    }
  };

  const actionFields = useMemo<FieldConfig[]>(() => {
    switch (action) {
      case "copy":
        return [
          { key: "source", label: "Source model" },
          { key: "destination", label: "Destination model" },
        ];
      case "pull":
      case "push":
      case "delete":
      case "show":
        return [{ key: "model", label: "Model" }];
      case "create":
        return [
          { key: "model", label: "Model" },
          { key: "modelfile", label: "Modelfile", textarea: true },
          { key: "keep_alive", label: "Keep alive" },
          { key: "quantize", label: "Quantization" },
        ];
      default:
        return [];
    }
  }, [action]);

  const showForceToggle = action === "delete";
  const resultDisplay =
    result && result.length
      ? result
      : loading
        ? isStreaming
          ? "Streaming..."
          : "Running..."
        : "Awaiting action";

  return (
    <section className="rounded-3xl border border-border bg-card/50 p-6 shadow-sm">
      <header className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <h2 className="text-xl font-semibold text-foreground">Backend console</h2>
        <button
          type="button"
          onClick={fetchBackends}
          className="rounded-full border border-border px-4 py-2 text-xs font-medium"
        >
          {listLoading ? "Refreshing…" : "Refresh list"}
        </button>
      </header>
      {error ? (
        <div className="mb-4 rounded-xl border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      ) : null}
      <div className="space-y-6">
        <div className="rounded-2xl border border-border bg-background/70 p-4">
          <h3 className="text-sm font-semibold text-foreground">Fleet status</h3>
          <div className="mt-4 space-y-3">
            {backends.map((backend) => (
              <div
                key={backend.id}
                className="rounded-xl border border-border/60 bg-card/40 p-4"
              >
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="text-base font-semibold">{backend.id}</p>
                    <p className="text-xs text-muted-foreground">{backend.address || "no address"}</p>
                  </div>
                  <span
                    className={`rounded-full px-3 py-1 text-xs font-semibold ${backend.healthy ? "bg-emerald-500/20 text-emerald-500" : "bg-destructive/20 text-destructive"}`}
                  >
                    {backend.healthy ? "Healthy" : "Unreachable"}
                  </span>
                </div>
                <dl className="mt-3 grid grid-cols-2 gap-2 text-xs text-muted-foreground">
                  <div>
                    <dt>Latency</dt>
                    <dd className="font-semibold text-foreground">{formatLatency(backend.latency_ms)}</dd>
                  </div>
                  <div>
                    <dt>Updated</dt>
                    <dd className="font-semibold text-foreground">{formatUpdated(backend.updated_at)}</dd>
                  </div>
                  <div className="col-span-2">
                    <dt>Loaded models</dt>
                    <dd className="text-foreground">
                      {(backend.loaded_models || []).length ? backend.loaded_models?.join(", ") : "—"}
                    </dd>
                  </div>
                  <div className="col-span-2">
                    <dt>Available models</dt>
                    <dd className="text-foreground">
                      {(backend.models || []).length ? backend.models?.join(", ") : "—"}
                    </dd>
                  </div>
                </dl>
              </div>
            ))}
            {backends.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                {listLoading ? "Discovering backends…" : "No backends registered"}
              </p>
            ) : null}
          </div>
        </div>
        <div className="space-y-4">
          <div className="rounded-2xl border border-border bg-background/70 p-4">
            <label className="text-xs text-muted-foreground">
              Backend
            </label>
            <select
              value={selectedBackend}
              onChange={(e) => setSelectedBackend(e.target.value)}
              className="mt-2 w-full rounded-xl border border-border bg-card/50 px-3 py-2 text-sm"
            >
              {backends.map((backend) => (
                <option key={backend.id} value={backend.id}>
                  {backend.id}
                </option>
              ))}
            </select>
            <label className="mt-4 block text-xs text-muted-foreground">
              Action
            </label>
            <select
              value={action}
              onChange={(e) => setAction(e.target.value as AdminAction)}
              className="mt-2 w-full rounded-xl border border-border bg-card/50 px-3 py-2 text-sm"
            >
              {(Object.keys(actionLabels) as AdminAction[]).map((key) => (
                <option key={key} value={key}>
                  {actionLabels[key]}
                </option>
              ))}
            </select>
            <form className="mt-4 space-y-3" onSubmit={onSubmit}>
              {actionFields.map((field) => (
                <div key={field.key}>
                  <label className="text-xs text-muted-foreground">
                    {field.label}
                  </label>
                  {field.textarea ? (
                    <textarea
                      value={fields[field.key] || ""}
                      onChange={(e) =>
                        setFields((prev) => ({
                          ...prev,
                          [field.key]: e.target.value,
                        }))
                      }
                      className="mt-2 h-24 w-full rounded-xl border border-border bg-card/50 px-3 py-2 text-sm"
                    />
                  ) : (
                    <input
                      type="text"
                      value={fields[field.key] || ""}
                      onChange={(e) =>
                        setFields((prev) => ({
                          ...prev,
                          [field.key]: e.target.value,
                        }))
                      }
                      className="mt-2 w-full rounded-xl border border-border bg-card/50 px-3 py-2 text-sm"
                    />
                  )}
                </div>
              ))}
              {showForceToggle ? (
                <label className="flex items-center gap-2 text-xs text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={Boolean(fields.force)}
                    onChange={(e) =>
                      setFields((prev) => ({
                        ...prev,
                        force: e.target.checked,
                      }))
                    }
                    className="h-4 w-4 rounded border border-border"
                  />
                  Force delete
                </label>
              ) : null}
              <button
                type="submit"
                disabled={loading}
                className="w-full rounded-xl bg-foreground px-4 py-2 text-sm font-semibold text-background transition hover:opacity-90 disabled:opacity-60"
              >
                {loading ? "Running…" : "Execute"}
              </button>
            </form>
          </div>
          <div className="rounded-2xl border border-border bg-background/70 p-4">
            <h3 className="text-sm font-semibold text-muted-foreground">Result</h3>
            <pre
              ref={resultRef}
              className="mt-3 max-h-80 overflow-auto rounded-xl bg-black/80 p-4 text-xs text-white"
            >
              {resultDisplay}
            </pre>
          </div>
        </div>
      </div>
    </section>
  );
};
