export function getGoAPIBaseURL(): string {
  const configuredURL = process.env.GO_API_BASE_URL?.trim();

  if (!configuredURL) {
    throw new Error("GO_API_BASE_URL is not configured");
  }

  const normalizedURL = configuredURL.replace(/\/+$/, "");
  if (!normalizedURL) {
    throw new Error("GO_API_BASE_URL is not configured");
  }

  return normalizedURL;
}
