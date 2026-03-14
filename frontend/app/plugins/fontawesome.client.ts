import { library, config } from "@fortawesome/fontawesome-svg-core";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import {
	faHome,
	faMusic,
	faPlay,
	faPause,
	faStop,
	faStepBackward,
	faStepForward,
	faVolumeUp,
	faVolumeDown,
	faVolumeMute,
	faList,
	faFolder,
	faUsers,
	faUser,
	faSearch,
	faPlus,
	faTrash,
	faEdit,
	faCog,
	faSync,
	faEye,
	faArrowLeft,
	faTimes,
	faHeart,
	faDownload,
	faShare,
	faRandom,
	faRepeat,
	faBars,
	fas
} from "@fortawesome/free-solid-svg-icons";

import { fab } from "@fortawesome/free-brands-svg-icons";

// This is important, we are going to let Nuxt worry about the CSS
config.autoAddCss = false;

// Add the icons to the library
library.add(
	fab,
	fas,
	faHome,
	faMusic,
	faPlay,
	faPause,
	faStop,
	faStepBackward,
	faStepForward,
	faVolumeUp,
	faVolumeDown,
	faVolumeMute,
	faList,
	faFolder,
	faUsers,
	faUser,
	faSearch,
	faPlus,
	faTrash,
	faEdit,
	faCog,
	faSync,
	faEye,
	faArrowLeft,
	faTimes,
	faHeart,
	faDownload,
	faShare,
	faRandom,
	faRepeat,
	faBars
);

export default defineNuxtPlugin((nuxtApp) => {
	nuxtApp.vueApp.component("FontAwesomeIcon", FontAwesomeIcon);
});
