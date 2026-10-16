import { Injectable, inject, signal, effect } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import {
  AttachBikePayload,
  BikeDTO,
  BikeEventDTO,
  BikeListResponse,
  BreadcrumbDTO,
  CreateBikePayload,
  PaginatedBreadcrumbsResponse,
  PaginatedEventsResponse,
  TrackerStatusDTO,
  UpdateBikePayload,
} from '../models/bike.models';
import { AuthService } from './auth.service';
import { environment } from '../../../environments/environment';

@Injectable({
  providedIn: 'root',
})
export class BikeService {
  private readonly http = inject(HttpClient);
  private readonly authService = inject(AuthService);

  readonly bikes = signal<BikeDTO[]>([]);
  readonly defaultSelectedBikeId = signal<string | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly error = signal<string | null>(null);
  readonly isConnected = signal<boolean>(false);
  readonly latestAlert = signal<any | null>(null);

  private eventSource: EventSource | null = null;

  constructor() {
    effect(() => {
      const token = this.authService.accessToken();
      if (!token) {
        this.disconnectSSE();
        return;
      }
      this.connectSSE();
    });
  }

  connectSSE(): void {
    if (typeof window === 'undefined') {
      return;
    }

    const token = this.authService.accessToken();
    if (!token) {
      this.disconnectSSE();
      return;
    }

    const isRunning =
      this.eventSource &&
      (this.eventSource.readyState === EventSource.OPEN ||
        this.eventSource.readyState === EventSource.CONNECTING);
    if (isRunning) {
      return;
    }

    this.disconnectSSE();

    const streamUrl = `${environment.apiBaseUrl}/v1/bikes/stream`;
    const es = new EventSource(streamUrl, { withCredentials: true });
    this.eventSource = es;

    es.onopen = () => {
      this.isConnected.set(true);
    };

    es.addEventListener('bikes_init', (event: MessageEvent) => {
      this.handleBikesInit(event.data);
    });

    es.addEventListener('bike_update', (event: MessageEvent) => {
      this.handleBikeUpdate(event.data);
    });

    es.addEventListener('alert', (event: MessageEvent) => {
      this.handleAlertEvent(event.data);
    });

    es.onerror = () => {
      this.isConnected.set(false);
      if (es.readyState === EventSource.CLOSED) {
        this.disconnectSSE();
      }
    };
  }

  private handleBikesInit(rawData: string): void {
    try {
      const res: BikeListResponse = JSON.parse(rawData);
      if (!res || !Array.isArray(res.data)) {
        return;
      }

      this.bikes.set(res.data);
      this.defaultSelectedBikeId.set(res.meta?.default_selected_bike_id || null);
      this.isLoading.set(false);
    } catch (err) {
      console.error('Failed to parse bikes_init event:', err);
    }
  }

  private handleBikeUpdate(rawData: string): void {
    try {
      const bike: BikeDTO = JSON.parse(rawData);
      if (!bike?.id) {
        return;
      }

      this.bikes.update((current) => {
        const index = current.findIndex((b) => b.id === bike.id);
        if (index === -1) {
          return [bike, ...current];
        }

        const updated = [...current];
        updated[index] = bike;
        return updated;
      });
    } catch (err) {
      console.error('Failed to parse bike_update event:', err);
    }
  }

  private handleAlertEvent(rawData: string): void {
    try {
      const alert = JSON.parse(rawData);
      this.latestAlert.set(alert);
    } catch (err) {
      console.error('Failed to parse alert event:', err);
    }
  }

  disconnectSSE(): void {
    if (!this.eventSource) {
      return;
    }
    this.eventSource.close();
    this.eventSource = null;
    this.isConnected.set(false);
  }

  loadBikes(cursor?: string, limit = 25): Observable<BikeListResponse> {
    this.isLoading.set(true);
    this.error.set(null);

    let params = new HttpParams().set('limit', limit.toString());
    if (cursor) {
      params = params.set('cursor', cursor);
    }

    return this.http.get<BikeListResponse>(`${environment.apiBaseUrl}/v1/bikes`, { params }).pipe(
      tap({
        next: (res) => {
          this.bikes.set(res.data);
          this.defaultSelectedBikeId.set(res.meta?.default_selected_bike_id || null);
          this.isLoading.set(false);
        },
        error: (err) => {
          this.error.set(err?.error?.detail || 'Failed to load bikes');
          this.isLoading.set(false);
        },
      })
    );
  }

  getBike(id: string): Observable<{ data: BikeDTO }> {
    return this.http.get<{ data: BikeDTO }>(`${environment.apiBaseUrl}/v1/bikes/${id}`);
  }

  getBikeBreadcrumbs(
    id: string,
    limit = 25,
    date?: string,
    startDate?: string,
    endDate?: string,
    cursor?: string
  ): Observable<PaginatedBreadcrumbsResponse> {
    let params = new HttpParams().set('limit', limit.toString());
    if (cursor) {
      params = params.set('cursor', cursor);
    }
    if (date) {
      params = params.set('date', date);
    }
    if (!date && startDate && endDate) {
      params = params.set('startDate', startDate).set('endDate', endDate);
    }

    return this.http.get<PaginatedBreadcrumbsResponse>(
      `${environment.apiBaseUrl}/v1/bikes/${id}/breadcrumbs`,
      { params }
    );
  }

  getBikeEvents(
    id: string,
    limit = 25,
    date?: string,
    startDate?: string,
    endDate?: string,
    cursor?: string
  ): Observable<PaginatedEventsResponse> {
    let params = new HttpParams().set('limit', limit.toString());
    if (cursor) {
      params = params.set('cursor', cursor);
    }
    if (date) {
      params = params.set('date', date);
    }
    if (!date && startDate && endDate) {
      params = params.set('startDate', startDate).set('endDate', endDate);
    }

    return this.http.get<PaginatedEventsResponse>(
      `${environment.apiBaseUrl}/v1/bikes/${id}/events`,
      { params }
    );
  }

  getAllEvents(
    limit = 10,
    cursor?: string,
    severity?: string
  ): Observable<PaginatedEventsResponse> {
    let params = new HttpParams().set('limit', limit.toString());
    if (cursor) {
      params = params.set('cursor', cursor);
    }
    if (severity && severity !== 'ALL') {
      params = params.set('severity', severity);
    }

    return this.http.get<PaginatedEventsResponse>(
      `${environment.apiBaseUrl}/v1/bikes/events`,
      { params }
    );
  }

  createBike(payload: CreateBikePayload): Observable<{ data: BikeDTO }> {
    this.isLoading.set(true);
    return this.http.post<{ data: BikeDTO }>(`${environment.apiBaseUrl}/v1/bikes`, payload).pipe(
      tap({
        next: (res) => {
          this.bikes.update((current) => [res.data, ...current]);
          this.isLoading.set(false);
        },
        error: () => {
          this.isLoading.set(false);
        },
      })
    );
  }

  updateBike(id: string, payload: UpdateBikePayload): Observable<{ data: BikeDTO }> {
    return this.http.patch<{ data: BikeDTO }>(`${environment.apiBaseUrl}/v1/bikes/${id}`, payload).pipe(
      tap({
        next: (res) => {
          this.bikes.update((current) =>
            current.map((b) => (b.id === id ? res.data : b))
          );
        },
      })
    );
  }

  deleteBike(id: string): Observable<void> {
    return this.http.delete<void>(`${environment.apiBaseUrl}/v1/bikes/${id}`).pipe(
      tap({
        next: () => {
          this.bikes.update((current) => current.filter((b) => b.id !== id));
        },
      })
    );
  }

  toggleImmobilize(id: string, immobilize: boolean): Observable<{ data: BikeDTO }> {
    const action = immobilize ? 'IMMOBILIZE' : 'REACTIVATE';
    return this.http.post<{ data: BikeDTO }>(`${environment.apiBaseUrl}/v1/bikes/${id}/immobilize`, { action }).pipe(
      tap({
        next: (res) => {
          this.bikes.update((current) =>
            current.map((b) => (b.id === id ? res.data : b))
          );
        },
      })
    );
  }

  verifyTracker(imei: string): Observable<{ data: TrackerStatusDTO }> {
    return this.http.post<{ data: TrackerStatusDTO }>(`${environment.apiBaseUrl}/v1/bikes/verify-tracker`, { imei });
  }

  attachBike(payload: AttachBikePayload): Observable<{ data: BikeDTO; message?: string }> {
    this.isLoading.set(true);
    return this.http.post<{ data: BikeDTO; message?: string }>(`${environment.apiBaseUrl}/v1/bikes/attach`, payload).pipe(
      tap({
        next: (res) => {
          this.bikes.update((current) => [res.data, ...current]);
          this.defaultSelectedBikeId.set(res.data.id);
          this.isLoading.set(false);
        },
        error: () => {
          this.isLoading.set(false);
        },
      })
    );
  }
}
