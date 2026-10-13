import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-duress-modal',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './duress-modal.component.html',
})
export class DuressModalComponent {
  readonly isOpen = input<boolean>(false);
  readonly countdown = input<number>(30);

  readonly cancel = output<void>();
  readonly escalate = output<void>();
}
