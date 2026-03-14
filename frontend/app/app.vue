<script setup lang="ts">
import { ref, onMounted } from "vue";
import Toast from "~/components/Toast.vue";
import { setToastComponent } from "~/composables/useToast";

const toastComponent = ref<InstanceType<typeof Toast>>();

onMounted(() => {
	if (toastComponent.value) {
		setToastComponent(toastComponent.value);
	}
});
</script>

<template>
	<div>
		<NuxtRouteAnnouncer />
		<NuxtLayout>
			<NuxtPage />
		</NuxtLayout>
		<Toast ref="toastComponent" />
	</div>
</template>
