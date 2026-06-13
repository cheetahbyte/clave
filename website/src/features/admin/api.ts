import { adminFetch, publicFetch } from "@/lib/api";

export interface AdminLoginResponse {
  id: string;
  email: string;
  role: string;
  organizationId: string;
  organizationName: string;
  mfaEnabled: boolean;
  mfaVerified: boolean;
  mfaSetupRequired?: boolean;
  mfaVerificationRequired?: boolean;
  created_at: string;
}

export interface AdminProfile {
  id: string;
  email: string;
  role: string;
  organizationId: string;
  organizationName: string;
  mfaEnabled: boolean;
  mfaVerified: boolean;
  last_login_at: string | null;
  created_at: string;
}

export interface SetupStartResponse {
  otpauthUrl: string;
  secret: string;
}

interface LoginRequest {
  email: string;
  password: string;
}

export async function loginAdmin(data: LoginRequest): Promise<AdminLoginResponse> {
  return adminFetch<AdminLoginResponse>("/api/v1/admin/auth/login", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function logoutAdmin(): Promise<{ ok: boolean }> {
  return adminFetch<{ ok: boolean }>("/api/v1/admin/auth/logout", {
    method: "POST",
  });
}

export async function getCurrentAdmin(): Promise<AdminProfile> {
  return adminFetch<AdminProfile>("/api/v1/admin/auth/me");
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  role: string;
  createdAt: string;
}

export function listOrganizations(): Promise<Organization[]> {
  return adminFetch<Organization[]>("/api/v1/admin/organizations");
}

export function createOrganization(name: string): Promise<Organization> {
  return adminFetch<Organization>("/api/v1/admin/organizations", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export function switchOrganization(organizationId: string): Promise<Organization> {
  return adminFetch<Organization>("/api/v1/admin/organizations/switch", {
    method: "POST",
    body: JSON.stringify({ organizationId }),
  });
}

export interface OrgMember {
  id: string;
  email: string;
  role: string;
  createdAt: string;
  lastLoginAt: string | null;
}

export interface OrgInvite {
  id: string;
  email: string;
  role: string;
  expiresAt: string;
  createdAt: string;
}

export interface MembersResponse {
  members: OrgMember[];
  invites: OrgInvite[];
}

export interface InviteResponse {
  invite: OrgInvite;
  token?: string;
}

export interface InvitePreview {
  valid: boolean;
  organizationName?: string;
  email?: string;
  requiresPassword: boolean;
}

export function listMembers(): Promise<MembersResponse> {
  return adminFetch<MembersResponse>("/api/v1/admin/organizations/members");
}

export function inviteMember(email: string, role: string): Promise<InviteResponse> {
  return adminFetch<InviteResponse>("/api/v1/admin/organizations/invites", {
    method: "POST",
    body: JSON.stringify({ email, role }),
  });
}

export function deleteInvite(id: string): Promise<{ ok: boolean }> {
  return adminFetch<{ ok: boolean }>(`/api/v1/admin/organizations/invites/${id}`, {
    method: "DELETE",
  });
}

export function removeMember(memberId: string): Promise<{ ok: boolean }> {
  return adminFetch<{ ok: boolean }>(`/api/v1/admin/organizations/members/${memberId}`, {
    method: "DELETE",
  });
}

// Public invite accept flow (no admin session).
export function previewInvite(token: string): Promise<InvitePreview> {
  return publicFetch<InvitePreview>(
    `/api/v1/admin/invites/accept?token=${encodeURIComponent(token)}`,
  );
}

export function acceptInvite(token: string, password?: string): Promise<{ ok: boolean }> {
  return publicFetch<{ ok: boolean }>("/api/v1/admin/invites/accept", {
    method: "POST",
    body: JSON.stringify({ token, password: password ?? "" }),
  });
}

export async function start2FASetup(): Promise<SetupStartResponse> {
  return adminFetch<SetupStartResponse>("/api/v1/admin/auth/2fa/setup/start", {
    method: "POST",
  });
}

export async function verify2FASetup(code: string): Promise<{ ok: boolean }> {
  return adminFetch<{ ok: boolean }>("/api/v1/admin/auth/2fa/setup/verify", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
}

export async function verify2FA(code: string): Promise<{ ok: boolean }> {
  return adminFetch<{ ok: boolean }>("/api/v1/admin/auth/2fa/verify", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
}

export interface AdminOverview {
  totalLicenses: number;
  activeLicenses: number;
  expiredLicenses: number;
  totalProducts: number;
  totalActivations: number;
  totalTrials: number;
  activeTrials: number;
  recentLicenses: AdminLicenseItem[];
}

export interface AdminLicenseItem {
  id: string;
  customerEmail: string;
  productName: string;
  isActive: boolean;
  isTrial: boolean;
  maxActivations: number;
  activationCount: number;
  createdAt: string | null;
  expiresAt: string | null;
}

export interface AdminLicenseListResponse {
  items: AdminLicenseItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface AdminLicenseDetail {
  id: string;
  customerEmail: string;
  productName: string;
  productId: string;
  isActive: boolean;
  isTrial: boolean;
  maxActivations: number;
  activationCount: number;
  features: string[];
  createdAt: string | null;
  expiresAt: string | null;
  activations: AdminActivationItem[];
}

export interface AdminActivationItem {
  id: string;
  deviceId: string;
  hostname: string | null;
  createdAt: string | null;
  checkedInAt: string | null;
}

export interface AdminProductItem {
  id: string;
  name: string;
  version: string | null;
  logoUrl: string | null;
  createdAt: string | null;
}

export interface UpdateLicenseRequest {
  customerEmail: string;
  maxActivations: number;
  isActive: boolean;
  expiresAt: string | null;
  features: string[];
}

export function createProduct(name: string, version: string | null, logoUrl: string | null): Promise<AdminProductItem> {
  return adminFetch<AdminProductItem>("/api/v1/admin/products", {
    method: "POST",
    body: JSON.stringify({ name, version, logoUrl }),
  });
}

export function updateProduct(
  id: string,
  name: string,
  version: string | null,
  logoUrl: string | null,
): Promise<AdminProductItem> {
  return adminFetch<AdminProductItem>(`/api/v1/admin/products/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ name, version, logoUrl }),
  });
}

export function updateLicense(
  id: string,
  data: UpdateLicenseRequest,
): Promise<AdminLicenseDetail> {
  return adminFetch<AdminLicenseDetail>(`/api/v1/admin/licenses/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export function deleteLicense(id: string): Promise<{ ok: boolean }> {
  return adminFetch<{ ok: boolean }>(`/api/v1/admin/licenses/${id}`, {
    method: "DELETE",
  });
}

export function deleteProduct(id: string): Promise<{ ok: boolean }> {
  return adminFetch<{ ok: boolean }>(`/api/v1/admin/products/${id}`, {
    method: "DELETE",
  });
}

export interface AuditLogItem {
  id: string;
  action: string;
  resourceType: string;
  resourceId: string | null;
  adminEmail: string | null;
  ip: string | null;
  userAgent: string | null;
  createdAt: string | null;
}

export interface AuditLogListResponse {
  items: AuditLogItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface AuditLogFilters {
  q?: string;
  action?: string;
  resourceType?: string;
  adminEmail?: string;
  from?: string;
  to?: string;
}

export function listAuditLogs(
  page = 1,
  pageSize = 50,
  filters?: AuditLogFilters,
): Promise<AuditLogListResponse> {
  const params = new URLSearchParams();
  params.set("page", String(page));
  params.set("pageSize", String(pageSize));
  if (filters?.q) params.set("q", filters.q);
  if (filters?.action) params.set("action", filters.action);
  if (filters?.resourceType) params.set("resourceType", filters.resourceType);
  if (filters?.adminEmail) params.set("adminEmail", filters.adminEmail);
  if (filters?.from) params.set("from", filters.from);
  if (filters?.to) params.set("to", filters.to);
  return adminFetch<AuditLogListResponse>(
    `/api/v1/admin/audit-logs?${params.toString()}`,
  );
}

export interface CreateLicenseRequest {
  productId: string;
  maxActivations: number;
  customerEmail: string;
  isTrial?: boolean;
  trialDays?: number;
  sendEmail?: boolean;
}

export interface CreateLicenseResponse {
  licenseKey: string;
  productName: string;
  isTrial: boolean;
}

export function getAdminOverview(productId?: string): Promise<AdminOverview> {
  const qs = productId ? `?productId=${encodeURIComponent(productId)}` : "";
  return adminFetch<AdminOverview>(`/api/v1/admin/overview${qs}`);
}

export interface TimeseriesPoint {
  date: string;
  licenses: number;
  trials: number;
  activations: number;
}

export function getAdminTimeseries(days = 30, productId?: string): Promise<TimeseriesPoint[]> {
  const params = new URLSearchParams({ days: String(days) });
  if (productId) params.set("productId", productId);
  return adminFetch<TimeseriesPoint[]>(`/api/v1/admin/stats/timeseries?${params.toString()}`);
}

export interface ListTrialsParams {
  q?: string;
  status?: string;
  productId?: string;
}

export function listAdminTrials(params?: ListTrialsParams): Promise<AdminLicenseItem[]> {
  const searchParams = new URLSearchParams();
  if (params?.q) searchParams.set("q", params.q);
  if (params?.status) searchParams.set("status", params.status);
  if (params?.productId) searchParams.set("productId", params.productId);
  const qs = searchParams.toString();
  return adminFetch<AdminLicenseItem[]>(`/api/v1/admin/trials${qs ? `?${qs}` : ""}`);
}

export interface ListLicensesParams {
  q?: string;
  status?: string;
  type?: string;
  productId?: string;
  page?: number;
  pageSize?: number;
}

export function listAdminLicenses(params?: ListLicensesParams): Promise<AdminLicenseListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.q) searchParams.set("q", params.q);
  if (params?.status) searchParams.set("status", params.status);
  if (params?.type) searchParams.set("type", params.type);
  if (params?.productId) searchParams.set("productId", params.productId);
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.pageSize) searchParams.set("pageSize", String(params.pageSize));
  const qs = searchParams.toString();
  return adminFetch<AdminLicenseListResponse>(`/api/v1/admin/licenses${qs ? `?${qs}` : ""}`);
}

export function getAdminLicense(id: string): Promise<AdminLicenseDetail> {
  return adminFetch<AdminLicenseDetail>(`/api/v1/admin/licenses/${id}`);
}

export function createLicense(data: CreateLicenseRequest): Promise<CreateLicenseResponse> {
  return adminFetch<CreateLicenseResponse>("/api/v1/admin/licenses", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export function listAdminProducts(): Promise<AdminProductItem[]> {
  return adminFetch<AdminProductItem[]>("/api/v1/admin/products");
}

export function getAdminProduct(id: string): Promise<AdminProductItem> {
  return adminFetch<AdminProductItem>(`/api/v1/admin/products/${encodeURIComponent(id)}`);
}

export interface ProductUpdateConfigDTO {
  id: string;
  productId: string;
  platform: string;
  channel: string;
  channelId: string;
  providerKey: string;
  enabled: boolean;
  config: Record<string, unknown>;
  feedUrl?: string;
}

export function listProductUpdateConfigs(productId: string): Promise<ProductUpdateConfigDTO[]> {
  return adminFetch<ProductUpdateConfigDTO[]>(
    `/api/v1/admin/products/${encodeURIComponent(productId)}/update-configs`,
  );
}

export function saveProductUpdateConfig(
  productId: string,
  data: { platform: string; channel: string; providerKey: string; enabled: boolean; config: Record<string, unknown> },
): Promise<ProductUpdateConfigDTO> {
  return adminFetch<ProductUpdateConfigDTO>(
    `/api/v1/admin/products/${encodeURIComponent(productId)}/update-configs`,
    {
      method: "POST",
      body: JSON.stringify(data),
    },
  );
}

export function deleteProductUpdateConfig(configId: string): Promise<{ ok: string }> {
  return adminFetch<{ ok: string }>(
    `/api/v1/admin/update-configs/${encodeURIComponent(configId)}`,
    { method: "DELETE" },
  );
}

export interface StorageConfigDTO {
  productId: string;
  backend: string;
  config: Record<string, unknown>;
}

export function getProductStorageConfig(productId: string): Promise<StorageConfigDTO> {
  return adminFetch<StorageConfigDTO>(
    `/api/v1/admin/products/${encodeURIComponent(productId)}/storage-config`,
  );
}

export function saveProductStorageConfig(
  productId: string,
  data: { backend: string; config: Record<string, unknown> },
): Promise<StorageConfigDTO> {
  return adminFetch<StorageConfigDTO>(
    `/api/v1/admin/products/${encodeURIComponent(productId)}/storage-config`,
    {
      method: "PUT",
      body: JSON.stringify(data),
    },
  );
}

export function testProductStorageConfig(
  productId: string,
  data: { backend: string; config: Record<string, unknown> },
): Promise<{ ok: boolean; error?: string }> {
  return adminFetch<{ ok: boolean; error?: string }>(
    `/api/v1/admin/products/${encodeURIComponent(productId)}/storage-config/test`,
    {
      method: "POST",
      body: JSON.stringify(data),
    },
  );
}

export interface ReleaseArtifactDTO {
  type: string;
  url: string;
  arch: string;
  sizeBytes?: number;
  sha256?: string;
  signature: string;
  filename?: string;
  mimeType?: string;
}

export interface ReleaseDTO {
  id: string;
  productId: string;
  productName: string;
  channel: string;
  channelId: string;
  platform: string;
  version: string;
  buildNumber?: string;
  status: string;
  releaseNotes?: string;
  changelogId?: string;
  publishedAt?: string;
  createdAt?: string;
  artifacts: ReleaseArtifactDTO[];
}

export interface ArtifactFullDTO {
  id: string;
  releaseId: string;
  artifactType: string;
  os: string;
  arch: string;
  url: string;
  sizeBytes?: number;
  checksumSha256?: string;
  signature?: string;
  filename?: string;
  mimeType?: string;
  minimumSystemVersion?: string;
}

export function listReleases(params?: {
  limit?: number;
  offset?: number;
  productId?: string;
}): Promise<ReleaseDTO[]> {
  const searchParams = new URLSearchParams();
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.offset) searchParams.set("offset", String(params.offset));
  if (params?.productId) searchParams.set("productId", params.productId);
  const qs = searchParams.toString();
  return adminFetch<ReleaseDTO[]>(`/api/v1/admin/update-releases${qs ? `?${qs}` : ""}`);
}

export function createRelease(data: {
  productId: string;
  platform: string;
  channel: string;
  version: string;
  buildNumber?: string;
  releaseNotes?: string;
  changelogId?: string;
}): Promise<ReleaseDTO> {
  return adminFetch<ReleaseDTO>("/api/v1/admin/update-releases", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export interface ChannelDTO {
  id: string;
  productId: string;
  name: string;
  isDefault: boolean;
  requiredFeatures: string[];
  description?: string;
}

export function listChannels(productId: string): Promise<ChannelDTO[]> {
  return adminFetch<ChannelDTO[]>(
    `/api/v1/admin/products/${encodeURIComponent(productId)}/channels`,
  );
}

export function createChannel(
  productId: string,
  data: { name: string; isDefault: boolean; requiredFeatures: string[]; description?: string },
): Promise<ChannelDTO> {
  return adminFetch<ChannelDTO>(
    `/api/v1/admin/products/${encodeURIComponent(productId)}/channels`,
    { method: "POST", body: JSON.stringify(data) },
  );
}

export function updateChannel(
  channelId: string,
  data: { name: string; isDefault: boolean; requiredFeatures: string[]; description?: string },
): Promise<ChannelDTO> {
  return adminFetch<ChannelDTO>(
    `/api/v1/admin/channels/${encodeURIComponent(channelId)}`,
    { method: "PATCH", body: JSON.stringify(data) },
  );
}

export function deleteChannel(channelId: string): Promise<{ ok: string }> {
  return adminFetch<{ ok: string }>(
    `/api/v1/admin/channels/${encodeURIComponent(channelId)}`,
    { method: "DELETE" },
  );
}

export function uploadArtifact(
  releaseId: string,
  file: File,
  artifactType: string,
  os: string,
  arch: string,
  metadata?: string,
): Promise<ArtifactFullDTO> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("artifactType", artifactType);
  formData.append("os", os);
  formData.append("arch", arch);
  if (metadata) {
    formData.append("metadata", metadata);
  }

  return adminFetch<ArtifactFullDTO>(
    `/api/v1/admin/update-releases/${encodeURIComponent(releaseId)}/artifacts`,
    {
      method: "POST",
      body: formData,
    },
  );
}

export function publishRelease(releaseId: string): Promise<ReleaseDTO> {
  return adminFetch<ReleaseDTO>(
    `/api/v1/admin/update-releases/${encodeURIComponent(releaseId)}/publish`,
    { method: "POST" },
  );
}

export function yankRelease(releaseId: string): Promise<ReleaseDTO> {
  return adminFetch<ReleaseDTO>(
    `/api/v1/admin/update-releases/${encodeURIComponent(releaseId)}/yank`,
    { method: "POST" },
  );
}

export function deleteRelease(releaseId: string): Promise<{ ok: string }> {
  return adminFetch<{ ok: string }>(
    `/api/v1/admin/update-releases/${encodeURIComponent(releaseId)}`,
    { method: "DELETE" },
  );
}

export interface ChangelogDTO {
  id: string;
  productId: string;
  title: string;
  body: string;
  createdAt?: string;
  updatedAt?: string;
}

export function listChangelogs(productId: string): Promise<ChangelogDTO[]> {
  return adminFetch<ChangelogDTO[]>(
    `/api/v1/admin/products/${encodeURIComponent(productId)}/changelogs`,
  );
}

export function createChangelog(
  productId: string,
  data: { title: string; body: string },
): Promise<ChangelogDTO> {
  return adminFetch<ChangelogDTO>(
    `/api/v1/admin/products/${encodeURIComponent(productId)}/changelogs`,
    { method: "POST", body: JSON.stringify(data) },
  );
}

export function updateChangelog(
  changelogId: string,
  data: { title: string; body: string },
): Promise<ChangelogDTO> {
  return adminFetch<ChangelogDTO>(
    `/api/v1/admin/changelogs/${encodeURIComponent(changelogId)}`,
    { method: "PATCH", body: JSON.stringify(data) },
  );
}

export function deleteChangelog(changelogId: string): Promise<{ ok: string }> {
  return adminFetch<{ ok: string }>(
    `/api/v1/admin/changelogs/${encodeURIComponent(changelogId)}`,
    { method: "DELETE" },
  );
}

export function attachReleaseChangelog(
  releaseId: string,
  changelogId: string | null,
): Promise<{ ok: string }> {
  return adminFetch<{ ok: string }>(
    `/api/v1/admin/update-releases/${encodeURIComponent(releaseId)}/changelog`,
    {
      method: "PATCH",
      body: JSON.stringify({ changelogId }),
    },
  );
}

export interface AdminDeviceItem {
  deviceId: string;
  hostname: string | null;
  activationId: string;
  activatedAt: string | null;
  checkedInAt: string | null;
  licenseId: string;
  customerEmail: string;
  licenseActive: boolean;
  productId: string;
  productName: string;
}

export interface AdminDeviceListResponse {
  items: AdminDeviceItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ListAdminDevicesParams {
  q?: string;
  productId?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}

export function listAdminDevices(params?: ListAdminDevicesParams): Promise<AdminDeviceListResponse> {
  const sp = new URLSearchParams();
  if (params?.page) sp.set("page", String(params.page));
  if (params?.pageSize) sp.set("pageSize", String(params.pageSize));
  if (params?.q) sp.set("q", params.q);
  if (params?.productId) sp.set("productId", params.productId);
  if (params?.status) sp.set("status", params.status);
  const qs = sp.toString();
  return adminFetch<AdminDeviceListResponse>(`/api/v1/admin/devices${qs ? `?${qs}` : ""}`);
}

export function deleteAdminDevice(deviceId: string): Promise<{ ok: boolean }> {
  return adminFetch<{ ok: boolean }>(`/api/v1/admin/devices/${encodeURIComponent(deviceId)}`, {
    method: "DELETE",
  });
}
