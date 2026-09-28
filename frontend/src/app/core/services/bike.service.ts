import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import {
  BikeDTO,
  BikeEventDTO,
  BikeListResponse,
  BreadcrumbDTO,
  CreateBikePayload,
  UpdateBikePayload,
} from '../models/bike.models';

@Injectable({
  providedIn: 'root',
})
export class BikeService {
  private readonly http = inject(HttpClient);

  readonly bikes = signal<BikeDTO[]>([]);
  readonly defaultSelectedBikeId = signal<string | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly error = signal<string | null>(null);

  loadBikes(): Observable<BikeListResponse> {
    this.isLoading.set(true);
    this.error.set(null);

    return this.http.get<BikeListResponse>('/v1/bikes').pipe(
      tap({
        next: (res) => {
          this.bikes.set(res.data);
          this.defaultSelectedBikeId.set(res.meta.default_selected_bike_id || null);
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
    return this.http.get<{ data: BikeDTO }>(`/v1/bikes/${id}`);
  }

  getBikeBreadcrumbs(
    id: string,
    limit = 1000,
    date?: string,
    startDate?: string,
    endDate?: string
  ): Observable<{ data: BreadcrumbDTO[] }> {
    let url = `/v1/bikes/${id}/breadcrumbs?limit=${limit}`;
    if (date) {
      url += `&date=${encodeURIComponent(date)}`;
    } else if (startDate && endDate) {
      url += `&startDate=${encodeURIComponent(startDate)}&endDate=${encodeURIComponent(endDate)}`;
    }
    return this.http.get<{ data: BreadcrumbDTO[] }>(url);
  }

  getBikeEvents(
    id: string,
    limit = 1000,
    date?: string,
    startDate?: string,
    endDate?: string
  ): Observable<{ data: BikeEventDTO[] }> {
    let url = `/v1/bikes/${id}/events?limit=${limit}`;
    if (date) {
      url += `&date=${encodeURIComponent(date)}`;
    } else if (startDate && endDate) {
      url += `&startDate=${encodeURIComponent(startDate)}&endDate=${encodeURIComponent(endDate)}`;
    }
    return this.http.get<{ data: BikeEventDTO[] }>(url);
  }

  createBike(payload: CreateBikePayload): Observable<{ data: BikeDTO }> {
    this.isLoading.set(true);
    return this.http.post<{ data: BikeDTO }>('/v1/bikes', payload).pipe(
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
    return this.http.patch<{ data: BikeDTO }>(`/v1/bikes/${id}`, payload).pipe(
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
    return this.http.delete<void>(`/v1/bikes/${id}`).pipe(
      tap({
        next: () => {
          this.bikes.update((current) => current.filter((b) => b.id !== id));
        },
      })
    );
  }

  toggleImmobilize(id: string, immobilize: boolean): Observable<{ data: BikeDTO }> {
    const action = immobilize ? 'IMMOBILIZE' : 'REACTIVATE';
    return this.http.post<{ data: BikeDTO }>(`/v1/bikes/${id}/immobilize`, { action }).pipe(
      tap({
        next: (res) => {
          this.bikes.update((current) =>
            current.map((b) => (b.id === id ? res.data : b))
          );
        },
      })
    );
  }
}

