import * as server from '../entries/pages/demo/better-auth/login/_page.server.ts.js';

export const index = 6;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/demo/better-auth/login/_page.svelte.js')).default;
export { server };
export const server_id = "src/routes/demo/better-auth/login/+page.server.ts";
export const imports = ["_app/immutable/nodes/6.BzZtG8ez.js","_app/immutable/chunks/C0a7Pvhc.js","_app/immutable/chunks/xihTtKlq.js","_app/immutable/chunks/BM0j4mF7.js","_app/immutable/chunks/COsExA2f.js"];
export const stylesheets = [];
export const fonts = [];
