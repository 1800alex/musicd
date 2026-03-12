<script setup lang="ts">
import { defineComponent } from "vue";
// import type { QTooltipProps } from "quasar";

defineComponent({
	name: "ToolTip"
});
const props = withDefaults(
	defineProps<{
		// Text for the tooltip to show
		text: string;
		// Options for where to display the tooltip
		// anchor?: QTooltipProps["anchor"];
		// self?: QTooltipProps["self"];
		// Delay in ms to show tooltip
		delay?: number;
	}>(),
	{
		anchor: "top middle",
		self: "bottom middle",
		delay: 500
	}
);
</script>

<template>
	<!-- <QTooltip class="bg-grey-8" :anchor="anchor" :self="self" :delay="delay" transition-show="scale" transition-hide="scale">
		{{ props.text }}
	</QTooltip> -->
	<div class="dropdown is-hoverable">
		<div class="dropdown-trigger">
			<button class="button" aria-haspopup="true" aria-controls="dropdown-menu4">
				<span>Hover me</span>
				<span class="icon is-small">
					<font-awesome-icon icon="fa-angle-down" />
				</span>
			</button>
		</div>
		<div id="dropdown-menu4" class="dropdown-menu" role="menu">
			<div class="dropdown-content">
				<div class="dropdown-item">
					<p>
						{{ props.text }}
					</p>
				</div>
			</div>
		</div>
	</div>
</template>
