import { Component, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRouteSnapshot, NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { NgTemplateOutlet } from '@angular/common';
import { filter, map, startWith } from 'rxjs';
import { DashboardSidebarComponent } from './dashboard-sidebar/dashboard-sidebar.component';
import { MobileNavComponent } from '../../ui/mobile-nav/mobile-nav.component';
import { AuthService } from '../services/auth.service';
import { NavDrawerService } from './nav-drawer.service';
import { TopBarService } from './top-bar.service';

/** Shared chrome for signed-in pages: sidebar, top bar with the page title, and the routed page beneath. */
@Component({
  selector: 'app-shell',
  standalone: true,
  imports: [RouterOutlet, NgTemplateOutlet, DashboardSidebarComponent, MobileNavComponent],
  template: `
    <div class="flex h-[100dvh] w-screen bg-surface font-sans text-ink overflow-hidden select-none">
      <div class="contents print:hidden">
        <app-dashboard-sidebar
          [mobileDrawerOpen]="drawer.open()"
          [user]="auth.currentUser()"
          (closeMobileDrawer)="drawer.close()"
          (openGeofenceModal)="openGeofence()"
          (logout)="logout()"
        />
      </div>

      <div class="flex-1 min-w-0 flex flex-col h-full relative overflow-hidden">
        <header class="h-16 bg-surface border-b border-line px-4 lg:px-6 flex items-center gap-3 shrink-0 z-30 relative print:hidden">
          <app-mobile-nav class="shrink-0"></app-mobile-nav>
          @if (topBar.content(); as content) {
            <div class="flex-1 min-w-0 h-full">
              <ng-container *ngTemplateOutlet="content"></ng-container>
            </div>
          } @else {
            <h1 class="font-bold text-base text-ink truncate tracking-tight">{{ pageTitle() }}</h1>
          }
        </header>

        <div class="flex-1 min-h-0 flex flex-col relative overflow-hidden">
          <router-outlet />
        </div>
      </div>
    </div>
  `,
})
export class AppShellComponent {
  readonly auth = inject(AuthService);
  readonly drawer = inject(NavDrawerService);
  readonly topBar = inject(TopBarService);
  private readonly router = inject(Router);

  /** Title of the deepest active route, without the " | MGuard" suffix. */
  readonly pageTitle = toSignal(
    this.router.events.pipe(
      filter((event) => event instanceof NavigationEnd),
      startWith(null),
      map(() => this.titleOf(this.router.routerState.snapshot.root))
    ),
    { initialValue: '' }
  );

  openGeofence(): void {
    this.drawer.close();
    this.router.navigate(['/dashboard'], { queryParams: { geofence: '1' } });
  }

  logout(): void {
    this.drawer.close();
    this.auth.logout().subscribe();
  }

  private titleOf(route: ActivatedRouteSnapshot): string {
    let deepest = route;
    while (deepest.firstChild) deepest = deepest.firstChild;
    return (deepest.title ?? '').replace(/\s*\|\s*MGuard$/, '');
  }
}
