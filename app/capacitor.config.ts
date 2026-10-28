import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
	appId: 'io.github.ilyasturki.ordo',
	appName: 'Ordo',
	webDir: 'build',
	android: {
		// The app runs at https://localhost, and some keep ordo serve on plain http in their LAN.
		allowMixedContent: true
	}
};

export default config;
