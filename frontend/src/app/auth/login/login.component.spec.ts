import { TestBed, ComponentFixture } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { throwError } from 'rxjs';
import { vi } from 'vitest';
import { LoginComponent } from './login.component';
import { AuthService } from '../../core/services/auth.service';

describe('LoginComponent', () => {
  let component: LoginComponent;
  let fixture: ComponentFixture<LoginComponent>;
  let authService: AuthService;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [LoginComponent],
      providers: [
        AuthService,
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LoginComponent);
    component = fixture.componentInstance;
    authService = TestBed.inject(AuthService);
    fixture.detectChanges();
  });

  it('should create login component', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize form with empty invalid fields', () => {
    expect(component.form.valid).toBe(false);
    expect(component.form.get('email')?.valid).toBe(false);
    expect(component.form.get('password')?.valid).toBe(false);
  });

  it('should validate email format and password length', () => {
    component.form.patchValue({
      email: 'invalid-email',
      password: '123',
    });
    expect(component.form.valid).toBe(false);

    component.form.patchValue({
      email: 'dispatcher@motoguard.co.ke',
      password: 'validpassword',
    });
    expect(component.form.valid).toBe(true);
  });

  it('should toggle password visibility', () => {
    expect(component.showPassword()).toBe(false);
    component.togglePasswordVisibility();
    expect(component.showPassword()).toBe(true);
  });

  it('should display generic error message on 401 response per FR-AUTH-02', () => {
    vi.spyOn(authService, 'login').mockReturnValue(
      throwError(() => ({ status: 401 }))
    );

    component.form.patchValue({
      email: 'dispatcher@motoguard.co.ke',
      password: 'wrongpassword',
    });

    component.onSubmit();
    expect(component.errorMessage()).toContain('Invalid email or password');
  });
});
