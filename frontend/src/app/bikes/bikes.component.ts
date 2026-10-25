import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { BikeService } from '../core/services/bike.service';
import { BikeDTO } from '../core/models/bike.models';
import Swal from 'sweetalert2';
import { OnboardingModalComponent } from '../dashboard/components/onboarding-modal/onboarding-modal.component';

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
  isImmobilized: boolean;
  pendingCommand?: string;
}

@Component({
  selector: 'app-bikes',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    OnboardingModalComponent,
  ],
  templateUrl: './bikes.component.html',
})
export class BikesComponent implements OnInit {
  readonly bikeService = inject(BikeService);
  private readonly router = inject(Router);
  private readonly fb = inject(FormBuilder);

  readonly searchQuery = signal<string>('');
  readonly selectedFilter = signal<'ALL' | 'RUNNING' | 'STOPPED'>('ALL');
  readonly showRegisterModal = signal<boolean>(false);
  readonly registrationError = signal<string | null>(null);
  readonly isSubmitting = signal<boolean>(false);

  // Bike Details Modal
  readonly selectedBikeForModal = signal<MotorcycleAsset | null>(null);
  readonly showDetailsModal = signal<boolean>(false);

  readonly canCreateBike = computed(() => true);

  readonly bikes = computed<MotorcycleAsset[]>(() => {
    return this.bikeService.bikes().map((b) => this.mapToAsset(b));
  });

  readonly registerForm: FormGroup = this.fb.group({
    plate: ['', [Validators.required, Validators.pattern(/^[A-Z]{4}\s?[0-9]{3}[A-Z]$/i)]],
    model: ['', [Validators.required]],
    imei: ['', [Validators.required, Validators.pattern(/^[0-9]{15}$/)]],
    county: ['', [Validators.required]],
    subCounty: ['', [Validators.required]],
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

  readonly currentPage = signal<number>(1);
  readonly pageSize = signal<number>(10);
  readonly pageSizes = [10, 25, 50, 100];

  readonly totalPages = computed(() => {
    return Math.max(1, Math.ceil(this.filteredBikes().length / this.pageSize()));
  });

  readonly paginatedBikes = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredBikes().slice(start, start + this.pageSize());
  });

  prevPage(): void {
    if (this.currentPage() > 1) {
      this.currentPage.update((p) => p - 1);
    }
  }

  nextPage(): void {
    if (this.currentPage() < this.totalPages()) {
      this.currentPage.update((p) => p + 1);
    }
  }

  setPageSize(size: number): void {
    this.pageSize.set(size);
    this.currentPage.set(1);
  }

  onPageSizeChange(event: Event): void {
    const val = parseInt((event.target as HTMLSelectElement).value, 10);
    if (!isNaN(val) && val > 0) {
      this.setPageSize(val);
    }
  }

  ngOnInit(): void {
    this.bikeService.loadBikes().subscribe();
  }

  setFilter(filter: 'ALL' | 'RUNNING' | 'STOPPED'): void {
    this.selectedFilter.set(filter);
    this.currentPage.set(1);
  }

  onSearch(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    this.searchQuery.set(val);
    this.currentPage.set(1);
  }

  openRegisterModal(): void {
    this.registrationError.set(null);
    this.registerForm.reset({
      plate: '',
      model: '',
      imei: '',
      county: '',
      subCounty: '',
    });
    this.showRegisterModal.set(true);
  }

  closeRegisterModal(): void {
    this.showRegisterModal.set(false);
    this.registrationError.set(null);
  }

  onBikeAttached(bike: BikeDTO): void {
    this.closeRegisterModal();
    this.bikeService.loadBikes().subscribe();
  }

  registerBike(): void {
    if (this.registerForm.invalid || this.isSubmitting()) return;

    this.isSubmitting.set(true);
    this.registrationError.set(null);

    const val = this.registerForm.value;
    const rawModel = (val.model || '').trim();
    const parts = rawModel.split(/\s+/);
    const make = parts[0] || '';
    const model = parts.length > 1 ? parts.slice(1).join(' ') : rawModel;

    this.bikeService
      .createBike({
        registration_number: val.plate.toUpperCase().trim(),
        label: val.model.trim(),
        make: make,
        model: model,
        imei: val.imei.trim(),
        county: val.county,
        sub_county: val.subCounty,
        latitude: 0,
        longitude: 0,
        current_address: '',
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
    this.closeBikeDetails();
    this.router.navigate(['/'], { queryParams: { bike: bikeId } });
  }

  navigateToHistory(bikeId: string): void {
    this.closeBikeDetails();
    this.router.navigate(['/history'], { queryParams: { bike: bikeId } });
  }

  openBikeDetails(bike: MotorcycleAsset): void {
    this.selectedBikeForModal.set(bike);
    this.showDetailsModal.set(true);
  }

  closeBikeDetails(): void {
    this.showDetailsModal.set(false);
    this.selectedBikeForModal.set(null);
  }



  toggleImmobilize(bike: MotorcycleAsset, event: Event): void {
    event.stopPropagation();
    if (bike.pendingCommand) {
      Swal.fire({
        toast: true,
        position: 'top-end',
        icon: 'info',
        title: `${bike.plate} is still awaiting confirmation`,
        showConfirmButton: false,
        timer: 3000,
        timerProgressBar: true,
      });
      return;
    }
    const isImmobilizing = !bike.isImmobilized;
    Swal.fire({
      title: isImmobilizing ? `Immobilize ${bike.plate}?` : `Reactivate ${bike.plate}?`,
      text: isImmobilizing
        ? 'This will immediately cut off power to the motorcycle starter.'
        : 'This will restore electrical starter power to the motorcycle.',
      icon: isImmobilizing ? 'warning' : 'question',
      showCancelButton: true,
      confirmButtonColor: isImmobilizing ? '#e11d48' : '#059669',
      cancelButtonColor: '#71717a',
      confirmButtonText: isImmobilizing ? 'Yes, Immobilize' : 'Yes, Reactivate',
      cancelButtonText: 'Cancel',
      reverseButtons: true,
    }).then((result) => {
      if (result.isConfirmed) {
        this.bikeService.toggleImmobilize(bike.id, isImmobilizing).subscribe({
          next: () => {
            Swal.fire({
              toast: true,
              position: 'top-end',
              icon: 'success',
              title: `${bike.plate} Awaiting Confirmation`,
              showConfirmButton: false,
              timer: 3000,
              timerProgressBar: true,
            });
          },
          error: (err) => {
            Swal.fire({
              icon: 'error',
              title: 'Action Failed',
              text: err?.error?.detail || 'Failed to update immobilization status',
              confirmButtonColor: '#09090b',
            });
          },
        });
      }
    });
  }

  private mapToAsset(b: BikeDTO): MotorcycleAsset {
    const modelText = [b.make, b.model].filter(Boolean).join(' ') || b.label || 'Motorcycle';
    const speedText = `${b.speed_kmh.toFixed(1)} km/h • Ignition ${b.ignition ? 'ON' : 'OFF'}`;

    let lastPing = 'No signal yet';
    const pingTime = b.last_telemetry_at;
    if (pingTime) {
      const diffSec = Math.max(0, Math.floor((Date.now() - new Date(pingTime).getTime()) / 1000));
      if (diffSec < 60) lastPing = `${Math.max(1, diffSec)}s ago`;
      else if (diffSec < 3600) lastPing = `${Math.floor(diffSec / 60)}m ago`;
      else if (diffSec < 86400) lastPing = `${Math.floor(diffSec / 3600)}h ago`;
      else lastPing = `${Math.floor(diffSec / 86400)}d ago`;
    }

    return {
      id: b.id,
      plate: b.registration_number,
      model: modelText,
      imei: b.imei,
      status: b.status,
      speedText: speedText,
      address: b.current_address ? b.current_address : 'Awaiting GPS lock...',
      county: b.county ? b.county : '',
      subCounty: b.sub_county ? b.sub_county : '',
      lat: b.latitude || 0,
      lng: b.longitude || 0,
      fenceName: b.geofence_id ? 'Assigned Geofence' : 'Not configured',
      fenceRadius: b.geofence_id ? 2500 : 0,
      batteryPercent: b.battery_percent ?? 0,
      signalQuality: b.signal_quality_csq ?? 0,
      lastPing: lastPing,
      isImmobilized: !!b.is_immobilized,
      pendingCommand: b.pending_command || undefined,
    };
  }
}
