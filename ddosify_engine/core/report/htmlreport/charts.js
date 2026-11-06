"use strict";
// Draws the report's charts as inline SVG from the JSON in #data. No library, no network.
(function () {
	const node = document.getElementById("data");
	if (!node) return;
	const D = JSON.parse(node.textContent);
	const NS = "http://www.w3.org/2000/svg";
	const num = (v, d = 0) => (v == null || Number.isNaN(v) ? "-" : Number(v).toLocaleString("en-US", { maximumFractionDigits: d, minimumFractionDigits: d }));
	const pct = (v, d = 1) => (v == null ? "-" : num(v, d) + "%");
	const elapsed = (s) => {
		s = Math.max(0, Math.round(s));
		const h = Math.floor(s / 3600), m = Math.floor(s / 60) % 60, x = s % 60;
		return h ? `${h}:${String(m).padStart(2, "0")}:${String(x).padStart(2, "0")}` : `${m}:${String(x).padStart(2, "0")}`;
	};
	function el(tag, attrs, parent, text) {
		const e = document.createElementNS(NS, tag);
		for (const k in attrs || {}) e.setAttribute(k, attrs[k]);
		if (text != null) e.textContent = text;
		if (parent) parent.appendChild(e);
		return e;
	}
	function h(tag, attrs, parent, text) {
		const e = document.createElement(tag);
		for (const k in attrs || {}) k === "class" ? (e.className = attrs[k]) : e.setAttribute(k, attrs[k]);
		if (text != null) e.textContent = text;
		if (parent) parent.appendChild(e);
		return e;
	}
	function niceStep(range, n) {
		const raw = range / Math.max(1, n);
		const p = Math.pow(10, Math.floor(Math.log10(raw || 1)));
		const e = raw / p;
		return (e >= 7.5 ? 10 : e >= 3.5 ? 5 : e >= 1.5 ? 2 : 1) * p;
	}
	function yTicks(max, n = 4) {
		if (!(max > 0)) max = 1;
		const s = niceStep(max, n);
		const top = Math.ceil(max / s) * s;
		const out = [];
		for (let v = 0; v <= top + s / 2; v += s) out.push(+v.toFixed(6));
		return out;
	}
	function nearest(arr, t) {
		let lo = 0, hi = arr.length - 1;
		if (hi < 0) return -1;
		while (lo < hi) {
			const mid = (lo + hi) >> 1;
			arr[mid][0] < t ? (lo = mid + 1) : (hi = mid);
		}
		if (lo > 0 && Math.abs(arr[lo - 1][0] - t) < Math.abs(arr[lo][0] - t)) lo--;
		return lo;
	}
	function watch(holder, draw) {
		const run = () => {
			const w = Math.round(holder.clientWidth);
			if (w && w !== holder._w) { holder._w = w; draw(); }
		};
		new ResizeObserver(run).observe(holder);
		run();
	}
	let clipN = 0;

	/* tooltip: the hovered chart's follows the pointer, the others sit beside their crosshair */
	const tip = h("div", { id: "tip", role: "status", "aria-live": "polite" }, document.body);
	function fillTip(box, head, rows) {
		box.replaceChildren();
		h("div", { class: "h" }, box, head);
		for (const r of rows) {
			const row = h("div", { class: "r" }, box);
			const k = h("i", r.dash ? { class: "dash" } : r.box ? { class: "box" } : {}, row);
			if (r.box) k.style.background = r.color || "transparent";
			else k.style.borderTopColor = r.color || "transparent";
			h("b", {}, row, r.value);
			h("span", {}, row, r.name);
		}
	}
	function showTip(x, y, head, rows) {
		fillTip(tip, head, rows);
		tip.style.display = "block";
		const w = tip.offsetWidth, hh = tip.offsetHeight;
		let left = x + 14, top = y + 14;
		if (left + w > innerWidth - 8) left = x - w - 14;
		if (top + hh > innerHeight - 8) top = y - hh - 14;
		tip.style.left = Math.max(8, left) + "px";
		tip.style.top = Math.max(8, top) + "px";
	}
	const hideTip = () => (tip.style.display = "none");

	/* the shared time axis */
	const S = D.stages || [];
	const TS = D.ts;
	let T0 = Infinity, T1 = -Infinity;
	for (const s of S) if (s.start != null) { T0 = Math.min(T0, s.start); T1 = Math.max(T1, s.end); }
	if (TS && TS.t.length) { T0 = Math.min(T0, TS.t[0]); T1 = Math.max(T1, TS.t[TS.t.length - 1]); }
	if (D.req && D.req.t.length) { T0 = Math.min(T0, D.req.t[0]); T1 = Math.max(T1, D.req.t[D.req.t.length - 1]); }
	// Without a time range (a log with only the report line) the time charts are left out, the rest is drawn.
	const hasTime = isFinite(T0) && T1 > T0;
	const stageAt = (t) => { let n = ""; for (const s of S) if (s.start != null && t >= s.start - 0.5) n = s.name || ""; return n; };
	const charts = [];

	function stageBands(svg, x, M, ih) {
		S.forEach((s, i) => {
			if (s.start == null) return;
			const x0 = x(s.start), x1 = x(s.end);
			el("rect", { x: x0, y: M.t, width: Math.max(0, x1 - x0), height: ih, fill: s.warmup ? "var(--bandw)" : i % 2 ? "var(--band)" : "transparent" }, svg);
			if (i > 0) el("line", { x1: x0, x2: x0, y1: M.t, y2: M.t + ih, stroke: "var(--grid)" }, svg);
			const label = s.name || `S${i + 1}`;
			if (x1 - x0 > label.length * 6.2 + 6) el("text", { x: (x0 + x1) / 2, y: M.t - 6, "text-anchor": "middle", class: "band-l" }, svg, label);
		});
	}
	function timeAxis(svg, x, M, iw, ih) {
		const span = T1 - T0;
		const step = [5, 10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600, 7200].find((m) => span / m <= Math.max(3, iw / 75)) || 7200;
		for (let t = Math.ceil(T0 / step) * step; t <= T1; t += step) {
			el("line", { x1: x(t), x2: x(t), y1: M.t + ih, y2: M.t + ih + 4, stroke: "var(--axis)" }, svg);
			el("text", { x: x(t), y: M.t + ih + 17, "text-anchor": "middle" }, svg, elapsed(t));
		}
	}
	function legendItem(lg, name, color, kind) {
		const sp = h("span", {}, lg);
		const i = h("i", kind ? { class: kind } : {}, sp);
		if (kind === "box") i.style.background = `var(${color})`;
		else i.style.borderTopColor = `var(${color})`;
		sp.append(name);
	}
	function endLabels(svg, items, top, bottom, xPos) {
		items.sort((a, b) => a.y - b.y);
		for (let i = 1; i < items.length; i++) items[i].y = Math.max(items[i].y, items[i - 1].y + 13);
		const over = items.length ? items[items.length - 1].y - bottom : 0;
		if (over > 0) for (const it of items) it.y = Math.max(top, it.y - over);
		for (const it of items) el("text", { x: xPos, y: it.y + 4, class: "lab" }, svg, it.text);
	}

	function timeChart(parent, cfg) {
		const card = h("div", { class: "chart" }, parent);
		h("h3", {}, card, cfg.title);
		if (cfg.sub) h("div", { class: "sub" }, card, cfg.sub);
		const lg = h("div", { class: "legend" }, card);
		for (const s of cfg.series) legendItem(lg, s.name, s.color, s.band ? "box" : s.dash ? "dash" : null);
		for (const r of cfg.refs || []) legendItem(lg, r.label, "--ref", "dash");
		const holder = h("div", { class: "holder" }, card);
		const H = cfg.height || 220, M = { l: 56, r: cfg.labels ? 92 : 14, t: 20, b: 24 };
		const c = { cross() {} };
		const fmt = cfg.fmt || ((v) => num(v, v < 10 && !Number.isInteger(v) ? 1 : 0));
		const top = (s, p) => (s.band ? p[2] : p[1]);
		function draw() {
			holder.replaceChildren();
			const W = Math.max(280, holder.clientWidth);
			const svg = el("svg", { viewBox: `0 0 ${W} ${H}`, height: H, tabindex: 0, role: "img", "aria-label": cfg.title }, holder);
			const iw = W - M.l - M.r, ih = H - M.t - M.b;
			let max = cfg.yMax || 0;
			for (const s of cfg.series) for (const p of s.data) { const v = top(s, p); if (v != null && v > max) max = v; }
			for (const r of cfg.refs || []) max = Math.max(max, r.y * 1.12);
			const ticks = cfg.yMax && max <= cfg.yMax ? yTicks(cfg.yMax) : yTicks(max * 1.04);
			const ymax = ticks[ticks.length - 1];
			const x = (t) => M.l + ((t - T0) / (T1 - T0 || 1)) * iw;
			const y = (v) => M.t + ih - (v / ymax) * ih;
			stageBands(svg, x, M, ih);
			for (const v of ticks) {
				el("line", { x1: M.l, x2: M.l + iw, y1: y(v), y2: y(v), stroke: v === 0 ? "var(--axis)" : "var(--grid)" }, svg);
				el("text", { x: M.l - 8, y: y(v) + 4, "text-anchor": "end" }, svg, fmt(v) + (cfg.unit && cfg.unit.length <= 3 ? cfg.unit : ""));
			}
			timeAxis(svg, x, M, iw, ih);
			const cid = "clip" + ++clipN;
			el("rect", { x: M.l, y: M.t - 6, width: iw, height: ih + 12 }, el("clipPath", { id: cid }, el("defs", {}, svg)));
			const g = el("g", { "clip-path": `url(#${cid})` }, svg);
			const labels = [];
			for (const s of cfg.series) {
				if (s.band) {
					let d = "", run = [];
					const flush = () => {
						if (run.length > 1) d += "M" + run.map((p) => `${x(p[0]).toFixed(1)},${y(p[2]).toFixed(1)}`).join("L") + "L" + run.slice().reverse().map((p) => `${x(p[0]).toFixed(1)},${y(p[1]).toFixed(1)}`).join("L") + "Z";
						run = [];
					};
					for (const p of s.data) (p[1] == null || p[2] == null ? flush() : run.push(p));
					flush();
					el("path", { d, fill: `var(${s.color})`, class: "fill" }, g);
					continue;
				}
				let d = "", on = false, prev = null, lastP = null;
				for (const p of s.data) {
					if (p[1] == null) { on = false; continue; }
					const px = x(p[0]).toFixed(1), py = y(p[1]).toFixed(1);
					if (s.step && on) d += `L${px},${prev}`;
					d += (on ? "L" : "M") + px + "," + py;
					on = true; prev = py; lastP = p;
				}
				if (s.area && d) {
					const pts = s.data.filter((p) => p[1] != null);
					el("path", { d: d + `L${x(pts[pts.length - 1][0]).toFixed(1)},${y(0)}L${x(pts[0][0]).toFixed(1)},${y(0)}Z`, fill: `var(${s.color})`, class: "area" }, g);
				}
				el("path", { d, fill: "none", stroke: `var(${s.color})`, "stroke-width": s.width || 2, "stroke-linejoin": "round", "stroke-linecap": "round", "stroke-dasharray": s.dash ? "5 4" : "none" }, g);
				if (cfg.labels && lastP) labels.push({ y: y(lastP[1]), text: s.short || s.name });
			}
			if (labels.length) endLabels(svg, labels, M.t, M.t + ih, M.l + iw + 8);
			for (const r of cfg.refs || []) {
				el("line", { x1: M.l, x2: M.l + iw, y1: y(r.y), y2: y(r.y), stroke: "var(--ref)", "stroke-dasharray": "4 4", "stroke-width": 1.25 }, svg);
				el("text", { x: M.l + 6, y: y(r.y) - 5, class: "halo" }, svg, r.label);
			}
			const hair = el("line", { y1: M.t, y2: M.t + ih, stroke: "var(--ink2)", "stroke-width": 1, opacity: 0 }, svg);
			const dots = cfg.series.map((s) => (s.band ? null : el("circle", { r: 4, fill: `var(${s.color})`, stroke: "var(--card)", "stroke-width": 2, opacity: 0 }, svg)));
			const hit = el("rect", { x: M.l, y: 0, width: iw, height: H, fill: "transparent", style: "cursor:crosshair" }, svg);
			const ptip = h("div", { class: "ptip", "aria-hidden": "true" }, holder);
			const snapT = (t) => {
				const ref = cfg.series.find((s) => s.data.length && !s.band && !s.flat) || cfg.series.find((s) => s.data.length);
				if (!ref) return t;
				const i = nearest(ref.data, t);
				return i < 0 ? t : ref.data[i][0];
			};
			c.cross = (t, active, ev) => {
				if (t == null) { hair.setAttribute("opacity", 0); dots.forEach((d) => d && d.setAttribute("opacity", 0)); ptip.style.display = "none"; return; }
				hair.setAttribute("x1", x(t)); hair.setAttribute("x2", x(t)); hair.setAttribute("opacity", 0.55);
				const rows = [];
				cfg.series.forEach((s, k) => {
					const i = nearest(s.data, t);
					const p = i < 0 ? null : s.data[i];
					const ok = p && p[1] != null && Math.abs(p[0] - t) <= (s.tol || cfg.tol || 30);
					if (s.band) { rows.push({ color: `var(${s.color})`, box: true, value: ok && p[2] != null ? `${fmt(p[1])} - ${fmt(p[2])}${cfg.unit || ""}` : "-", name: s.name }); return; }
					if (ok) { dots[k].setAttribute("cx", x(p[0])); dots[k].setAttribute("cy", y(p[1])); dots[k].setAttribute("opacity", 1); }
					else dots[k].setAttribute("opacity", 0);
					rows.push({ color: `var(${s.color})`, dash: s.dash, value: ok ? fmt(p[1]) + (cfg.unit || "") : "-", name: s.name });
				});
				if (cfg.extra) for (const r of cfg.extra(t)) rows.push(r);
				const head = `${elapsed(t)}${stageAt(t) ? " · " + stageAt(t) : ""}`;
				if (active) { ptip.style.display = "none"; if (ev) showTip(ev.clientX, ev.clientY, head, rows); return; }
				fillTip(ptip, head, rows);
				ptip.style.display = "block";
				const sc = holder.clientWidth / W, px = x(t) * sc, w = ptip.offsetWidth;
				ptip.style.left = (px + 12 + w > holder.clientWidth ? Math.max(0, px - 12 - w) : px + 12) + "px";
			};
			let last = null;
			hit.addEventListener("pointermove", (ev) => {
				const r = svg.getBoundingClientRect();
				const px = ((ev.clientX - r.left) / r.width) * W;
				const t = snapT(T0 + ((px - M.l) / iw) * (T1 - T0));
				last = t;
				for (const o of charts) o.cross(t, o === c, o === c ? ev : null);
			});
			hit.addEventListener("pointerleave", () => { hideTip(); for (const o of charts) o.cross(null); });
			svg.addEventListener("keydown", (ev) => {
				if (ev.key !== "ArrowLeft" && ev.key !== "ArrowRight") return;
				ev.preventDefault();
				const t = snapT((last ?? (T0 + T1) / 2) + (ev.key === "ArrowRight" ? 1 : -1) * Math.max(1, (T1 - T0) / 120));
				last = t;
				const r = svg.getBoundingClientRect();
				for (const o of charts) o.cross(t, o === c, o === c ? { clientX: r.left + (x(t) / W) * r.width, clientY: r.top + 30 } : null);
			});
			svg.addEventListener("blur", () => { hideTip(); for (const o of charts) o.cross(null); });
		}
		charts.push(c);
		watch(holder, draw);
		return c;
	}

	/* milliseconds shown in seconds once they reach two seconds */
	function msScale(slo, ...arrays) {
		let m = slo || 0;
		for (const a of arrays) for (const v of a || []) if (v != null && v > m) m = v;
		const small = (v) => num(v, Number.isInteger(v) ? 0 : v < 1 ? 2 : v < 10 ? 1 : 0);
		return m >= 2000 ? { k: 1000, unit: " s", fmt: small } : { k: 1, unit: " ms", fmt: small };
	}
	const zip = (t, a, k = 1) => t.map((x, i) => [x, a[i] == null ? null : a[i] / k]);
	const zip2 = (t, a, b, k = 1) => t.map((x, i) => [x, a[i] == null ? null : a[i] / k, b[i] == null ? null : b[i] / k]);
	function latency(parent, title, sub, t, p50, p95, slo, extra, tol) {
		const sc = msScale(slo, p50, p95);
		timeChart(parent, {
			title, sub, tol,
			series: [{ name: "p50 - p95", color: "--s1", band: true, data: zip2(t, p50, p95, sc.k) }, { name: "p50", color: "--s1", data: zip(t, p50, sc.k) }],
			refs: slo ? [{ y: slo / sc.k, label: `SLO ${sc.fmt(slo / sc.k)}${sc.unit}` }] : [],
			fmt: sc.fmt, unit: sc.unit, extra,
		});
	}

	/* load */
	const L = hasTime ? document.getElementById("load") : null;
	const per = "/" + (D.rateUnit || "s");
	if (TS && L) {
		const t = TS.t;
		const flow = [
			{ name: "Planned", color: "--ref", data: zip(t, TS.target), dash: true, width: 1.5, flat: true, step: true },
			{ name: "Started", color: "--s1", data: zip(t, TS.achieved) },
		];
		if (TS.failed.some((v) => v)) flow.push({ name: "Failed", color: "--crit", data: zip(t, TS.failed) });
		timeChart(L, { title: `Arrival rate (iterations${per})`, sub: `Iterations started against each stage's planned rate, and failed requests; ${num(D.window)} s sliding window`, series: flow, unit: per, labels: true });
		timeChart(L, { title: "In flight", sub: "Requests sent and not yet ended", series: [{ name: "In flight", color: "--s1", data: zip(t, TS.inflight), step: true, area: true }] });
	}

	/* latency */
	const LT = hasTime ? document.getElementById("latency") : null;
	const R = D.req;
	if (LT && R && R.t.length) {
		const extra = (t) => { const i = nearest(R.t.map((x) => [x]), t); return i < 0 ? [] : [{ color: null, value: num(R.n[i]), name: `answers started in the last ${num(R.win[i] / 60000, 1)} min` }]; };
		const sub = `By start time; window max(${num(D.window)} s, 30 requests), shaded from p50 to p95; points under 20 answers are not drawn` + (R.sampled ? "; from a uniform sample of the requests" : "");
		if (R.stream) latency(LT, "Time to first token", sub, R.t, R.ftP50, R.ftP95, D.slo.ft, extra, 20);
		latency(LT, "Total time", sub, R.t, R.totP50, R.totP95, D.slo.total, extra, 20);
	} else if (LT && TS) {
		const sub = "The running stage's answers so far, shaded from p50 to p95";
		if (TS.ttft) latency(LT, "Time to first token", sub, TS.t, TS.ttft.p50, TS.ttft.p95, D.slo.ft);
		latency(LT, "End to end", "Iteration wall time, all steps, of the iterations that succeeded; " + sub.charAt(0).toLowerCase() + sub.slice(1), TS.t, TS.e2e.p50, TS.e2e.p95, D.slo.total);
	}
	if (LT && TS && TS.itl) {
		const sc = msScale(0, TS.itl.p50, TS.itl.p95);
		timeChart(LT, { title: D.itlName || "ITL", sub: "Time between tokens, or between SSE events when tokens are not counted", series: [{ name: "p50 - p95", color: "--s1", band: true, data: zip2(TS.t, TS.itl.p50, TS.itl.p95, sc.k) }, { name: "p50", color: "--s1", data: zip(TS.t, TS.itl.p50, sc.k) }], fmt: sc.fmt, unit: sc.unit });
	}
	if (LT && TS && TS.goodput.some((v) => v != null)) {
		timeChart(LT, { title: "Goodput", sub: "Share of the running stage's finished iterations that met the SLO", series: [{ name: "goodput", color: "--s3", data: zip(TS.t, TS.goodput) }], fmt: (v) => num(v, 0), unit: "%", yMax: 100 });
	}

	/* stages: outcome per stage, latency distribution */
	const named = (s, i) => (s.warmup && !/warm/i.test(s.name) ? `${s.name || "S" + (i + 1)} (warm-up)` : s.name || `S${i + 1}`);
	function rowChart(card, title, sub, legend, rowsH, drawRows) {
		h("h3", {}, card, title);
		h("div", { class: "sub" }, card, sub);
		const lg = h("div", { class: "legend" }, card);
		for (const [n, c, k] of legend) legendItem(lg, n, c, k);
		const holder = h("div", { class: "holder" }, card);
		watch(holder, () => { holder.replaceChildren(); drawRows(holder, Math.max(280, holder.clientWidth)); });
	}
	const OC = document.getElementById("stage-outcomes");
	if (OC && S.length) {
		const hasGood = S.some((s) => s.good != null);
		const parts = (hasGood ? [["good", "Within SLO", "--good"], ["late", "Succeeded, outside SLO", "--warn"]] : [["ok", "Succeeded", "--good"]]).concat([["rejected", "503", "--crit"], ["rateLimited", "429", "--serious"], ["otherFail", "Other failure", "--muted"]]);
		rowChart(OC, "Outcome per stage", "Share of iterations; the count is beside each bar", parts.map(([, n, c]) => [n, c, "box"]), 34, (holder, W) => {
			const rowH = 34, M = { l: 110, r: 70, t: 8, b: 8 };
			const H = M.t + M.b + rowH * S.length, iw = W - M.l - M.r;
			const svg = el("svg", { viewBox: `0 0 ${W} ${H}`, height: H, role: "img", "aria-label": "Outcome per stage" }, holder);
			S.forEach((s, i) => {
				const y0 = M.t + i * rowH, bh = 14, yb = y0 + (rowH - bh) / 2;
				if (s.warmup) el("rect", { x: 0, y: y0, width: W, height: rowH, fill: "var(--bandw)" }, svg);
				el("text", { x: M.l - 10, y: y0 + rowH / 2 + 4, "text-anchor": "end", class: "lab" }, svg, named(s, i));
				el("text", { x: W - M.r + 8, y: y0 + rowH / 2 + 4 }, svg, num(s.iterations));
				const failed = s.rejected + s.rateLimited + s.otherFail;
				const n = Math.max(1, s.iterations, s.ok + failed);
				const v = { good: s.good || 0, late: Math.max(0, s.ok - (s.good || 0)), ok: s.ok, rejected: s.rejected, rateLimited: s.rateLimited, otherFail: s.otherFail };
				let x0 = M.l;
				const segs = parts.filter(([k]) => v[k] > 0);
				segs.forEach(([k, , col], j) => {
					const w = (v[k] / n) * iw;
					const r = el("rect", { x: x0, y: yb, width: Math.max(1, w - (j < segs.length - 1 ? 2 : 0)), height: bh, rx: 3, fill: `var(${col})`, tabindex: 0 }, svg);
					const show = (ev) => {
						const b = r.getBoundingClientRect();
						showTip(ev.clientX ?? b.left + b.width / 2, ev.clientY ?? b.top, named(s, i), parts.map(([kk, nn, cc]) => ({ color: `var(${cc})`, box: true, value: `${num(v[kk])} (${pct((v[kk] / n) * 100)})`, name: nn })));
					};
					r.addEventListener("pointermove", show);
					r.addEventListener("focus", show);
					r.addEventListener("pointerleave", hideTip);
					r.addEventListener("blur", hideTip);
					x0 += w;
				});
			});
		});
	} else if (OC) OC.remove();

	const DC = document.getElementById("stage-dist");
	if (DC && S.some((s) => s.ft || s.total)) {
		const hasFT = S.some((s) => s.ft);
		const kinds = (hasFT ? [["ft", "First token", "--s1"]] : []).concat([["total", "Total", "--s2"]]);
		let mx = 0;
		for (const s of S) for (const [k] of kinds) if (s[k]) mx = Math.max(mx, s[k].p99 ?? s[k].p95 ?? 0);
		for (const v of [D.slo.ft, D.slo.total]) if (v) mx = Math.max(mx, v);
		const sc = mx >= 2000 ? { k: 1000, u: " s" } : { k: 1, u: " ms" };
		const val = (v) => (v == null ? "-" : num(v / sc.k, sc.k > 1 && v < 10000 ? 1 : 0) + sc.u);
		const leg = kinds.map(([, n, c]) => [n, c]);
		if (D.slo.ft || D.slo.total) leg.push([`SLO: ${[D.slo.ft && hasFT ? "first token " + val(D.slo.ft) : "", D.slo.total ? "total " + val(D.slo.total) : ""].filter(Boolean).join(", ")}`, "--ref", "dash"]);
		rowChart(DC, "Latency distribution per stage", "Dot p50, tick p95, ring p99; the line runs from p50 to p99", leg, 34, (holder, W) => {
			const rowH = 34, M = { l: 110, r: 16, t: 22, b: 24 };
			const H = M.t + M.b + rowH * S.length, iw = W - M.l - M.r;
			const svg = el("svg", { viewBox: `0 0 ${W} ${H}`, height: H, role: "img", "aria-label": "Latency distribution per stage" }, holder);
			const ticks = yTicks((mx / sc.k) * 1.03, Math.max(2, Math.min(5, Math.floor(iw / 70)))), top = ticks[ticks.length - 1];
			const x = (v) => M.l + (v / sc.k / top) * iw;
			for (const v of ticks) {
				el("line", { x1: M.l + (v / top) * iw, x2: M.l + (v / top) * iw, y1: M.t, y2: H - M.b, stroke: v === 0 ? "var(--axis)" : "var(--grid)" }, svg);
				el("text", { x: M.l + (v / top) * iw, y: H - M.b + 16, "text-anchor": "middle" }, svg, num(v) + sc.u);
			}
			for (const ms of [hasFT ? D.slo.ft : 0, D.slo.total]) if (ms) {
				el("line", { x1: x(ms), x2: x(ms), y1: M.t - 4, y2: H - M.b, stroke: "var(--ref)", "stroke-dasharray": "4 4" }, svg);
				el("text", { x: x(ms), y: M.t - 8, "text-anchor": "middle", class: "halo" }, svg, val(ms));
			}
			S.forEach((s, i) => {
				const y0 = M.t + i * rowH;
				if (s.warmup) el("rect", { x: 0, y: y0, width: W, height: rowH, fill: "var(--bandw)" }, svg);
				el("text", { x: M.l - 10, y: y0 + rowH / 2 + 4, "text-anchor": "end", class: "lab" }, svg, named(s, i));
				kinds.forEach(([k, , col], j) => {
					const q = s[k];
					if (!q || q.p50 == null) return;
					const yy = y0 + rowH / 2 + (kinds.length > 1 ? (j ? 6 : -6) : 0);
					el("line", { x1: x(q.p50), x2: x(q.p99 ?? q.p95), y1: yy, y2: yy, stroke: `var(${col})`, "stroke-width": 2, "stroke-linecap": "round" }, svg);
					if (q.p95 != null) el("line", { x1: x(q.p95), x2: x(q.p95), y1: yy - 4, y2: yy + 4, stroke: `var(${col})`, "stroke-width": 2 }, svg);
					if (q.p99 != null) el("circle", { cx: x(q.p99), cy: yy, r: 3.5, fill: "var(--card)", stroke: `var(${col})`, "stroke-width": 2 }, svg);
					el("circle", { cx: x(q.p50), cy: yy, r: 4.5, fill: `var(${col})`, stroke: "var(--card)", "stroke-width": 2 }, svg);
				});
				const hit = el("rect", { x: 0, y: y0, width: W, height: rowH, fill: "transparent", tabindex: 0 }, svg);
				const show = (ev) => {
					hit.setAttribute("fill", "var(--band)");
					const rows = [];
					for (const [k, n, c] of kinds) {
						const q = s[k];
						if (!q) continue;
						rows.push({ color: `var(${c})`, value: `${val(q.p50)} / ${val(q.p95)} / ${val(q.p99)}`, name: `${n.toLowerCase()} p50 / p95 / p99` });
						rows.push({ color: null, value: `${val(q.min)} - ${val(q.max)}`, name: `${n.toLowerCase()} min - max` });
					}
					rows.push({ color: null, value: num(s.ok), name: "succeeded" });
					const r = hit.getBoundingClientRect();
					showTip(ev.clientX ?? r.left + r.width / 2, ev.clientY ?? r.top, named(s, i), rows);
				};
				const off = () => { hit.setAttribute("fill", "transparent"); hideTip(); };
				hit.addEventListener("pointermove", show);
				hit.addEventListener("focus", show);
				hit.addEventListener("pointerleave", off);
				hit.addEventListener("blur", off);
			});
		});
	} else if (DC) DC.remove();
	const grid = document.getElementById("stagecharts");
	if (grid && !grid.children.length) grid.remove();
	for (const id of ["load", "latency"]) {
		const g = document.getElementById(id);
		if (g && !g.children.length) g.closest("section").remove();
	}

	const btn = document.getElementById("theme");
	if (btn) {
		btn.hidden = false;
		btn.addEventListener("click", () => {
			const root = document.documentElement;
			const dark = root.dataset.theme ? root.dataset.theme === "dark" : matchMedia("(prefers-color-scheme: dark)").matches;
			root.dataset.theme = dark ? "light" : "dark";
		});
	}
})();
