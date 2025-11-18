"use client";

import { useEffect, useMemo, useState } from "react";
import { createApiClient } from "@/lib/api-client";
import { useAuth } from "@/components/AuthProvider";
import type { Model, ModelList } from "@/lib/api/data-contracts";

const formatDate = (value?: string | number) => {
  if (!value) return "—";
  const numeric = typeof value === "number" ? value : Number(value);
  const date =
    !Number.isNaN(numeric) && numeric < 1e12
      ? new Date(numeric * 1000)
      : new Date(value);
  return new Intl.DateTimeFormat("en-US", {
    timeZone: "UTC",
    year: "numeric",
    month: "short",
    day: "2-digit",
  }).format(date);
};

export const ModelsPanel = () => {
  const { token } = useAuth();
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchModels = async () => {
      if (!token) return;
      setLoading(true);
      setError(null);
      try {
        const api = createApiClient(token);
        const response = await api.modelsList();
        const list = (response.data as ModelList | undefined)?.data || [];
        setModels(list);
      } catch (err) {
        console.error("load models", err);
        setError("Unable to load models");
      } finally {
        setLoading(false);
      }
    };
    fetchModels();
  }, [token]);

  const tableRows = useMemo(() => models || [], [models]);

  return (
    <section className="rounded-3xl border border-border bg-card/60 p-6 shadow-sm">
      <header className="mb-4 flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 className="text-xl font-semibold text-foreground">Models</h2>
        </div>
      </header>
      {error ? (
        <div className="mb-4 rounded-xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      ) : null}
      <div className="overflow-hidden rounded-2xl border border-border bg-background/60">
        <table className="w-full text-left text-sm">
          <thead className="text-muted-foreground">
            <tr>
              <th className="px-4 py-3 font-normal">ID</th>
              <th className="px-4 py-3 font-normal">Owner</th>
              <th className="px-4 py-3 font-normal">Created</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td
                  className="px-4 py-4 text-sm text-muted-foreground"
                  colSpan={3}
                >
                  Loading models…
                </td>
              </tr>
            ) : tableRows.length === 0 ? (
              <tr>
                <td
                  className="px-4 py-4 text-sm text-muted-foreground"
                  colSpan={3}
                >
                  No models found
                </td>
              </tr>
            ) : (
              tableRows.map((model) => (
                <tr
                  key={model.id}
                  className="border-t border-border/60 hover:bg-muted/30"
                >
                  <td className="px-4 py-3 font-semibold text-foreground">
                    {model.id}
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    {model.owned_by || "n/a"}
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    {formatDate(model.created)}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
};
