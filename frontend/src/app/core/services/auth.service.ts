import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, tap, catchError, throwError, of } from 'rxjs';
import {
  AuthResponse,
  LoginCredentials,
  Organization,
  SessionData,
  User,
} from '../models/auth.models';

const STORAGE_KEY_TOKEN = 'motoguard_access_token';
const STORAGE_KEY_USER = 'motoguard_user';
const STORAGE_KEY_ORG = 'motoguard_org';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);

  // Signal-based reactive state
  readonly accessToken = signal<string | null>(this.getStoredToken());
  readonly currentUser = signal<User | null>(this.getStoredUser());
  readonly currentOrg = signal<Organization | null>(this.getStoredOrg());
  readonly isLoading = signal<boolean>(false);

  readonly isAuthenticated = computed(() => !!this.accessToken());

  constructor() {
    // If token exists, verify or refresh
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

  private getStoredOrg(): Organization | null {
    if (typeof window === 'undefined') return null;
    const raw = localStorage.getItem(STORAGE_KEY_ORG);
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
    this.currentOrg.set(session.organization || null);

    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY_TOKEN, session.access_token);
      localStorage.setItem(STORAGE_KEY_USER, JSON.stringify(session.user));
      if (session.organization) {
        localStorage.setItem(STORAGE_KEY_ORG, JSON.stringify(session.organization));
      } else {
        localStorage.removeItem(STORAGE_KEY_ORG);
      }
    }
  }

  private clearSession(): void {
    this.accessToken.set(null);
    this.currentUser.set(null);
    this.currentOrg.set(null);

    if (typeof window !== 'undefined') {
      localStorage.removeItem(STORAGE_KEY_TOKEN);
      localStorage.removeItem(STORAGE_KEY_USER);
      localStorage.removeItem(STORAGE_KEY_ORG);
    }
  }

  /**
   * POST /v1/sessions
   * Login with email and password (FR-AUTH-01, FR-AUTH-03)
   */
  login(credentials: LoginCredentials): Observable<AuthResponse> {
    this.isLoading.set(true);
    return this.http.post<AuthResponse>('/v1/sessions', credentials).pipe(
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
   * Revoke current session on server and clear local state (FR-AUTH-04/05)
   */
  logout(): Observable<void> {
    this.isLoading.set(true);
    return this.http.delete<void>('/v1/sessions/current').pipe(
      tap(() => {
        this.clearSession();
        this.isLoading.set(false);
        this.router.navigate(['/login']);
      }),
      catchError(() => {
        // Even if server call errors (e.g. offline), clear locally
        this.clearSession();
        this.isLoading.set(false);
        this.router.navigate(['/login']);
        return of(undefined);
      })
    );
  }

  /**
   * PUT /v1/sessions/current
   * Refresh access token via HttpOnly refresh cookie (FR-AUTH-03)
   */
  refreshToken(): Observable<AuthResponse> {
    return this.http.put<AuthResponse>('/v1/sessions/current', {}).pipe(
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
   * Request password recovery (FR-PWD-01/03)
   * Enumeration-safe: Always returns 202 Accepted regardless of email existence
   */
  requestPasswordReset(email: string): Observable<{ message: string }> {
    this.isLoading.set(true);
    return this.http
      .post<{ message: string }>('/v1/password-resets', { email })
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
   * Consume reset token and set new password (FR-PWD-04)
   * Revokes all sessions server-side
   */
  confirmPasswordReset(token: string, password: string): Observable<{ message: string }> {
    this.isLoading.set(true);
    return this.http
      .patch<{ message: string }>(`/v1/password-resets/${encodeURIComponent(token)}`, {
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

  /**
   * Check or verify current session status
   */
  private verifySession(): void {
    // If needed, check current organization / membership or refresh
  }
}
