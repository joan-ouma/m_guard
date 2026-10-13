import { Component, effect, inject, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';

import { AuthService } from '../../../core/services/auth.service';

export interface GeofenceData {
  fenceName: string;
  fenceRadius: number;
  fenceTrigger: string;
  fenceEmail: string;
}

@Component({
  selector: 'app-geofence-modal',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './geofence-modal.component.html',
})
export class GeofenceModalComponent {
  private readonly fb = inject(FormBuilder);
  private readonly authService = inject(AuthService);

  readonly isOpen = input<boolean>(false);
  readonly currentData = input<GeofenceData | null>(null);

  readonly close = output<void>();
  readonly save = output<GeofenceData>();

  readonly form: FormGroup = this.fb.group({
    fenceName: ['Ongata Rongai Delivery Zone', [Validators.required]],
    fenceRadius: [3500, [Validators.required, Validators.min(100), Validators.max(50000)]],
    fenceTrigger: ['Alert on Exit', [Validators.required]],
    fenceEmail: [this.authService.currentUser()?.email || '', [Validators.required, Validators.email]],
  });

  constructor() {
    effect(() => {
      const data = this.currentData();
      if (data) {
        this.form.patchValue({
          fenceName: data.fenceName,
          fenceRadius: data.fenceRadius,
          fenceTrigger: data.fenceTrigger,
          fenceEmail: data.fenceEmail,
        });
      }
    });
  }

  onSubmit(e: Event): void {
    e.preventDefault();
    if (this.form.invalid) return;

    this.save.emit({
      fenceName: this.form.value.fenceName,
      fenceRadius: parseInt(this.form.value.fenceRadius, 10),
      fenceTrigger: this.form.value.fenceTrigger,
      fenceEmail: this.form.value.fenceEmail,
    });
  }
}
