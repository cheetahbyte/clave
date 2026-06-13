import { selfServiceFetch } from "@/lib/api";

export interface License {
  is_active: boolean;
  id: string;
  max_activations: number;
  name: string;
  expires_at?: string;
  logo_url?: string | null;
}

export interface Device {
  id: string;
  name: string;
  last_seen?: string;
  registered_at?: string;
}

export async function requestSelfServiceLink(email: string, orgSlug: string): Promise<{ ok: boolean }> {
  return selfServiceFetch<{ ok: boolean }>("/api/v1/self-service/auth/request-token", {
    method: "POST",
    body: JSON.stringify({ email, orgSlug }),
  });
}

export async function validateSelfServiceToken(token: string, orgSlug: string): Promise<{ ok: boolean; expiresAt: number }> {
  return selfServiceFetch<{ ok: boolean; expiresAt: number }>("/api/v1/self-service/auth/validate", {
    method: "POST",
    body: JSON.stringify({ token, orgSlug }),
  });
}

export async function checkSelfServiceSession(): Promise<boolean> {
  try {
    const res = await fetch("/api/v1/self-service/session", { credentials: "include" });
    return res.status === 204;
  } catch {
    return false;
  }
}

export async function logoutSelfService(): Promise<{ ok: boolean }> {
  return selfServiceFetch<{ ok: boolean }>("/api/v1/self-service/logout", {
    method: "POST",
  });
}

export async function listSelfServiceLicenses(): Promise<License[]> {
  return selfServiceFetch<License[]>("/api/v1/self-service/licenses");
}

export async function listSelfServiceDevices(licenseId: string): Promise<Device[]> {
  return selfServiceFetch<Device[]>(`/api/v1/self-service/licenses/${licenseId}/devices`);
}

export async function removeSelfServiceDevice(licenseId: string, deviceId: string): Promise<{ ok: boolean }> {
  return selfServiceFetch<{ ok: boolean }>(
    `/api/v1/self-service/licenses/${licenseId}/devices/${deviceId}`,
    { method: "DELETE" },
  );
}

export async function revokeSelfServiceLicense(licenseId: string): Promise<{ ok: boolean; newLicenseId: string }> {
  return selfServiceFetch<{ ok: boolean; newLicenseId: string }>(
    `/api/v1/self-service/licenses/${licenseId}/revoke`,
    { method: "POST" },
  );
}
