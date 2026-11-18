import {
	argbFromHex,
	Hct,
	hexFromArgb,
	MaterialDynamicColors,
	SchemeTonalSpot
} from '@material/material-color-utilities';

export const seed = '#3949AB';

const roles = [
	'primary',
	'onPrimary',
	'primaryContainer',
	'onPrimaryContainer',
	'inversePrimary',
	'secondary',
	'onSecondary',
	'secondaryContainer',
	'onSecondaryContainer',
	'tertiary',
	'onTertiary',
	'tertiaryContainer',
	'onTertiaryContainer',
	'error',
	'onError',
	'errorContainer',
	'onErrorContainer',
	'background',
	'onBackground',
	'surface',
	'onSurface',
	'surfaceVariant',
	'onSurfaceVariant',
	'surfaceDim',
	'surfaceBright',
	'surfaceContainerLowest',
	'surfaceContainerLow',
	'surfaceContainer',
	'surfaceContainerHigh',
	'surfaceContainerHighest',
	'inverseSurface',
	'inverseOnSurface',
	'outline',
	'outlineVariant',
	'scrim',
	'shadow'
] as const;

// The phone's marks, Theme.kt: fixed, whatever the scheme.
const marks = {
	'mark-green': ['#1E7C43', '#7DD99C'],
	'mark-blue': ['#3557C8', '#B3C5FF'],
	'mark-magenta': ['#9A2F8D', '#F2A6E4'],
	'mark-cyan': ['#006A6F', '#6ED5DB'],
	'mark-up': ['#BA1A1A', '#FFB4AB'],
	'mark-down': ['#006A6F', '#6ED5DB']
};

const kebab = (role: string) => role.replace(/[A-Z]/g, (c) => '-' + c.toLowerCase());

/** The colour tokens as Tailwind theme variables, each light-dark(): the phone's dynamic scheme, tonal spot from the logo's indigo. */
export function colorsCSS(): string {
	const hct = Hct.fromInt(argbFromHex(seed));
	const light = new SchemeTonalSpot(hct, false, 0);
	const dark = new SchemeTonalSpot(hct, true, 0);
	const pair = (role: (typeof roles)[number]) => {
		const color = MaterialDynamicColors[role];
		return [hexFromArgb(color.getArgb(light)), hexFromArgb(color.getArgb(dark))];
	};
	const lines = roles.map((role) => {
		// The picked landing (proposals/landing/b-surfaces.html) keeps the logo's indigo as the light primary.
		const [l, d] = role === 'primary' ? [seed.toLowerCase(), pair(role)[1]] : pair(role);
		return `\t--color-${kebab(role)}: light-dark(${l}, ${d});`;
	});
	for (const [name, [l, d]] of Object.entries(marks)) {
		lines.push(`\t--color-${name}: light-dark(${l.toLowerCase()}, ${d.toLowerCase()});`);
	}
	return `/* Written by \`bun run theme\` from src/lib/theme/scheme.ts. */\n@theme static {\n${lines.join('\n')}\n}\n`;
}
