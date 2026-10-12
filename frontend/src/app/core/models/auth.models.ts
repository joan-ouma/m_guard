export interface User {
  id: string;
  email: string;
  name: string;
<<<<<<< HEAD
=======
  national_id?: string;
  phone_number?: string;
>>>>>>> origin/main
  role?: 'owner';
  created_at?: string;
}

export interface LoginCredentials {
  email: string;
  password: string;
}

<<<<<<< HEAD
=======
export interface RegisterCredentials {
  name: string;
  email: string;
  password: string;
  national_id?: string;
  phone_number?: string;
}

>>>>>>> origin/main
export interface SessionData {
  access_token: string;
  token_type: string;
  expires_in: number;
  user: User;
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
