import { Api } from "@/lib/api/Api";
import { API_BASE_URL } from "@/lib/env";

export const createApiClient = (token?: string) => {
  const client = new Api<string>({
    baseUrl: API_BASE_URL || undefined,
    securityWorker: (securityData) => {
      if (!securityData) return {};
      return {
        headers: {
          Authorization: `Bearer ${securityData}`,
        },
      };
    },
  });

  if (token) {
    client.setSecurityData(token);
  }

  return client;
};
