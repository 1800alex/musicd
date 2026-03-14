<script setup lang="ts">
import { ref, computed, onMounted } from "vue";

export interface ToastMessage {
	id: string;
	message: string;
	type: "success" | "error" | "info" | "warning";
	duration?: number; // ms, 0 = permanent
}

const toasts = ref<ToastMessage[]>([]);

const showToast = (message: string, type: "success" | "error" | "info" | "warning" = "info", duration: number = 3000) => {
	const id = Math.random().toString(36).substr(2, 9);
	const toast: ToastMessage = { id, message, type, duration };
	toasts.value.push(toast);

	if (duration > 0) {
		setTimeout(() => {
			removeToast(id);
		}, duration);
	}

	return id;
};

const removeToast = (id: string) => {
	const index = toasts.value.findIndex((t) => t.id === id);
	if (index >= 0) {
		toasts.value.splice(index, 1);
	}
};

const getIcon = (type: string) => {
	switch (type) {
		case "success":
			return "fa-check-circle";
		case "error":
			return "fa-exclamation-circle";
		case "warning":
			return "fa-exclamation-triangle";
		default:
			return "fa-info-circle";
	}
};

// Export for use in other components
defineExpose({
	showToast,
	removeToast,
	toasts: computed(() => toasts.value)
});
</script>

<template>
	<div class="toast-container">
		<transition-group name="toast" tag="div">
			<div
				v-for="toast in toasts"
				:key="toast.id"
				:class="['toast', `toast-${toast.type}`]"
				:data-testid="`toast-${toast.type}`"
			>
				<div class="toast-content">
					<font-awesome-icon :icon="getIcon(toast.type)" class="toast-icon" />
					<span class="toast-message">{{ toast.message }}</span>
				</div>
				<button class="toast-close" @click="removeToast(toast.id)">
					<font-awesome-icon icon="fa-times" />
				</button>
			</div>
		</transition-group>
	</div>
</template>

<style scoped>
.toast-container {
	position: fixed;
	top: 20px;
	right: 20px;
	z-index: 2000;
	display: flex;
	flex-direction: column;
	gap: 10px;
	pointer-events: none;
}

.toast {
	display: flex;
	align-items: center;
	justify-content: space-between;
	background: var(--clr-surface-elevated);
	color: var(--clr-text-primary);
	padding: 1rem;
	border-radius: 4px;
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
	min-width: 300px;
	pointer-events: auto;
	animation: slideIn 0.3s ease-out;
}

.toast-success {
	background: rgba(76, 175, 80, 1);
	border-left: 4px solid #4caf50;
	color: #4caf50;
}

.toast-error {
	background: rgba(244, 67, 54, 1);
	border-left: 4px solid #f44336;
	color: #f44336;
}

.toast-warning {
	background: rgba(255, 193, 7, 1);
	border-left: 4px solid #ffc107;
	color: #ffc107;
}

.toast-info {
	background: rgba(33, 150, 243, 1);
	border-left: 4px solid #2196f3;
	color: #2196f3;
}

.toast-content {
	display: flex;
	align-items: center;
	gap: 12px;
	flex: 1;
}

.toast-icon {
	flex-shrink: 0;
	font-size: 1.25rem;
}

.toast-message {
	flex: 1;
	line-height: 1.4;
}

.toast-close {
	background: none;
	border: none;
	color: inherit;
	cursor: pointer;
	padding: 4px 8px;
	margin-left: 8px;
	opacity: 0.7;
	transition: opacity 0.2s;
}

.toast-close:hover {
	opacity: 1;
}

@keyframes slideIn {
	from {
		transform: translateX(100%);
		opacity: 0;
	}
	to {
		transform: translateX(0);
		opacity: 1;
	}
}

.toast-enter-active {
	animation: slideIn 0.3s ease-out;
}

.toast-leave-active {
	animation: slideOut 0.3s ease-in;
}

@keyframes slideOut {
	from {
		transform: translateX(0);
		opacity: 1;
	}
	to {
		transform: translateX(100%);
		opacity: 0;
	}
}
</style>
