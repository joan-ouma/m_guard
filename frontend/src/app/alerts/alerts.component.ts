import { Component, OnInit, computed, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { MobileNavComponent } from '../ui/mobile-nav/mobile-nav.component';
import { BikeService } from '../core/services/bike.service';
import { BikeDTO, BikeEventDTO, BikeListResponse, PaginatedEventsResponse } from '../core/models/bike.models';
import { Observable, forkJoin, of } from 'rxjs';
import { catchError } from 'rxjs/operators';

export interface DisplayAlert {
  id: string | number;
  bikeId: string;
  plate: string;
  model: string;
  eventType: string;
  title: string;
  description: string;
  severity: 'critical' | 'warning' | 'info';
  latitude: number;
  longitude: number;
  timestamp: string;
}

@Component({
  selector: 'app-alerts',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive, MobileNavComponent],
  templateUrl: './alerts.component.html'
})
export class AlertsComponent implements OnInit {
  private readonly bikeService = inject(BikeService);

  readonly isLoading = signal<boolean>(true);
  readonly bikes = signal<BikeDTO[]>([]);
  readonly alerts = signal<DisplayAlert[]>([]);
  readonly selectedSeverity = signal<'ALL' | 'critical' | 'warning' | 'info'>('ALL');

  readonly criticalCount = computed(() => this.alerts().filter((a) => a.severity === 'critical').length);
  readonly warningCount = computed(() => this.alerts().filter((a) => a.severity === 'warning').length);
  readonly infoCount = computed(() => this.alerts().filter((a) => a.severity === 'info').length);

  readonly filteredAlerts = computed(() => {
    const sev = this.selectedSeverity();
    const list = this.alerts();
    if (sev === 'ALL') return list;
    return list.filter((a) => a.severity === sev);
  });

  constructor() {
    // Listen to live SSE alerts from the backend
    effect(() => {
      const live = this.bikeService.latestAlert();
      if (!live) return;

      const evType = live.event_type || live.eventType || 'ALERT';
      const plate = live.plate || live.registration_number || 'Motorcycle';
      const newAlert: DisplayAlert = {
        id: 'live_' + Date.now(),
        bikeId: live.bike_id || live.bikeId || '',
        plate: plate,
        model: live.model || '',
        eventType: evType,
        title: this.formatTitle(evType),
        description: this.formatDescription(evType, plate),
        severity: this.classifySeverity(evType),
        latitude: Number(live.latitude) || 0,
        longitude: Number(live.longitude) || 0,
        timestamp: live.recorded_at || live.recordedAt || new Date().toISOString(),
      };

      this.alerts.update((prev) => [newAlert, ...prev]);
    });
  }

  ngOnInit(): void {
    this.loadAlerts();
  }

  loadAlerts(): void {
    this.isLoading.set(true);

    this.bikeService.loadBikes().subscribe({
      next: (res: BikeListResponse) => {
        const bikeList: BikeDTO[] = res.data || [];
        this.bikes.set(bikeList);

        if (bikeList.length === 0) {
          this.alerts.set([]);
          this.isLoading.set(false);
          return;
        }

        // Query events across all bikes
        const reqs: Observable<PaginatedEventsResponse>[] = bikeList.map((bike: BikeDTO) =>
          this.bikeService.getBikeEvents(bike.id, 100).pipe(
            catchError(() => of({ data: [] as BikeEventDTO[], pagination: { total: 0, limit: 100, has_more: false } }))
          )
        );

        forkJoin(reqs).subscribe({
          next: (results: PaginatedEventsResponse[]) => {
            const all: DisplayAlert[] = [];

            results.forEach((evRes: PaginatedEventsResponse, index: number) => {
              const bike = bikeList[index];
              const evs: BikeEventDTO[] = evRes.data || [];

              evs.forEach((ev: BikeEventDTO) => {
                all.push({
                  id: ev.id,
                  bikeId: bike.id,
                  plate: bike.registration_number,
                  model: bike.model || bike.label || '',
                  eventType: ev.event_type,
                  title: this.formatTitle(ev.event_type),
                  description: this.formatDescription(ev.event_type, bike.registration_number),
                  severity: this.classifySeverity(ev.event_type),
                  latitude: ev.latitude,
                  longitude: ev.longitude,
                  timestamp: ev.recorded_at,
                });
              });
            });

            // Sort newest first
            all.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
            this.alerts.set(all);
            this.isLoading.set(false);
          },
          error: () => {
            this.isLoading.set(false);
          },
        });
      },
      error: () => {
        this.isLoading.set(false);
      },
    });
  }

  private classifySeverity(type?: string): 'critical' | 'warning' | 'info' {
    const t = (type || '').toUpperCase();
    if (t.includes('BREACH') || t.includes('THEFT') || t.includes('CRASH') || t.includes('ACCIDENT') || t.includes('DURESS') || t.includes('SOS')) {
      return 'critical';
    }
    if (t.includes('TOUCH') || t.includes('TAMPER') || t.includes('TILT') || t.includes('LIFT') || t.includes('SHOCK') || t.includes('VIBRAT') || t.includes('BURGLARY')) {
      return 'warning';
    }
    return 'info';
  }

  private formatTitle(type: string): string {
    const t = (type || '').toUpperCase();
    if (t.includes('TILT') || t.includes('LIFT')) return `Bike Tilted or Moved`;
    if (t.includes('BURGLARY') || t.includes('THEFT')) return `Theft Alert`;
    if (t.includes('BREACH') || t.includes('EXIT')) return `Left Safe Area`;
    if (t.includes('ENTER') || t.includes('ENTRY')) return `Entered Safe Area`;
    if (t.includes('ACCIDENT') || t.includes('CRASH')) return `Possible Crash Detected`;
    if (t.includes('TAMPER') || t.includes('TOUCH') || t.includes('SHOCK') || t.includes('VIBRAT')) return `Tampering Detected`;
    if (t.includes('DURESS') || t.includes('SOS')) return `Emergency SOS Alert`;
    if (t.includes('SPEED')) return `High Speed Warning`;
    if (t.includes('POWER') || t.includes('BATTERY')) return `Low Battery Warning`;
    if (t.includes('ON') || t.includes('IGNITION_ON')) return `Engine Started`;
    if (t.includes('OFF') || t.includes('IGNITION_OFF')) return `Engine Turned Off`;
    if (t.includes('IMMOBILIZ')) return `Engine Locked`;
    if (t.includes('REACTIVAT')) return `Engine Unlocked`;
    return `Security Alert`;
  }

  private formatDescription(type: string, plate: string): string {
    const t = (type || '').toUpperCase();
    if (t.includes('TILT') || t.includes('LIFT')) return `Your motorcycle was moved or tilted while parked.`;
    if (t.includes('BURGLARY') || t.includes('THEFT')) return `Suspicious movement detected while your motorcycle is parked and locked.`;
    if (t.includes('BREACH')) return `Your motorcycle has moved outside of your safe boundary.`;
    if (t.includes('ENTER') || t.includes('ENTRY')) return `Your motorcycle has returned inside your safe boundary.`;
    if (t.includes('ACCIDENT') || t.includes('CRASH')) return `A sudden heavy impact was detected. Please verify rider safety.`;
    if (t.includes('TAMPER') || t.includes('TOUCH') || t.includes('SHOCK') || t.includes('VIBRAT')) return `Someone may be touching, shaking, or tampering with your motorcycle.`;
    if (t.includes('DURESS') || t.includes('SOS')) return `The emergency SOS button was pressed on your motorcycle.`;
    if (t.includes('SPEED')) return `Your motorcycle exceeded the maximum speed limit.`;
    if (t.includes('POWER') || t.includes('BATTERY')) return `Your motorcycle tracker battery is critically low.`;
    if (t.includes('ON')) return `The motorcycle ignition was switched on.`;
    if (t.includes('OFF')) return `The motorcycle engine was shut down.`;
    if (t.includes('IMMOBILIZ')) return `Engine locked remotely. Starter cannot be turned on.`;
    if (t.includes('REACTIVAT')) return `Engine unlocked. Your motorcycle is ready to ride.`;
    return `New security activity recorded for your motorcycle.`;
  }

  formatTimestamp(iso: string): string {
    if (!iso) return '';
    try {
      const d = new Date(iso);
      const now = Date.now();
      const diffMs = now - d.getTime();
      const diffMins = Math.floor(diffMs / 60000);

      if (diffMins < 1) return 'Just now';
      if (diffMins < 60) return `${diffMins}m ago`;
      const diffHours = Math.floor(diffMins / 60);
      if (diffHours < 24) return `${diffHours}h ago`;

      return d.toLocaleDateString('en-GB', {
        day: '2-digit',
        month: 'short',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return iso;
    }
  }
}
