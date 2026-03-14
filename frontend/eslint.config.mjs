import steeringWaves from "@steeringwaves/eslint-config";
import { globalIgnores } from "eslint/config";
import { withNuxt } from "./.nuxt/eslint.config.mjs";

export default withNuxt(steeringWaves, globalIgnores(["dist/", "node_modules/"]), {
	ignores: ["nuxt.config.ts", "tsconfig.json"],
	rules: {
		"@typescript-eslint/no-unused-expressions": "off",
		"vue/no-multiple-template-root": "off",
		"vue/html-self-closing": "off",
		"vue/multi-word-component-names": "off",
		"@typescript-eslint/no-explicit-any": "off",
		"import/prefer-default-export": "off",
		"@typescript-eslint/consistent-type-definitions": "error"
	}
});
