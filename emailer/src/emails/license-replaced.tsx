import { Section, Text, Button } from "react-email"
import { EmailLayout } from "./_components/email-layout.js"

interface LicenseReplacedProps {
  licenseKey: string
  productName?: string
  portalLink?: string
}

export default function LicenseReplaced({
  licenseKey,
  productName,
  portalLink,
}: LicenseReplacedProps) {
  const heading = productName
    ? `Your ${productName} license has been replaced`
    : "Your license has been replaced"
  const preview = productName
    ? `New license key for ${productName}`
    : "Your new license key"

  return (
    <EmailLayout
      preview={preview}
      heading={heading}
      footer="Didn't expect this email? You can safely ignore it."
      brandName={productName}
    >
      <Text style={body}>
        Your old license key was revoked and replaced with a new one. Your new
        key is below.
      </Text>
      <Text style={body}>
        Keep this key somewhere safe — you&apos;ll need it to activate the
        product. The previous key no longer works.
      </Text>
      <Section style={codeBlock}>{licenseKey}</Section>
      {portalLink && safeUrl(portalLink) ? (
        <>
          <Button href={portalLink} style={button}>
            Manage your licenses
          </Button>
          <Text style={urlFallback}>Or paste this link into your browser:</Text>
          <Text style={urlFallbackLink}>{portalLink}</Text>
        </>
      ) : null}
    </EmailLayout>
  )
}

LicenseReplaced.PreviewProps = {
  licenseKey: "CLAVE-XXXX-YYYY-ZZZZ",
  productName: "Acme Pro",
  portalLink: "https://app.clave.dev/portal",
} as LicenseReplacedProps

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

const codeBlock = {
  margin: "0 0 20px",
  padding: "14px 16px",
  backgroundColor: "#f4f4f5",
  border: "1px solid #e4e4e7",
  borderRadius: 8,
  fontFamily: "ui-monospace,Menlo,Consolas,monospace",
  fontSize: 14,
  color: "#18181b",
  wordBreak: "break-all" as const,
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
