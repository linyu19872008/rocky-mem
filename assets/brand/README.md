# Brand assets

The mascot is **Rocky**, a water drop in work overalls: experience is recorded one
drop at a time, and the stone is what gets worn into shape.

| File | Use |
|---|---|
| `icon-512.png` | README, docs, listings (transparent, with a 3 px rim) |
| `mascot-1024.png` | large editorial use (transparent, no rim) |

Both are derived from one source illustration. The build script and the rest of
the size ladder (16–1024, favicon, avatar) live in the plugin repository:
[`rocky-mem-dsh`](https://github.com/linyu19872008/rocky-mem-dsh), under
`assets/brand/` and `scripts/build-brand-assets.py`. Regenerate there and copy
the two files here; do not hand-edit them.

Ground shadow and background are intentionally removed: the icon must sit on any
surface. The rim exists only for legibility on light backgrounds and is invisible
on dark ones.
