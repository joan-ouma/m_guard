import {
  Component,
  OnInit,
  AfterViewInit,
  OnDestroy,
  inject,
  signal,
  computed,
  effect,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../core/services/auth.service';
import { BikeService } from '../core/services/bike.service';
import { NotificationService } from '../core/services/notification.service';
import { PwaService } from '../core/services/pwa.service';
import { BikeDTO, BreadcrumbDTO } from '../core/models/bike.models';
import * as L from 'leaflet';
import Swal from 'sweetalert2';

import {
  FleetBike,
  NotificationItem,
} from './models/dashboard.models';
import { DashboardSidebarComponent } from './components/dashboard-sidebar/dashboard-sidebar.component';
import { DashboardHeaderComponent } from './components/dashboard-header/dashboard-header.component';
import { BikeInspectorComponent } from './components/bike-inspector/bike-inspector.component';
import {
  GeofenceModalComponent,
  GeofenceData,
} from './components/geofence-modal/geofence-modal.component';
import { RegisterBikeModalComponent } from './components/register-bike-modal/register-bike-modal.component';
import { DuressModalComponent } from './components/duress-modal/duress-modal.component';

export type { FleetBike, NotificationItem };

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    DashboardSidebarComponent,
    DashboardHeaderComponent,
    BikeInspectorComponent,
    GeofenceModalComponent,
    RegisterBikeModalComponent,
    DuressModalComponent,
  ],
  templateUrl: './dashboard.component.html',
})
export class DashboardComponent implements OnInit, AfterViewInit, OnDestroy {
  readonly authService = inject(AuthService);
  readonly bikeService = inject(BikeService);
  readonly notificationService = inject(NotificationService);
  readonly pwa = inject(PwaService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

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
  readonly notifications = signal<NotificationItem[]>([]);
  readonly unreadAlertCount = computed(() => this.notifications().filter((n) => !n.read).length);

  readonly showRegisterModal = signal<boolean>(false);
  readonly registrationError = signal<string | null>(null);
  readonly isSubmitting = signal<boolean>(false);

  private tickerInterval?: ReturnType<typeof setInterval>;
  private duressInterval?: ReturnType<typeof setInterval>;

  constructor() {
    effect(() => {
      const alert = this.bikeService.latestAlert();
      if (!alert) return;

      const notif = this.createNotificationFromAlert(alert);
      this.notifications.update((prev) => {
        if (prev.some((p) => p.id === notif.id)) return prev;
        return [notif, ...prev];
      });
      this.notificationService.notify(
        notif.title,
        notif.activity,
        notif.bikeId ? `/history?bike=${notif.bikeId}` : '/history'
      );
    });

    effect(() => {
      const bikes = this.bikeService.bikes();
      if (!bikes || bikes.length === 0) return;

      this.fleet.update((prev) => {
        const nextMap: Record<string, FleetBike> = { ...prev };
        for (const b of bikes) {
          const currentBike = prev[b.id];
          const mapped = this.mapToFleetBike(b);
          if (currentBike && currentBike.hasGeofence) {
            mapped.hasGeofence = currentBike.hasGeofence;
            mapped.fenceCenterLat = currentBike.fenceCenterLat;
            mapped.fenceCenterLng = currentBike.fenceCenterLng;
            mapped.fenceName = currentBike.fenceName;
            mapped.fenceRadius = currentBike.fenceRadius;
            mapped.triggerCondition = currentBike.triggerCondition;
            mapped.email = currentBike.email;
          }
          nextMap[b.id] = mapped;
        }
        return nextMap;
      });

      const currentActive = this.activeKey();
      if (!currentActive || !this.fleet()[currentActive]) {
        const bikeParam = this.route.snapshot.queryParamMap.get('bike');
        const defaultId = this.bikeService.defaultSelectedBikeId();
        const targetId =
          bikeParam && this.fleet()[bikeParam]
            ? bikeParam
            : defaultId && this.fleet()[defaultId]
            ? defaultId
            : bikes[0]?.id;
        if (targetId) {
          this.activeKey.set(targetId);
        }
      }

      this.updateMapForCurrentBike(false);
    });
  }

  readonly bikeList = computed<FleetBike[]>(() => Object.values(this.fleet()));

  readonly selectedBike = computed<FleetBike | null>(() => {
    const f = this.fleet();
    const key = this.activeKey();
    if (key && f[key]) {
      return f[key];
    }
    const keys = Object.keys(f);
    if (keys.length > 0) {
      return f[keys[keys.length - 1]];
    }
    return null;
  });

  readonly currentGeofenceData = computed<GeofenceData | null>(() => {
    const bike = this.selectedBike();
    if (!bike) return null;
    return {
      fenceName: bike.fenceName,
      fenceRadius: bike.fenceRadius,
      fenceTrigger: bike.triggerCondition,
      fenceEmail: bike.email,
    };
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

        this.updateMapForCurrentBike(true);
      },
    });

    this.route.queryParamMap.subscribe((params) => {
      const bikeParam = params.get('bike');
      if (bikeParam && this.fleet()[bikeParam]) {
        this.selectBike(bikeParam);
      }
    });

    if (typeof window !== 'undefined') {
      try {
        localStorage.removeItem('motorguard_onboarding_dismissed');
      } catch {}
    }
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
    if (this.duressInterval) {
      clearInterval(this.duressInterval);
      this.duressInterval = undefined;
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

    const centerLat = bike ? bike.lat : -1.286389;
    const centerLng = bike ? bike.lng : 36.817223;
    const zoomLevel = bike ? 13 : 11;

    this.map = L.map('map', {
      zoomControl: false,
    }).setView([centerLat, centerLng], zoomLevel);

    // Custom Dark Basemap: Esri World Dark Gray Canvas
    L.tileLayer(
      'https://server.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Base/MapServer/tile/{z}/{y}/{x}',
      {
        maxZoom: 16,
        attribution: '&copy; <a href="https://www.esri.com/">Esri</a> &mdash; DeLorme, NAVTEQ',
      }
    ).addTo(this.map);

    // Muted Reference Labels Layer
    L.tileLayer(
      'https://server.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Reference/MapServer/tile/{z}/{y}/{x}',
      {
        maxZoom: 16,
        opacity: 0.45,
        attribution: '',
      }
    ).addTo(this.map);

    if (bike) {
      this.bikeMarker = L.marker([bike.lat, bike.lng], {
        icon: this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, false),
      }).addTo(this.map);

      if (bike.hasGeofence && bike.fenceRadius > 0 && bike.fenceCenterLat !== undefined && bike.fenceCenterLng !== undefined) {
        this.geofenceLayer = L.circle([bike.fenceCenterLat, bike.fenceCenterLng], {
          color: '#0EA5E9',
          fillColor: '#0EA5E9',
          fillOpacity: 0.08,
          weight: 2,
          dashArray: '6, 6',
          radius: bike.fenceRadius,
        }).addTo(this.map);
      } else {
        this.geofenceLayer = null;
      }

      this.loadAndDrawPath(bike);
    } else {
      this.bikeMarker = null;
      this.geofenceLayer = null;
    }

    this.map.invalidateSize();
  }

  private updateMapForCurrentBike(recenter: boolean = false): void {
    if (!this.map) return;
    const bike = this.selectedBike();

    if (!bike) {
      if (this.bikeMarker) {
        this.bikeMarker.remove();
        this.bikeMarker = null;
      }
      if (this.geofenceLayer) {
        this.geofenceLayer.remove();
        this.geofenceLayer = null;
      }
      this.clearPathLayers();
      return;
    }

    if (!this.bikeMarker) {
      this.bikeMarker = L.marker([bike.lat, bike.lng], {
        icon: this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, false),
      }).addTo(this.map);
    } else {
      this.bikeMarker.setLatLng([bike.lat, bike.lng]);
      this.bikeMarker.setIcon(this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, false));
    }

    if (bike.hasGeofence && bike.fenceRadius > 0 && bike.fenceCenterLat !== undefined && bike.fenceCenterLng !== undefined) {
      if (!this.geofenceLayer) {
        this.geofenceLayer = L.circle([bike.fenceCenterLat, bike.fenceCenterLng], {
          color: '#0EA5E9',
          fillColor: '#0EA5E9',
          fillOpacity: 0.08,
          weight: 2,
          dashArray: '6, 6',
          radius: bike.fenceRadius,
        }).addTo(this.map);
      } else {
        // FIXED CENTER: Never move geofence when bike moves!
        this.geofenceLayer
          .setLatLng([bike.fenceCenterLat, bike.fenceCenterLng])
          .setRadius(bike.fenceRadius)
          .setStyle({
            color: '#0EA5E9',
            fillColor: '#0EA5E9',
            fillOpacity: 0.08,
            weight: 2,
            dashArray: '6, 6',
          });
      }
    } else {
      // User didn't set geofence -> do NOT show it!
      if (this.geofenceLayer) {
        this.geofenceLayer.remove();
        this.geofenceLayer = null;
      }
    }

    if (recenter) {
      const currentZoom = this.map.getZoom() || 13;
      this.map.setView([bike.lat, bike.lng], currentZoom);
      this.map.invalidateSize();
    }
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

    const coords: [number, number][] = points.map(
      (p) => [p.latitude, p.longitude] as [number, number]
    );

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
    if (!bike) return;
    this.bikeMarker.setIcon(
      this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, this.isBreachActive())
    );
  }

  private createBadgeIcon(
    plate: string,
    status: string = 'RUNNING',
    isImmobilized: boolean = false,
    isBreach: boolean = false
  ): L.DivIcon {
    const isImmobilizedState = isImmobilized;
    const isRunning = status === 'RUNNING' && !isImmobilizedState;

    let statusClass = 'stopped';
    let haloClass = 'radar-halo radar-halo-stopped';

    if (isBreach) {
      statusClass = 'breached';
      haloClass = 'radar-halo radar-halo-breach';
    } else if (isImmobilizedState) {
      statusClass = 'immobilized';
      haloClass = 'radar-halo radar-halo-breach';
    } else if (isRunning) {
      statusClass = 'running';
      haloClass = 'radar-halo radar-halo-running';
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
          <div style="background: rgba(9, 9, 11, 0.94); backdrop-filter: blur(4px); color: #ffffff; font-family: 'Barlow Semi Condensed', sans-serif; font-size: 13.5px; font-weight: 900; letter-spacing: 0.06em; text-transform: uppercase; padding: 4px 14px; border-radius: 9999px; border: 1px solid rgba(255, 255, 255, 0.18); box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.5); display: flex; align-items: center; justify-content: center; white-space: nowrap; margin-bottom: 8px;">
            <span>${plate}</span>
          </div>
          <div style="position: relative; width: 48px; height: 48px; display: flex; align-items: center; justify-content: center;">
            <div class="${haloClass}"></div>
            <div class="bike-marker-pin-body ${statusClass}">
              ${pinIconSvg}
            </div>
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
    this.updateMapForCurrentBike(true);
  }

  recenterMap(): void {
    if (!this.map) return;
    const bike = this.selectedBike();
    if (!bike) return;
    this.map.panTo([bike.lat, bike.lng]);
  }

  copyCoordinates(): void {
    const bike = this.selectedBike();
    if (!bike) return;
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

  saveGeofence(data: GeofenceData): void {
    const key = this.activeKey();
    const bike = this.selectedBike();
    if (!bike) {
      this.toggleGeofenceModal(false);
      return;
    }
    const centerLat = bike.fenceCenterLat ?? bike.lat;
    const centerLng = bike.fenceCenterLng ?? bike.lng;

    this.fleet.update((prev) => {
      const current = prev[key];
      if (!current) return prev;
      return {
        ...prev,
        [key]: {
          ...current,
          hasGeofence: true,
          fenceCenterLat: centerLat,
          fenceCenterLng: centerLng,
          fenceName: data.fenceName,
          fenceRadius: data.fenceRadius,
          triggerCondition: data.fenceTrigger,
          email: data.fenceEmail,
        },
      };
    });

    if (this.map) {
      if (!this.geofenceLayer) {
        this.geofenceLayer = L.circle([centerLat, centerLng], {
          color: '#0EA5E9',
          fillColor: '#0EA5E9',
          fillOpacity: 0.08,
          weight: 2,
          dashArray: '6, 6',
          radius: data.fenceRadius,
        }).addTo(this.map);
      } else {
        this.geofenceLayer
          .setLatLng([centerLat, centerLng])
          .setRadius(data.fenceRadius)
          .setStyle({
            color: '#0EA5E9',
            fillColor: '#0EA5E9',
            fillOpacity: 0.08,
            weight: 2,
            dashArray: '6, 6',
          });
      }
    }

    this.toggleGeofenceModal(false);
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

  openRegisterModal(): void {
    this.showRegisterModal.set(true);
  }

  closeRegisterModal(): void {
    this.showRegisterModal.set(false);
    this.registrationError.set(null);
  }

  registerBike(val: any): void {
    if (this.isSubmitting()) return;

    this.isSubmitting.set(true);
    this.registrationError.set(null);

    const parts = (val.model || '').trim().split(' ');
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
        next: (res) => {
          this.isSubmitting.set(false);
          this.closeRegisterModal();
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
    if (!bike) return;
    const centerLat = bike.fenceCenterLat ?? bike.lat;
    const centerLng = bike.fenceCenterLng ?? bike.lng;
    const radius = bike.fenceRadius > 0 ? bike.fenceRadius : 2500;
    const exitLat = centerLat + 0.036;
    const exitLng = centerLng + 0.024;
    const nowIso = new Date().toISOString();

    this.fleet.update((prev) => {
      const key = this.activeKey();
      if (!prev[key]) return prev;
      return {
        ...prev,
        [key]: {
          ...prev[key],
          hasGeofence: true,
          fenceCenterLat: centerLat,
          fenceCenterLng: centerLng,
          fenceName: bike.fenceName || 'Safe Operational Zone',
          fenceRadius: radius,
          lat: exitLat,
          lng: exitLng,
          lastSeenAt: nowIso,
        },
      };
    });

    if (this.bikeMarker) {
      this.bikeMarker.setLatLng([exitLat, exitLng]);
      this.bikeMarker.setIcon(
        this.createBadgeIcon(bike.plate, bike.status, bike.isImmobilized, true)
      );
    }

    if (this.map) {
      if (!this.geofenceLayer) {
        this.geofenceLayer = L.circle([centerLat, centerLng], {
          color: '#F43F5E',
          fillColor: '#F43F5E',
          fillOpacity: 0.16,
          weight: 2.5,
          dashArray: '4, 4',
          radius: radius,
        }).addTo(this.map);
      } else {
        this.geofenceLayer
          .setLatLng([centerLat, centerLng])
          .setRadius(radius)
          .setStyle({
            color: '#F43F5E',
            fillColor: '#F43F5E',
            fillOpacity: 0.16,
            weight: 2.5,
            dashArray: '4, 4',
          });
      }
    }

    if (this.pathPolyline) {
      this.pathPolyline.addLatLng([exitLat, exitLng]);
    }
    if (this.pathGlow) {
      this.pathGlow.addLatLng([exitLat, exitLng]);
    }

    this.isBreachActive.set(true);
    this.alertBannerText.set(`Breach email sent to ${bike.email}`);

    this.addNotification({
      id: `breach-${Date.now()}`,
      title: `${bike.plate} • Geofence Breach`,
      activity: `Vehicle exited ${bike.fenceName} perimeter`,
      bikeId: bike.id,
      time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      read: false,
    });

    this.notificationService.notify(
      `${bike.plate} • Geofence Breach`,
      `Vehicle exited ${bike.fenceName} perimeter`,
      bike.id ? `/history?bike=${bike.id}` : '/history'
    );

    this.map?.panTo([exitLat, exitLng]);
  }

  simulateSuddenStop(): void {
    if (this.showDuressPrompt()) return;
    const bike = this.selectedBike();
    if (!bike) return;
    this.showDuressPrompt.set(true);
    this.duressCountdown.set(30);
    this.addNotification({
      id: `stop-${Date.now()}`,
      title: `${bike.plate} • Sudden Stop`,
      activity: 'Rapid deceleration anomaly detected',
      bikeId: bike.id,
      time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      read: false,
    });

    this.notificationService.notify(
      `${bike.plate} • Sudden Stop`,
      'Rapid deceleration anomaly detected',
      bike.id ? `/history?bike=${bike.id}` : '/history'
    );

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

    const bike = this.selectedBike();
    if (!bike) return;

    this.addNotification({
      id: `duress-${Date.now()}`,
      title: `${bike.plate} • Duress Escalated`,
      activity: 'Emergency alert dispatched to contacts',
      bikeId: bike.id,
      time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      read: false,
    });

    this.notificationService.notify(
      `${bike.plate} • Duress Escalated`,
      'Emergency alert dispatched to Sacco and emergency contacts',
      bike.id ? `/history?bike=${bike.id}` : '/history'
    );

    Swal.fire({
      icon: 'error',
      title: 'Emergency Alert Sent',
      text: 'Your Sacco and emergency contacts have been notified of a potential incident.',
      confirmButtonColor: '#e11d48',
    });
  }

  markAllNotificationsRead(): void {
    this.notifications.update((prev) => prev.map((n) => ({ ...n, read: true })));
  }

  addNotification(item: NotificationItem): void {
    this.notifications.update((prev) => [item, ...prev]);
  }

  viewNotification(item: NotificationItem): void {
    this.notifications.update((prev) =>
      prev.map((n) => (n.id === item.id ? { ...n, read: true } : n))
    );
    if (item.bikeId) {
      if (this.fleet()[item.bikeId]) {
        this.selectBike(item.bikeId);
      }
      this.router.navigate(['/history'], { queryParams: { bike: item.bikeId } });
    } else {
      this.router.navigate(['/history']);
    }
  }

  private createNotificationFromAlert(alert: any): NotificationItem {
    const bikeId = alert.bike_id || alert.bikeId || '';
    const bike = this.fleet()[bikeId];
    const bikeName = bike ? bike.plate : alert.plate || 'Motorcycle';
    const rawType = (alert.event_type || alert.eventType || alert.title || 'Alert')
      .toUpperCase()
      .replace(/_/g, ' ');

    let title = bikeName ? `${bikeName} • ${rawType}` : rawType;
    let activity = alert.activity || alert.message || alert.details;

    if (!activity) {
      if (rawType.includes('GEOFENCE') || rawType.includes('BREACH')) {
        activity = 'Boundary breach detected outside perimeter';
      } else if (rawType.includes('STOP') || rawType.includes('DURESS')) {
        activity = 'Sudden stop anomaly recorded';
      } else if (rawType.includes('CRASH')) {
        activity = 'Collision impact detected';
      } else if (rawType.includes('THEFT') || rawType.includes('TAMPER')) {
        activity = 'Unauthorized movement or tamper alert';
      } else {
        activity = 'Security event detected';
      }
    }

    const timeStr = alert.recorded_at
      ? new Date(alert.recorded_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      : new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

    const id = alert.id || `${bikeId}-${rawType}-${alert.recorded_at || Date.now()}`;

    return {
      id,
      title,
      activity,
      bikeId,
      time: timeStr,
      read: false,
    };
  }

  handleLogout(): void {
    this.authService.logout().subscribe();
  }

  toggleImmobilize(bike: FleetBike): void {
    if (!bike.id) return;
    const isImmobilizing = !bike.isImmobilized;
    const isMoving = bike.movementStatus === 'Moving';

    const actionText = isImmobilizing
      ? isMoving
        ? "The bike is currently moving. This will queue the command to disable the ignition once the bike stops. The bike won't start until you restore it."
        : "This will disable the ignition. The bike won't start until you restore it."
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
        input:
          'text-center tracking-[0.5em] text-2xl font-bold rounded-xl border-zinc-300 focus:border-rose-500 focus:ring-rose-500',
      },
      icon: isImmobilizing ? 'warning' : 'question',
      showCancelButton: true,
      confirmButtonColor: isImmobilizing
        ? isMoving
          ? '#f59e0b'
          : '#e11d48'
        : '#059669',
      cancelButtonColor: '#71717a',
      confirmButtonText: isImmobilizing
        ? isMoving
          ? 'Queue Immobilization'
          : 'Yes, Immobilize'
        : 'Yes, Reactivate',
      cancelButtonText: 'Cancel',
      reverseButtons: true,
      preConfirm: (pin) => {
        if (!pin || pin.length !== 4 || !/^\d{4}$/.test(pin)) {
          Swal.showValidationMessage('Please enter a valid 4-digit PIN');
          return false;
        }
        return pin;
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
              title: isImmobilizing
                ? isMoving
                  ? `${bike.plate} Immobilization Queued`
                  : `${bike.plate} Immobilized`
                : `${bike.plate} Reactivated`,
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
    const movementStatus = b.status === 'RUNNING' || b.speed_kmh > 0 ? 'Moving' : 'Stationary';
    const hasGeofence = !!b.geofence_id;

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
      hasGeofence: hasGeofence,
      fenceCenterLat: hasGeofence ? b.latitude || -1.286389 : undefined,
      fenceCenterLng: hasGeofence ? b.longitude || 36.817223 : undefined,
      fenceName: hasGeofence ? 'Assigned Geofence' : '',
      fenceRadius: hasGeofence ? 2500 : 0,
      triggerCondition: 'Alert on Exit',
      email: this.authService.currentUser()?.email || '',
      lastSeenAt: timestamp,
      isImmobilized: !!b.is_immobilized,
      riderName: this.authService.currentUser()?.name || 'Primary Rider',
      riderNationalId: this.authService.currentUser()?.national_id || 'N/A',
      riderPhone: this.authService.currentUser()?.phone_number || 'N/A',
      signalQuality: b.signal_quality_csq,
      batteryPercent: b.battery_percent ?? 100,
    };
  }
}
