import { db } from "@/db/drizzle";
import { betterAuth } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { nextCookies } from "better-auth/next-js";
import { magicLink } from "better-auth/plugins";
import transporter from "./mail";
import {account, user, verification,session } from "@/db/schema/auth-schema"

export const auth = betterAuth({
    database: drizzleAdapter(db, {
      provider: "pg",
      schema: {
        user,
        account,
        verification,
        session
      }
    }),
  plugins: [magicLink({
    sendMagicLink: async ({ email, token, url }, ctx) => {
        await transporter.sendMail({
                  from: '"My App" <noreply@example.com>',
                  to: email,
                  subject: "Your Magic Link",
                  text: `Click the link to sign in: ${url}`,
                  html: `
                    <div style="font-family: sans-serif;">
                      <h1>Sign in to Clave</h1>
                      <p>Click the button below to authenticate instantly.</p>
                      <a href="${url}" style="background: #000; color: #fff; padding: 10px 20px; text-decoration: none; border-radius: 5px;">
                        Verify Email
                      </a>
                      <p style="margin-top: 20px; color: #666;">Or copy this link: ${url}</p>
                    </div>
                  `,
                });
        }
    }), nextCookies()]
});
