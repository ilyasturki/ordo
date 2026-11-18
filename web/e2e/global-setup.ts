import { ordo } from './serve';

// Workers get the binary through the environment, so no test pays for the build in its own timeout.
export default function setup(): void {
	process.env.ORDO_E2E_BIN = ordo();
}
