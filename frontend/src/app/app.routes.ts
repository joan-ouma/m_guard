import { inject } from '@angular/core';
import { Routes } from '@angular/router';
import { authGuard, guestGuard } from './core/guards/auth.guard';
import { AuthService } from './core/services/auth.service';
import { LoginComponent } from './auth/login/login.component';
import { SignupComponent } from './auth/signup/signup.component';
import { ForgotPasswordComponent } from './auth/forgot-password/forgot-password.component';
import { ResetPasswordComponent } from './auth/reset-password/reset-password.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import { BikesComponent } from './bikes/bikes.component';
import { HistoryComponent } from './history/history.component';
import { AlertsComponent } from './alerts/alerts.component';
import { SettingsComponent } from './settings/settings.component';
import { LegalComponent } from './legal/legal.component';
import { EmergencyComponent } from './emergency/emergency.component';
import { LandingComponent } from './landing/landing.component';

export const routes: Routes = [
  {
    path: 'terms',
    component: LegalComponent,
    title: 'Terms of Service | MotoGuard',
    data: {
      description:
        'Read the terms and conditions governing the use of MotoGuard motorcycle tracking hardware, software, and fleet management services.',
      keywords: 'MotoGuard terms and conditions, terms of service, telematics service agreement',
      ogType: 'article',
    },
  },
  {
    path: 'privacy',
    component: LegalComponent,
    title: 'Privacy Policy | MotoGuard',
    data: {
      description:
        'Learn how MotoGuard collects, protects, and handles GPS telemetry, rider personal information, and motorcycle location data.',
      keywords: 'MotoGuard privacy policy, GPS data protection, telemetry privacy, rider data security',
      ogType: 'article',
    },
  },
  {
    path: 'legal',
    component: LegalComponent,
    title: 'Legal | MotoGuard',
    data: {
      description:
        'Access MotoGuard terms of service, privacy policy, and compliance standards for motorcycle fleet management.',
      keywords: 'MotoGuard legal, compliance, terms, privacy, fleet telematics regulations',
      ogType: 'article',
    },
  },
  {
    path: 'alerts',
    component: AlertsComponent,
    canActivate: [authGuard],
    title: 'Alerts | MotoGuard',
    data: {
      description:
        'View critical motorcycle safety alerts including tilt tamper warnings, geofence breaches, duress escalation, and emergency accident notifications.',
      keywords: 'motorcycle security alerts, bike theft notification, tamper alarm, geofence breach, duress alert',
    },
  },
  {
    path: 'emergency',
    component: EmergencyComponent,
    canActivate: [authGuard],
    title: 'Emergency Contacts & Police | MotoGuard',
    data: {
      description:
        'Quickly access police station hotlines, desk numbers, ambulance emergency services, and live motorcycle incident reporting across Kenya.',
      keywords: 'police stations Kenya, motorcycle theft report, emergency contacts, ambulance dispatch, Nairobi police numbers',
    },
  },
  {
    path: 'settings',
    component: SettingsComponent,
    canActivate: [authGuard],
    title: 'Settings | MotoGuard',
    data: {
      description:
        'Configure your SACCO leader contacts, emergency escalation phone numbers, notification channels, and fleet security preferences.',
      keywords: 'SACCO emergency contacts, motorcycle settings, alert escalation, fleet configuration',
    },
  },
  {
    path: '',
    canMatch: [() => inject(AuthService).isAuthenticated()],
    component: DashboardComponent,
    canActivate: [authGuard],
    title: 'Dashboard | MotoGuard',
    data: {
      description:
        'Track your motorcycles in real-time with high-precision GPS, live telemetry, geofence boundary alerts, and remote anti-theft starter immobilization.',
      keywords: 'motorcycle GPS tracking, live bike location, fleet dashboard, IoT anti-theft, ignition control, Nairobi motorcycle tracking',
    },
  },
  {
    path: '',
    component: LandingComponent,
    title: 'MotoGuard | Smart Motorcycle GPS Tracking & Security',
    data: {
      description:
        'Real-time IoT motorcycle tracking, intelligent geofencing, remote starter immobilization, and automated theft prevention for riders and fleet operators.',
      keywords: 'motorcycle tracking, GPS bike security, fleet telematics, IoT immobilizer, Kenya bike tracking, duress alerts, live route history, SACCO motorcycle tracking, MotoGuard, MGuard',
    },
  },
  {
    path: 'home',
    component: LandingComponent,
    title: 'MotoGuard | Smart Motorcycle GPS Tracking & Security',
  },
  {
    path: 'dashboard',
    component: DashboardComponent,
    canActivate: [authGuard],
    title: 'Dashboard | MotoGuard',
  },
  {
    path: 'bikes',
    component: BikesComponent,
    canActivate: [authGuard],
    title: 'Bikes | MotoGuard',
    data: {
      description:
        'Manage all motorcycles in your fleet. Monitor tracker hardware health, GPS signal quality, battery voltage, and remote immobilization status.',
      keywords: 'motorcycle fleet management, bike inventory, tracker status, battery health, IoT telematics',
    },
  },
  {
    path: 'history',
    component: HistoryComponent,
    canActivate: [authGuard],
    title: 'History | MotoGuard',
    data: {
      description:
        'Replay historical motorcycle routes, inspect speed records, identify idle times, and review verified incident logs and trip audit trails.',
      keywords: 'motorcycle route history, GPS breadcrumbs, trip replay, speed tracking, bike incident audit',
    },
  },
  {
    path: 'login',
    component: LoginComponent,
    canActivate: [guestGuard],
    title: 'Sign In | MotoGuard',
    data: {
      description:
        'Sign in to access your MotoGuard motorcycle tracking portal, monitor active assets on live maps, and control ignition starter security.',
      keywords: 'MotoGuard login, motorcycle portal sign in, GPS tracking access',
    },
  },
  {
    path: 'signup',
    component: SignupComponent,
    canActivate: [guestGuard],
    title: 'Sign Up | MotoGuard',
    data: {
      description:
        'Create your MotoGuard account, pair your IoT GPS tracking device, and safeguard your motorcycle with intelligent remote fleet protection.',
      keywords: 'motorcycle tracker registration, create MotoGuard account, bike security onboarding',
    },
  },
  {
    path: 'forgot-password',
    component: ForgotPasswordComponent,
    canActivate: [guestGuard],
    title: 'Forgot Password | MotoGuard',
    data: {
      description: 'Request a secure password recovery link for your MotoGuard motorcycle tracking account.',
      keywords: 'MotoGuard forgot password, recover tracking account',
    },
  },
  {
    path: 'reset-password',
    component: ResetPasswordComponent,
    canActivate: [guestGuard],
    title: 'Reset Password | MotoGuard',
    data: {
      description: 'Securely set a new password for your MotoGuard motorcycle security and fleet tracking account.',
      keywords: 'MotoGuard reset password, account security',
    },
  },
  {
    path: 'reset-password/:token',
    component: ResetPasswordComponent,
    canActivate: [guestGuard],
    title: 'Reset Password | MotoGuard',
    data: {
      description: 'Securely set a new password for your MotoGuard motorcycle security and fleet tracking account.',
      keywords: 'MotoGuard reset password, account security',
    },
  },
  {
    path: '**',
    redirectTo: '',
  },
];
