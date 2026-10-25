import { Component, inject } from '@angular/core';
import { NavDrawerService } from '../../core/layout/nav-drawer.service';

/** Menu button for page headers. The drawer itself is rendered once by AppShellComponent. */
@Component({
  selector: 'app-mobile-nav',
  standalone: true,
  template: `
    <button
      type="button"
      (click)="drawer.toggle()"
      title="Open Menu"
      class="lg:hidden w-10 h-10 rounded-xl flex items-center justify-center text-ink-soft hover:bg-fill transition-colors cursor-pointer shrink-0"
    >
      <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
      </svg>
    </button>
  `,
})
export class MobileNavComponent {
  readonly drawer = inject(NavDrawerService);
}
