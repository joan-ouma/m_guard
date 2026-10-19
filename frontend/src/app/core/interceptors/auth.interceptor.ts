import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, switchMap, throwError } from 'rxjs';
import { AuthService } from '../services/auth.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  const token = authService.accessToken();

  const isAuthEndpoint =
    req.url.includes('/v1/sessions') ||
    req.url.includes('/v1/password-resets');

  const shouldAttachToken = !!token && !req.headers.has('Authorization');
  const authReq = req.clone({
    setHeaders: shouldAttachToken ? { Authorization: `Bearer ${token}` } : {},
    withCredentials: true,
  });

  const shouldProactivelyRefresh = !!token && !isAuthEndpoint && authService.isTokenExpired(token);
  if (shouldProactivelyRefresh) {
    return authService.refreshToken().pipe(
      switchMap((newAuth) => {
        const refreshedReq = req.clone({
          setHeaders: {
            Authorization: `Bearer ${newAuth.data.access_token}`,
          },
          withCredentials: true,
        });
        return next(refreshedReq);
      }),
      catchError((refreshErr) => {
        authService.logout().subscribe();
        return throwError(() => refreshErr);
      })
    );
  }

  return next(authReq).pipe(
    catchError((error: HttpErrorResponse) => {
      const isExpired401 = error.status === 401 && !isAuthEndpoint && !!token && authService.isTokenExpired(token);
      if (!isExpired401) {
        return throwError(() => error);
      }

      return authService.refreshToken().pipe(
        switchMap((newAuth) => {
          const refreshedReq = req.clone({
            setHeaders: {
              Authorization: `Bearer ${newAuth.data.access_token}`,
            },
            withCredentials: true,
          });
          return next(refreshedReq);
        }),
        catchError((refreshErr) => {
          authService.logout().subscribe();
          return throwError(() => refreshErr);
        })
      );
    })
  );
};
