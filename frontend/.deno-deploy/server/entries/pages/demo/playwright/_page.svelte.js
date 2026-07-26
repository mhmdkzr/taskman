import "../../../../chunks/server.js";
//#region src/routes/demo/playwright/+page.svelte
function _page($$renderer) {
	$$renderer.push(`<h1>Playwright e2e test demo</h1>`);
}
//#endregion
export { _page as default };
