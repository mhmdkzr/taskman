import { t as building } from "../chunks/internal2.js";
import { n as svelteKitHandler, t as auth } from "../chunks/auth.js";
//#region src/hooks.server.ts
var handleBetterAuth = async ({ event, resolve }) => {
	const session = await auth.api.getSession({ headers: event.request.headers });
	if (session) {
		event.locals.session = session.session;
		event.locals.user = session.user;
	}
	return svelteKitHandler({
		event,
		resolve,
		auth,
		building
	});
};
var handle = handleBetterAuth;
//#endregion
export { handle };
