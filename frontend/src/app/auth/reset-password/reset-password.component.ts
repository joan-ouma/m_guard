import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AbstractControl, FormBuilder, FormGroup, ReactiveFormsModule, ValidationErrors, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { AuthLayoutComponent } from '../auth-layout/auth-layout.component';

function passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
  const password = control.get('password');
  const confirmPassword = control.get('confirmPassword');
  if (password && confirmPassword && password.value !== confirmPassword.value) {
    return { passwordMismatch: true };
  }
  return null;
}

@Component({
  selector: 'app-reset-password',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink, AuthLayoutComponent],
  templateUrl: './reset-password.component.html',
})
export class ResetPasswordComponent implements OnInit {
  private readonly fb = inject(FormBuilder);
  readonly authService = inject(AuthService);
  private readonly route = inject(ActivatedRoute);

  readonly showPassword = signal<boolean>(false);
  readonly isSuccess = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);
  readonly formSubmitted = signal<boolean>(false);

  tokenFromRoute: string | null = null;

  readonly form: FormGroup = this.fb.group(
    {
      token: ['', [Validators.required]],
      password: ['', [Validators.required, Validators.minLength(8)]],
      confirmPassword: ['', [Validators.required]],
    },
    { validators: [passwordMatchValidator] }
  );

  ngOnInit(): void {
    const routeToken = this.route.snapshot.paramMap.get('token') || this.route.snapshot.queryParamMap.get('token');
    if (routeToken) {
      this.tokenFromRoute = routeToken;
      this.form.patchValue({ token: routeToken });
    }
  }

  togglePasswordVisibility(): void {
    this.showPassword.update((prev) => !prev);
  }

  isFieldInvalid(field: string): boolean {
    const control = this.form.get(field);
    if (!control) return false;
    return control.invalid && (this.formSubmitted() || (control.touched && !!control.value));
  }

  onSubmit(): void {
    this.formSubmitted.set(true);

    if (this.form.invalid || this.authService.isLoading()) {
      return;
    }

    this.errorMessage.set(null);
    const { token, password } = this.form.value;

    this.authService.confirmPasswordReset(token, password).subscribe({
      next: () => {
        this.isSuccess.set(true);
      },
      error: (err) => {
        if (err.status === 400 || err.status === 404) {
          this.errorMessage.set('The password reset link is invalid or has expired. Please request a new recovery link.');
        } else {
          this.errorMessage.set(err.error?.detail || err.error?.message || 'Failed to reset password. Please try again.');
        }
      },
    });
  }
}
