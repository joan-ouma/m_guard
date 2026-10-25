import { Injectable, TemplateRef, signal } from '@angular/core';

/**
 * Lets a page place its own controls into the shell's top bar.
 * The page owns the template, so its bindings and events still run against the page.
 */
@Injectable({ providedIn: 'root' })
export class TopBarService {
  readonly content = signal<TemplateRef<unknown> | null>(null);
}
