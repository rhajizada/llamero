type StringRecord = Record<string, unknown>;

const isStringRecord = (value: unknown): value is StringRecord =>
  typeof value === "object" && value !== null;

const getNestedString = (value: unknown, path: string[]): string | null => {
  let current: unknown = value;

  for (const key of path) {
    if (!isStringRecord(current)) return null;
    current = current[key];
  }

  return typeof current === "string" ? current : null;
};

export const getErrorMessage = (err: unknown, fallback: string): string => {
  if (!err) return fallback;

  const candidatePaths: string[][] = [
    ["error"],
    ["error", "error"],
    ["data", "error"],
    ["message"],
    ["statusText"],
  ];

  const candidate =
    typeof err === "string"
      ? err
      : candidatePaths
          .map((path) => getNestedString(err, path))
          .find((value) => typeof value === "string") ?? null;

  if (candidate && typeof candidate === "string" && candidate.trim().length) {
    return candidate.trim();
  }

  return fallback;
};
