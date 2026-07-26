import { _ as escape_html } from "../../../../chunks/server.js";
import "../../../../chunks/forms.js";
//#region src/routes/demo/better-auth/+page.svelte
function _page($$renderer, $$props) {
	$$renderer.component(($$renderer) => {
		let { data } = $$props;
		$$renderer.push(`<h1>Hi, ${escape_html(data.user.name)}!</h1> <p>Your user ID is ${escape_html(data.user.id)}.</p> <form method="post" action="?/signOut"><button class="rounded-md bg-blue-600 px-4 py-2 text-white transition hover:bg-blue-700">Sign out</button></form>`);
	});
}
//#endregion
export { _page as default };
