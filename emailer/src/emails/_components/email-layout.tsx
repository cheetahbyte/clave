import {
  Html,
  Head,
  Preview,
  Body,
  Container,
  Section,
  Text,
  Heading,
  Hr,
} from "react-email"

const FONT =
  "-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif"

interface EmailLayoutProps {
  preview: string
  heading: string
  children: React.ReactNode
  footer?: string
  brandName?: string
}

export function EmailLayout({
  preview,
  heading,
  children,
  footer,
  brandName = "Clave",
}: EmailLayoutProps) {
  const badge = brandName.trim().charAt(0).toUpperCase() || "C"
  return (
    <Html lang="en">
      <Head />
      <Preview>{preview}</Preview>
      <Body style={body}>
        <Container style={outer}>
          <Section style={card}>
            <Section style={logoRow}>
              <table>
                <tr>
                  <td style={logoBadge}>{badge}</td>
                  <td style={logoText}>{brandName}</td>
                </tr>
              </table>
            </Section>
            <Section style={content}>
              <Heading as="h1" style={h1}>
                {heading}
              </Heading>
              {children}
              {footer ? (
                <>
                  <Hr style={hr} />
                  <Text style={footerText}>{footer}</Text>
                </>
              ) : null}
            </Section>
          </Section>
          <Text style={sentBy}>Sent by {brandName} · License management</Text>
        </Container>
      </Body>
    </Html>
  )
}

const body = {
  margin: 0,
  padding: 0,
  backgroundColor: "#f4f4f5",
  fontFamily: FONT,
}

const outer = {
  margin: "0 auto",
  padding: "32px 16px",
  maxWidth: 512,
}

const card = {
  backgroundColor: "#ffffff",
  borderRadius: 12,
  border: "1px solid #e4e4e7",
}

const logoRow = {
  padding: "28px 32px 8px",
}

const logoBadge = {
  width: 36,
  height: 36,
  backgroundColor: "#18181b",
  borderRadius: 8,
  textAlign: "center" as const,
  color: "#ffffff",
  fontSize: 18,
  fontWeight: 700,
}

const logoText = {
  paddingLeft: 12,
  fontSize: 16,
  fontWeight: 600,
  color: "#18181b",
}

const content = {
  padding: "8px 32px 24px",
}

const h1 = {
  margin: "16px 0 8px",
  fontSize: 22,
  lineHeight: 1.3,
  color: "#18181b",
}

const hr = {
  margin: "16px 0",
  borderColor: "#e4e4e7",
}

const footerText = {
  margin: 0,
  fontSize: 12,
  lineHeight: 1.5,
  color: "#a1a1aa",
}

const sentBy = {
  margin: "16px 0 0",
  fontSize: 11,
  color: "#a1a1aa",
  textAlign: "center" as const,
}
