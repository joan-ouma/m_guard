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
import { ActivatedRoute, RouterLink } from '@angular/router';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { AuthService } from '../core/services/auth.service';
import { BikeService } from '../core/services/bike.service';
import { BikeDTO, BreadcrumbDTO } from '../core/models/bike.models';
import * as L from 'leaflet';

export interface FleetBike {
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
  triggerCondition: string;
  email: string;
}

const DEFAULT_BIKE: FleetBike = {
  id: '',
  plate: 'NO BIKE',
  model: 'No asset registered',
  imei: '',
  status: 'STOPPED',
  speedText: '0.0 km/h • Ignition OFF',
  address: 'Nairobi CBD',
  county: 'Nairobi County',
  subCounty: 'Starehe',
  lat: -1.286389,
  lng: 36.817223,
  fenceName: 'Central Logistics Depot',
  fenceRadius: 2500,
  triggerCondition: 'Alert on Exit',
  email: 'operator@mototrack.ke',
};

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, RouterLink, ReactiveFormsModule],
  templateUrl: './dashboard.component.html',
})
export class DashboardComponent implements OnInit, AfterViewInit, OnDestroy {
  readonly authService = inject(AuthService);
  readonly bikeService = inject(BikeService);
  private readonly fb = inject(FormBuilder);
  private readonly route = inject(ActivatedRoute);

  private map: L.Map | null = null;
  private bikeMarker: L.Marker | null = null;
  private geofenceLayer: L.Circle | null = null;
  private pathPolyline: L.Polyline | null = null;
  private pathGlow: L.Polyline | null = null;
  private pathOriginMarker: L.CircleMarker | null = null;

  readonly fleet = signal<Record<string, FleetBike>>({});
  readonly activeKey = signal<string>('');
  readonly isBreachActive = signal<boolean>(false);
  readonly alertBannerText = signal<string>('');
  readonly showGeofenceModal = signal<boolean>(false);
  readonly copiedCoordinates = signal<boolean>(false);

  readonly bikeList = computed<FleetBike[]>(() => Object.values(this.fleet()));

  readonly selectedBike = computed<FleetBike>(() => {
    const f = this.fleet();
    const key = this.activeKey();
    if (key && f[key]) {
      return f[key];
    }
    const keys = Object.keys(f);
    if (keys.length > 0) {
      return f[keys[keys.length - 1]];
    }
    return DEFAULT_BIKE;
  });

  readonly geofenceForm: FormGroup = this.fb.group({
    fenceName: ['Ongata Rongai Delivery Zone', [Validators.required]],
    fenceRadius: [3500, [Validators.required, Validators.min(100), Validators.max(50000)]],
    fenceTrigger: ['Alert on Exit', [Validators.required]],
    fenceEmail: ['operator@mototrack.ke', [Validators.required, Validators.email]],
  });

  ngOnInit(): void {
    this.bikeService.loadBikes().subscribe({
      next: (res) => {
        const map: Record<string, FleetBike> = {};
        for (const b of res.data) {
          map[b.id] = this.mapToFleetBike(b);
        }
        this.fleet.set(map);

        const bikeList = Object.values(map);
        const lastBikeId = bikeList.length > 0 ? bikeList[bikeList.length - 1].id : '';
        const bikeParam = this.route.snapshot.queryParamMap.get('bike');
        const targetId =
          bikeParam && map[bikeParam]
            ? bikeParam
            : res.meta.default_selected_bike_id && map[res.meta.default_selected_bike_id]
            ? res.meta.default_selected_bike_id
            : lastBikeId;

        if (targetId) {
          this.activeKey.set(targetId);
        }

        const current = this.selectedBike();
        this.geofenceForm.patchValue({
          fenceName: current.fenceName,
          fenceRadius: current.fenceRadius,
          fenceTrigger: current.triggerCondition,
          fenceEmail: current.email,
        });

        this.updateMapForCurrentBike();
      },
    });

    this.route.queryParamMap.subscribe((params) => {
      const bikeParam = params.get('bike');
      if (bikeParam && this.fleet()[bikeParam]) {
        this.selectBike(bikeParam);
      }
    });
  }

  ngAfterViewInit(): void {
    if (typeof window === 'undefined') return;

    setTimeout(() => {
      this.initMap();
      this.map?.invalidateSize();
    }, 60);
  }

  ngOnDestroy(): void {
    this.clearPathLayers();
    if (this.map) {
      this.map.remove();
      this.map = null;
    }
  }

  private initMap(): void {
    const bike = this.selectedBike();
    const mapEl = document.getElementById('map');
    if (!mapEl) return;

    this.map = L.map('map', {
      zoomControl: false,
    }).setView([bike.lat, bike.lng], 13);

    // Custom Dark Basemap: Esri World Dark Gray Canvas
    L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Base/MapServer/tile/{z}/{y}/{x}', {
      maxZoom: 16,
      attribution: '&copy; <a href="https://www.esri.com/">Esri</a> &mdash; DeLorme, NAVTEQ',
    }).addTo(this.map);

    // Muted Reference Labels Layer (subtle, low opacity, no clutter)
    L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Reference/MapServer/tile/{z}/{y}/{x}', {
      maxZoom: 16,
      opacity: 0.45,
      attribution: '',
    }).addTo(this.map);

    const shortCode = bike.plate.split(' ')[1] || bike.plate;
    this.bikeMarker = L.marker([bike.lat, bike.lng], {
      icon: this.createBadgeIcon(shortCode, bike.status, false),
    }).addTo(this.map);

    this.geofenceLayer = L.circle([bike.lat, bike.lng], {
      color: '#0EA5E9',
      fillColor: '#0EA5E9',
      fillOpacity: 0.08,
      weight: 2,
      dashArray: '6, 6',
      radius: bike.fenceRadius,
    }).addTo(this.map);

    this.map.on('zoom', () => {
      this.updateMarkerSize();
    });

    this.loadAndDrawPath(bike);
    this.map.invalidateSize();
  }

  onSelectBikeChange(event: Event): void {
    const target = event.target as HTMLSelectElement;
    if (target && target.value) {
      this.selectBike(target.value);
    }
  }

  private updateMapForCurrentBike(): void {
    if (!this.map) return;
    const bike = this.selectedBike();
    const shortCode = bike.plate.split(' ')[1] || bike.plate;

    if (!this.bikeMarker) {
      this.bikeMarker = L.marker([bike.lat, bike.lng], {
        icon: this.createBadgeIcon(shortCode, bike.status, false),
      }).addTo(this.map);
    } else {
      this.bikeMarker.setLatLng([bike.lat, bike.lng]);
      this.bikeMarker.setIcon(this.createBadgeIcon(shortCode, bike.status, false));
    }

    if (!this.geofenceLayer) {
      this.geofenceLayer = L.circle([bike.lat, bike.lng], {
        color: '#0EA5E9',
        fillColor: '#0EA5E9',
        fillOpacity: 0.08,
        weight: 2,
        dashArray: '6, 6',
        radius: bike.fenceRadius,
      }).addTo(this.map);
    } else {
      this.geofenceLayer
        .setLatLng([bike.lat, bike.lng])
        .setRadius(bike.fenceRadius)
        .setStyle({
          color: '#0EA5E9',
          fillColor: '#0EA5E9',
          fillOpacity: 0.08,
          weight: 2,
          dashArray: '6, 6',
        });
    }

    this.map.setView([bike.lat, bike.lng], 13);
    this.map.invalidateSize();
    this.loadAndDrawPath(bike);
  }

  private loadAndDrawPath(bike: FleetBike): void {
    if (!bike.id || !this.map) return;

    this.bikeService.getBikeBreadcrumbs(bike.id).subscribe({
      next: (res) => {
        this.drawPathOnMap(res.data, bike);
      },
      error: () => {
        this.drawPathOnMap([], bike);
      },
    });
  }

  private drawPathOnMap(points: BreadcrumbDTO[], currentBike: FleetBike): void {
    if (!this.map) return;

    this.clearPathLayers();

    const coords: [number, number][] = points.map((p) => [p.latitude, p.longitude] as [number, number]);

    const isLastSame =
      coords.length > 0 &&
      Math.abs(coords[coords.length - 1][0] - currentBike.lat) < 0.0001 &&
      Math.abs(coords[coords.length - 1][1] - currentBike.lng) < 0.0001;

    if (!isLastSame) {
      coords.push([currentBike.lat, currentBike.lng]);
    }

    if (coords.length > 1) {
      this.pathGlow = L.polyline(coords, {
        color: '#38BDF8',
        weight: 6,
        opacity: 0.35,
        lineCap: 'round',
        lineJoin: 'round',
      }).addTo(this.map);

      this.pathPolyline = L.polyline(coords, {
        color: '#0EA5E9',
        weight: 3,
        opacity: 0.95,
        dashArray: '6, 6',
        lineCap: 'round',
        lineJoin: 'round',
      }).addTo(this.map);

      this.pathOriginMarker = L.circleMarker(coords[0], {
        radius: 5,
        color: '#38BDF8',
        fillColor: '#09090b',
        fillOpacity: 1,
        weight: 2,
      })
        .addTo(this.map)
        .bindTooltip('Trip Origin', { direction: 'top', offset: [0, -6] });
    }
  }

  private clearPathLayers(): void {
    if (!this.map) return;
    if (this.pathPolyline) {
      this.map.removeLayer(this.pathPolyline);
      this.pathPolyline = null;
    }
    if (this.pathGlow) {
      this.map.removeLayer(this.pathGlow);
      this.pathGlow = null;
    }
    if (this.pathOriginMarker) {
      this.map.removeLayer(this.pathOriginMarker);
      this.pathOriginMarker = null;
    }
  }

  private updateMarkerSize(): void {
    if (!this.bikeMarker || !this.map) return;
    const bike = this.selectedBike();
    const shortCode = bike.plate.split(' ')[1] || bike.plate;
    this.bikeMarker.setIcon(this.createBadgeIcon(shortCode, bike.status, this.isBreachActive()));
  }

  private createBadgeIcon(label: string, status: string = 'RUNNING', isBreach: boolean = false): L.DivIcon {
    const isRunning = status === 'RUNNING';
    const statusClass = isBreach ? 'breached' : isRunning ? 'running' : 'stopped';
    const haloClass = isBreach ? 'radar-halo radar-halo-breach' : 'radar-halo';
    const dotColor = isBreach ? '#F43F5E' : isRunning ? '#10B981' : '#F59E0B';

    return L.divIcon({
      className: 'bike-marker-pin',
      html: `
        <div class="bike-marker-badge">
          <!-- Plate Label Pill -->
          <div style="background: rgba(9, 9, 11, 0.92); backdrop-filter: blur(4px); color: #f4f4f5; font-family: 'JetBrains Mono', monospace; font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 9999px; border: 1px solid rgba(255, 255, 255, 0.18); box-shadow: 0 4px 12px rgba(0,0,0,0.5); white-space: nowrap; margin-bottom: 6px; display: flex; align-items: center; gap: 5px;">
            <span style="display: inline-block; width: 6px; height: 6px; border-radius: 9999px; background: ${dotColor};"></span>
            <span>${label}</span>
          </div>

          <!-- Pulsing Live Halo & Pin Container -->
          <div style="position: relative; width: 36px; height: 36px; display: flex; align-items: center; justify-content: center;">
            <div class="${haloClass}"></div>
            <div class="bike-marker-pin-body ${statusClass}">
              <svg style="width: 20px; height: 20px; color: #ffffff;" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="5.5" cy="17.5" r="3.5"/>
                <circle cx="18.5" cy="17.5" r="3.5"/>
                <path d="M15 6h-3l-2.5 5h6.5"/>
                <path d="M12 17.5V14l-3-3 4-3 2 3h2"/>
              </svg>
            </div>
            <!-- Directional Heading Arrow -->
            <div class="bike-marker-heading-arrow ${statusClass}"></div>
          </div>
        </div>
      `,
      iconSize: [84, 76],
      iconAnchor: [42, 70],
    });
  }

  selectBike(key: string): void {
    this.activeKey.set(key);
    this.isBreachActive.set(false);
    this.alertBannerText.set('');

    const bike = this.selectedBike();

    this.geofenceForm.patchValue({
      fenceName: bike.fenceName,
      fenceRadius: bike.fenceRadius,
      fenceTrigger: bike.triggerCondition,
      fenceEmail: bike.email,
    });

    if (this.map && this.bikeMarker && this.geofenceLayer) {
      const shortCode = bike.plate.split(' ')[1] || bike.plate;
      this.bikeMarker.setIcon(this.createBadgeIcon(shortCode, bike.status, false));
      this.bikeMarker.setLatLng([bike.lat, bike.lng]);

      this.geofenceLayer
        .setLatLng([bike.lat, bike.lng])
        .setRadius(bike.fenceRadius)
        .setStyle({
          color: '#0EA5E9',
          fillColor: '#0EA5E9',
          fillOpacity: 0.08,
          weight: 2,
          dashArray: '6, 6',
        });

      this.map.setView([bike.lat, bike.lng], 13);
      this.map.invalidateSize();
      this.loadAndDrawPath(bike);
    }
  }

  recenterMap(): void {
    if (!this.map) return;
    const bike = this.selectedBike();
    this.map.setView([bike.lat, bike.lng], 14);
  }

  copyCoordinates(): void {
    const bike = this.selectedBike();
    const text = `${bike.lat.toFixed(6)}, ${bike.lng.toFixed(6)}`;
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      navigator.clipboard.writeText(text);
      this.copiedCoordinates.set(true);
      setTimeout(() => this.copiedCoordinates.set(false), 2000);
    }
  }

  zoomIn(): void {
    this.map?.zoomIn();
  }


  zoomOut(): void {
    this.map?.zoomOut();
  }

  toggleGeofenceModal(open: boolean): void {
    this.showGeofenceModal.set(open);
  }

  saveGeofence(e?: Event): void {
    if (e) {
      e.preventDefault();
    }
    if (this.geofenceForm.invalid) return;

    const val = this.geofenceForm.value;
    const key = this.activeKey();

    this.fleet.update((prev) => {
      const current = prev[key];
      if (!current) return prev;
      return {
        ...prev,
        [key]: {
          ...current,
          fenceName: val.fenceName,
          fenceRadius: parseInt(val.fenceRadius, 10),
          triggerCondition: val.fenceTrigger,
          email: val.fenceEmail,
        },
      };
    });

    if (this.geofenceLayer) {
      this.geofenceLayer.setRadius(parseInt(val.fenceRadius, 10));
    }

    this.toggleGeofenceModal(false);
  }

  simulateBreach(): void {
    const bike = this.selectedBike();
    const exitLat = bike.lat + 0.036;
    const exitLng = bike.lng + 0.024;

    this.fleet.update((prev) => {
      const key = this.activeKey();
      if (!prev[key]) return prev;
      return {
        ...prev,
        [key]: {
          ...prev[key],
          lat: exitLat,
          lng: exitLng,
        },
      };
    });

    if (this.bikeMarker) {
      this.bikeMarker.setLatLng([exitLat, exitLng]);
      const shortCode = bike.plate.split(' ')[1] || bike.plate;
      this.bikeMarker.setIcon(this.createBadgeIcon(shortCode, bike.status, true));
    }

    if (this.geofenceLayer) {
      this.geofenceLayer.setStyle({
        color: '#F43F5E',
        fillColor: '#F43F5E',
        fillOpacity: 0.16,
        weight: 2.5,
        dashArray: '4, 4',
      });
    }

    if (this.pathPolyline) {
      this.pathPolyline.addLatLng([exitLat, exitLng]);
    }
    if (this.pathGlow) {
      this.pathGlow.addLatLng([exitLat, exitLng]);
    }

    this.isBreachActive.set(true);
    this.alertBannerText.set(`Breach email sent to ${bike.email}`);

    this.map?.panTo([exitLat, exitLng]);
  }

  handleLogout(): void {
    this.authService.logout().subscribe();
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

  private mapToFleetBike(b: BikeDTO): FleetBike {
    const modelText = [b.make, b.model].filter(Boolean).join(' ') || b.label || 'Motorcycle';
    const speedText = `${b.speed_kmh.toFixed(1)} km/h • Ignition ${b.ignition ? 'ON' : 'OFF'}`;

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
      fenceName: b.geofence_id ? 'Assigned Geofence' : 'Nairobi Delivery Zone',
      fenceRadius: 3000,
      triggerCondition: 'Alert on Exit',
      email: 'operator@mototrack.ke',
    };
  }
}
