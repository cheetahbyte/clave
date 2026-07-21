import { Section, Text } from "react-email";
import { EmailLayout } from "./_components/email-layout.js";

interface AdminTwoFactorCodeProps {
	code: string;
	ttlMinutes: number;
}

export default function AdminTwoFactorCode({
	code,
	ttlMinutes,
}: AdminTwoFactorCodeProps) {
	return (
		<EmailLayout
			preview="Your Clave verification code."
			heading="Your verification code"
			footer="If you didn't try to sign in, change your password — someone else may know it."
		>
			<Text style={body}>
				Enter this code to finish signing in. It expires in {ttlMinutes} minutes
				and can only be used once.
			</Text>
			<Section style={codeBlock}>{code}</Section>
		</EmailLayout>
	);
}

AdminTwoFactorCode.PreviewProps = {
	code: "123456",
	ttlMinutes: 10,
} as AdminTwoFactorCodeProps;

const body = {
	fontSize: 14,
	lineHeight: 1.6,
	color: "#52525b",
	margin: "0 0 16px 0",
};

const codeBlock = {
	margin: "0 0 20px",
	padding: "14px 16px",
	backgroundColor: "#f4f4f5",
	border: "1px solid #e4e4e7",
	borderRadius: 8,
	fontFamily: "ui-monospace,Menlo,Consolas,monospace",
	fontSize: 24,
	fontWeight: 700,
	letterSpacing: 4,
	color: "#18181b",
	textAlign: "center" as const,
};
