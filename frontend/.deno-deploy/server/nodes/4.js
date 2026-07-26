

export const index = 4;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/demo/playwright/_page.svelte.js')).default;
export const imports = ["_app/immutable/nodes/4.aO35c6s0.js","_app/immutable/chunks/C0a7Pvhc.js","_app/immutable/chunks/xihTtKlq.js"];
export const stylesheets = [];
export const fonts = [];
