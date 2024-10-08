import type { Branding } from "../branding";
import colors from "../tailwindColors";

export const branding: Branding = {
	enterprise: {
		background: colors.blue[950],
		divider: colors.blue[900],
		border: colors.blue[400],
		text: colors.blue[50],
	},
	premium: {
		background: colors.violet[950],
		divider: colors.violet[900],
		border: colors.violet[400],
		text: colors.violet[50],
	},

	featureStage: {
		background: colors.sky[900],
		divider: colors.sky[800],
		border: colors.sky[400],
		text: colors.sky[400],

		hover: {
			background: colors.gray[900],
			divider: colors.gray[800],
			border: colors.sky[400],
			text: colors.sky[400],
		},
	},
};

export default branding;
