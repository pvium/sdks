export interface RequestOptions {
  accessToken?: string;
  apiKey?: string;
  headers?: Record<string, string>;
  signal?: AbortSignal;
  skipApiKey?: boolean;
}

export interface OAuthTokenData {
  accessToken: string;
  refreshToken?: string;
  expiresIn?: number;
  expiresAt?: string;
  tokenType?: string;
  [key: string]: unknown;
}

export interface OAuthTokenResponse {
  meta: ApiMeta;
  data: OAuthTokenData;
}

export interface OAuthSocialHandle {
  provider: string;
  handle: string;
  subject?: string;
  name?: string;
  email?: string;
  verifiedAt?: string;
  [key: string]: unknown;
}

export interface OAuthLinkedAccount {
  type?: string;
  username?: string;
  login?: string;
  handle?: string;
  profile?: {
    username?: string;
    login?: string;
    [key: string]: unknown;
  };
  [key: string]: unknown;
}

export interface OAuthUserInfo {
  id?: string;
  _id?: string;
  email?: string;
  handle?: string;
  socialHandles?: OAuthSocialHandle[];
  privyLinkedAccounts?: OAuthLinkedAccount[];
  authorizedWallets?: unknown[];
  [key: string]: unknown;
}

export interface OAuthUserInfoResponse {
  meta: ApiMeta;
  data: OAuthUserInfo;
}

export interface CreateInvoiceRequest {
  name: string;
  description: string;
  amount: number;
  dueDate: string;
  paymentChannels: { chain: string; currency: string }[];
  redirectUri: string;
}

export interface ApiMeta {
  statusCode: number;
  success: boolean;
  message?: string;
  developerMessage?: string;
}

export interface PaginationMeta {
  totalCount: number;
  perPage: number;
  current: number;
  currentPage?: string;
  next?: number;
  nextPage?: string;
}

export interface Quantity {
  amount: string;
  unit: string;
}

export interface InvoiceItem {
  name: string;
  price: number;
  quantity: Quantity;
  [key: string]: unknown;
}

export interface InstallmentPlanItem {
  amount: number;
  dueDate: string;
  [key: string]: unknown;
}

export interface InvoiceListItem {
  id: number;
  code: string;
  name: string;
  documentType?: string;
  contractType?: string;
  currencySymbol?: string;
  actualAmount?: number;
  totalPaid?: number;
  totalUnpaid?: number;
  plan?: InstallmentPlanItem[];
  items?: InvoiceItem[];
  [key: string]: unknown;
}

export interface CreateInvoiceData extends InvoiceListItem {
  url?: string;
}

export interface CreateInvoiceResponse {
  meta: ApiMeta;
  data: CreateInvoiceData;
}

export interface ListInvoicesResponse {
  meta: ApiMeta & { pagination?: PaginationMeta };
  data: InvoiceListItem[];
}

export interface InvoiceStatusInstallment {
  id: number;
  amount: number;
  dueDate: string;
  totalPaid: number;
  totalUnpaid: number;
  payments: unknown[];
  [key: string]: unknown;
}

export interface InvoiceStatusData {
  contractId: number;
  contractCode: string;
  contractName: string;
  currencySymbol: string;
  totalAmount: number;
  totalPaid: number;
  totalUnpaid: number;
  installments: InvoiceStatusInstallment[];
  [key: string]: unknown;
}

export interface InvoiceStatusResponse {
  meta: ApiMeta;
  data: InvoiceStatusData;
}

export interface CancelInvoiceResponse {
  meta: ApiMeta;
  data: InvoiceListItem & { active?: boolean };
}

export interface InstallmentPayment {
  id: number;
  installment: number;
  amount: number;
  status: string;
  paymentMethod?: string;
  transactionHash?: string;
  paymentDate?: string;
  [key: string]: unknown;
}

export interface InstallmentPaymentsResponse {
  meta: ApiMeta;
  data: InstallmentPayment[];
}
