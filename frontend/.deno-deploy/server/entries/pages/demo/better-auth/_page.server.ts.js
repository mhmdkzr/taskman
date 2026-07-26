import { t as auth } from "../../../../chunks/auth.js";
import { redirect } from "@sveltejs/kit";
//#region src/routes/demo/better-auth/+page.server.ts
var load = (event) => {
	if (!event.locals.user) return redirect(302, "/demo/better-auth/login");
	return { user: event.locals.user };
};
var actions = { signOut: async (event) => {
	await auth.api.signOut({ headers: event.request.headers });
	return redirect(302, "/demo/better-auth/login");
} };
//#endregion
export { actions, load };
