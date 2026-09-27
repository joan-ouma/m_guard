import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { Component } from '@angular/core';
import { AuthService } from './auth.service';
import { AuthResponse } from '../models/auth.models';

@Component({ standalone: true, template: '' })
class DummyLoginComponent {}

describe('AuthService', () => {
  let service: AuthService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({
      providers: [
        AuthService,
        provideRouter([{ path: 'login', component: DummyLoginComponent }]),
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });
    service = TestBed.inject(AuthService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
    localStorage.clear();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
    expect(service.isAuthenticated()).toBe(false);
  });

  it('should login and set session data upon successful POST /v1/sessions', () => {
    const mockResponse: AuthResponse = {
      data: {
        access_token: 'test_access_token_jwt',
        token_type: 'Bearer',
        expires_in: 900,
        user: {
          id: 'usr_123',
          email: 'dispatcher@motoguard.co.ke',
          name: 'Chief Dispatcher',
          role: 'dispatcher',
        },
        organization: {
          id: 'org_abc',
          name: 'MotoGuard Nairobi Fleet',
          plan_tier: 'fleet',
        },
      },
    };

    service
      .login({ email: 'dispatcher@motoguard.co.ke', password: 'password123' })
      .subscribe((res) => {
        expect(res.data.access_token).toBe('test_access_token_jwt');
        expect(service.isAuthenticated()).toBe(true);
        expect(service.currentUser()?.email).toBe('dispatcher@motoguard.co.ke');
        expect(service.currentOrg()?.name).toBe('MotoGuard Nairobi Fleet');
      });

    const req = httpTesting.expectOne('/v1/sessions');
    expect(req.request.method).toBe('POST');
    req.flush(mockResponse);
  });

  it('should request password reset via POST /v1/password-resets', () => {
    service.requestPasswordReset('dispatcher@motoguard.co.ke').subscribe((res) => {
      expect(res.message).toBe('Accepted');
    });

    const req = httpTesting.expectOne('/v1/password-resets');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ email: 'dispatcher@motoguard.co.ke' });
    req.flush({ message: 'Accepted' }, { status: 202, statusText: 'Accepted' });
  });

  it('should confirm password reset via PATCH /v1/password-resets/:token', () => {
    const token = 'xyz_crypto_token_32_bytes';
    service.confirmPasswordReset(token, 'new_secret_password').subscribe((res) => {
      expect(res.message).toBe('Password updated');
    });

    const req = httpTesting.expectOne(`/v1/password-resets/${token}`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ password: 'new_secret_password' });
    req.flush({ message: 'Password updated' });
  });

  it('should logout and clear session state', () => {
    service.accessToken.set('mock_token');
    expect(service.isAuthenticated()).toBe(true);

    service.logout().subscribe(() => {
      expect(service.isAuthenticated()).toBe(false);
      expect(service.accessToken()).toBeNull();
      expect(service.currentUser()).toBeNull();
    });

    const req = httpTesting.expectOne('/v1/sessions/current');
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });
  });
});
