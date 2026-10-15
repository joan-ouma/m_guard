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
  displayType: string;
  title: string;
  description: string;
  severity: 'critical' | 'warning' | 'info';
  latitude: number;
  longitude: number;
  timestamp: string;
  count?: number;
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
        displayType: this.formatDisplayType(evType),
        title: live.title && !live.title.toLowerCase().includes('security event:') ? live.title : this.formatTitle(evType, plate),
        description: live.body || live.description || this.formatDescription(evType, plate),
        severity: this.classifySeverity(evType),
        latitude: Number(live.latitude) || 0,
        longitude: Number(live.longitude) || 0,
        timestamp: live.recorded_at || live.recordedAt || new Date().toISOString(),
        count: 1,
      };

      this.alerts.update((prev) => {
        const existingIdx = prev.findIndex(
          (p) =>
            p.bikeId === newAlert.bikeId &&
            p.eventType.toUpperCase() === newAlert.eventType.toUpperCase() &&
            Math.abs(new Date(p.timestamp).getTime() - new Date(newAlert.timestamp).getTime()) < 30 * 60 * 1000
        );
        if (existingIdx !== -1) {
          const updated = [...prev];
          updated[existingIdx] = {
            ...updated[existingIdx],
            count: (updated[existingIdx].count || 1) + 1,
            timestamp: newAlert.timestamp,
            latitude: newAlert.latitude || updated[existingIdx].latitude,
            longitude: newAlert.longitude || updated[existingIdx].longitude,
          };
          return updated;
        }
        return [newAlert, ...prev];
      });
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
                  displayType: this.formatDisplayType(ev.event_type),
                  title: this.formatTitle(ev.event_type, bike.registration_number),
                  description: this.formatDescription(ev.event_type, bike.registration_number),
                  severity: this.classifySeverity(ev.event_type),
                  latitude: ev.latitude,
                  longitude: ev.longitude,
                  timestamp: ev.recorded_at,
                  count: 1,
                });
              });
            });

            // Sort newest first
            all.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

            // Group repetitive alerts (same bike & same eventType within 30 min)
            const deduplicated: DisplayAlert[] = [];
            all.forEach((item) => {
              const existing = deduplicated.find(
                (g) =>
                  g.bikeId === item.bikeId &&
                  g.eventType.toUpperCase() === item.eventType.toUpperCase() &&
                  Math.abs(new Date(g.timestamp).getTime() - new Date(item.timestamp).getTime()) < 30 * 60 * 1000
              );
              if (existing) {
                existing.count = (existing.count || 1) + 1;
              } else {
                deduplicated.push({ ...item, count: 1 });
              }
            });

            this.alerts.set(deduplicated);
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

  formatDisplayType(type?: string): string {
    const t = (type || '').toUpperCase();
    if (t.includes('TILT') || t.includes('LIFT')) return 'Tilt & Lift';
    if (t.includes('BURGLARY') || t.includes('THEFT')) return 'Theft Attempt';
    if (t.includes('BREACH') || t.includes('EXIT')) return 'Geofence Breach';
    if (t.includes('ENTER') || t.includes('ENTRY')) return 'Geofence Entry';
    if (t.includes('TAMPER') || t.includes('TOUCH') || t.includes('SHOCK') || t.includes('VIBRAT')) return 'Tampering';
    if (t.includes('ACCIDENT') || t.includes('CRASH')) return 'Impact / Crash';
    if (t.includes('DURESS') || t.includes('SOS')) return 'SOS Emergency';
    if (t.includes('SPEED')) return 'Overspeed';
    if (t.includes('POWER') || t.includes('BATTERY')) return 'Power Alert';
    if (t.includes('ON') || t.includes('IGNITION_ON')) return 'Ignition On';
    if (t.includes('OFF') || t.includes('IGNITION_OFF')) return 'Ignition Off';
    if (t.includes('IMMOBILIZ')) return 'Immobilized';
    if (t.includes('REACTIVAT')) return 'Reactivated';
    return type ? type.replace(/_/g, ' ') : 'Security Event';
  }

  private formatTitle(type: string, plate: string): string {
    const t = type.toUpperCase();
    if (t.includes('TILT') || t.includes('LIFT')) return `Vehicle Tilt & Lift Detected`;
    if (t.includes('BURGLARY') || t.includes('THEFT')) return `Potential Theft / Break-in Detected`;
    if (t.includes('BREACH')) return `Geofence Perimeter Breached`;
    if (t.includes('ENTER') || t.includes('ENTRY')) return `Geofence Safe Zone Entered`;
    if (t.includes('ACCIDENT') || t.includes('CRASH')) return `Accident / Crash Detected`;
    if (t.includes('TAMPER') || t.includes('TOUCH') || t.includes('SHOCK') || t.includes('VIBRAT')) return `Tampering / Vibration Detected`;
    if (t.includes('DURESS') || t.includes('SOS')) return `Emergency Driver SOS Signal`;
    if (t.includes('SPEED')) return `Speed Threshold Exceeded`;
    if (t.includes('POWER') || t.includes('BATTERY')) return `Tracker Power Cut / Low Battery`;
    if (t.includes('ON') || t.includes('IGNITION_ON')) return `Engine Started`;
    if (t.includes('OFF') || t.includes('IGNITION_OFF')) return `Engine Switched Off`;
    if (t.includes('IMMOBILIZ')) return `Starter Immobilized Remotely`;
    if (t.includes('REACTIVAT')) return `Starter Reactivated`;
    return `Security Event: ${type ? type.replace(/_/g, ' ') : 'Alert'}`;
  }

  private formatDescription(type: string, plate: string): string {
    const t = type.toUpperCase();
    if (t.includes('TILT') || t.includes('LIFT')) return `Gyro sensor detected unauthorized tilting or physical lifting while stationary.`;
    if (t.includes('BURGLARY') || t.includes('THEFT')) return `Suspicious movement and intrusion sensors triggered while parked.`;
    if (t.includes('BREACH')) return `Motorcycle ${plate} has exited its assigned safety geofence boundary.`;
    if (t.includes('ENTER') || t.includes('ENTRY')) return `Motorcycle ${plate} has entered the designated boundary.`;
    if (t.includes('ACCIDENT') || t.includes('CRASH')) return `Impact sensor triggered rapid deceleration. Immediate check advised.`;
    if (t.includes('TAMPER') || t.includes('TOUCH') || t.includes('SHOCK') || t.includes('VIBRAT')) return `Motion sensor detected unauthorized handling, vibration, or tampering.`;
    if (t.includes('DURESS') || t.includes('SOS')) return `Emergency assistance button engaged by rider on ${plate}.`;
    if (t.includes('SPEED')) return `Telemetry reported speed higher than established maximum safety limit.`;
    if (t.includes('POWER') || t.includes('BATTERY')) return `Tracker external power supply disconnected or battery level is critically low.`;
    if (t.includes('ON')) return `Electrical starter engaged for ${plate}.`;
    if (t.includes('OFF')) return `Engine turned off for ${plate}.`;
    if (t.includes('IMMOBILIZ')) return `Starter relay cut-off engaged remotely. Engine cannot be started.`;
    if (t.includes('REACTIVAT')) return `Starter cut-off relay restored. Motorcycle ready for normal operation.`;
    return `Security event logged for ${plate}. Verify live coordinates and vehicle status.`;
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
