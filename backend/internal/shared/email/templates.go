package email

import (
	"bytes"
	"fmt"
	"html/template"
)

type templateData struct {
	Preheader   string
	Heading     string
	Paragraphs  []string
	ButtonText  string
	ButtonURL   string
	FallbackURL string
	CodeBlock   string
	FooterNote  string
}

const baseLayout = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="color-scheme" content="light dark" />
    <title>{{ .Heading }}</title>
  </head>
  <body style="margin:0;padding:0;background-color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
    <span style="display:none;max-height:0;overflow:hidden;opacity:0;">{{ .Preheader }}</span>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f5;padding:32px 16px;">
      <tr>
        <td align="center">
          <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:480px;background-color:#ffffff;border-radius:12px;border:1px solid #e4e4e7;overflow:hidden;">
            <tr>
              <td style="padding:28px 32px 8px 32px;">
                <table role="presentation" cellpadding="0" cellspacing="0">
                  <tr>
                    <td style="width:36px;height:36px;background-color:#18181b;border-radius:8px;text-align:center;vertical-align:middle;color:#ffffff;font-size:18px;font-weight:700;">C</td>
                    <td style="padding-left:12px;font-size:16px;font-weight:600;color:#18181b;">Clave</td>
                  </tr>
                </table>
              </td>
            </tr>
            <tr>
              <td style="padding:8px 32px 24px 32px;">
                <h1 style="margin:16px 0 8px 0;font-size:22px;line-height:1.3;color:#18181b;">{{ .Heading }}</h1>
                {{ range .Paragraphs }}
                <p style="margin:0 0 16px 0;font-size:14px;line-height:1.6;color:#52525b;">{{ . }}</p>
                {{ end }}
                {{ if .CodeBlock }}
                <div style="margin:0 0 20px 0;padding:14px 16px;background-color:#f4f4f5;border:1px solid #e4e4e7;border-radius:8px;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:14px;color:#18181b;word-break:break-all;">{{ .CodeBlock }}</div>
                {{ end }}
                {{ if .ButtonURL }}
                <table role="presentation" cellpadding="0" cellspacing="0" style="margin:24px 0;">
                  <tr>
                    <td style="border-radius:8px;background-color:#18181b;">
                      <a href="{{ .ButtonURL }}" target="_blank" style="display:inline-block;padding:12px 24px;font-size:14px;font-weight:600;color:#ffffff;text-decoration:none;border-radius:8px;">{{ .ButtonText }}</a>
                    </td>
                  </tr>
                </table>
                {{ if .FallbackURL }}
                <p style="margin:0 0 4px 0;font-size:12px;color:#a1a1aa;">Or paste this link into your browser:</p>
                <p style="margin:0 0 16px 0;font-size:12px;line-height:1.5;word-break:break-all;"><a href="{{ .FallbackURL }}" target="_blank" style="color:#3f3f46;">{{ .FallbackURL }}</a></p>
                {{ end }}
                {{ end }}
                {{ if .FooterNote }}
                <p style="margin:16px 0 0 0;font-size:12px;line-height:1.5;color:#a1a1aa;">{{ .FooterNote }}</p>
                {{ end }}
              </td>
            </tr>
          </table>
          <p style="margin:16px 0 0 0;font-size:11px;color:#a1a1aa;">Sent by Clave · License management</p>
        </td>
      </tr>
    </table>
  </body>
</html>`

var baseTemplate = template.Must(template.New("email").Parse(baseLayout))

func render(d templateData) (string, error) {
	var buf bytes.Buffer
	if err := baseTemplate.Execute(&buf, d); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func MagicLinkEmail(link string) (Message, error) {
	html, err := render(templateData{
		Preheader:   "Your secure sign-in link for Clave self-service.",
		Heading:     "Sign in to self-service",
		Paragraphs:  []string{"Click the button below to view and manage your licenses. This link expires in 15 minutes and can only be used once."},
		ButtonText:  "View my licenses",
		ButtonURL:   link,
		FallbackURL: link,
		FooterNote:  "If you didn't request this, you can safely ignore this email.",
	})
	return Message{subject: "Your Clave sign-in link", html: html}, err
}

func InviteEmail(orgName, link string) (Message, error) {
	heading := "You've been invited"
	intro := "You've been invited to join an organization on Clave."
	if orgName != "" {
		heading = "Join " + orgName
		intro = "You've been invited to join " + orgName + " on Clave."
	}
	html, err := render(templateData{
		Preheader:   intro,
		Heading:     heading,
		Paragraphs:  []string{intro, "Accept the invite to get access. This link expires in 7 days."},
		ButtonText:  "Accept invite",
		ButtonURL:   link,
		FallbackURL: link,
		FooterNote:  "If you weren't expecting this invitation, you can ignore this email.",
	})
	return Message{subject: "You've been invited to Clave", html: html}, err
}

func TwoFactorCodeEmail(code string, ttlMinutes int) (Message, error) {
	html, err := render(templateData{
		Preheader:  "Your Clave verification code.",
		Heading:    "Your verification code",
		Paragraphs: []string{fmt.Sprintf("Enter this code to finish signing in. It expires in %d minutes and can only be used once.", ttlMinutes)},
		CodeBlock:  code,
		FooterNote: "If you didn't try to sign in, change your password — someone else may know it.",
	})
	return Message{subject: "Your Clave verification code", html: html}, err
}

func LicenseCreatedEmail(productName, licenseKey, portalLink string, isTrial bool) (Message, error) {
	noun := "license"
	if isTrial {
		noun = "trial"
	}
	heading := "Your " + noun + " is ready"
	intro := "Thanks for trying us out. Your " + noun + " key is below."
	if productName != "" {
		heading = "Your " + productName + " " + noun
		intro = "Your " + noun + " key for " + productName + " is below."
	}
	paragraphs := []string{intro, "Keep this key somewhere safe — you'll need it to activate the product."}
	if isTrial {
		paragraphs = append(paragraphs, "This is a trial and will expire automatically. Reach out if you'd like to continue afterwards.")
	}
	data := templateData{
		Preheader:  "Your new " + noun + " key for " + productName,
		Heading:    heading,
		Paragraphs: paragraphs,
		CodeBlock:  licenseKey,
		FooterNote: "Didn't expect this email? You can safely ignore it.",
	}
	if portalLink != "" {
		data.ButtonText = "Manage your licenses"
		data.ButtonURL = portalLink
		data.FallbackURL = portalLink
	}
	html, err := render(data)
	subject := "Your Clave license key"
	if isTrial {
		subject = "Your Clave trial key"
	}
	return Message{subject: subject, html: html}, err
}
