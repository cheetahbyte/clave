import { render, toPlainText } from "react-email"
import LicenseCreated from "./emails/license-created.js"
import LicenseReplaced from "./emails/license-replaced.js"
import MagicLink from "./emails/magic-link.js"
import OrganizationInvite from "./emails/organization-invite.js"

export type EmailEvent =
  | {
      type: "license.created"
      email: string
      data: {
        licenseKey: string
        productName?: string
        portalLink?: string
        isTrial?: boolean
      }
    }
  | {
      type: "license.replaced"
      email: string
      data: {
        licenseKey: string
        productName?: string
        portalLink?: string
      }
    }
  | {
      type: "selfservice.magic_link"
      email: string
      data: {
        link: string
      }
    }
  | {
      type: "organization.invite"
      email: string
      data: {
        orgName?: string
        link: string
      }
    }

export type Rendered = { subject: string; html: string; text: string }

export async function renderEmail(event: EmailEvent): Promise<Rendered> {
  switch (event.type) {
    case "license.created": {
      const d = event.data
      if (!d.licenseKey) throw new Error("license.created requires licenseKey")

      const component = LicenseCreated({
        licenseKey: d.licenseKey,
        productName: d.productName,
        portalLink: d.portalLink,
        isTrial: d.isTrial,
      })

      const subject = d.isTrial
        ? "Your Clave trial key"
        : "Your Clave license key"

      const html = await render(component)
      const text = toPlainText(html)
      return { subject, html, text }
    }

    case "license.replaced": {
      const d = event.data
      if (!d.licenseKey) throw new Error("license.replaced requires licenseKey")

      const component = LicenseReplaced({
        licenseKey: d.licenseKey,
        productName: d.productName,
        portalLink: d.portalLink,
      })

      const product = d.productName ? ` for ${d.productName}` : ""
      const subject = `Your Clave license key has been replaced${product}`

      const html = await render(component)
      const text = toPlainText(html)
      return { subject, html, text }
    }

    case "selfservice.magic_link": {
      const d = event.data
      if (!d.link) throw new Error("selfservice.magic_link requires link")

      const component = MagicLink({ link: d.link })

      const html = await render(component)
      const text = toPlainText(html)
      return { subject: "Your Clave sign-in link", html, text }
    }

    case "organization.invite": {
      const d = event.data
      if (!d.link) throw new Error("organization.invite requires link")

      const component = OrganizationInvite({
        orgName: d.orgName,
        link: d.link,
      })

      const heading = d.orgName ? d.orgName : "Clave"
      const subject = `You've been invited to ${heading}`

      const html = await render(component)
      const text = toPlainText(html)
      return { subject, html, text }
    }
  }
}
