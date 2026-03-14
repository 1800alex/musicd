// import AuthService from "@/composables/auth.service";
// import checkUnauthorized from "@/composables/checkUnauthorized";
// import useAppState from "@/stores/appState";
// import StorageService from "@/utils/Services/storage.service";

export default async function (): Promise<void> {
	// const appState = useAppState();
	// if (!appState.user) {
	// 	try {
	// 		let userInfo = StorageService.getStorage("user");
	// 		if (userInfo) {
	// 			// console.log("User info found in storage", userInfo);
	// 			appState.Login(userInfo);
	// 		} else {
	// 			console.log("No user info found in storage, fetching from server");
	// 			userInfo = await AuthService.GetUserInfo();
	// 			if (userInfo) {
	// 				// console.log("User info fetched from server", userInfo);
	// 				appState.Login(userInfo);
	// 				StorageService.setStorage("user", userInfo);
	// 			} else {
	// 				if (appState.loggedIn) {
	// 					console.error(`Error fetching user info`);
	// 					DisplayErrorNotificationMessage(`Error fetching user info`);
	// 					await checkUnauthorized({ status: 401 });
	// 				}
	// 			}
	// 		}
	// 	} catch (error) {
	// 		if (appState.loggedIn) {
	// 			console.error(`Error fetching user info`, error);
	// 			DisplayErrorNotificationMessage(`Error fetching user info`);
	// 			await checkUnauthorized({ status: 401 });
	// 		}
	// 		return;
	// 	}
	// }
}
