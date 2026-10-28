import { ordo } from '$lib/state.svelte';

export const MAX_IMAGE_BYTES = 25 << 20;

function extOf(b: Uint8Array): string | null {
	if (b[0] === 0x89 && b[1] === 0x50 && b[2] === 0x4e && b[3] === 0x47) {
		return 'png';
	}
	if (b[0] === 0xff && b[1] === 0xd8 && b[2] === 0xff) {
		return 'jpg';
	}
	if (b[0] === 0x47 && b[1] === 0x49 && b[2] === 0x46 && b[3] === 0x38) {
		return 'gif';
	}
	const ascii = (from: number, to: number) => String.fromCharCode(...b.slice(from, to));
	if (ascii(0, 4) === 'RIFF' && ascii(8, 12) === 'WEBP') {
		return 'webp';
	}
	return null;
}

function altOf(file: File): string {
	const dot = file.name.lastIndexOf('.');
	const name = (dot > 0 ? file.name.slice(0, dot) : file.name).replace(/[[\]\p{Cc}]/gu, ' ');
	const flat = name.split(/\s+/u).filter(Boolean).join(' ');
	return flat === '' || flat === 'image' ? 'photo' : flat;
}

/** Stores the image under the name ordo serve gives the same bytes, and returns its line for the details. */
export async function keepImage(file: File): Promise<string> {
	if (file.size > MAX_IMAGE_BYTES) {
		throw new Error(`Images over ${MAX_IMAGE_BYTES >> 20} MB can't be attached`);
	}
	const bytes = new Uint8Array(await file.arrayBuffer());
	const ext = extOf(bytes);
	if (ext === null) {
		throw new Error('Only png, jpeg, gif and webp images can be attached');
	}
	const sum = new Uint8Array(await crypto.subtle.digest('SHA-256', bytes));
	const hash = Array.from(sum.slice(0, 16), (x) => x.toString(16).padStart(2, '0')).join('');
	const name = `${hash}.${ext}`;
	await ordo.keepImage(name, new Blob([bytes], { type: file.type || `image/${ext}` }));
	return `![${altOf(file)}](attachments/${name})`;
}
