import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, switchMap, throwError } from 'rxjs';
import { AuthService } from '../services/auth.service';

/**
 * Functional HTTP Interceptor:
 * 1. Attaches Bearer JWT access token to requests.
 * 2. Ensures credentials (cookies) are sent for session renewal.
 * 3. Handles 401 Unauthorized by attempting session refresh.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  const token = authService.accessToken();

  // Public authentication endpoints that shouldn't attach Bearer token or loop refresh
  const isAuthEndpoint =
    req.url.includes('/v1/sessions') ||
    req.url.includes('/v1/password-resets');

  let authReq = req;

  // Clone request with Bearer header if token exists and not logging in
  if (token && !req.headers.has('Authorization')) {
    authReq = req.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`,
      },
      withCredentials: true,
    });
  } else {
    authReq = req.clone({
      withCredentials: true,
    });
  }

  return next(authReq).pipe(
    catchError((error: HttpErrorResponse) => {
      // If 401 on login or password reset, let the component handle the error message directly
      if (error.status === 401 && !isAuthEndpoint && token) {
        // Attempt token refresh via HttpOnly refresh cookie
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
            authService.logout();
            return throwError(() => refreshErr);
          })
        );
      }

      return throwError(() => error);
    })
  );
};
