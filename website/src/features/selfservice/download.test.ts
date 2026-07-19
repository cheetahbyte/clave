import { describe, expect, it } from "vitest";
import {
  isLicenseDownloadable,
  latestDownloadURL,
  normalizeDownloadPlatform,
} from "./download";

describe("normalizeDownloadPlatform", () => {
  it.each([
    ["MacIntel", "macos"],
    ["macOS", "macos"],
    ["Win32", "windows"],
    ["Windows", "windows"],
    ["Linux x86_64", "linux"],
  ])("maps %s to %s", (input, expected) => {
    expect(normalizeDownloadPlatform(input)).toBe(expected);
  });

  it("returns null for unsupported platforms", () => {
    expect(normalizeDownloadPlatform("FreeBSD")).toBeNull();
  });
});

it("adds an encoded platform while preserving the base URL", () => {
  expect(latestDownloadURL("/api/download", "macos")).toBe(
    "/api/download?platform=macos",
  );
});

describe("isLicenseDownloadable", () => {
  const now = new Date("2026-07-19T12:00:00Z");

  it("accepts an active unexpired license", () => {
    expect(
      isLicenseDownloadable(
        { is_active: true, expires_at: "2026-07-20T12:00:00Z" },
        now,
      ),
    ).toBe(true);
  });

  it("rejects inactive, expired, and invalid-expiry licenses", () => {
    expect(isLicenseDownloadable({ is_active: false }, now)).toBe(false);
    expect(
      isLicenseDownloadable(
        { is_active: true, expires_at: "2026-07-18T12:00:00Z" },
        now,
      ),
    ).toBe(false);
    expect(
      isLicenseDownloadable({ is_active: true, expires_at: "invalid" }, now),
    ).toBe(false);
  });
});
