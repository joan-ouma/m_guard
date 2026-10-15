import { Component, OnInit, computed, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { MobileNavComponent } from '../ui/mobile-nav/mobile-nav.component';
import { BikeService } from '../core/services/bike.service';
import { BikeEventDTO, PaginatedEventsResponse } from '../core/models/bike.models';

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
  readonly alerts = signal<DisplayAlert[]>([]);
  readonly selectedSeverity = signal<'ALL' | 'critical' | 'warning' | 'info'>('ALL');

  // Aggregate metrics from backend
  readonly totalEvents = signal<number>(0);
  readonly criticalCount = signal<number>(0);
  readonly warningCount = signal<number>(0);
  readonly infoCount = signal<number>(0);
  readonly activeBikesCount = signal<number>(0);

  // Cursor pagination state
  readonly pageSize = signal<number>(10);
  readonly pageSizes = [10, 25, 50, 100];
  readonly cursorStack = signal<string[]>([]);
  readonly nextCursor = signal<string | null>(null);
  readonly hasMore = signal<boolean>(false);

  readonly currentPage = computed<number>(() => this.cursorStack().length + 1);
  readonly hasPrev = computed<boolean>(() => this.cursorStack().length > 0);
  readonly hasNext = computed<boolean>(() => this.hasMore());
  readonly totalPages = computed<number>(() => {
    return Math.max(1, Math.ceil(this.totalEvents() / this.pageSize()));
  });

  constructor() {
    // Listen to live SSE alerts from the backend
    effect(() => {
      const live = this.bikeService.latestAlert();
      if (!live) return;

      const evType = live.event_type || live.eventType || 'ALERT';
      const plate = live.plate || live.registration_number || 'Motorcycle';
      const sev = this.classifySeverity(evType);

      const newAlert: DisplayAlert = {
        id: 'live_' + Date.now(),
        bikeId: live.bike_id || live.bikeId || '',
        plate: plate,
        model: live.model || '',
        eventType: evType,
        title: this.formatTitle(evType),
        description: this.formatDescription(evType, plate),
        severity: sev,
        latitude: Number(live.latitude) || 0,
        longitude: Number(live.longitude) || 0,
        timestamp: live.recorded_at || live.recordedAt || new Date().toISOString(),
      };

      // If user is on page 1, prepend immediately
      if (this.cursorStack().length === 0) {
        const curSev = this.selectedSeverity();
        if (curSev === 'ALL' || curSev === sev) {
          this.alerts.update((prev) => [newAlert, ...prev]);
        }
      }

      this.totalEvents.update((n) => n + 1);
      if (sev === 'critical') this.criticalCount.update((c) => c + 1);
      else if (sev === 'warning') this.warningCount.update((c) => c + 1);
      else if (sev === 'info') this.infoCount.update((c) => c + 1);
    });
  }

  ngOnInit(): void {
    // Load bikes count for top metric card
    this.bikeService.loadBikes().subscribe({
      next: (res) => this.activeBikesCount.set(res.data?.length || 0),
    });

    this.loadAlerts();
  }

  loadAlerts(): void {
    this.isLoading.set(true);

    const cursor = this.cursorStack().length > 0 
      ? this.cursorStack()[this.cursorStack().length - 1] 
      : undefined;

    this.bikeService.getAllEvents(this.pageSize(), cursor, this.selectedSeverity()).subscribe({
      next: (res: PaginatedEventsResponse) => {
        const list: DisplayAlert[] = (res.data || []).map((ev: BikeEventDTO) => ({
          id: ev.id,
          bikeId: ev.bike_id,
          plate: ev.registration_number || 'Motorcycle',
          model: ev.model || '',
          eventType: ev.event_type,
          title: this.formatTitle(ev.event_type),
          description: this.formatDescription(ev.event_type, ev.registration_number || 'Motorcycle'),
          severity: this.classifySeverity(ev.event_type),
          latitude: ev.latitude,
          longitude: ev.longitude,
          timestamp: ev.recorded_at,
        }));

        this.alerts.set(list);
        this.nextCursor.set(res.pagination?.next_cursor || null);
        this.hasMore.set(res.pagination?.has_more || false);
        this.totalEvents.set(res.pagination?.total || 0);

        if (res.critical_count !== undefined) this.criticalCount.set(res.critical_count);
        if (res.warning_count !== undefined) this.warningCount.set(res.warning_count);
        if (res.info_count !== undefined) this.infoCount.set(res.info_count);

        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
      },
    });
  }

  prevPage(): void {
    if (this.cursorStack().length > 0) {
      this.cursorStack.update((stack) => stack.slice(0, -1));
      this.loadAlerts();
    }
  }

  nextPage(): void {
    const next = this.nextCursor();
    if (next && this.hasMore()) {
      this.cursorStack.update((stack) => [...stack, next]);
      this.loadAlerts();
    }
  }

  setPageSize(size: number): void {
    this.pageSize.set(size);
    this.cursorStack.set([]);
    this.loadAlerts();
  }

  onPageSizeChange(event: Event): void {
    const val = parseInt((event.target as HTMLSelectElement).value, 10);
    if (!isNaN(val) && val > 0) {
      this.setPageSize(val);
    }
  }

  setSeverity(sev: 'ALL' | 'critical' | 'warning' | 'info'): void {
    this.selectedSeverity.set(sev);
    this.cursorStack.set([]);
    this.loadAlerts();
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
