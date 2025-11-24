export const getErrorMessage = (err: unknown, fallback: string) => {
  if (!err) return fallback;

  const candidate =
    typeof err === "string"
      ? err
      : typeof (err as any)?.error === "string"
        ? (err as any).error
        : typeof (err as any)?.error?.error === "string"
          ? (err as any).error.error
          : typeof (err as any)?.data?.error === "string"
            ? (err as any).data.error
            : typeof (err as any)?.message === "string"
              ? (err as any).message
              : typeof (err as any)?.statusText === "string"
                ? (err as any).statusText
                : null;

  if (candidate && typeof candidate === "string" && candidate.trim().length) {
    return candidate.trim();
  }

  return fallback;
};
