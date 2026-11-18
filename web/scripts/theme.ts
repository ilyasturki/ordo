import { writeFileSync } from 'node:fs';
import { colorsCSS } from '../src/lib/theme/scheme';

writeFileSync(new URL('../src/lib/theme/colors.css', import.meta.url), colorsCSS());
