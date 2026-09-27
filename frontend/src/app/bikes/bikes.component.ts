import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink } from '@angular/router';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { AuthService } from '../core/services/auth.service';
import { BikeService } from '../core/services/bike.service';
import { BikeDTO } from '../core/models/bike.models';

export interface MotorcycleAsset {
  id: string;
  plate: string;
  model: string;
  imei: string;
  status: 'RUNNING' | 'STOPPED';
  speedText: string;
  address: string;
  county: string;
  subCounty: string;
  lat: number;
  lng: number;
  fenceName: string;
  fenceRadius: number;
  batteryPercent: number;
  signalQuality: number;
  lastPing: string;
}

@Component({
  selector: 'app-bikes',
  standalone: true,
  imports: [CommonModule, RouterLink, ReactiveFormsModule],
  templateUrl: './bikes.component.html',
})
export class BikesComponent implements OnInit {
  readonly authService = inject(AuthService);
  readonly bikeService = inject(BikeService);
  private readonly router = inject(Router);
  private readonly fb = inject(FormBuilder);

  readonly searchQuery = signal<string>('');
  readonly selectedFilter = signal<'ALL' | 'RUNNING' | 'STOPPED'>('ALL');
  readonly showRegisterModal = signal<boolean>(false);
  readonly registrationError = signal<string | null>(null);
  readonly isSubmitting = signal<boolean>(false);

  readonly canCreateBike = computed(() => {
    const role = this.authService.currentUser()?.role;
    return role === 'admin' || role === 'owner';
  });

  readonly bikes = computed<MotorcycleAsset[]>(() => {
    return this.bikeService.bikes().map((b) => this.mapToAsset(b));
  });

  readonly registerForm: FormGroup = this.fb.group({
    plate: ['', [Validators.required, Validators.pattern(/^[A-Z]{4}\s?[0-9]{3}[A-Z]$/i)]],
    model: ['', [Validators.required]],
    imei: ['', [Validators.required, Validators.pattern(/^[0-9]{15}$/)]],
    county: ['Nairobi County', [Validators.required]],
    subCounty: ['Westlands', [Validators.required]],
  });

  readonly filteredBikes = computed(() => {
    const query = this.searchQuery().toLowerCase().trim();
    const filter = this.selectedFilter();

    return this.bikes().filter((b) => {
      const matchesFilter = filter === 'ALL' || b.status === filter;

      const matchesQuery =
        !query ||
        b.plate.toLowerCase().includes(query) ||
        b.model.toLowerCase().includes(query) ||
        b.imei.toLowerCase().includes(query) ||
        b.county.toLowerCase().includes(query) ||
        b.subCounty.toLowerCase().includes(query) ||
        b.address.toLowerCase().includes(query);

      return matchesFilter && matchesQuery;
    });
  });

  readonly runningCount = computed(() => {
    return this.bikes().filter((b) => b.status === 'RUNNING').length;
  });

  readonly stoppedCount = computed(() => {
    return this.bikes().filter((b) => b.status === 'STOPPED').length;
  });

  ngOnInit(): void {
    this.bikeService.loadBikes().subscribe();
  }

  setFilter(filter: 'ALL' | 'RUNNING' | 'STOPPED'): void {
    this.selectedFilter.set(filter);
  }

  onSearch(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    this.searchQuery.set(val);
  }

  openRegisterModal(): void {
    this.registrationError.set(null);
    this.registerForm.reset({
      county: 'Nairobi County',
      subCounty: 'Westlands',
    });
    this.showRegisterModal.set(true);
  }

  closeRegisterModal(): void {
    this.showRegisterModal.set(false);
    this.registrationError.set(null);
  }

  registerBike(): void {
    if (this.registerForm.invalid || this.isSubmitting()) return;

    this.isSubmitting.set(true);
    this.registrationError.set(null);

    const val = this.registerForm.value;
    const parts = val.model.trim().split(' ');
    const make = parts[0] || 'Bajaj';
    const model = parts.slice(1).join(' ') || val.model.trim();

    this.bikeService
      .createBike({
        registration_number: val.plate.toUpperCase().trim(),
        label: val.model.trim(),
        make: make,
        model: model,
        imei: val.imei.trim(),
        county: val.county,
        sub_county: val.subCounty,
        latitude: -1.286389,
        longitude: 36.817223,
        current_address: 'Nairobi CBD Staging Yard',
      })
      .subscribe({
        next: () => {
          this.isSubmitting.set(false);
          this.closeRegisterModal();
        },
        error: (err) => {
          this.isSubmitting.set(false);
          const detail =
            err?.error?.detail ||
            (err?.error?.errors
              ? Object.values(err.error.errors).flat().join(', ')
              : 'Failed to register motorcycle');
          this.registrationError.set(detail);
        },
      });
  }

  navigateToMap(bikeId: string): void {
    this.router.navigate(['/'], { queryParams: { bike: bikeId } });
  }

  getUserInitials(): string {
    const name = this.authService.currentUser()?.name || 'Brian Otieno';
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .slice(0, 2)
      .toUpperCase();
  }

  handleLogout(): void {
    this.authService.logout().subscribe();
  }

  private mapToAsset(b: BikeDTO): MotorcycleAsset {
    const modelText = [b.make, b.model].filter(Boolean).join(' ') || b.label || 'Motorcycle';
    const speedText = `${b.speed_kmh.toFixed(1)} km/h • Ignition ${b.ignition ? 'ON' : 'OFF'}`;

    let lastPing = 'Just now';
    const pingTime = b.last_telemetry_at || b.updated_at;
    if (pingTime) {
      const diffSec = Math.floor((Date.now() - new Date(pingTime).getTime()) / 1000);
      if (diffSec < 60) lastPing = `${Math.max(1, diffSec)}s ago`;
      else if (diffSec < 3600) lastPing = `${Math.floor(diffSec / 60)}m ago`;
      else lastPing = `${Math.floor(diffSec / 3600)}h ago`;
    }

    return {
      id: b.id,
      plate: b.registration_number,
      model: modelText,
      imei: b.imei,
      status: b.status,
      speedText: speedText,
      address: b.current_address || 'Depot / Staging Yard',
      county: b.county || 'Nairobi County',
      subCounty: b.sub_county || 'Starehe',
      lat: b.latitude || -1.286389,
      lng: b.longitude || 36.817223,
      fenceName: b.geofence_id ? 'Assigned Geofence' : 'Nairobi Central Zone',
      fenceRadius: 2500,
      batteryPercent: b.battery_percent ?? 100,
      signalQuality: b.signal_quality_csq ?? 20,
      lastPing: lastPing,
    };
  }
}
