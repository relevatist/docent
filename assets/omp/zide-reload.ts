// oh-my-pi extension: reload the zide editor after omp writes files.
// Install: cp zide-reload.ts ~/.omp/agent/extensions/
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";

// `resolve` applies changes that `ast_edit` previews.
const FILE_TOOLS = new Set(["edit", "write", "resolve"]);

export default function zideReload(pi: ExtensionAPI) {
	pi.on("tool_result", async event => {
		if (event.isError || !FILE_TOOLS.has(event.toolName)) return;
		try {
			await pi.exec("zide", ["reload", "--quiet"]);
		} catch {
			// zide is not on PATH: nothing to reload.
		}
	});
}
