import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { AuthLayoutComponent } from '../auth-layout/auth-layout.component';

@Component({
  selector: 'app-forgot-password',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink, AuthLayoutComponent],
  templateUrl: './forgot-password.component.html',
})
export class ForgotPasswordComponent {
  private readonly fb = inject(FormBuilder);
  readonly authService = inject(AuthService);

  readonly isSubmitted = signal<boolean>(false);
  readonly submittedEmail = signal<string>('');
  readonly errorMessage = signal<string | null>(null);
  readonly formSubmitted = signal<boolean>(false);

  readonly form: FormGroup = this.fb.group({
    email: ['', [Validators.required, Validators.email]],
  });

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
    const email = this.form.value.email;
    this.submittedEmail.set(email);

    this.authService.requestPasswordReset(email).subscribe({
      next: () => {
        this.isSubmitted.set(true);
      },
      error: (err) => {
        if (err.status === 429) {
          this.errorMessage.set('Too many reset attempts. Please wait a moment before trying again.');
        } else {
          this.isSubmitted.set(true);
        }
      },
    });
  }
}
