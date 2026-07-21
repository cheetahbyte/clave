import { expect, test } from "bun:test";
import { renderEmail } from "./templates.ts";

test("renders an admin two-factor code email", async () => {
	const rendered = await renderEmail({
		type: "admin.2fa_code",
		email: "admin@example.com",
		data: { code: "123456", ttlMinutes: 10 },
	});

	expect(rendered.subject).toBe("Your Clave verification code");
	expect(rendered.html).toContain("123456");
	expect(rendered.text).toContain("10 minutes");
});

test.each(["12345a", "1234567"])("rejects invalid code %s", async (code) => {
	await expect(
		renderEmail({
			type: "admin.2fa_code",
			email: "admin@example.com",
			data: { code, ttlMinutes: 10 },
		}),
	).rejects.toThrow("six-digit code");
});

test("rejects a non-positive two-factor code TTL", async () => {
	await expect(
		renderEmail({
			type: "admin.2fa_code",
			email: "admin@example.com",
			data: { code: "123456", ttlMinutes: 0 },
		}),
	).rejects.toThrow("positive integer ttlMinutes");
});
