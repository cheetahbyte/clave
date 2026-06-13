import { Section, Text, Button } from "react-email"
import { EmailLayout } from "./_components/email-layout.js"

interface LicenseCreatedProps {
  licenseKey: string
  productName?: string
  portalLink?: string
  isTrial?: boolean
}

export default function LicenseCreated({
  licenseKey,
  productName,
  portalLink,
  isTrial,
}: LicenseCreatedProps) {
  const noun = isTrial ? "trial" : "license"
  const heading = productName
    ? `Your ${productName} ${noun}`
    : `Your ${noun} is ready`
  const intro = productName
    ? `Your ${noun} key for ${productName} is below.`
    : `Thanks for trying us out. Your ${noun} key is below.`
  const preview = `Your new ${noun} key${productName ? ` for ${productName}` : ""}`

  return (
    <EmailLayout
      preview={preview}
      heading={heading}
      footer="Didn't expect this email? You can safely ignore it."
      brandName={productName}
    >
      <Text style={body}>{intro}</Text>
      <Text style={body}>
        Keep this key somewhere safe — you&apos;ll need it to activate the
        product.
      </Text>
      {isTrial ? (
        <Text style={body}>
          This is a trial and will expire automatically. Reach out if
          you&apos;d like to continue afterwards.
        </Text>
      ) : null}
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

LicenseCreated.PreviewProps = {
  licenseKey: "CLAVE-XXXX-YYYY-ZZZZ",
  productName: "Acme Pro",
  portalLink: "https://app.clave.dev/portal",
  isTrial: false,
} as LicenseCreatedProps

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
