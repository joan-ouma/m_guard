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
import Swal from 'sweetalert2';

export interface FleetBike {
  id: string;
  plate: string;
  model: string;
  imei: string;
  status: 'RUNNING' | 'STOPPED';
  movementStatus: 'Moving' | 'Stationary';
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
  lastSeenAt?: string;
  isImmobilized: boolean;
  riderName: string;
  riderNationalId: string;
  riderPhone: string;
}

const DEFAULT_BIKE: FleetBike = {
  id: '',
  plate: 'NO BIKE',
  model: 'No asset registered',
  imei: '',
  status: 'STOPPED',
  movementStatus: 'Stationary',
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
  lastSeenAt: new Date().toISOString(),
  isImmobilized: false,
  riderName: 'Unassigned',
  riderNationalId: 'N/A',
  riderPhone: 'N/A',
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
  readonly showDuressPrompt = signal<boolean>(false);
  readonly duressCountdown = signal<number>(30);
  readonly copiedCoordinates = signal<boolean>(false);
  readonly currentTime = signal<number>(Date.now());
  readonly mobileDrawerOpen = signal<boolean>(false);
  readonly mobileSheetExpanded = signal<boolean>(false);
  private tickerInterval?: ReturnType<typeof setInterval>;
  private pollInterval?: ReturnType<typeof setInterval>;
  private duressInterval?: ReturnType<typeof setInterval>;

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

  readonly showRegisterModal = signal<boolean>(false);
  readonly registrationError = signal<string | null>(null);
  readonly isSubmitting = signal<boolean>(false);

  readonly registerForm: FormGroup = this.fb.group({
    plate: ['', [Validators.required, Validators.pattern(/^[A-Z]{4}\s?[0-9]{3}[A-Z]$/i)]],
    model: ['', [Validators.required]],
    imei: ['', [Validators.required, Validators.pattern(/^[0-9]{15}$/)]],
    county: ['Nairobi County', [Validators.required]],
    subCounty: ['Westlands', [Validators.required]],
    isSelfRider: [false],
    riderName: ['', [Validators.required, Validators.minLength(2)]],
    riderNationalId: ['', [Validators.required, Validators.pattern(/^[0-9]{6,10}$/)]],
    riderPhone: ['', [Validators.required, Validators.pattern(/^(\+?[0-9\s-]{9,15})$/)]],
  });

  ngOnInit(): void {
    this.tickerInterval = setInterval(() => {
      this.currentTime.set(Date.now());
    }, 1000);

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

    this.pollInterval = setInterval(() => {
      this.bikeService.loadBikes().subscribe({
        next: (res) => {
          this.fleet.update((prev) => {
            const nextMap: Record<string, FleetBike> = { ...prev };
            for (const b of res.data) {
              const currentBike = prev[b.id];
              const mapped = this.mapToFleetBike(b);
              if (currentBike && currentBike.fenceRadius) {
                mapped.fenceName = currentBike.fenceName;
                mapped.fenceRadius = currentBike.fenceRadius;
                mapped.triggerCondition = currentBike.triggerCondition;
                mapped.email = currentBike.email;
              }
              nextMap[b.id] = mapped;
            }
            return nextMap;
          });
          this.updateMapForCurrentBike();
        },
      });
    }, 3000);

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
    if (this.tickerInterval) {
      clearInterval(this.tickerInterval);
      this.tickerInterval = undefined;
    }
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
      this.pollInterval = undefined;
    }
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

    this.bikeMarker = L.marker([bike.lat, bike.lng], {
      icon: this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, false),
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

    if (!this.bikeMarker) {
      this.bikeMarker = L.marker([bike.lat, bike.lng], {
        icon: this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, false),
      }).addTo(this.map);
    } else {
      this.bikeMarker.setLatLng([bike.lat, bike.lng]);
      this.bikeMarker.setIcon(this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, false));
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
    this.bikeMarker.setIcon(this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, this.isBreachActive()));
  }

  private createBadgeIcon(plate: string, status: string = 'RUNNING', isImmobilized: boolean = false, isBreach: boolean = false): L.DivIcon {
    const isImmobilizedState = isImmobilized;
    const isRunning = status === 'RUNNING' && !isImmobilizedState;

    let statusClass = 'stopped';
    let haloClass = 'radar-halo radar-halo-stopped';
    let dotColor = '#f59e0b';
    let statusBg = 'rgba(245, 158, 11, 0.2)';

    if (isBreach) {
      statusClass = 'breached';
      haloClass = 'radar-halo radar-halo-breach';
      dotColor = '#f43f5e';
      statusBg = 'rgba(244, 63, 94, 0.25)';
    } else if (isImmobilizedState) {
      statusClass = 'immobilized';
      haloClass = 'radar-halo radar-halo-breach';
      dotColor = '#e11d48';
      statusBg = 'rgba(225, 29, 72, 0.25)';
    } else if (isRunning) {
      statusClass = 'running';
      haloClass = 'radar-halo radar-halo-running';
      dotColor = '#10b981';
      statusBg = 'rgba(16, 185, 129, 0.2)';
    }

    const pinIconSvg = isImmobilizedState
      ? `<svg style="width: 26px; height: 26px; color: #ffffff;" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
           <path d="M5.636 5.636a9 9 0 1012.728 0M12 3v9"/>
         </svg>`
      : `<svg style="width: 26px; height: 26px; color: #ffffff;" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
           <circle cx="5.5" cy="17.5" r="3.5"/>
           <circle cx="18.5" cy="17.5" r="3.5"/>
           <path d="M15 6h-3l-2.5 5h6.5"/>
           <path d="M12 17.5V14l-3-3 4-3 2 3h2"/>
         </svg>`;

    return L.divIcon({
      className: 'bike-marker-pin',
      html: `
        <div class="bike-marker-badge">
          <!-- Full Plate Label Pill with Status Heartbeat Dot -->
          <div style="background: rgba(9, 9, 11, 0.94); backdrop-filter: blur(4px); color: #ffffff; font-family: 'JetBrains Mono', monospace; font-size: 13px; font-weight: 800; padding: 4px 14px; border-radius: 9999px; border: 1.5px solid ${dotColor}; box-shadow: 0 4px 14px rgba(0,0,0,0.65), 0 0 10px ${statusBg}; white-space: nowrap; margin-bottom: 8px; display: flex; align-items: center; gap: 7px; letter-spacing: 0.5px;">
            <span style="position: relative; display: inline-flex; width: 8px; height: 8px; align-items: center; justify-content: center;">
              <span class="marker-dot-ping" style="position: absolute; inset: 0; border-radius: 9999px; background: ${dotColor}; opacity: 0.85;"></span>
              <span style="position: relative; display: inline-block; width: 8px; height: 8px; border-radius: 9999px; background: ${dotColor};"></span>
            </span>
            <span>${plate}</span>
          </div>

          <!-- Pulsing Live Halo & Pin Container (Thumb-Sized 48px) -->
          <div style="position: relative; width: 48px; height: 48px; display: flex; align-items: center; justify-content: center;">
            <div class="${haloClass}"></div>
            <div class="bike-marker-pin-body ${statusClass}">
              ${pinIconSvg}
            </div>
            <!-- Directional Heading Arrow -->
            <div class="bike-marker-heading-arrow ${statusClass}"></div>
          </div>
        </div>
      `,
      iconSize: [160, 94],
      iconAnchor: [80, 88],
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
      this.bikeMarker.setIcon(this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, false));
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

  toggleMobileDrawer(open?: boolean): void {
    const nextState = open !== undefined ? open : !this.mobileDrawerOpen();
    this.mobileDrawerOpen.set(nextState);
    setTimeout(() => this.map?.invalidateSize(), 300);
  }

  closeMobileDrawer(): void {
    this.mobileDrawerOpen.set(false);
  }

  toggleMobileSheet(): void {
    this.mobileSheetExpanded.update((v) => !v);
    setTimeout(() => this.map?.invalidateSize(), 300);
  }

  expandMobileSheet(): void {
    this.mobileSheetExpanded.set(true);
    setTimeout(() => this.map?.invalidateSize(), 300);
  }

  collapseMobileSheet(): void {
    this.mobileSheetExpanded.set(false);
    setTimeout(() => this.map?.invalidateSize(), 300);
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

  openRegisterModal(): void {
    this.registrationError.set(null);
    this.registerForm.reset({
      county: 'Nairobi County',
      subCounty: 'Westlands',
      isSelfRider: false,
      riderName: '',
      riderNationalId: '',
      riderPhone: '',
    });
    this.showRegisterModal.set(true);
  }

  onToggleSelfRider(event: Event): void {
    const checked = (event.target as HTMLInputElement).checked;
    this.registerForm.patchValue({ isSelfRider: checked });
    if (checked) {
      const user = this.authService.currentUser();
      if (user) {
        this.registerForm.patchValue({
          riderName: user.name || '',
          riderNationalId: user.national_id || '',
          riderPhone: user.phone_number || '',
        });
      }
    }
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
        rider_name: val.riderName ? val.riderName.trim() : undefined,
        rider_national_id: val.riderNationalId ? val.riderNationalId.trim() : undefined,
        rider_phone: val.riderPhone ? val.riderPhone.trim() : undefined,
        latitude: -1.286389,
        longitude: 36.817223,
        current_address: 'Nairobi CBD Staging Yard',
      })
      .subscribe({
        next: (res) => {
          this.isSubmitting.set(false);
          this.closeRegisterModal();
          
          // Add the newly created bike directly to the fleet and select it
          const mapped = this.mapToFleetBike(res.data);
          this.fleet.update((prev) => ({ ...prev, [mapped.id]: mapped }));
          this.selectBike(mapped.id);
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

  simulateBreach(): void {
    const bike = this.selectedBike();
    const exitLat = bike.lat + 0.036;
    const exitLng = bike.lng + 0.024;
    const nowIso = new Date().toISOString();

    this.fleet.update((prev) => {
      const key = this.activeKey();
      if (!prev[key]) return prev;
      return {
        ...prev,
        [key]: {
          ...prev[key],
          lat: exitLat,
          lng: exitLng,
          lastSeenAt: nowIso,
        },
      };
    });

    if (this.bikeMarker) {
      this.bikeMarker.setLatLng([exitLat, exitLng]);
      this.bikeMarker.setIcon(this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, true));
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

  simulateSuddenStop(): void {
    if (this.showDuressPrompt()) return;
    this.showDuressPrompt.set(true);
    this.duressCountdown.set(30);

    this.duressInterval = setInterval(() => {
      const current = this.duressCountdown() - 1;
      this.duressCountdown.set(current);
      if (current <= 0) {
        this.escalateDuress();
      }
    }, 1000);
  }

  cancelDuress(): void {
    this.showDuressPrompt.set(false);
    if (this.duressInterval) clearInterval(this.duressInterval);

    Swal.fire({
      toast: true,
      position: 'top-end',
      icon: 'success',
      title: 'Status cleared. Safe riding.',
      showConfirmButton: false,
      timer: 3000,
      timerProgressBar: true,
    });
  }

  escalateDuress(): void {
    this.showDuressPrompt.set(false);
    if (this.duressInterval) clearInterval(this.duressInterval);

    this.isBreachActive.set(true);
    this.alertBannerText.set(`DURESS ESCALATED: Alert sent to Sacco & Emergency Contacts`);

    Swal.fire({
      icon: 'error',
      title: 'Emergency Alert Sent',
      text: 'Your Sacco and emergency contacts have been notified of a potential incident.',
      confirmButtonColor: '#e11d48',
    });
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

  getLastSeen(bike: FleetBike): string {
    const now = this.currentTime();
    return this.formatLastSeen(bike.lastSeenAt, now);
  }

  formatLastSeen(dateStr?: string | null, now = Date.now()): string {
    if (!dateStr) return '3s ago';
    const timestamp = new Date(dateStr).getTime();
    if (isNaN(timestamp)) return '3s ago';

    const diffSec = Math.max(0, Math.floor((now - timestamp) / 1000));
    if (diffSec < 60) {
      return `${Math.max(1, diffSec)}s ago`;
    }
    const diffMin = Math.floor(diffSec / 60);
    if (diffMin < 60) {
      return `${diffMin}m ago`;
    }
    const diffHour = Math.floor(diffMin / 60);
    if (diffHour < 24) {
      return `${diffHour}h ago`;
    }
    const diffDay = Math.floor(diffHour / 24);
    return `${diffDay}d ago`;
  }

  toggleImmobilize(bike: FleetBike): void {
    if (!bike.id) return;
    const isImmobilizing = !bike.isImmobilized;
    const isMoving = bike.movementStatus === 'Moving';

    const actionText = isImmobilizing
      ? (isMoving 
          ? "The bike is currently moving. This will queue the command to disable the ignition once the bike stops. The bike won't start until you restore it." 
          : "This will disable the ignition. The bike won't start until you restore it.")
      : 'This will restore electrical starter power to the motorcycle.';

    Swal.fire({
      title: isImmobilizing ? `Immobilize ${bike.plate}?` : `Reactivate ${bike.plate}?`,
      html: `
        <div class="text-left mb-5">
          <p class="mb-4 text-zinc-600 text-sm">${actionText}</p>
          <div class="p-3 bg-zinc-50 border border-zinc-200 rounded-lg">
            <label class="block text-xs font-bold text-zinc-500 uppercase tracking-wider mb-1">Security Authorization</label>
            <p class="text-sm font-medium text-zinc-700">Enter your 4-digit operator PIN to confirm this action.</p>
          </div>
        </div>
      `,
      input: 'password',
      inputPlaceholder: '••••',
      inputAttributes: {
        maxlength: '4',
        autocapitalize: 'off',
        autocorrect: 'off',
        pattern: '[0-9]*',
        inputmode: 'numeric',
      },
      customClass: {
        input: 'text-center tracking-[0.5em] text-2xl font-bold rounded-xl border-zinc-300 focus:border-rose-500 focus:ring-rose-500',
      },
      icon: isImmobilizing ? 'warning' : 'question',
      showCancelButton: true,
      confirmButtonColor: isImmobilizing ? (isMoving ? '#f59e0b' : '#e11d48') : '#059669',
      cancelButtonColor: '#71717a',
      confirmButtonText: isImmobilizing ? (isMoving ? 'Queue Immobilization' : 'Yes, Immobilize') : 'Yes, Reactivate',
      cancelButtonText: 'Cancel',
      reverseButtons: true,
      preConfirm: (pin) => {
        if (!pin || pin.length !== 4 || !/^\d{4}$/.test(pin)) {
          Swal.showValidationMessage('Please enter a valid 4-digit PIN');
          return false;
        }
        return pin; // In a real app, this PIN would be sent to the backend API for validation
      },
    }).then((result) => {
      if (result.isConfirmed) {
        this.bikeService.toggleImmobilize(bike.id, isImmobilizing).subscribe({
          next: (res) => {
            this.fleet.update((prev) => {
              if (!prev[bike.id]) return prev;
              return {
                ...prev,
                [bike.id]: {
                  ...prev[bike.id],
                  isImmobilized: res.data.is_immobilized,
                },
              };
            });
            this.updateMarkerSize();
            Swal.fire({
              toast: true,
              position: 'top-end',
              icon: 'success',
              title: isImmobilizing ? (isMoving ? `${bike.plate} Immobilization Queued` : `${bike.plate} Immobilized`) : `${bike.plate} Reactivated`,
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

  private mapToFleetBike(b: BikeDTO): FleetBike {
    const modelText = [b.make, b.model].filter(Boolean).join(' ') || b.label || 'Motorcycle';
    const speedText = `${b.speed_kmh.toFixed(1)} km/h • Ignition ${b.ignition ? 'ON' : 'OFF'}`;
    const timestamp = b.last_telemetry_at || b.updated_at || b.created_at;
    const movementStatus = (b.status === 'RUNNING' || b.speed_kmh > 0) ? 'Moving' : 'Stationary';

    return {
      id: b.id,
      plate: b.registration_number,
      model: modelText,
      imei: b.imei,
      status: b.status,
      movementStatus,
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
      lastSeenAt: timestamp,
      isImmobilized: !!b.is_immobilized,
      riderName: b.rider_name || 'Primary Rider',
      riderNationalId: b.rider_national_id || 'N/A',
      riderPhone: b.rider_phone || 'N/A',
    };
  }
}
