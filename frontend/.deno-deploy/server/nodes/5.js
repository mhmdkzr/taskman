import * as server from '../entries/pages/demo/better-auth/_page.server.ts.js';

export const index = 5;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/demo/better-auth/_page.svelte.js')).default;
export { server };
export const server_id = "src/routes/demo/better-auth/+page.server.ts";
export const imports = ["_app/immutable/nodes/5.fMAraMfL.js","_app/immutable/chunks/C0a7Pvhc.js","_app/immutable/chunks/xihTtKlq.js","_app/immutable/chunks/BM0j4mF7.js","_app/immutable/chunks/COsExA2f.js"];
export const stylesheets = [];
export const fonts = [];
