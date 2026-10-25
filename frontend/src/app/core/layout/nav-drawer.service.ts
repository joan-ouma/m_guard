import { Injectable, signal } from '@angular/core';

/** Open state of the mobile navigation drawer, shared by the shell sidebar and every page's menu button. */
@Injectable({ providedIn: 'root' })
export class NavDrawerService {
  readonly open = signal(false);

  toggle(): void {
    this.open.update((v) => !v);
  }

  close(): void {
    this.open.set(false);
  }
}
