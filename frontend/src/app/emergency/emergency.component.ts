import { Component, OnInit, AfterViewInit, OnDestroy, computed, inject, signal, ViewEncapsulation } from '@angular/core';
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
export class EmergencyComponent implements OnInit, AfterViewInit, OnDestroy {
  private readonly bikeService = inject(BikeService);

  readonly contacts = signal<EmergencyContact[]>(EMERGENCY_CONTACTS);
  readonly searchQuery = signal<string>('');
  readonly selectedType = signal<'ALL' | 'police' | 'ambulance' | 'national'>('ALL');
  readonly selectedCounty = signal<string>('ALL');
  readonly activeView = signal<'split' | 'list' | 'map'>('map');
  readonly selectedContact = signal<EmergencyContact | null>(null);

  // User bikes for location proximity
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
      if (type !== 'ALL' && c.type !== type) return false;
      if (county !== 'ALL' && c.county !== county && c.county !== 'National') return false;

      if (!q) return true;
      const haystack = `${c.name} ${c.county} ${c.subCounty} ${c.address} ${c.phone} ${c.altPhone || ''}`.toLowerCase();
      return haystack.includes(q);
    });

    if (bike && bike.latitude && bike.longitude) {
      list = [...list].sort((a, b) => {
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
    this.bikeService.loadBikes().subscribe({
      next: (res: BikeListResponse) => {
        const list = res.data || [];
        this.bikes.set(list);
        if (list.length > 0) {
          this.selectedBike.set(list[0]);
          if (this.map && list[0].latitude && list[0].longitude) {
            this.map.setView([list[0].latitude, list[0].longitude], 12);
          }
          this.updateMapMarkers();
        }
      },
    });

    if (typeof window !== 'undefined' && window.innerWidth >= 1024) {
      this.activeView.set('split');
    } else {
      this.activeView.set('map');
    }
  }

  ngAfterViewInit(): void {
    setTimeout(() => {
      this.initMapIfNeeded();
      this.map?.invalidateSize();
    }, 100);
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
      setTimeout(() => {
        this.initMapIfNeeded();
        this.map?.invalidateSize();
      }, 50);
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

  private initMapIfNeeded(): void {
    const el = document.getElementById('emergencyMap');
    if (!el) return;

    if (!this.map) {
      const bike = this.selectedBike();
      const initialLat = bike && bike.latitude ? bike.latitude : -1.2864;
      const initialLng = bike && bike.longitude ? bike.longitude : 36.8172;

      this.map = L.map('emergencyMap', {
        zoomControl: false,
      }).setView([initialLat, initialLng], 12);

      // Free OpenStreetMap tiles with custom muted filter (NO API KEY REQUIRED)
      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        className: 'emergency-osm-tiles',
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
      }).addTo(this.map);

      this.markersLayer = L.layerGroup().addTo(this.map);
    }

    setTimeout(() => {
      this.map?.invalidateSize();
    }, 100);

    this.updateMapMarkers();
  }

  private updateMapMarkers(): void {
    if (!this.map || !this.markersLayer) return;
    this.markersLayer.clearLayers();
    this.contactMarkers.clear();

    const bike = this.selectedBike();
    // Plot user bike if present
    if (bike && bike.latitude && bike.longitude) {
      const bikeIcon = L.icon({
        iconUrl: '/motorbike.png',
        iconSize: [42, 42],
        iconAnchor: [21, 21],
        popupAnchor: [0, -21],
      });

      this.bikeMarker = L.marker([bike.latitude, bike.longitude], { icon: bikeIcon })
        .addTo(this.markersLayer)
        .bindPopup(`
          <div style="min-width: 190px; font-family: -apple-system, BlinkMacSystemFont, sans-serif; padding: 12px;">
            <div style="font-size: 11px; font-weight: 700; text-transform: uppercase; color: #52525b; margin-bottom: 2px;">Your Motorcycle</div>
            <div style="font-size: 14px; font-weight: 700; color: #18181b; margin-bottom: 4px;">${bike.registration_number}</div>
            <div style="font-size: 12px; color: #71717a;">${bike.current_address || 'Current GPS position'}</div>
          </div>
        `, {
          autoClose: true,
          closeOnClick: true,
        });
    }

    const policeIcon = L.icon({
      iconUrl: '/siren.png',
      iconSize: [38, 38],
      iconAnchor: [19, 19],
      popupAnchor: [0, -19],
    });

    const ambulanceIcon = L.icon({
      iconUrl: '/ambulance.png',
      iconSize: [38, 38],
      iconAnchor: [19, 19],
      popupAnchor: [0, -19],
    });

    const hotlineIcon = L.icon({
      iconUrl: '/emergency-phone.png',
      iconSize: [38, 38],
      iconAnchor: [19, 19],
      popupAnchor: [0, -19],
    });

    // Plot station markers
    const list = this.filteredContacts();
    list.forEach((c) => {
      if (!c.latitude || !c.longitude) return;

      const isPolice = c.type === 'police';
      const isAmbulance = c.type === 'ambulance';
      const typeLabel = isPolice ? 'Police Station' : isAmbulance ? 'Ambulance Dispatch' : 'National Hotline';
      const typeColor = isPolice ? '#1e40af' : isAmbulance ? '#b91c1c' : '#475569';
      const icon = isPolice ? policeIcon : isAmbulance ? ambulanceIcon : hotlineIcon;

      const dist = this.getDistanceToBike(c);
      const distHtml = dist
        ? `<span style="font-size: 11px; font-weight: 600; color: #52525b; background: #f4f4f5; padding: 2px 6px; border-radius: 4px;">${dist}</span>`
        : '';

      const popupContent = `
        <div style="min-width: 210px; font-family: -apple-system, BlinkMacSystemFont, sans-serif; padding: 12px;">
          <div style="display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 4px;">
            <span style="font-size: 11px; font-weight: 700; text-transform: uppercase; color: ${typeColor};">${typeLabel}</span>
            ${distHtml}
          </div>
          <div style="font-size: 14px; font-weight: 700; color: #18181b; margin-bottom: 2px;">${c.name}</div>
          <div style="font-size: 12px; color: #71717a; margin-bottom: 10px;">${c.address} (${c.subCounty})</div>
          <a href="tel:${c.phone}" style="display: block; width: 100%; text-align: center; background-color: #18181b; color: #ffffff; padding: 7px 10px; border-radius: 6px; font-size: 12px; font-weight: 600; text-decoration: none; box-sizing: border-box;">
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
