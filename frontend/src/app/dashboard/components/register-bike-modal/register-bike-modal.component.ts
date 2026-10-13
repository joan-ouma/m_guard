import { Component, effect, inject, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';

@Component({
  selector: 'app-register-bike-modal',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './register-bike-modal.component.html',
})
export class RegisterBikeModalComponent {
  private readonly fb = inject(FormBuilder);

  readonly isOpen = input<boolean>(false);
  readonly isSubmitting = input<boolean>(false);
  readonly registrationError = input<string | null>(null);
  readonly currentUser = input<any>();

  readonly close = output<void>();
  readonly register = output<any>();

  readonly form: FormGroup = this.fb.group({
    plate: ['', [Validators.required, Validators.pattern(/^[A-Z]{4}\s?[0-9]{3}[A-Z]$/i)]],
    model: ['', [Validators.required]],
    imei: ['', [Validators.required, Validators.pattern(/^[0-9]{15}$/)]],
    county: ['Nairobi County', [Validators.required]],
    subCounty: ['Westlands', [Validators.required]],
    isSelfRider: [false],
    riderName: ['', [Validators.required, Validators.minLength(2)]],
    riderNationalId: ['', [Validators.required, Validators.pattern(/^[0-9]{6,10}$/)]],
    riderPhone: ['', [Validators.required, Validators.pattern(/^(\+?[0-9\s-]{9,15})$/)]],
  });

  constructor() {
    effect(() => {
      if (this.isOpen()) {
        this.form.reset({
          county: 'Nairobi County',
          subCounty: 'Westlands',
          isSelfRider: false,
          riderName: '',
          riderNationalId: '',
          riderPhone: '',
        });
      }
    });
  }

  onToggleSelfRider(event: Event): void {
    const checked = (event.target as HTMLInputElement).checked;
    this.form.patchValue({ isSelfRider: checked });
    if (checked) {
      const user = this.currentUser();
      if (user) {
        this.form.patchValue({
          riderName: user.name || '',
          riderNationalId: user.national_id || '',
          riderPhone: user.phone_number || '',
        });
      }
    }
  }

  onSubmit(): void {
    if (this.form.invalid || this.isSubmitting()) return;
    this.register.emit(this.form.value);
  }
}
