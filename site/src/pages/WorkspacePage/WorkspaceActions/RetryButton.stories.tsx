import type { Meta, StoryObj } from "@storybook/react";
import { expect, userEvent, waitFor, within } from "@storybook/test";
import { MockWorkspace } from "testHelpers/entities";
import { RetryButton } from "./RetryButton";

const meta: Meta<typeof RetryButton> = {
	title: "pages/WorkspacePage/RetryButton",
	component: RetryButton,
};

export default meta;
type Story = StoryObj<typeof RetryButton>;

export const Default: Story = {};

export const WithBuildParameters: Story = {
	args: {
		enableBuildParameters: true,
		workspace: MockWorkspace,
	},
	parameters: {
		queries: [
			{
				key: ["workspace", MockWorkspace.id, "parameters"],
				data: { templateVersionRichParameters: [], buildParameters: [] },
			},
		],
	},
};

export const WithOpenBuildParameters: Story = {
	args: {
		enableBuildParameters: true,
		workspace: MockWorkspace,
	},
	parameters: {
		queries: [
			{
				key: ["workspace", MockWorkspace.id, "parameters"],
				data: { templateVersionRichParameters: [], buildParameters: [] },
			},
		],
	},
	play: async ({ canvasElement, step }) => {
		const screen = within(canvasElement);

		await step("open popover", async () => {
			await userEvent.click(screen.getByTestId("build-parameters-button"));
			await waitFor(() =>
				expect(screen.getByText("Build Options")).toBeInTheDocument(),
			);
		});
	},
};
