import { Api } from "@/lib/api/Api";

export const createApiClient = (token?: string) => {
  const client = new Api<string>({
    baseUrl: "",
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
