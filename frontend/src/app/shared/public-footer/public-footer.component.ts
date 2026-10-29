import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-public-footer',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './public-footer.component.html',
  styleUrls: ['./public-footer.component.css'],
})
export class PublicFooterComponent {
  readonly year = new Date().getFullYear();
}
