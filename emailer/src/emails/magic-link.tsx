import { Section, Text, Button } from "react-email"
import { EmailLayout } from "./_components/email-layout.js"

interface MagicLinkProps {
  link: string
  brandName?: string
}

export default function MagicLink({ link, brandName }: MagicLinkProps) {
  return (
    <EmailLayout
      preview="Your secure sign-in link for Clave self-service."
      heading="Sign in to self-service"
      footer="If you didn't request this, you can safely ignore this email."
      brandName={brandName}
    >
      <Text style={body}>
        Click the button below to view and manage your licenses. This link
        expires in 15 minutes and can only be used once.
      </Text>
      {safeUrl(link) ? (
        <>
          <Button href={link} style={button}>
            View my licenses
          </Button>
          <Text style={urlFallback}>Or paste this link into your browser:</Text>
          <Text style={urlFallbackLink}>{link}</Text>
        </>
      ) : null}
    </EmailLayout>
  )
}

MagicLink.PreviewProps = {
  link: "https://app.clave.dev/selfservice/acme/auth?token=abc123",
} as MagicLinkProps

function safeUrl(u: string): string | null {
  try {
    const { protocol } = new URL(u)
    return protocol === "http:" || protocol === "https:" ? u : null
  } catch {
    return null
  }
}

const body = {
  fontSize: 14,
  lineHeight: 1.6,
  color: "#52525b",
  margin: "0 0 16px 0",
}

const button = {
  display: "inline-block",
  padding: "12px 24px",
  fontSize: 14,
  fontWeight: 600,
  color: "#ffffff",
  backgroundColor: "#18181b",
  borderRadius: 8,
  marginTop: 8,
  marginBottom: 16,
}

const urlFallback = {
  margin: "0 0 4px",
  fontSize: 12,
  lineHeight: 1.5,
  color: "#a1a1aa",
}

const urlFallbackLink = {
  margin: "0 0 16px",
  fontSize: 12,
  lineHeight: 1.5,
  color: "#3f3f46",
  wordBreak: "break-all" as const,
}
