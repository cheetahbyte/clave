"use server"

import transporter from "@/lib/mail";
import { cookies } from "next/headers";

export async function sendMagicSelfServiceLink(formData: FormData) {
  const email = String(formData.get("email") ?? "").trim().toLowerCase();

  if (!email) {
    return { ok: true };
  }

  const res = await fetch(`${process.env.BACKEND_URL}/api/v1/self-service/auth/request-token`, {method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Accept": "application/json",
    },
    body: JSON.stringify({ email }),
    cache: "no-store",
    credentials: "include"
  })

  if (!res.ok) {
    const text = await res.text();
    console.error("Backend error:", res.status, text);
    throw new Error(`Backend returned ${res.status}: ${text}`);
  }

  const { token, ok } = await res.json()
  const url = `http://localhost:3000/selfservice/auth?token=${token}`
  await transporter.sendMail({
            from: '"My App" <noreply@example.com>',
            to: email,
            subject: "Your Magic Link For Self Service",
            text: `Click the link to sign in: ${url}`,
            html: `
              <div style="font-family: sans-serif;">
                <h1>Sign in to Clave Self Service</h1>
                <p>Click the button below to authenticate instantly.</p>
                <a href="${url}" style="background: #000; color: #fff; padding: 10px 20px; text-decoration: none; border-radius: 5px;">
                Go to Self Service
                </a>
                <p style="margin-top: 20px; color: #666;">Or copy this link: ${url}</p>
              </div>
            `,
          });
}


export async function checkTokenValidity(): Promise<boolean> {
  const cookieStore = await cookies();
  const session = cookieStore.get("selfservice_session")?.value;

  if (!session) return false;

  const res = await fetch(`${process.env.BACKEND_URL}/api/v1/self-service/session`, {
    method: "GET",
    headers: {
      Cookie: `selfservice_session=${session}`,
      Accept: "application/json",
    },
    cache: "no-store",
  });

  return res.status === 204;
}


export async function validateToken(token: string): Promise<string> {
  const res = await fetch(`${process.env.BACKEND_URL}/api/v1/self-service/auth/validate`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify({ token }),
    cache: "no-store",
  });

  if (!res.ok) return "";
  const json = await res.json()
  return json.token ?? ""
}


export type License = {
  is_active: boolean
  id: string
  max_activations: number
  name: string
  expires_at?: Date
}

export async function getLicenses(): Promise<License[]> {
  const cookieStore = await cookies();
  const session = cookieStore.get("selfservice_session")?.value;

  if (!session) return [];
  const res = await fetch(`${process.env.BACKEND_URL}/api/v1/self-service/licenses`, {
    headers: {
      "Content-Type": "application/json",
       Cookie: `selfservice_session=${session}`,
      Accept: "application/json",
    },
    cache: "no-store",
    credentials: "include"
  });
  if (!res.ok)
    return []
  const json = await res.json()
  return json as License[]
}
