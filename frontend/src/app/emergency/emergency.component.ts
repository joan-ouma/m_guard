import { Component, OnInit, OnDestroy, computed, inject, signal, ViewEncapsulation } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { MobileNavComponent } from '../ui/mobile-nav/mobile-nav.component';
import { BikeService } from '../core/services/bike.service';
import { BikeDTO, BikeListResponse } from '../core/models/bike.models';
import { EMERGENCY_CONTACTS, EmergencyContact } from './emergency.data';
import * as L from 'leaflet';

@Component({
  selector: 'app-emergency',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, RouterLinkActive, MobileNavComponent],
  templateUrl: './emergency.component.html',
  styleUrl: './emergency.component.css',
  encapsulation: ViewEncapsulation.None,
})
export class EmergencyComponent implements OnInit, OnDestroy {
  private readonly bikeService = inject(BikeService);

  readonly contacts = signal<EmergencyContact[]>(EMERGENCY_CONTACTS);
  readonly searchQuery = signal<string>('');
  readonly selectedType = signal<'ALL' | 'police' | 'ambulance' | 'national'>('ALL');
  readonly selectedCounty = signal<string>('ALL');
  readonly activeView = signal<'split' | 'list' | 'map'>('list');
  readonly selectedContact = signal<EmergencyContact | null>(null);
  readonly copiedReport = signal<boolean>(false);

  // User bikes for location proximity and incident report
  readonly bikes = signal<BikeDTO[]>([]);
  readonly selectedBike = signal<BikeDTO | null>(null);

  // Available unique counties from directory
  readonly counties = computed<string[]>(() => {
    const set = new Set<string>();
    EMERGENCY_CONTACTS.forEach((c) => {
      if (c.county && c.county !== 'National') set.add(c.county);
    });
    return Array.from(set).sort();
  });

  // Filtered contacts list
  readonly filteredContacts = computed<EmergencyContact[]>(() => {
    const q = this.searchQuery().toLowerCase().trim();
    const type = this.selectedType();
    const county = this.selectedCounty();
    const bike = this.selectedBike();

    let list = this.contacts().filter((c) => {
      // Type filter
      if (type !== 'ALL' && c.type !== type) return false;
      // County filter
      if (county !== 'ALL' && c.county !== county && c.county !== 'National') return false;

      // Text search
      if (!q) return true;
      const haystack = `${c.name} ${c.county} ${c.subCounty} ${c.address} ${c.phone} ${c.altPhone || ''}`.toLowerCase();
      return haystack.includes(q);
    });

    // If bike location is known, sort by distance
    if (bike && bike.latitude && bike.longitude) {
      list = [...list].sort((a, b) => {
        // Keep national hotlines at top
        if (a.type === 'national' && b.type !== 'national') return -1;
        if (b.type === 'national' && a.type !== 'national') return 1;

        const distA = this.calculateDistance(bike.latitude, bike.longitude, a.latitude, a.longitude);
        const distB = this.calculateDistance(bike.latitude, bike.longitude, b.latitude, b.longitude);
        return distA - distB;
      });
    }

    return list;
  });

  // Leaflet map references
  private map: L.Map | null = null;
  private markersLayer: L.LayerGroup | null = null;
  private bikeMarker: L.Marker | null = null;
  private contactMarkers = new Map<string, L.Marker>();

  ngOnInit(): void {
    // Load user bikes for nearest station calculation and report generator
    this.bikeService.loadBikes().subscribe({
      next: (res: BikeListResponse) => {
        const list = res.data || [];
        this.bikes.set(list);
        if (list.length > 0) {
          this.selectedBike.set(list[0]);
        }
      },
    });

    // Check screen width to default to split on desktop
    if (typeof window !== 'undefined' && window.innerWidth >= 1024) {
      this.activeView.set('split');
    }
  }

  ngOnDestroy(): void {
    if (this.map) {
      this.map.remove();
      this.map = null;
    }
  }

  setView(mode: 'split' | 'list' | 'map'): void {
    this.activeView.set(mode);
    if (mode === 'map' || mode === 'split') {
      setTimeout(() => this.initMapIfNeeded(), 100);
    }
  }

  setTypeFilter(type: 'ALL' | 'police' | 'ambulance' | 'national'): void {
    this.selectedType.set(type);
    this.updateMapMarkers();
  }

  setCountyFilter(county: string): void {
    this.selectedCounty.set(county);
    this.updateMapMarkers();
  }

  selectContact(c: EmergencyContact): void {
    this.selectedContact.set(c);

    // If on mobile list view, switch to map to show location
    if (this.activeView() === 'list' && typeof window !== 'undefined' && window.innerWidth < 1024) {
      this.activeView.set('map');
      setTimeout(() => {
        this.initMapIfNeeded();
        this.flyToContact(c);
      }, 150);
    } else {
      this.initMapIfNeeded();
      this.flyToContact(c);
    }
  }

  flyToContact(c: EmergencyContact): void {
    if (!this.map) return;
    this.map.flyTo([c.latitude, c.longitude], 15, { duration: 0.8 });
    const marker = this.contactMarkers.get(c.id);
    if (marker) {
      setTimeout(() => marker.openPopup(), 800);
    }
  }

  getDistanceToBike(c: EmergencyContact): string | null {
    const bike = this.selectedBike();
    if (!bike || !bike.latitude || !bike.longitude) return null;
    if (c.type === 'national' && c.county === 'National') return null;

    const km = this.calculateDistance(bike.latitude, bike.longitude, c.latitude, c.longitude);
    if (km < 1) {
      return `${Math.round(km * 1000)}m away`;
    }
    return `${km.toFixed(1)} km away`;
  }

  private calculateDistance(lat1: number, lon1: number, lat2: number, lon2: number): number {
    const R = 6371; // Earth radius in km
    const dLat = ((lat2 - lat1) * Math.PI) / 180;
    const dLon = ((lon2 - lon1) * Math.PI) / 180;
    const a =
      Math.sin(dLat / 2) * Math.sin(dLat / 2) +
      Math.cos((lat1 * Math.PI) / 180) *
        Math.cos((lat2 * Math.PI) / 180) *
        Math.sin(dLon / 2) *
        Math.sin(dLon / 2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    return R * c;
  }

  copyPoliceReport(): void {
    const bike = this.selectedBike();
    const now = new Date().toLocaleString('en-GB', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });

    let text = `🚨 MOTOGUARD EMERGENCY POLICE REPORT\n`;
    text += `Time: ${now}\n`;

    if (bike) {
      text += `Motorcycle: ${bike.registration_number} (${bike.model || 'Motorcycle'})\n`;
      text += `Color: ${bike.color || 'N/A'} | Year: ${bike.year || 'N/A'}\n`;
      text += `Tracker IMEI: ${bike.imei}\n`;
      if (bike.latitude && bike.longitude) {
        text += `Last GPS Coordinates: ${bike.latitude.toFixed(5)}, ${bike.longitude.toFixed(5)}\n`;
        text += `Google Maps Link: https://maps.google.com/?q=${bike.latitude},${bike.longitude}\n`;
      }
      if (bike.current_address) {
        text += `Last Known Landmark: ${bike.current_address}\n`;
      }
      text += `Engine Starter: ${bike.is_immobilized ? 'IMMOBILIZED (LOCKED)' : 'ACTIVE'}\n`;
    } else {
      text += `Motorcycle details not selected.\n`;
    }

    navigator.clipboard.writeText(text).then(() => {
      this.copiedReport.set(true);
      setTimeout(() => this.copiedReport.set(false), 3000);
    });
  }

  private initMapIfNeeded(): void {
    const el = document.getElementById('emergencyMap');
    if (!el) return;

    if (!this.map) {
      // Center default on Nairobi CBD or user bike
      const bike = this.selectedBike();
      const initialLat = bike && bike.latitude ? bike.latitude : -1.2864;
      const initialLng = bike && bike.longitude ? bike.longitude : 36.8172;

      this.map = L.map('emergencyMap', {
        zoomControl: true,
      }).setView([initialLat, initialLng], 13);

      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '© OpenStreetMap contributors',
      }).addTo(this.map);

      this.markersLayer = L.layerGroup().addTo(this.map);
    } else {
      this.map.invalidateSize();
    }

    this.updateMapMarkers();
  }

  private updateMapMarkers(): void {
    if (!this.map || !this.markersLayer) return;
    this.markersLayer.clearLayers();
    this.contactMarkers.clear();

    const bike = this.selectedBike();
    // Plot user bike if present
    if (bike && bike.latitude && bike.longitude) {
      const bikeIcon = L.divIcon({
        className: 'map-bike-icon',
        html: `
          <div class="map-marker-pin map-marker-bike">
            <div class="pin-body">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="5" cy="16" r="3"/>
                <circle cx="19" cy="16" r="3"/>
                <path d="M5 16h4l3-6h5l2 3"/>
                <path d="M12 10V6h2"/>
              </svg>
            </div>
            <div class="pin-tip"></div>
          </div>
        `,
        iconSize: [32, 38],
        iconAnchor: [16, 38],
        popupAnchor: [0, -36],
      });

      this.bikeMarker = L.marker([bike.latitude, bike.longitude], { icon: bikeIcon })
        .addTo(this.markersLayer)
        .bindPopup(`
          <div style="min-width: 190px; font-family: -apple-system, BlinkMacSystemFont, sans-serif; padding: 12px;">
            <div style="font-size: 11px; font-weight: 700; text-transform: uppercase; color: #10b981; margin-bottom: 2px;">Your Motorcycle</div>
            <div style="font-size: 14px; font-weight: 700; color: #18181b; margin-bottom: 4px;">${bike.registration_number}</div>
            <div style="font-size: 12px; color: #71717a;">${bike.current_address || 'Current GPS position'}</div>
          </div>
        `, {
          autoClose: true,
          closeOnClick: true,
        });
    }

    // Plot station markers
    const list = this.filteredContacts();
    list.forEach((c) => {
      if (!c.latitude || !c.longitude) return;

      const isPolice = c.type === 'police';
      const isAmbulance = c.type === 'ambulance';
      const typeClass = isPolice ? 'map-marker-police' : isAmbulance ? 'map-marker-ambulance' : 'map-marker-national';
      const typeLabel = isPolice ? 'Police Station' : isAmbulance ? 'Ambulance Dispatch' : 'National Hotline';
      const typeColor = isPolice ? '#1d4ed8' : isAmbulance ? '#dc2626' : '#4338ca';

      // Utilitarian icon based on service type
      let iconSvg = '';
      if (isPolice) {
        // Police Siren
        iconSvg = `
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 2v2"/>
            <path d="M4.93 4.93l1.41 1.41"/>
            <path d="M19.07 4.93l-1.41 1.41"/>
            <path d="M7 11a5 5 0 0 1 10 0v5H7v-5z"/>
            <rect x="5" y="16" width="14" height="4" rx="1"/>
          </svg>
        `;
      } else if (isAmbulance) {
        // Ambulance Vehicle
        iconSvg = `
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="7" width="14" height="10" rx="1"/>
            <path d="M16 10h4l2 3v4h-6v-7z"/>
            <circle cx="6" cy="18" r="2"/>
            <circle cx="18" cy="18" r="2"/>
            <path d="M9 10v4"/>
            <path d="M7 12h4"/>
          </svg>
        `;
      } else {
        // Emergency Phone
        iconSvg = `
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/>
          </svg>
        `;
      }

      const icon = L.divIcon({
        className: 'map-contact-icon',
        html: `
          <div class="map-marker-pin ${typeClass}">
            <div class="pin-body">
              ${iconSvg}
            </div>
            <div class="pin-tip"></div>
          </div>
        `,
        iconSize: [32, 38],
        iconAnchor: [16, 38],
        popupAnchor: [0, -36],
      });

      const dist = this.getDistanceToBike(c);
      const distHtml = dist
        ? `<span style="font-size: 11px; font-weight: 700; color: #1d4ed8; background: #eff6ff; padding: 2px 6px; border-radius: 4px;">${dist}</span>`
        : '';

      const popupContent = `
        <div style="min-width: 210px; font-family: -apple-system, BlinkMacSystemFont, sans-serif; padding: 12px;">
          <div style="display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 4px;">
            <span style="font-size: 11px; font-weight: 700; text-transform: uppercase; color: ${typeColor};">${typeLabel}</span>
            ${distHtml}
          </div>
          <div style="font-size: 14px; font-weight: 700; color: #18181b; margin-bottom: 2px;">${c.name}</div>
          <div style="font-size: 12px; color: #71717a; margin-bottom: 10px;">${c.address} (${c.subCounty})</div>
          <a href="tel:${c.phone}" style="display: block; width: 100%; text-align: center; background-color: #059669; color: #ffffff; padding: 8px 12px; border-radius: 8px; font-size: 13px; font-weight: 700; text-decoration: none; box-sizing: border-box;">
            Call ${c.phone}
          </a>
        </div>
      `;

      const marker = L.marker([c.latitude, c.longitude], { icon })
        .addTo(this.markersLayer!)
        .bindPopup(popupContent, {
          autoClose: true,
          closeOnClick: true,
        });

      marker.on('click', () => {
        this.selectedContact.set(c);
      });

      this.contactMarkers.set(c.id, marker);
    });
  }
}
