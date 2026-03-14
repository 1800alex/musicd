import { ref } from "vue";

let toastComponent: any = null;

export const setToastComponent = (component: any) => {
	toastComponent = component;
};

export const useToast = () => {
	const showSuccess = (message: string, duration: number = 3000) => {
		toastComponent?.showToast(message, "success", duration);
	};

	const showError = (message: string, duration: number = 4000) => {
		toastComponent?.showToast(message, "error", duration);
	};

	const showWarning = (message: string, duration: number = 3000) => {
		toastComponent?.showToast(message, "warning", duration);
	};

	const showInfo = (message: string, duration: number = 3000) => {
		toastComponent?.showToast(message, "info", duration);
	};

	return {
		showSuccess,
		showError,
		showWarning,
		showInfo
	};
};
