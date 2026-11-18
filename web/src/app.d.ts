declare global {
	interface ImportMetaEnv {
		/** The build: `site` (the landing at /, the app at /app/) or `private` (the app at /). */
		readonly ORDO_WEB: 'site' | 'private';
	}
}

export {};
