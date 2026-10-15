import { Component, OnInit, computed, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { MobileNavComponent } from '../ui/mobile-nav/mobile-nav.component';
import { BikeService } from '../core/services/bike.service';
import { BikeDTO, BikeEventDTO } from '../core/models/bike.models';
import { forkJoin, of } from 'rxjs';
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
  template: `
<div class="flex h-screen w-screen bg-white font-sans text-zinc-900 overflow-hidden">

  <!-- LEFT NAVIGATION SIDEBAR (DESKTOP) -->
  <aside class="hidden lg:flex w-72 bg-white border-r border-zinc-200 flex-col justify-between shrink-0 z-30">
    <div>
      <div class="h-20 px-6 flex items-center gap-3.5 border-b border-zinc-100">
        <div class="w-11 h-11 rounded-full bg-zinc-900 flex items-center justify-center text-white shrink-0 shadow-xs">
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" /></svg>
        </div>
        <span class="font-bold text-xl tracking-tight text-zinc-900">MGuard</span>
      </div>
      <nav class="p-4 space-y-2">
        <a routerLink="/" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" /></svg> Home
        </a>
        <a routerLink="/bikes" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 17a2 2 0 11-4 0 2 2 0 014 0zM19 17a2 2 0 11-4 0 2 2 0 014 0z" /><path stroke-linecap="round" stroke-linejoin="round" d="M13 16V6a1 1 0 00-1-1H4a1 1 0 00-1 1v10a1 1 0 001 1h1m8-1a1 1 0 01-1 1H9m4-1V8a1 1 0 011-1h2.586a1 1 0 01.707.293l3.414 3.414a1 1 0 01.293.707V16a1 1 0 01-1 1h-1m-6-1a1 1 0 001 1h1M5 17a2 2 0 104 0m-4 0a2 2 0 114 0m6 0a2 2 0 104 0m-4 0a2 2 0 114 0" /></svg> Bikes
        </a>
        <a routerLink="/history" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg> History
        </a>
        <a routerLink="/" [fragment]="'geofence'" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" /></svg> Geofence
        </a>
        <a routerLink="/alerts" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" /></svg> Alerts
        </a>
        <a routerLink="/settings" routerLinkActive="bg-zinc-900 text-white shadow-xs" [routerLinkActiveOptions]="{ exact: true }" class="min-h-[52px] px-4 py-3.5 rounded-2xl flex items-center gap-4 text-base font-semibold text-zinc-500 hover:bg-zinc-100 transition-colors cursor-pointer">
          <svg class="w-6 h-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg> Settings
        </a>
      </nav>
    </div>
  </aside>

  <!-- MAIN VIEW -->
  <div class="flex-1 flex flex-col h-full overflow-hidden">
    <header class="h-16 lg:hidden bg-white border-b border-zinc-200 px-4 flex items-center justify-between shrink-0 z-20 relative">
      <div class="flex items-center gap-3">
        <app-mobile-nav></app-mobile-nav>
        <span class="font-bold text-base text-zinc-900">Alerts</span>
      </div>
      <button
        type="button"
        (click)="loadAlerts()"
        class="text-xs px-2.5 py-1 rounded-lg border border-zinc-200 text-zinc-600 hover:bg-zinc-100"
      >
        Refresh
      </button>
    </header>

    <div class="flex-1 overflow-y-auto bg-zinc-50 p-4 lg:p-8">
      <div class="max-w-4xl mx-auto w-full">
        <!-- Title & Header -->
        <div class="flex items-start justify-between gap-4 mb-6">
          <div>
            <h1 class="text-2xl lg:text-3xl font-black tracking-tight text-zinc-900 mb-1">Security Alerts &amp; Incidents</h1>
            <p class="text-zinc-500 text-sm">Real-time geofence breaches, tampering, theft triggers, and ignition events.</p>
          </div>
          <button
            type="button"
            (click)="loadAlerts()"
            [disabled]="isLoading()"
            class="hidden sm:inline-flex items-center gap-2 px-3.5 py-2 text-xs font-bold rounded-xl border border-zinc-200 bg-white hover:bg-zinc-100 text-zinc-800 transition-colors shadow-xs cursor-pointer disabled:opacity-50"
          >
            <svg class="w-4 h-4" [class.animate-spin]="isLoading()" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {{ isLoading() ? 'Syncing...' : 'Sync Alerts' }}
          </button>
        </div>

        <!-- Metric Badges -->
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
          <div class="p-4 rounded-2xl bg-white border border-zinc-200 shadow-xs">
            <span class="text-xs font-semibold text-zinc-400 block mb-1">Total Events</span>
            <span class="text-2xl font-black text-zinc-900">{{ alerts().length }}</span>
          </div>
          <div class="p-4 rounded-2xl bg-white border border-rose-200/80 shadow-xs">
            <span class="text-xs font-semibold text-rose-600 block mb-1">Critical (Breach/Theft)</span>
            <span class="text-2xl font-black text-rose-600">{{ criticalCount() }}</span>
          </div>
          <div class="p-4 rounded-2xl bg-white border border-amber-200/80 shadow-xs">
            <span class="text-xs font-semibold text-amber-600 block mb-1">Warnings (Tamper)</span>
            <span class="text-2xl font-black text-amber-600">{{ warningCount() }}</span>
          </div>
          <div class="p-4 rounded-2xl bg-white border border-zinc-200 shadow-xs">
            <span class="text-xs font-semibold text-zinc-500 block mb-1">Active Bikes</span>
            <span class="text-2xl font-black text-zinc-900">{{ bikes().length }}</span>
          </div>
        </div>

        <!-- Filters -->
        <div class="flex flex-wrap items-center gap-2 mb-6">
          <button
            type="button"
            (click)="selectedSeverity.set('ALL')"
            [class.bg-zinc-900]="selectedSeverity() === 'ALL'"
            [class.text-white]="selectedSeverity() === 'ALL'"
            [class.bg-white]="selectedSeverity() !== 'ALL'"
            [class.text-zinc-600]="selectedSeverity() !== 'ALL'"
            class="px-3.5 py-1.5 rounded-xl text-xs font-bold border border-zinc-200 hover:border-zinc-300 transition-colors cursor-pointer"
          >
            All Alerts ({{ alerts().length }})
          </button>
          <button
            type="button"
            (click)="selectedSeverity.set('critical')"
            [class.bg-rose-600]="selectedSeverity() === 'critical'"
            [class.text-white]="selectedSeverity() === 'critical'"
            [class.bg-white]="selectedSeverity() !== 'critical'"
            [class.text-rose-700]="selectedSeverity() !== 'critical'"
            class="px-3.5 py-1.5 rounded-xl text-xs font-bold border border-rose-200 hover:border-rose-300 transition-colors cursor-pointer"
          >
            Critical ({{ criticalCount() }})
          </button>
          <button
            type="button"
            (click)="selectedSeverity.set('warning')"
            [class.bg-amber-500]="selectedSeverity() === 'warning'"
            [class.text-white]="selectedSeverity() === 'warning'"
            [class.bg-white]="selectedSeverity() !== 'warning'"
            [class.text-amber-800]="selectedSeverity() !== 'warning'"
            class="px-3.5 py-1.5 rounded-xl text-xs font-bold border border-amber-200 hover:border-amber-300 transition-colors cursor-pointer"
          >
            Warnings ({{ warningCount() }})
          </button>
          <button
            type="button"
            (click)="selectedSeverity.set('info')"
            [class.bg-zinc-700]="selectedSeverity() === 'info'"
            [class.text-white]="selectedSeverity() === 'info'"
            [class.bg-white]="selectedSeverity() !== 'info'"
            [class.text-zinc-600]="selectedSeverity() !== 'info'"
            class="px-3.5 py-1.5 rounded-xl text-xs font-bold border border-zinc-200 hover:border-zinc-300 transition-colors cursor-pointer"
          >
            Info / Ignition ({{ infoCount() }})
          </button>
        </div>

        <!-- Alert Cards Feed -->
        @if (isLoading()) {
          <div class="p-12 text-center text-zinc-400 text-sm flex flex-col items-center gap-3">
            <svg class="w-8 h-8 animate-spin text-zinc-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span>Loading security events...</span>
          </div>
        } @else if (filteredAlerts().length === 0) {
          <div class="bg-white rounded-2xl border border-zinc-200 p-10 text-center">
            <div class="w-16 h-16 bg-zinc-50 rounded-full flex items-center justify-center mx-auto mb-4 text-zinc-400">
              <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <h3 class="text-lg font-bold text-zinc-900 mb-1">No Alerts Recorded</h3>
            <p class="text-zinc-500 text-sm max-w-sm mx-auto">
              No security breaches or incidents detected for the selected filter. Live alerts will trigger here in real time.
            </p>
          </div>
        } @else {
          <div class="space-y-3">
            @for (alert of filteredAlerts(); track alert.id) {
              <div
                class="p-4 sm:p-5 rounded-2xl border transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-white shadow-xs"
                [class.border-rose-300]="alert.severity === 'critical'"
                [class.bg-rose-50/20]="alert.severity === 'critical'"
                [class.border-amber-300]="alert.severity === 'warning'"
                [class.border-zinc-200]="alert.severity === 'info'"
              >
                <div class="flex items-start gap-3.5 min-w-0">
                  <div
                    class="w-10 h-10 rounded-xl flex items-center justify-center shrink-0 text-base"
                    [class.bg-rose-100]="alert.severity === 'critical'"
                    [class.text-rose-700]="alert.severity === 'critical'"
                    [class.bg-amber-100]="alert.severity === 'warning'"
                    [class.text-amber-800]="alert.severity === 'warning'"
                    [class.bg-zinc-100]="alert.severity === 'info'"
                    [class.text-zinc-700]="alert.severity === 'info'"
                  >
                    @if (alert.severity === 'critical') {
                      🚨
                    } @else if (alert.severity === 'warning') {
                      ⚠️
                    } @else {
                      ⚡
                    }
                  </div>

                  <div class="min-w-0">
                    <div class="flex items-center gap-2 flex-wrap">
                      <span class="text-xs font-black uppercase tracking-wider text-zinc-900 bg-zinc-100 px-2 py-0.5 rounded-md">
                        {{ alert.plate }}
                      </span>
                      <span class="text-xs font-semibold text-zinc-400">
                        {{ alert.model }}
                      </span>
                      <span
                        class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full"
                        [class.bg-rose-100]="alert.severity === 'critical'"
                        [class.text-rose-700]="alert.severity === 'critical'"
                        [class.bg-amber-100]="alert.severity === 'warning'"
                        [class.text-amber-800]="alert.severity === 'warning'"
                        [class.bg-zinc-100]="alert.severity === 'info'"
                        [class.text-zinc-600]="alert.severity === 'info'"
                      >
                        {{ alert.eventType }}
                      </span>
                    </div>

                    <h4 class="text-sm font-bold text-zinc-900 mt-1">{{ alert.title }}</h4>
                    <p class="text-xs text-zinc-500 mt-0.5">{{ alert.description }}</p>

                    <div class="flex items-center gap-3 mt-2 text-[11px] text-zinc-400">
                      <span>{{ formatTimestamp(alert.timestamp) }}</span>
                      @if (alert.latitude && alert.longitude) {
                        <span>•</span>
                        <span class="font-mono">{{ alert.latitude | number:'1.4-4' }}, {{ alert.longitude | number:'1.4-4' }}</span>
                      }
                    </div>
                  </div>
                </div>

                <div class="flex items-center gap-2 shrink-0 self-end sm:self-center">
                  <a
                    [routerLink]="['/history']"
                    [queryParams]="{ bike: alert.bikeId }"
                    class="text-xs font-bold px-3 py-1.5 rounded-xl border border-zinc-200 hover:bg-zinc-900 hover:text-white transition-colors cursor-pointer"
                  >
                    View History
                  </a>
                  <a
                    [routerLink]="['/']"
                    [queryParams]="{ bike: alert.bikeId }"
                    class="text-xs font-bold px-3 py-1.5 rounded-xl bg-zinc-900 text-white hover:bg-black transition-colors cursor-pointer"
                  >
                    Locate
                  </a>
                </div>
              </div>
            }
          </div>
        }
      </div>
    </div>
  </div>
</div>
  `
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

      const newAlert: DisplayAlert = {
        id: 'live_' + Date.now(),
        bikeId: live.bike_id || live.bikeId || '',
        plate: live.plate || live.registration_number || 'Motorcycle',
        model: live.model || '',
        eventType: live.event_type || live.eventType || 'ALERT',
        title: live.title || 'Security Alert',
        description: live.body || live.description || 'Unexpected security event reported.',
        severity: this.classifySeverity(live.event_type || live.eventType),
        latitude: Number(live.latitude) || 0,
        longitude: Number(live.longitude) || 0,
        timestamp: live.recorded_at || live.recordedAt || new Date().toISOString(),
      };

      this.alerts.update((prev) => {
        if (prev.some((p) => p.title === newAlert.title && p.timestamp === newAlert.timestamp)) return prev;
        return [newAlert, ...prev];
      });
    });
  }

  ngOnInit(): void {
    this.loadAlerts();
  }

  loadAlerts(): void {
    this.isLoading.set(true);

    this.bikeService.getBikes().subscribe({
      next: (res) => {
        const bikeList = res.data || [];
        this.bikes.set(bikeList);

        if (bikeList.length === 0) {
          this.alerts.set([]);
          this.isLoading.set(false);
          return;
        }

        // Query events across all bikes
        const reqs = bikeList.map((bike) =>
          this.bikeService.getBikeEvents(bike.id, 100).pipe(
            catchError(() => of({ data: [] as BikeEventDTO[], pagination: { total: 0, limit: 100, has_more: false } }))
          )
        );

        forkJoin(reqs).subscribe({
          next: (results) => {
            const all: DisplayAlert[] = [];

            results.forEach((evRes, index) => {
              const bike = bikeList[index];
              const evs = evRes.data || [];

              evs.forEach((ev) => {
                all.push({
                  id: ev.id,
                  bikeId: bike.id,
                  plate: bike.registration_number,
                  model: bike.model || bike.label || '',
                  eventType: ev.event_type,
                  title: this.formatTitle(ev.event_type, bike.registration_number),
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
    if (t.includes('BREACH') || t.includes('THEFT') || t.includes('CRASH') || t.includes('ACCIDENT')) {
      return 'critical';
    }
    if (t.includes('TOUCH') || t.includes('TAMPER') || t.includes('TILT') || t.includes('SHOCK')) {
      return 'warning';
    }
    return 'info';
  }

  private formatTitle(type: string, plate: string): string {
    const t = type.toUpperCase();
    if (t.includes('BREACH')) return `Geofence Perimeter Breached`;
    if (t.includes('THEFT')) return `Potential Vehicle Theft`;
    if (t.includes('ACCIDENT') || t.includes('CRASH')) return `Accident / Crash Detected`;
    if (t.includes('TOUCH') || t.includes('TAMPER')) return `Tampering / Vibration Detected`;
    if (t.includes('ON') || t.includes('IGNITION_ON')) return `Ignition Started`;
    if (t.includes('OFF') || t.includes('IGNITION_OFF')) return `Ignition Switched Off`;
    if (t.includes('IMMOBILIZ')) return `Starter Immobilized`;
    if (t.includes('REACTIVAT')) return `Starter Reactivated`;
    return `Security Event: ${type}`;
  }

  private formatDescription(type: string, plate: string): string {
    const t = type.toUpperCase();
    if (t.includes('BREACH')) return `Motorcycle ${plate} has exited its assigned safety geofence boundary.`;
    if (t.includes('THEFT')) return `Suspicious movement detected while vehicle was parked.`;
    if (t.includes('ACCIDENT') || t.includes('CRASH')) return `Impact sensor triggered rapid deceleration.`;
    if (t.includes('TOUCH') || t.includes('TAMPER')) return `Motion sensor detected unauthorized handling or tilt.`;
    if (t.includes('ON')) return `Electrical starter engaged for ${plate}.`;
    if (t.includes('OFF')) return `Engine turned off for ${plate}.`;
    return `Event ${type} logged for motorcycle ${plate}.`;
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
