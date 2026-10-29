import { describe, expect, it } from 'vitest';
import {
	buildCategories,
	canonicalTag,
	TAGS,
	tagName,
	descriptionFields,
	formatEstimate,
	imagesOf,
	joinDescription,
	preview,
	type Project
} from '$lib/model';
import { parsePairing } from '$lib/api';

const img = '![shot](attachments/0123456789abcdef0123456789abcdef.png)';

describe('descriptionFields', () => {
	it('splits the trailing image lines from the text', () => {
		expect(descriptionFields(`notes\nmore\n${img}`)).toEqual({
			text: 'notes\nmore',
			images: [img]
		});
	});

	it('keeps a description whole when an image sits inside the text', () => {
		const d = `before ${img} after`;
		expect(descriptionFields(d)).toEqual({ text: d, images: [] });
	});

	it('joins back to the same description', () => {
		const d = `notes\n${img}`;
		const { text, images } = descriptionFields(d);
		expect(joinDescription(text, images)).toBe(d);
	});
});

describe('images and previews', () => {
	it('finds stored images and drops them from the preview', () => {
		expect(imagesOf(`a ${img}`)).toEqual([
			{
				name: 'shot',
				target: 'attachments/0123456789abcdef0123456789abcdef.png',
				file: '0123456789abcdef0123456789abcdef.png'
			}
		]);
		expect(preview(`first  line\n\nsecond\nthird ${img}`)).toBe('first line\nsecond');
	});

	it('ignores a reference outside the store', () => {
		expect(imagesOf('![x](https://example.com/a.png)')).toEqual([]);
	});
});

it('formats estimates like the TUI', () => {
	expect([0, 15, 60, 479, 480, 2400].map((m) => formatEstimate(m))).toEqual([
		'',
		'15m',
		'1h',
		'7h',
		'1d',
		'5d'
	]);
});

describe('buildCategories', () => {
	const p: Project = {
		id: 'p',
		name: 'P',
		created_at: '',
		updated_at: '',
		categories: [
			{
				id: 'c',
				name: 'C',
				created_at: '',
				tasks: [
					{ id: 'a', title: 'open', status: 'todo', created_at: '', updated_at: '' },
					{ id: 'b', title: 'done parent', status: 'completed', created_at: '', updated_at: '' },
					{
						id: 'b1',
						parent_id: 'b',
						title: 'open child',
						status: 'todo',
						created_at: '',
						updated_at: ''
					},
					{ id: 'd', title: 'finished', status: 'cancelled', created_at: '', updated_at: '' }
				]
			}
		]
	};

	it('moves a block to completed only once its subtasks are finished too', () => {
		const [v] = buildCategories(p);
		expect(v.open.map((r) => r.task.id)).toEqual(['a', 'b']);
		expect(v.open[1].children.map((r) => r.task.id)).toEqual(['b1']);
		expect(v.completed.map((r) => r.task.id)).toEqual(['d']);
	});

	it('keeps a parent for a matching child', () => {
		const [v] = buildCategories(
			p,
			{ statuses: [], priorities: [], categories: [], tags: [] },
			'child'
		);
		expect(v.open.map((r) => r.task.id)).toEqual(['b']);
	});
});

describe('parsePairing', () => {
	it('reads the server and code out of a link', () => {
		expect(parsePairing(' https://ordo.example.com/pair/ABC123 ')).toEqual({
			server: 'https://ordo.example.com',
			code: 'ABC123'
		});
		expect(parsePairing('http://host:7319/sub/pair/X')).toEqual({
			server: 'http://host:7319/sub',
			code: 'X'
		});
	});

	it('takes a bare code and refuses anything else', () => {
		expect(parsePairing('ABC123')).toEqual({ server: '', code: 'ABC123' });
		expect(parsePairing('https://example.com/other')).toBeNull();
		expect(parsePairing('')).toBeNull();
	});
});

describe('tags', () => {
	it('offers the four dots and none', () => {
		expect(TAGS).toEqual(['', 'green', 'blue', 'magenta', 'cyan']);
	});

	it('reads a diamond from an older build as its dot', () => {
		expect(canonicalTag('magenta-diamond')).toBe('magenta');
		expect(canonicalTag('cyan')).toBe('cyan');
		expect(canonicalTag('red-diamond')).toBe('red-diamond');
		expect(tagName('green-diamond')).toBe('Green');
	});

	it('filters a diamond under its dot', () => {
		const p: Project = {
			id: 'p',
			name: 'P',
			created_at: '',
			updated_at: '',
			categories: [
				{
					id: 'c',
					name: 'C',
					created_at: '',
					tasks: [
						{
							id: 'a',
							title: 'a',
							status: 'todo',
							tag_color: 'blue-diamond',
							created_at: '',
							updated_at: ''
						},
						{
							id: 'b',
							title: 'b',
							status: 'todo',
							tag_color: 'green',
							created_at: '',
							updated_at: ''
						}
					]
				}
			]
		};
		const [v] = buildCategories(p, {
			statuses: [],
			priorities: [],
			categories: [],
			tags: ['blue']
		});
		expect(v.open.map((r) => r.task.id)).toEqual(['a']);
	});
});
