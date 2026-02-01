
import type { Meta, StoryObj } from '@storybook/nextjs-vite';
import SelfServiceLicenseCard from '@/components/selfservice/Card';

const meta: Meta<typeof SelfServiceLicenseCard> = {
  title: 'License/SelfServiceLicenseCard',
  component: SelfServiceLicenseCard,
  args: {
    activations: { current: 3, max: 5 },
    expiryDate: new Date(),
    maskedKey: "••••-••••-••••-••••-••••",
    product: "Hondicard",
    status: "active"
  },
};

export default meta;

type Story = StoryObj<typeof SelfServiceLicenseCard>;

export const Default: Story = {};
