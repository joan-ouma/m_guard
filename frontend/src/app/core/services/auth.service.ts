import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, tap, catchError, throwError, of } from 'rxjs';
import {
  AuthResponse,
  LoginCredentials,
  RegisterCredentials,
  SessionData,
  User,
} from '../models/auth.models';
import { environment } from '../../../environments/environment';

const STORAGE_KEY_TOKEN = 'motoguard_access_token';
const STORAGE_KEY_USER = 'motoguard_user';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);

  // Signal-based reactive state
  readonly accessToken = signal<string | null>(this.getStoredToken());
  readonly currentUser = signal<User | null>(this.getStoredUser());
  readonly isLoading = signal<boolean>(false);

  readonly isAuthenticated = computed(() => !!this.accessToken());

  constructor() {
    if (this.accessToken()) {
      this.verifySession();
    }
  }

  private getStoredToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem(STORAGE_KEY_TOKEN);
  }

  private getStoredUser(): User | null {
    if (typeof window === 'undefined') return null;
    const raw = localStorage.getItem(STORAGE_KEY_USER);
    if (!raw) return null;
    try {
      return JSON.parse(raw);
    } catch {
      return null;
    }
  }

  private setSession(session: SessionData): void {
    this.accessToken.set(session.access_token);
    this.currentUser.set(session.user);

    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY_TOKEN, session.access_token);
      localStorage.setItem(STORAGE_KEY_USER, JSON.stringify(session.user));
      document.cookie = `motoguard_token=${encodeURIComponent(session.access_token)}; path=/; SameSite=Lax`;
    }
  }

  private clearSession(): void {
    this.accessToken.set(null);
    this.currentUser.set(null);

    if (typeof window !== 'undefined') {
      localStorage.removeItem(STORAGE_KEY_TOKEN);
      localStorage.removeItem(STORAGE_KEY_USER);
      document.cookie = 'motoguard_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT; SameSite=Lax';
    }
  }

  /**
   * POST /v1/users/register
   * Register a new user account
   */
  register(credentials: RegisterCredentials): Observable<{ data: User }> {
    this.isLoading.set(true);
    return this.http.post<{ data: User }>(`${environment.apiBaseUrl}/v1/users/register`, credentials).pipe(
      tap(() => this.isLoading.set(false)),
      catchError((err) => {
        this.isLoading.set(false);
        return throwError(() => err);
      })
    );
  }

  /**
   * POST /v1/sessions
   * Login with email and password
   */
  login(credentials: LoginCredentials): Observable<AuthResponse> {
    this.isLoading.set(true);
    return this.http.post<AuthResponse>(`${environment.apiBaseUrl}/v1/sessions`, credentials, { withCredentials: true }).pipe(
      tap((res) => {
        this.isLoading.set(false);
        if (res?.data) {
          this.setSession(res.data);
        }
      }),
      catchError((err) => {
        this.isLoading.set(false);
        return throwError(() => err);
      })
    );
  }

  /**
   * DELETE /v1/sessions/current
   * Revoke current session on server and clear local state
   */
  logout(): Observable<void> {
    this.isLoading.set(true);
    return this.http.delete<void>(`${environment.apiBaseUrl}/v1/sessions/current`, { withCredentials: true }).pipe(
      tap(() => {
        this.clearSession();
        this.isLoading.set(false);
        this.router.navigate(['/login']);
      }),
      catchError(() => {
        this.clearSession();
        this.isLoading.set(false);
        this.router.navigate(['/login']);
        return of(undefined);
      })
    );
  }

  /**
   * PUT /v1/sessions/current
   * Refresh access token via HttpOnly refresh cookie
   */
  refreshToken(): Observable<AuthResponse> {
    return this.http.put<AuthResponse>(`${environment.apiBaseUrl}/v1/sessions/current`, {}, { withCredentials: true }).pipe(
      tap((res) => {
        if (res?.data) {
          this.setSession(res.data);
        }
      }),
      catchError((err) => {
        this.clearSession();
        return throwError(() => err);
      })
    );
  }

  /**
   * POST /v1/password-resets
   * Request password recovery
   */
  requestPasswordReset(email: string): Observable<{ message: string }> {
    this.isLoading.set(true);
    return this.http
      .post<{ message: string }>(`${environment.apiBaseUrl}/v1/password-resets`, { email })
      .pipe(
        tap(() => this.isLoading.set(false)),
        catchError((err) => {
          this.isLoading.set(false);
          return throwError(() => err);
        })
      );
  }

  /**
   * PATCH /v1/password-resets/{token}
   * Consume reset token and set new password
   */
  confirmPasswordReset(token: string, password: string): Observable<{ message: string }> {
    this.isLoading.set(true);
    return this.http
      .patch<{ message: string }>(`${environment.apiBaseUrl}/v1/password-resets/${encodeURIComponent(token)}`, {
        password,
      })
      .pipe(
        tap(() => {
          this.isLoading.set(false);
          this.clearSession();
        }),
        catchError((err) => {
          this.isLoading.set(false);
          return throwError(() => err);
        })
      );
  }

  private verifySession(): void {
    // Session validation
  }
}
