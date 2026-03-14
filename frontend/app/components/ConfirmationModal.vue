<script setup lang="ts">
interface Props {
	isOpen: boolean;
	title: string;
	message: string;
	confirmText?: string;
	cancelText?: string;
	isDanger?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
	confirmText: "Confirm",
	cancelText: "Cancel",
	isDanger: false
});

const emit = defineEmits<{
	confirm: [];
	cancel: [];
}>();

const handleConfirm = () => {
	emit("confirm");
};

const handleCancel = () => {
	emit("cancel");
};
</script>

<template>
	<div v-if="isOpen" class="modal is-active">
		<div class="modal-background" @click="handleCancel"></div>
		<div class="modal-card">
			<header class="modal-card-head">
				<p class="modal-card-title">{{ title }}</p>
				<button class="delete" @click="handleCancel"></button>
			</header>
			<section class="modal-card-body">
				<p>{{ message }}</p>
			</section>
			<footer class="modal-card-foot">
				<button class="button" @click="handleCancel">{{ cancelText }}</button>
				<button
					:class="{
						'button is-danger': isDanger,
						'button is-primary': !isDanger
					}"
					@click="handleConfirm"
				>
					{{ confirmText }}
				</button>
			</footer>
		</div>
	</div>
</template>
