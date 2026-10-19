import { Component, OnDestroy, inject, signal } from '@angular/core';
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
export class LoginComponent implements OnDestroy {
  private readonly fb = inject(FormBuilder);
  readonly authService = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  readonly showPassword = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);
  readonly formSubmitted = signal<boolean>(false);
  readonly cooldownSeconds = signal<number>(0);
  readonly attemptsUsed = signal<number>(0);
  private cooldownTimer?: ReturnType<typeof setInterval>;

  readonly form: FormGroup = this.fb.group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required]],
  });

  ngOnDestroy(): void {
    this.clearCooldown();
  }

  togglePasswordVisibility(): void {
    this.showPassword.update((prev) => !prev);
  }

  isFieldInvalid(field: string): boolean {
    const control = this.form.get(field);
    if (!control) return false;
    return control.invalid && (this.formSubmitted() || (control.touched && !!control.value));
  }

  startCooldown(seconds: number): void {
    this.clearCooldown();
    this.cooldownSeconds.set(seconds);

    this.cooldownTimer = setInterval(() => {
      const current = this.cooldownSeconds() - 1;
      if (current <= 0) {
        this.clearCooldown();
      } else {
        this.cooldownSeconds.set(current);
      }
    }, 1000);
  }

  clearCooldown(): void {
    if (this.cooldownTimer) {
      clearInterval(this.cooldownTimer);
      this.cooldownTimer = undefined;
    }
    this.cooldownSeconds.set(0);
  }

  onSubmit(): void {
    this.formSubmitted.set(true);

    if (this.form.invalid || this.authService.isLoading() || this.cooldownSeconds() > 0) {
      return;
    }

    this.errorMessage.set(null);
    const credentials = this.form.value;

    this.authService.login(credentials).subscribe({
      next: () => {
        this.clearCooldown();
        const returnUrl = this.route.snapshot.queryParams['returnUrl'] || '/';
        this.router.navigateByUrl(returnUrl);
      },
      error: (err) => {
        this.attemptsUsed.update((prev) => prev + 1);

        let retrySeconds = 0;
        const retryHeader = err.headers?.get('Retry-After');
        if (retryHeader) {
          const parsed = parseInt(retryHeader, 10);
          if (!isNaN(parsed) && parsed > 0) {
            retrySeconds = parsed;
          }
        }

        if (err.status === 429) {
          if (!retrySeconds) {
            retrySeconds = 60 * Math.pow(2, Math.max(0, this.attemptsUsed() - 5));
          }
          this.startCooldown(retrySeconds);
        } else if (err.status === 401) {
          this.errorMessage.set('Invalid email or password.');
          if (retrySeconds > 0) {
            this.startCooldown(retrySeconds);
          } else if (this.attemptsUsed() >= 5) {
            const expSec = 60 * Math.pow(2, Math.max(0, this.attemptsUsed() - 5));
            this.startCooldown(expSec);
          }
        } else if (err.status === 0) {
          this.errorMessage.set('Unable to connect to server. Please check your network.');
        } else {
          this.errorMessage.set(err.error?.detail || err.error?.message || 'Authentication failed. Please try again.');
        }
      },
    });
  }
}
