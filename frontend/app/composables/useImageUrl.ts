import { useBackendURL } from "~/composables/useBackendURL";

/**
 * Constructs a proper image URL that works across web, capacitor, and electron platforms.
 * Wraps useBackendURL.getHTTPURL for image-specific URLs.
 */
export const useImageUrl = () => {
	const { getHTTPURL } = useBackendURL();

	const getImageUrl = (path: string): string => {
		// Ensure path starts with /
		const normalizedPath = path.startsWith("/") ? path : `/${path}`;
		return getHTTPURL(normalizedPath);
	};

	return {
		getImageUrl
	};
};
