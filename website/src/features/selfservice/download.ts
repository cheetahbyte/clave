import type { License } from "./api";

export type DownloadPlatform = "macos" | "windows" | "linux";

export function normalizeDownloadPlatform(
  input: string,
): DownloadPlatform | null {
  const value = input.toLowerCase();
  if (value.includes("mac")) return "macos";
  if (value.includes("win")) return "windows";
  if (value.includes("linux")) return "linux";
  return null;
}

export function browserDownloadPlatform(): DownloadPlatform | null {
  if (typeof navigator === "undefined") return null;
  const browserNavigator = navigator as Navigator & {
    userAgentData?: { platform?: string };
  };
  return normalizeDownloadPlatform(
    browserNavigator.userAgentData?.platform ??
      navigator.platform ??
      navigator.userAgent,
  );
}

export function latestDownloadURL(
  baseURL: string,
  platform: DownloadPlatform,
): string {
  const origin =
    typeof window === "undefined" ? "http://localhost" : window.location.origin;
  const url = new URL(baseURL, origin);
  url.searchParams.set("platform", platform);
  return /^https?:\/\//i.test(baseURL)
    ? url.toString()
    : `${url.pathname}${url.search}`;
}

export function isLicenseDownloadable(
  license: Pick<License, "is_active" | "expires_at">,
  now = new Date(),
): boolean {
  if (!license.is_active) return false;
  if (!license.expires_at) return true;
  const expiry = new Date(license.expires_at);
  return !Number.isNaN(expiry.getTime()) && expiry > now;
}
