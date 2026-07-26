//#region node_modules/.deno/@sveltejs+kit@2.70.1/node_modules/@sveltejs/kit/src/runtime/app/paths/internal/server.js
var base = "";
var assets = base;
var app_dir = "_app";
var initial = {
	base,
	assets
};
/**
* `base` could be overridden during rendering to be relative;
* this one's the original non-relative base path
*/
var initial_base = initial.base;
/**
* @param {{ base: string, assets: string }} paths
*/
function override(paths) {
	base = paths.base;
	assets = paths.assets;
}
function reset() {
	base = initial.base;
	assets = initial.assets;
}
/** @param {string} path */
function set_assets(path) {
	assets = initial.assets = path;
}
//#endregion
//#region node_modules/.deno/@sveltejs+kit@2.70.1/node_modules/@sveltejs/kit/src/runtime/app/env/internal.js
var version = "1785060103789";
var building = false;
var prerendering = false;
function set_building() {
	building = true;
}
function set_prerendering() {
	prerendering = true;
}
//#endregion
export { version as a, base as c, reset as d, set_assets as f, set_prerendering as i, initial_base as l, prerendering as n, app_dir as o, set_building as r, assets as s, building as t, override as u };
