<script lang="ts">
	import Icon from './Icon.svelte';
</script>

<div class="stage">
	<div class="surfaces" role="img" aria-label="The same project in the terminal, the browser and the Android app">
		<figure class="s-term">
			<div class="mock term tty" aria-hidden="true">
				<div class="bar"><i></i><i></i><i></i></div>
				<pre>  <b>■ ferry</b>

  <b>▼ next</b> <span class="t-yellow">[ ]</span>
<span class="t-sel">&gt; [ ] Shell completions for fish</span>
  [ ] <span class="t-cyan">▼ Page long output in ferry diff</span>

  <b>▼ v1.3.0</b> <span class="t-blue">[/]</span>
  <span class="t-blue">[/]</span> <span class="t-mag">●</span> <span class="t-red">▲ Encrypt secrets with age</span> <span class="t-dim">1/3</span>
    <span class="t-dim">[x] Generate a key on first run</span>
    <span class="t-blue">[/]</span> Decrypt on apply
    [ ] Rotate keys without re-cloning
  [ ] <span class="t-cyan">●</span> <span class="t-red"><svg class="g" viewBox="0 0 10 14"><path d="M3 13V3M1 5l2-2.5L5 5M7 13V3M5 5l2-2.5L9 5" /></svg> Crash on a symlink loop in the source tree</span>
  [ ] <span class="t-blue">●</span> Per-host overrides <span class="t-dim">¶</span>
  [ ] <span class="t-cyan">▼ Dry-run flag for apply</span>

    <b>▼ docs</b> <span class="t-yellow">[ ]</span>
    [ ] <span class="t-green">●</span> Man page for ferry apply
    [ ] <span class="t-green">●</span> <span class="t-cyan"><svg class="g" viewBox="0 0 10 14"><path d="M3 1v10M1 9l2 2.5L5 9M7 1v10M5 9l2 2.5L9 9" /></svg> Migration guide from v1.2</span>

  <b>▶ v1.2.1</b> <span class="t-dim">[x]</span>
  <b>▶ v1.2.0</b> <span class="t-dim">[x]</span>
  <b>▶ v1.1.0</b> <span class="t-dim">[x]</span>
  <b>▶ v1.0.1</b> <span class="t-dim">[x]</span>
  <b>▶ v1.0.0</b> <span class="t-dim">[x]</span></pre>
				<div class="foot"><b>?</b> help  <b>a</b> add  <b>m</b> move  <b>f</b> filter  <b>/</b> search</div>
			</div>
			<figcaption>Terminal</figcaption>
		</figure>

		<figure class="s-browser">
			<div class="mock browser" aria-hidden="true">
				<div class="chrome"><div class="tab"><Icon name="logo" />Ordo</div></div>
				<div class="app">
					<div class="list">
						<div class="appbar">
							<span class="title">v1.3.0</span>
							<span class="ib"><Icon name="search" /></span>
							<span class="ib"><Icon name="filter" /></span>
							<span class="ib"><Icon name="more" /></span>
						</div>
						<div class="tabs"><span class="on">ferry</span><span>homelab</span><Icon name="add" /></div>
						<div class="rows">
							<div class="cat">next<Icon name="less" /></div>
							<div class="row">
								<Icon name="todo" class="st" />
								<div class="body"><div class="t"><span>Shell completions for fish</span></div></div>
							</div>
							<div class="row">
								<Icon name="todo" class="st" />
								<div class="body">
									<div class="t"><span>Page long output in ferry diff</span></div>
									<div class="chips"><span class="chip lo"><Icon name="down" />Low</span></div>
								</div>
							</div>
							<div class="cat">v1.3.0<Icon name="less" /></div>
							<div class="row sel">
								<Icon name="prog" class="st on" />
								<div class="body">
									<div class="t"><span>Encrypt secrets with age</span><i class="dot magenta"></i></div>
									<div class="pv">Encrypt the values marked secret with age before they are written, so a dotfiles repo is safe to push.</div>
									<div class="chips">
										<span class="chip hi"><Icon name="up" />High</span>
										<span class="chip"><Icon name="subtask" />1/3</span>
									</div>
								</div>
							</div>
							<div class="row sub done">
								<Icon name="done" class="st on" />
								<div class="body"><div class="t"><span>Generate a key on first run</span></div></div>
							</div>
							<div class="row sub">
								<Icon name="prog" class="st on" />
								<div class="body"><div class="t"><span>Decrypt on apply</span></div></div>
							</div>
							<div class="row sub">
								<Icon name="todo" class="st" />
								<div class="body"><div class="t"><span>Rotate keys without re-cloning</span></div></div>
							</div>
							<div class="row">
								<Icon name="todo" class="st" />
								<div class="body">
									<div class="t"><span>Crash on a symlink loop in the source tree</span><i class="dot cyan"></i></div>
									<div class="chips"><span class="chip hi"><Icon name="upup" />Critical</span></div>
								</div>
							</div>
							<div class="row">
								<Icon name="todo" class="st" />
								<div class="body">
									<div class="t"><span>Per-host overrides</span><i class="dot blue"></i></div>
									<div class="pv">A host picks its own values for a file without forking it.</div>
								</div>
							</div>
							<div class="row">
								<Icon name="todo" class="st" />
								<div class="body">
									<div class="t"><span>Dry-run flag for apply</span></div>
									<div class="chips"><span class="chip lo"><Icon name="down" />Low</span></div>
								</div>
							</div>
						</div>
					</div>
					<div class="detail">
						<div class="head">
							<span class="crumb">v1.3.0<Icon name="drop" /></span>
							<span class="ib"><Icon name="photo" /></span>
							<span class="ib"><Icon name="more" /></span>
						</div>
						<h4>Encrypt secrets with age</h4>
						<p class="desc">
							Encrypt the values marked secret with age before they are written, so a dotfiles repo is safe to push. One key per host,
							under ~/.config/ferry/keys.
						</p>
						<div class="chips">
							<span class="chip"><Icon name="prog" />In Progress</span>
							<span class="chip hi"><Icon name="up" />High</span>
							<span class="chip"><i class="dot magenta"></i>Magenta</span>
						</div>
						<div class="subs">
							<div class="done"><Icon name="done" class="st on" /><span>Generate a key on first run</span></div>
							<div><Icon name="prog" class="st on" /><span>Decrypt on apply</span></div>
							<div><Icon name="todo" class="st" /><span>Rotate keys without re-cloning</span></div>
							<div class="add"><Icon name="subtask" class="st" /><span>Add a subtask</span></div>
						</div>
						<div class="fab"><Icon name="check" />Mark done</div>
					</div>
				</div>
			</div>
			<figcaption>Browser</figcaption>
		</figure>

		<figure class="s-phone">
			<picture class="mock phone">
				<source media="(prefers-color-scheme: dark)" srcset="/landing/phone-dark.webp" />
				<img src="/landing/phone-light.webp" alt="" width="480" height="1066" loading="lazy" decoding="async" />
			</picture>
			<figcaption>Android</figcaption>
		</figure>
	</div>
</div>

<style>
	.stage {
		background: light-dark(#e4e6ff, #1d2040);
		border-radius: 32px;
		padding: 36px 32px 24px;
		max-width: 1376px;
		margin: 0 auto;
	}
	.surfaces {
		display: grid;
		grid-template-columns: 364px minmax(0, 1fr) 244px;
		gap: 20px;
		align-items: end;
	}
	figure {
		margin: 0;
	}
	figcaption {
		margin-top: 12px;
		text-align: center;
		font-size: 14px;
		font-weight: 500;
		color: var(--color-on-primary-container);
	}
	.mock {
		user-select: none;
		box-shadow: 0 1px 2px light-dark(rgb(30 34 80 / 0.08), rgb(0 0 0 / 0.3)), 0 8px 28px light-dark(rgb(30 34 80 / 0.1), rgb(0 0 0 / 0.35));
	}

	.term {
		border-radius: 12px;
		height: 540px;
		overflow: hidden;
		font-size: 11px;
		line-height: 1.62;
		display: flex;
		flex-direction: column;
	}
	.term .bar {
		display: flex;
		gap: 6px;
		padding: 12px 14px 4px;
	}
	.term .bar i {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: var(--t-dim);
		opacity: 0.55;
	}
	.term pre {
		margin: 0;
		padding: 10px 8px 0;
		font: inherit;
		white-space: pre;
		flex: 1;
	}
	.term .foot {
		padding: 6px 14px 12px 10px;
		color: var(--t-dim);
		white-space: pre;
		overflow: hidden;
	}
	.g {
		display: inline-block;
		width: 1ch;
		height: 1.15em;
		vertical-align: -0.2em;
		fill: none;
		stroke: currentColor;
		stroke-width: 1.3;
	}
	.browser {
		background: var(--color-surface-container-high);
		border-radius: 14px;
		overflow: hidden;
		height: 540px;
		display: flex;
		flex-direction: column;
	}
	.chrome {
		display: flex;
		align-items: end;
		height: 40px;
		padding: 0 10px;
		flex: none;
	}
	.chrome .tab {
		display: flex;
		align-items: center;
		gap: 8px;
		height: 32px;
		padding: 0 14px;
		background: var(--color-surface);
		border-radius: 10px 10px 0 0;
		font-size: 12.5px;
		width: 180px;
	}
	.chrome .tab :global(svg) {
		width: 16px;
		height: 16px;
	}
	.app {
		flex: 1;
		min-height: 0;
		background: var(--color-surface);
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.08fr);
		font-size: 15px;
		line-height: 1.55;
	}
	.list {
		min-width: 0;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
	.appbar {
		display: flex;
		align-items: center;
		height: 56px;
		padding: 4px 4px 0 18px;
		flex: none;
	}
	.appbar .title {
		font-size: 22px;
		flex: 1;
	}
	.ib {
		width: 40px;
		height: 40px;
		display: grid;
		place-items: center;
		color: var(--color-on-surface-variant);
	}
	.ib :global(svg) {
		width: 22px;
		height: 22px;
	}
	.tabs {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 0 8px;
		flex: none;
		font-weight: 500;
	}
	.tabs span {
		padding: 10px 12px 12px;
		color: var(--color-on-surface-variant);
		position: relative;
	}
	.tabs span.on {
		color: var(--color-primary);
	}
	.tabs span.on::after {
		content: '';
		position: absolute;
		left: 12px;
		right: 12px;
		bottom: 0;
		height: 3px;
		border-radius: 3px 3px 0 0;
		background: var(--color-primary);
	}
	.tabs :global(svg) {
		width: 22px;
		height: 22px;
		color: var(--color-on-surface-variant);
		margin-left: 6px;
	}
	.rows {
		overflow: hidden;
		flex: 1;
		border-top: 1px solid var(--color-outline-variant);
		padding-bottom: 8px;
	}
	.cat {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 10px 4px 20px;
		color: var(--color-primary);
		font-size: 13.5px;
		font-weight: 500;
	}
	.cat :global(svg) {
		width: 22px;
		height: 22px;
		color: var(--color-on-surface-variant);
	}
	.row {
		display: flex;
		gap: 12px;
		padding: 9px 14px;
		margin: 0 8px;
		border-radius: 16px;
	}
	.row.sel {
		background: var(--color-secondary-container);
	}
	.row.sub {
		padding-left: 46px;
	}
	.surfaces :global(.st) {
		width: 22px;
		height: 22px;
		flex: none;
		color: var(--color-on-surface-variant);
		margin-top: 1px;
	}
	.surfaces :global(.st.on) {
		color: var(--color-primary);
	}
	.body {
		flex: 1;
		min-width: 0;
	}
	.t {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.t span {
		flex: 1;
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.row.done .t span,
	.subs .done span {
		text-decoration: line-through;
		color: var(--color-on-surface-variant);
	}
	.pv {
		font-size: 13.5px;
		line-height: 1.4;
		color: var(--color-on-surface-variant);
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
		margin-top: 2px;
	}
	.dot {
		width: 9px;
		height: 9px;
		border-radius: 50%;
		flex: none;
	}
	.dot.magenta {
		background: var(--color-mark-magenta);
	}
	.dot.cyan {
		background: var(--color-mark-cyan);
	}
	.dot.blue {
		background: var(--color-mark-blue);
	}
	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-top: 7px;
	}
	.chip {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		height: 24px;
		padding: 0 8px 0 6px;
		border: 1px solid var(--color-outline);
		border-radius: 7px;
		font-size: 12.5px;
		font-weight: 500;
		color: var(--color-on-surface-variant);
	}
	.chip :global(svg) {
		width: 14px;
		height: 14px;
	}
	.chip.hi {
		color: var(--color-mark-up);
	}
	.chip.lo {
		color: var(--color-mark-down);
	}
	.detail {
		margin: 0 10px 10px 2px;
		background: var(--color-surface-container);
		border-radius: 22px;
		padding: 6px 22px 22px;
		position: relative;
		overflow: hidden;
		min-width: 0;
	}
	.detail .head {
		display: flex;
		align-items: center;
		height: 52px;
		gap: 2px;
		margin: 0 -10px 0 -4px;
	}
	.crumb {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		color: var(--color-primary);
		font-weight: 500;
		flex: 1;
	}
	.crumb :global(svg) {
		width: 20px;
		height: 20px;
	}
	.detail h4 {
		margin: 8px 0 12px;
		font-size: 24px;
		font-weight: 400;
		line-height: 1.25;
	}
	.desc {
		line-height: 1.55;
	}
	.detail .chips {
		margin-top: 16px;
		gap: 8px;
	}
	.detail .chip {
		height: 32px;
		padding: 0 12px 0 8px;
		border-radius: 8px;
		font-size: 14px;
		gap: 6px;
		color: var(--color-on-surface);
		border-color: var(--color-outline-variant);
	}
	.detail .chip.hi {
		color: var(--color-mark-up);
	}
	.detail .chip :global(svg) {
		width: 18px;
		height: 18px;
	}
	.detail .chip .dot {
		width: 12px;
		height: 12px;
	}
	.subs {
		margin-top: 14px;
	}
	.subs div {
		display: flex;
		align-items: center;
		gap: 14px;
		padding: 9px 0;
	}
	.subs .add {
		color: var(--color-on-surface-variant);
	}
	.fab {
		position: absolute;
		right: 18px;
		bottom: 18px;
		display: flex;
		align-items: center;
		gap: 10px;
		height: 52px;
		padding: 0 20px 0 16px;
		border-radius: 16px;
		background: var(--color-primary-container);
		color: var(--color-on-primary-container);
		font-weight: 500;
		box-shadow: 0 2px 6px rgb(0 0 0 / 0.15);
	}
	.fab :global(svg) {
		width: 22px;
		height: 22px;
	}

	.phone {
		height: 540px;
		display: block;
		border-radius: 24px;
	}
	.phone img {
		height: 100%;
		width: auto;
		display: block;
		border-radius: 24px;
	}

	@media (max-width: 1399px) {
		.surfaces {
			grid-template-columns: minmax(0, 1fr) auto;
		}
		.s-browser {
			grid-column: 1 / -1;
			order: -1;
		}
		.term,
		.phone {
			height: 516px;
		}
		.term {
			font-size: clamp(10px, 1.4vw, 11.25px);
		}
		.browser {
			height: 560px;
		}
	}
	@media (min-width: 1024px) and (max-width: 1399px) {
		.term,
		.phone {
			height: 600px;
		}
		.term {
			font-size: 13px;
		}
	}
	@media (max-width: 719px) {
		.stage {
			padding: 16px 12px 20px;
			border-radius: 24px;
			margin: 0 -4px;
		}
		.surfaces {
			grid-template-columns: minmax(0, 1fr);
		}
		.s-phone {
			display: none;
		}
		.app {
			grid-template-columns: minmax(0, 1fr);
		}
		.detail {
			display: none;
		}
		.term {
			height: auto;
			font-size: 10.2px;
		}
		.term pre {
			padding-bottom: 6px;
		}
		.term .foot {
			display: none;
		}
		.chrome .tab {
			width: 150px;
		}
	}
</style>
