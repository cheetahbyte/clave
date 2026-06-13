import { Section, Text, Button } from "react-email"
import { EmailLayout } from "./_components/email-layout.js"

interface OrganizationInviteProps {
  orgName?: string
  link: string
}

export default function OrganizationInvite({
  orgName,
  link,
}: OrganizationInviteProps) {
  const heading = orgName ? `Join ${orgName}` : "You've been invited"
  const intro = orgName
    ? `You've been invited to join ${orgName} on Clave.`
    : "You've been invited to join an organization on Clave."
  const preview = intro

  return (
    <EmailLayout
      preview={preview}
      heading={heading}
      footer="If you weren't expecting this invitation, you can ignore this email."
      brandName={orgName}
    >
      <Text style={body}>{intro}</Text>
      <Text style={body}>
        Accept the invite to get access. This link expires in 7 days.
      </Text>
      {safeUrl(link) ? (
        <>
          <Button href={link} style={button}>
            Accept invite
          </Button>
          <Text style={urlFallback}>Or paste this link into your browser:</Text>
          <Text style={urlFallbackLink}>{link}</Text>
        </>
      ) : null}
    </EmailLayout>
  )
}

OrganizationInvite.PreviewProps = {
  orgName: "Acme Corp",
  link: "https://app.clave.dev/invite/abc123",
} as OrganizationInviteProps

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
