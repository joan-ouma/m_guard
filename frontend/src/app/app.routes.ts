import { Routes } from '@angular/router';
import { authGuard, guestGuard } from './core/guards/auth.guard';
import { LoginComponent } from './auth/login/login.component';
<<<<<<< HEAD
import { ForgotPasswordComponent } from './auth/forgot-password/forgot-password.component';
import { ResetPasswordComponent } from './auth/reset-password/reset-password.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import { BikesComponent } from './bikes/bikes.component';
import { HistoryComponent } from './history/history.component';

export const routes: Routes = [
  {
    path: '',
    component: DashboardComponent,
    canActivate: [authGuard],
  },
  {
    path: 'bikes',
    component: BikesComponent,
=======
import { SignupComponent } from './auth/signup/signup.component';
import { ForgotPasswordComponent } from './auth/forgot-password/forgot-password.component';
import { ResetPasswordComponent } from './auth/reset-password/reset-password.component';
import { DashboardComponent } from './dashboard/dashboard.component';

import { HistoryComponent } from './history/history.component';
import { AlertsComponent } from './alerts/alerts.component';
import { SettingsComponent } from './settings/settings.component';

export const routes: Routes = [
  {
    path: 'alerts',
    component: AlertsComponent,
    canActivate: [authGuard],
  },
  {
    path: 'settings',
    component: SettingsComponent,
    canActivate: [authGuard],
  },
  {
    path: '',
    component: DashboardComponent,
>>>>>>> origin/main
    canActivate: [authGuard],
  },
  {
    path: 'history',
    component: HistoryComponent,
    canActivate: [authGuard],
  },
  {
    path: 'login',
    component: LoginComponent,
    canActivate: [guestGuard],
  },
  {
<<<<<<< HEAD
=======
    path: 'signup',
    component: SignupComponent,
    canActivate: [guestGuard],
  },
  {
>>>>>>> origin/main
    path: 'forgot-password',
    component: ForgotPasswordComponent,
    canActivate: [guestGuard],
  },
  {
    path: 'reset-password',
    component: ResetPasswordComponent,
    canActivate: [guestGuard],
  },
  {
    path: 'reset-password/:token',
    component: ResetPasswordComponent,
    canActivate: [guestGuard],
  },
  {
    path: '**',
    redirectTo: '',
  },
];
