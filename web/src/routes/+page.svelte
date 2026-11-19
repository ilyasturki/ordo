<script lang="ts">
	import { app } from '#lib/paths.ts';
	import Bump from '#lib/landing/Bump.svelte';
	import Command from '#lib/landing/Command.svelte';
	import Icon from '#lib/landing/Icon.svelte';
	import Surfaces from '#lib/landing/Surfaces.svelte';
	import { origin as site } from '#lib/landing/origin.ts';
	import '#lib/landing/landing.css';

	const title = 'Ordo: plan your projects one version at a time';
	const description =
		'A keyboard-driven project planner, in the terminal, on Android and in your browser. Each project is a plain JSON file.';
	const repo = 'https://github.com/ilyasturki/ordo';
</script>

<svelte:head>
	<title>{title}</title>
	<meta name="description" content={description} />
	<link rel="canonical" href={site} />
	<link rel="preload" href="/fonts/fira-code.woff2" as="font" type="font/woff2" crossorigin="anonymous" />
	<meta property="og:type" content="website" />
	<meta property="og:site_name" content="Ordo" />
	<meta property="og:url" content={site} />
	<meta property="og:title" content={title} />
	<meta property="og:description" content={description} />
	<meta property="og:image" content="{site}landing/social-preview.png" />
	<meta property="og:image:width" content="1280" />
	<meta property="og:image:height" content="640" />
	<meta property="og:image:alt" content="Ordo in a terminal: a project's tasks grouped under version categories" />
	<meta name="twitter:card" content="summary_large_image" />
</svelte:head>

<div class="landing">
<header class="wrap top">
	<a class="brand" href="/"><Icon name="logo" />Ordo</a>
	<nav class="nav" aria-label="Site">
		<a class="btn text small" href="#get">Install</a>
		<a class="btn text small gh" href={repo}>GitHub</a>
	</nav>
</header>

<main>
	<div class="wrap hero">
		<h1>Plan your projects one version at a time</h1>
		<p class="lead">
			Ordo is a project planner you run from the keyboard, in the terminal, on Android or in your browser. In the terminal, each
			project is a plain JSON file on your disk.
		</p>
		<div class="cta">
			<a class="btn filled" href={app()}>Open Ordo in your browser</a>
			<span class="alt"><a class="btn text" href="#get">Install the CLI</a><a class="btn text" href="#get">Get the Android app</a></span>
		</div>
		<p class="note">Nothing to install. Your projects stay in your browser.</p>
	</div>

	<div class="stage-wrap">
		<Surfaces />
	</div>

	<section class="wrap" aria-labelledby="versions">
		<div class="split">
			<div class="copy">
				<h2 id="versions">Name a category after a version</h2>
				<p>
					Call it <strong>v1.3.0</strong>, or <strong>v1.3.0 - polish</strong>. <kbd>ctrl+a</kbd> adds one to the patch,
					<kbd>gm</kbd> bumps the minor and <kbd>gM</kbd> the major. Only the digits change, so the label stays.
				</p>
				<p>
					The keys are vim's where vim has one: <kbd>j</kbd> <kbd>k</kbd> to move, <kbd>v</kbd> for a range, <kbd>/</kbd> to search,
					<kbd>u</kbd> to undo. Shipped versions fold to one row each.
				</p>
			</div>
			<Bump />
		</div>
	</section>

	<section class="wrap" aria-labelledby="files">
		<div class="split">
			<div class="copy">
				<h2 id="files">Plain JSON files on your disk</h2>
				<p>
					Each project is one JSON file, written atomically. Syncthing can carry it between machines, and when it changes on disk
					while Ordo is open, Ordo merges it within a second.
				</p>
				<p>
					The terminal app never touches the network. The <a href="{repo}/blob/main/docs/file-format.md">file format is documented</a>, so
					other programs can read and write it.
				</p>
			</div>
			<figure class="file">
				<figcaption class="name"><Icon name="doc" />ferry.json</figcaption>
				<pre>{'{'}
  <span class="k">"schema"</span>: <span class="n">2</span>,
  <span class="k">"name"</span>: <span class="s">"ferry"</span>,
  <span class="k">"categories"</span>: [
    {'{'}
      <span class="k">"name"</span>: <span class="s">"v1.3.0"</span>,
      <span class="k">"tasks"</span>: [
        {'{'}
          <span class="k">"title"</span>: <span class="s">"Encrypt secrets with age"</span>,
          <span class="k">"status"</span>: <span class="s">"in_progress"</span>,
          <span class="k">"priority"</span>: <span class="s">"high"</span>,
          <span class="k">"tag_color"</span>: <span class="s">"magenta"</span>
        {'}'},
<span class="c">        ...</span></pre>
			</figure>
		</div>
	</section>

	<section class="wrap" id="get" aria-labelledby="get-title">
		<h2 id="get-title">Get Ordo</h2>
		<div class="cards">
			<div class="card main">
				<h3>In your browser</h3>
				<p>The whole app runs in the page.</p>
				<p>Host your own copy next to <code>ordo serve</code> and it syncs with your other devices.</p>
				<div class="act"><a class="btn filled" href={app()}>Open Ordo in your browser</a></div>
			</div>
			<div class="card">
				<h3>In the terminal</h3>
				<p>One binary for Linux and macOS.</p>
				<div class="cmds">
					<Command command="brew install ilyasturki/ordo/ordo" label="Copy the Homebrew command" />
					<Command command="yay -S ordo-bin" label="Copy the AUR command" />
					<Command command="nix run github:ilyasturki/ordo" label="Copy the Nix command" />
				</div>
				<div class="act"><a class="more" href="{repo}#install">Go, binaries and other ways</a></div>
			</div>
			<div class="card">
				<h3>On Android</h3>
				<p>
					The app keeps a copy of every project and works without signal. Edits wait in a queue and reach your own <code>ordo serve</code>
					when it answers.
				</p>
				<div class="act"><a class="btn tonal" href="{repo}/releases/latest">Download the APK</a></div>
			</div>
		</div>
	</section>
</main>

<footer>
	<div class="wrap">
		<span>Ordo is MIT-licensed.</span>
		<nav aria-label="Project">
			<a href={repo}>Source</a>
			<a href="{repo}/blob/main/CHANGELOG.md">Changelog</a>
			<a href="{repo}/blob/main/docs/file-format.md">File format</a>
		</nav>
	</div>
</footer>
</div>

<style>
	.landing {
		font-size: 17px;
		line-height: 1.55;
	}
	a {
		color: var(--color-primary);
	}
	h1,
	h2,
	h3 {
		text-wrap: balance;
		margin: 0;
		font-weight: 500;
	}
	p {
		margin: 0;
	}
	code,
	kbd {
		font-family: var(--font-code);
	}
	code {
		font-size: 0.9em;
	}
	:focus-visible {
		outline: 2px solid var(--color-primary);
		outline-offset: 2px;
	}

	.wrap {
		max-width: 1240px;
		margin: 0 auto;
		padding: 0 32px;
	}
	.stage-wrap {
		padding: 0 32px;
	}

	.top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 72px;
	}
	.brand {
		display: flex;
		align-items: center;
		gap: 10px;
		color: var(--color-on-surface);
		text-decoration: none;
		font-size: 20px;
		font-weight: 600;
	}
	.brand :global(svg) {
		width: 32px;
		height: 32px;
	}
	.nav {
		display: flex;
		gap: 4px;
	}

	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		height: 48px;
		padding: 0 24px;
		border-radius: 24px;
		font-size: 16px;
		font-weight: 500;
		line-height: 1;
		letter-spacing: 0.01em;
		text-decoration: none;
		white-space: nowrap;
		transition: box-shadow 0.15s ease-out;
	}
	.btn.filled {
		background: var(--color-primary);
		color: var(--color-on-primary);
	}
	.btn.tonal {
		background: var(--color-secondary-container);
		color: var(--color-on-secondary-container);
	}
	.btn.text {
		color: var(--color-primary);
		padding: 0 16px;
	}
	.btn.small {
		height: 40px;
		padding: 0 16px;
		font-size: 15px;
	}
	@media (hover: hover) {
		.btn.filled:hover {
			background: color-mix(in srgb, var(--color-primary) 90%, var(--color-on-primary));
			box-shadow: 0 1px 3px rgb(0 0 0 / 0.2);
		}
		.btn.tonal:hover {
			background: color-mix(in srgb, var(--color-secondary-container) 90%, var(--color-on-secondary-container));
		}
		.btn.text:hover {
			background: color-mix(in srgb, var(--color-primary) 8%, transparent);
		}
	}
	.btn.filled:active {
		background: color-mix(in srgb, var(--color-primary) 84%, var(--color-on-primary));
	}
	.btn.tonal:active {
		background: color-mix(in srgb, var(--color-secondary-container) 84%, var(--color-on-secondary-container));
	}
	.btn.text:active {
		background: color-mix(in srgb, var(--color-primary) 12%, transparent);
	}

	.hero {
		text-align: center;
		padding-top: 40px;
		padding-bottom: 36px;
	}
	.hero h1 {
		font-size: clamp(32px, 3.6vw, 46px);
		line-height: 1.12;
		letter-spacing: -0.015em;
		max-width: 15ch;
		margin: 0 auto;
	}
	.lead {
		max-width: 34em;
		margin: 16px auto 0;
		color: var(--color-on-surface-variant);
		font-size: 18px;
	}
	.cta {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 8px 4px;
		margin-top: 28px;
	}
	.alt {
		display: flex;
		justify-content: center;
		gap: 4px;
	}
	.cta .btn.filled {
		height: 56px;
		padding: 0 28px;
		border-radius: 28px;
		font-size: 17px;
	}
	.note {
		margin-top: 14px;
		font-size: 14px;
		color: var(--color-on-surface-variant);
	}

	section.wrap {
		padding-top: 96px;
	}
	.split {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.05fr);
		gap: 64px;
		align-items: center;
	}
	.split h2,
	#get h2 {
		font-size: clamp(28px, 3vw, 36px);
		line-height: 1.2;
		letter-spacing: -0.01em;
	}
	.copy p {
		margin-top: 16px;
		color: var(--color-on-surface-variant);
	}
	.copy a {
		text-decoration: underline;
		text-underline-offset: 0.15em;
	}
	.copy strong {
		color: var(--color-on-surface);
		font-weight: 500;
	}
	kbd {
		font-size: 0.82em;
		padding: 2px 6px;
		border-radius: 6px;
		background: var(--color-surface-container-high);
		color: var(--color-on-surface);
		white-space: nowrap;
	}

	.file {
		margin: 0;
		background: var(--color-surface-container);
		border-radius: 16px;
		overflow: hidden;
		min-width: 0;
	}
	.file .name {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 14px 20px;
		font-size: 14px;
		color: var(--color-on-surface-variant);
		border-bottom: 1px solid var(--color-outline-variant);
	}
	.file .name :global(svg) {
		width: 18px;
		height: 18px;
	}
	.file pre {
		margin: 0;
		padding: 16px 20px 20px;
		font: 13.5px/1.65 var(--font-code);
		overflow-x: auto;
	}
	.file .k {
		color: var(--color-primary);
	}
	.file .s {
		color: var(--color-mark-green);
	}
	.file .n {
		color: var(--color-mark-magenta);
	}
	.file .c {
		color: var(--color-on-surface-variant);
	}

	#get h2 {
		text-align: center;
	}
	.cards {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 16px;
		margin-top: 32px;
	}
	.card {
		background: var(--color-surface-container);
		border-radius: 24px;
		padding: 28px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		min-width: 0;
	}
	.card.main {
		background: var(--color-primary-container);
		color: var(--color-on-primary-container);
	}
	.card h3 {
		font-size: 22px;
	}
	.card p {
		color: var(--color-on-surface-variant);
		font-size: 16px;
	}
	.card.main p {
		color: var(--color-on-primary-container);
	}
	.act {
		margin-top: auto;
		padding-top: 8px;
	}
	.cmds {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.more {
		font-size: 15px;
	}

	footer {
		margin-top: 96px;
		border-top: 1px solid var(--color-outline-variant);
	}
	footer .wrap {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 12px 24px;
		padding-top: 28px;
		padding-bottom: 40px;
		font-size: 15px;
		color: var(--color-on-surface-variant);
	}
	footer nav {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 20px;
	}
	footer a {
		color: var(--color-on-surface-variant);
		text-decoration: none;
		padding: 4px 0;
	}
	@media (hover: hover) {
		footer a:hover {
			color: var(--color-on-surface);
			text-decoration: underline;
		}
	}

	@media (max-width: 1023px) {
		.cards {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	@media (max-width: 899px) {
		.split {
			grid-template-columns: minmax(0, 1fr);
			gap: 32px;
		}
	}
	@media (max-width: 719px) {
		.landing {
			font-size: 16px;
		}
		.wrap,
		.stage-wrap {
			padding: 0 16px;
		}
		.top {
			height: 64px;
		}
		.gh {
			display: none;
		}
		.hero {
			padding-top: 28px;
			padding-bottom: 28px;
		}
		.lead {
			font-size: 17px;
		}
		.cta {
			flex-direction: column;
			align-items: stretch;
			gap: 8px;
			margin-top: 24px;
		}
		.alt .btn {
			padding: 0 10px;
		}
		section.wrap {
			padding-top: 72px;
		}
		.card {
			padding: 22px 20px;
		}
		.file pre {
			font-size: 12.5px;
			padding: 14px 16px 18px;
		}
		footer {
			margin-top: 72px;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.btn {
			transition: none;
		}
	}
</style>
