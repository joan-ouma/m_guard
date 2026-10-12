import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { AuthLayoutComponent } from '../auth-layout/auth-layout.component';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink, AuthLayoutComponent],
  templateUrl: './login.component.html',
})
export class LoginComponent {
  private readonly fb = inject(FormBuilder);
  readonly authService = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  readonly showPassword = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);
  readonly formSubmitted = signal<boolean>(false);

  readonly form: FormGroup = this.fb.group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required]],
  });

  togglePasswordVisibility(): void {
    this.showPassword.update((prev) => !prev);
  }

  isFieldInvalid(field: string): boolean {
    const control = this.form.get(field);
    if (!control) return false;
    // Prevent premature errors: only show if form was submitted or user typed invalid content
    return control.invalid && (this.formSubmitted() || (control.touched && !!control.value));
  }

  onSubmit(): void {
    this.formSubmitted.set(true);

    if (this.form.invalid || this.authService.isLoading()) {
      return;
    }

    this.errorMessage.set(null);
    const credentials = this.form.value;

    this.authService.login(credentials).subscribe({
      next: () => {
        const returnUrl = this.route.snapshot.queryParams['returnUrl'] || '/';
        this.router.navigateByUrl(returnUrl);
      },
      error: (err) => {
        if (err.status === 401) {
          // FR-AUTH-02: Generic 401 on failed login, no email/password disclosure
          this.errorMessage.set('Invalid email or password. Please verify your credentials and try again.');
        } else if (err.status === 429) {
          // §6.2: 5 attempts/minute/IP rate limit
          this.errorMessage.set('Too many login attempts. Please wait a moment before trying again.');
        } else if (err.status === 0) {
          this.errorMessage.set('Unable to connect to the authentication server. Please check your network connection.');
        } else {
          this.errorMessage.set(err.error?.detail || err.error?.message || 'An unexpected error occurred. Please try again.');
        }
      },
    });
  }
}
