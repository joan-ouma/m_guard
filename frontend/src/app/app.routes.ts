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
import { RecoveryComponent } from './recovery/recovery.component';
import { ContactsComponent } from './contacts/contacts.component';
import { LandingComponent } from './landing/landing.component';
import { AboutComponent } from './pages/about/about.component';
import { ServicesComponent } from './pages/services/services.component';
import { ContactComponent } from './pages/contact/contact.component';
import { AppShellComponent } from './core/layout/app-shell.component';

export const routes: Routes = [
  {
    path: '',
    // Authenticated pages share one sidebar. A bare URL only renders the shell for signed-in users; otherwise it falls through to the landing page.
    canMatch: [(_route, segments) => segments.length > 0 || inject(AuthService).isAuthenticated()],
    component: AppShellComponent,
    canActivate: [authGuard],
    children: [
      {
        path: '',
        component: DashboardComponent,
        title: 'Dashboard | MGuard',
        data: {
          description:
            'Track your motorcycle in real-time with high-precision GPS, live telemetry, geofence boundary alerts, and remote anti-theft security lock.',
          keywords: 'motorcycle GPS tracking, live bike location, rider dashboard, anti-theft, ignition control, Nairobi motorcycle tracking',
        },
      },
      {
        path: 'dashboard',
        component: DashboardComponent,
        title: 'Dashboard | MGuard',
      },
      {
        path: 'bikes',
        component: BikesComponent,
        title: 'Bikes | MGuard',
        data: {
          description:
            'Manage your motorcycle. Monitor connection health, GPS signal quality, battery voltage, and remote security lock status.',
          keywords: 'motorcycle management, bike status, battery health, rider security',
        },
      },
      {
        path: 'history',
        component: HistoryComponent,
        title: 'History | MGuard',
        data: {
          description:
            'Replay historical motorcycle routes, inspect speed records, identify idle times, and review verified incident logs and trip audit trails.',
          keywords: 'motorcycle route history, GPS breadcrumbs, trip replay, speed tracking, bike incident audit',
        },
      },
      {
        path: 'alerts',
        component: AlertsComponent,
        title: 'Alerts | MGuard',
        data: {
          description:
            'View critical motorcycle safety alerts including tilt tamper warnings, geofence breaches, duress escalation, and emergency accident notifications.',
          keywords: 'motorcycle security alerts, bike theft notification, tamper alarm, geofence breach, duress alert',
        },
      },
      {
        path: 'emergency',
        component: EmergencyComponent,
        title: 'Emergency Contacts & Police | MGuard',
        data: {
          description:
            'Quickly access police station hotlines, desk numbers, ambulance emergency services, and live motorcycle incident reporting across Kenya.',
          keywords: 'police stations Kenya, motorcycle theft report, emergency contacts, ambulance dispatch, Nairobi police numbers',
        },
      },
      {
        path: 'recovery',
        component: RecoveryComponent,
        title: 'Bike Recovery | MGuard',
        data: {
          description: 'Report a lost or stolen motorcycle, record how and where it was lost, and file your police OB number.',
          keywords: 'motorcycle recovery, bike theft report, OB number, police report Kenya',
        },
      },
      {
        path: 'contacts',
        component: ContactsComponent,
        title: 'Contacts | MGuard',
        data: {
          description: 'Manage emergency contacts and insurance brokers to notify when your motorcycle is lost or stolen.',
          keywords: 'emergency contacts, insurance broker, motorcycle insurance contact',
        },
      },
      {
        path: 'settings',
        component: SettingsComponent,
        title: 'Settings | MGuard',
        data: {
          description:
            'Configure your trusted emergency contacts, notification channels, and motorcycle security preferences.',
          keywords: 'emergency contacts, motorcycle settings, alert escalation, security configuration',
        },
      },
    ],
  },
  {
    path: 'terms',
    component: LegalComponent,
    title: 'Terms of Service | MGuard',
    data: {
      description:
        'Read the terms and conditions governing the use of MGuard motorcycle tracking and security services.',
      keywords: 'MGuard terms and conditions, terms of service, motorcycle security agreement',
      ogType: 'article',
    },
  },
  {
    path: 'privacy',
    component: LegalComponent,
    title: 'Privacy Policy | MGuard',
    data: {
      description:
        'Learn how MGuard collects, protects, and handles GPS telemetry, rider personal information, and motorcycle location data.',
      keywords: 'MGuard privacy policy, GPS data protection, telemetry privacy, rider data security',
      ogType: 'article',
    },
  },
  {
    path: 'legal',
    component: LegalComponent,
    title: 'Legal | MGuard',
    data: {
      description:
        'Access MGuard terms of service, privacy policy, and compliance standards for motorcycle security.',
      keywords: 'MGuard legal, compliance, terms, privacy, rider data protection',
      ogType: 'article',
    },
  },
  {
    path: '',
    component: LandingComponent,
    title: 'MGuard | Smart Motorcycle GPS Tracking & Security',
    data: {
      description:
        'Real-time motorcycle tracking, intelligent geofencing, remote security lock, and automated theft prevention built for boda boda riders.',
      keywords: 'motorcycle tracking, GPS bike security, boda boda tracking, Kenya bike tracking, emergency alerts, live route history, MGuard',
    },
  },
  {
    path: 'home',
    component: LandingComponent,
    title: 'MGuard | Smart Motorcycle GPS Tracking & Security',
  },
  {
    path: 'about',
    component: AboutComponent,
    title: 'About Us | MGuard',
    data: {
      description:
        'Learn how MGuard safeguards boda boda riders across Kenya through smart motorcycle tracking and real-time security.',
      keywords: 'about MGuard, motorcycle security Kenya, boda boda tracking, boda boda anti-theft',
    },
  },
  {
    path: 'services',
    component: ServicesComponent,
    title: 'Services | MGuard',
    data: {
      description:
        'Explore MGuard services: real-time GPS tracking, remote security lock, instant motion alerts, and route history.',
      keywords: 'motorcycle tracking services, remote engine cutoff, boda boda tracking, GPS telemetry',
    },
  },
  {
    path: 'contact',
    component: ContactComponent,
    title: 'Contact Us | MGuard',
    data: {
      description:
        'Get in touch with MGuard. Reach our support team in Nairobi for motorcycle security activation and technical support.',
      keywords: 'contact MGuard, motorcycle GPS Nairobi, tracker support',
    },
  },
  {
    path: 'login',
    component: LoginComponent,
    canActivate: [guestGuard],
    title: 'Sign In | MGuard',
    data: {
      description:
        'Sign in to access your MGuard motorcycle tracking portal, monitor your bike on live maps, and control remote security lock.',
      keywords: 'MGuard login, motorcycle portal sign in, GPS tracking access',
    },
  },
  {
    path: 'signup',
    component: SignupComponent,
    canActivate: [guestGuard],
    title: 'Sign Up | MGuard',
    data: {
      description:
        'Create your MGuard account, pair your motorcycle security, and safeguard your bike with 24/7 intelligent protection.',
      keywords: 'motorcycle registration, create MGuard account, bike security onboarding',
    },
  },
  {
    path: 'forgot-password',
    component: ForgotPasswordComponent,
    canActivate: [guestGuard],
    title: 'Forgot Password | MGuard',
    data: {
      description: 'Request a secure password recovery link for your MGuard motorcycle tracking account.',
      keywords: 'MGuard forgot password, recover tracking account',
    },
  },
  {
    path: 'reset-password',
    component: ResetPasswordComponent,
    canActivate: [guestGuard],
    title: 'Reset Password | MGuard',
    data: {
      description: 'Securely set a new password for your MGuard motorcycle security account.',
      keywords: 'MGuard reset password, account security',
    },
  },
  {
    path: 'reset-password/:token',
    component: ResetPasswordComponent,
    canActivate: [guestGuard],
    title: 'Reset Password | MGuard',
    data: {
      description: 'Securely set a new password for your MGuard motorcycle security account.',
      keywords: 'MGuard reset password, account security',
    },
  },
  {
    path: '**',
    redirectTo: '',
  },
];
