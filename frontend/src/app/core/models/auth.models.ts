export interface User {
  id: string;
  email: string;
  name: string;
  role?: 'owner' | 'admin' | 'dispatcher' | 'viewer';
  created_at?: string;
}

export interface Organization {
  id: string;
  name: string;
  plan_tier?: 'individual' | 'fleet' | 'enterprise';
  role?: 'owner' | 'admin' | 'dispatcher' | 'viewer';
  breadcrumb_retention_days?: number;
}

export interface LoginCredentials {
  email: string;
  password: string;
}

export interface SessionData {
  access_token: string;
  token_type: string;
  expires_in: number;
  user: User;
  organization?: Organization;
}

export interface AuthResponse {
  data: SessionData;
}

export interface PasswordResetRequest {
  email: string;
}

export interface PasswordResetConfirm {
  password: string;
}

export interface ProblemDetails {
  type?: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  errors?: Record<string, string[]>;
}
