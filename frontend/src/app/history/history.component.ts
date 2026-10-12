import {
  Component,
  OnInit,
  AfterViewInit,
  OnDestroy,
  inject,
  signal,
  computed,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { forkJoin, of, catchError } from 'rxjs';
import { AuthService } from '../core/services/auth.service';
import { BikeService } from '../core/services/bike.service';
import { BikeDTO, BreadcrumbDTO, BikeEventDTO } from '../core/models/bike.models';
import * as L from 'leaflet';

export interface TrailEvent {
  id: number | string;
  time: string;
  type: 'BIKE_ON' | 'BIKE_OFF' | 'MOVING' | 'IDLE' | 'BIKE_TOUCHED' | 'GEOFENCE_APPROACHING' | 'ACCIDENT_DETECTED' | 'IMMOBILIZED' | 'REACTIVATED';
  title: string;
  speed: number;
  heading: number;
  ignition: boolean;
  battery: number;
  address: string;
  lat: number;
  lng: number;
}

@Component({
  selector: 'app-history',
  standalone: true,
  imports: [CommonModule, RouterLink, FormsModule],
  templateUrl: './history.component.html',
})
export class HistoryComponent implements OnInit, AfterViewInit, OnDestroy {
  readonly authService = inject(AuthService);
  readonly bikeService = inject(BikeService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  // View mode: 'text' (default) or 'map'
  readonly viewMode = signal<'text' | 'map'>('text');

  // Filter state
  readonly selectedDate = signal<string>(this.formatDate(new Date()));
  readonly bikes = signal<BikeDTO[]>([]);
  readonly selectedBikeId = signal<string>('');
  readonly breadcrumbs = signal<BreadcrumbDTO[]>([]);
  readonly bikeEvents = signal<BikeEventDTO[]>([]);
  readonly isLoading = signal<boolean>(false);

  // Leaflet references
  private map: L.Map | null = null;
  private pathPolyline: L.Polyline | null = null;
  private pathGlow: L.Polyline | null = null;
  private startMarker: L.Marker | null = null;
  private endMarker: L.Marker | null = null;
  private eventMarkers: L.Marker[] = [];

  // Computed signals
  readonly selectedBike = computed<BikeDTO | null>(() => {
    const id = this.selectedBikeId();
    return this.bikes().find((b) => b.id === id) || this.bikes()[0] || null;
  });

  readonly isSelectedDateToday = computed<boolean>(() => {
    return this.selectedDate() === this.formatDate(new Date());
  });

  // Chronological trail & events list for the selected day
  readonly trailEvents = computed<TrailEvent[]>(() => {
    const pts = this.breadcrumbs();
    const bevs = this.bikeEvents();
    if ((!pts || pts.length === 0) && (!bevs || bevs.length === 0)) return [];

    const events: TrailEvent[] = [];

    // Helper to find nearest address from breadcrumbs
    const findNearestAddress = (lat: number, lng: number): string => {
      if (!pts || pts.length === 0) return 'Calculating...';
      let closest = pts[0];
      let minDist = 999999;
      for (const p of pts) {
        const d = this.haversine(lat, lng, p.latitude, p.longitude);
        if (d < minDist) {
          minDist = d;
          closest = p;
        }
      }
      return closest.address || 'Calculating...';
    };

    for (let i = 0; i < pts.length; i++) {
      const p = pts[i];
      const prev = i > 0 ? pts[i - 1] : null;

      const isBikeOn = p.ignition && (!prev || !prev.ignition);
      const isBikeOff = !p.ignition && prev && prev.ignition;

      let eventType: 'BIKE_ON' | 'BIKE_OFF' | 'MOVING' | 'IDLE';
      let title: string;

      if (isBikeOn) {
        eventType = 'BIKE_ON';
        title = 'Bike Switched ON';
      } else if (isBikeOff) {
        eventType = 'BIKE_OFF';
        title = 'Bike Switched OFF';
      } else if (p.speed_kmh > 0) {
        eventType = 'MOVING';
        title = `Moving • ${p.speed_kmh} km/h`;
      } else {
        eventType = 'IDLE';
        title = p.ignition ? 'Idling (Engine ON)' : 'Parked';
      }

      events.push({
        id: p.id || `bc-${i + 1}`,
        time: p.recorded_at,
        type: eventType,
        title,
        speed: p.speed_kmh,
        heading: p.heading_deg,
        ignition: p.ignition,
        battery: p.battery_percent,
        address: p.address || 'Calculating...',
        lat: p.latitude,
        lng: p.longitude,
      });
    }

    for (const ev of bevs) {
      let type: 'BIKE_TOUCHED' | 'GEOFENCE_APPROACHING' | 'ACCIDENT_DETECTED' | 'BIKE_ON' | 'BIKE_OFF' | 'IMMOBILIZED' | 'REACTIVATED' = 'BIKE_TOUCHED';
      let title = 'Bike touched while parked';

      if (ev.event_type === 'ACCIDENT_DETECTED') {
        type = 'ACCIDENT_DETECTED';
        title = 'Accident Incident Detected';
      } else if (ev.event_type === 'BIKE_TOUCHED') {
        type = 'BIKE_TOUCHED';
        title = 'Bike touched while parked';
      } else if (ev.event_type === 'GEOFENCE_APPROACHING') {
        type = 'GEOFENCE_APPROACHING';
        title = 'Bike near geofence border';
      } else if (ev.event_type === 'IGNITION_ON') {
        type = 'BIKE_ON';
        title = 'Bike Switched ON';
      } else if (ev.event_type === 'IGNITION_OFF') {
        type = 'BIKE_OFF';
        title = 'Bike Switched OFF';
      } else if (ev.event_type === 'IMMOBILIZED' || ev.event_type === 'BIKE_IMMOBILIZED') {
        type = 'IMMOBILIZED';
        title = 'Bike Immobilized';
      } else if (ev.event_type === 'REACTIVATED' || ev.event_type === 'BIKE_REACTIVATED') {
        type = 'REACTIVATED';
        title = 'Bike Reactivated';
      } else {
        type = 'BIKE_TOUCHED';
        title = ev.event_type.replace(/_/g, ' ');
      }

      events.push({
        id: `ev-${ev.id}`,
        time: ev.recorded_at,
        type,
        title,
        speed: 0,
        heading: 0,
        ignition: type === 'BIKE_ON',
        battery: 95,
        address: findNearestAddress(ev.latitude, ev.longitude),
        lat: ev.latitude,
        lng: ev.longitude,
      });
    }

    // Sort in reverse chronological order (latest first)
    return events.sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime());
  });

  readonly totalDistanceKm = computed<number>(() => {
    const pts = this.breadcrumbs();
    if (pts.length < 2) return 0;
    let total = 0;
    for (let i = 1; i < pts.length; i++) {
      total += this.haversine(pts[i - 1].latitude, pts[i - 1].longitude, pts[i].latitude, pts[i].longitude);
    }
    return Math.round(total * 10) / 10;
  });

  readonly onEventsCount = computed<number>(() => {
    return this.trailEvents().filter((e) => e.type === 'BIKE_ON').length;
  });

  readonly offEventsCount = computed<number>(() => {
    return this.trailEvents().filter((e) => e.type === 'BIKE_OFF').length;
  });

  readonly touchEventsCount = computed<number>(() => {
    return this.trailEvents().filter((e) => e.type === 'BIKE_TOUCHED').length;
  });

  readonly geofenceEventsCount = computed<number>(() => {
    return this.trailEvents().filter((e) => e.type === 'GEOFENCE_APPROACHING').length;
  });

  readonly accidentEventsCount = computed<number>(() => {
    return this.trailEvents().filter((e) => e.type === 'ACCIDENT_DETECTED').length;
  });

  readonly immobilizedEventsCount = computed<number>(() => {
    return this.trailEvents().filter((e) => e.type === 'IMMOBILIZED').length;
  });

  readonly reactivatedEventsCount = computed<number>(() => {
    return this.trailEvents().filter((e) => e.type === 'REACTIVATED').length;
  });

  ngOnInit(): void {
    this.bikeService.loadBikes().subscribe({
      next: (res) => {
        this.bikes.set(res.data);

        const bikeParam = this.route.snapshot.queryParamMap.get('bike');
        const dateParam = this.route.snapshot.queryParamMap.get('date');

        if (dateParam) {
          this.selectedDate.set(dateParam);
        }

        const validBike =
          bikeParam && res.data.some((b) => b.id === bikeParam)
            ? bikeParam
            : res.data.length > 0
            ? res.data[0].id
            : '';

        if (validBike) {
          this.selectedBikeId.set(validBike);
          this.fetchBreadcrumbs();
        }
      },
    });

    this.route.queryParamMap.subscribe((params) => {
      const bikeParam = params.get('bike');
      const dateParam = params.get('date');

      let changed = false;
      if (bikeParam && bikeParam !== this.selectedBikeId() && this.bikes().some((b) => b.id === bikeParam)) {
        this.selectedBikeId.set(bikeParam);
        changed = true;
      }
      if (dateParam && dateParam !== this.selectedDate()) {
        this.selectedDate.set(dateParam);
        changed = true;
      }
      if (changed) {
        this.fetchBreadcrumbs();
      }
    });
  }

  ngAfterViewInit(): void {
    // Leaflet map initialized lazily
  }

  ngOnDestroy(): void {
    this.clearMapLayers();
    if (this.map) {
      this.map.remove();
      this.map = null;
    }
  }

  fetchBreadcrumbs(): void {
    const bikeId = this.selectedBikeId();
    const date = this.selectedDate();
    if (!bikeId || !date) return;

    this.isLoading.set(true);

    forkJoin({
      breadcrumbs: this.bikeService.getBikeBreadcrumbs(bikeId, 1000, date).pipe(
        catchError(() => of({ data: [] as BreadcrumbDTO[] }))
      ),
      events: this.bikeService.getBikeEvents(bikeId, 1000, date).pipe(
        catchError(() => of({ data: [] as BikeEventDTO[] }))
      ),
    }).subscribe({
      next: ({ breadcrumbs, events }) => {
        this.breadcrumbs.set(breadcrumbs.data || []);
        this.bikeEvents.set(events.data || []);
        this.isLoading.set(false);

        if (this.viewMode() === 'map') {
          this.renderRouteOnMap();
        }
      },
      error: () => {
        this.breadcrumbs.set([]);
        this.bikeEvents.set([]);
        this.isLoading.set(false);
        this.clearMapLayers();
      },
    });
  }

  selectBike(bikeId: string): void {
    this.selectedBikeId.set(bikeId);
    this.updateQueryParams();
    this.fetchBreadcrumbs();
  }

  prevDay(): void {
    const cur = new Date(this.selectedDate());
    cur.setDate(cur.getDate() - 1);
    this.selectedDate.set(this.formatDate(cur));
    this.updateQueryParams();
    this.fetchBreadcrumbs();
  }

  nextDay(): void {
    if (this.isSelectedDateToday()) return;
    const cur = new Date(this.selectedDate());
    cur.setDate(cur.getDate() + 1);
    this.selectedDate.set(this.formatDate(cur));
    this.updateQueryParams();
    this.fetchBreadcrumbs();
  }

  setPreset(preset: 'today' | 'yesterday'): void {
    const d = new Date();
    if (preset === 'yesterday') {
      d.setDate(d.getDate() - 1);
    }
    this.selectedDate.set(this.formatDate(d));
    this.updateQueryParams();
    this.fetchBreadcrumbs();
  }

  onCustomDateChange(event: Event): void {
    const target = event.target as HTMLInputElement;
    if (target && target.value) {
      this.selectedDate.set(target.value);
      this.updateQueryParams();
      this.fetchBreadcrumbs();
    }
  }

  seeTrailInMap(): void {
    this.viewMode.set('map');
    setTimeout(() => {
      if (!this.map) {
        this.initMap();
      } else {
        this.map.invalidateSize();
        this.renderRouteOnMap();
      }
    }, 100);
  }

  backToTextualData(): void {
    this.viewMode.set('text');
  }

  private initMap(): void {
    const mapEl = document.getElementById('historyMap');
    if (!mapEl) return;

    this.map = L.map('historyMap', {
      zoomControl: false,
    }).setView([-1.414046, 36.744676], 13);

    L.tileLayer(
      'https://server.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Base/MapServer/tile/{z}/{y}/{x}',
      {
        maxZoom: 16,
        attribution: '&copy; Esri',
      }
    ).addTo(this.map);

    L.tileLayer(
      'https://server.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Reference/MapServer/tile/{z}/{y}/{x}',
      {
        maxZoom: 16,
        opacity: 0.45,
        attribution: '',
      }
    ).addTo(this.map);

    if (this.breadcrumbs().length > 0) {
      this.renderRouteOnMap();
    }
  }

  private renderRouteOnMap(): void {
    if (!this.map) return;
    this.clearMapLayers();

    const pts = this.breadcrumbs().filter((p) => p.latitude !== 0 && p.longitude !== 0);
    if (pts.length === 0) return;

    const coords: [number, number][] = pts.map((p) => [p.latitude, p.longitude]);

    this.pathGlow = L.polyline(coords, {
      color: '#38BDF8',
      weight: 8,
      opacity: 0.35,
      lineCap: 'round',
      lineJoin: 'round',
    }).addTo(this.map);

    this.pathPolyline = L.polyline(coords, {
      color: '#0284C7',
      weight: 4,
      opacity: 0.95,
      lineCap: 'round',
      lineJoin: 'round',
    }).addTo(this.map);

    // Start Pin (A)
    const startPt = pts[0];
    const startIcon = L.divIcon({
      className: 'custom-trip-marker',
      html: `
        <div class="w-8 h-8 rounded-full bg-green-700 text-white flex items-center justify-center font-bold text-xs shadow-lg border-2 border-white ring-2 ring-green-700/50">
          A
        </div>
      `,
      iconSize: [32, 32],
      iconAnchor: [16, 16],
    });
    this.startMarker = L.marker([startPt.latitude, startPt.longitude], { icon: startIcon })
      .addTo(this.map)
      .bindTooltip(`Day Origin: ${this.formatTime(startPt.recorded_at)}`, { direction: 'top', offset: [0, -10] });

    // End Pin (B)
    const endPt = pts[pts.length - 1];
    const endIcon = L.divIcon({
      className: 'custom-trip-marker',
      html: `
        <div class="w-8 h-8 rounded-full bg-rose-500 text-white flex items-center justify-center font-bold text-xs shadow-lg border-2 border-white ring-2 ring-rose-500/50">
          B
        </div>
      `,
      iconSize: [32, 32],
      iconAnchor: [16, 16],
    });
    this.endMarker = L.marker([endPt.latitude, endPt.longitude], { icon: endIcon })
      .addTo(this.map)
      .bindTooltip(`Last Location: ${this.formatTime(endPt.recorded_at)}`, { direction: 'top', offset: [0, -10] });

    // Plot accident, touch, geofence, and immobilization events on the map
    const specialEvents = this.trailEvents().filter(
      (e) => e.type === 'BIKE_TOUCHED' || e.type === 'GEOFENCE_APPROACHING' || e.type === 'ACCIDENT_DETECTED' || e.type === 'IMMOBILIZED' || e.type === 'REACTIVATED'
    );

    for (const sev of specialEvents) {
      const isAccident = sev.type === 'ACCIDENT_DETECTED';
      const isTouch = sev.type === 'BIKE_TOUCHED';
      const isImmobilized = sev.type === 'IMMOBILIZED';
      const isReactivated = sev.type === 'REACTIVATED';
      const bgClass = isAccident
        ? 'bg-red-600 ring-red-500/50 animate-bounce'
        : isTouch
        ? 'bg-rose-600 ring-rose-500/40 animate-pulse'
        : isImmobilized
        ? 'bg-rose-600 ring-rose-500/40'
        : isReactivated
        ? 'bg-green-700 ring-green-600/40'
        : 'bg-violet-600 ring-violet-500/40';

      const sevIcon = L.divIcon({
        className: 'custom-event-marker',
        html: `
          <div class="w-8 h-8 rounded-full ${bgClass} text-white flex items-center justify-center font-bold text-xs shadow-xl border-2 border-white ring-4">
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              ${isAccident
                ? '<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />'
                : isTouch 
                ? '<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />'
                : isImmobilized || isReactivated
                ? '<path stroke-linecap="round" stroke-linejoin="round" d="M5.636 5.636a9 9 0 1012.728 0M12 3v9" />'
                : '<path stroke-linecap="round" stroke-linejoin="round" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />'
              }
            </svg>
          </div>
        `,
        iconSize: [32, 32],
        iconAnchor: [16, 16],
      });

      const m = L.marker([sev.lat, sev.lng], { icon: sevIcon })
        .addTo(this.map)
        .bindTooltip(`<b>${sev.title}</b><br>${this.formatTime(sev.time)} • ${sev.address}`, { direction: 'top', offset: [0, -10] });
      this.eventMarkers.push(m);
    }

    const bounds = L.latLngBounds(coords);
    this.map.fitBounds(bounds, { padding: [50, 50], maxZoom: 15 });
  }

  private clearMapLayers(): void {
    if (!this.map) return;
    if (this.pathPolyline) {
      this.map.removeLayer(this.pathPolyline);
      this.pathPolyline = null;
    }
    if (this.pathGlow) {
      this.map.removeLayer(this.pathGlow);
      this.pathGlow = null;
    }
    if (this.startMarker) {
      this.map.removeLayer(this.startMarker);
      this.startMarker = null;
    }
    if (this.endMarker) {
      this.map.removeLayer(this.endMarker);
      this.endMarker = null;
    }
    for (const m of this.eventMarkers) {
      this.map.removeLayer(m);
    }
    this.eventMarkers = [];
  }

  private updateQueryParams(): void {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {
        bike: this.selectedBikeId(),
        date: this.selectedDate(),
      },
      queryParamsHandling: 'merge',
    });
  }

  private haversine(lat1: number, lon1: number, lat2: number, lon2: number): number {
    const R = 6371;
    const dLat = this.deg2rad(lat2 - lat1);
    const dLon = this.deg2rad(lon2 - lon1);
    const a =
      Math.sin(dLat / 2) * Math.sin(dLat / 2) +
      Math.cos(this.deg2rad(lat1)) *
        Math.cos(this.deg2rad(lat2)) *
        Math.sin(dLon / 2) *
        Math.sin(dLon / 2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    return R * c;
  }

  private deg2rad(deg: number): number {
    return deg * (Math.PI / 180);
  }

  formatDate(d: Date): string {
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  formatReadableDate(dateStr: string): string {
    if (!dateStr) return '';
    try {
      const parts = dateStr.split('-');
      if (parts.length === 3) {
        const d = new Date(parseInt(parts[0], 10), parseInt(parts[1], 10) - 1, parseInt(parts[2], 10));
        return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' });
      }
      return dateStr;
    } catch {
      return dateStr;
    }
  }

  formatTime(isoStr?: string): string {
    if (!isoStr) return '--:--';
    try {
      const d = new Date(isoStr);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    } catch {
      return isoStr;
    }
  }

  getUserInitials(): string {
    const user = this.authService.currentUser();
    if (!user || !user.name) return 'FD';
    const parts = user.name.trim().split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return user.name.slice(0, 2).toUpperCase();
  }

  logout(): void {
    this.authService.logout();
  }
}
