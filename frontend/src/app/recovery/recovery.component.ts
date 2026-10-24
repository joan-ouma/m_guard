import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { RouterLink, RouterLinkActive } from '@angular/router';
import Swal from 'sweetalert2';
import { MobileNavComponent } from '../ui/mobile-nav/mobile-nav.component';
import { BikeService } from '../core/services/bike.service';
import { BikeListResponse } from '../core/models/bike.models';
import { RecoveryService, THEFT_MODES, TheftMode } from '../core/services/recovery.service';
import { EMERGENCY_CONTACTS, EmergencyContact, distanceKm } from '../emergency/emergency.data';

type Step = 'start' | 'how' | 'when' | 'recorded' | 'police' | 'ob' | 'done';

const STEP_ORDER: Step[] = ['how', 'when', 'recorded', 'police', 'ob'];

export interface NearbyStation extends EmergencyContact {
  distanceLabel: string | null;
}

@Component({
  selector: 'app-recovery',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, RouterLinkActive, MobileNavComponent],
  templateUrl: './recovery.component.html',
})
export class RecoveryComponent implements OnInit {
  private readonly bikeService = inject(BikeService);
  private readonly recoveryService = inject(RecoveryService);

  readonly theftModes = THEFT_MODES;
  readonly step = signal<Step>('start');
  readonly stepIndex = computed(() => STEP_ORDER.indexOf(this.step()) + 1);
  readonly totalSteps = STEP_ORDER.length;
  readonly saving = signal<boolean>(false);

  // Step 1: how it happened
  readonly mode = signal<TheftMode | null>(null);

  // Step 2: when and where
  readonly lostAt = signal<string>(this.toLocalInputValue(new Date()));
  readonly locationText = signal<string>('');
  readonly latitude = signal<number | null>(null);
  readonly longitude = signal<number | null>(null);
  readonly locating = signal<boolean>(false);

  // Step 3 onwards: the saved record
  readonly incidentId = signal<string | null>(null);
  readonly obNumber = signal<string>('');
  readonly selectedFile = signal<File | null>(null);
  readonly formError = signal<string | null>(null);

  readonly bikeId = signal<string | null>(null);

  // Step 4: police stations
  readonly referenceLocation = signal<{ lat: number; lng: number } | null>(null);
  readonly nearbyStations = signal<NearbyStation[]>([]);

  readonly modeLabel = computed(() => THEFT_MODES.find((m) => m.value === this.mode())?.label ?? '');
  readonly attachmentName = computed(() => this.selectedFile()?.name ?? null);

  ngOnInit(): void {
    this.bikeService.loadBikes().subscribe({
      next: (res: BikeListResponse) => {
        const bike = (res.data || []).find((b) => b.latitude && b.longitude);
        if (bike) {
          this.bikeId.set(bike.id);
          this.referenceLocation.set({ lat: bike.latitude, lng: bike.longitude });
        }
      },
    });
  }

  // Start: confirm before starting the wizard
  confirmTheft(): void {
    Swal.fire({
      title: 'Report your bike as lost?',
      text: 'This starts the recovery process. You will record what happened, then be guided to report it to the police.',
      icon: 'warning',
      showCancelButton: true,
      confirmButtonColor: '#e11d48',
      cancelButtonColor: '#71717a',
      confirmButtonText: 'Yes, I lost my bike',
      cancelButtonText: 'Not now',
      reverseButtons: true,
    }).then((result) => {
      if (result.isConfirmed) {
        this.resetWizard();
        this.step.set('how');
      }
    });
  }

  next(): void {
    this.formError.set(null);
    const current = this.step();

    if (current === 'how') {
      if (!this.mode()) {
        this.formError.set('Choose how the bike was lost.');
        return;
      }
      this.step.set('when');
      return;
    }

    if (current === 'when') {
      if (!this.lostAt()) {
        this.formError.set('Enter the date and time the bike was lost.');
        return;
      }
      if (!this.locationText().trim() && this.latitude() === null) {
        this.formError.set('Enter where it was lost, or use your current location.');
        return;
      }
      this.saveIncident();
      return;
    }

    if (current === 'recorded') {
      this.step.set('police');
      return;
    }

    if (current === 'police') {
      this.step.set('ob');
    }
  }

  back(): void {
    this.formError.set(null);
    const current = this.step();
    const index = STEP_ORDER.indexOf(current);
    if (index > 0) {
      this.step.set(STEP_ORDER[index - 1]);
    }
  }

  useCurrentLocation(): void {
    if (typeof navigator === 'undefined' || !navigator.geolocation) {
      this.formError.set('Location is not available on this device. Type the place instead.');
      return;
    }
    this.locating.set(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        this.latitude.set(pos.coords.latitude);
        this.longitude.set(pos.coords.longitude);
        if (!this.locationText().trim()) {
          this.locationText.set(`GPS ${pos.coords.latitude.toFixed(5)}, ${pos.coords.longitude.toFixed(5)}`);
        }
        this.locating.set(false);
      },
      () => {
        this.locating.set(false);
        this.formError.set('Could not get your location. Type the place instead.');
      },
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }

  suggestPoliceStations(): void {
    const ref = this.referenceLocation() ?? this.currentPoint();
    const police = EMERGENCY_CONTACTS.filter((c) => c.type === 'police' && c.latitude && c.longitude);

    if (!ref) {
      this.nearbyStations.set(police.slice(0, 5).map((c) => ({ ...c, distanceLabel: null })));
      return;
    }

    this.referenceLocation.set(ref);
    const ranked = police
      .map((c) => ({ ...c, km: distanceKm(ref.lat, ref.lng, c.latitude, c.longitude) }))
      .sort((a, b) => a.km - b.km)
      .slice(0, 5);

    this.nearbyStations.set(
      ranked.map(({ km, ...c }) => ({
        ...c,
        distanceLabel: km < 1 ? `${Math.round(km * 1000)}m away` : `${km.toFixed(1)} km away`,
      })),
    );
  }

  onFileSelected(event: Event): void {
    this.selectedFile.set((event.target as HTMLInputElement).files?.[0] ?? null);
  }

  saveOb(): void {
    this.formError.set(null);
    const ob = this.obNumber().trim();
    const id = this.incidentId();

    if (!ob) {
      this.formError.set('Enter the OB number from the police receipt.');
      return;
    }
    if (!/^[A-Za-z0-9\/\-\s]{3,30}$/.test(ob)) {
      this.formError.set('The OB number should contain only letters, numbers, / or -.');
      return;
    }
    if (!id) {
      this.formError.set('Your recovery record is missing. Start again.');
      return;
    }

    this.saving.set(true);
    this.recoveryService.attachOb(id, ob, this.selectedFile()).subscribe({
      next: () => {
        this.saving.set(false);
        this.step.set('done');
      },
      error: (err: HttpErrorResponse) => {
        this.saving.set(false);
        this.formError.set(this.errorMessage(err));
      },
    });
  }

  startAgain(): void {
    this.resetWizard();
    this.step.set('start');
  }

  private saveIncident(): void {
    const mode = this.mode();
    if (!mode) return;

    const previousId = this.incidentId();
    const lostAt = new Date(this.lostAt());
    if (isNaN(lostAt.getTime())) {
      this.formError.set('Enter a valid date and time.');
      return;
    }

    this.saving.set(true);
    this.recoveryService
      .createIncident({
        bike_id: this.bikeId() ?? undefined,
        mode,
        lost_at: lostAt.toISOString(),
        location_lost: this.locationText().trim() || 'Unknown location',
        latitude: this.latitude(),
        longitude: this.longitude(),
      })
      .subscribe({
        next: (incident) => {
          this.saving.set(false);
          this.incidentId.set(incident.id);
          // Going back and re-saving replaces the earlier record instead of duplicating it.
          if (previousId) {
            this.recoveryService.removeIncident(previousId).subscribe({ error: () => undefined });
          }
          this.step.set('recorded');
        },
        error: (err: HttpErrorResponse) => {
          this.saving.set(false);
          this.formError.set(this.errorMessage(err));
        },
      });
  }

  private errorMessage(err: HttpErrorResponse): string {
    const errors = err?.error?.errors as Record<string, string[]> | undefined;
    if (errors) {
      const first = Object.values(errors).flat()[0];
      if (first) return first;
    }
    return err?.error?.detail || 'Something went wrong. Please try again.';
  }

  private currentPoint(): { lat: number; lng: number } | null {
    const lat = this.latitude();
    const lng = this.longitude();
    return lat !== null && lng !== null ? { lat, lng } : null;
  }

  private resetWizard(): void {
    this.mode.set(null);
    this.lostAt.set(this.toLocalInputValue(new Date()));
    this.locationText.set('');
    this.latitude.set(null);
    this.longitude.set(null);
    this.incidentId.set(null);
    this.obNumber.set('');
    this.selectedFile.set(null);
    this.nearbyStations.set([]);
    this.formError.set(null);
  }

  private toLocalInputValue(date: Date): string {
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }
}
