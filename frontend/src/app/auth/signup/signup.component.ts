import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators, AbstractControl, ValidationErrors } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { AuthLayoutComponent } from '../auth-layout/auth-layout.component';

export function passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
  const password = control.get('password');
  const confirmPassword = control.get('confirmPassword');
  
  if (password && confirmPassword && password.value !== confirmPassword.value) {
    confirmPassword.setErrors({ ...confirmPassword.errors, passwordMismatch: true });
    return { passwordMismatch: true };
  } else {
    // Need to clear the mismatch error if they now match, but retain other errors
    if (confirmPassword?.hasError('passwordMismatch')) {
      const errors = { ...confirmPassword.errors };
      delete errors['passwordMismatch'];
      confirmPassword.setErrors(Object.keys(errors).length ? errors : null);
    }
  }
  return null;
}

@Component({
  selector: 'app-signup',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, RouterLink, AuthLayoutComponent],
  templateUrl: './signup.component.html',
})
export class SignupComponent {
  private readonly fb = inject(FormBuilder);
  readonly authService = inject(AuthService);
  private readonly router = inject(Router);

  readonly showPassword = signal<boolean>(false);
  readonly showConfirmPassword = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);
  readonly formSubmitted = signal<boolean>(false);

  readonly form: FormGroup = this.fb.group({
    name: ['', [Validators.required, Validators.minLength(2)]],
    email: ['', [Validators.required, Validators.email]],
    nationalId: ['', [Validators.required, Validators.pattern(/^[0-9]{6,10}$/)]],
    phoneNumber: ['', [Validators.required, Validators.pattern(/^(\+?[0-9\s-]{9,15})$/)]],
    password: ['', [Validators.required, Validators.minLength(8)]],
    confirmPassword: ['', [Validators.required]],
  }, { validators: passwordMatchValidator });

  togglePasswordVisibility(): void {
    this.showPassword.update((prev) => !prev);
  }
  
  toggleConfirmPasswordVisibility(): void {
    this.showConfirmPassword.update((prev) => !prev);
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
    const { name, email, nationalId, phoneNumber, password } = this.form.value;

    this.authService.register({
      name,
      email,
      national_id: nationalId ? nationalId.trim() : undefined,
      phone_number: phoneNumber ? phoneNumber.trim() : undefined,
      password,
    }).subscribe({
      next: () => {
        // Automatically login after successful registration or route to login with success message
        this.router.navigate(['/login']);
      },
      error: (err) => {
        if (err.status === 409) {
          this.errorMessage.set('An account with this email already exists.');
        } else if (err.status === 0) {
          this.errorMessage.set('Unable to connect to the server. Please check your network connection.');
        } else {
          this.errorMessage.set(err.error?.detail || err.error?.message || 'An unexpected error occurred. Please try again.');
        }
      },
    });
  }
}
